// Package report — value-объекты отчётов (FR-BAL, FR-REP).
//
// Все суммы в отчётах выражены в базовой валюте (BYN): пересчёт выполняется на
// уровне usecase по курсу на дату каждой операции (FR-BAL-4, FR-REP-5). Поэтому
// арифметика над money.Money здесь всегда в одной валюте.
package report

import "advisor/internal/domain/money"

// CategoryPlanVsFact — строка сравнения план/факт по одной категории (FR-BAL-1).
type CategoryPlanVsFact struct {
	CategoryID string
	Plan       money.Money // план в BYN
	Fact       money.Money // факт в BYN
}

// Deviation возвращает отклонение (факт − план). Положительное => перерасход.
func (c CategoryPlanVsFact) Deviation() money.Money {
	d, err := c.Fact.Sub(c.Plan)
	if err != nil {
		// Валюты в отчёте всегда BYN; несовпадение здесь — программная ошибка.
		panic(err)
	}
	return d
}

// Remaining возвращает «осталось до конца месяца» (план − факт) (FR-BAL-3).
// Отрицательное значение означает перерасход.
func (c CategoryPlanVsFact) Remaining() money.Money {
	r, err := c.Plan.Sub(c.Fact)
	if err != nil {
		panic(err)
	}
	return r
}

// PercentExecuted возвращает процент исполнения плана (факт/план*100), целочисленно.
// Если план нулевой, возвращает 0 при нулевом факте и 100 при ненулевом факте.
func (c CategoryPlanVsFact) PercentExecuted() int {
	plan := c.Plan.Minor()
	fact := c.Fact.Minor()
	if plan == 0 {
		if fact == 0 {
			return 0
		}
		return 100
	}
	return int(fact * 100 / plan)
}

// IsOverspent сообщает о перерасходе (факт > план).
func (c CategoryPlanVsFact) IsOverspent() bool {
	return c.Fact.Minor() > c.Plan.Minor()
}

// PlanVsFact — сводка план/факт за месяц (FR-BAL-1/2).
type PlanVsFact struct {
	Year  int
	Month int
	Lines []CategoryPlanVsFact

	PlanIncome  money.Money
	PlanExpense money.Money
	FactIncome  money.Money
	FactExpense money.Money
}

// Balance возвращает фактический остаток месяца: доход_факт − расход_факт (FR-BAL-2).
func (p PlanVsFact) Balance() money.Money {
	b, err := p.FactIncome.Sub(p.FactExpense)
	if err != nil {
		panic(err)
	}
	return b
}

// RemainingExpense возвращает «осталось до конца месяца» в целом по расходам (FR-BAL-3).
func (p PlanVsFact) RemainingExpense() money.Money {
	r, err := p.PlanExpense.Sub(p.FactExpense)
	if err != nil {
		panic(err)
	}
	return r
}

// CategoryAmount — сумма по категории за период (для разбивок отчётов).
type CategoryAmount struct {
	CategoryID string
	Amount     money.Money // в BYN
}

// PeriodSummary — агрегированный отчёт за произвольный период (FR-REP).
type PeriodSummary struct {
	From string // YYYY-MM-DD
	To   string // YYYY-MM-DD

	TotalIncome  money.Money
	TotalExpense money.Money

	// ExpenseByCategory — расходы по категориям, отсортированы usecase-слоем
	// по убыванию (для «топ категорий», FR-REP-4).
	ExpenseByCategory []CategoryAmount
	IncomeByCategory  []CategoryAmount

	// ByCurrency — разбивка исходных валют операций периода (FR-BAL-4).
	// Ключ — ISO-код, значение — суммарный оборот в этой валюте (в исходной валюте).
	ExpenseByCurrency map[money.Currency]money.Money
}

// Balance возвращает остаток за период: доход − расход.
func (s PeriodSummary) Balance() money.Money {
	b, err := s.TotalIncome.Sub(s.TotalExpense)
	if err != nil {
		panic(err)
	}
	return b
}
