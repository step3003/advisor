package i18n

import (
	"fmt"
	"strings"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// monthNames — родительный/именительный набор названий месяцев для заголовков.
var monthNames = [12]string{
	"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

// MonthName возвращает название месяца (1..12) или пустую строку при выходе за диапазон.
func MonthName(m int) string {
	if m < 1 || m > 12 {
		return ""
	}
	return monthNames[m-1]
}

// MonthTitle форматирует месяц как «Июль 2026».
func MonthTitle(ym core.YearMonth) string {
	return fmt.Sprintf("%s %d", MonthName(ym.Month), ym.Year)
}

// TypeLabel возвращает подпись типа операции.
func TypeLabel(t core.EntryType) string {
	if t == core.Income {
		return TypeIncomeLabel
	}
	return TypeExpenseLabel
}

// FormatDate форматирует дату как «18.07.2026».
func FormatDate(d core.Date) string {
	return fmt.Sprintf("%02d.%02d.%04d", d.Day, d.Month, d.Year)
}

// FormatAmount форматирует сумму без кода валюты в русском стиле: разряды целой
// части разделяются пробелом, дробная — запятой («1 234,56»). Знак сохраняется.
func FormatAmount(m money.Money) string {
	return formatDecimal(m.Decimal())
}

// FormatMoney форматирует сумму с кодом валюты: «1 234,56 BYN».
func FormatMoney(m money.Money) string {
	return FormatAmount(m) + " " + m.Currency().String()
}

// FormatSignedAmount форматирует сумму с явным знаком «+»/«-» (для отклонений).
// Ноль выводится без знака.
func FormatSignedAmount(m money.Money) string {
	s := FormatAmount(m)
	if m.IsPositive() {
		return "+" + s
	}
	return s
}

// FormatPercent форматирует процент исполнения как «85 %».
func FormatPercent(p int) string {
	return fmt.Sprintf("%d %%", p)
}

// formatDecimal превращает десятичную строку money.Decimal() («-1234.56») в
// локализованное представление «-1 234,56».
func formatDecimal(dec string) string {
	neg := strings.HasPrefix(dec, "-")
	dec = strings.TrimPrefix(dec, "-")

	intPart := dec
	frac := ""
	if i := strings.IndexByte(dec, '.'); i >= 0 {
		intPart = dec[:i]
		frac = dec[i+1:]
	}

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString(groupThousands(intPart))
	if frac != "" {
		b.WriteByte(',')
		b.WriteString(frac)
	}
	return b.String()
}

// groupThousands разбивает целую часть на группы по три цифры, разделяя пробелом.
func groupThousands(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		b.WriteByte(' ')
	}
	for i := pre; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(' ')
		}
	}
	return b.String()
}
