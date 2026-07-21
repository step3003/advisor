package core

import (
	"testing"
	"time"
)

func TestMetaTouchAndWins(t *testing.T) {
	t0 := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	m := NewMeta("id-1", t0)
	if m.Rev != 1 {
		t.Fatalf("initial rev = %d, want 1", m.Rev)
	}
	m2 := m.Touch(t0.Add(time.Hour))
	if m2.Rev != 2 {
		t.Fatalf("touched rev = %d, want 2", m2.Rev)
	}
	// Большая ревизия выигрывает.
	if !m2.Wins(m) {
		t.Error("higher rev must win")
	}
	// При равной ревизии выигрывает более позднее время.
	later := Meta{ID: "id-1", Rev: 2, UpdatedAt: t0.Add(2 * time.Hour)}
	if !later.Wins(m2) {
		t.Error("later updated_at must win on equal rev")
	}
}

func TestYearMonthNav(t *testing.T) {
	jan := YearMonth{Year: 2026, Month: 1}
	if jan.Prev().String() != "2025-12" {
		t.Errorf("Prev(2026-01) = %s", jan.Prev())
	}
	dec := YearMonth{Year: 2026, Month: 12}
	if dec.Next().String() != "2027-01" {
		t.Errorf("Next(2026-12) = %s", dec.Next())
	}
	if !jan.Before(YearMonth{2026, 2}) {
		t.Error("jan before feb")
	}
}

func TestYearMonthDaysIn(t *testing.T) {
	cases := map[YearMonth]int{
		{2026, 2}: 28,
		{2024, 2}: 29, // високосный
		{2026, 4}: 30,
		{2026, 7}: 31,
	}
	for ym, want := range cases {
		if got := ym.DaysIn(); got != want {
			t.Errorf("DaysIn(%s) = %d, want %d", ym, got, want)
		}
	}
}

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-07-18")
	if err != nil {
		t.Fatal(err)
	}
	if d.Year != 2026 || d.Month != 7 || d.Day != 18 {
		t.Errorf("ParseDate = %+v", d)
	}
	if d.String() != "2026-07-18" {
		t.Errorf("String = %s", d.String())
	}
	if _, err := ParseDate("not-a-date"); err == nil {
		t.Error("bad date must error")
	}
}

func TestNewDateValidation(t *testing.T) {
	if _, err := NewDate(2026, 2, 30); err == nil {
		t.Error("Feb 30 must be invalid")
	}
	if _, err := NewDate(2026, 13, 1); err == nil {
		t.Error("month 13 must be invalid")
	}
	if _, err := NewDate(2024, 2, 29); err != nil {
		t.Errorf("Feb 29 2024 must be valid: %v", err)
	}
}

func TestDateCompare(t *testing.T) {
	a := Date{2026, 7, 1}
	b := Date{2026, 7, 15}
	if !a.Before(b) || !b.After(a) || a.Equal(b) {
		t.Error("date comparison failed")
	}
}
