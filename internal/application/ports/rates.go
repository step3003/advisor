package ports

import (
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// RatePrecisionFactor — множитель точности курса. Официальные курсы НБ РБ
// публикуются с 4 знаками после запятой, поэтому курс хранится как целое,
// умноженное на 10^4 (без float, NFR-2).
const RatePrecisionFactor int64 = 10000

// Rate — официальный курс валюты к BYN на дату (НБ РБ).
//
// Смысл (раздел 7 ТЗ): Scale единиц валюты стоят OfficialRate BYN. Например
// 1 USD = 3.2567 BYN → Scale=1, RateBYNScaled=32567 (3.2567 × 10^4).
//
// В таблице exchange_rates значение хранится в колонке rate_byn_minor.
type Rate struct {
	Currency money.Currency
	Date     core.Date
	Scale    int64 // Cur_Scale
	// RateBYNScaled — Cur_OfficialRate, умноженный на RatePrecisionFactor (10^4).
	RateBYNScaled int64
}

// ToBYN пересчитывает сумму в этой валюте в BYN по курсу (учитывая Scale).
//
//	byn_minor = amount_minor * RateBYNScaled / (RatePrecisionFactor * Scale)
//
// Вывод единиц: OfficialRate = RateBYNScaled / 10^4 (BYN-major за Scale единиц);
// amount_minor — минорные единицы валюты (×100); результат — минорные BYN (×100).
func (r Rate) ToBYN(amount money.Money) (money.Money, error) {
	return money.Convert(amount, money.BaseCurrency, r.RateBYNScaled, RatePrecisionFactor*r.Scale)
}

// RateProvider — внешний поставщик курсов (реализуется HTTP-клиентом НБ РБ).
// Сетевые вызовы; в unit-тестах usecase мокается.
type RateProvider interface {
	// RatesOn возвращает все курсы на дату.
	RatesOn(date core.Date) ([]Rate, error)
	// RateFor возвращает курс одной валюты на дату.
	RateFor(currency money.Currency, date core.Date) (Rate, error)
}

// RateCache — локальный кэш курсов (в SQLite-индексе, НЕ в vault; FR-SYNC-7).
type RateCache interface {
	// SaveRate сохраняет/обновляет курс в кэше.
	SaveRate(rate Rate) error
	// GetRate возвращает курс ровно на дату (ok=false, если нет).
	GetRate(currency money.Currency, date core.Date) (rate Rate, ok bool, err error)
	// GetLatestBefore возвращает последний доступный курс с датой ≤ date (FR-CUR-4).
	GetLatestBefore(currency money.Currency, date core.Date) (rate Rate, ok bool, err error)
}
