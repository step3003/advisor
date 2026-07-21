package screens

import (
	"errors"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/widgets"
)

// deviationColor окрашивает отклонение: перерасход (>0) — красным, экономия
// (<0) — зелёным, ноль — нейтрально.
func deviationColor(m money.Money) color.Color {
	switch {
	case m.IsPositive():
		return widgets.ColorExpense
	case m.IsNegative():
		return widgets.ColorIncome
	default:
		return nil
	}
}

// coloredText создаёт цветной текст (nil-цвет → тема по умолчанию).
func coloredText(s string, c color.Color, align fyne.TextAlign) *canvas.Text {
	t := canvas.NewText(s, c)
	t.Alignment = align
	t.TextSize = 13
	return t
}

// monthNavigator — панель «‹ Июль 2026 ›» с колбэком смены месяца.
type monthNavigator struct {
	Container *fyne.Container
	label     *widget.Label
	current   core.YearMonth
	onChange  func(core.YearMonth)
}

func newMonthNavigator(initial core.YearMonth, onChange func(core.YearMonth)) *monthNavigator {
	n := &monthNavigator{current: initial, onChange: onChange}
	n.label = widget.NewLabelWithStyle(i18n.MonthTitle(initial), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	prev := widget.NewButton(i18n.BtnPrev, func() { n.shift(-1) })
	next := widget.NewButton(i18n.BtnNext, func() { n.shift(1) })
	n.Container = container.NewBorder(nil, nil, prev, next, n.label)
	return n
}

func (n *monthNavigator) shift(delta int) {
	if delta < 0 {
		n.current = n.current.Prev()
	} else {
		n.current = n.current.Next()
	}
	n.label.SetText(i18n.MonthTitle(n.current))
	if n.onChange != nil {
		n.onChange(n.current)
	}
}

// showError показывает диалог ошибки, если err != nil. Возвращает true при ошибке.
func showError(w fyne.Window, err error) bool {
	if err == nil {
		return false
	}
	dialog.ShowError(err, w)
	return true
}

// showInfo показывает информационный диалог.
func showInfo(w fyne.Window, title, msg string) {
	dialog.ShowInformation(title, msg, w)
}

// makeGrid раскладывает ячейки в сетку с заданным числом колонок.
func makeGrid(cols int, cells ...fyne.CanvasObject) *fyne.Container {
	return container.New(newColumnsLayout(cols), cells...)
}

// currencyOptions возвращает список ISO-кодов валют из справочника; при ошибке
// откатывается к базовой BYN.
func currencyOptions(d *Deps) []string {
	infos, err := d.Settings.ListCurrencies()
	if err != nil || len(infos) == 0 {
		return []string{"BYN"}
	}
	out := make([]string, 0, len(infos))
	for _, ci := range infos {
		out = append(out, ci.Code.String())
	}
	return out
}

// defaultCurrency возвращает валюту по умолчанию для ввода (FR-SET-1).
func defaultCurrency(d *Deps) money.Currency {
	c, err := d.Settings.DefaultCurrency()
	if err != nil || c.IsZero() {
		return money.Currency("BYN")
	}
	return c
}

// newCurrencySelect создаёт выпадающий список валют с предвыбранным значением.
func newCurrencySelect(d *Deps, selected money.Currency) *widget.Select {
	opts := currencyOptions(d)
	sel := widget.NewSelect(opts, nil)
	if !selected.IsZero() {
		sel.SetSelected(selected.String())
	} else if len(opts) > 0 {
		sel.SetSelected(opts[0])
	}
	return sel
}

// headerCell — жирная подпись столбца таблицы.
func headerCell(s string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// errText оборачивает строковое сообщение в error для показа в диалоге.
func errText(msg string) error { return errors.New(msg) }

// formDialog показывает модальную форму с кнопками «Сохранить/Отмена».
// onConfirm вызывается при подтверждении (валидацию выполняет вызывающий).
func formDialog(w fyne.Window, title string, items []*widget.FormItem, onConfirm func()) {
	d := dialog.NewForm(title, i18n.BtnSave, i18n.BtnCancel, items, func(ok bool) {
		if ok && onConfirm != nil {
			onConfirm()
		}
	}, w)
	d.Resize(fyne.NewSize(420, 320))
	d.Show()
}

// confirm показывает диалог подтверждения да/нет и вызывает onYes при согласии.
func confirm(w fyne.Window, message string, onYes func()) {
	dialog.NewConfirm(i18n.MsgConfirm, message, func(ok bool) {
		if ok && onYes != nil {
			onYes()
		}
	}, w).Show()
}
