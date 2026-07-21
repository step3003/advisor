package reporting

import (
	"testing"
	"time"

	"advisor/internal/application/currency"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/transaction"
	"advisor/internal/infrastructure/memory"
)

func TestPeriodSummary(t *testing.T) {
	store := memory.NewStore()
	cur := currency.New(store.Rates(), nil)
	svc := New(store.Transactions(), cur)
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

	add := func(id string, typ core.EntryType, day int, catID string, minor int64) {
		d, _ := core.NewDate(2026, 7, day)
		tx, _ := transaction.New(id, typ, d, catID, money.MustNew(minor, "BYN"), "", now)
		_ = store.Transactions().Save(tx)
	}
	add("t-0000000000000000000000000001", core.Expense, 5, "food", 30000)    // 300
	add("t-0000000000000000000000000002", core.Expense, 6, "food", 20000)    // 200
	add("t-0000000000000000000000000003", core.Expense, 7, "taxi", 10000)    // 100
	add("t-0000000000000000000000000004", core.Income, 10, "salary", 500000) // 5000

	from, _ := core.NewDate(2026, 7, 1)
	to, _ := core.NewDate(2026, 7, 31)
	sum, err := svc.PeriodSummary(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if sum.TotalExpense.Minor() != 60000 {
		t.Errorf("expense = %d, want 60000", sum.TotalExpense.Minor())
	}
	if sum.TotalIncome.Minor() != 500000 {
		t.Errorf("income = %d, want 500000", sum.TotalIncome.Minor())
	}
	if sum.Balance().Minor() != 440000 {
		t.Errorf("balance = %d, want 440000", sum.Balance().Minor())
	}
	// Топ категорий по расходу: food (500) > taxi (100).
	if len(sum.ExpenseByCategory) != 2 {
		t.Fatalf("expense categories = %d, want 2", len(sum.ExpenseByCategory))
	}
	if sum.ExpenseByCategory[0].CategoryID != "food" || sum.ExpenseByCategory[0].Amount.Minor() != 50000 {
		t.Errorf("top category = %+v, want food/50000", sum.ExpenseByCategory[0])
	}
}

func TestMonthlyDynamics(t *testing.T) {
	store := memory.NewStore()
	cur := currency.New(store.Rates(), nil)
	svc := New(store.Transactions(), cur)
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

	d1, _ := core.NewDate(2026, 6, 5)
	d2, _ := core.NewDate(2026, 7, 5)
	tx1, _ := transaction.New("t-0000000000000000000000000001", core.Expense, d1, "food", money.MustNew(10000, "BYN"), "", now)
	tx2, _ := transaction.New("t-0000000000000000000000000002", core.Expense, d2, "food", money.MustNew(20000, "BYN"), "", now)
	_ = store.Transactions().Save(tx1)
	_ = store.Transactions().Save(tx2)

	points, err := svc.MonthlyDynamics(core.YearMonth{Year: 2026, Month: 6}, core.YearMonth{Year: 2026, Month: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("points = %d, want 2", len(points))
	}
	if points[0].Expense.Minor() != 10000 || points[1].Expense.Minor() != 20000 {
		t.Errorf("dynamics wrong: %d, %d", points[0].Expense.Minor(), points[1].Expense.Minor())
	}
}
