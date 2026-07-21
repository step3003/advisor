package ports

import "time"

// Логические коллекции vault (соответствуют папкам раздела 6.1 ТЗ).
const (
	CollectionCategories   = "categories"
	CollectionTransactions = "transactions"
	CollectionPlans        = "plans"
	CollectionRecurring    = "recurring"
)

// RecordRef — ссылка на запись vault: логическая коллекция, необязательная
// партиция (подпапка "YYYY-MM" для транзакций/планов) и стабильный UUID.
//
// ModTime/Hash заполняются реализацией vault при листинге и используются для
// инкрементальной пересборки индекса (таблица vault_state, FR-SYNC-4).
type RecordRef struct {
	Collection string
	Partition  string // "" для плоских коллекций; "YYYY-MM" для дат
	ID         string

	ModTime time.Time
	Hash    string // хеш содержимого файла
}

// Record — запись vault: ссылка + сырой JSON-байтовый payload.
type Record struct {
	RecordRef
	Data []byte
}

// Conflict — обнаруженная iCloud-конфликтная копия записи (FR-SYNC-5).
type Conflict struct {
	Ref         RecordRef // запись, для которой найден конфликт
	WinnerPath  string    // путь оставленной («победившей») версии
	LoserPath   string    // путь проигравшей версии (перемещена в _conflicts/)
	Description string
}

// Vault — файловое хранилище-источник правды (FR-SYNC-1).
//
// Каждая запись — отдельный JSON-файл <collection>/[partition/]<uuid>.json.
// Запись атомарна (temp-файл + rename). Реализация не знает о доменных типах —
// оперирует сырыми байтами; сериализацию обеспечивает слой репозиториев.
//
// Путь к папке задаётся параметром конструктора (в тестах — t.TempDir()),
// поэтому подключение реального iCloud-контейнера на iOS/macOS не требует
// изменений домена/усечейсов — меняется только путь при сборке платформы.
type Vault interface {
	// Put атомарно записывает запись в vault (создаёт партицию при необходимости).
	Put(rec Record) error
	// Get читает запись по ссылке.
	Get(ref RecordRef) (Record, error)
	// Delete удаляет запись.
	Delete(ref RecordRef) error
	// List перечисляет все записи всех коллекций (для пересборки индекса).
	List() ([]RecordRef, error)

	// ReadMeta/WriteMeta работают со служебными файлами верхнего уровня
	// (advisor.json, settings.json).
	ReadMeta(name string) ([]byte, error)
	WriteMeta(name string, data []byte) error

	// ResolveConflicts находит iCloud-конфликтные копии, оставляет версию с
	// большей (rev, updated_at), проигравшую переносит в _conflicts/ (FR-SYNC-5).
	ResolveConflicts() ([]Conflict, error)

	// Path возвращает корневой путь хранилища.
	Path() string
}

// ErrRecordNotFound возвращается Vault/репозиториями, когда запись отсутствует.
var ErrRecordNotFound = errNotFound{}

type errNotFound struct{}

func (errNotFound) Error() string { return "запись не найдена" }
