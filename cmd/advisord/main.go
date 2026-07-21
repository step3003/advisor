// Command advisord — HTTP API-сервер «Advisor» (ТЗ v2.0).
//
// Разворачивается на VPS: переиспользует ядро (domain + application), серверный
// SQLite как источник правды, отдаёт JSON API за токен-авторизацией. Конфиг — env.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	accountsvc "advisor/internal/application/account"
	catalogsvc "advisor/internal/application/catalog"
	currencysvc "advisor/internal/application/currency"
	iosvc "advisor/internal/application/io"
	ledgersvc "advisor/internal/application/ledger"
	planningsvc "advisor/internal/application/planning"
	recurringsvc "advisor/internal/application/recurring"
	reportingsvc "advisor/internal/application/reporting"
	settingssvc "advisor/internal/application/settings"
	smssvc "advisor/internal/application/sms"
	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/infrastructure/auth"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/id"
	"advisor/internal/infrastructure/index/sqlite"
	"advisor/internal/infrastructure/nbrb"
	"advisor/internal/infrastructure/nopvault"
	apihttp "advisor/internal/transport/http"
)

func main() {
	cfg := loadConfig()
	if err := run(cfg); err != nil {
		log.Fatalf("advisord: %v", err)
	}
}

type config struct {
	addr   string
	dbPath string
	token  string
	cors   string
	webDir string
}

func loadConfig() config {
	return config{
		addr:   env("ADVISOR_ADDR", ":8080"),
		dbPath: env("ADVISOR_DB", defaultDB()),
		token:  os.Getenv("ADVISOR_TOKEN"),
		cors:   os.Getenv("ADVISOR_CORS"),
		webDir: os.Getenv("ADVISOR_WEB"), // путь к собранному web/dist; пусто => только API
	}
}

func run(cfg config) error {
	verifier := auth.New(cfg.token)
	if !verifier.Enabled() {
		return errors.New("не задан ADVISOR_TOKEN — нельзя поднимать открытый API")
	}

	if err := os.MkdirAll(filepath.Dir(cfg.dbPath), 0o755); err != nil {
		return err
	}
	// Источник правды — серверная БД; vault-операции репозиториев — no-op.
	idx, err := sqlite.Open(cfg.dbPath, nopvault.New())
	if err != nil {
		return err
	}
	defer func() { _ = idx.Close() }()

	if err := idx.SeedCurrencies(); err != nil {
		return err
	}

	sysClock := clock.New()
	idGen := id.New()
	currency := currencysvc.New(idx.Rates(), nbrb.New())
	accounts := accountsvc.New(idx.Users(), idx.Sessions(), sysClock, idGen)

	// Глобальные сервисы + фабрика user-scoped сервисов (данные пользователя по owner_id).
	g := apihttp.Global{
		Accounts: accounts,
		Currency: currency,
		Clock:    sysClock,
		ForUser: func(uid string) apihttp.UserServices {
			cats := idx.Categories(uid)
			txs := idx.Transactions(uid)
			userLedger := ledgersvc.New(txs, cats, currency, sysClock, idGen)
			return apihttp.UserServices{
				Catalog:   catalogsvc.New(cats, sysClock, idGen),
				Ledger:    userLedger,
				Planning:  planningsvc.New(idx.Plans(uid), txs, cats, currency, sysClock, idGen),
				Recurring: recurringsvc.New(idx.Recurring(uid), idx.Plans(uid), txs, sysClock, idGen),
				Reporting: reportingsvc.New(txs, currency),
				Settings:  settingssvc.New(idx.Settings(uid), idx.Currencies()),
				IO:        iosvc.New(cats, txs, idx.Plans(uid), idx.Recurring(uid)),
				// Шаблоны — глобальные (админ-пресеты для всех); правила и черновики — персональные.
				SMS: smssvc.New(idx.SMSTemplates(), idx.Drafts(uid), idx.Rules(uid), userLedger, sysClock, idGen),
			}
		},
	}

	// Глобальный пресет «Приорбанк» — если шаблонов ещё нет.
	seedPriorTemplate(idx.SMSTemplates(), sysClock, idGen)

	api := apihttp.NewServer(g, verifier, cfg.cors, cfg.webDir)
	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           api,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("advisord слушает %s (БД: %s)", cfg.addr, cfg.dbPath)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-stop:
		log.Println("advisord: завершение…")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

// seedPriorTemplate создаёт глобальный пресет разбора SMS Приорбанка, если
// шаблонов ещё нет. Захватывает сумму(1), валюту(2) и продавца(3); категорию
// не задаёт — категоризация идёт правилами «продавец → категория».
func seedPriorTemplate(templates smssvc.TemplateRepo, clk ports.Clock, ids ports.IDGenerator) {
	existing, err := templates.List()
	if err != nil || len(existing) > 0 {
		return
	}
	_ = templates.Save(&smssvc.Template{
		ID:            ids.NewID(),
		Name:          "Приорбанк",
		Sender:        "Priorbank",
		Pattern:       `(?:Oplata|Perevod|Snyatie) ([0-9]+[.,][0-9]{2}) ([A-Z]{3})(?:\. BLR (.+?)\. Balance)?`,
		AmountGroup:   1,
		CurrencyGroup: 2,
		MerchantGroup: 3,
		FixedCurrency: "BYN",
		Type:          core.Expense,
		Enabled:       true,
		CreatedAt:     clk.Now().UTC(),
	})
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func defaultDB() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		cfg = "."
	}
	return filepath.Join(cfg, "advisor", "server.db")
}
