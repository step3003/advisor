package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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
	"advisor/internal/infrastructure/auth"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/id"
	"advisor/internal/infrastructure/index/sqlite"
	"advisor/internal/infrastructure/nbrb"
	"advisor/internal/infrastructure/nopvault"
)

const testToken = "test-token-123"

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	idx, err := sqlite.Open(dbPath, nopvault.New())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	if err := idx.SeedCurrencies(); err != nil {
		t.Fatalf("seed currencies: %v", err)
	}
	sysClock := clock.New()
	idGen := id.New()
	currency := currencysvc.New(idx.Rates(), nbrb.New())
	ledger := ledgersvc.New(idx.Transactions(), idx.Categories(), currency, sysClock, idGen)
	svc := Services{
		Catalog:   catalogsvc.New(idx.Categories(), sysClock, idGen),
		Ledger:    ledger,
		Planning:  planningsvc.New(idx.Plans(), idx.Transactions(), idx.Categories(), currency, sysClock, idGen),
		Recurring: recurringsvc.New(idx.Recurring(), idx.Plans(), idx.Transactions(), sysClock, idGen),
		Reporting: reportingsvc.New(idx.Transactions(), currency),
		Settings:  settingssvc.New(idx.Settings(), idx.Currencies()),
		IO:        iosvc.New(idx.Categories(), idx.Transactions(), idx.Plans(), idx.Recurring()),
		SMS:       smssvc.New(idx.SMSTemplates(), idx.Drafts(), idx.Rules(), ledger, sysClock, idGen),
		Accounts:  accountsvc.New(idx.Users(), idx.Sessions(), sysClock, idGen),
		Currency:  currency,
		Clock:     sysClock,
	}
	if _, err := svc.Catalog.SeedDefaults(); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	return NewServer(svc, auth.New(testToken), "", "")
}

// do выполняет запрос к API с токеном (если withAuth) и декодирует JSON-ответ.
func do(t *testing.T, s *Server, method, path string, body any, withAuth bool) (*httptest.ResponseRecorder, []byte) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if withAuth {
		req.Header.Set("Authorization", "Bearer "+testToken)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec, rec.Body.Bytes()
}

func TestHealthNoAuth(t *testing.T) {
	s := newTestServer(t)
	rec, _ := do(t, s, http.MethodGet, "/api/health", nil, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("health: want 200, got %d", rec.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	s := newTestServer(t)
	rec, _ := do(t, s, http.MethodGet, "/api/categories", nil, false)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", rec.Code)
	}
	rec2, _ := do(t, s, http.MethodGet, "/api/categories", nil, true)
	if rec2.Code != http.StatusOK {
		t.Fatalf("with token: want 200, got %d", rec2.Code)
	}
}

func TestSeededCategories(t *testing.T) {
	s := newTestServer(t)
	rec, body := do(t, s, http.MethodGet, "/api/categories", nil, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var cats []categoryDTO
	if err := json.Unmarshal(body, &cats); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cats) < 20 {
		t.Fatalf("ожидались предустановленные категории, got %d", len(cats))
	}
}

func TestFullFlow(t *testing.T) {
	s := newTestServer(t)

	// 1. Создать категорию расхода.
	rec, body := do(t, s, http.MethodPost, "/api/categories",
		createCategoryReq{Name: "Тестовая", Type: "expense"}, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: %d — %s", rec.Code, body)
	}
	var cat categoryDTO
	mustJSON(t, body, &cat)
	if cat.ID == "" {
		t.Fatal("нет id категории")
	}

	// 2. Задать план на месяц.
	rec, body = do(t, s, http.MethodPut, "/api/plans", setPlanReq{
		Period: "2026-07", CategoryID: cat.ID,
		Amount: moneyDTO{Amount: "300.00", Currency: "BYN"},
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("set plan: %d — %s", rec.Code, body)
	}

	// 3. Внести факт.
	rec, body = do(t, s, http.MethodPost, "/api/transactions", txReq{
		Date: "2026-07-15", Type: "expense", CategoryID: cat.ID,
		Amount: moneyDTO{Amount: "42.50", Currency: "BYN"}, Note: "обед",
	}, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tx: %d — %s", rec.Code, body)
	}
	var tx transactionDTO
	mustJSON(t, body, &tx)

	// 4. Список операций месяца содержит внесённую.
	rec, body = do(t, s, http.MethodGet, "/api/transactions?ym=2026-07", nil, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tx: %d", rec.Code)
	}
	var txs []transactionDTO
	mustJSON(t, body, &txs)
	if len(txs) != 1 || txs[0].Amount.Amount != "42.50" {
		t.Fatalf("ожидалась 1 операция 42.50, got %+v", txs)
	}

	// 5. План/факт: план 300, факт 42.50.
	rec, body = do(t, s, http.MethodGet, "/api/reports/plan-vs-fact?ym=2026-07", nil, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan-vs-fact: %d — %s", rec.Code, body)
	}
	var pvf planVsFactDTO
	mustJSON(t, body, &pvf)
	if pvf.PlanExpense.Amount != "300.00" {
		t.Fatalf("план расхода: want 300.00, got %s", pvf.PlanExpense.Amount)
	}
	if pvf.FactExpense.Amount != "42.50" {
		t.Fatalf("факт расхода: want 42.50, got %s", pvf.FactExpense.Amount)
	}

	// 6. Удалить операцию.
	rec, _ = do(t, s, http.MethodDelete, "/api/transactions/"+tx.ID, nil, true)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete tx: want 204, got %d", rec.Code)
	}
}

func TestDeleteCategoryWithRefsBlocked(t *testing.T) {
	s := newTestServer(t)
	// Создать категорию и операцию по ней.
	_, body := do(t, s, http.MethodPost, "/api/categories",
		createCategoryReq{Name: "СРефами", Type: "expense"}, true)
	var cat categoryDTO
	mustJSON(t, body, &cat)
	do(t, s, http.MethodPost, "/api/transactions", txReq{
		Date: "2026-07-15", Type: "expense", CategoryID: cat.ID,
		Amount: moneyDTO{Amount: "10.00", Currency: "BYN"},
	}, true)
	// Жёсткое удаление должно быть заблокировано (409).
	rec, _ := do(t, s, http.MethodDelete, "/api/categories/"+cat.ID, nil, true)
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete with refs: want 409, got %d", rec.Code)
	}
}

func mustJSON(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode json: %v — %s", err, data)
	}
}
