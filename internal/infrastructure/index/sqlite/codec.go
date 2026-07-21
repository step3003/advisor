package sqlite

import (
	"encoding/json"
	"fmt"
	"time"

	"advisor/internal/application/ports"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
)

// Кодек преобразует доменные сущности в JSON-формат записей vault (раздел 6.1)
// и обратно. Используется репозиториями при записи в vault и при пересборке
// индекса из vault (RebuildFromVault).
//
// Деньги сериализуются как десятичная строка + ISO-код валюты (без float).

const rfc3339 = time.RFC3339

// --- Категория ---

type categoryDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	ParentID   string  `json:"parent_id,omitempty"`
	Color      string  `json:"color,omitempty"`
	Icon       string  `json:"icon,omitempty"`
	IsBuiltin  bool    `json:"is_builtin"`
	ArchivedAt *string `json:"archived_at"`
	CreatedAt  string  `json:"created_at"`
	Rev        int64   `json:"rev"`
	UpdatedAt  string  `json:"updated_at"`
}

func encodeCategory(c *category.Category) ([]byte, error) {
	dto := categoryDTO{
		ID:        c.Meta.ID,
		Name:      c.Name,
		Type:      string(c.Type),
		ParentID:  c.ParentID,
		Color:     c.Color,
		Icon:      c.Icon,
		IsBuiltin: c.IsBuiltin,
		CreatedAt: c.CreatedAt.UTC().Format(rfc3339),
		Rev:       c.Meta.Rev,
		UpdatedAt: c.Meta.UpdatedAt.UTC().Format(rfc3339),
	}
	if c.ArchivedAt != nil {
		s := c.ArchivedAt.UTC().Format(rfc3339)
		dto.ArchivedAt = &s
	}
	return json.MarshalIndent(dto, "", "  ")
}

func decodeCategory(data []byte) (*category.Category, error) {
	var dto categoryDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("codec: категория: %w", err)
	}
	createdAt, err := parseTime(dto.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(dto.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c := &category.Category{
		Meta:      core.Meta{ID: dto.ID, Rev: dto.Rev, UpdatedAt: updatedAt},
		Name:      dto.Name,
		Type:      core.EntryType(dto.Type),
		ParentID:  dto.ParentID,
		Color:     dto.Color,
		Icon:      dto.Icon,
		IsBuiltin: dto.IsBuiltin,
		CreatedAt: createdAt,
	}
	if dto.ArchivedAt != nil {
		t, err := parseTime(*dto.ArchivedAt)
		if err != nil {
			return nil, err
		}
		c.ArchivedAt = &t
	}
	return c, nil
}

func refForCategory(c *category.Category) ports.RecordRef {
	return ports.RecordRef{Collection: ports.CollectionCategories, ID: c.Meta.ID}
}

// --- Транзакция ---

type transactionDTO struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	OccurredOn  string `json:"occurred_on"`
	CategoryID  string `json:"category_id"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Note        string `json:"note,omitempty"`
	RecurringID string `json:"recurring_id,omitempty"`
	Rev         int64  `json:"rev"`
	UpdatedAt   string `json:"updated_at"`
	CreatedAt   string `json:"created_at"`
}

func encodeTransaction(t *transaction.Transaction) ([]byte, error) {
	dto := transactionDTO{
		ID:          t.Meta.ID,
		Type:        string(t.Type),
		OccurredOn:  t.OccurredOn.String(),
		CategoryID:  t.CategoryID,
		Amount:      t.Amount.Decimal(),
		Currency:    t.Amount.Currency().String(),
		Note:        t.Note,
		RecurringID: t.RecurringID,
		Rev:         t.Meta.Rev,
		UpdatedAt:   t.Meta.UpdatedAt.UTC().Format(rfc3339),
		CreatedAt:   t.CreatedAt.UTC().Format(rfc3339),
	}
	return json.MarshalIndent(dto, "", "  ")
}

func decodeTransaction(data []byte) (*transaction.Transaction, error) {
	var dto transactionDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("codec: транзакция: %w", err)
	}
	occurredOn, err := core.ParseDate(dto.OccurredOn)
	if err != nil {
		return nil, err
	}
	amount, err := money.Parse(dto.Amount, money.Currency(dto.Currency))
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(dto.UpdatedAt)
	if err != nil {
		return nil, err
	}
	createdAt, err := parseTime(dto.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &transaction.Transaction{
		Meta:        core.Meta{ID: dto.ID, Rev: dto.Rev, UpdatedAt: updatedAt},
		Type:        core.EntryType(dto.Type),
		OccurredOn:  occurredOn,
		CategoryID:  dto.CategoryID,
		Amount:      amount,
		Note:        dto.Note,
		RecurringID: dto.RecurringID,
		CreatedAt:   createdAt,
	}, nil
}

func refForTransaction(t *transaction.Transaction) ports.RecordRef {
	return ports.RecordRef{
		Collection: ports.CollectionTransactions,
		Partition:  t.OccurredOn.YearMonth().String(),
		ID:         t.Meta.ID,
	}
}

// --- Плановая позиция ---

type planDTO struct {
	ID         string `json:"id"`
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	CategoryID string `json:"category_id"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note,omitempty"`
	Rev        int64  `json:"rev"`
	UpdatedAt  string `json:"updated_at"`
	CreatedAt  string `json:"created_at"`
}

func encodePlan(p *plan.PlanItem) ([]byte, error) {
	dto := planDTO{
		ID:         p.Meta.ID,
		Year:       p.Period.Year,
		Month:      p.Period.Month,
		CategoryID: p.CategoryID,
		Amount:     p.Amount.Decimal(),
		Currency:   p.Amount.Currency().String(),
		Note:       p.Note,
		Rev:        p.Meta.Rev,
		UpdatedAt:  p.Meta.UpdatedAt.UTC().Format(rfc3339),
		CreatedAt:  p.CreatedAt.UTC().Format(rfc3339),
	}
	return json.MarshalIndent(dto, "", "  ")
}

func decodePlan(data []byte) (*plan.PlanItem, error) {
	var dto planDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("codec: план: %w", err)
	}
	period, err := core.NewYearMonth(dto.Year, dto.Month)
	if err != nil {
		return nil, err
	}
	amount, err := money.Parse(dto.Amount, money.Currency(dto.Currency))
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(dto.UpdatedAt)
	if err != nil {
		return nil, err
	}
	createdAt, err := parseTime(dto.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &plan.PlanItem{
		Meta:       core.Meta{ID: dto.ID, Rev: dto.Rev, UpdatedAt: updatedAt},
		Period:     period,
		CategoryID: dto.CategoryID,
		Amount:     amount,
		Note:       dto.Note,
		CreatedAt:  createdAt,
	}, nil
}

func refForPlan(p *plan.PlanItem) ports.RecordRef {
	return ports.RecordRef{
		Collection: ports.CollectionPlans,
		Partition:  p.Period.String(),
		ID:         p.Meta.ID,
	}
}

// --- Шаблон повторяющейся операции ---

type recurringDTO struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	CategoryID     string  `json:"category_id"`
	Amount         string  `json:"amount"`
	Currency       string  `json:"currency"`
	DayOfMonth     int     `json:"day_of_month"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date"`
	AutoCreateFact bool    `json:"auto_create_fact"`
	Active         bool    `json:"active"`
	Rev            int64   `json:"rev"`
	UpdatedAt      string  `json:"updated_at"`
	CreatedAt      string  `json:"created_at"`
}

func encodeRecurring(t *recurring.Template) ([]byte, error) {
	dto := recurringDTO{
		ID:             t.Meta.ID,
		Type:           string(t.Type),
		CategoryID:     t.CategoryID,
		Amount:         t.Amount.Decimal(),
		Currency:       t.Amount.Currency().String(),
		DayOfMonth:     t.DayOfMonth,
		StartDate:      t.StartDate.String(),
		AutoCreateFact: t.AutoCreateFact,
		Active:         t.Active,
		Rev:            t.Meta.Rev,
		UpdatedAt:      t.Meta.UpdatedAt.UTC().Format(rfc3339),
		CreatedAt:      t.CreatedAt.UTC().Format(rfc3339),
	}
	if t.EndDate != nil {
		s := t.EndDate.String()
		dto.EndDate = &s
	}
	return json.MarshalIndent(dto, "", "  ")
}

func decodeRecurring(data []byte) (*recurring.Template, error) {
	var dto recurringDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("codec: шаблон: %w", err)
	}
	start, err := core.ParseDate(dto.StartDate)
	if err != nil {
		return nil, err
	}
	amount, err := money.Parse(dto.Amount, money.Currency(dto.Currency))
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(dto.UpdatedAt)
	if err != nil {
		return nil, err
	}
	createdAt, err := parseTime(dto.CreatedAt)
	if err != nil {
		return nil, err
	}
	tpl := &recurring.Template{
		Meta:           core.Meta{ID: dto.ID, Rev: dto.Rev, UpdatedAt: updatedAt},
		Type:           core.EntryType(dto.Type),
		CategoryID:     dto.CategoryID,
		Amount:         amount,
		DayOfMonth:     dto.DayOfMonth,
		StartDate:      start,
		AutoCreateFact: dto.AutoCreateFact,
		Active:         dto.Active,
		CreatedAt:      createdAt,
	}
	if dto.EndDate != nil {
		d, err := core.ParseDate(*dto.EndDate)
		if err != nil {
			return nil, err
		}
		tpl.EndDate = &d
	}
	return tpl, nil
}

func refForRecurring(t *recurring.Template) ports.RecordRef {
	return ports.RecordRef{Collection: ports.CollectionRecurring, ID: t.Meta.ID}
}

// parseTime разбирает RFC3339-время в UTC.
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(rfc3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("codec: время %q: %w", s, err)
	}
	return t.UTC(), nil
}
