// Package widgets — переиспользуемые Fyne-виджеты (графики, поля ввода).
//
// Графики построены на canvas-примитивах Fyne (прямоугольники, текст) без
// внешних графических библиотек. Вся арифметика раскладки (доли, высоты
// столбцов) вынесена в чистые функции (Normalize) и покрыта unit-тестами;
// сами виджеты не тестируются.
package widgets

import "image/color"

// Палитра графиков (доход/расход и акцент для разбивок).
var (
	ColorIncome  = color.NRGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF} // зелёный
	ColorExpense = color.NRGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xFF} // красный
	ColorAccent  = color.NRGBA{R: 0x15, G: 0x65, B: 0xC0, A: 0xFF} // синий
	ColorTrack   = color.NRGBA{R: 0x9E, G: 0x9E, B: 0x9E, A: 0x55} // подложка столбца
)

// BarDatum — один столбец графика.
type BarDatum struct {
	Label string      // подпись (категория/месяц)
	Value int64       // значение в минорных единицах (BYN)
	Note  string      // вторичная подпись (например отформатированная сумма)
	Color color.Color // цвет столбца
}

// ColumnGroup — группа столбцов под одной подписью (например доход и расход месяца).
type ColumnGroup struct {
	Label string
	Bars  []BarDatum
}

// Normalize возвращает доли значений относительно максимального по модулю.
// Если все значения нулевые (или срез пуст) — возвращает нули соответствующей длины.
// Результат каждой доли — в диапазоне [0, 1].
func Normalize(values []int64) []float64 {
	out := make([]float64, len(values))
	var max int64
	for _, v := range values {
		if a := abs64(v); a > max {
			max = a
		}
	}
	if max == 0 {
		return out
	}
	for i, v := range values {
		out[i] = float64(abs64(v)) / float64(max)
	}
	return out
}

// normalizeGroups нормирует значения по всем группам сразу — так столбцы разных
// групп сравнимы между собой (общая шкала).
func normalizeGroups(groups []ColumnGroup) [][]float64 {
	var all []int64
	for _, g := range groups {
		for _, b := range g.Bars {
			all = append(all, b.Value)
		}
	}
	flat := Normalize(all)
	out := make([][]float64, len(groups))
	idx := 0
	for gi, g := range groups {
		out[gi] = make([]float64, len(g.Bars))
		for bi := range g.Bars {
			out[gi][bi] = flat[idx]
			idx++
		}
	}
	return out
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
