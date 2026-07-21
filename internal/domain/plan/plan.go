// Package plan — доменные сущности планирования (FR-PLAN).
//
// PlanItem — плановая позиция на календарный месяц. Уникальность обеспечивается
// на уровне usecase/индекса по паре (месяц, категория, валюта) (FR-PLAN-3),
// доменный объект хранит эти поля и предоставляет ключ уникальности.
package plan

import (
	"errors"
	"strings"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// Ошибки домена планирования.
var (
	ErrEmptyCategory = errors.New("plan: не указана категория")
	ErrNonPositive   = errors.New("plan: плановая сумма должна быть положительной")
)

// PlanItem — плановая позиция месяца (FR-PLAN-2).
type PlanItem struct {
	Meta core.Meta

	Period     core.YearMonth
	CategoryID string
	Amount     money.Money
	Note       string

	CreatedAt time.Time // UTC
}

// New создаёт плановую позицию с валидацией.
func New(id string, period core.YearMonth, categoryID string, amount money.Money, note string, now time.Time) (*PlanItem, error) {
	if strings.TrimSpace(categoryID) == "" {
		return nil, ErrEmptyCategory
	}
	if !amount.IsPositive() {
		return nil, ErrNonPositive
	}
	return &PlanItem{
		Meta:       core.NewMeta(id, now),
		Period:     period,
		CategoryID: categoryID,
		Amount:     amount,
		Note:       strings.TrimSpace(note),
		CreatedAt:  now.UTC(),
	}, nil
}

// SetAmount изменяет плановую сумму (FR-PLAN-3: сумма редактируется, не дублируется).
func (p *PlanItem) SetAmount(amount money.Money, now time.Time) error {
	if !amount.IsPositive() {
		return ErrNonPositive
	}
	p.Amount = amount
	p.Meta = p.Meta.Touch(now)
	return nil
}

// SetNote обновляет комментарий позиции.
func (p *PlanItem) SetNote(note string, now time.Time) {
	p.Note = strings.TrimSpace(note)
	p.Meta = p.Meta.Touch(now)
}

// Key — ключ уникальности плановой позиции в месяце (FR-PLAN-3).
type Key struct {
	Period     core.YearMonth
	CategoryID string
	Currency   money.Currency
}

// UniqueKey возвращает ключ уникальности данной позиции.
func (p *PlanItem) UniqueKey() Key {
	return Key{Period: p.Period, CategoryID: p.CategoryID, Currency: p.Amount.Currency()}
}
