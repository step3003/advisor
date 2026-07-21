// Package money реализует денежный тип на основе int64 (минорные единицы).
//
// Принципиальное правило (NFR-2): деньги НИКОГДА не хранятся и не считаются во float.
// Внутреннее представление — целое число минорных единиц (копеек/центов),
// внешнее (файлы vault, экспорт) — десятичная строка вида "42.50".
package money

import (
	"errors"
	"fmt"
	"strings"
)

// Scale — количество знаков после запятой (минорных единиц в мажорной).
// Для всех поддерживаемых валют используется 2 знака (копейки/центы).
const Scale = 2

// minorPerMajor = 10^Scale.
const minorPerMajor int64 = 100

// Currency — ISO-4217 буквенный код валюты (например "BYN", "USD").
type Currency string

// Базовая валюта приложения (FR-CUR-2).
const BaseCurrency Currency = "BYN"

// String возвращает код валюты как строку.
func (c Currency) String() string { return string(c) }

// IsZero сообщает, что валюта не задана.
func (c Currency) IsZero() bool { return c == "" }

// Ошибки пакета.
var (
	ErrCurrencyMismatch = errors.New("money: несовпадение валют")
	ErrInvalidFormat    = errors.New("money: неверный формат суммы")
	ErrEmptyCurrency    = errors.New("money: пустой код валюты")
)

// Money — денежная сумма в минорных единицах конкретной валюты.
//
// Тип immutable: все операции возвращают новое значение.
type Money struct {
	minor    int64
	currency Currency
}

// New создаёт сумму из числа минорных единиц (например копеек).
func New(minor int64, currency Currency) (Money, error) {
	if currency.IsZero() {
		return Money{}, ErrEmptyCurrency
	}
	return Money{minor: minor, currency: currency}, nil
}

// MustNew — как New, но паникует при ошибке. Для констант/тестов.
func MustNew(minor int64, currency Currency) Money {
	m, err := New(minor, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// Zero возвращает нулевую сумму в указанной валюте.
func Zero(currency Currency) Money {
	return Money{minor: 0, currency: currency}
}

// Minor возвращает количество минорных единиц.
func (m Money) Minor() int64 { return m.minor }

// Currency возвращает валюту суммы.
func (m Money) Currency() Currency { return m.currency }

// IsZero сообщает, что сумма равна нулю.
func (m Money) IsZero() bool { return m.minor == 0 }

// IsNegative сообщает, что сумма отрицательна.
func (m Money) IsNegative() bool { return m.minor < 0 }

// IsPositive сообщает, что сумма строго положительна.
func (m Money) IsPositive() bool { return m.minor > 0 }

// sameCurrency проверяет совпадение валют для бинарных операций.
func (m Money) sameCurrency(other Money) error {
	if m.currency != other.currency {
		return fmt.Errorf("%w: %s != %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return nil
}

// Add складывает две суммы одной валюты.
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	return Money{minor: m.minor + other.minor, currency: m.currency}, nil
}

// Sub вычитает сумму той же валюты.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	return Money{minor: m.minor - other.minor, currency: m.currency}, nil
}

// Neg возвращает сумму с противоположным знаком.
func (m Money) Neg() Money {
	return Money{minor: -m.minor, currency: m.currency}
}

// Abs возвращает модуль суммы.
func (m Money) Abs() Money {
	if m.minor < 0 {
		return m.Neg()
	}
	return m
}

// Cmp сравнивает суммы: -1 если m<other, 0 если равны, 1 если m>other.
// При несовпадении валют возвращает ошибку.
func (m Money) Cmp(other Money) (int, error) {
	if err := m.sameCurrency(other); err != nil {
		return 0, err
	}
	switch {
	case m.minor < other.minor:
		return -1, nil
	case m.minor > other.minor:
		return 1, nil
	default:
		return 0, nil
	}
}

// Equal сообщает о равенстве суммы и валюты.
func (m Money) Equal(other Money) bool {
	return m.currency == other.currency && m.minor == other.minor
}

// String форматирует сумму как десятичную строку без символа валюты, например "-42.50".
func (m Money) String() string {
	return formatMinor(m.minor)
}

// Decimal возвращает десятичную строку суммы (для сериализации в vault/экспорт).
func (m Money) Decimal() string {
	return formatMinor(m.minor)
}

// Format возвращает сумму с кодом валюты, например "42.50 USD".
func (m Money) Format() string {
	return formatMinor(m.minor) + " " + string(m.currency)
}

// formatMinor превращает минорные единицы в десятичную строку с фиксированным Scale.
func formatMinor(minor int64) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	major := minor / minorPerMajor
	frac := minor % minorPerMajor

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	fmt.Fprintf(&b, "%d.%0*d", major, Scale, frac)
	return b.String()
}

// Parse разбирает десятичную строку ("42.50", "-3", "0.05") в Money заданной валюты.
// Разделитель дробной части — точка. Лишние знаки после Scale не допускаются,
// чтобы не терять точность молча.
func Parse(s string, currency Currency) (Money, error) {
	if currency.IsZero() {
		return Money{}, ErrEmptyCurrency
	}
	minor, err := parseDecimal(s)
	if err != nil {
		return Money{}, err
	}
	return Money{minor: minor, currency: currency}, nil
}

// parseDecimal переводит десятичную строку в минорные единицы без float.
func parseDecimal(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%w: пустая строка", ErrInvalidFormat)
	}

	neg := false
	switch s[0] {
	case '-':
		neg = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	if s == "" {
		return 0, fmt.Errorf("%w: нет цифр", ErrInvalidFormat)
	}

	intPart := s
	fracPart := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart = s[:i]
		fracPart = s[i+1:]
	}

	if len(fracPart) > Scale {
		return 0, fmt.Errorf("%w: слишком много знаков после запятой в %q", ErrInvalidFormat, s)
	}

	// Целая часть может быть пустой (например ".50").
	var intVal int64
	if intPart != "" {
		v, err := parseDigits(intPart)
		if err != nil {
			return 0, err
		}
		intVal = v
	}

	// Дополняем дробную часть нулями до Scale.
	fracVal := int64(0)
	if fracPart != "" {
		v, err := parseDigits(fracPart)
		if err != nil {
			return 0, err
		}
		fracVal = v
	}
	for i := len(fracPart); i < Scale; i++ {
		fracVal *= 10
	}

	minor := intVal*minorPerMajor + fracVal
	if neg {
		minor = -minor
	}
	return minor, nil
}

// parseDigits разбирает строку исключительно из цифр в int64.
func parseDigits(s string) (int64, error) {
	var v int64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("%w: недопустимый символ %q", ErrInvalidFormat, string(c))
		}
		v = v*10 + int64(c-'0')
	}
	return v, nil
}
