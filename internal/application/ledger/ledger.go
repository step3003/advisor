// Package ledger — usecase учёта фактических операций (FR-TX, FR-BAL).
//
// Отвечает за CRUD транзакций и расчёт баланса месяца в базовой валюте (BYN)
// через currency.Service. Зависит только от портов и домена.
package ledger

import (
	"errors"
	"fmt"

	"advisor/internal/application/currency"
	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/transaction"
)

// ErrCategoryArchived — операция на архивной категории запрещена (FR-CAT-4).
var (
	ErrCategoryArchived = errors.New("ledger: категория архивирована")
	ErrTypeMismatch     = errors.New("ledger: тип операции не совпадает с типом категории")
)

// Service — сервис учёта операций.
type Service struct {
	txs   ports.TransactionRepository
	cats  ports.CategoryRepository
	cur   *currency.Service
	clock ports.Clock
	ids   ports.IDGenerator
}

// New собирает сервис.
func New(txs ports.TransactionRepository, cats ports.CategoryRepository, cur *currency.Service, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{txs: txs, cats: cats, cur: cur, clock: clock, ids: ids}
}

// Add создаёт фактическую операцию (FR-TX-1/2).
func (s *Service) Add(typ core.EntryType, on core.Date, categoryID string, amount money.Money, note string) (*transaction.Transaction, error) {
	if err := s.validateCategory(categoryID, typ); err != nil {
		return nil, err
	}
	t, err := transaction.New(s.ids.NewID(), typ, on, categoryID, amount, note, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.txs.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

// AddFromRecurring создаёт факт, порождённый шаблоном (FR-REC-3), с привязкой recurring_id.
func (s *Service) AddFromRecurring(typ core.EntryType, on core.Date, categoryID string, amount money.Money, note, recurringID string) (*transaction.Transaction, error) {
	if err := s.validateCategory(categoryID, typ); err != nil {
		return nil, err
	}
	t, err := transaction.New(s.ids.NewID(), typ, on, categoryID, amount, note, s.clock.Now())
	if err != nil {
		return nil, err
	}
	t.RecurringID = recurringID
	if err := s.txs.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Edit изменяет операцию (FR-TX-2).
func (s *Service) Edit(id string, on core.Date, categoryID string, amount money.Money, note string) (*transaction.Transaction, error) {
	t, err := s.txs.Get(id)
	if err != nil {
		return nil, err
	}
	if err := s.validateCategory(categoryID, t.Type); err != nil {
		return nil, err
	}
	if err := t.Edit(on, categoryID, amount, note, s.clock.Now()); err != nil {
		return nil, err
	}
	if err := s.txs.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete удаляет операцию.
func (s *Service) Delete(id string) error { return s.txs.Delete(id) }

// Get возвращает операцию по id.
func (s *Service) Get(id string) (*transaction.Transaction, error) { return s.txs.Get(id) }

// ListMonth возвращает операции месяца.
func (s *Service) ListMonth(ym core.YearMonth) ([]*transaction.Transaction, error) {
	return s.txs.ListByMonth(ym)
}

func (s *Service) validateCategory(categoryID string, typ core.EntryType) error {
	c, err := s.cats.Get(categoryID)
	if err != nil {
		return fmt.Errorf("ledger: категория %s: %w", categoryID, err)
	}
	if c.IsArchived() {
		return ErrCategoryArchived
	}
	if c.Type != typ {
		return ErrTypeMismatch
	}
	return nil
}

// Balance — итоги месяца в базовой валюте (FR-BAL-2).
type Balance struct {
	Period      core.YearMonth
	FactIncome  money.Money // BYN
	FactExpense money.Money // BYN
	Approximate bool        // использовался приблизительный курс хотя бы раз (FR-CUR-4)
}

// Remainder возвращает остаток: доход − расход.
func (b Balance) Remainder() money.Money {
	r, err := b.FactIncome.Sub(b.FactExpense)
	if err != nil {
		panic(err)
	}
	return r
}

// MonthBalance считает фактические доход/расход месяца в BYN.
func (s *Service) MonthBalance(ym core.YearMonth) (Balance, error) {
	txs, err := s.txs.ListByMonth(ym)
	if err != nil {
		return Balance{}, err
	}
	income := money.Zero(money.BaseCurrency)
	expense := money.Zero(money.BaseCurrency)
	approx := false
	for _, t := range txs {
		conv, err := s.cur.ToBase(t.Amount, t.OccurredOn)
		if err != nil {
			return Balance{}, err
		}
		if conv.Approximate {
			approx = true
		}
		if t.IsIncome() {
			income, err = income.Add(conv.Amount)
		} else {
			expense, err = expense.Add(conv.Amount)
		}
		if err != nil {
			return Balance{}, err
		}
	}
	return Balance{Period: ym, FactIncome: income, FactExpense: expense, Approximate: approx}, nil
}
