package settings

import (
	"testing"

	"advisor/internal/application/ports"
	"advisor/internal/domain/money"
)

// fakeStore — in-memory реализация ports.SettingsStore для тестов.
type fakeStore struct{ m map[string]string }

func newFakeStore() *fakeStore { return &fakeStore{m: map[string]string{}} }

func (f *fakeStore) Get(key string) (string, bool, error) {
	v, ok := f.m[key]
	return v, ok, nil
}

func (f *fakeStore) Set(key, value string) error {
	f.m[key] = value
	return nil
}

// fakeCatalog — фиктивный справочник валют.
type fakeCatalog struct{ list []ports.CurrencyInfo }

func (f *fakeCatalog) ListCurrencies() ([]ports.CurrencyInfo, error) { return f.list, nil }

func TestDefaultCurrencyFallsBackToBase(t *testing.T) {
	s := New(newFakeStore(), &fakeCatalog{})
	got, err := s.DefaultCurrency()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if got != money.BaseCurrency {
		t.Fatalf("ожидалась базовая валюта %s, получено %s", money.BaseCurrency, got)
	}
}

func TestSetAndGetDefaultCurrency(t *testing.T) {
	s := New(newFakeStore(), &fakeCatalog{})
	if err := s.SetDefaultCurrency("USD"); err != nil {
		t.Fatalf("SetDefaultCurrency: %v", err)
	}
	got, err := s.DefaultCurrency()
	if err != nil {
		t.Fatalf("DefaultCurrency: %v", err)
	}
	if got != money.Currency("USD") {
		t.Fatalf("ожидалось USD, получено %s", got)
	}
}

func TestSetDefaultCurrencyRejectsEmpty(t *testing.T) {
	s := New(newFakeStore(), &fakeCatalog{})
	if err := s.SetDefaultCurrency(""); err == nil {
		t.Fatal("ожидалась ошибка для пустой валюты")
	}
}

func TestListCurrenciesDelegates(t *testing.T) {
	cat := &fakeCatalog{list: []ports.CurrencyInfo{{Code: "BYN", Name: "Белорусский рубль"}}}
	s := New(newFakeStore(), cat)
	got, err := s.ListCurrencies()
	if err != nil {
		t.Fatalf("ListCurrencies: %v", err)
	}
	if len(got) != 1 || got[0].Code != "BYN" {
		t.Fatalf("неожиданный справочник: %+v", got)
	}
}
