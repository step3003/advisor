package io

import (
	"time"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
)

// snapshot — версионированный контейнер снапшота (FR-IO-1).
type snapshot struct {
	FormatVersion int               `json:"format_version"`
	ExportedAt    string            `json:"exported_at"`
	BaseCurrency  string            `json:"base_currency"`
	Categories    []categoryJSON    `json:"categories"`
	Transactions  []transactionJSON `json:"transactions"`
	Plans         []planJSON        `json:"plans"`
	Recurring     []recurringJSON   `json:"recurring"`
}

const rfc = time.RFC3339

func parseT(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(rfc, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// --- Категория ---

type categoryJSON struct {
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

func toCategoryJSON(c *category.Category) categoryJSON {
	cj := categoryJSON{
		ID: c.Meta.ID, Name: c.Name, Type: string(c.Type), ParentID: c.ParentID,
		Color: c.Color, Icon: c.Icon, IsBuiltin: c.IsBuiltin,
		CreatedAt: c.CreatedAt.UTC().Format(rfc), Rev: c.Meta.Rev,
		UpdatedAt: c.Meta.UpdatedAt.UTC().Format(rfc),
	}
	if c.ArchivedAt != nil {
		s := c.ArchivedAt.UTC().Format(rfc)
		cj.ArchivedAt = &s
	}
	return cj
}

func (cj categoryJSON) toDomain() (*category.Category, error) {
	created, err := parseT(cj.CreatedAt)
	if err != nil {
		return nil, err
	}
	updated, err := parseT(cj.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c := &category.Category{
		Meta:      core.Meta{ID: cj.ID, Rev: cj.Rev, UpdatedAt: updated},
		Name:      cj.Name,
		Type:      core.EntryType(cj.Type),
		ParentID:  cj.ParentID,
		Color:     cj.Color,
		Icon:      cj.Icon,
		IsBuiltin: cj.IsBuiltin,
		CreatedAt: created,
	}
	if cj.ArchivedAt != nil {
		t, err := parseT(*cj.ArchivedAt)
		if err != nil {
			return nil, err
		}
		c.ArchivedAt = &t
	}
	return c, nil
}

// --- Транзакция ---

type transactionJSON struct {
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

func toTransactionJSON(t *transaction.Transaction) transactionJSON {
	return transactionJSON{
		ID: t.Meta.ID, Type: string(t.Type), OccurredOn: t.OccurredOn.String(),
		CategoryID: t.CategoryID, Amount: t.Amount.Decimal(), Currency: t.Amount.Currency().String(),
		Note: t.Note, RecurringID: t.RecurringID, Rev: t.Meta.Rev,
		UpdatedAt: t.Meta.UpdatedAt.UTC().Format(rfc), CreatedAt: t.CreatedAt.UTC().Format(rfc),
	}
}

func (tj transactionJSON) toDomain() (*transaction.Transaction, error) {
	date, err := core.ParseDate(tj.OccurredOn)
	if err != nil {
		return nil, err
	}
	amount, err := money.Parse(tj.Amount, money.Currency(tj.Currency))
	if err != nil {
		return nil, err
	}
	updated, err := parseT(tj.UpdatedAt)
	if err != nil {
		return nil, err
	}
	created, err := parseT(tj.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &transaction.Transaction{
		Meta:        core.Meta{ID: tj.ID, Rev: tj.Rev, UpdatedAt: updated},
		Type:        core.EntryType(tj.Type),
		OccurredOn:  date,
		CategoryID:  tj.CategoryID,
		Amount:      amount,
		Note:        tj.Note,
		RecurringID: tj.RecurringID,
		CreatedAt:   created,
	}, nil
}

// --- План ---

type planJSON struct {
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

func toPlanJSON(p *plan.PlanItem) planJSON {
	return planJSON{
		ID: p.Meta.ID, Year: p.Period.Year, Month: p.Period.Month, CategoryID: p.CategoryID,
		Amount: p.Amount.Decimal(), Currency: p.Amount.Currency().String(), Note: p.Note,
		Rev: p.Meta.Rev, UpdatedAt: p.Meta.UpdatedAt.UTC().Format(rfc), CreatedAt: p.CreatedAt.UTC().Format(rfc),
	}
}

func (pj planJSON) toDomain() (*plan.PlanItem, error) {
	period, err := core.NewYearMonth(pj.Year, pj.Month)
	if err != nil {
		return nil, err
	}
	amount, err := money.Parse(pj.Amount, money.Currency(pj.Currency))
	if err != nil {
		return nil, err
	}
	updated, err := parseT(pj.UpdatedAt)
	if err != nil {
		return nil, err
	}
	created, err := parseT(pj.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &plan.PlanItem{
		Meta:       core.Meta{ID: pj.ID, Rev: pj.Rev, UpdatedAt: updated},
		Period:     period,
		CategoryID: pj.CategoryID,
		Amount:     amount,
		Note:       pj.Note,
		CreatedAt:  created,
	}, nil
}

// --- Шаблон ---

type recurringJSON struct {
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

func toRecurringJSON(t *recurring.Template) recurringJSON {
	rj := recurringJSON{
		ID: t.Meta.ID, Type: string(t.Type), CategoryID: t.CategoryID,
		Amount: t.Amount.Decimal(), Currency: t.Amount.Currency().String(),
		DayOfMonth: t.DayOfMonth, StartDate: t.StartDate.String(),
		AutoCreateFact: t.AutoCreateFact, Active: t.Active, Rev: t.Meta.Rev,
		UpdatedAt: t.Meta.UpdatedAt.UTC().Format(rfc), CreatedAt: t.CreatedAt.UTC().Format(rfc),
	}
	if t.EndDate != nil {
		s := t.EndDate.String()
		rj.EndDate = &s
	}
	return rj
}

func (rj recurringJSON) toDomain() (*recurring.Template, error) {
	start, err := core.ParseDate(rj.StartDate)
	if err != nil {
		return nil, err
	}
	amount, err := money.Parse(rj.Amount, money.Currency(rj.Currency))
	if err != nil {
		return nil, err
	}
	updated, err := parseT(rj.UpdatedAt)
	if err != nil {
		return nil, err
	}
	created, err := parseT(rj.CreatedAt)
	if err != nil {
		return nil, err
	}
	t := &recurring.Template{
		Meta:           core.Meta{ID: rj.ID, Rev: rj.Rev, UpdatedAt: updated},
		Type:           core.EntryType(rj.Type),
		CategoryID:     rj.CategoryID,
		Amount:         amount,
		DayOfMonth:     rj.DayOfMonth,
		StartDate:      start,
		AutoCreateFact: rj.AutoCreateFact,
		Active:         rj.Active,
		CreatedAt:      created,
	}
	if rj.EndDate != nil {
		d, err := core.ParseDate(*rj.EndDate)
		if err != nil {
			return nil, err
		}
		t.EndDate = &d
	}
	return t, nil
}
