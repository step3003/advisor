// Package i18n — все строки интерфейса (только русский, NFR-8).
//
// Строки UI и локале-зависимое форматирование (деньги, даты, проценты, названия
// месяцев) собраны в одном пакете, чтобы добавление новых языков не требовало
// правки экранов. Экраны и виджеты не содержат хардкод-строк.
package i18n

// Название приложения и разделы навигации.
const (
	AppTitle = "Advisor — финансовый планировщик"

	NavBalance    = "План/Факт"
	NavAddFact    = "Ввод факта"
	NavReports    = "Отчёты"
	NavCategories = "Категории"
	NavRecurring  = "Повторяющиеся"
	NavSettings   = "Настройки"
)

// Общие кнопки и подписи.
const (
	BtnAdd      = "Добавить"
	BtnSave     = "Сохранить"
	BtnEdit     = "Изменить"
	BtnDelete   = "Удалить"
	BtnCancel   = "Отмена"
	BtnClose    = "Закрыть"
	BtnPrev     = "‹"
	BtnNext     = "›"
	BtnRefresh  = "Обновить"
	BtnArchive  = "Архивировать"
	BtnRestore  = "Вернуть из архива"
	BtnRename   = "Переименовать"
	BtnPause    = "Приостановить"
	BtnResume   = "Возобновить"
	BtnApply    = "Применить"
	BtnBrowse   = "Выбрать файл…"
	BtnYes      = "Да"
	BtnNo       = "Нет"
	LabelTotal  = "Итого"
	LabelType   = "Тип"
	LabelDate   = "Дата"
	LabelAmount = "Сумма"
	LabelCurr   = "Валюта"
	LabelNote   = "Комментарий"
	LabelCat    = "Категория"
	LabelSubcat = "Подкатегория"
	Dash        = "—"
	NoneOption  = "(нет)"
)

// Типы операций.
const (
	TypeExpenseLabel = "Расход"
	TypeIncomeLabel  = "Доход"
)

// Экран План/Факт (FR-BAL).
const (
	BalanceTitle       = "План и факт по категориям"
	BalanceColCategory = "Категория"
	BalanceColPlan     = "План"
	BalanceColFact     = "Факт"
	BalanceColDev      = "Отклонение"
	BalanceColPercent  = "% исполнения"
	BalanceColRemain   = "Осталось"
	BalancePlanIncome  = "Плановый доход"
	BalancePlanExpense = "Плановый расход"
	BalanceFactIncome  = "Фактический доход"
	BalanceFactExpense = "Фактический расход"
	BalanceRemainder   = "Остаток (доход − расход)"
	BalanceRemainMonth = "Осталось до конца месяца"
	BalanceOverspent   = "перерасход"
	BalanceCopyPrev    = "Копировать план из прошлого месяца"
	BalanceEditPlan    = "План категории"
	BalanceSetPlan     = "Задать план"
	BalanceApproxNote  = "* использован приблизительный курс (нет курса на дату)"
	BalanceEmpty       = "Нет данных за месяц. Задайте план или внесите факт."
)

// Экран ввода факта (FR-TX).
const (
	TxTitle        = "Ввод фактической операции"
	TxRecent       = "Последние операции"
	TxAdded        = "Операция добавлена"
	TxUpdated      = "Операция обновлена"
	TxDeleted      = "Операция удалена"
	TxConfirmDel   = "Удалить операцию?"
	TxEditTitle    = "Изменить операцию"
	TxColDate      = "Дата"
	TxColType      = "Тип"
	TxColCategory  = "Категория"
	TxColAmount    = "Сумма"
	TxColNote      = "Комментарий"
	TxEmpty        = "В этом месяце операций пока нет."
	TxNeedCategory = "Выберите категорию"
)

// Экран отчётов (FR-REP).
const (
	ReportsTitle     = "Отчёты за период"
	ReportPreset     = "Период"
	ReportPresetDay  = "День"
	ReportPresetWeek = "Неделя"
	ReportPresetMon  = "Месяц"
	ReportPresetQ    = "Квартал"
	ReportPresetYear = "Год"
	ReportPresetCust = "Произвольный"
	ReportFrom       = "С"
	ReportTo         = "По"
	ReportBuild      = "Построить"
	ReportByCategory = "Расходы по категориям (BYN)"
	ReportDynamics   = "Динамика доход/расход по месяцам (BYN)"
	ReportTopCat     = "Топ категорий по расходу"
	ReportIncome     = "Доход"
	ReportExpense    = "Расход"
	ReportTotalInc   = "Доход за период"
	ReportTotalExp   = "Расход за период"
	ReportBalance    = "Баланс за период"
	ReportByCurrency = "Обороты расходов по исходным валютам"
	ReportEmpty      = "За выбранный период данных нет."
	ReportBadDates   = "Некорректный период: дата «С» позже даты «По»."
)

// Экран категорий (FR-CAT).
const (
	CategoriesTitle    = "Категории и подкатегории"
	CatAddTop          = "Новая категория"
	CatAddSub          = "Новая подкатегория"
	CatName            = "Название"
	CatBuiltin         = "встроенная"
	CatArchived        = "в архиве"
	CatConfirmDelete   = "Удалить категорию без возможности восстановления?"
	CatDeleteBlocked   = "Нельзя удалить: по категории есть операции или планы. Используйте архивацию."
	CatRenameTitle     = "Переименовать категорию"
	CatCreated         = "Категория создана"
	CatSelectParent    = "Выберите родительскую категорию"
	CatShowArchived    = "Показывать архивные"
	CatExpensesSection = "Расходы"
	CatIncomesSection  = "Доходы"
)

// Экран повторяющихся операций (FR-REC).
const (
	RecTitle         = "Повторяющиеся операции"
	RecAdd           = "Новый шаблон"
	RecEditTitle     = "Шаблон повторяющейся операции"
	RecColType       = "Тип"
	RecColCategory   = "Категория"
	RecColAmount     = "Сумма"
	RecColDay        = "День месяца"
	RecColPeriod     = "Действует"
	RecColStatus     = "Статус"
	RecDay           = "День месяца (1–28)"
	RecStart         = "Дата начала"
	RecEnd           = "Дата окончания (опц.)"
	RecAutoFact      = "Автоматически создавать факт"
	RecActive        = "активен"
	RecPaused        = "приостановлен"
	RecEmpty         = "Шаблонов пока нет."
	RecConfirmDelete = "Удалить шаблон?"
	RecOpenEnded     = "бессрочно"
)

// Экран настроек (FR-SET).
const (
	SettingsTitle       = "Настройки"
	SetDefaultCurrency  = "Валюта по умолчанию для ввода"
	SetRates            = "Курсы валют"
	SetUpdateRates      = "Обновить курсы валют"
	SetRatesUpdated     = "Курсы обновлены"
	SetRatesOffline     = "Не удалось обновить курсы (нет сети). Приложение работает на кэше."
	SetExportImport     = "Экспорт и импорт данных"
	SetExportJSON       = "Экспорт снапшота (JSON)"
	SetImportJSON       = "Импорт снапшота (JSON)"
	SetExportCSV        = "Экспорт операций за период (CSV)"
	SetImportMode       = "Режим импорта"
	SetImportMerge      = "Объединить"
	SetImportReplace    = "Заменить всё"
	SetExportDone       = "Экспорт завершён"
	SetImportDone       = "Импорт завершён"
	SetSync             = "Синхронизация (vault / iCloud)"
	SetVaultPath        = "Папка-хранилище"
	SetSyncLastRebuild  = "Последняя пересборка индекса"
	SetSyncCounts       = "Записей в индексе"
	SetSyncConflicts    = "Конфликтов обработано"
	SetSyncErrors       = "Ошибок пересборки"
	SetRescan           = "Пересканировать хранилище"
	SetRescanDone       = "Хранилище пересканировано"
	SetCurrencySaved    = "Валюта по умолчанию сохранена"
	SetPeriodForCSV     = "Период выгрузки"
)

// Общие сообщения об ошибках/подтверждениях.
const (
	MsgError       = "Ошибка"
	MsgDone        = "Готово"
	MsgInvalidNum  = "Введите корректную сумму (например 42.50)"
	MsgEmptyName   = "Название не может быть пустым"
	MsgConfirm     = "Подтверждение"
	MsgYearMonth   = "Месяц"
	StatusBarReady = "Готово"
)
