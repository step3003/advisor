package recurring

import (
	"testing"
	"time"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	domrec "advisor/internal/domain/recurring"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/memory"
)

func setup(t *testing.T) (*Service, *memory.Store, string) {
	t.Helper()
	store := memory.NewStore()
	clk := clock.Fixed{T: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)}
	ids := memory.NewSeqIDs()
	cat, _ := category.New(ids.NewID(), "Аренда", core.Expense, clk.T)
	_ = store.Categories().Save(cat)
	svc := New(store.Recurring(), store.Plans(), store.Transactions(), clk, ids)
	return svc, store, cat.Meta.ID
}

func mustDate(y, m, d int) core.Date {
	dt, _ := core.NewDate(y, m, d)
	return dt
}

func TestGeneratePlanIdempotent(t *testing.T) {
	svc, store, catID := setup(t)
	tpl, _ := domrec.New("rec-000000000000000000000000000001", core.Expense, catID,
		money.MustNew(80000, "BYN"), 5, mustDate(2026, 1, 1), nil, false, time.Now())
	_ = svc.CreateTemplate(tpl)

	ym := core.YearMonth{Year: 2026, Month: 7}
	stats, err := svc.GenerateForMonth(ym)
	if err != nil {
		t.Fatal(err)
	}
	if stats.PlansCreated != 1 {
		t.Errorf("first gen plans = %d, want 1", stats.PlansCreated)
	}
	// Повторная генерация не дублирует (FR-REC-2).
	stats2, _ := svc.GenerateForMonth(ym)
	if stats2.PlansCreated != 0 {
		t.Errorf("second gen plans = %d, want 0", stats2.PlansCreated)
	}
	plans, _ := store.Plans().ListByMonth(ym)
	if len(plans) != 1 {
		t.Errorf("plans in month = %d, want 1", len(plans))
	}
}

func TestGenerateFactIdempotent(t *testing.T) {
	svc, store, catID := setup(t)
	tpl, _ := domrec.New("rec-000000000000000000000000000002", core.Expense, catID,
		money.MustNew(80000, "BYN"), 5, mustDate(2026, 1, 1), nil, true, time.Now()) // autoCreateFact=true
	_ = svc.CreateTemplate(tpl)

	ym := core.YearMonth{Year: 2026, Month: 7}
	stats, err := svc.GenerateForMonth(ym)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FactsCreated != 1 {
		t.Errorf("first gen facts = %d, want 1", stats.FactsCreated)
	}
	stats2, _ := svc.GenerateForMonth(ym)
	if stats2.FactsCreated != 0 {
		t.Errorf("second gen facts = %d, want 0", stats2.FactsCreated)
	}
	txs, _ := store.Transactions().ListByMonth(ym)
	if len(txs) != 1 {
		t.Errorf("facts in month = %d, want 1", len(txs))
	}
	if txs[0].RecurringID != tpl.Meta.ID {
		t.Error("fact must reference template id")
	}
	if txs[0].OccurredOn.String() != "2026-07-05" {
		t.Errorf("fact date = %s, want 2026-07-05", txs[0].OccurredOn)
	}
}

func TestResumeAndNonApplicable(t *testing.T) {
	svc, _, catID := setup(t)
	end := mustDate(2026, 3, 31)
	tpl, _ := domrec.New("rec-000000000000000000000000000009", core.Expense, catID,
		money.MustNew(80000, "BYN"), 5, mustDate(2026, 1, 1), &end, false, time.Now())
	_ = svc.CreateTemplate(tpl)

	// Месяц вне интервала шаблона — ничего не создаётся.
	stats, err := svc.GenerateForMonth(core.YearMonth{Year: 2026, Month: 7})
	if err != nil {
		t.Fatal(err)
	}
	if stats.PlansCreated != 0 {
		t.Errorf("out-of-range template must not generate, got %d", stats.PlansCreated)
	}

	// Pause/Resume переключают активность.
	if err := svc.Pause(tpl.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Resume(tpl.Meta.ID); err != nil {
		t.Fatal(err)
	}
	// В марте (в интервале) после resume план создаётся.
	stats2, _ := svc.GenerateForMonth(core.YearMonth{Year: 2026, Month: 3})
	if stats2.PlansCreated != 1 {
		t.Errorf("resumed in-range template plans = %d, want 1", stats2.PlansCreated)
	}
}

func TestPausedTemplateNotGenerated(t *testing.T) {
	svc, store, catID := setup(t)
	tpl, _ := domrec.New("rec-000000000000000000000000000003", core.Expense, catID,
		money.MustNew(80000, "BYN"), 5, mustDate(2026, 1, 1), nil, false, time.Now())
	_ = svc.CreateTemplate(tpl)
	if err := svc.Pause(tpl.Meta.ID); err != nil {
		t.Fatal(err)
	}
	stats, _ := svc.GenerateForMonth(core.YearMonth{Year: 2026, Month: 7})
	if stats.PlansCreated != 0 {
		t.Errorf("paused template must not generate, got %d", stats.PlansCreated)
	}
	_ = store
}
