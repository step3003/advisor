package sqlite

import (
	"database/sql"

	smssvc "advisor/internal/application/sms"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// MerchantRepo реализует sms.MerchantRepo — авто-накапливаемый справочник
// контрагентов для одного владельца.
type MerchantRepo struct {
	idx   *Index
	owner string
}

// Merchants возвращает репозиторий справочника контрагентов владельца ownerID.
func (i *Index) Merchants(ownerID string) *MerchantRepo { return &MerchantRepo{idx: i, owner: ownerID} }

// Observe регистрирует встречу контрагента: +1 к счётчику и +сумма к обороту.
// Оборот суммируется только по совпадающей валюте (чтобы не смешивать BYN и USD);
// при иной валюте счётчик всё равно растёт, а валюта записи сохраняется прежней.
func (r *MerchantRepo) Observe(name string, amount money.Money, on core.Date) error {
	d := on.String()
	_, err := r.idx.db.Exec(`INSERT INTO merchants(owner_id,name,seen_count,total_minor,currency,first_seen,last_seen)
			VALUES(?,?,1,?,?,?,?)
			ON CONFLICT(owner_id,name) DO UPDATE SET
				seen_count = seen_count + 1,
				total_minor = total_minor + CASE WHEN merchants.currency=excluded.currency THEN excluded.total_minor ELSE 0 END,
				last_seen = excluded.last_seen`,
		r.owner, name, amount.Minor(), amount.Currency().String(), d, d)
	return err
}

func (r *MerchantRepo) List() ([]*smssvc.MerchantEntry, error) {
	rows, err := r.idx.db.Query(`SELECT name,seen_count,total_minor,currency,last_seen,category_id
			FROM merchants WHERE owner_id=? ORDER BY seen_count DESC, last_seen DESC`, r.owner)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*smssvc.MerchantEntry
	for rows.Next() {
		var (
			name, cur, lastSeen, catID string
			count                      int
			minor                      int64
		)
		if err := rows.Scan(&name, &count, &minor, &cur, &lastSeen, &catID); err != nil {
			return nil, err
		}
		total, err := money.New(minor, money.Currency(cur))
		if err != nil {
			total, _ = money.New(0, money.BaseCurrency)
		}
		out = append(out, &smssvc.MerchantEntry{
			Name: name, SeenCount: count, Total: total, LastSeen: lastSeen, CategoryID: catID,
		})
	}
	return out, rows.Err()
}

// SetCategory закрепляет категорию за контрагентом (точное совпадение по имени).
// Создаёт запись, если контрагента ещё нет (ручное добавление). categoryID="" — сброс.
func (r *MerchantRepo) SetCategory(name, categoryID string) error {
	_, err := r.idx.db.Exec(`INSERT INTO merchants(owner_id,name,seen_count,total_minor,currency,first_seen,last_seen,category_id)
			VALUES(?,?,0,0,?,'','',?)
			ON CONFLICT(owner_id,name) DO UPDATE SET category_id=excluded.category_id`,
		r.owner, name, money.BaseCurrency.String(), categoryID)
	return err
}

// CategoryOf возвращает категорию, закреплённую за контрагентом (или "").
func (r *MerchantRepo) CategoryOf(name string) (string, error) {
	var catID string
	err := r.idx.db.QueryRow(`SELECT category_id FROM merchants WHERE owner_id=? AND name=?`,
		r.owner, name).Scan(&catID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return catID, nil
}
