package screens

import "fyne.io/fyne/v2"

// columnsLayout раскладывает объекты построчно в фиксированное число колонок
// равной ширины (как строки таблицы). Ячейки заполняются слева направо, при
// переполнении переносятся на следующую строку. Колонки выровнены между строками.
type columnsLayout struct {
	cols    int
	padding float32
}

func newColumnsLayout(cols int) *columnsLayout {
	if cols < 1 {
		cols = 1
	}
	return &columnsLayout{cols: cols, padding: 4}
}

func (l *columnsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	colW := make([]float32, l.cols)
	rowH := float32(0)
	rows := float32(0)
	var curRowH float32
	for i, o := range objects {
		c := i % l.cols
		ms := o.MinSize()
		if ms.Width > colW[c] {
			colW[c] = ms.Width
		}
		if ms.Height > curRowH {
			curRowH = ms.Height
		}
		if c == l.cols-1 || i == len(objects)-1 {
			rowH += curRowH
			rows++
			curRowH = 0
		}
	}
	var totalW float32
	for _, w := range colW {
		totalW += w
	}
	totalW += l.padding * float32(l.cols-1)
	if rows > 1 {
		rowH += l.padding * (rows - 1)
	}
	return fyne.NewSize(totalW, rowH)
}

func (l *columnsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	cellW := (size.Width - l.padding*float32(l.cols-1)) / float32(l.cols)
	if cellW < 0 {
		cellW = 0
	}
	// Высота строки — по максимуму минимальной высоты в строке.
	rowStart := 0
	y := float32(0)
	for rowStart < len(objects) {
		end := rowStart + l.cols
		if end > len(objects) {
			end = len(objects)
		}
		var rowH float32
		for _, o := range objects[rowStart:end] {
			if h := o.MinSize().Height; h > rowH {
				rowH = h
			}
		}
		for i := rowStart; i < end; i++ {
			c := i - rowStart
			x := float32(c) * (cellW + l.padding)
			objects[i].Move(fyne.NewPos(x, y))
			objects[i].Resize(fyne.NewSize(cellW, rowH))
		}
		y += rowH + l.padding
		rowStart = end
	}
}
