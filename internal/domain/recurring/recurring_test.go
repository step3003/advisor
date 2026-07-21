package recurring

import (
	"testing"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

var now = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func mustDate(y, m, d int) core.Date {
	dt, err := core.NewDate(y, m, d)
	if err != nil {
		panic(err)
	}
	return dt
}

func TestNewTemplate(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	tpl, err := New("rec-1", core.Expense, "cat-rent", amt, 5, mustDate(2026, 1, 1), nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.Active {
		t.Error("must be active")
	}
	if tpl.DayOfMonth != 5 {
		t.Errorf("day = %d", tpl.DayOfMonth)
	}
}

func TestTemplateValidation(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	if _, err := New("id", core.Expense, "cat", amt, 0, mustDate(2026, 1, 1), nil, false, now); err == nil {
		t.Error("day 0 must error")
	}
	if _, err := New("id", core.Expense, "cat", amt, 29, mustDate(2026, 1, 1), nil, false, now); err == nil {
		t.Error("day 29 must error")
	}
	end := mustDate(2025, 12, 1)
	if _, err := New("id", core.Expense, "cat", amt, 5, mustDate(2026, 1, 1), &end, false, now); err == nil {
		t.Error("end before start must error")
	}
}

func TestAppliesTo(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	end := mustDate(2026, 6, 30)
	tpl, _ := New("rec-1", core.Expense, "cat", amt, 5, mustDate(2026, 2, 1), &end, false, now)

	if tpl.AppliesTo(core.YearMonth{Year: 2026, Month: 1}) {
		t.Error("before start must not apply")
	}
	if !tpl.AppliesTo(core.YearMonth{Year: 2026, Month: 2}) {
		t.Error("start month must apply")
	}
	if !tpl.AppliesTo(core.YearMonth{Year: 2026, Month: 6}) {
		t.Error("end month must apply")
	}
	if tpl.AppliesTo(core.YearMonth{Year: 2026, Month: 7}) {
		t.Error("after end must not apply")
	}
}

func TestPauseResume(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	tpl, _ := New("rec-1", core.Expense, "cat", amt, 5, mustDate(2026, 1, 1), nil, false, now)
	tpl.Pause(now.Add(time.Hour))
	if tpl.Active {
		t.Error("must be paused")
	}
	if tpl.AppliesTo(core.YearMonth{Year: 2026, Month: 7}) {
		t.Error("paused must not apply")
	}
	tpl.Resume(now.Add(2 * time.Hour))
	if !tpl.Active {
		t.Error("must be active")
	}
	if tpl.Meta.Rev != 3 {
		t.Errorf("rev = %d, want 3", tpl.Meta.Rev)
	}
}

func TestOccurrenceDate(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	tpl, _ := New("rec-1", core.Expense, "cat", amt, 5, mustDate(2026, 1, 1), nil, false, now)
	d := tpl.OccurrenceDate(core.YearMonth{Year: 2026, Month: 7})
	if d.String() != "2026-07-05" {
		t.Errorf("occurrence = %s", d)
	}
}
