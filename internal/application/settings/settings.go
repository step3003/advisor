// Package settings — usecase пользовательских настроек (FR-SET).
//
// Хранит валюту по умолчанию для ввода (FR-SET-1) и предоставляет справочник
// валют для выбора в UI. Зависит только от портов и домена.
package settings

import (
	"errors"

	"advisor/internal/application/ports"
	"advisor/internal/domain/money"
)

// keyDefaultCurrency — ключ настройки «валюта по умолчанию».
const keyDefaultCurrency = "default_currency"

// Service — сервис настроек.
type Service struct {
	store   ports.SettingsStore
	catalog ports.CurrencyCatalog
}

// New собирает сервис.
func New(store ports.SettingsStore, catalog ports.CurrencyCatalog) *Service {
	return &Service{store: store, catalog: catalog}
}

// DefaultCurrency возвращает валюту ввода по умолчанию (FR-SET-1).
// Если не задана — базовую валюту приложения (BYN).
func (s *Service) DefaultCurrency() (money.Currency, error) {
	v, ok, err := s.store.Get(keyDefaultCurrency)
	if err != nil {
		return "", err
	}
	if !ok || v == "" {
		return money.BaseCurrency, nil
	}
	return money.Currency(v), nil
}

// SetDefaultCurrency сохраняет валюту ввода по умолчанию (FR-SET-1).
func (s *Service) SetDefaultCurrency(c money.Currency) error {
	if c.IsZero() {
		return errors.New("settings: пустой код валюты")
	}
	return s.store.Set(keyDefaultCurrency, c.String())
}

// ListCurrencies возвращает справочник валют для выбора (FR-CUR-1).
func (s *Service) ListCurrencies() ([]ports.CurrencyInfo, error) {
	return s.catalog.ListCurrencies()
}
