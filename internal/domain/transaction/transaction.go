// Package transaction — доменная сущность фактической операции (FR-TX).
//
// Сумма хранится в исходной валюте операции (FR-TX-3). Пересчёт в базовую
// валюту (BYN) выполняется на уровне usecase по курсу на дату операции.
package transaction

import (
	"errors"
	"strings"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// Ошибки домена транзакций.
var (
	ErrEmptyCategory = errors.New("transaction: не указана категория")
	ErrZeroDate      = errors.New("transaction: не указана дата операции")
	ErrNonPositive   = errors.New("transaction: сумма должна быть положительной")
)

// Transaction — факт расхода или дохода.
//
// Сумма всегда положительна; направление задаётся полем Type
// (expense/income), чтобы агрегаты считались явно, без «знаковых» сумм.
type Transaction struct {
	Meta core.Meta

	Type        core.EntryType
	OccurredOn  core.Date // дата операции YYYY-MM-DD
	CategoryID  string
	Amount      money.Money
	Note        string
	RecurringID string // "" => обычная операция; иначе ID породившего шаблона

	CreatedAt time.Time // UTC
}

// New создаёт фактическую операцию с валидацией инвариантов.
func New(id string, typ core.EntryType, occurredOn core.Date, categoryID string, amount money.Money, note string, now time.Time) (*Transaction, error) {
	if !typ.Valid() {
		return nil, core.ErrInvalidEntryType
	}
	if strings.TrimSpace(categoryID) == "" {
		return nil, ErrEmptyCategory
	}
	if occurredOn.IsZero() {
		return nil, ErrZeroDate
	}
	if !amount.IsPositive() {
		return nil, ErrNonPositive
	}
	return &Transaction{
		Meta:       core.NewMeta(id, now),
		Type:       typ,
		OccurredOn: occurredOn,
		CategoryID: categoryID,
		Amount:     amount,
		Note:       strings.TrimSpace(note),
		CreatedAt:  now.UTC(),
	}, nil
}

// Edit изменяет параметры операции (FR-TX-2) и повышает ревизию.
func (t *Transaction) Edit(occurredOn core.Date, categoryID string, amount money.Money, note string, now time.Time) error {
	if strings.TrimSpace(categoryID) == "" {
		return ErrEmptyCategory
	}
	if occurredOn.IsZero() {
		return ErrZeroDate
	}
	if !amount.IsPositive() {
		return ErrNonPositive
	}
	t.OccurredOn = occurredOn
	t.CategoryID = categoryID
	t.Amount = amount
	t.Note = strings.TrimSpace(note)
	t.Meta = t.Meta.Touch(now)
	return nil
}

// IsExpense сообщает, что это расход.
func (t *Transaction) IsExpense() bool { return t.Type == core.Expense }

// IsIncome сообщает, что это доход.
func (t *Transaction) IsIncome() bool { return t.Type == core.Income }
