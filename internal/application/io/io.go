// Package io — usecase экспорта/импорта данных (FR-IO).
//
// Экспорт снапшота всех данных в один версионированный JSON (для бэкапа/переноса
// вне iCloud) и импорт с режимами «заменить всё»/«объединить» по стабильным UUID.
// Плюс экспорт транзакций за период в CSV. Round-trip не теряет данные.
package io

import (
	"encoding/csv"
	stdio "io"
	"time"

	"encoding/json"
	"fmt"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// SnapshotFormatVersion — версия формата снапшота (FR-IO-1).
const SnapshotFormatVersion = 1

// ImportMode — режим импорта (FR-IO-2).
type ImportMode int

const (
	// ModeMerge объединяет по UUID (upsert), не удаляя отсутствующее.
	ModeMerge ImportMode = iota
	// ModeReplace заменяет все данные снапшотом.
	ModeReplace
)

// Service — сервис экспорта/импорта.
type Service struct {
	cats ports.CategoryRepository
	txs  ports.TransactionRepository
	plan ports.PlanRepository
	rec  ports.RecurringRepository
}

// New собирает сервис.
func New(cats ports.CategoryRepository, txs ports.TransactionRepository, plan ports.PlanRepository, rec ports.RecurringRepository) *Service {
	return &Service{cats: cats, txs: txs, plan: plan, rec: rec}
}

// Export пишет снапшот всех данных в JSON (FR-IO-1).
func (s *Service) Export(w stdio.Writer) error {
	snap := snapshot{
		FormatVersion: SnapshotFormatVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		BaseCurrency:  money.BaseCurrency.String(),
	}

	cats, err := s.cats.List(true)
	if err != nil {
		return err
	}
	for _, c := range cats {
		snap.Categories = append(snap.Categories, toCategoryJSON(c))
	}
	txs, err := s.txs.ListAll()
	if err != nil {
		return err
	}
	for _, t := range txs {
		snap.Transactions = append(snap.Transactions, toTransactionJSON(t))
	}
	plans, err := s.plan.ListAll()
	if err != nil {
		return err
	}
	for _, p := range plans {
		snap.Plans = append(snap.Plans, toPlanJSON(p))
	}
	recs, err := s.rec.List(false)
	if err != nil {
		return err
	}
	for _, r := range recs {
		snap.Recurring = append(snap.Recurring, toRecurringJSON(r))
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(snap)
}

// Import загружает снапшот из JSON (FR-IO-2).
func (s *Service) Import(r stdio.Reader, mode ImportMode) error {
	var snap snapshot
	if err := json.NewDecoder(r).Decode(&snap); err != nil {
		return fmt.Errorf("io: разбор снапшота: %w", err)
	}
	if snap.FormatVersion != SnapshotFormatVersion {
		return fmt.Errorf("io: неподдерживаемая версия формата: %d", snap.FormatVersion)
	}

	if mode == ModeReplace {
		if err := s.clearAll(); err != nil {
			return err
		}
	}

	// Порядок: категории раньше зависимых сущностей (внешние ключи).
	for _, cj := range snap.Categories {
		c, err := cj.toDomain()
		if err != nil {
			return err
		}
		if err := s.cats.Save(c); err != nil {
			return err
		}
	}
	for _, pj := range snap.Plans {
		p, err := pj.toDomain()
		if err != nil {
			return err
		}
		if err := s.plan.Save(p); err != nil {
			return err
		}
	}
	for _, tj := range snap.Transactions {
		t, err := tj.toDomain()
		if err != nil {
			return err
		}
		if err := s.txs.Save(t); err != nil {
			return err
		}
	}
	for _, rj := range snap.Recurring {
		r, err := rj.toDomain()
		if err != nil {
			return err
		}
		if err := s.rec.Save(r); err != nil {
			return err
		}
	}
	return nil
}

// clearAll удаляет все данные (для режима «заменить всё»).
func (s *Service) clearAll() error {
	txs, err := s.txs.ListAll()
	if err != nil {
		return err
	}
	for _, t := range txs {
		if err := s.txs.Delete(t.Meta.ID); err != nil {
			return err
		}
	}
	plans, err := s.plan.ListAll()
	if err != nil {
		return err
	}
	for _, p := range plans {
		if err := s.plan.Delete(p.Meta.ID); err != nil {
			return err
		}
	}
	recs, err := s.rec.List(false)
	if err != nil {
		return err
	}
	for _, r := range recs {
		if err := s.rec.Delete(r.Meta.ID); err != nil {
			return err
		}
	}
	cats, err := s.cats.List(true)
	if err != nil {
		return err
	}
	for _, c := range cats {
		if err := s.cats.Delete(c.Meta.ID); err != nil {
			return err
		}
	}
	return nil
}

// ExportTransactionsCSV пишет транзакции периода в CSV (FR-IO-3).
func (s *Service) ExportTransactionsCSV(w stdio.Writer, from, to core.Date) error {
	txs, err := s.txs.ListByPeriod(from, to)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"occurred_on", "type", "category_id", "amount", "currency", "note"}); err != nil {
		return err
	}
	for _, t := range txs {
		row := []string{
			t.OccurredOn.String(),
			string(t.Type),
			t.CategoryID,
			t.Amount.Decimal(),
			t.Amount.Currency().String(),
			t.Note,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
