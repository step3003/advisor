package sqlite

import (
	"fmt"

	"advisor/internal/application/ports"
)

// RebuildStats — итог пересборки индекса из vault.
type RebuildStats struct {
	Categories   int
	Transactions int
	Plans        int
	Recurring    int
	Errors       []string
}

// RebuildFromVault полностью пересобирает производные таблицы индекса из файлов
// vault (критерий приёмки 6: удалил индекс → приложение восстановило его без
// потерь). Кэш курсов и настройки не затрагиваются — это локальные данные.
//
// Порядок важен из-за внешних ключей: категории загружаются первыми.
func (i *Index) RebuildFromVault() (RebuildStats, error) {
	refs, err := i.vault.List()
	if err != nil {
		return RebuildStats{}, fmt.Errorf("rebuild: листинг vault: %w", err)
	}

	// Группируем по коллекциям для корректного порядка вставки.
	byCollection := map[string][]ports.RecordRef{}
	for _, ref := range refs {
		byCollection[ref.Collection] = append(byCollection[ref.Collection], ref)
	}

	tx, err := i.db.Begin()
	if err != nil {
		return RebuildStats{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// Откладываем проверку внешних ключей до COMMIT: файлы vault перечисляются в
	// произвольном порядке ФС, поэтому подкатегория может встретиться раньше своего
	// родителя. На момент фиксации транзакции все записи уже вставлены.
	if _, err := tx.Exec(`PRAGMA defer_foreign_keys = ON`); err != nil {
		return RebuildStats{}, fmt.Errorf("rebuild: defer_foreign_keys: %w", err)
	}

	// Очищаем производные таблицы.
	for _, table := range []string{"transactions", "plan_items", "recurring_templates", "categories", "vault_state"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return RebuildStats{}, fmt.Errorf("rebuild: очистка %s: %w", table, err)
		}
	}

	var stats RebuildStats

	// 1) Категории (на них ссылаются планы/транзакции).
	for _, ref := range byCollection[ports.CollectionCategories] {
		rec, err := i.vault.Get(ref)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		c, err := decodeCategory(rec.Data)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertCategoryRow(tx, "", c); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertVaultState(tx, rec.RecordRef, c.Meta.Rev); err != nil {
			return stats, err
		}
		stats.Categories++
	}

	// 2) Планы.
	for _, ref := range byCollection[ports.CollectionPlans] {
		rec, err := i.vault.Get(ref)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		p, err := decodePlan(rec.Data)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertPlanRow(tx, "", p); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertVaultState(tx, rec.RecordRef, p.Meta.Rev); err != nil {
			return stats, err
		}
		stats.Plans++
	}

	// 3) Транзакции.
	for _, ref := range byCollection[ports.CollectionTransactions] {
		rec, err := i.vault.Get(ref)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		t, err := decodeTransaction(rec.Data)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertTransactionRow(tx, "", t); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertVaultState(tx, rec.RecordRef, t.Meta.Rev); err != nil {
			return stats, err
		}
		stats.Transactions++
	}

	// 4) Шаблоны повторяющихся операций.
	for _, ref := range byCollection[ports.CollectionRecurring] {
		rec, err := i.vault.Get(ref)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		tpl, err := decodeRecurring(rec.Data)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertRecurringRow(tx, "", tpl); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", vaultPath(ref), err))
			continue
		}
		if err := upsertVaultState(tx, rec.RecordRef, tpl.Meta.Rev); err != nil {
			return stats, err
		}
		stats.Recurring++
	}

	if err := tx.Commit(); err != nil {
		return stats, err
	}
	return stats, nil
}
