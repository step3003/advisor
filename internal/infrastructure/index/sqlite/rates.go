package sqlite

import (
	"database/sql"
	"errors"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// RateCacheRepo реализует ports.RateCache поверх таблицы exchange_rates.
//
// Кэш курсов локален (FR-SYNC-7): в vault не попадает, каждое устройство
// скачивает курсы само.
type RateCacheRepo struct{ idx *Index }

// Rates возвращает кэш курсов.
func (i *Index) Rates() *RateCacheRepo { return &RateCacheRepo{idx: i} }

func (r *RateCacheRepo) SaveRate(rate ports.Rate) error {
	_, err := r.idx.db.Exec(`INSERT INTO exchange_rates(currency, rate_date, scale, rate_byn_minor)
		VALUES(?,?,?,?)
		ON CONFLICT(currency, rate_date) DO UPDATE SET scale=excluded.scale, rate_byn_minor=excluded.rate_byn_minor`,
		rate.Currency.String(), rate.Date.String(), rate.Scale, rate.RateBYNScaled)
	return err
}

func (r *RateCacheRepo) GetRate(currency money.Currency, date core.Date) (ports.Rate, bool, error) {
	row := r.idx.db.QueryRow(`SELECT scale, rate_byn_minor FROM exchange_rates
		WHERE currency=? AND rate_date=?`, currency.String(), date.String())
	var scale, rateMinor int64
	if err := row.Scan(&scale, &rateMinor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.Rate{}, false, nil
		}
		return ports.Rate{}, false, err
	}
	return ports.Rate{Currency: currency, Date: date, Scale: scale, RateBYNScaled: rateMinor}, true, nil
}

// GetLatestBefore возвращает последний доступный курс с датой ≤ date (FR-CUR-4).
func (r *RateCacheRepo) GetLatestBefore(currency money.Currency, date core.Date) (ports.Rate, bool, error) {
	row := r.idx.db.QueryRow(`SELECT rate_date, scale, rate_byn_minor FROM exchange_rates
		WHERE currency=? AND rate_date <= ? ORDER BY rate_date DESC LIMIT 1`,
		currency.String(), date.String())
	var rateDate string
	var scale, rateMinor int64
	if err := row.Scan(&rateDate, &scale, &rateMinor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.Rate{}, false, nil
		}
		return ports.Rate{}, false, err
	}
	d, err := core.ParseDate(rateDate)
	if err != nil {
		return ports.Rate{}, false, err
	}
	return ports.Rate{Currency: currency, Date: d, Scale: scale, RateBYNScaled: rateMinor}, true, nil
}

// SeedCurrencies заполняет справочник валют минимальным набором (FR-CUR-1),
// если он пуст. BYN — базовая валюта.
func (i *Index) SeedCurrencies() error {
	defaults := []struct {
		code, name string
		num        int
	}{
		{"BYN", "Белорусский рубль", 933},
		{"USD", "Доллар США", 840},
		{"EUR", "Евро", 978},
		{"RUB", "Российский рубль", 643},
	}
	for _, c := range defaults {
		if _, err := i.db.Exec(`INSERT INTO currencies(code, name, num_code) VALUES(?,?,?)
			ON CONFLICT(code) DO NOTHING`, c.code, c.name, c.num); err != nil {
			return err
		}
	}
	return nil
}

// compile-time проверка соответствия интерфейсу.
var _ ports.RateCache = (*RateCacheRepo)(nil)
