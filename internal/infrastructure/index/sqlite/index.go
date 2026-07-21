// Package sqlite — локальный SQLite-индекс (производный, пересобираемый из vault).
//
// Это НЕ источник правды (FR-SYNC-2): индекс живёт на устройстве, хранит те же
// поля для быстрых запросов/агрегаций (NFR-5) плюс кэш курсов и служебное
// состояние синхронизации (vault_state). Полностью восстанавливается из файлов
// vault методом RebuildFromVault (критерий приёмки 6).
//
// Драйвер — modernc.org/sqlite (чистый Go, без CGo): кроссплатформенная сборка,
// в т.ч. под мобильные цели (NFR-7).
package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"advisor/internal/application/ports"

	_ "modernc.org/sqlite"
)

// Index — обёртка над SQLite-соединением и связанным vault.
type Index struct {
	db    *sql.DB
	vault ports.Vault
}

// Open открывает (создаёт) индекс по пути dbPath и применяет миграции.
// vault используется для пересборки индекса и как источник правды при записи.
func Open(dbPath string, v ports.Vault) (*Index, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	// Прагмы: внешние ключи и разумная долговечность для локального индекса.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: pragma: %w", err)
	}
	idx := &Index{db: db, vault: v}
	if err := idx.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return idx, nil
}

// DB возвращает соединение (для сервисов кэша курсов и т.п.).
func (i *Index) DB() *sql.DB { return i.db }

// Close закрывает соединение.
func (i *Index) Close() error { return i.db.Close() }

// migration — одна версия схемы.
type migration struct {
	version int
	stmts   []string
}

// migrations — упорядоченный список версий схемы (раздел 6.2 ТЗ, расширено
// колонками rev/updated_at для реконструкции сущностей и разрешения конфликтов).
var migrations = []migration{
	{
		version: 1,
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS categories(
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				type TEXT NOT NULL CHECK(type IN ('expense','income')),
				parent_id TEXT NULL REFERENCES categories(id),
				color TEXT NULL,
				icon TEXT NULL,
				is_builtin INTEGER NOT NULL DEFAULT 0,
				archived_at TEXT NULL,
				created_at TEXT NOT NULL,
				rev INTEGER NOT NULL DEFAULT 1,
				updated_at TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS currencies(
				code TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				num_code INTEGER
			)`,
			`CREATE TABLE IF NOT EXISTS exchange_rates(
				currency TEXT NOT NULL,
				rate_date TEXT NOT NULL,
				scale INTEGER NOT NULL,
				rate_byn_minor INTEGER NOT NULL,
				PRIMARY KEY(currency, rate_date)
			)`,
			`CREATE TABLE IF NOT EXISTS plan_items(
				id TEXT PRIMARY KEY,
				year INTEGER NOT NULL,
				month INTEGER NOT NULL,
				category_id TEXT NOT NULL REFERENCES categories(id),
				amount_minor INTEGER NOT NULL,
				currency TEXT NOT NULL,
				note TEXT NULL,
				created_at TEXT NOT NULL,
				rev INTEGER NOT NULL DEFAULT 1,
				updated_at TEXT NOT NULL,
				UNIQUE(year, month, category_id, currency)
			)`,
			`CREATE TABLE IF NOT EXISTS transactions(
				id TEXT PRIMARY KEY,
				occurred_on TEXT NOT NULL,
				type TEXT NOT NULL CHECK(type IN ('expense','income')),
				category_id TEXT NOT NULL REFERENCES categories(id),
				amount_minor INTEGER NOT NULL,
				currency TEXT NOT NULL,
				note TEXT NULL,
				recurring_id TEXT NULL,
				created_at TEXT NOT NULL,
				rev INTEGER NOT NULL DEFAULT 1,
				updated_at TEXT NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS ix_tx_date ON transactions(occurred_on)`,
			`CREATE INDEX IF NOT EXISTS ix_tx_cat ON transactions(category_id)`,
			`CREATE TABLE IF NOT EXISTS recurring_templates(
				id TEXT PRIMARY KEY,
				type TEXT NOT NULL,
				category_id TEXT NOT NULL,
				amount_minor INTEGER NOT NULL,
				currency TEXT NOT NULL,
				day_of_month INTEGER NOT NULL,
				start_date TEXT NOT NULL,
				end_date TEXT NULL,
				auto_create_fact INTEGER NOT NULL DEFAULT 0,
				active INTEGER NOT NULL DEFAULT 1,
				created_at TEXT NOT NULL,
				rev INTEGER NOT NULL DEFAULT 1,
				updated_at TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS settings(
				key TEXT PRIMARY KEY,
				value TEXT
			)`,
			`CREATE TABLE IF NOT EXISTS vault_state(
				path TEXT PRIMARY KEY,
				collection TEXT NOT NULL,
				partition TEXT NOT NULL,
				id TEXT NOT NULL,
				rev INTEGER NOT NULL,
				mtime TEXT NOT NULL,
				hash TEXT NOT NULL
			)`,
		},
	},
	{
		// v2: разбор банковских SMS (FR-SMS) — настраиваемые шаблоны + входящие черновики.
		version: 2,
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS sms_templates(
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				sender TEXT NOT NULL,              -- подстрока отправителя SMS (напр. "Priorbank")
				pattern TEXT NOT NULL,             -- regex по тексту SMS
				amount_group INTEGER NOT NULL,     -- номер группы суммы
				currency_group INTEGER NOT NULL,   -- номер группы валюты; 0 => fixed_currency
				fixed_currency TEXT NOT NULL,      -- валюта, если currency_group=0
				type TEXT NOT NULL CHECK(type IN ('expense','income')),
				default_category_id TEXT NULL REFERENCES categories(id),
				enabled INTEGER NOT NULL DEFAULT 1,
				priority INTEGER NOT NULL DEFAULT 0,
				created_at TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS inbox_drafts(
				id TEXT PRIMARY KEY,
				source TEXT NOT NULL,              -- 'sms'
				raw_sender TEXT NOT NULL,
				raw_text TEXT NOT NULL,
				received_at TEXT NOT NULL,         -- YYYY-MM-DD
				parsed_amount_minor INTEGER NULL,
				parsed_currency TEXT NULL,
				parsed_type TEXT NULL,
				template_id TEXT NULL,
				resolved INTEGER NOT NULL DEFAULT 0,
				created_at TEXT NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS ix_drafts_unresolved ON inbox_drafts(resolved, received_at)`,
		},
	},
}

// migrate применяет недостающие версии схемы (NFR-6: версионирование, авто-миграция).
func (i *Index) migrate() error {
	if _, err := i.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("sqlite: создание schema_migrations: %w", err)
	}

	var current int
	// COALESCE вернёт 0, если версий ещё нет.
	if err := i.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("sqlite: чтение версии схемы: %w", err)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		tx, err := i.db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range m.stmts {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("sqlite: миграция v%d: %w", m.version, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			m.version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// SchemaVersion возвращает текущую применённую версию схемы.
func (i *Index) SchemaVersion() (int, error) {
	var v int
	err := i.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&v)
	return v, err
}
