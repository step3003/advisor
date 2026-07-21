// Package nopvault — заглушка ports.Vault для серверного режима.
//
// В клиент-серверной архитектуре (ТЗ v2.0) источник правды — серверная БД
// (SQLite), а не файловый vault. Репозитории слоя index/sqlite исторически
// пишут и в vault, и в БД; здесь vault-операции превращаются в no-op, так что
// данные живут только в БД. Чтения репозиториев идут из БД напрямую, поэтому
// заглушка не влияет на корректность.
package nopvault

import "advisor/internal/application/ports"

// Vault — no-op реализация ports.Vault (ничего не хранит).
type Vault struct{}

// New создаёт no-op vault.
func New() *Vault { return &Vault{} }

func (*Vault) Put(ports.Record) error       { return nil }
func (*Vault) Delete(ports.RecordRef) error { return nil }

func (*Vault) Get(ports.RecordRef) (ports.Record, error) {
	return ports.Record{}, ports.ErrRecordNotFound
}

func (*Vault) List() ([]ports.RecordRef, error) { return nil, nil }

func (*Vault) ReadMeta(string) ([]byte, error) { return nil, ports.ErrRecordNotFound }
func (*Vault) WriteMeta(string, []byte) error  { return nil }

func (*Vault) ResolveConflicts() ([]ports.Conflict, error) { return nil, nil }

func (*Vault) Path() string { return "(server db — no vault)" }
