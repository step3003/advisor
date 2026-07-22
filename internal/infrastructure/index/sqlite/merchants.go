package sqlite

import (
	"database/sql"

	smssvc "advisor/internal/application/sms"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// MerchantRepo реализует sms.MerchantRepo — строгий список распознавания
// (контрагенты и счета) для одного владельца.
type MerchantRepo struct {
	idx   *Index
	owner string
}

// Merchants возвращает репозиторий строгого списка распознавания владельца ownerID.
func (i *Index) Merchants(ownerID string) *MerchantRepo { return &MerchantRepo{idx: i, owner: ownerID} }

// Observe регистрирует встречу признака: +1 к счётчику и +сумма к обороту. Тип
// (kind) проставляется только при создании записи (первая встреча). Оборот
// суммируется только по совпадающей валюте (чтобы не смешивать BYN и USD).
func (r *MerchantRepo) Observe(name, kind string, amount money.Money, on core.Date) error {
	d := on.String()
	_, err := r.idx.db.Exec(`INSERT INTO merchants(owner_id,name,kind,seen_count,total_minor,currency,first_seen,last_seen)
			VALUES(?,?,?,1,?,?,?,?)
			ON CONFLICT(owner_id,name) DO UPDATE SET
				kind = excluded.kind,
				seen_count = seen_count + 1,
				total_minor = total_minor + CASE WHEN merchants.currency=excluded.currency THEN excluded.total_minor ELSE 0 END,
				last_seen = excluded.last_seen`,
		r.owner, name, kind, amount.Minor(), amount.Currency().String(), d, d)
	return err
}

func (r *MerchantRepo) List() ([]*smssvc.MerchantEntry, error) {
	rows, err := r.idx.db.Query(`SELECT name,kind,label,seen_count,total_minor,currency,last_seen,category_id
			FROM merchants WHERE owner_id=? ORDER BY seen_count DESC, last_seen DESC`, r.owner)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*smssvc.MerchantEntry
	for rows.Next() {
		e, err := scanMerchant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Assign закрепляет категорию и ярлык за признаком (точное совпадение по имени).
// Создаёт запись, если её ещё нет (ручное добавление). categoryID/label="" — сброс.
func (r *MerchantRepo) Assign(name, categoryID, label string) error {
	_, err := r.idx.db.Exec(`INSERT INTO merchants(owner_id,name,kind,seen_count,total_minor,currency,first_seen,last_seen,category_id,label)
			VALUES(?,?,?,0,0,?,'','',?,?)
			ON CONFLICT(owner_id,name) DO UPDATE SET category_id=excluded.category_id, label=excluded.label`,
		r.owner, name, smssvc.KindMerchant, money.BaseCurrency.String(), categoryID, label)
	return err
}

// Entry возвращает запись признака или nil, если её нет.
func (r *MerchantRepo) Entry(name string) (*smssvc.MerchantEntry, error) {
	row := r.idx.db.QueryRow(`SELECT name,kind,label,seen_count,total_minor,currency,last_seen,category_id
			FROM merchants WHERE owner_id=? AND name=?`, r.owner, name)
	e, err := scanMerchant(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return e, err
}

func scanMerchant(sc scanner) (*smssvc.MerchantEntry, error) {
	var (
		name, kind, label, cur, lastSeen, catID string
		count                                   int
		minor                                   int64
	)
	if err := sc.Scan(&name, &kind, &label, &count, &minor, &cur, &lastSeen, &catID); err != nil {
		return nil, err
	}
	total, err := money.New(minor, money.Currency(cur))
	if err != nil {
		total, _ = money.New(0, money.BaseCurrency)
	}
	return &smssvc.MerchantEntry{
		Name: name, Kind: kind, Label: label, SeenCount: count,
		Total: total, LastSeen: lastSeen, CategoryID: catID,
	}, nil
}
