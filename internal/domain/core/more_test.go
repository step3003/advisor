package core

import (
	"testing"
	"time"
)

func TestYearMonthOfAndDateOf(t *testing.T) {
	tm := time.Date(2026, 7, 18, 9, 30, 0, 0, time.UTC)
	if ym := YearMonthOf(tm); ym.Year != 2026 || ym.Month != 7 {
		t.Errorf("YearMonthOf = %+v", ym)
	}
	if d := DateOf(tm); d.Year != 2026 || d.Month != 7 || d.Day != 18 {
		t.Errorf("DateOf = %+v", d)
	}
}

func TestYearMonthFirstDayAndConversions(t *testing.T) {
	ym := YearMonth{Year: 2026, Month: 7}
	fd := ym.FirstDay()
	if fd.String() != "2026-07-01" {
		t.Errorf("FirstDay = %s", fd)
	}
	if !ym.Equal(YearMonth{Year: 2026, Month: 7}) {
		t.Error("Equal failed")
	}
	if ym.Before(ym) {
		t.Error("month not before itself")
	}
}

func TestDateTimeAndYearMonth(t *testing.T) {
	d := Date{Year: 2026, Month: 7, Day: 18}
	tm := d.Time()
	if tm.Hour() != 0 || tm.Location() != time.UTC {
		t.Errorf("Time = %v", tm)
	}
	if d.YearMonth().String() != "2026-07" {
		t.Errorf("YearMonth = %s", d.YearMonth())
	}
	if d.IsZero() {
		t.Error("must not be zero")
	}
	if !(Date{}).IsZero() {
		t.Error("empty date must be zero")
	}
}

func TestEntryTypeValid(t *testing.T) {
	if !Expense.Valid() || !Income.Valid() {
		t.Error("expense/income must be valid")
	}
	if EntryType("bad").Valid() {
		t.Error("bad type must be invalid")
	}
}

func TestNewYearMonthValidation(t *testing.T) {
	if _, err := NewYearMonth(2026, 0); err == nil {
		t.Error("month 0 invalid")
	}
	if _, err := NewYearMonth(0, 7); err == nil {
		t.Error("year 0 invalid")
	}
	if _, err := NewYearMonth(2026, 7); err != nil {
		t.Errorf("valid month errored: %v", err)
	}
}
