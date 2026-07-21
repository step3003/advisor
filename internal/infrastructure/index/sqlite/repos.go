package sqlite

import (
	"database/sql"
	"errors"
	"path/filepath"
	"time"

	"advisor/internal/application/ports"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
)

// vaultPath строит логический путь файла записи для таблицы vault_state.
func vaultPath(ref ports.RecordRef) string {
	if ref.Partition == "" {
		return filepath.Join(ref.Collection, ref.ID+".json")
	}
	return filepath.Join(ref.Collection, ref.Partition, ref.ID+".json")
}

// upsertVaultState фиксирует состояние записи для инкрементальной пересборки.
func upsertVaultState(q execer, ref ports.RecordRef, rev int64) error {
	_, err := q.Exec(`INSERT INTO vault_state(path, collection, partition, id, rev, mtime, hash)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET rev=excluded.rev, mtime=excluded.mtime, hash=excluded.hash`,
		vaultPath(ref), ref.Collection, ref.Partition, ref.ID, rev,
		time.Now().UTC().Format(time.RFC3339), ref.Hash)
	return err
}

// execer — общий интерфейс *sql.DB и *sql.Tx.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// ---------------------------------------------------------------------------
// Категории (данные пользователя — фильтр по owner_id)
// ---------------------------------------------------------------------------

// CategoryRepo реализует ports.CategoryRepository для одного владельца.
type CategoryRepo struct {
	idx   *Index
	owner string
}

// Categories возвращает репозиторий категорий владельца ownerID.
func (i *Index) Categories(ownerID string) *CategoryRepo {
	return &CategoryRepo{idx: i, owner: ownerID}
}

func (r *CategoryRepo) Save(c *category.Category) error {
	data, err := encodeCategory(c)
	if err != nil {
		return err
	}
	ref := refForCategory(c)
	ref.Hash = ""
	if err := r.idx.vault.Put(ports.Record{RecordRef: ref, Data: data}); err != nil {
		return err
	}
	if err := upsertCategoryRow(r.idx.db, r.owner, c); err != nil {
		return err
	}
	return upsertVaultState(r.idx.db, ref, c.Meta.Rev)
}

func upsertCategoryRow(q execer, owner string, c *category.Category) error {
	var archived any
	if c.ArchivedAt != nil {
		archived = c.ArchivedAt.UTC().Format(time.RFC3339)
	}
	var parent any
	if c.ParentID != "" {
		parent = c.ParentID
	}
	_, err := q.Exec(`INSERT INTO categories(id,owner_id,name,type,parent_id,color,icon,is_builtin,archived_at,created_at,rev,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, type=excluded.type, parent_id=excluded.parent_id,
			color=excluded.color, icon=excluded.icon, is_builtin=excluded.is_builtin,
			archived_at=excluded.archived_at, rev=excluded.rev, updated_at=excluded.updated_at`,
		c.Meta.ID, owner, c.Name, string(c.Type), parent, nullStr(c.Color), nullStr(c.Icon),
		boolToInt(c.IsBuiltin), archived, c.CreatedAt.UTC().Format(time.RFC3339),
		c.Meta.Rev, c.Meta.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

const catCols = `id,name,type,parent_id,color,icon,is_builtin,archived_at,created_at,rev,updated_at`

func (r *CategoryRepo) Get(id string) (*category.Category, error) {
	row := r.idx.db.QueryRow(`SELECT `+catCols+` FROM categories WHERE id=? AND owner_id=?`, id, r.owner)
	return scanCategory(row)
}

func (r *CategoryRepo) List(includeArchived bool) ([]*category.Category, error) {
	q := `SELECT ` + catCols + ` FROM categories WHERE owner_id=?`
	if !includeArchived {
		q += ` AND archived_at IS NULL`
	}
	q += ` ORDER BY type, parent_id NULLS FIRST, name`
	rows, err := r.idx.db.Query(q, r.owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*category.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CategoryRepo) HasReferences(id string) (bool, error) {
	var n int
	err := r.idx.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM transactions WHERE category_id=? AND owner_id=?) +
		(SELECT COUNT(*) FROM plan_items WHERE category_id=? AND owner_id=?) +
		(SELECT COUNT(*) FROM recurring_templates WHERE category_id=? AND owner_id=?) +
		(SELECT COUNT(*) FROM categories WHERE parent_id=? AND owner_id=?)`,
		id, r.owner, id, r.owner, id, r.owner, id, r.owner).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *CategoryRepo) Delete(id string) error {
	if err := r.idx.vault.Delete(ports.RecordRef{Collection: ports.CollectionCategories, ID: id}); err != nil {
		return err
	}
	if _, err := r.idx.db.Exec(`DELETE FROM categories WHERE id=? AND owner_id=?`, id, r.owner); err != nil {
		return err
	}
	_, err := r.idx.db.Exec(`DELETE FROM vault_state WHERE collection=? AND id=?`, ports.CollectionCategories, id)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCategory(s scanner) (*category.Category, error) {
	var (
		id, name, typ, createdAt, updatedAt string
		parent, color, icon, archivedAt     sql.NullString
		isBuiltin                           int
		rev                                 int64
	)
	if err := s.Scan(&id, &name, &typ, &parent, &color, &icon, &isBuiltin, &archivedAt, &createdAt, &rev, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, err
	}
	created, _ := parseTime(createdAt)
	updated, _ := parseTime(updatedAt)
	c := &category.Category{
		Meta:      core.Meta{ID: id, Rev: rev, UpdatedAt: updated},
		Name:      name,
		Type:      core.EntryType(typ),
		ParentID:  parent.String,
		Color:     color.String,
		Icon:      icon.String,
		IsBuiltin: isBuiltin != 0,
		CreatedAt: created,
	}
	if archivedAt.Valid {
		t, _ := parseTime(archivedAt.String)
		c.ArchivedAt = &t
	}
	return c, nil
}

// ---------------------------------------------------------------------------
// Транзакции
// ---------------------------------------------------------------------------

// TransactionRepo реализует ports.TransactionRepository для одного владельца.
type TransactionRepo struct {
	idx   *Index
	owner string
}

// Transactions возвращает репозиторий транзакций владельца ownerID.
func (i *Index) Transactions(ownerID string) *TransactionRepo {
	return &TransactionRepo{idx: i, owner: ownerID}
}

func (r *TransactionRepo) Save(t *transaction.Transaction) error {
	data, err := encodeTransaction(t)
	if err != nil {
		return err
	}
	ref := refForTransaction(t)
	if err := r.idx.vault.Put(ports.Record{RecordRef: ref, Data: data}); err != nil {
		return err
	}
	if err := upsertTransactionRow(r.idx.db, r.owner, t); err != nil {
		return err
	}
	return upsertVaultState(r.idx.db, ref, t.Meta.Rev)
}

func upsertTransactionRow(q execer, owner string, t *transaction.Transaction) error {
	_, err := q.Exec(`INSERT INTO transactions(id,owner_id,occurred_on,type,category_id,amount_minor,currency,note,recurring_id,created_at,rev,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET occurred_on=excluded.occurred_on, type=excluded.type,
			category_id=excluded.category_id, amount_minor=excluded.amount_minor, currency=excluded.currency,
			note=excluded.note, recurring_id=excluded.recurring_id, rev=excluded.rev, updated_at=excluded.updated_at`,
		t.Meta.ID, owner, t.OccurredOn.String(), string(t.Type), t.CategoryID, t.Amount.Minor(),
		t.Amount.Currency().String(), nullStr(t.Note), nullStr(t.RecurringID),
		t.CreatedAt.UTC().Format(time.RFC3339), t.Meta.Rev, t.Meta.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

const txCols = `id,occurred_on,type,category_id,amount_minor,currency,note,recurring_id,created_at,rev,updated_at`

func (r *TransactionRepo) Get(id string) (*transaction.Transaction, error) {
	row := r.idx.db.QueryRow(`SELECT `+txCols+` FROM transactions WHERE id=? AND owner_id=?`, id, r.owner)
	return scanTransaction(row)
}

func (r *TransactionRepo) ListByMonth(ym core.YearMonth) ([]*transaction.Transaction, error) {
	from := ym.FirstDay()
	to := core.Date{Year: ym.Year, Month: ym.Month, Day: ym.DaysIn()}
	return r.ListByPeriod(from, to)
}

func (r *TransactionRepo) ListByPeriod(from, to core.Date) ([]*transaction.Transaction, error) {
	rows, err := r.idx.db.Query(`SELECT `+txCols+` FROM transactions
		WHERE owner_id=? AND occurred_on >= ? AND occurred_on <= ? ORDER BY occurred_on, created_at`,
		r.owner, from.String(), to.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTxRows(rows)
}

func (r *TransactionRepo) ListAll() ([]*transaction.Transaction, error) {
	rows, err := r.idx.db.Query(`SELECT `+txCols+` FROM transactions WHERE owner_id=? ORDER BY occurred_on, created_at`, r.owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTxRows(rows)
}

func scanTxRows(rows *sql.Rows) ([]*transaction.Transaction, error) {
	var out []*transaction.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TransactionRepo) Delete(id string) error {
	t, err := r.Get(id)
	if err != nil {
		if errors.Is(err, ports.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if err := r.idx.vault.Delete(refForTransaction(t)); err != nil {
		return err
	}
	if _, err := r.idx.db.Exec(`DELETE FROM transactions WHERE id=? AND owner_id=?`, id, r.owner); err != nil {
		return err
	}
	_, err = r.idx.db.Exec(`DELETE FROM vault_state WHERE collection=? AND id=?`, ports.CollectionTransactions, id)
	return err
}

func scanTransaction(s scanner) (*transaction.Transaction, error) {
	var (
		id, occurredOn, typ, categoryID, currency, createdAt, updatedAt string
		note, recurringID                                               sql.NullString
		amountMinor, rev                                                int64
	)
	if err := s.Scan(&id, &occurredOn, &typ, &categoryID, &amountMinor, &currency, &note, &recurringID, &createdAt, &rev, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, err
	}
	date, err := core.ParseDate(occurredOn)
	if err != nil {
		return nil, err
	}
	amount, err := money.New(amountMinor, money.Currency(currency))
	if err != nil {
		return nil, err
	}
	created, _ := parseTime(createdAt)
	updated, _ := parseTime(updatedAt)
	return &transaction.Transaction{
		Meta:        core.Meta{ID: id, Rev: rev, UpdatedAt: updated},
		Type:        core.EntryType(typ),
		OccurredOn:  date,
		CategoryID:  categoryID,
		Amount:      amount,
		Note:        note.String,
		RecurringID: recurringID.String,
		CreatedAt:   created,
	}, nil
}

// ---------------------------------------------------------------------------
// Планы
// ---------------------------------------------------------------------------

// PlanRepo реализует ports.PlanRepository для одного владельца.
type PlanRepo struct {
	idx   *Index
	owner string
}

// Plans возвращает репозиторий плановых позиций владельца ownerID.
func (i *Index) Plans(ownerID string) *PlanRepo { return &PlanRepo{idx: i, owner: ownerID} }

func (r *PlanRepo) Save(p *plan.PlanItem) error {
	data, err := encodePlan(p)
	if err != nil {
		return err
	}
	ref := refForPlan(p)
	if err := r.idx.vault.Put(ports.Record{RecordRef: ref, Data: data}); err != nil {
		return err
	}
	if err := upsertPlanRow(r.idx.db, r.owner, p); err != nil {
		return err
	}
	return upsertVaultState(r.idx.db, ref, p.Meta.Rev)
}

func upsertPlanRow(q execer, owner string, p *plan.PlanItem) error {
	_, err := q.Exec(`INSERT INTO plan_items(id,owner_id,year,month,category_id,amount_minor,currency,note,created_at,rev,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET year=excluded.year, month=excluded.month, category_id=excluded.category_id,
			amount_minor=excluded.amount_minor, currency=excluded.currency, note=excluded.note,
			rev=excluded.rev, updated_at=excluded.updated_at`,
		p.Meta.ID, owner, p.Period.Year, p.Period.Month, p.CategoryID, p.Amount.Minor(),
		p.Amount.Currency().String(), nullStr(p.Note), p.CreatedAt.UTC().Format(time.RFC3339),
		p.Meta.Rev, p.Meta.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

const planCols = `id,year,month,category_id,amount_minor,currency,note,created_at,rev,updated_at`

func (r *PlanRepo) Get(id string) (*plan.PlanItem, error) {
	row := r.idx.db.QueryRow(`SELECT `+planCols+` FROM plan_items WHERE id=? AND owner_id=?`, id, r.owner)
	return scanPlan(row)
}

func (r *PlanRepo) ListByMonth(ym core.YearMonth) ([]*plan.PlanItem, error) {
	rows, err := r.idx.db.Query(`SELECT `+planCols+` FROM plan_items WHERE owner_id=? AND year=? AND month=? ORDER BY category_id`,
		r.owner, ym.Year, ym.Month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlanRows(rows)
}

func (r *PlanRepo) FindByKey(key plan.Key) (*plan.PlanItem, error) {
	row := r.idx.db.QueryRow(`SELECT `+planCols+` FROM plan_items
		WHERE owner_id=? AND year=? AND month=? AND category_id=? AND currency=?`,
		r.owner, key.Period.Year, key.Period.Month, key.CategoryID, key.Currency.String())
	return scanPlan(row)
}

func (r *PlanRepo) ListAll() ([]*plan.PlanItem, error) {
	rows, err := r.idx.db.Query(`SELECT `+planCols+` FROM plan_items WHERE owner_id=? ORDER BY year, month, category_id`, r.owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlanRows(rows)
}

func scanPlanRows(rows *sql.Rows) ([]*plan.PlanItem, error) {
	var out []*plan.PlanItem
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PlanRepo) Delete(id string) error {
	p, err := r.Get(id)
	if err != nil {
		if errors.Is(err, ports.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if err := r.idx.vault.Delete(refForPlan(p)); err != nil {
		return err
	}
	if _, err := r.idx.db.Exec(`DELETE FROM plan_items WHERE id=? AND owner_id=?`, id, r.owner); err != nil {
		return err
	}
	_, err = r.idx.db.Exec(`DELETE FROM vault_state WHERE collection=? AND id=?`, ports.CollectionPlans, id)
	return err
}

func scanPlan(s scanner) (*plan.PlanItem, error) {
	var (
		id, categoryID, currency, createdAt, updatedAt string
		note                                           sql.NullString
		year, month                                    int
		amountMinor, rev                               int64
	)
	if err := s.Scan(&id, &year, &month, &categoryID, &amountMinor, &currency, &note, &createdAt, &rev, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, err
	}
	period, err := core.NewYearMonth(year, month)
	if err != nil {
		return nil, err
	}
	amount, err := money.New(amountMinor, money.Currency(currency))
	if err != nil {
		return nil, err
	}
	created, _ := parseTime(createdAt)
	updated, _ := parseTime(updatedAt)
	return &plan.PlanItem{
		Meta:       core.Meta{ID: id, Rev: rev, UpdatedAt: updated},
		Period:     period,
		CategoryID: categoryID,
		Amount:     amount,
		Note:       note.String,
		CreatedAt:  created,
	}, nil
}

// ---------------------------------------------------------------------------
// Шаблоны повторяющихся операций
// ---------------------------------------------------------------------------

// RecurringRepo реализует ports.RecurringRepository для одного владельца.
type RecurringRepo struct {
	idx   *Index
	owner string
}

// Recurring возвращает репозиторий шаблонов владельца ownerID.
func (i *Index) Recurring(ownerID string) *RecurringRepo {
	return &RecurringRepo{idx: i, owner: ownerID}
}

func (r *RecurringRepo) Save(t *recurring.Template) error {
	data, err := encodeRecurring(t)
	if err != nil {
		return err
	}
	ref := refForRecurring(t)
	if err := r.idx.vault.Put(ports.Record{RecordRef: ref, Data: data}); err != nil {
		return err
	}
	if err := upsertRecurringRow(r.idx.db, r.owner, t); err != nil {
		return err
	}
	return upsertVaultState(r.idx.db, ref, t.Meta.Rev)
}

func upsertRecurringRow(q execer, owner string, t *recurring.Template) error {
	var end any
	if t.EndDate != nil {
		end = t.EndDate.String()
	}
	_, err := q.Exec(`INSERT INTO recurring_templates(id,owner_id,type,category_id,amount_minor,currency,day_of_month,start_date,end_date,auto_create_fact,active,created_at,rev,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET type=excluded.type, category_id=excluded.category_id,
			amount_minor=excluded.amount_minor, currency=excluded.currency, day_of_month=excluded.day_of_month,
			start_date=excluded.start_date, end_date=excluded.end_date, auto_create_fact=excluded.auto_create_fact,
			active=excluded.active, rev=excluded.rev, updated_at=excluded.updated_at`,
		t.Meta.ID, owner, string(t.Type), t.CategoryID, t.Amount.Minor(), t.Amount.Currency().String(),
		t.DayOfMonth, t.StartDate.String(), end, boolToInt(t.AutoCreateFact), boolToInt(t.Active),
		t.CreatedAt.UTC().Format(time.RFC3339), t.Meta.Rev, t.Meta.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

const recCols = `id,type,category_id,amount_minor,currency,day_of_month,start_date,end_date,auto_create_fact,active,created_at,rev,updated_at`

func (r *RecurringRepo) Get(id string) (*recurring.Template, error) {
	row := r.idx.db.QueryRow(`SELECT `+recCols+` FROM recurring_templates WHERE id=? AND owner_id=?`, id, r.owner)
	return scanRecurring(row)
}

func (r *RecurringRepo) List(activeOnly bool) ([]*recurring.Template, error) {
	q := `SELECT ` + recCols + ` FROM recurring_templates WHERE owner_id=?`
	if activeOnly {
		q += ` AND active=1`
	}
	q += ` ORDER BY day_of_month`
	rows, err := r.idx.db.Query(q, r.owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*recurring.Template
	for rows.Next() {
		t, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *RecurringRepo) Delete(id string) error {
	if err := r.idx.vault.Delete(ports.RecordRef{Collection: ports.CollectionRecurring, ID: id}); err != nil {
		return err
	}
	if _, err := r.idx.db.Exec(`DELETE FROM recurring_templates WHERE id=? AND owner_id=?`, id, r.owner); err != nil {
		return err
	}
	_, err := r.idx.db.Exec(`DELETE FROM vault_state WHERE collection=? AND id=?`, ports.CollectionRecurring, id)
	return err
}

func scanRecurring(s scanner) (*recurring.Template, error) {
	var (
		id, typ, categoryID, currency, startDate, createdAt, updatedAt string
		endDate                                                        sql.NullString
		dayOfMonth, autoCreate, active                                 int
		amountMinor, rev                                               int64
	)
	if err := s.Scan(&id, &typ, &categoryID, &amountMinor, &currency, &dayOfMonth, &startDate, &endDate, &autoCreate, &active, &createdAt, &rev, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, err
	}
	start, err := core.ParseDate(startDate)
	if err != nil {
		return nil, err
	}
	amount, err := money.New(amountMinor, money.Currency(currency))
	if err != nil {
		return nil, err
	}
	created, _ := parseTime(createdAt)
	updated, _ := parseTime(updatedAt)
	t := &recurring.Template{
		Meta:           core.Meta{ID: id, Rev: rev, UpdatedAt: updated},
		Type:           core.EntryType(typ),
		CategoryID:     categoryID,
		Amount:         amount,
		DayOfMonth:     dayOfMonth,
		StartDate:      start,
		AutoCreateFact: autoCreate != 0,
		Active:         active != 0,
		CreatedAt:      created,
	}
	if endDate.Valid {
		d, err := core.ParseDate(endDate.String)
		if err != nil {
			return nil, err
		}
		t.EndDate = &d
	}
	return t, nil
}

// --- вспомогательные ---

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
