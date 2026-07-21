package io

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/plan"
	"advisor/internal/domain/recurring"
	"advisor/internal/domain/transaction"
	"advisor/internal/infrastructure/memory"
)

func seedStore(t *testing.T) *memory.Store {
	t.Helper()
	store := memory.NewStore()
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	ids := memory.NewSeqIDs()

	cat, _ := category.New(ids.NewID(), "Еда", core.Expense, now)
	_ = store.Categories().Save(cat)

	d, _ := core.NewDate(2026, 7, 5)
	tx, _ := transaction.New(ids.NewID(), core.Expense, d, cat.Meta.ID, money.MustNew(4250, "USD"), "обед", now)
	_ = store.Transactions().Save(tx)

	pi, _ := plan.New(ids.NewID(), core.YearMonth{Year: 2026, Month: 7}, cat.Meta.ID, money.MustNew(50000, "BYN"), "", now)
	_ = store.Plans().Save(pi)

	start, _ := core.NewDate(2026, 1, 1)
	end, _ := core.NewDate(2026, 12, 31)
	tpl, _ := recurring.New(ids.NewID(), core.Expense, cat.Meta.ID, money.MustNew(80000, "BYN"), 5, start, &end, true, now)
	_ = store.Recurring().Save(tpl)
	return store
}

func newService(store *memory.Store) *Service {
	return New(store.Categories(), store.Transactions(), store.Plans(), store.Recurring())
}

func TestExportImportRoundTrip(t *testing.T) {
	src := seedStore(t)
	var buf bytes.Buffer
	if err := newService(src).Export(&buf); err != nil {
		t.Fatal(err)
	}

	// Импортируем в свежее хранилище (merge).
	dst := memory.NewStore()
	if err := newService(dst).Import(&buf, ModeMerge); err != nil {
		t.Fatal(err)
	}

	cats, _ := dst.Categories().List(true)
	if len(cats) != 1 {
		t.Fatalf("categories = %d, want 1", len(cats))
	}
	txs, _ := dst.Transactions().ListAll()
	if len(txs) != 1 {
		t.Fatalf("transactions = %d, want 1", len(txs))
	}
	// Деньги round-trip без потери точности.
	if txs[0].Amount.Decimal() != "42.50" || txs[0].Amount.Currency() != "USD" {
		t.Errorf("amount round-trip failed: %s %s", txs[0].Amount.Decimal(), txs[0].Amount.Currency())
	}
	if txs[0].Note != "обед" {
		t.Errorf("note lost: %q", txs[0].Note)
	}
	plans, _ := dst.Plans().ListAll()
	if len(plans) != 1 || plans[0].Amount.Minor() != 50000 {
		t.Errorf("plan round-trip failed: %+v", plans)
	}
	recs, _ := dst.Recurring().List(false)
	if len(recs) != 1 || recs[0].EndDate == nil || !recs[0].AutoCreateFact {
		t.Errorf("recurring round-trip failed: %+v", recs)
	}
}

func TestImportReplace(t *testing.T) {
	src := seedStore(t)
	var buf bytes.Buffer
	if err := newService(src).Export(&buf); err != nil {
		t.Fatal(err)
	}

	// Целевое хранилище уже содержит «мусорную» категорию, которую replace удалит.
	dst := memory.NewStore()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	junk, _ := category.New("junk-00000000-0000-4000-8000-000000000000", "Мусор", core.Income, now)
	_ = dst.Categories().Save(junk)

	if err := newService(dst).Import(&buf, ModeReplace); err != nil {
		t.Fatal(err)
	}
	cats, _ := dst.Categories().List(true)
	if len(cats) != 1 {
		t.Fatalf("after replace categories = %d, want 1", len(cats))
	}
	if cats[0].Name == "Мусор" {
		t.Error("replace must remove pre-existing junk data")
	}
}

func TestImportBadVersion(t *testing.T) {
	dst := memory.NewStore()
	bad := `{"format_version":999,"categories":[]}`
	if err := newService(dst).Import(strings.NewReader(bad), ModeMerge); err == nil {
		t.Error("unsupported format version must error")
	}
}

func TestExportCSV(t *testing.T) {
	store := seedStore(t)
	var buf bytes.Buffer
	from, _ := core.NewDate(2026, 7, 1)
	to, _ := core.NewDate(2026, 7, 31)
	if err := newService(store).ExportTransactionsCSV(&buf, from, to); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "occurred_on,type,category_id,amount,currency,note") {
		t.Errorf("CSV header missing: %s", out)
	}
	if !strings.Contains(out, "2026-07-05,expense,") || !strings.Contains(out, "42.50,USD,обед") {
		t.Errorf("CSV row missing: %s", out)
	}
}
