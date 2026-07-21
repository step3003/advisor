package screens

import (
	"time"

	"fyne.io/fyne/v2"

	catalogsvc "advisor/internal/application/catalog"
	currencysvc "advisor/internal/application/currency"
	iosvc "advisor/internal/application/io"
	ledgersvc "advisor/internal/application/ledger"
	planningsvc "advisor/internal/application/planning"
	recurringsvc "advisor/internal/application/recurring"
	reportingsvc "advisor/internal/application/reporting"
	settingssvc "advisor/internal/application/settings"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
)

// SyncInfo — сведения о состоянии синхронизации/индекса для экрана настроек
// (FR-SET-5). Заполняется в main из инфраструктуры и передаётся как обычные
// значения — presentation не зависит от infrastructure.
type SyncInfo struct {
	VaultPath    string
	Categories   int
	Transactions int
	Plans        int
	Recurring    int
	Errors       int
	Conflicts    int
	LastRun      string
}

// Deps — зависимости UI: usecase-сервисы и вспомогательные значения/колбэки.
// Только application-сервисы и домен; никакой инфраструктуры.
type Deps struct {
	Catalog   *catalogsvc.Service
	Ledger    *ledgersvc.Service
	Planning  *planningsvc.Service
	Recurring *recurringsvc.Service
	Reporting *reportingsvc.Service
	Currency  *currencysvc.Service
	Settings  *settingssvc.Service
	IO        *iosvc.Service

	// Now — текущее время (из инфраструктурных часов), для значений по умолчанию.
	Now func() time.Time
	// Window — главное окно (для диалогов). Устанавливается слоем app.
	Window fyne.Window
	// Sync — снимок состояния синхронизации на момент старта (FR-SET-5).
	Sync SyncInfo
	// Rescan — пересканирование vault и пересборка индекса (реализуется в main).
	Rescan func() (SyncInfo, error)
}

// Today возвращает сегодняшнюю дату из часов приложения.
func (d *Deps) Today() core.Date {
	return core.DateOf(d.Now())
}

// CurrentMonth возвращает текущий календарный месяц.
func (d *Deps) CurrentMonth() core.YearMonth {
	return core.YearMonthOf(d.Now())
}

// activeCategories возвращает неархивные категории (для выбора в формах).
func (d *Deps) activeCategories() ([]*category.Category, error) {
	return d.Catalog.List(false)
}

// allCategories возвращает все категории, включая архивные (для отчётов/имён).
func (d *Deps) allCategories() ([]*category.Category, error) {
	return d.Catalog.List(true)
}

// Screen — общий контракт экрана: заголовок, иконка вкладки, построение и
// обновление содержимого при активации.
type Screen interface {
	Title() string
	Icon() fyne.Resource
	Build() fyne.CanvasObject
	Refresh()
}
