package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ---------------------------------------------------------------------------
// CategoryBarChart — горизонтальная столбчатая диаграмма (разбивка по категориям).
// ---------------------------------------------------------------------------

// CategoryBarChart отображает значения строками: подпись слева, пропорциональный
// столбец в центре, отформатированное значение справа.
type CategoryBarChart struct {
	widget.BaseWidget
	data []BarDatum
}

// NewCategoryBarChart создаёт горизонтальную диаграмму.
func NewCategoryBarChart(data []BarDatum) *CategoryBarChart {
	c := &CategoryBarChart{data: data}
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer реализует fyne.Widget.
func (c *CategoryBarChart) CreateRenderer() fyne.WidgetRenderer {
	r := &hbarRenderer{chart: c}
	r.build()
	return r
}

type hbarRenderer struct {
	chart   *CategoryBarChart
	labels  []*canvas.Text
	tracks  []*canvas.Rectangle
	bars    []*canvas.Rectangle
	notes   []*canvas.Text
	objects []fyne.CanvasObject
}

func (r *hbarRenderer) build() {
	fg := theme.Color(theme.ColorNameForeground)
	r.objects = nil
	for _, d := range r.chart.data {
		lbl := canvas.NewText(d.Label, fg)
		lbl.TextSize = 12
		track := canvas.NewRectangle(ColorTrack)
		track.CornerRadius = 3
		bar := canvas.NewRectangle(barColor(d.Color))
		bar.CornerRadius = 3
		note := canvas.NewText(d.Note, fg)
		note.TextSize = 12
		note.Alignment = fyne.TextAlignTrailing

		r.labels = append(r.labels, lbl)
		r.tracks = append(r.tracks, track)
		r.bars = append(r.bars, bar)
		r.notes = append(r.notes, note)
		r.objects = append(r.objects, track, bar, lbl, note)
	}
}

func (r *hbarRenderer) Layout(size fyne.Size) {
	n := len(r.chart.data)
	if n == 0 {
		return
	}
	values := make([]int64, n)
	for i, d := range r.chart.data {
		values[i] = d.Value
	}
	fr := Normalize(values)

	const pad float32 = 4
	rowH := size.Height / float32(n)
	labelW := size.Width * 0.38
	valueW := size.Width * 0.22
	barX := labelW + pad
	barMaxW := size.Width - labelW - valueW - 2*pad
	if barMaxW < 1 {
		barMaxW = 1
	}
	barH := rowH * 0.5
	if barH > 22 {
		barH = 22
	}

	for i := range r.chart.data {
		y := float32(i) * rowH
		barY := y + (rowH-barH)/2

		r.labels[i].Move(fyne.NewPos(0, y+(rowH-16)/2))
		r.labels[i].Resize(fyne.NewSize(labelW-pad, 16))

		r.tracks[i].Move(fyne.NewPos(barX, barY))
		r.tracks[i].Resize(fyne.NewSize(barMaxW, barH))

		w := float32(fr[i]) * barMaxW
		if w < 2 && r.chart.data[i].Value != 0 {
			w = 2
		}
		r.bars[i].Move(fyne.NewPos(barX, barY))
		r.bars[i].Resize(fyne.NewSize(w, barH))

		r.notes[i].Move(fyne.NewPos(size.Width-valueW, y+(rowH-16)/2))
		r.notes[i].Resize(fyne.NewSize(valueW, 16))
	}
}

func (r *hbarRenderer) MinSize() fyne.Size {
	h := float32(len(r.chart.data)) * 26
	if h < 40 {
		h = 40
	}
	return fyne.NewSize(280, h)
}

func (r *hbarRenderer) Refresh()                     { r.build(); canvas.Refresh(r.chart) }
func (r *hbarRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *hbarRenderer) Destroy()                     {}

// ---------------------------------------------------------------------------
// ColumnChart — вертикальная сгруппированная столбчатая диаграмма (динамика).
// ---------------------------------------------------------------------------

// ColumnChart отображает группы столбцов (например доход/расход по каждому месяцу).
type ColumnChart struct {
	widget.BaseWidget
	groups []ColumnGroup
}

// NewColumnChart создаёт вертикальную сгруппированную диаграмму.
func NewColumnChart(groups []ColumnGroup) *ColumnChart {
	c := &ColumnChart{groups: groups}
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer реализует fyne.Widget.
func (c *ColumnChart) CreateRenderer() fyne.WidgetRenderer {
	r := &columnRenderer{chart: c}
	r.build()
	return r
}

type columnRenderer struct {
	chart   *ColumnChart
	bars    [][]*canvas.Rectangle
	labels  []*canvas.Text
	objects []fyne.CanvasObject
}

func (r *columnRenderer) build() {
	fg := theme.Color(theme.ColorNameForeground)
	r.objects = nil
	r.bars = nil
	r.labels = nil
	for _, g := range r.chart.groups {
		row := make([]*canvas.Rectangle, len(g.Bars))
		for bi, b := range g.Bars {
			rect := canvas.NewRectangle(barColor(b.Color))
			rect.CornerRadius = 2
			row[bi] = rect
			r.objects = append(r.objects, rect)
		}
		r.bars = append(r.bars, row)
		lbl := canvas.NewText(g.Label, fg)
		lbl.TextSize = 11
		lbl.Alignment = fyne.TextAlignCenter
		r.labels = append(r.labels, lbl)
		r.objects = append(r.objects, lbl)
	}
}

func (r *columnRenderer) Layout(size fyne.Size) {
	g := len(r.chart.groups)
	if g == 0 {
		return
	}
	fr := normalizeGroups(r.chart.groups)

	const labelH float32 = 18
	const topPad float32 = 8
	chartH := size.Height - labelH
	if chartH < 10 {
		chartH = 10
	}
	groupW := size.Width / float32(g)

	for gi := range r.chart.groups {
		b := len(r.chart.groups[gi].Bars)
		gx := float32(gi) * groupW
		var innerPad float32 = 6
		barW := (groupW - innerPad*float32(b+1)) / float32(max(b, 1))
		if barW < 3 {
			barW = 3
		}
		for bi := range r.chart.groups[gi].Bars {
			h := float32(fr[gi][bi]) * (chartH - topPad)
			if h < 2 && r.chart.groups[gi].Bars[bi].Value != 0 {
				h = 2
			}
			x := gx + innerPad + float32(bi)*(barW+innerPad)
			y := chartH - h
			r.bars[gi][bi].Move(fyne.NewPos(x, y))
			r.bars[gi][bi].Resize(fyne.NewSize(barW, h))
		}
		r.labels[gi].Move(fyne.NewPos(gx, chartH+2))
		r.labels[gi].Resize(fyne.NewSize(groupW, labelH))
	}
}

func (r *columnRenderer) MinSize() fyne.Size {
	w := float32(len(r.chart.groups)) * 56
	if w < 200 {
		w = 200
	}
	return fyne.NewSize(w, 180)
}

func (r *columnRenderer) Refresh()                     { r.build(); canvas.Refresh(r.chart) }
func (r *columnRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *columnRenderer) Destroy()                     {}

// barColor возвращает цвет столбца или акцент по умолчанию, если цвет не задан.
func barColor(c color.Color) color.Color {
	if c == nil {
		return ColorAccent
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
