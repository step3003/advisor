package screens

import (
	"testing"
	"time"

	"advisor/internal/application/reporting"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/report"
	"advisor/internal/domain/transaction"
)

var vmNow = time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

func mkCat(id, name, parent string) *category.Category {
	var c *category.Category
	var err error
	if parent == "" {
		c, err = category.New(id, name, core.Expense, vmNow)
	} else {
		c, err = category.NewSub(id, name, core.Expense, parent, vmNow)
	}
	if err != nil {
		panic(err)
	}
	return c
}

func testIndex() *CatIndex {
	return NewCatIndex([]*category.Category{
		mkCat("food", "Еда", ""),
		mkCat("food-groc", "Продукты", "food"),
		mkCat("transport", "Транспорт", ""),
	})
}

func TestDisplayName(t *testing.T) {
	ci := testIndex()
	if got := ci.DisplayName("food"); got != "Еда" {
		t.Errorf("верхний уровень: %q", got)
	}
	if got := ci.DisplayName("food-groc"); got != "Еда / Продукты" {
		t.Errorf("подкатегория: %q", got)
	}
	if got := ci.DisplayName("unknown"); got != "unknown" {
		t.Errorf("неизвестная: %q", got)
	}
}

func TestBuildBalanceRowsSortedAndDerived(t *testing.T) {
	byn := func(m int64) money.Money { return money.MustNew(m, "BYN") }
	pvf := report.PlanVsFact{
		Lines: []report.CategoryPlanVsFact{
			{CategoryID: "transport", Plan: byn(10000), Fact: byn(12000)}, // перерасход
			{CategoryID: "food", Plan: byn(50000), Fact: byn(30000)},
		},
	}
	rows := BuildBalanceRows(pvf, testIndex())
	if len(rows) != 2 {
		t.Fatalf("строк: %d", len(rows))
	}
	// Отсортировано по имени: «Еда» < «Транспорт».
	if rows[0].Name != "Еда" || rows[1].Name != "Транспорт" {
		t.Fatalf("порядок: %q, %q", rows[0].Name, rows[1].Name)
	}
	// Еда: факт 300 < план 500 → отклонение -200, остаток 200, 60%.
	if rows[0].Deviation.Minor() != -20000 {
		t.Errorf("отклонение Еды: %d", rows[0].Deviation.Minor())
	}
	if rows[0].Remaining.Minor() != 20000 {
		t.Errorf("остаток Еды: %d", rows[0].Remaining.Minor())
	}
	if rows[0].Percent != 60 {
		t.Errorf("процент Еды: %d", rows[0].Percent)
	}
	if rows[0].Overspent {
		t.Error("Еда не перерасход")
	}
	// Транспорт: перерасход.
	if !rows[1].Overspent {
		t.Error("Транспорт — перерасход")
	}
}

func TestBuildTxRowsSortedByDateDesc(t *testing.T) {
	mk := func(id string, day int) *transaction.Transaction {
		d, _ := core.NewDate(2026, 7, day)
		tx, err := transaction.New(id, core.Expense, d, "food", money.MustNew(1000, "BYN"), "", vmNow)
		if err != nil {
			panic(err)
		}
		return tx
	}
	rows := BuildTxRows([]*transaction.Transaction{mk("a", 5), mk("b", 20), mk("c", 10)}, testIndex())
	if rows[0].ID != "b" || rows[1].ID != "c" || rows[2].ID != "a" {
		t.Fatalf("порядок дат: %s, %s, %s", rows[0].ID, rows[1].ID, rows[2].ID)
	}
	if rows[0].CategoryName != "Еда" {
		t.Errorf("имя категории: %q", rows[0].CategoryName)
	}
}

func TestBuildCategoryBarsLimit(t *testing.T) {
	items := []report.CategoryAmount{
		{CategoryID: "food", Amount: money.MustNew(30000, "BYN")},
		{CategoryID: "transport", Amount: money.MustNew(20000, "BYN")},
	}
	bars := BuildCategoryBars(items, testIndex(), 1)
	if len(bars) != 1 {
		t.Fatalf("ожидался лимит 1, получено %d", len(bars))
	}
	if bars[0].Label != "Еда" || bars[0].Value != 30000 {
		t.Errorf("бар: %+v", bars[0])
	}
	if bars[0].Note != "300,00" {
		t.Errorf("подпись суммы: %q", bars[0].Note)
	}
}

func TestBuildMonthlyColumns(t *testing.T) {
	points := []reporting.MonthPoint{
		{Period: core.YearMonth{Year: 2026, Month: 7}, Income: money.MustNew(100000, "BYN"), Expense: money.MustNew(60000, "BYN")},
	}
	cols := BuildMonthlyColumns(points)
	if len(cols) != 1 {
		t.Fatalf("групп: %d", len(cols))
	}
	if cols[0].Label != "07.26" {
		t.Errorf("подпись месяца: %q", cols[0].Label)
	}
	if len(cols[0].Bars) != 2 || cols[0].Bars[0].Value != 100000 || cols[0].Bars[1].Value != 60000 {
		t.Errorf("столбцы: %+v", cols[0].Bars)
	}
}

func TestCatIndexGet(t *testing.T) {
	ci := testIndex()
	if _, ok := ci.Get("food"); !ok {
		t.Error("food должен найтись")
	}
	if _, ok := ci.Get("nope"); ok {
		t.Error("nope не должен найтись")
	}
}
