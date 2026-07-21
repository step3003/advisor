package ports

import "advisor/internal/domain/money"

// SettingsStore — хранилище пользовательских настроек «ключ-значение» (FR-SET).
//
// Реализуется локально в SQLite-индексе (таблица settings). Настройки в MVP
// (например валюта по умолчанию) не синхронизируются через vault — это выбор
// устройства/ввода; при необходимости их легко перенести в vault позже.
type SettingsStore interface {
	// Get возвращает значение по ключу; ok=false, если ключа нет.
	Get(key string) (value string, ok bool, err error)
	// Set сохраняет/обновляет значение по ключу.
	Set(key, value string) error
}

// CurrencyInfo — элемент справочника валют (FR-CUR-1) для выбора в UI.
type CurrencyInfo struct {
	Code money.Currency
	Name string
}

// CurrencyCatalog — справочник поддерживаемых валют.
type CurrencyCatalog interface {
	// ListCurrencies возвращает валюты справочника (базовая валюта — первой).
	ListCurrencies() ([]CurrencyInfo, error)
}
