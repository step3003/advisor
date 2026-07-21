// Package ports — интерфейсы (порты) прикладного слоя.
//
// Прикладные сервисы (application) зависят ТОЛЬКО от этих интерфейсов и от
// домена. Конкретные реализации (SQLite-индекс, файловый vault, HTTP-клиент
// НБ РБ, системные часы) живут в infrastructure и внедряются в cmd/advisor.
package ports

import "time"

// Clock — источник текущего времени (для тестируемости, NFR-3).
type Clock interface {
	Now() time.Time
}

// IDGenerator — генератор стабильных идентификаторов записей (UUID).
type IDGenerator interface {
	NewID() string
}
