package ports

import (
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
)

// CategoryRepository — хранилище категорий.
type CategoryRepository interface {
	Save(c *category.Category) error
	Get(id string) (*category.Category, error)
	// List возвращает категории; при includeArchived=false архивные скрыты (FR-CAT-4).
	List(includeArchived bool) ([]*category.Category, error)
	// HasReferences сообщает, есть ли транзакции/планы по категории (FR-CAT-4:
	// запрет жёсткого удаления).
	HasReferences(id string) (bool, error)
	Delete(id string) error
}

// TransactionRepository — хранилище фактических операций.
type TransactionRepository interface {
	Save(t *transaction.Transaction) error
	Get(id string) (*transaction.Transaction, error)
	Delete(id string) error
	ListByMonth(ym core.YearMonth) ([]*transaction.Transaction, error)
	// ListByPeriod возвращает операции с occurred_on в [from, to] включительно.
	ListByPeriod(from, to core.Date) ([]*transaction.Transaction, error)
	// ListAll возвращает все операции (для экспорта снапшота).
	ListAll() ([]*transaction.Transaction, error)
}

// PlanRepository — хранилище плановых позиций.
type PlanRepository interface {
	Save(p *plan.PlanItem) error
	Get(id string) (*plan.PlanItem, error)
	Delete(id string) error
	ListByMonth(ym core.YearMonth) ([]*plan.PlanItem, error)
	// FindByKey ищет позицию по ключу уникальности (месяц+категория+валюта, FR-PLAN-3).
	FindByKey(key plan.Key) (*plan.PlanItem, error)
	// ListAll возвращает все плановые позиции (для экспорта снапшота).
	ListAll() ([]*plan.PlanItem, error)
}

// RecurringRepository — хранилище шаблонов повторяющихся операций.
type RecurringRepository interface {
	Save(t *recurring.Template) error
	Get(id string) (*recurring.Template, error)
	Delete(id string) error
	List(activeOnly bool) ([]*recurring.Template, error)
}
