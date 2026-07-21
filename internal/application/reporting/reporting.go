// Package reporting — usecase отчётов за произвольный период (FR-REP).
//
// Все итоги — в базовой валюте BYN (пересчёт по курсу на дату каждой операции,
// FR-REP-5), плюс разбивка по исходным валютам (FR-BAL-4). Разбивки по
// категориям сортируются по убыванию для «топ категорий» (FR-REP-4).
package reporting

import (
	"sort"

	"advisor/internal/application/currency"
	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/report"
)

// Service — сервис отчётов.
type Service struct {
	txs ports.TransactionRepository
	cur *currency.Service
}

// New собирает сервис.
func New(txs ports.TransactionRepository, cur *currency.Service) *Service {
	return &Service{txs: txs, cur: cur}
}

// PeriodSummary агрегирует операции за период [from, to] (FR-REP-1/2/4/5).
func (s *Service) PeriodSummary(from, to core.Date) (report.PeriodSummary, error) {
	txs, err := s.txs.ListByPeriod(from, to)
	if err != nil {
		return report.PeriodSummary{}, err
	}

	zero := money.Zero(money.BaseCurrency)
	out := report.PeriodSummary{
		From:              from.String(),
		To:                to.String(),
		TotalIncome:       zero,
		TotalExpense:      zero,
		ExpenseByCurrency: map[money.Currency]money.Money{},
	}
	expenseByCat := map[string]money.Money{}
	incomeByCat := map[string]money.Money{}

	for _, t := range txs {
		conv, err := s.cur.ToBase(t.Amount, t.OccurredOn)
		if err != nil {
			return report.PeriodSummary{}, err
		}
		if t.IsIncome() {
			out.TotalIncome, _ = out.TotalIncome.Add(conv.Amount)
			incomeByCat[t.CategoryID] = addTo(incomeByCat, t.CategoryID, conv.Amount, zero)
		} else {
			out.TotalExpense, _ = out.TotalExpense.Add(conv.Amount)
			expenseByCat[t.CategoryID] = addTo(expenseByCat, t.CategoryID, conv.Amount, zero)
			// Разбивка расходов по исходной валюте (FR-BAL-4).
			bucket, ok := out.ExpenseByCurrency[t.Amount.Currency()]
			if !ok {
				bucket = money.Zero(t.Amount.Currency())
			}
			out.ExpenseByCurrency[t.Amount.Currency()], _ = bucket.Add(t.Amount)
		}
	}

	out.ExpenseByCategory = sortedDesc(expenseByCat)
	out.IncomeByCategory = sortedDesc(incomeByCat)
	return out, nil
}

// MonthPoint — точка динамики по месяцам (FR-REP-3).
type MonthPoint struct {
	Period  core.YearMonth
	Income  money.Money // BYN
	Expense money.Money // BYN
}

// MonthlyDynamics возвращает доход/расход по каждому месяцу диапазона (FR-REP-3).
func (s *Service) MonthlyDynamics(from, to core.YearMonth) ([]MonthPoint, error) {
	var points []MonthPoint
	for ym := from; !to.Before(ym); ym = ym.Next() {
		firstDay := ym.FirstDay()
		lastDay := core.Date{Year: ym.Year, Month: ym.Month, Day: ym.DaysIn()}
		txs, err := s.txs.ListByPeriod(firstDay, lastDay)
		if err != nil {
			return nil, err
		}
		income := money.Zero(money.BaseCurrency)
		expense := money.Zero(money.BaseCurrency)
		for _, t := range txs {
			conv, err := s.cur.ToBase(t.Amount, t.OccurredOn)
			if err != nil {
				return nil, err
			}
			if t.IsIncome() {
				income, _ = income.Add(conv.Amount)
			} else {
				expense, _ = expense.Add(conv.Amount)
			}
		}
		points = append(points, MonthPoint{Period: ym, Income: income, Expense: expense})
	}
	return points, nil
}

func addTo(m map[string]money.Money, key string, add, zero money.Money) money.Money {
	cur, ok := m[key]
	if !ok {
		cur = zero
	}
	res, _ := cur.Add(add)
	return res
}

// sortedDesc превращает map в срез, отсортированный по убыванию суммы.
func sortedDesc(m map[string]money.Money) []report.CategoryAmount {
	out := make([]report.CategoryAmount, 0, len(m))
	for id, amt := range m {
		out = append(out, report.CategoryAmount{CategoryID: id, Amount: amt})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Amount.Minor() != out[j].Amount.Minor() {
			return out[i].Amount.Minor() > out[j].Amount.Minor()
		}
		return out[i].CategoryID < out[j].CategoryID
	})
	return out
}
