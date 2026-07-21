// Package recurring — usecase генерации операций из шаблонов (FR-REC).
//
// При переходе на новый месяц активные шаблоны идемпотентно подставляются в
// план этого месяца (FR-REC-2). Опционально создаётся факт (FR-REC-3). Повторный
// вызов для того же месяца не создаёт дублей.
package recurring

import (
	"errors"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	domplan "advisor/internal/domain/plan"
	domrec "advisor/internal/domain/recurring"
	domtx "advisor/internal/domain/transaction"
)

// Service — сервис повторяющихся операций.
type Service struct {
	tmpl  ports.RecurringRepository
	plans ports.PlanRepository
	txs   ports.TransactionRepository
	clock ports.Clock
	ids   ports.IDGenerator
}

// New собирает сервис.
func New(tmpl ports.RecurringRepository, plans ports.PlanRepository, txs ports.TransactionRepository, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{tmpl: tmpl, plans: plans, txs: txs, clock: clock, ids: ids}
}

// CreateTemplate сохраняет заранее собранный шаблон (FR-REC-1/4).
func (s *Service) CreateTemplate(t *domrec.Template) error {
	return s.tmpl.Save(t)
}

// Create собирает и сохраняет шаблон из параметров (FR-REC-1/4), проставляя id и
// время создания через инфраструктурные порты (используется UI).
func (s *Service) Create(typ core.EntryType, categoryID string, amount money.Money, dayOfMonth int, start core.Date, end *core.Date, autoCreateFact bool) (*domrec.Template, error) {
	t, err := domrec.New(s.ids.NewID(), typ, categoryID, amount, dayOfMonth, start, end, autoCreateFact, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.tmpl.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update изменяет существующий шаблон (FR-REC-4).
func (s *Service) Update(id string, typ core.EntryType, categoryID string, amount money.Money, dayOfMonth int, start core.Date, end *core.Date, autoCreateFact bool) (*domrec.Template, error) {
	t, err := s.tmpl.Get(id)
	if err != nil {
		return nil, err
	}
	if err := t.Edit(typ, categoryID, amount, dayOfMonth, start, end, autoCreateFact, s.clock.Now()); err != nil {
		return nil, err
	}
	if err := s.tmpl.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

// List возвращает шаблоны (только активные при activeOnly=true) (FR-REC-4).
func (s *Service) List(activeOnly bool) ([]*domrec.Template, error) {
	return s.tmpl.List(activeOnly)
}

// Get возвращает шаблон по id.
func (s *Service) Get(id string) (*domrec.Template, error) {
	return s.tmpl.Get(id)
}

// Delete удаляет шаблон (FR-REC-4).
func (s *Service) Delete(id string) error {
	return s.tmpl.Delete(id)
}

// Pause приостанавливает шаблон (FR-REC-4).
func (s *Service) Pause(id string) error {
	t, err := s.tmpl.Get(id)
	if err != nil {
		return err
	}
	t.Pause(s.clock.Now())
	return s.tmpl.Save(t)
}

// Resume возобновляет шаблон.
func (s *Service) Resume(id string) error {
	t, err := s.tmpl.Get(id)
	if err != nil {
		return err
	}
	t.Resume(s.clock.Now())
	return s.tmpl.Save(t)
}

// GenStats — итог генерации за месяц.
type GenStats struct {
	PlansCreated int
	FactsCreated int
}

// GenerateForMonth идемпотентно подставляет активные шаблоны в план месяца
// (FR-REC-2) и при необходимости создаёт факты (FR-REC-3).
func (s *Service) GenerateForMonth(ym core.YearMonth) (GenStats, error) {
	templates, err := s.tmpl.List(true)
	if err != nil {
		return GenStats{}, err
	}

	var stats GenStats
	for _, t := range templates {
		if !t.AppliesTo(ym) {
			continue
		}
		if err := s.ensurePlan(ym, t, &stats); err != nil {
			return stats, err
		}
		if t.AutoCreateFact {
			if err := s.ensureFact(ym, t, &stats); err != nil {
				return stats, err
			}
		}
	}
	return stats, nil
}

// ensurePlan создаёт плановую позицию, если её ещё нет (идемпотентность по ключу).
func (s *Service) ensurePlan(ym core.YearMonth, t *domrec.Template, stats *GenStats) error {
	key := domplan.Key{Period: ym, CategoryID: t.CategoryID, Currency: t.Amount.Currency()}
	existing, err := s.plans.FindByKey(key)
	if err != nil && !errors.Is(err, ports.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		return nil // уже подставлено ранее — не дублируем
	}
	item, err := domplan.New(s.ids.NewID(), ym, t.CategoryID, t.Amount, "из шаблона", s.clock.Now())
	if err != nil {
		return err
	}
	if err := s.plans.Save(item); err != nil {
		return err
	}
	stats.PlansCreated++
	return nil
}

// ensureFact создаёт факт из шаблона, если факт этого шаблона в месяце ещё не создан.
func (s *Service) ensureFact(ym core.YearMonth, t *domrec.Template, stats *GenStats) error {
	monthTxs, err := s.txs.ListByMonth(ym)
	if err != nil {
		return err
	}
	for _, tx := range monthTxs {
		if tx.RecurringID == t.Meta.ID {
			return nil // факт уже создан
		}
	}
	fact, err := domtx.New(s.ids.NewID(), t.Type, t.OccurrenceDate(ym), t.CategoryID, t.Amount, "из шаблона", s.clock.Now())
	if err != nil {
		return err
	}
	fact.RecurringID = t.Meta.ID
	if err := s.txs.Save(fact); err != nil {
		return err
	}
	stats.FactsCreated++
	return nil
}
