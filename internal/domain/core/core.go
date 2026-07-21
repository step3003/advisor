// Package core — общий доменный «kernel»: разделяемые value-объекты и метаданные
// сущностей. Не зависит ни от каких внешних пакетов, только stdlib.
//
// Здесь живут вещи, которые нужны нескольким доменным сущностям (category,
// transaction, plan, recurring): тип операции, метаданные версии/ревизии для
// разрешения конфликтов синхронизации (FR-SYNC-6), календарный месяц и дата.
package core

import (
	"errors"
	"fmt"
	"time"
)

// EntryType — тип операции: расход или доход (FR-CAT-2).
type EntryType string

const (
	Expense EntryType = "expense"
	Income  EntryType = "income"
)

// Valid проверяет, что тип операции допустим.
func (t EntryType) Valid() bool {
	return t == Expense || t == Income
}

// ErrInvalidEntryType — недопустимый тип операции.
var ErrInvalidEntryType = errors.New("core: недопустимый тип операции (ожидается expense|income)")

// Meta — метаданные любой синхронизируемой записи vault.
//
// ID    — стабильный UUID записи (не меняется никогда).
// Rev   — монотонная ревизия, растёт при каждом изменении.
// UpdatedAt — момент последнего изменения в UTC.
//
// Пара (Rev, UpdatedAt) даёт детерминированное разрешение конфликтов iCloud
// (FR-SYNC-5/6): выигрывает большая Rev, при равенстве — более поздний UpdatedAt.
type Meta struct {
	ID        string
	Rev       int64
	UpdatedAt time.Time
}

// NewMeta создаёт метаданные первой ревизии.
func NewMeta(id string, now time.Time) Meta {
	return Meta{ID: id, Rev: 1, UpdatedAt: now.UTC()}
}

// Touch возвращает метаданные следующей ревизии с новым временем.
func (m Meta) Touch(now time.Time) Meta {
	return Meta{ID: m.ID, Rev: m.Rev + 1, UpdatedAt: now.UTC()}
}

// Wins сообщает, побеждает ли данная запись конкурирующую при разрешении конфликта.
// Правило: большая Rev выигрывает; при равной Rev — более поздний UpdatedAt.
func (m Meta) Wins(other Meta) bool {
	if m.Rev != other.Rev {
		return m.Rev > other.Rev
	}
	return m.UpdatedAt.After(other.UpdatedAt)
}

// YearMonth — календарный месяц планирования (FR-PLAN-1).
type YearMonth struct {
	Year  int
	Month int // 1..12
}

// NewYearMonth создаёт и валидирует календарный месяц.
func NewYearMonth(year, month int) (YearMonth, error) {
	if month < 1 || month > 12 {
		return YearMonth{}, fmt.Errorf("core: месяц вне диапазона 1..12: %d", month)
	}
	if year < 1 {
		return YearMonth{}, fmt.Errorf("core: некорректный год: %d", year)
	}
	return YearMonth{Year: year, Month: month}, nil
}

// YearMonthOf извлекает месяц из даты.
func YearMonthOf(t time.Time) YearMonth {
	return YearMonth{Year: t.Year(), Month: int(t.Month())}
}

// String возвращает представление "YYYY-MM".
func (ym YearMonth) String() string {
	return fmt.Sprintf("%04d-%02d", ym.Year, ym.Month)
}

// Prev возвращает предыдущий календарный месяц (для копирования плана, FR-PLAN-4).
func (ym YearMonth) Prev() YearMonth {
	if ym.Month == 1 {
		return YearMonth{Year: ym.Year - 1, Month: 12}
	}
	return YearMonth{Year: ym.Year, Month: ym.Month - 1}
}

// Next возвращает следующий календарный месяц.
func (ym YearMonth) Next() YearMonth {
	if ym.Month == 12 {
		return YearMonth{Year: ym.Year + 1, Month: 1}
	}
	return YearMonth{Year: ym.Year, Month: ym.Month + 1}
}

// Before сообщает, что месяц раньше другого.
func (ym YearMonth) Before(other YearMonth) bool {
	if ym.Year != other.Year {
		return ym.Year < other.Year
	}
	return ym.Month < other.Month
}

// Equal сравнивает месяцы.
func (ym YearMonth) Equal(other YearMonth) bool {
	return ym.Year == other.Year && ym.Month == other.Month
}

// FirstDay возвращает первый день месяца (UTC).
func (ym YearMonth) FirstDay() Date {
	return Date{Year: ym.Year, Month: ym.Month, Day: 1}
}

// DaysIn возвращает число дней в месяце.
func (ym YearMonth) DaysIn() int {
	// День 0 следующего месяца = последний день текущего.
	t := time.Date(ym.Year, time.Month(ym.Month)+1, 0, 0, 0, 0, 0, time.UTC)
	return t.Day()
}

// Date — календарная дата операции без времени и часового пояса (FR-TX-1).
type Date struct {
	Year  int
	Month int
	Day   int
}

// NewDate валидирует и создаёт дату.
func NewDate(year, month, day int) (Date, error) {
	if month < 1 || month > 12 {
		return Date{}, fmt.Errorf("core: месяц вне диапазона: %d", month)
	}
	ym := YearMonth{Year: year, Month: month}
	if day < 1 || day > ym.DaysIn() {
		return Date{}, fmt.Errorf("core: день вне диапазона для %s: %d", ym, day)
	}
	return Date{Year: year, Month: month, Day: day}, nil
}

// DateOf извлекает дату из time.Time.
func DateOf(t time.Time) Date {
	return Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

// ParseDate разбирает дату формата "YYYY-MM-DD".
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("core: неверный формат даты %q: %w", s, err)
	}
	return DateOf(t), nil
}

// String возвращает "YYYY-MM-DD".
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// YearMonth возвращает месяц этой даты.
func (d Date) YearMonth() YearMonth {
	return YearMonth{Year: d.Year, Month: d.Month}
}

// Time превращает дату в time.Time (полночь UTC).
func (d Date) Time() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

// Before сравнивает даты по возрастанию.
func (d Date) Before(other Date) bool {
	return d.Time().Before(other.Time())
}

// After сравнивает даты по убыванию.
func (d Date) After(other Date) bool {
	return d.Time().After(other.Time())
}

// Equal сравнивает даты.
func (d Date) Equal(other Date) bool {
	return d == other
}

// IsZero сообщает, что дата не задана.
func (d Date) IsZero() bool {
	return d == Date{}
}
