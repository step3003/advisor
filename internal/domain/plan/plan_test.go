package plan

import (
	"testing"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

var now = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func TestNewPlanItem(t *testing.T) {
	ym := core.YearMonth{Year: 2026, Month: 7}
	amt := money.MustNew(50000, "BYN")
	pi, err := New("plan-1", ym, "cat-1", amt, "продукты", now)
	if err != nil {
		t.Fatal(err)
	}
	if pi.Amount.Minor() != 50000 {
		t.Errorf("amount = %d", pi.Amount.Minor())
	}
	key := pi.UniqueKey()
	if key.CategoryID != "cat-1" || key.Currency != "BYN" || !key.Period.Equal(ym) {
		t.Errorf("key = %+v", key)
	}
}

func TestPlanValidation(t *testing.T) {
	ym := core.YearMonth{Year: 2026, Month: 7}
	if _, err := New("id", ym, "", money.MustNew(1, "BYN"), "", now); err == nil {
		t.Error("empty category must error")
	}
	if _, err := New("id", ym, "cat", money.MustNew(0, "BYN"), "", now); err == nil {
		t.Error("zero amount must error")
	}
}

func TestSetAmount(t *testing.T) {
	ym := core.YearMonth{Year: 2026, Month: 7}
	pi, _ := New("plan-1", ym, "cat-1", money.MustNew(50000, "BYN"), "", now)
	if err := pi.SetAmount(money.MustNew(60000, "BYN"), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if pi.Amount.Minor() != 60000 {
		t.Errorf("amount = %d", pi.Amount.Minor())
	}
	if pi.Meta.Rev != 2 {
		t.Errorf("rev = %d", pi.Meta.Rev)
	}
	if err := pi.SetAmount(money.MustNew(-1, "BYN"), now); err == nil {
		t.Error("negative must error")
	}
}
