package ledger

import (
	"errors"
	"testing"
	"time"

	"advisor/internal/application/currency"
	"advisor/internal/application/ports"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/memory"
)

func setup(t *testing.T) (*Service, *memory.Store, *category.Category, *category.Category) {
	t.Helper()
	store := memory.NewStore()
	clk := clock.Fixed{T: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)}
	ids := memory.NewSeqIDs()
	cur := currency.New(store.Rates(), nil)

	expCat, _ := category.New(ids.NewID(), "Еда", core.Expense, clk.T)
	incCat, _ := category.New(ids.NewID(), "Зарплата", core.Income, clk.T)
	_ = store.Categories().Save(expCat)
	_ = store.Categories().Save(incCat)

	svc := New(store.Transactions(), store.Categories(), cur, clk, ids)
	return svc, store, expCat, incCat
}

func date(y, m, d int) core.Date { return core.Date{Year: y, Month: m, Day: d} }

func TestAddAndBalance(t *testing.T) {
	svc, store, expCat, incCat := setup(t)
	// Курс USD для дохода в валюте.
	_ = store.Rates().SaveRate(ports.Rate{Currency: "USD", Date: date(2026, 7, 10), Scale: 1, RateBYNScaled: 30000}) // 3.0000

	if _, err := svc.Add(core.Expense, date(2026, 7, 5), expCat.Meta.ID, money.MustNew(5000, "BYN"), "продукты"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Add(core.Income, date(2026, 7, 10), incCat.Meta.ID, money.MustNew(10000, "USD"), "зп"); err != nil {
		t.Fatal(err)
	}

	bal, err := svc.MonthBalance(core.YearMonth{Year: 2026, Month: 7})
	if err != nil {
		t.Fatal(err)
	}
	// Доход: 100.00 USD * 3.0 = 300.00 BYN = 30000 minor.
	if bal.FactIncome.Minor() != 30000 {
		t.Errorf("income = %d, want 30000", bal.FactIncome.Minor())
	}
	if bal.FactExpense.Minor() != 5000 {
		t.Errorf("expense = %d, want 5000", bal.FactExpense.Minor())
	}
	if bal.Remainder().Minor() != 25000 {
		t.Errorf("remainder = %d, want 25000", bal.Remainder().Minor())
	}
}

func TestAddTypeMismatch(t *testing.T) {
	svc, _, expCat, _ := setup(t)
	// Доход на расходную категорию — запрещено.
	_, err := svc.Add(core.Income, date(2026, 7, 5), expCat.Meta.ID, money.MustNew(100, "BYN"), "")
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("expected ErrTypeMismatch, got %v", err)
	}
}

func TestAddArchivedCategory(t *testing.T) {
	svc, store, expCat, _ := setup(t)
	_ = expCat.Archive(time.Now())
	_ = store.Categories().Save(expCat)
	_, err := svc.Add(core.Expense, date(2026, 7, 5), expCat.Meta.ID, money.MustNew(100, "BYN"), "")
	if !errors.Is(err, ErrCategoryArchived) {
		t.Errorf("expected ErrCategoryArchived, got %v", err)
	}
}

func TestAddFromRecurring(t *testing.T) {
	svc, _, expCat, _ := setup(t)
	tx, err := svc.AddFromRecurring(core.Expense, date(2026, 7, 5), expCat.Meta.ID, money.MustNew(8000, "BYN"), "аренда", "rec-1")
	if err != nil {
		t.Fatal(err)
	}
	if tx.RecurringID != "rec-1" {
		t.Errorf("recurring id = %q, want rec-1", tx.RecurringID)
	}
	got, _ := svc.Get(tx.Meta.ID)
	if got.RecurringID != "rec-1" {
		t.Error("recurring id not persisted")
	}
}

func TestListMonth(t *testing.T) {
	svc, _, expCat, _ := setup(t)
	_, _ = svc.Add(core.Expense, date(2026, 7, 5), expCat.Meta.ID, money.MustNew(100, "BYN"), "")
	_, _ = svc.Add(core.Expense, date(2026, 7, 6), expCat.Meta.ID, money.MustNew(200, "BYN"), "")
	list, err := svc.ListMonth(core.YearMonth{Year: 2026, Month: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("month list = %d, want 2", len(list))
	}
}

func TestEditAndDelete(t *testing.T) {
	svc, _, expCat, _ := setup(t)
	tx, err := svc.Add(core.Expense, date(2026, 7, 5), expCat.Meta.ID, money.MustNew(5000, "BYN"), "x")
	if err != nil {
		t.Fatal(err)
	}
	edited, err := svc.Edit(tx.Meta.ID, date(2026, 7, 6), expCat.Meta.ID, money.MustNew(7000, "BYN"), "y")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Amount.Minor() != 7000 || edited.Meta.Rev != 2 {
		t.Errorf("edit failed: amount=%d rev=%d", edited.Amount.Minor(), edited.Meta.Rev)
	}
	if err := svc.Delete(tx.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(tx.Meta.ID); err == nil {
		t.Error("deleted tx must not be found")
	}
}
