package recurring

import (
	"testing"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

func TestTemplateEdit(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	tpl, err := New("rec-1", core.Expense, "cat-rent", amt, 5, mustDate(2026, 1, 1), nil, false, now)
	if err != nil {
		t.Fatal(err)
	}
	revBefore := tpl.Meta.Rev

	later := now.Add(time.Hour)
	newAmt := money.MustNew(95000, "BYN")
	if err := tpl.Edit(core.Expense, "cat-rent", newAmt, 10, mustDate(2026, 2, 1), nil, true, later); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if tpl.DayOfMonth != 10 {
		t.Errorf("день не обновился: %d", tpl.DayOfMonth)
	}
	if !tpl.Amount.Equal(newAmt) {
		t.Errorf("сумма не обновилась: %s", tpl.Amount.Format())
	}
	if !tpl.AutoCreateFact {
		t.Error("флаг автосоздания факта не обновился")
	}
	if tpl.Meta.Rev != revBefore+1 {
		t.Errorf("ревизия должна вырасти: было %d, стало %d", revBefore, tpl.Meta.Rev)
	}
}

func TestTemplateEditValidation(t *testing.T) {
	amt := money.MustNew(80000, "BYN")
	tpl, _ := New("rec-1", core.Expense, "cat-rent", amt, 5, mustDate(2026, 1, 1), nil, false, now)

	if err := tpl.Edit(core.Expense, "cat", amt, 0, mustDate(2026, 1, 1), nil, false, now); err == nil {
		t.Error("день 0 должен давать ошибку")
	}
	if err := tpl.Edit(core.Expense, "", amt, 5, mustDate(2026, 1, 1), nil, false, now); err == nil {
		t.Error("пустая категория должна давать ошибку")
	}
	end := mustDate(2025, 12, 1)
	if err := tpl.Edit(core.Expense, "cat", amt, 5, mustDate(2026, 1, 1), &end, false, now); err == nil {
		t.Error("окончание раньше начала должно давать ошибку")
	}
}
