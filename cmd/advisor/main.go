// Command advisor — точка входа приложения.
//
// main выполняет ручную сборку зависимостей (DI): инициализирует хранилище-vault
// и локальный SQLite-индекс, разрешает конфликты iCloud, пересобирает индекс из
// vault, засевает предустановленные категории, генерирует плановые позиции из
// повторяющихся шаблонов на текущий месяц и запускает UI (Fyne).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	catalogsvc "advisor/internal/application/catalog"
	currencysvc "advisor/internal/application/currency"
	iosvc "advisor/internal/application/io"
	ledgersvc "advisor/internal/application/ledger"
	planningsvc "advisor/internal/application/planning"
	recurringsvc "advisor/internal/application/recurring"
	reportingsvc "advisor/internal/application/reporting"
	settingssvc "advisor/internal/application/settings"
	"advisor/internal/domain/core"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/id"
	"advisor/internal/infrastructure/index/sqlite"
	"advisor/internal/infrastructure/nbrb"
	"advisor/internal/infrastructure/vault"
	fyneapp "advisor/internal/presentation/app"
	"advisor/internal/presentation/screens"
)

func main() {
	defaultVault, defaultDB := defaultPaths()
	vaultDir := flag.String("vault", defaultVault, "путь к папке-хранилищу (vault, источник правды)")
	dbPath := flag.String("db", defaultDB, "путь к локальному SQLite-индексу")
	flag.Parse()

	if err := run(*vaultDir, *dbPath); err != nil {
		log.Fatalf("advisor: %v", err)
	}
}

func run(vaultDir, dbPath string) error {
	// --- Инфраструктура ---
	// Примечание: реальный iCloud-контейнер (iOS/macOS) подключается на этапе
	// сборки платформы; здесь путь к vault — обычный параметр, домен об этом не знает.
	store, err := vault.New(vaultDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	idx, err := sqlite.Open(dbPath, store)
	if err != nil {
		return err
	}
	defer func() { _ = idx.Close() }()

	if err := idx.SeedCurrencies(); err != nil {
		return err
	}

	sysClock := clock.New()
	idGen := id.New()
	provider := nbrb.New() // сеть не вызывается, пока пользователь не обновит курсы

	// --- Application (DI) ---
	currency := currencysvc.New(idx.Rates(), provider)
	catalog := catalogsvc.New(idx.Categories(), sysClock, idGen)
	ledger := ledgersvc.New(idx.Transactions(), idx.Categories(), currency, sysClock, idGen)
	planning := planningsvc.New(idx.Plans(), idx.Transactions(), idx.Categories(), currency, sysClock, idGen)
	recurring := recurringsvc.New(idx.Recurring(), idx.Plans(), idx.Transactions(), sysClock, idGen)
	reporting := reportingsvc.New(idx.Transactions(), currency)
	settings := settingssvc.New(idx.Settings(), idx.Currencies())
	exportImport := iosvc.New(idx.Categories(), idx.Transactions(), idx.Plans(), idx.Recurring())

	// initSync выполняет разрешение конфликтов iCloud, пересборку индекса из
	// vault, сидинг категорий и генерацию повторяющихся на текущий месяц.
	// Используется на старте и кнопкой «Пересканировать» в настройках.
	initSync := func() (screens.SyncInfo, error) {
		conflicts, err := store.ResolveConflicts()
		if err != nil {
			return screens.SyncInfo{}, err
		}
		stats, err := idx.RebuildFromVault()
		if err != nil {
			return screens.SyncInfo{}, err
		}
		if _, err := catalog.SeedDefaults(); err != nil {
			return screens.SyncInfo{}, err
		}
		// Идемпотентная подстановка повторяющихся трат в план текущего месяца (FR-REC-2).
		if _, err := recurring.GenerateForMonth(core.YearMonthOf(sysClock.Now())); err != nil {
			log.Printf("recurring generate: %v", err)
		}
		cats, err := catalog.List(true)
		if err != nil {
			return screens.SyncInfo{}, err
		}
		return screens.SyncInfo{
			VaultPath:    store.Path(),
			Categories:   len(cats),
			Transactions: stats.Transactions,
			Plans:        stats.Plans,
			Recurring:    stats.Recurring,
			Errors:       len(stats.Errors),
			Conflicts:    len(conflicts),
			LastRun:      sysClock.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	sync, err := initSync()
	if err != nil {
		return err
	}
	logStartup(dbPath, idx, sync)

	// --- Presentation ---
	deps := &screens.Deps{
		Catalog:   catalog,
		Ledger:    ledger,
		Planning:  planning,
		Recurring: recurring,
		Reporting: reporting,
		Currency:  currency,
		Settings:  settings,
		IO:        exportImport,
		Now:       func() time.Time { return sysClock.Now() },
		Sync:      sync,
		Rescan:    initSync,
	}
	fyneapp.Run(deps)
	return nil
}

func logStartup(dbPath string, idx *sqlite.Index, sync screens.SyncInfo) {
	schemaVer, _ := idx.SchemaVersion()
	fmt.Println("Advisor запускается…")
	fmt.Printf("  Vault (источник правды): %s\n", sync.VaultPath)
	fmt.Printf("  SQLite-индекс:           %s (схема v%d)\n", dbPath, schemaVer)
	fmt.Printf("  Категорий=%d, транзакций=%d, планов=%d, шаблонов=%d\n",
		sync.Categories, sync.Transactions, sync.Plans, sync.Recurring)
	fmt.Printf("  Конфликты iCloud: %d, ошибок пересборки: %d\n", sync.Conflicts, sync.Errors)
}

// defaultPaths возвращает пути по умолчанию для vault и индекса.
func defaultPaths() (vaultDir, dbPath string) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	vaultDir = filepath.Join(home, "Advisor")
	cfg, err := os.UserConfigDir()
	if err != nil {
		cfg = filepath.Join(home, ".config")
	}
	dbPath = filepath.Join(cfg, "advisor", "index.db")
	return vaultDir, dbPath
}
