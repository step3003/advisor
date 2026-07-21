package i18n

import (
	"testing"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		minor int64
		want  string
	}{
		{0, "0,00"},
		{5, "0,05"},
		{4250, "42,50"},
		{123456, "1 234,56"},
		{123456789, "1 234 567,89"},
		{-123456, "-1 234,56"},
		{100000, "1 000,00"},
	}
	for _, c := range cases {
		got := FormatAmount(money.MustNew(c.minor, "BYN"))
		if got != c.want {
			t.Errorf("FormatAmount(%d) = %q, ожидалось %q", c.minor, got, c.want)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	got := FormatMoney(money.MustNew(123456, "USD"))
	if got != "1 234,56 USD" {
		t.Errorf("FormatMoney = %q", got)
	}
}

func TestFormatSignedAmount(t *testing.T) {
	if got := FormatSignedAmount(money.MustNew(4250, "BYN")); got != "+42,50" {
		t.Errorf("положительное: %q", got)
	}
	if got := FormatSignedAmount(money.MustNew(-4250, "BYN")); got != "-42,50" {
		t.Errorf("отрицательное: %q", got)
	}
	if got := FormatSignedAmount(money.MustNew(0, "BYN")); got != "0,00" {
		t.Errorf("ноль: %q", got)
	}
}

func TestFormatPercent(t *testing.T) {
	if got := FormatPercent(85); got != "85 %" {
		t.Errorf("FormatPercent = %q", got)
	}
}

func TestMonthTitle(t *testing.T) {
	ym := core.YearMonth{Year: 2026, Month: 7}
	if got := MonthTitle(ym); got != "Июль 2026" {
		t.Errorf("MonthTitle = %q", got)
	}
	if MonthName(0) != "" || MonthName(13) != "" {
		t.Error("MonthName вне диапазона должен быть пустым")
	}
}

func TestFormatDate(t *testing.T) {
	d := core.Date{Year: 2026, Month: 7, Day: 8}
	if got := FormatDate(d); got != "08.07.2026" {
		t.Errorf("FormatDate = %q", got)
	}
}

func TestTypeLabel(t *testing.T) {
	if TypeLabel(core.Income) != TypeIncomeLabel {
		t.Error("доход")
	}
	if TypeLabel(core.Expense) != TypeExpenseLabel {
		t.Error("расход")
	}
}
