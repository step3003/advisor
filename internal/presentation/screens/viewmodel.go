// Package screens — экраны приложения (Fyne). View-model (подготовка данных для
// отображения) вынесена в чистые функции этого файла и покрыта unit-тестами;
// сами Fyne-виджеты экранов не тестируются.
package screens

import (
	"fmt"
	"sort"

	"advisor/internal/application/reporting"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/report"
	"advisor/internal/domain/transaction"
	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/widgets"
)

// CatIndex — индекс категорий по идентификатору для быстрого разрешения имён.
type CatIndex struct {
	byID map[string]*category.Category
}

// NewCatIndex строит индекс из списка категорий.
func NewCatIndex(cats []*category.Category) *CatIndex {
	m := make(map[string]*category.Category, len(cats))
	for _, c := range cats {
		m[c.Meta.ID] = c
	}
	return &CatIndex{byID: m}
}

// Get возвращает категорию по id.
func (ci *CatIndex) Get(id string) (*category.Category, bool) {
	c, ok := ci.byID[id]
	return c, ok
}

// DisplayName возвращает отображаемое имя: «Родитель / Подкатегория» для
// подкатегории и «Категория» для верхнего уровня. Для неизвестного id — сам id.
func (ci *CatIndex) DisplayName(id string) string {
	c, ok := ci.byID[id]
	if !ok {
		return id
	}
	if c.IsTopLevel() {
		return c.Name
	}
	if parent, ok := ci.byID[c.ParentID]; ok {
		return parent.Name + " / " + c.Name
	}
	return c.Name
}

// BalanceRow — строка таблицы «план/факт» по категории (FR-BAL-1).
type BalanceRow struct {
	CategoryID string
	Name       string
	Plan       money.Money
	Fact       money.Money
	Deviation  money.Money
	Remaining  money.Money
	Percent    int
	Overspent  bool
}

// BuildBalanceRows строит строки таблицы из отчёта план/факт, сортируя по имени.
func BuildBalanceRows(pvf report.PlanVsFact, ci *CatIndex) []BalanceRow {
	rows := make([]BalanceRow, 0, len(pvf.Lines))
	for _, l := range pvf.Lines {
		rows = append(rows, BalanceRow{
			CategoryID: l.CategoryID,
			Name:       ci.DisplayName(l.CategoryID),
			Plan:       l.Plan,
			Fact:       l.Fact,
			Deviation:  l.Deviation(),
			Remaining:  l.Remaining(),
			Percent:    l.PercentExecuted(),
			Overspent:  l.IsOverspent(),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Name != rows[j].Name {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].CategoryID < rows[j].CategoryID
	})
	return rows
}

// TxRow — строка списка транзакций (FR-TX).
type TxRow struct {
	ID           string
	Date         core.Date
	Type         core.EntryType
	CategoryName string
	Amount       money.Money
	Note         string
}

// BuildTxRows строит строки списка операций, сортируя по дате (свежие сверху).
func BuildTxRows(txs []*transaction.Transaction, ci *CatIndex) []TxRow {
	rows := make([]TxRow, 0, len(txs))
	for _, t := range txs {
		rows = append(rows, TxRow{
			ID:           t.Meta.ID,
			Date:         t.OccurredOn,
			Type:         t.Type,
			CategoryName: ci.DisplayName(t.CategoryID),
			Amount:       t.Amount,
			Note:         t.Note,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].Date.Equal(rows[j].Date) {
			return rows[i].Date.After(rows[j].Date)
		}
		return rows[i].ID < rows[j].ID
	})
	return rows
}

// BuildCategoryBars строит данные горизонтальной диаграммы из разбивки по
// категориям (FR-REP-2/4). limit>0 ограничивает число строк (топ категорий).
func BuildCategoryBars(items []report.CategoryAmount, ci *CatIndex, limit int) []widgets.BarDatum {
	var out []widgets.BarDatum
	for i, it := range items {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, widgets.BarDatum{
			Label: ci.DisplayName(it.CategoryID),
			Value: it.Amount.Minor(),
			Note:  i18n.FormatAmount(it.Amount),
			Color: widgets.ColorExpense,
		})
	}
	return out
}

// BuildMonthlyColumns строит группы столбцов «доход/расход по месяцам» (FR-REP-3).
func BuildMonthlyColumns(points []reporting.MonthPoint) []widgets.ColumnGroup {
	out := make([]widgets.ColumnGroup, 0, len(points))
	for _, p := range points {
		out = append(out, widgets.ColumnGroup{
			Label: monthShort(p.Period),
			Bars: []widgets.BarDatum{
				{Label: i18n.ReportIncome, Value: p.Income.Minor(), Color: widgets.ColorIncome},
				{Label: i18n.ReportExpense, Value: p.Expense.Minor(), Color: widgets.ColorExpense},
			},
		})
	}
	return out
}

// monthShort форматирует месяц компактно как «07.26» для подписи под столбцом.
func monthShort(ym core.YearMonth) string {
	return fmt.Sprintf("%02d.%02d", ym.Month, ym.Year%100)
}
