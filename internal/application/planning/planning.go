// Package planning — usecase планирования на календарный месяц (FR-PLAN, FR-BAL).
//
// Управляет плановыми позициями (уникальность по месяц+категория+валюта),
// копированием плана из прошлого месяца и расчётом «план vs факт» в BYN.
package planning

import (
	"errors"

	"advisor/internal/application/currency"
	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/report"
)

// ErrCategoryArchived — планирование на архивную категорию запрещено.
var ErrCategoryArchived = errors.New("planning: категория архивирована")

// Service — сервис планирования.
type Service struct {
	plans ports.PlanRepository
	txs   ports.TransactionRepository
	cats  ports.CategoryRepository
	cur   *currency.Service
	clock ports.Clock
	ids   ports.IDGenerator
}

// New собирает сервис.
func New(plans ports.PlanRepository, txs ports.TransactionRepository, cats ports.CategoryRepository, cur *currency.Service, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{plans: plans, txs: txs, cats: cats, cur: cur, clock: clock, ids: ids}
}

// SetPlan создаёт или обновляет плановую позицию (FR-PLAN-2/3): в рамках месяца
// по паре (категория, валюта) сумма редактируется, а не дублируется.
func (s *Service) SetPlan(period core.YearMonth, categoryID string, amount money.Money, note string) (*plan.PlanItem, error) {
	c, err := s.cats.Get(categoryID)
	if err != nil {
		return nil, err
	}
	if c.IsArchived() {
		return nil, ErrCategoryArchived
	}

	key := plan.Key{Period: period, CategoryID: categoryID, Currency: amount.Currency()}
	existing, err := s.plans.FindByKey(key)
	if err != nil && !errors.Is(err, ports.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		if err := existing.SetAmount(amount, s.clock.Now()); err != nil {
			return nil, err
		}
		existing.SetNote(note, s.clock.Now())
		if err := s.plans.Save(existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	item, err := plan.New(s.ids.NewID(), period, categoryID, amount, note, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.plans.Save(item); err != nil {
		return nil, err
	}
	return item, nil
}

// ListMonth возвращает плановые позиции месяца.
func (s *Service) ListMonth(period core.YearMonth) ([]*plan.PlanItem, error) {
	return s.plans.ListByMonth(period)
}

// CopyFromPreviousMonth копирует план прошлого месяца в текущий (FR-PLAN-4).
// Идемпотентно: уже существующие позиции (по ключу) не дублируются.
func (s *Service) CopyFromPreviousMonth(period core.YearMonth) (int, error) {
	prev := period.Prev()
	items, err := s.plans.ListByMonth(prev)
	if err != nil {
		return 0, err
	}
	copied := 0
	for _, src := range items {
		key := plan.Key{Period: period, CategoryID: src.CategoryID, Currency: src.Amount.Currency()}
		existing, err := s.plans.FindByKey(key)
		if err != nil && !errors.Is(err, ports.ErrRecordNotFound) {
			return copied, err
		}
		if existing != nil {
			continue // уже есть — не дублируем
		}
		item, err := plan.New(s.ids.NewID(), period, src.CategoryID, src.Amount, src.Note, s.clock.Now())
		if err != nil {
			return copied, err
		}
		if err := s.plans.Save(item); err != nil {
			return copied, err
		}
		copied++
	}
	return copied, nil
}

// PlanVsFact строит сравнение план/факт за месяц в BYN (FR-BAL-1/2).
func (s *Service) PlanVsFact(period core.YearMonth) (report.PlanVsFact, error) {
	cats, err := s.cats.List(true)
	if err != nil {
		return report.PlanVsFact{}, err
	}
	catType := map[string]core.EntryType{}
	for _, c := range cats {
		catType[c.Meta.ID] = c.Type
	}

	plans, err := s.plans.ListByMonth(period)
	if err != nil {
		return report.PlanVsFact{}, err
	}
	txs, err := s.txs.ListByMonth(period)
	if err != nil {
		return report.PlanVsFact{}, err
	}

	zero := money.Zero(money.BaseCurrency)
	planByCat := map[string]money.Money{}
	factByCat := map[string]money.Money{}

	out := report.PlanVsFact{
		Year:        period.Year,
		Month:       period.Month,
		PlanIncome:  zero,
		PlanExpense: zero,
		FactIncome:  zero,
		FactExpense: zero,
	}

	// Плановые суммы конвертируем на первый день месяца (у плана нет даты операции).
	planDate := period.FirstDay()
	for _, p := range plans {
		conv, err := s.cur.ToBase(p.Amount, planDate)
		if err != nil {
			return report.PlanVsFact{}, err
		}
		planByCat[p.CategoryID], err = addOrInit(planByCat, p.CategoryID, conv.Amount, zero)
		if err != nil {
			return report.PlanVsFact{}, err
		}
		if catType[p.CategoryID] == core.Income {
			out.PlanIncome, _ = out.PlanIncome.Add(conv.Amount)
		} else {
			out.PlanExpense, _ = out.PlanExpense.Add(conv.Amount)
		}
	}

	for _, t := range txs {
		conv, err := s.cur.ToBase(t.Amount, t.OccurredOn)
		if err != nil {
			return report.PlanVsFact{}, err
		}
		factByCat[t.CategoryID], err = addOrInit(factByCat, t.CategoryID, conv.Amount, zero)
		if err != nil {
			return report.PlanVsFact{}, err
		}
		if t.IsIncome() {
			out.FactIncome, _ = out.FactIncome.Add(conv.Amount)
		} else {
			out.FactExpense, _ = out.FactExpense.Add(conv.Amount)
		}
	}

	// Собираем строки по всем категориям, встречающимся в плане или факте.
	seen := map[string]bool{}
	for id := range planByCat {
		seen[id] = true
	}
	for id := range factByCat {
		seen[id] = true
	}
	for id := range seen {
		out.Lines = append(out.Lines, report.CategoryPlanVsFact{
			CategoryID: id,
			Plan:       valueOr(planByCat, id, zero),
			Fact:       valueOr(factByCat, id, zero),
		})
	}
	return out, nil
}

func addOrInit(m map[string]money.Money, key string, add, zero money.Money) (money.Money, error) {
	cur, ok := m[key]
	if !ok {
		cur = zero
	}
	return cur.Add(add)
}

func valueOr(m map[string]money.Money, key string, zero money.Money) money.Money {
	if v, ok := m[key]; ok {
		return v
	}
	return zero
}
