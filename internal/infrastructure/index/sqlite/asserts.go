package sqlite

import "advisor/internal/application/ports"

// Проверки соответствия репозиториев портам на этапе компиляции.
var (
	_ ports.CategoryRepository    = (*CategoryRepo)(nil)
	_ ports.TransactionRepository = (*TransactionRepo)(nil)
	_ ports.PlanRepository        = (*PlanRepo)(nil)
	_ ports.RecurringRepository   = (*RecurringRepo)(nil)
)
