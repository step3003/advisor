package nbrb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

func TestParseRateToScaled(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"3.2567", 32567},
		{"3", 30000},
		{"3.25", 32500},
		{"0.0123", 123},
		{"4.10", 41000},
		{"3.25674", 32567}, // усечение до 4 знаков (4-я цифра <5, без округления)
		{"3.25675", 32568}, // округление вверх по 5-й цифре
	}
	for _, c := range cases {
		got, err := parseRateToScaled(c.in)
		if err != nil {
			t.Fatalf("parseRateToScaled(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseRateToScaled(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRateForBYNPassthrough(t *testing.T) {
	c := New()
	r, err := c.RateFor(money.BaseCurrency, core.Date{Year: 2026, Month: 7, Day: 18})
	if err != nil {
		t.Fatal(err)
	}
	// BYN к BYN = 1:1 (Scale=1, RateBYNScaled = 10^4), сеть не дёргается.
	if r.Scale != 1 || r.RateBYNScaled != ports.RatePrecisionFactor {
		t.Errorf("BYN passthrough = %+v", r)
	}
}

func TestRatesOnHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/exrates/rates" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"Cur_ID":431,"Cur_Abbreviation":"USD","Cur_Scale":1,"Cur_Name":"Доллар США","Cur_OfficialRate":3.2567},
			{"Cur_ID":298,"Cur_Abbreviation":"RUB","Cur_Scale":100,"Cur_Name":"Рос. рублей","Cur_OfficialRate":4.1023}
		]`))
	}))
	defer srv.Close()

	client := NewWithBase(srv.URL)
	rates, err := client.RatesOn(core.Date{Year: 2026, Month: 7, Day: 18})
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 2 {
		t.Fatalf("rates = %d, want 2", len(rates))
	}
	usd := findRate(rates, "USD")
	if usd == nil || usd.RateBYNScaled != 32567 || usd.Scale != 1 {
		t.Errorf("USD rate = %+v", usd)
	}
	rub := findRate(rates, "RUB")
	if rub == nil || rub.RateBYNScaled != 41023 || rub.Scale != 100 {
		t.Errorf("RUB rate = %+v", rub)
	}

	// Проверяем корректность пересчёта: 100.00 USD => 325.67 BYN.
	byn, err := usd.ToBYN(money.MustNew(10000, "USD"))
	if err != nil {
		t.Fatal(err)
	}
	if byn.Minor() != 32567 {
		t.Errorf("100 USD -> %d BYN minor, want 32567", byn.Minor())
	}
}

func findRate(rates []ports.Rate, cur money.Currency) *ports.Rate {
	for i := range rates {
		if rates[i].Currency == cur {
			return &rates[i]
		}
	}
	return nil
}
