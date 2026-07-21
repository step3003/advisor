package sqlite

import (
	"path/filepath"
	"testing"
	"time"

	"advisor/internal/application/ports"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
	"advisor/internal/infrastructure/vault"
)

var now = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func rateOf(cur money.Currency, d core.Date, scale, scaled int64) ports.Rate {
	return ports.Rate{Currency: cur, Date: d, Scale: scale, RateBYNScaled: scaled}
}

func openIndex(t *testing.T, vaultDir, dbPath string) *Index {
	t.Helper()
	v, err := vault.New(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := Open(dbPath, v)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	return idx
}

func seed(t *testing.T, idx *Index) (catID, txID, planID, recID string) {
	t.Helper()
	cat, _ := category.New("00000000-0000-4000-8000-000000000001", "Еда", core.Expense, now)
	if err := idx.Categories("").Save(cat); err != nil {
		t.Fatal(err)
	}
	d, _ := core.NewDate(2026, 7, 5)
	tx, _ := transaction.New("00000000-0000-4000-8000-000000000002", core.Expense, d, cat.Meta.ID, money.MustNew(4250, "USD"), "обед", now)
	if err := idx.Transactions("").Save(tx); err != nil {
		t.Fatal(err)
	}
	pi, _ := plan.New("00000000-0000-4000-8000-000000000003", core.YearMonth{Year: 2026, Month: 7}, cat.Meta.ID, money.MustNew(50000, "BYN"), "", now)
	if err := idx.Plans("").Save(pi); err != nil {
		t.Fatal(err)
	}
	start, _ := core.NewDate(2026, 1, 1)
	tpl, _ := recurring.New("00000000-0000-4000-8000-000000000004", core.Expense, cat.Meta.ID, money.MustNew(80000, "BYN"), 5, start, nil, false, now)
	if err := idx.Recurring("").Save(tpl); err != nil {
		t.Fatal(err)
	}
	return cat.Meta.ID, tx.Meta.ID, pi.Meta.ID, tpl.Meta.ID
}

func TestWriteThroughAndQuery(t *testing.T) {
	dir := t.TempDir()
	idx := openIndex(t, filepath.Join(dir, "vault"), filepath.Join(dir, "index.db"))
	catID, txID, _, _ := seed(t, idx)

	got, err := idx.Transactions("").Get(txID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount.Decimal() != "42.50" || got.Amount.Currency() != "USD" {
		t.Errorf("amount = %s %s", got.Amount.Decimal(), got.Amount.Currency())
	}
	txs, _ := idx.Transactions("").ListByMonth(core.YearMonth{Year: 2026, Month: 7})
	if len(txs) != 1 {
		t.Errorf("month txs = %d, want 1", len(txs))
	}
	// FindByKey для плана.
	pi, err := idx.Plans("").FindByKey(plan.Key{Period: core.YearMonth{Year: 2026, Month: 7}, CategoryID: catID, Currency: "BYN"})
	if err != nil {
		t.Fatalf("FindByKey: %v", err)
	}
	if pi.Amount.Minor() != 50000 {
		t.Errorf("plan amount = %d", pi.Amount.Minor())
	}
	// HasReferences.
	has, _ := idx.Categories("").HasReferences(catID)
	if !has {
		t.Error("category must have references")
	}
}

// TestRebuildFromVault проверяет критерий приёмки 6: удалили индекс — приложение
// восстановило его из файлов vault без потерь.
func TestRebuildFromVault(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")

	// Первый индекс: наполняем данными (они пишутся в vault).
	idx1 := openIndex(t, vaultDir, filepath.Join(dir, "index1.db"))
	catID, txID, planID, recID := seed(t, idx1)
	_ = idx1.Close()

	// Полностью новый индекс (другой файл БД) над тем же vault.
	idx2 := openIndex(t, vaultDir, filepath.Join(dir, "index2.db"))
	stats, err := idx2.RebuildFromVault()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Errors) != 0 {
		t.Fatalf("rebuild errors: %v", stats.Errors)
	}
	if stats.Categories != 1 || stats.Transactions != 1 || stats.Plans != 1 || stats.Recurring != 1 {
		t.Fatalf("rebuild stats = %+v, want 1/1/1/1", stats)
	}

	// Данные восстановлены и совпадают.
	if c, err := idx2.Categories("").Get(catID); err != nil || c.Name != "Еда" {
		t.Errorf("category not restored: %v", err)
	}
	if tx, err := idx2.Transactions("").Get(txID); err != nil || tx.Amount.Decimal() != "42.50" {
		t.Errorf("transaction not restored: %v", err)
	}
	if p, err := idx2.Plans("").Get(planID); err != nil || p.Amount.Minor() != 50000 {
		t.Errorf("plan not restored: %v", err)
	}
	if r, err := idx2.Recurring("").Get(recID); err != nil || r.DayOfMonth != 5 {
		t.Errorf("recurring not restored: %v", err)
	}
}

// TestRebuildWithSubcategories защищает от регрессии: файлы vault читаются в
// произвольном порядке, поэтому подкатегория может встретиться раньше родителя.
func TestRebuildWithSubcategories(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	idx1 := openIndex(t, vaultDir, filepath.Join(dir, "index1.db"))

	// Создаём много пар родитель/ребёнок, чтобы порядок листинга гарантированно
	// иногда ставил ребёнка раньше родителя.
	for i := 0; i < 10; i++ {
		p, _ := category.New(uuidN(2*i+1), "Родитель", core.Expense, now)
		if err := idx1.Categories("").Save(p); err != nil {
			t.Fatal(err)
		}
		c, _ := category.NewSub(uuidN(2*i+2), "Ребёнок", core.Expense, p.Meta.ID, now)
		if err := idx1.Categories("").Save(c); err != nil {
			t.Fatal(err)
		}
	}
	_ = idx1.Close()

	idx2 := openIndex(t, vaultDir, filepath.Join(dir, "index2.db"))
	stats, err := idx2.RebuildFromVault()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Errors) != 0 {
		t.Fatalf("rebuild errors (FK ordering?): %v", stats.Errors)
	}
	if stats.Categories != 20 {
		t.Errorf("rebuilt categories = %d, want 20", stats.Categories)
	}
}

func uuidN(n int) string {
	// Каноничный 36-символьный UUID с номером в хвосте.
	return "00000000-0000-4000-8000-" + pad12(n)
}

func pad12(n int) string {
	s := ""
	for i := 0; i < 12; i++ {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func TestMigrationsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index.db")
	vaultDir := filepath.Join(dir, "vault")
	latest := len(migrations) // ожидаемая последняя версия схемы
	idx := openIndex(t, vaultDir, dbPath)
	v1, _ := idx.SchemaVersion()
	if v1 != latest {
		t.Errorf("schema version = %d, want %d", v1, latest)
	}
	_ = idx.Close()
	// Повторное открытие не должно ломать миграции.
	idx2 := openIndex(t, vaultDir, dbPath)
	v2, _ := idx2.SchemaVersion()
	if v2 != latest {
		t.Errorf("schema version after reopen = %d, want %d", v2, latest)
	}
}

func TestRateCache(t *testing.T) {
	dir := t.TempDir()
	idx := openIndex(t, filepath.Join(dir, "vault"), filepath.Join(dir, "index.db"))
	rates := idx.Rates()

	d1, _ := core.NewDate(2026, 7, 10)
	d2, _ := core.NewDate(2026, 7, 15)
	_ = rates.SaveRate(rateOf("USD", d1, 1, 32000))
	_ = rates.SaveRate(rateOf("USD", d2, 1, 33000))

	// Точный курс.
	if r, ok, _ := rates.GetRate("USD", d2); !ok || r.RateBYNScaled != 33000 {
		t.Errorf("exact rate = %+v ok=%v", r, ok)
	}
	// Последний ≤ даты (fallback).
	d3, _ := core.NewDate(2026, 7, 20)
	r, ok, _ := rates.GetLatestBefore("USD", d3)
	if !ok || r.Date.String() != "2026-07-15" {
		t.Errorf("latest-before = %+v ok=%v", r, ok)
	}
}
