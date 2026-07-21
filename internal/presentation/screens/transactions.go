package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/transaction"
	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/widgets"
)

// TxScreen — экран ввода фактических операций и списка последних (FR-TX).
type TxScreen struct {
	d      *Deps
	period core.YearMonth
	nav    *monthNavigator
	list   *fyne.Container
	root   *fyne.Container
	cats   []*category.Category
	idx    *CatIndex
}

// NewTxScreen создаёт экран ввода факта.
func NewTxScreen(d *Deps) *TxScreen {
	return &TxScreen{d: d, period: d.CurrentMonth()}
}

func (s *TxScreen) Title() string       { return i18n.NavAddFact }
func (s *TxScreen) Icon() fyne.Resource { return theme.ContentAddIcon() }

func (s *TxScreen) Build() fyne.CanvasObject {
	s.nav = newMonthNavigator(s.period, func(ym core.YearMonth) {
		s.period = ym
		s.Refresh()
	})
	addBtn := widget.NewButtonWithIcon(i18n.BtnAdd, theme.ContentAddIcon(), func() { s.openForm(nil) })
	s.list = container.NewVBox()
	header := container.NewVBox(
		s.nav.Container,
		addBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle(i18n.TxRecent, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	s.root = container.NewBorder(header, nil, nil, nil, container.NewVScroll(s.list))
	s.Refresh()
	return s.root
}

func (s *TxScreen) Refresh() {
	if s.list == nil {
		return
	}
	var err error
	s.cats, err = s.d.allCategories()
	if showError(s.d.Window, err) {
		return
	}
	s.idx = NewCatIndex(s.cats)

	txs, err := s.d.Ledger.ListMonth(s.period)
	if showError(s.d.Window, err) {
		return
	}
	rows := BuildTxRows(txs, s.idx)

	s.list.Objects = nil
	if len(rows) == 0 {
		s.list.Add(widget.NewLabel(i18n.TxEmpty))
		s.list.Refresh()
		return
	}
	for _, r := range rows {
		s.list.Add(s.txRow(r))
	}
	s.list.Refresh()
}

func (s *TxScreen) txRow(r TxRow) fyne.CanvasObject {
	id := r.ID
	amt := i18n.FormatMoney(r.Amount)
	amtColor := widgets.ColorExpense
	if r.Type == core.Income {
		amtColor = widgets.ColorIncome
	}
	info := makeGrid(4,
		widget.NewLabel(i18n.FormatDate(r.Date)),
		widget.NewLabel(r.CategoryName),
		coloredText(amt, amtColor, fyne.TextAlignTrailing),
		widget.NewLabel(r.Note),
	)
	edit := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() { s.editByID(id) })
	del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		confirm(s.d.Window, i18n.TxConfirmDel, func() {
			if showError(s.d.Window, s.d.Ledger.Delete(id)) {
				return
			}
			s.Refresh()
		})
	})
	return container.NewBorder(nil, nil, nil, container.NewHBox(edit, del), info)
}

func (s *TxScreen) editByID(id string) {
	t, err := s.d.Ledger.Get(id)
	if showError(s.d.Window, err) {
		return
	}
	s.openForm(t)
}

// openForm показывает форму создания/редактирования операции. tx==nil => новая.
func (s *TxScreen) openForm(tx *transaction.Transaction) {
	typeSel := widget.NewSelect([]string{i18n.TypeExpenseLabel, i18n.TypeIncomeLabel}, nil)
	amount := widgets.NewAmountEntry()
	curSel := newCurrencySelect(s.d, defaultCurrency(s.d))
	dateEntry := widget.NewEntry()
	note := widget.NewEntry()

	catHolder := container.NewStack()
	var catSel *widgets.CategorySelect
	reloadCat := func(typ core.EntryType) {
		catSel = widgets.NewCategorySelect(s.cats, typ)
		catHolder.Objects = []fyne.CanvasObject{catSel.Container}
		catHolder.Refresh()
	}
	typeSel.OnChanged = func(label string) { reloadCat(typeFromLabel(label)) }

	// Значения по умолчанию / из редактируемой операции.
	if tx == nil {
		typeSel.SetSelected(i18n.TypeExpenseLabel)
		dateEntry.SetText(s.d.Today().String())
	} else {
		typeSel.SetSelected(i18n.TypeLabel(tx.Type))
		reloadCat(tx.Type)
		catSel.SetByID(tx.CategoryID)
		amount.SetText(tx.Amount.Decimal())
		curSel.SetSelected(tx.Amount.Currency().String())
		dateEntry.SetText(tx.OccurredOn.String())
		note.SetText(tx.Note)
	}
	if catSel == nil {
		reloadCat(core.Expense)
	}

	items := []*widget.FormItem{
		{Text: i18n.LabelType, Widget: typeSel},
		{Text: i18n.LabelDate, Widget: dateEntry},
		{Text: i18n.LabelCat, Widget: catHolder},
		{Text: i18n.LabelAmount, Widget: amount},
		{Text: i18n.LabelCurr, Widget: curSel},
		{Text: i18n.LabelNote, Widget: note},
	}
	title := i18n.TxTitle
	if tx != nil {
		title = i18n.TxEditTitle
	}
	dlg := dialog.NewForm(title, i18n.BtnSave, i18n.BtnCancel, items, func(ok bool) {
		if !ok {
			return
		}
		s.submit(tx, typeFromLabel(typeSel.Selected), dateEntry.Text, catSel, amount.Text, curSel.Selected, note.Text)
	}, s.d.Window)
	dlg.Resize(fyne.NewSize(440, 420))
	dlg.Show()
}

func (s *TxScreen) submit(tx *transaction.Transaction, typ core.EntryType, dateStr string, catSel *widgets.CategorySelect, amountStr, cur, note string) {
	date, err := core.ParseDate(dateStr)
	if err != nil {
		showError(s.d.Window, errText(i18n.LabelDate+": "+dateStr))
		return
	}
	catID := catSel.SelectedID()
	if catID == "" {
		showError(s.d.Window, errText(i18n.TxNeedCategory))
		return
	}
	amt, err := widgets.ParseAmount(amountStr, money.Currency(cur))
	if err != nil {
		showError(s.d.Window, errText(i18n.MsgInvalidNum))
		return
	}
	if tx == nil {
		_, err = s.d.Ledger.Add(typ, date, catID, amt, note)
	} else {
		_, err = s.d.Ledger.Edit(tx.Meta.ID, date, catID, amt, note)
	}
	if showError(s.d.Window, err) {
		return
	}
	s.Refresh()
}

func typeFromLabel(label string) core.EntryType {
	if label == i18n.TypeIncomeLabel {
		return core.Income
	}
	return core.Expense
}
