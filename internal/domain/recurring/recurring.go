// Package recurring — доменная сущность шаблона повторяющейся операции (FR-REC).
//
// В MVP поддерживается только ежемесячная периодичность с указанием дня месяца
// (1..28, чтобы шаблон существовал в любом месяце — FR-REC-1). Шаблон активен
// в интервале [StartDate, EndDate]; при переходе на месяц из него идемпотентно
// генерируется плановая позиция (FR-REC-2), опционально — факт (FR-REC-3).
package recurring

import (
	"errors"
	"strings"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// Ошибки домена шаблонов.
var (
	ErrEmptyCategory = errors.New("recurring: не указана категория")
	ErrNonPositive   = errors.New("recurring: сумма должна быть положительной")
	ErrBadDay        = errors.New("recurring: день месяца должен быть в диапазоне 1..28")
	ErrBadInterval   = errors.New("recurring: дата окончания раньше даты начала")
)

// Template — шаблон повторяющейся операции.
type Template struct {
	Meta core.Meta

	Type           core.EntryType
	CategoryID     string
	Amount         money.Money
	DayOfMonth     int // 1..28
	StartDate      core.Date
	EndDate        *core.Date // nil => бессрочно
	AutoCreateFact bool       // FR-REC-3: создавать ли факт, по умолчанию только план
	Active         bool       // FR-REC-4: приостановка

	CreatedAt time.Time // UTC
}

// New создаёт шаблон повторяющейся операции с валидацией.
func New(id string, typ core.EntryType, categoryID string, amount money.Money, dayOfMonth int, start core.Date, end *core.Date, autoCreateFact bool, now time.Time) (*Template, error) {
	if !typ.Valid() {
		return nil, core.ErrInvalidEntryType
	}
	if strings.TrimSpace(categoryID) == "" {
		return nil, ErrEmptyCategory
	}
	if !amount.IsPositive() {
		return nil, ErrNonPositive
	}
	if dayOfMonth < 1 || dayOfMonth > 28 {
		return nil, ErrBadDay
	}
	if end != nil && end.Before(start) {
		return nil, ErrBadInterval
	}
	return &Template{
		Meta:           core.NewMeta(id, now),
		Type:           typ,
		CategoryID:     categoryID,
		Amount:         amount,
		DayOfMonth:     dayOfMonth,
		StartDate:      start,
		EndDate:        end,
		AutoCreateFact: autoCreateFact,
		Active:         true,
		CreatedAt:      now.UTC(),
	}, nil
}

// Edit изменяет параметры шаблона (FR-REC-4) и повышает ревизию.
func (t *Template) Edit(typ core.EntryType, categoryID string, amount money.Money, dayOfMonth int, start core.Date, end *core.Date, autoCreateFact bool, now time.Time) error {
	if !typ.Valid() {
		return core.ErrInvalidEntryType
	}
	if strings.TrimSpace(categoryID) == "" {
		return ErrEmptyCategory
	}
	if !amount.IsPositive() {
		return ErrNonPositive
	}
	if dayOfMonth < 1 || dayOfMonth > 28 {
		return ErrBadDay
	}
	if end != nil && end.Before(start) {
		return ErrBadInterval
	}
	t.Type = typ
	t.CategoryID = categoryID
	t.Amount = amount
	t.DayOfMonth = dayOfMonth
	t.StartDate = start
	t.EndDate = end
	t.AutoCreateFact = autoCreateFact
	t.Meta = t.Meta.Touch(now)
	return nil
}

// Pause приостанавливает шаблон (FR-REC-4).
func (t *Template) Pause(now time.Time) {
	t.Active = false
	t.Meta = t.Meta.Touch(now)
}

// Resume возобновляет шаблон.
func (t *Template) Resume(now time.Time) {
	t.Active = true
	t.Meta = t.Meta.Touch(now)
}

// AppliesTo сообщает, действует ли шаблон в указанном месяце.
// Активность проверяется по пересечению месяца с интервалом [Start, End].
func (t *Template) AppliesTo(ym core.YearMonth) bool {
	if !t.Active {
		return false
	}
	// Месяц должен быть не раньше месяца начала.
	startYM := t.StartDate.YearMonth()
	if ym.Before(startYM) {
		return false
	}
	if t.EndDate != nil {
		endYM := t.EndDate.YearMonth()
		if endYM.Before(ym) {
			return false
		}
	}
	return true
}

// OccurrenceDate возвращает дату операции шаблона в указанном месяце.
func (t *Template) OccurrenceDate(ym core.YearMonth) core.Date {
	return core.Date{Year: ym.Year, Month: ym.Month, Day: t.DayOfMonth}
}
