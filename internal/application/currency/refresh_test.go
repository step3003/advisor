package currency

import (
	"testing"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/infrastructure/memory"
)

// fakeProvider — тестовый поставщик курсов (сеть не используется).
type fakeProvider struct {
	rates []ports.Rate
}

func (f fakeProvider) RatesOn(date core.Date) ([]ports.Rate, error) { return f.rates, nil }
func (f fakeProvider) RateFor(cur money.Currency, date core.Date) (ports.Rate, error) {
	for _, r := range f.rates {
		if r.Currency == cur {
			return r, nil
		}
	}
	return ports.Rate{}, ErrNoRate
}

func TestRefreshRatesStoresToCache(t *testing.T) {
	store := memory.NewStore()
	d := core.Date{Year: 2026, Month: 7, Day: 18}
	provider := fakeProvider{rates: []ports.Rate{
		{Currency: "USD", Date: d, Scale: 1, RateBYNScaled: 32567},
		{Currency: "EUR", Date: d, Scale: 1, RateBYNScaled: 35000},
	}}
	svc := New(store.Rates(), provider)

	n, err := svc.RefreshRates(d)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("saved = %d, want 2", n)
	}
	// После обновления пересчёт работает из кэша.
	conv, err := svc.ToBase(money.MustNew(10000, "EUR"), d)
	if err != nil {
		t.Fatal(err)
	}
	if conv.Amount.Minor() != 35000 {
		t.Errorf("100 EUR -> %d, want 35000", conv.Amount.Minor())
	}
}
