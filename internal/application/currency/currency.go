// Package currency — usecase пересчёта сумм в базовую валюту BYN (FR-CUR).
//
// Использует локальный кэш курсов (ports.RateCache). Если курса на нужную дату
// нет — берёт последний доступный ≤ даты и помечает результат приблизительным
// (FR-CUR-4). Сетевое обновление курсов идёт через ports.RateProvider.
package currency

import (
	"errors"
	"fmt"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// ErrNoRate — нет ни одного курса для пересчёта.
var ErrNoRate = errors.New("currency: нет курса для пересчёта")

// Service — сервис пересчёта валют.
type Service struct {
	cache    ports.RateCache
	provider ports.RateProvider
}

// New создаёт сервис. provider может быть nil, если офлайн-режим (только кэш).
func New(cache ports.RateCache, provider ports.RateProvider) *Service {
	return &Service{cache: cache, provider: provider}
}

// Conversion — результат пересчёта.
type Conversion struct {
	Amount      money.Money // сумма в BYN
	Approximate bool        // true => курс на точную дату отсутствовал (FR-CUR-4)
	RateDate    core.Date   // дата использованного курса
}

// ToBase пересчитывает сумму в BYN на дату on.
func (s *Service) ToBase(amount money.Money, on core.Date) (Conversion, error) {
	if amount.Currency() == money.BaseCurrency {
		return Conversion{Amount: amount, Approximate: false, RateDate: on}, nil
	}

	// Точный курс на дату.
	if rate, ok, err := s.cache.GetRate(amount.Currency(), on); err != nil {
		return Conversion{}, err
	} else if ok {
		byn, err := rate.ToBYN(amount)
		if err != nil {
			return Conversion{}, err
		}
		return Conversion{Amount: byn, Approximate: false, RateDate: on}, nil
	}

	// Fallback: последний доступный курс ≤ даты.
	rate, ok, err := s.cache.GetLatestBefore(amount.Currency(), on)
	if err != nil {
		return Conversion{}, err
	}
	if !ok {
		// Нет в кэше — пробуем дотянуть курс из провайдера по требованию
		// (онлайн-режим сервера, FR-CUR-3). При офлайне/ошибке — ErrNoRate.
		if s.provider != nil {
			if fetched, ferr := s.provider.RateFor(amount.Currency(), on); ferr == nil {
				_ = s.cache.SaveRate(fetched)
				byn, cerr := fetched.ToBYN(amount)
				if cerr != nil {
					return Conversion{}, cerr
				}
				approx := !fetched.Date.Equal(on)
				return Conversion{Amount: byn, Approximate: approx, RateDate: fetched.Date}, nil
			}
		}
		return Conversion{}, fmt.Errorf("%w: %s на %s", ErrNoRate, amount.Currency(), on)
	}
	byn, err := rate.ToBYN(amount)
	if err != nil {
		return Conversion{}, err
	}
	return Conversion{Amount: byn, Approximate: true, RateDate: rate.Date}, nil
}

// RefreshRates загружает курсы на дату из провайдера и сохраняет в кэш (FR-CUR-5).
// Требует наличия provider; при офлайне возвращает ошибку, не мешая работе кэша.
func (s *Service) RefreshRates(on core.Date) (int, error) {
	if s.provider == nil {
		return 0, errors.New("currency: провайдер курсов не сконфигурирован")
	}
	rates, err := s.provider.RatesOn(on)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rates {
		if err := s.cache.SaveRate(r); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
