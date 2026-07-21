package sqlite

import (
	"database/sql"
	"errors"

	"advisor/internal/application/ports"
	"advisor/internal/domain/money"
)

// SettingsRepo реализует ports.SettingsStore для настроек одного пользователя.
type SettingsRepo struct {
	idx   *Index
	owner string
}

// Settings возвращает репозиторий настроек владельца ownerID.
func (i *Index) Settings(ownerID string) *SettingsRepo { return &SettingsRepo{idx: i, owner: ownerID} }

func (r *SettingsRepo) Get(key string) (string, bool, error) {
	var v string
	err := r.idx.db.QueryRow(`SELECT value FROM settings WHERE owner_id=? AND key=?`, r.owner, key).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return v, true, nil
}

func (r *SettingsRepo) Set(key, value string) error {
	_, err := r.idx.db.Exec(`INSERT INTO settings(owner_id, key, value) VALUES(?,?,?)
		ON CONFLICT(owner_id, key) DO UPDATE SET value=excluded.value`, r.owner, key, value)
	return err
}

// CurrencyRepo реализует ports.CurrencyCatalog поверх таблицы currencies.
type CurrencyRepo struct{ idx *Index }

// Currencies возвращает справочник валют.
func (i *Index) Currencies() *CurrencyRepo { return &CurrencyRepo{idx: i} }

// ListCurrencies возвращает валюты справочника; базовая валюта (BYN) — первой,
// далее по алфавиту, чтобы выбор в UI был предсказуем.
func (r *CurrencyRepo) ListCurrencies() ([]ports.CurrencyInfo, error) {
	rows, err := r.idx.db.Query(`SELECT code, name FROM currencies
		ORDER BY (code = ?) DESC, code ASC`, money.BaseCurrency.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []ports.CurrencyInfo
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		out = append(out, ports.CurrencyInfo{Code: money.Currency(code), Name: name})
	}
	return out, rows.Err()
}

// compile-time проверки соответствия интерфейсам.
var (
	_ ports.SettingsStore   = (*SettingsRepo)(nil)
	_ ports.CurrencyCatalog = (*CurrencyRepo)(nil)
)
