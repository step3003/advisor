package sqlite

import (
	"database/sql"

	smssvc "advisor/internal/application/sms"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// --- Шаблоны SMS ---

// SMSTemplateRepo реализует sms.TemplateRepo поверх таблицы sms_templates.
type SMSTemplateRepo struct{ idx *Index }

// SMSTemplates возвращает репозиторий шаблонов SMS.
func (i *Index) SMSTemplates() *SMSTemplateRepo { return &SMSTemplateRepo{idx: i} }

func (r *SMSTemplateRepo) Save(t *smssvc.Template) error {
	_, err := r.idx.db.Exec(`INSERT INTO sms_templates(
			id,name,sender,pattern,amount_group,currency_group,fixed_currency,type,
			default_category_id,enabled,priority,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, sender=excluded.sender, pattern=excluded.pattern,
			amount_group=excluded.amount_group, currency_group=excluded.currency_group,
			fixed_currency=excluded.fixed_currency, type=excluded.type,
			default_category_id=excluded.default_category_id, enabled=excluded.enabled,
			priority=excluded.priority`,
		t.ID, t.Name, t.Sender, t.Pattern, t.AmountGroup, t.CurrencyGroup, t.FixedCurrency,
		string(t.Type), nullStr(t.DefaultCategoryID), boolToInt(t.Enabled), t.Priority,
		t.CreatedAt.UTC().Format(rfc3339))
	return err
}

func (r *SMSTemplateRepo) Get(id string) (*smssvc.Template, error) {
	row := r.idx.db.QueryRow(templateCols+` WHERE id=?`, id)
	return scanTemplate(row)
}

func (r *SMSTemplateRepo) List() ([]*smssvc.Template, error) {
	return r.query(templateCols + ` ORDER BY priority DESC, created_at ASC`)
}

func (r *SMSTemplateRepo) ListEnabled() ([]*smssvc.Template, error) {
	return r.query(templateCols + ` WHERE enabled=1 ORDER BY priority DESC, created_at ASC`)
}

func (r *SMSTemplateRepo) Delete(id string) error {
	_, err := r.idx.db.Exec(`DELETE FROM sms_templates WHERE id=?`, id)
	return err
}

const templateCols = `SELECT id,name,sender,pattern,amount_group,currency_group,fixed_currency,type,default_category_id,enabled,priority,created_at FROM sms_templates`

func (r *SMSTemplateRepo) query(q string, args ...any) ([]*smssvc.Template, error) {
	rows, err := r.idx.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*smssvc.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTemplate(sc scanner) (*smssvc.Template, error) {
	var t smssvc.Template
	var typ string
	var defCat sql.NullString
	var enabled int
	var created string
	if err := sc.Scan(&t.ID, &t.Name, &t.Sender, &t.Pattern, &t.AmountGroup, &t.CurrencyGroup,
		&t.FixedCurrency, &typ, &defCat, &enabled, &t.Priority, &created); err != nil {
		return nil, err
	}
	t.Type = core.EntryType(typ)
	t.DefaultCategoryID = defCat.String
	t.Enabled = enabled != 0
	t.CreatedAt, _ = parseTime(created)
	return &t, nil
}

// --- Входящие черновики ---

// DraftRepo реализует sms.DraftRepo поверх таблицы inbox_drafts.
type DraftRepo struct{ idx *Index }

// Drafts возвращает репозиторий входящих черновиков.
func (i *Index) Drafts() *DraftRepo { return &DraftRepo{idx: i} }

func (r *DraftRepo) Save(d *smssvc.Draft) error {
	var minor sql.NullInt64
	var cur, ptype sql.NullString
	if d.ParsedAmount != nil {
		minor = sql.NullInt64{Int64: d.ParsedAmount.Minor(), Valid: true}
		cur = sql.NullString{String: d.ParsedAmount.Currency().String(), Valid: true}
	}
	if d.ParsedType != "" {
		ptype = sql.NullString{String: string(d.ParsedType), Valid: true}
	}
	_, err := r.idx.db.Exec(`INSERT INTO inbox_drafts(
			id,source,raw_sender,raw_text,received_at,parsed_amount_minor,parsed_currency,
			parsed_type,template_id,resolved,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET resolved=excluded.resolved`,
		d.ID, d.Source, d.RawSender, d.RawText, d.ReceivedAt.String(), minor, cur, ptype,
		nullStr(d.TemplateID), boolToInt(d.Resolved), d.CreatedAt.UTC().Format(rfc3339))
	return err
}

func (r *DraftRepo) Get(id string) (*smssvc.Draft, error) {
	row := r.idx.db.QueryRow(draftCols+` WHERE id=?`, id)
	return scanDraft(row)
}

func (r *DraftRepo) List(unresolvedOnly bool) ([]*smssvc.Draft, error) {
	q := draftCols
	if unresolvedOnly {
		q += ` WHERE resolved=0`
	}
	q += ` ORDER BY received_at DESC, created_at DESC`
	rows, err := r.idx.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*smssvc.Draft
	for rows.Next() {
		d, err := scanDraft(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DraftRepo) Delete(id string) error {
	_, err := r.idx.db.Exec(`DELETE FROM inbox_drafts WHERE id=?`, id)
	return err
}

const draftCols = `SELECT id,source,raw_sender,raw_text,received_at,parsed_amount_minor,parsed_currency,parsed_type,template_id,resolved,created_at FROM inbox_drafts`

func scanDraft(sc scanner) (*smssvc.Draft, error) {
	var d smssvc.Draft
	var recv, created string
	var minor sql.NullInt64
	var cur, ptype, tmpl sql.NullString
	var resolved int
	if err := sc.Scan(&d.ID, &d.Source, &d.RawSender, &d.RawText, &recv, &minor, &cur, &ptype,
		&tmpl, &resolved, &created); err != nil {
		return nil, err
	}
	d.ReceivedAt, _ = core.ParseDate(recv)
	if minor.Valid && cur.Valid {
		m, err := money.New(minor.Int64, money.Currency(cur.String))
		if err == nil {
			d.ParsedAmount = &m
		}
	}
	d.ParsedType = core.EntryType(ptype.String)
	d.TemplateID = tmpl.String
	d.Resolved = resolved != 0
	d.CreatedAt, _ = parseTime(created)
	return &d, nil
}
