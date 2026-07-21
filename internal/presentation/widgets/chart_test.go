package widgets

import (
	"math"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestNormalizeBasic(t *testing.T) {
	got := Normalize([]int64{50, 100, 25})
	want := []float64{0.5, 1.0, 0.25}
	if len(got) != len(want) {
		t.Fatalf("длина %d", len(got))
	}
	for i := range want {
		if !almost(got[i], want[i]) {
			t.Errorf("[%d] = %v, ожидалось %v", i, got[i], want[i])
		}
	}
}

func TestNormalizeAllZero(t *testing.T) {
	got := Normalize([]int64{0, 0, 0})
	for i, v := range got {
		if v != 0 {
			t.Errorf("[%d] = %v, ожидался 0", i, v)
		}
	}
}

func TestNormalizeEmpty(t *testing.T) {
	if got := Normalize(nil); len(got) != 0 {
		t.Errorf("ожидался пустой срез, получено %v", got)
	}
}

func TestNormalizeNegative(t *testing.T) {
	got := Normalize([]int64{-100, 50})
	if !almost(got[0], 1.0) || !almost(got[1], 0.5) {
		t.Errorf("модуль не учтён: %v", got)
	}
}

func TestNormalizeGroupsSharedScale(t *testing.T) {
	groups := []ColumnGroup{
		{Label: "Янв", Bars: []BarDatum{{Value: 100}, {Value: 50}}},
		{Label: "Фев", Bars: []BarDatum{{Value: 200}, {Value: 0}}},
	}
	got := normalizeGroups(groups)
	// максимум 200 → доли относительно 200 у всех групп.
	if !almost(got[0][0], 0.5) || !almost(got[0][1], 0.25) {
		t.Errorf("группа 0: %v", got[0])
	}
	if !almost(got[1][0], 1.0) || !almost(got[1][1], 0.0) {
		t.Errorf("группа 1: %v", got[1])
	}
}
