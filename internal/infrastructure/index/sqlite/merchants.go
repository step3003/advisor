package sqlite

import (
	smssvc "advisor/internal/application/sms"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// MerchantRepo реализует sms.MerchantRepo — авто-накапливаемый справочник
// продавцов для одного владельца.
type MerchantRepo struct {
	idx   *Index
	owner string
}

// Merchants возвращает репозиторий справочника продавцов владельца ownerID.
func (i *Index) Merchants(ownerID string) *MerchantRepo { return &MerchantRepo{idx: i, owner: ownerID} }

// Observe регистрирует встречу продавца: +1 к счётчику и +сумма к обороту.
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
	rows, err := r.idx.db.Query(`SELECT name,seen_count,total_minor,currency,last_seen
			FROM merchants WHERE owner_id=? ORDER BY seen_count DESC, last_seen DESC`, r.owner)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*smssvc.MerchantEntry
	for rows.Next() {
		var (
			name, cur, lastSeen string
			count               int
			minor               int64
		)
		if err := rows.Scan(&name, &count, &minor, &cur, &lastSeen); err != nil {
			return nil, err
		}
		total, err := money.New(minor, money.Currency(cur))
		if err != nil {
			total, _ = money.New(0, money.BaseCurrency)
		}
		out = append(out, &smssvc.MerchantEntry{
			Name: name, SeenCount: count, Total: total, LastSeen: lastSeen,
		})
	}
	return out, rows.Err()
}
