package widgets

import (
	"errors"

	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/money"
)

// errBadAmount — валидатор поля суммы (текст — на русском для подсказки в UI).
var errBadAmount = errors.New("введите сумму, например 42.50")

// NewAmountEntry создаёт поле ввода суммы с валидацией десятичного формата.
// Значение хранится в исходной строке; парсинг в money — через ParseAmount,
// который не использует float (NFR-2).
func NewAmountEntry() *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder("0.00")
	e.Validator = func(s string) error {
		if _, err := money.Parse(s, money.BaseCurrency); err != nil {
			return errBadAmount
		}
		return nil
	}
	return e
}

// ParseAmount разбирает строку суммы в money.Money заданной валюты.
func ParseAmount(s string, cur money.Currency) (money.Money, error) {
	return money.Parse(s, cur)
}
