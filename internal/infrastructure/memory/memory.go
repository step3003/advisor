// Package memory — in-memory реализация портов репозиториев и кэша курсов.
//
// Не использует vault/SQLite и служит:
//   - лёгким адаптером для unit-тестов usecase (внешние зависимости за портами),
//   - опорной точкой для прототипирования.
//
// Реализация потокобезопасна на уровне грубой блокировки (для тестов достаточно).
package memory

import (
	"sort"
	"sync"

	"advisor/internal/application/ports"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
)

// Store — общее in-memory хранилище всех сущностей.
type Store struct {
	mu    sync.Mutex
	cats  map[string]*category.Category
	txs   map[string]*transaction.Transaction
	plans map[string]*plan.PlanItem
	recs  map[string]*recurring.Template
	rates map[string]ports.Rate // ключ: currency|date
}

// NewStore создаёт пустое хранилище.
func NewStore() *Store {
	return &Store{
		cats:  map[string]*category.Category{},
		txs:   map[string]*transaction.Transaction{},
		plans: map[string]*plan.PlanItem{},
		recs:  map[string]*recurring.Template{},
		rates: map[string]ports.Rate{},
	}
}

// --- Категории ---

func (s *Store) Categories() ports.CategoryRepository { return (*catRepo)(s) }

type catRepo Store

func (r *catRepo) Save(c *category.Category) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *c
	s.cats[c.Meta.ID] = &cp
	return nil
}

func (r *catRepo) Get(id string) (*category.Category, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok {
		return nil, ports.ErrRecordNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *catRepo) List(includeArchived bool) ([]*category.Category, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*category.Category
	for _, c := range s.cats {
		if !includeArchived && c.IsArchived() {
			continue
		}
		cp := *c
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out, nil
}

func (r *catRepo) HasReferences(id string) (bool, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.txs {
		if t.CategoryID == id {
			return true, nil
		}
	}
	for _, p := range s.plans {
		if p.CategoryID == id {
			return true, nil
		}
	}
	for _, t := range s.recs {
		if t.CategoryID == id {
			return true, nil
		}
	}
	for _, c := range s.cats {
		if c.ParentID == id {
			return true, nil
		}
	}
	return false, nil
}

func (r *catRepo) Delete(id string) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cats, id)
	return nil
}

// --- Транзакции ---

func (s *Store) Transactions() ports.TransactionRepository { return (*txRepo)(s) }

type txRepo Store

func (r *txRepo) Save(t *transaction.Transaction) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *t
	s.txs[t.Meta.ID] = &cp
	return nil
}

func (r *txRepo) Get(id string) (*transaction.Transaction, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.txs[id]
	if !ok {
		return nil, ports.ErrRecordNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *txRepo) Delete(id string) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.txs, id)
	return nil
}

func (r *txRepo) ListByMonth(ym core.YearMonth) ([]*transaction.Transaction, error) {
	from := ym.FirstDay()
	to := core.Date{Year: ym.Year, Month: ym.Month, Day: ym.DaysIn()}
	return r.ListByPeriod(from, to)
}

func (r *txRepo) ListByPeriod(from, to core.Date) ([]*transaction.Transaction, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*transaction.Transaction
	for _, t := range s.txs {
		if t.OccurredOn.Before(from) || t.OccurredOn.After(to) {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	sortTx(out)
	return out, nil
}

func (r *txRepo) ListAll() ([]*transaction.Transaction, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*transaction.Transaction
	for _, t := range s.txs {
		cp := *t
		out = append(out, &cp)
	}
	sortTx(out)
	return out, nil
}

func sortTx(out []*transaction.Transaction) {
	sort.Slice(out, func(i, j int) bool {
		if !out[i].OccurredOn.Equal(out[j].OccurredOn) {
			return out[i].OccurredOn.Before(out[j].OccurredOn)
		}
		return out[i].Meta.ID < out[j].Meta.ID
	})
}

// --- Планы ---

func (s *Store) Plans() ports.PlanRepository { return (*planRepo)(s) }

type planRepo Store

func (r *planRepo) Save(p *plan.PlanItem) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.plans[p.Meta.ID] = &cp
	return nil
}

func (r *planRepo) Get(id string) (*plan.PlanItem, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.plans[id]
	if !ok {
		return nil, ports.ErrRecordNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *planRepo) Delete(id string) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.plans, id)
	return nil
}

func (r *planRepo) ListByMonth(ym core.YearMonth) ([]*plan.PlanItem, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*plan.PlanItem
	for _, p := range s.plans {
		if p.Period.Equal(ym) {
			cp := *p
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out, nil
}

func (r *planRepo) FindByKey(key plan.Key) (*plan.PlanItem, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.plans {
		if p.UniqueKey() == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ports.ErrRecordNotFound
}

func (r *planRepo) ListAll() ([]*plan.PlanItem, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*plan.PlanItem
	for _, p := range s.plans {
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out, nil
}

// --- Шаблоны ---

func (s *Store) Recurring() ports.RecurringRepository { return (*recRepo)(s) }

type recRepo Store

func (r *recRepo) Save(t *recurring.Template) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *t
	s.recs[t.Meta.ID] = &cp
	return nil
}

func (r *recRepo) Get(id string) (*recurring.Template, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.recs[id]
	if !ok {
		return nil, ports.ErrRecordNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *recRepo) Delete(id string) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.recs, id)
	return nil
}

func (r *recRepo) List(activeOnly bool) ([]*recurring.Template, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*recurring.Template
	for _, t := range s.recs {
		if activeOnly && !t.Active {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out, nil
}

// --- Кэш курсов ---

func (s *Store) Rates() ports.RateCache { return (*rateRepo)(s) }

type rateRepo Store

func rateKey(cur money.Currency, d core.Date) string { return cur.String() + "|" + d.String() }

func (r *rateRepo) SaveRate(rate ports.Rate) error {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates[rateKey(rate.Currency, rate.Date)] = rate
	return nil
}

func (r *rateRepo) GetRate(cur money.Currency, d core.Date) (ports.Rate, bool, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	rate, ok := s.rates[rateKey(cur, d)]
	return rate, ok, nil
}

func (r *rateRepo) GetLatestBefore(cur money.Currency, d core.Date) (ports.Rate, bool, error) {
	s := (*Store)(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	var best ports.Rate
	found := false
	for _, rate := range s.rates {
		if rate.Currency != cur {
			continue
		}
		if rate.Date.After(d) {
			continue
		}
		if !found || rate.Date.After(best.Date) {
			best = rate
			found = true
		}
	}
	return best, found, nil
}

// compile-time проверки соответствия портам.
var (
	_ ports.CategoryRepository    = (*catRepo)(nil)
	_ ports.TransactionRepository = (*txRepo)(nil)
	_ ports.PlanRepository        = (*planRepo)(nil)
	_ ports.RecurringRepository   = (*recRepo)(nil)
	_ ports.RateCache             = (*rateRepo)(nil)
)
