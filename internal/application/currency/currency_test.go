package currency

import (
	"errors"
	"testing"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/infrastructure/memory"
)

func date(y, m, d int) core.Date { return core.Date{Year: y, Month: m, Day: d} }

func TestToBaseBYNPassthrough(t *testing.T) {
	store := memory.NewStore()
	svc := New(store.Rates(), nil)
	amt := money.MustNew(10000, money.BaseCurrency)
	conv, err := svc.ToBase(amt, date(2026, 7, 18))
	if err != nil {
		t.Fatal(err)
	}
	if !conv.Amount.Equal(amt) || conv.Approximate {
		t.Errorf("BYN passthrough failed: %+v", conv)
	}
}

func TestToBaseExactRate(t *testing.T) {
	store := memory.NewStore()
	cache := store.Rates()
	// 1 USD = 3.2567 BYN.
	_ = cache.SaveRate(ports.Rate{Currency: "USD", Date: date(2026, 7, 18), Scale: 1, RateBYNScaled: 32567})
	svc := New(cache, nil)

	// 42.50 USD => 4250 * 32567 / 10000 = 13840.975 => 13841 (138.41 BYN).
	conv, err := svc.ToBase(money.MustNew(4250, "USD"), date(2026, 7, 18))
	if err != nil {
		t.Fatal(err)
	}
	if conv.Amount.Minor() != 13841 {
		t.Errorf("converted = %d (%s), want 13841", conv.Amount.Minor(), conv.Amount.Decimal())
	}
	if conv.Approximate {
		t.Error("exact rate must not be approximate")
	}
	if conv.Amount.Currency() != money.BaseCurrency {
		t.Errorf("currency = %s", conv.Amount.Currency())
	}
}

func TestToBaseFallback(t *testing.T) {
	store := memory.NewStore()
	cache := store.Rates()
	_ = cache.SaveRate(ports.Rate{Currency: "USD", Date: date(2026, 7, 18), Scale: 1, RateBYNScaled: 32567})
	svc := New(cache, nil)

	// Курса на 20-е нет — берётся последний ≤ (18-е), результат приблизительный.
	conv, err := svc.ToBase(money.MustNew(4250, "USD"), date(2026, 7, 20))
	if err != nil {
		t.Fatal(err)
	}
	if !conv.Approximate {
		t.Error("fallback must be marked approximate (FR-CUR-4)")
	}
	if conv.RateDate.String() != "2026-07-18" {
		t.Errorf("rate date = %s, want 2026-07-18", conv.RateDate)
	}
}

func TestToBaseNoRate(t *testing.T) {
	store := memory.NewStore()
	svc := New(store.Rates(), nil)
	_, err := svc.ToBase(money.MustNew(100, "EUR"), date(2026, 7, 18))
	if !errors.Is(err, ErrNoRate) {
		t.Errorf("expected ErrNoRate, got %v", err)
	}
}

func TestRefreshRatesNoProvider(t *testing.T) {
	store := memory.NewStore()
	svc := New(store.Rates(), nil)
	if _, err := svc.RefreshRates(date(2026, 7, 18)); err == nil {
		t.Error("refresh without provider must error")
	}
}
