package transaction

import (
	"testing"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

var now = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func mustDate(y, m, d int) core.Date {
	dt, err := core.NewDate(y, m, d)
	if err != nil {
		panic(err)
	}
	return dt
}

func TestNewTransaction(t *testing.T) {
	amt := money.MustNew(4250, "USD")
	tx, err := New("tx-1", core.Expense, mustDate(2026, 7, 18), "cat-1", amt, "обед", now)
	if err != nil {
		t.Fatal(err)
	}
	if !tx.IsExpense() {
		t.Error("must be expense")
	}
	if tx.Amount.Minor() != 4250 {
		t.Errorf("amount = %d", tx.Amount.Minor())
	}
	if tx.Meta.Rev != 1 {
		t.Errorf("rev = %d", tx.Meta.Rev)
	}
}

func TestNewValidation(t *testing.T) {
	amt := money.MustNew(4250, "USD")
	neg := money.MustNew(-1, "USD")
	zero := money.MustNew(0, "USD")

	if _, err := New("id", core.EntryType("bad"), mustDate(2026, 7, 18), "cat", amt, "", now); err == nil {
		t.Error("bad type must error")
	}
	if _, err := New("id", core.Expense, mustDate(2026, 7, 18), "", amt, "", now); err == nil {
		t.Error("empty category must error")
	}
	if _, err := New("id", core.Expense, core.Date{}, "cat", amt, "", now); err == nil {
		t.Error("zero date must error")
	}
	if _, err := New("id", core.Expense, mustDate(2026, 7, 18), "cat", neg, "", now); err == nil {
		t.Error("negative amount must error")
	}
	if _, err := New("id", core.Expense, mustDate(2026, 7, 18), "cat", zero, "", now); err == nil {
		t.Error("zero amount must error")
	}
}

func TestEdit(t *testing.T) {
	amt := money.MustNew(4250, "USD")
	tx, _ := New("tx-1", core.Expense, mustDate(2026, 7, 18), "cat-1", amt, "обед", now)
	newAmt := money.MustNew(5000, "EUR")
	if err := tx.Edit(mustDate(2026, 7, 19), "cat-2", newAmt, "ужин", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if tx.CategoryID != "cat-2" || tx.Amount.Currency() != "EUR" || tx.Note != "ужин" {
		t.Errorf("edit not applied: %+v", tx)
	}
	if tx.Meta.Rev != 2 {
		t.Errorf("rev = %d, want 2", tx.Meta.Rev)
	}
}
