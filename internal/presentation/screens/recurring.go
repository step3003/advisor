package screens

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	domrec "advisor/internal/domain/recurring"
	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/widgets"
)

// RecurringScreen — шаблоны повторяющихся операций (FR-REC).
type RecurringScreen struct {
	d    *Deps
	list *fyne.Container
	root *fyne.Container
	cats []*category.Category
	idx  *CatIndex
}

// NewRecurringScreen создаёт экран повторяющихся операций.
func NewRecurringScreen(d *Deps) *RecurringScreen {
	return &RecurringScreen{d: d}
}

func (s *RecurringScreen) Title() string       { return i18n.NavRecurring }
func (s *RecurringScreen) Icon() fyne.Resource { return theme.ViewRefreshIcon() }

func (s *RecurringScreen) Build() fyne.CanvasObject {
	addBtn := widget.NewButtonWithIcon(i18n.RecAdd, theme.ContentAddIcon(), func() { s.openForm(nil) })
	s.list = container.NewVBox()
	header := container.NewVBox(addBtn, widget.NewSeparator())
	s.root = container.NewBorder(header, nil, nil, nil, container.NewVScroll(s.list))
	s.Refresh()
	return s.root
}

func (s *RecurringScreen) Refresh() {
	if s.list == nil {
		return
	}
	var err error
	s.cats, err = s.d.allCategories()
	if showError(s.d.Window, err) {
		return
	}
	s.idx = NewCatIndex(s.cats)

	tmpls, err := s.d.Recurring.List(false)
	if showError(s.d.Window, err) {
		return
	}
	s.list.Objects = nil
	if len(tmpls) == 0 {
		s.list.Add(widget.NewLabel(i18n.RecEmpty))
		s.list.Refresh()
		return
	}
	s.list.Add(makeGrid(6,
		headerCell(i18n.RecColType), headerCell(i18n.RecColCategory), headerCell(i18n.RecColAmount),
		headerCell(i18n.RecColDay), headerCell(i18n.RecColPeriod), headerCell(i18n.RecColStatus),
	))
	for _, t := range tmpls {
		s.list.Add(s.tmplRow(t))
	}
	s.list.Refresh()
}

func (s *RecurringScreen) tmplRow(t *domrec.Template) fyne.CanvasObject {
	status := i18n.RecActive
	if !t.Active {
		status = i18n.RecPaused
	}
	period := t.StartDate.String() + " – " + i18n.RecOpenEnded
	if t.EndDate != nil {
		period = t.StartDate.String() + " – " + t.EndDate.String()
	}
	info := makeGrid(6,
		widget.NewLabel(i18n.TypeLabel(t.Type)),
		widget.NewLabel(s.idx.DisplayName(t.CategoryID)),
		widget.NewLabel(i18n.FormatMoney(t.Amount)),
		widget.NewLabel(strconv.Itoa(t.DayOfMonth)),
		widget.NewLabel(period),
		widget.NewLabel(status),
	)

	id := t.Meta.ID
	edit := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() { s.openForm(t) })
	var toggle *widget.Button
	if t.Active {
		toggle = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
			if !showError(s.d.Window, s.d.Recurring.Pause(id)) {
				s.Refresh()
			}
		})
	} else {
		toggle = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
			if !showError(s.d.Window, s.d.Recurring.Resume(id)) {
				s.Refresh()
			}
		})
	}
	del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		confirm(s.d.Window, i18n.RecConfirmDelete, func() {
			if !showError(s.d.Window, s.d.Recurring.Delete(id)) {
				s.Refresh()
			}
		})
	})
	return container.NewBorder(nil, nil, nil, container.NewHBox(edit, toggle, del), info)
}

// openForm показывает форму создания/редактирования шаблона. t==nil => новый.
func (s *RecurringScreen) openForm(t *domrec.Template) {
	typeSel := widget.NewSelect([]string{i18n.TypeExpenseLabel, i18n.TypeIncomeLabel}, nil)
	amount := widgets.NewAmountEntry()
	curSel := newCurrencySelect(s.d, defaultCurrency(s.d))
	dayEntry := widget.NewEntry()
	startEntry := widget.NewEntry()
	endEntry := widget.NewEntry()
	autoFact := widget.NewCheck(i18n.RecAutoFact, nil)

	catHolder := container.NewStack()
	var catSel *widgets.CategorySelect
	reloadCat := func(typ core.EntryType) {
		catSel = widgets.NewCategorySelect(s.cats, typ)
		catHolder.Objects = []fyne.CanvasObject{catSel.Container}
		catHolder.Refresh()
	}
	typeSel.OnChanged = func(label string) { reloadCat(typeFromLabel(label)) }

	if t == nil {
		typeSel.SetSelected(i18n.TypeExpenseLabel)
		dayEntry.SetText("1")
		startEntry.SetText(s.d.Today().String())
	} else {
		typeSel.SetSelected(i18n.TypeLabel(t.Type))
		reloadCat(t.Type)
		catSel.SetByID(t.CategoryID)
		amount.SetText(t.Amount.Decimal())
		curSel.SetSelected(t.Amount.Currency().String())
		dayEntry.SetText(strconv.Itoa(t.DayOfMonth))
		startEntry.SetText(t.StartDate.String())
		if t.EndDate != nil {
			endEntry.SetText(t.EndDate.String())
		}
		autoFact.SetChecked(t.AutoCreateFact)
	}
	if catSel == nil {
		reloadCat(core.Expense)
	}

	items := []*widget.FormItem{
		{Text: i18n.LabelType, Widget: typeSel},
		{Text: i18n.LabelCat, Widget: catHolder},
		{Text: i18n.LabelAmount, Widget: amount},
		{Text: i18n.LabelCurr, Widget: curSel},
		{Text: i18n.RecDay, Widget: dayEntry},
		{Text: i18n.RecStart, Widget: startEntry},
		{Text: i18n.RecEnd, Widget: endEntry},
		{Text: "", Widget: autoFact},
	}
	dlg := dialog.NewForm(i18n.RecEditTitle, i18n.BtnSave, i18n.BtnCancel, items, func(ok bool) {
		if !ok {
			return
		}
		s.submit(t, recFormData{
			typ:      typeFromLabel(typeSel.Selected),
			catSel:   catSel,
			amount:   amount.Text,
			cur:      curSel.Selected,
			day:      dayEntry.Text,
			start:    startEntry.Text,
			end:      endEntry.Text,
			autoFact: autoFact.Checked,
		})
	}, s.d.Window)
	dlg.Resize(fyne.NewSize(440, 480))
	dlg.Show()
}

type recFormData struct {
	typ      core.EntryType
	catSel   *widgets.CategorySelect
	amount   string
	cur      string
	day      string
	start    string
	end      string
	autoFact bool
}

func (s *RecurringScreen) submit(t *domrec.Template, f recFormData) {
	catID := f.catSel.SelectedID()
	if catID == "" {
		showError(s.d.Window, errText(i18n.TxNeedCategory))
		return
	}
	amt, err := widgets.ParseAmount(f.amount, money.Currency(f.cur))
	if err != nil {
		showError(s.d.Window, errText(i18n.MsgInvalidNum))
		return
	}
	day, err := strconv.Atoi(f.day)
	if err != nil || day < 1 || day > 28 {
		showError(s.d.Window, errText(i18n.RecDay))
		return
	}
	start, err := core.ParseDate(f.start)
	if err != nil {
		showError(s.d.Window, errText(i18n.RecStart+": "+f.start))
		return
	}
	var end *core.Date
	if f.end != "" {
		e, err := core.ParseDate(f.end)
		if err != nil {
			showError(s.d.Window, errText(i18n.RecEnd+": "+f.end))
			return
		}
		end = &e
	}

	if t == nil {
		_, err = s.d.Recurring.Create(f.typ, catID, amt, day, start, end, f.autoFact)
	} else {
		_, err = s.d.Recurring.Update(t.Meta.ID, f.typ, catID, amt, day, start, end, f.autoFact)
	}
	if showError(s.d.Window, err) {
		return
	}
	s.Refresh()
}
