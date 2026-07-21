package planning

import (
	"testing"
	"time"

	"advisor/internal/application/currency"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/transaction"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/memory"
)

func txNew(id string, typ core.EntryType, on core.Date, catID string, amount money.Money, now time.Time) (*transaction.Transaction, error) {
	return transaction.New(id, typ, on, catID, amount, "", now)
}

func setup(t *testing.T) (*Service, *memory.Store, string) {
	t.Helper()
	store := memory.NewStore()
	clk := clock.Fixed{T: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)}
	ids := memory.NewSeqIDs()
	cur := currency.New(store.Rates(), nil)
	cat, _ := category.New(ids.NewID(), "Еда", core.Expense, clk.T)
	_ = store.Categories().Save(cat)
	svc := New(store.Plans(), store.Transactions(), store.Categories(), cur, clk, ids)
	return svc, store, cat.Meta.ID
}

func TestSetPlanUpsert(t *testing.T) {
	svc, _, catID := setup(t)
	ym := core.YearMonth{Year: 2026, Month: 7}

	first, err := svc.SetPlan(ym, catID, money.MustNew(50000, "BYN"), "план")
	if err != nil {
		t.Fatal(err)
	}
	// Повторный вызов по той же паре (категория, валюта) => обновление, не дубль.
	second, err := svc.SetPlan(ym, catID, money.MustNew(60000, "BYN"), "новый план")
	if err != nil {
		t.Fatal(err)
	}
	if first.Meta.ID != second.Meta.ID {
		t.Error("upsert must reuse the same plan item")
	}
	if second.Amount.Minor() != 60000 {
		t.Errorf("amount = %d, want 60000", second.Amount.Minor())
	}

	items, _ := svc.ListMonth(ym)
	if len(items) != 1 {
		t.Errorf("expected 1 plan item, got %d", len(items))
	}
}

func TestCopyFromPreviousMonth(t *testing.T) {
	svc, _, catID := setup(t)
	june := core.YearMonth{Year: 2026, Month: 6}
	july := core.YearMonth{Year: 2026, Month: 7}

	if _, err := svc.SetPlan(june, catID, money.MustNew(50000, "BYN"), "июнь"); err != nil {
		t.Fatal(err)
	}
	n, err := svc.CopyFromPreviousMonth(july)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("copied = %d, want 1", n)
	}
	// Идемпотентность: повторное копирование не дублирует.
	n2, _ := svc.CopyFromPreviousMonth(july)
	if n2 != 0 {
		t.Errorf("second copy = %d, want 0", n2)
	}
	items, _ := svc.ListMonth(july)
	if len(items) != 1 {
		t.Errorf("july items = %d, want 1", len(items))
	}
}

func TestPlanVsFact(t *testing.T) {
	svc, store, catID := setup(t)
	ym := core.YearMonth{Year: 2026, Month: 7}
	clk := clock.Fixed{T: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)}
	ids := memory.NewSeqIDs()

	if _, err := svc.SetPlan(ym, catID, money.MustNew(50000, "BYN"), ""); err != nil {
		t.Fatal(err)
	}
	// Факт 600.00 BYN через прямое сохранение транзакции.
	tx, _ := txNew(ids.NewID(), core.Expense, core.Date{Year: 2026, Month: 7, Day: 15}, catID, money.MustNew(60000, "BYN"), clk.T)
	_ = store.Transactions().Save(tx)

	pvf, err := svc.PlanVsFact(ym)
	if err != nil {
		t.Fatal(err)
	}
	if pvf.PlanExpense.Minor() != 50000 {
		t.Errorf("plan expense = %d, want 50000", pvf.PlanExpense.Minor())
	}
	if pvf.FactExpense.Minor() != 60000 {
		t.Errorf("fact expense = %d, want 60000", pvf.FactExpense.Minor())
	}
	if len(pvf.Lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(pvf.Lines))
	}
	line := pvf.Lines[0]
	if line.Deviation().Minor() != 10000 {
		t.Errorf("deviation = %d, want 10000", line.Deviation().Minor())
	}
	if !line.IsOverspent() {
		t.Error("must be overspent")
	}
}
