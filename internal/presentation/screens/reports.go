package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/core"
	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/widgets"
)

// ReportsScreen — отчёты за произвольный период с графиками (FR-REP).
type ReportsScreen struct {
	d         *Deps
	presetSel *widget.Select
	fromEntry *widget.Entry
	toEntry   *widget.Entry
	content   *fyne.Container
	root      *fyne.Container
}

// NewReportsScreen создаёт экран отчётов.
func NewReportsScreen(d *Deps) *ReportsScreen {
	return &ReportsScreen{d: d}
}

func (s *ReportsScreen) Title() string       { return i18n.NavReports }
func (s *ReportsScreen) Icon() fyne.Resource { return theme.InfoIcon() }

func (s *ReportsScreen) Build() fyne.CanvasObject {
	presets := []string{
		i18n.ReportPresetDay, i18n.ReportPresetWeek, i18n.ReportPresetMon,
		i18n.ReportPresetQ, i18n.ReportPresetYear, i18n.ReportPresetCust,
	}
	s.presetSel = widget.NewSelect(presets, func(string) { s.applyPreset() })
	s.fromEntry = widget.NewEntry()
	s.toEntry = widget.NewEntry()
	buildBtn := widget.NewButtonWithIcon(i18n.ReportBuild, theme.ViewRefreshIcon(), s.rebuild)

	controls := container.NewVBox(
		container.NewHBox(widget.NewLabel(i18n.ReportPreset), s.presetSel),
		makeGrid(4,
			widget.NewLabel(i18n.ReportFrom), s.fromEntry,
			widget.NewLabel(i18n.ReportTo), s.toEntry,
		),
		buildBtn,
		widget.NewSeparator(),
	)
	s.content = container.NewVBox()
	s.root = container.NewBorder(controls, nil, nil, nil, container.NewVScroll(s.content))

	s.presetSel.SetSelected(i18n.ReportPresetMon) // применит период и построит
	return s.root
}

func (s *ReportsScreen) Refresh() {
	if s.content != nil {
		s.rebuild()
	}
}

// applyPreset заполняет поля «с/по» по выбранному пресету и перестраивает отчёт.
func (s *ReportsScreen) applyPreset() {
	preset := s.presetSel.Selected
	editable := preset == i18n.ReportPresetCust
	if editable {
		s.fromEntry.Enable()
		s.toEntry.Enable()
		s.rebuild()
		return
	}
	from, to := presetRange(preset, s.d.Today())
	s.fromEntry.SetText(from.String())
	s.toEntry.SetText(to.String())
	s.fromEntry.Disable()
	s.toEntry.Disable()
	s.rebuild()
}

func (s *ReportsScreen) rebuild() {
	if s.content == nil {
		return
	}
	from, err := core.ParseDate(s.fromEntry.Text)
	if err != nil {
		s.showMessage(i18n.ReportBadDates)
		return
	}
	to, err := core.ParseDate(s.toEntry.Text)
	if err != nil {
		s.showMessage(i18n.ReportBadDates)
		return
	}
	if to.Before(from) {
		s.showMessage(i18n.ReportBadDates)
		return
	}

	cats, err := s.d.allCategories()
	if showError(s.d.Window, err) {
		return
	}
	idx := NewCatIndex(cats)

	sum, err := s.d.Reporting.PeriodSummary(from, to)
	if showError(s.d.Window, err) {
		return
	}
	points, err := s.d.Reporting.MonthlyDynamics(from.YearMonth(), to.YearMonth())
	if showError(s.d.Window, err) {
		return
	}

	s.content.Objects = nil
	if sum.TotalIncome.IsZero() && sum.TotalExpense.IsZero() {
		s.content.Add(widget.NewLabel(i18n.ReportEmpty))
		s.content.Refresh()
		return
	}

	// Итоги за период.
	s.content.Add(totalLine(i18n.ReportTotalInc, i18n.FormatMoney(sum.TotalIncome)))
	s.content.Add(totalLine(i18n.ReportTotalExp, i18n.FormatMoney(sum.TotalExpense)))
	s.content.Add(totalLine(i18n.ReportBalance, i18n.FormatMoney(sum.Balance())))
	s.content.Add(widget.NewSeparator())

	// Расходы по категориям (горизонтальная диаграмма).
	bars := BuildCategoryBars(sum.ExpenseByCategory, idx, 0)
	if len(bars) > 0 {
		s.content.Add(sectionTitle(i18n.ReportByCategory))
		chart := widgets.NewCategoryBarChart(bars)
		s.content.Add(sizedChart(chart, float32(len(bars))*28+16))
	}

	// Динамика по месяцам (столбчатая диаграмма).
	cols := BuildMonthlyColumns(points)
	if len(cols) > 0 {
		s.content.Add(sectionTitle(i18n.ReportDynamics))
		col := widgets.NewColumnChart(cols)
		s.content.Add(sizedChart(col, 200))
	}

	// Разбивка по исходным валютам расходов (FR-BAL-4).
	if len(sum.ExpenseByCurrency) > 0 {
		s.content.Add(widget.NewSeparator())
		s.content.Add(sectionTitle(i18n.ReportByCurrency))
		for cur, amt := range sum.ExpenseByCurrency {
			s.content.Add(totalLine(cur.String(), i18n.FormatMoney(amt)))
		}
	}
	s.content.Refresh()
}

func (s *ReportsScreen) showMessage(msg string) {
	s.content.Objects = nil
	s.content.Add(widget.NewLabel(msg))
	s.content.Refresh()
}

func sectionTitle(s string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func sizedChart(o fyne.CanvasObject, height float32) fyne.CanvasObject {
	return container.New(&fixedHeightLayout{height: height}, o)
}

// fixedHeightLayout растягивает единственный объект по ширине и фиксирует высоту.
type fixedHeightLayout struct{ height float32 }

func (l *fixedHeightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w := float32(0)
	for _, o := range objects {
		if ms := o.MinSize(); ms.Width > w {
			w = ms.Width
		}
	}
	return fyne.NewSize(w, l.height)
}

func (l *fixedHeightLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(size.Width, l.height))
		o.Move(fyne.NewPos(0, 0))
	}
}

// presetRange вычисляет диапазон дат для пресета относительно опорной даты.
func presetRange(preset string, today core.Date) (core.Date, core.Date) {
	t := today.Time()
	switch preset {
	case i18n.ReportPresetDay:
		return today, today
	case i18n.ReportPresetWeek:
		return core.DateOf(t.AddDate(0, 0, -6)), today
	case i18n.ReportPresetMon:
		ym := today.YearMonth()
		return ym.FirstDay(), lastDay(ym)
	case i18n.ReportPresetQ:
		return core.DateOf(t.AddDate(0, -2, 0)).YearMonth().FirstDay(), lastDay(today.YearMonth())
	case i18n.ReportPresetYear:
		return core.DateOf(t.AddDate(0, -11, 0)).YearMonth().FirstDay(), lastDay(today.YearMonth())
	default:
		return today, today
	}
}

func lastDay(ym core.YearMonth) core.Date {
	d, _ := core.NewDate(ym.Year, ym.Month, ym.DaysIn())
	return d
}
