package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/widgets"
)

// BalanceScreen — экран «План/Факт по категориям» за месяц (FR-BAL).
type BalanceScreen struct {
	d       *Deps
	period  core.YearMonth
	nav     *monthNavigator
	table   *fyne.Container
	totals  *fyne.Container
	approx  *widget.Label
	root    *fyne.Container
	catsIdx *CatIndex
}

// NewBalanceScreen создаёт экран План/Факт.
func NewBalanceScreen(d *Deps) *BalanceScreen {
	return &BalanceScreen{d: d, period: d.CurrentMonth()}
}

func (s *BalanceScreen) Title() string       { return i18n.NavBalance }
func (s *BalanceScreen) Icon() fyne.Resource { return theme.DocumentIcon() }

func (s *BalanceScreen) Build() fyne.CanvasObject {
	s.nav = newMonthNavigator(s.period, func(ym core.YearMonth) {
		s.period = ym
		s.Refresh()
	})
	copyBtn := widget.NewButtonWithIcon(i18n.BalanceCopyPrev, theme.ContentCopyIcon(), s.copyFromPrev)
	setBtn := widget.NewButtonWithIcon(i18n.BalanceSetPlan, theme.DocumentCreateIcon(), s.openSetPlan)
	actions := container.NewHBox(setBtn, copyBtn)

	s.table = container.NewVBox()
	s.totals = container.NewVBox()
	s.approx = widget.NewLabel("")
	s.approx.Hide()

	header := container.NewVBox(s.nav.Container, actions, widget.NewSeparator())
	body := container.NewVScroll(container.NewVBox(s.table, widget.NewSeparator(), s.totals, s.approx))
	s.root = container.NewBorder(header, nil, nil, nil, body)
	s.Refresh()
	return s.root
}

func (s *BalanceScreen) Refresh() {
	if s.table == nil {
		return
	}
	cats, err := s.d.allCategories()
	if err != nil {
		showError(s.d.Window, err)
		return
	}
	s.catsIdx = NewCatIndex(cats)

	pvf, err := s.d.Planning.PlanVsFact(s.period)
	if err != nil {
		showError(s.d.Window, err)
		return
	}
	rows := BuildBalanceRows(pvf, s.catsIdx)

	s.table.Objects = nil
	s.table.Add(makeGrid(6,
		headerCell(i18n.BalanceColCategory),
		headerCell(i18n.BalanceColPlan),
		headerCell(i18n.BalanceColFact),
		headerCell(i18n.BalanceColDev),
		headerCell(i18n.BalanceColPercent),
		headerCell(i18n.BalanceColRemain),
	))
	if len(rows) == 0 {
		s.table.Add(widget.NewLabel(i18n.BalanceEmpty))
	}
	for _, r := range rows {
		name := r.Name
		if r.Overspent {
			name += " ⚠"
		}
		s.table.Add(makeGrid(6,
			widget.NewLabel(name),
			coloredText(i18n.FormatMoney(r.Plan), nil, fyne.TextAlignLeading),
			coloredText(i18n.FormatMoney(r.Fact), nil, fyne.TextAlignLeading),
			coloredText(i18n.FormatSignedAmount(r.Deviation), deviationColor(r.Deviation), fyne.TextAlignLeading),
			widget.NewLabel(i18n.FormatPercent(r.Percent)),
			coloredText(i18n.FormatMoney(r.Remaining), deviationColor(r.Remaining.Neg()), fyne.TextAlignLeading),
		))
	}
	s.table.Refresh()

	s.totals.Objects = nil
	s.totals.Add(totalLine(i18n.BalancePlanIncome, i18n.FormatMoney(pvf.PlanIncome)))
	s.totals.Add(totalLine(i18n.BalancePlanExpense, i18n.FormatMoney(pvf.PlanExpense)))
	s.totals.Add(totalLine(i18n.BalanceFactIncome, i18n.FormatMoney(pvf.FactIncome)))
	s.totals.Add(totalLine(i18n.BalanceFactExpense, i18n.FormatMoney(pvf.FactExpense)))
	s.totals.Add(totalLine(i18n.BalanceRemainder, i18n.FormatMoney(pvf.Balance())))
	rem := pvf.RemainingExpense()
	remRow := container.NewHBox(
		widget.NewLabelWithStyle(i18n.BalanceRemainMonth, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		coloredText(i18n.FormatMoney(rem), deviationColor(rem.Neg()), fyne.TextAlignLeading),
	)
	s.totals.Add(remRow)
	s.totals.Refresh()
}

func totalLine(label, value string) fyne.CanvasObject {
	return container.NewHBox(
		widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(value),
	)
}

func (s *BalanceScreen) copyFromPrev() {
	n, err := s.d.Planning.CopyFromPreviousMonth(s.period)
	if showError(s.d.Window, err) {
		return
	}
	s.Refresh()
	showInfo(s.d.Window, i18n.MsgDone, i18n.MonthTitle(s.period))
	_ = n
}

// openSetPlan открывает форму задания/изменения плана по категории (FR-PLAN-2/3).
func (s *BalanceScreen) openSetPlan() {
	cats, err := s.d.activeCategories()
	if showError(s.d.Window, err) {
		return
	}
	catSel := widgets.NewCategorySelect(cats, core.Expense)
	amount := widgets.NewAmountEntry()
	curSel := newCurrencySelect(s.d, defaultCurrency(s.d))
	note := widget.NewEntry()

	items := []*widget.FormItem{
		{Text: i18n.LabelCat, Widget: catSel.Container},
		{Text: i18n.LabelAmount, Widget: amount},
		{Text: i18n.LabelCurr, Widget: curSel},
		{Text: i18n.LabelNote, Widget: note},
	}
	formDialog(s.d.Window, i18n.BalanceEditPlan, items, func() {
		catID := catSel.SelectedID()
		if catID == "" {
			showError(s.d.Window, errText(i18n.TxNeedCategory))
			return
		}
		amt, err := widgets.ParseAmount(amount.Text, money.Currency(curSel.Selected))
		if err != nil {
			showError(s.d.Window, errText(i18n.MsgInvalidNum))
			return
		}
		if _, err := s.d.Planning.SetPlan(s.period, catID, amt, note.Text); showError(s.d.Window, err) {
			return
		}
		s.Refresh()
	})
}
