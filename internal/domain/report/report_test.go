package report

import (
	"testing"

	"advisor/internal/domain/money"
)

func byn(minor int64) money.Money { return money.MustNew(minor, money.BaseCurrency) }

func TestCategoryPlanVsFact(t *testing.T) {
	line := CategoryPlanVsFact{
		CategoryID: "cat-1",
		Plan:       byn(50000), // 500.00
		Fact:       byn(60000), // 600.00
	}
	if line.Deviation().Minor() != 10000 {
		t.Errorf("deviation = %d, want 10000", line.Deviation().Minor())
	}
	if line.Remaining().Minor() != -10000 {
		t.Errorf("remaining = %d, want -10000", line.Remaining().Minor())
	}
	if line.PercentExecuted() != 120 {
		t.Errorf("percent = %d, want 120", line.PercentExecuted())
	}
	if !line.IsOverspent() {
		t.Error("must be overspent")
	}
}

func TestPercentEdgeCases(t *testing.T) {
	// План ноль, факт ноль => 0%.
	l1 := CategoryPlanVsFact{Plan: byn(0), Fact: byn(0)}
	if l1.PercentExecuted() != 0 {
		t.Errorf("0/0 = %d, want 0", l1.PercentExecuted())
	}
	// План ноль, факт ненулевой => 100% (перерасход при отсутствии плана).
	l2 := CategoryPlanVsFact{Plan: byn(0), Fact: byn(5000)}
	if l2.PercentExecuted() != 100 {
		t.Errorf("x/0 = %d, want 100", l2.PercentExecuted())
	}
}

func TestPlanVsFactBalance(t *testing.T) {
	p := PlanVsFact{
		Year: 2026, Month: 7,
		PlanExpense: byn(100000),
		FactIncome:  byn(300000),
		FactExpense: byn(120000),
	}
	if p.Balance().Minor() != 180000 {
		t.Errorf("balance = %d, want 180000", p.Balance().Minor())
	}
	if p.RemainingExpense().Minor() != -20000 {
		t.Errorf("remaining = %d, want -20000", p.RemainingExpense().Minor())
	}
}

func TestPeriodSummaryBalance(t *testing.T) {
	s := PeriodSummary{
		TotalIncome:  byn(500000),
		TotalExpense: byn(320000),
	}
	if s.Balance().Minor() != 180000 {
		t.Errorf("balance = %d, want 180000", s.Balance().Minor())
	}
}
