package http

import (
	"fmt"
	"strconv"
	"strings"

	"advisor/internal/application/reporting"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/report"
	"advisor/internal/domain/transaction"
)

// moneyDTO — денежная сумма как десятичная строка + ISO-код (без float).
type moneyDTO struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func toMoney(m money.Money) moneyDTO {
	return moneyDTO{Amount: m.Decimal(), Currency: m.Currency().String()}
}

func (d moneyDTO) parse() (money.Money, error) {
	return money.Parse(d.Amount, money.Currency(strings.TrimSpace(d.Currency)))
}

// categoryDTO
type categoryDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	ParentID  string `json:"parentId,omitempty"`
	Color     string `json:"color,omitempty"`
	Icon      string `json:"icon,omitempty"`
	IsBuiltin bool   `json:"isBuiltin"`
	Archived  bool   `json:"archived"`
}

func toCategoryDTO(c *category.Category) categoryDTO {
	return categoryDTO{
		ID: c.Meta.ID, Name: c.Name, Type: string(c.Type), ParentID: c.ParentID,
		Color: c.Color, Icon: c.Icon, IsBuiltin: c.IsBuiltin, Archived: c.IsArchived(),
	}
}

func toCategoryDTOs(cs []*category.Category) []categoryDTO {
	out := make([]categoryDTO, 0, len(cs))
	for _, c := range cs {
		out = append(out, toCategoryDTO(c))
	}
	return out
}

// transactionDTO
type transactionDTO struct {
	ID          string   `json:"id"`
	Date        string   `json:"date"` // YYYY-MM-DD
	Type        string   `json:"type"` // expense|income
	CategoryID  string   `json:"categoryId"`
	Amount      moneyDTO `json:"amount"`
	Note        string   `json:"note,omitempty"`
	RecurringID string   `json:"recurringId,omitempty"`
}

func toTransactionDTO(t *transaction.Transaction) transactionDTO {
	return transactionDTO{
		ID: t.Meta.ID, Date: t.OccurredOn.String(), Type: string(t.Type),
		CategoryID: t.CategoryID, Amount: toMoney(t.Amount), Note: t.Note, RecurringID: t.RecurringID,
	}
}

func toTransactionDTOs(ts []*transaction.Transaction) []transactionDTO {
	out := make([]transactionDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, toTransactionDTO(t))
	}
	return out
}

// planItemDTO
type planItemDTO struct {
	ID         string   `json:"id"`
	Period     string   `json:"period"` // YYYY-MM
	CategoryID string   `json:"categoryId"`
	Amount     moneyDTO `json:"amount"`
	Note       string   `json:"note,omitempty"`
}

func toPlanItemDTO(p *plan.PlanItem) planItemDTO {
	return planItemDTO{
		ID: p.Meta.ID, Period: p.Period.String(), CategoryID: p.CategoryID,
		Amount: toMoney(p.Amount), Note: p.Note,
	}
}

func toPlanItemDTOs(ps []*plan.PlanItem) []planItemDTO {
	out := make([]planItemDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, toPlanItemDTO(p))
	}
	return out
}

// planVsFactDTO — таблица план/факт за месяц (FR-BAL).
type planVsFactRowDTO struct {
	CategoryID string   `json:"categoryId"`
	Plan       moneyDTO `json:"plan"`
	Fact       moneyDTO `json:"fact"`
	Deviation  moneyDTO `json:"deviation"`
	Remaining  moneyDTO `json:"remaining"`
	Percent    int      `json:"percent"`
	Overspent  bool     `json:"overspent"`
}

type planVsFactDTO struct {
	Period      string             `json:"period"`
	Rows        []planVsFactRowDTO `json:"rows"`
	PlanIncome  moneyDTO           `json:"planIncome"`
	PlanExpense moneyDTO           `json:"planExpense"`
	FactIncome  moneyDTO           `json:"factIncome"`
	FactExpense moneyDTO           `json:"factExpense"`
	Balance     moneyDTO           `json:"balance"`
	Remaining   moneyDTO           `json:"remainingExpense"`
}

func toPlanVsFactDTO(p report.PlanVsFact) planVsFactDTO {
	rows := make([]planVsFactRowDTO, 0, len(p.Lines))
	for _, l := range p.Lines {
		rows = append(rows, planVsFactRowDTO{
			CategoryID: l.CategoryID,
			Plan:       toMoney(l.Plan),
			Fact:       toMoney(l.Fact),
			Deviation:  toMoney(l.Deviation()),
			Remaining:  toMoney(l.Remaining()),
			Percent:    l.PercentExecuted(),
			Overspent:  l.IsOverspent(),
		})
	}
	return planVsFactDTO{
		Period:      fmt.Sprintf("%04d-%02d", p.Year, p.Month),
		Rows:        rows,
		PlanIncome:  toMoney(p.PlanIncome),
		PlanExpense: toMoney(p.PlanExpense),
		FactIncome:  toMoney(p.FactIncome),
		FactExpense: toMoney(p.FactExpense),
		Balance:     toMoney(p.Balance()),
		Remaining:   toMoney(p.RemainingExpense()),
	}
}

// periodSummaryDTO — отчёт за произвольный период (FR-REP).
type categoryAmountDTO struct {
	CategoryID string   `json:"categoryId"`
	Amount     moneyDTO `json:"amount"`
}

type periodSummaryDTO struct {
	From              string              `json:"from"`
	To                string              `json:"to"`
	TotalIncome       moneyDTO            `json:"totalIncome"`
	TotalExpense      moneyDTO            `json:"totalExpense"`
	Balance           moneyDTO            `json:"balance"`
	ExpenseByCategory []categoryAmountDTO `json:"expenseByCategory"`
	IncomeByCategory  []categoryAmountDTO `json:"incomeByCategory"`
	ExpenseByCurrency []moneyDTO          `json:"expenseByCurrency"`
}

func toCategoryAmountDTOs(items []report.CategoryAmount) []categoryAmountDTO {
	out := make([]categoryAmountDTO, 0, len(items))
	for _, it := range items {
		out = append(out, categoryAmountDTO{CategoryID: it.CategoryID, Amount: toMoney(it.Amount)})
	}
	return out
}

func toPeriodSummaryDTO(s report.PeriodSummary) periodSummaryDTO {
	byCur := make([]moneyDTO, 0, len(s.ExpenseByCurrency))
	for _, amt := range s.ExpenseByCurrency {
		byCur = append(byCur, toMoney(amt))
	}
	return periodSummaryDTO{
		From: s.From, To: s.To,
		TotalIncome: toMoney(s.TotalIncome), TotalExpense: toMoney(s.TotalExpense),
		Balance:           toMoney(s.Balance()),
		ExpenseByCategory: toCategoryAmountDTOs(s.ExpenseByCategory),
		IncomeByCategory:  toCategoryAmountDTOs(s.IncomeByCategory),
		ExpenseByCurrency: byCur,
	}
}

// monthPointDTO — точка динамики доход/расход по месяцам (FR-REP-3).
type monthPointDTO struct {
	Period  string   `json:"period"`
	Income  moneyDTO `json:"income"`
	Expense moneyDTO `json:"expense"`
}

func toMonthPointDTOs(points []reporting.MonthPoint) []monthPointDTO {
	out := make([]monthPointDTO, 0, len(points))
	for _, p := range points {
		out = append(out, monthPointDTO{
			Period: p.Period.String(), Income: toMoney(p.Income), Expense: toMoney(p.Expense),
		})
	}
	return out
}

// recurringDTO
type recurringDTO struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	CategoryID     string   `json:"categoryId"`
	Amount         moneyDTO `json:"amount"`
	DayOfMonth     int      `json:"dayOfMonth"`
	StartDate      string   `json:"startDate"`
	EndDate        string   `json:"endDate,omitempty"`
	AutoCreateFact bool     `json:"autoCreateFact"`
	Active         bool     `json:"active"`
}

func toRecurringDTO(t *recurring.Template) recurringDTO {
	d := recurringDTO{
		ID: t.Meta.ID, Type: string(t.Type), CategoryID: t.CategoryID,
		Amount: toMoney(t.Amount), DayOfMonth: t.DayOfMonth, StartDate: t.StartDate.String(),
		AutoCreateFact: t.AutoCreateFact, Active: t.Active,
	}
	if t.EndDate != nil {
		d.EndDate = t.EndDate.String()
	}
	return d
}

func toRecurringDTOs(ts []*recurring.Template) []recurringDTO {
	out := make([]recurringDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, toRecurringDTO(t))
	}
	return out
}

// --- Парсеры параметров ---

// parseYearMonth разбирает "YYYY-MM".
func parseYearMonth(s string) (core.YearMonth, error) {
	parts := strings.SplitN(strings.TrimSpace(s), "-", 2)
	if len(parts) != 2 {
		return core.YearMonth{}, fmt.Errorf("некорректный месяц %q (ожидается YYYY-MM)", s)
	}
	y, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return core.YearMonth{}, fmt.Errorf("некорректный месяц %q", s)
	}
	return core.NewYearMonth(y, m)
}
