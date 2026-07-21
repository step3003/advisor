// Package vault — файловое хранилище-источник правды (FR-SYNC).
//
// Модель как у Obsidian: каждая запись — отдельный маленький JSON-файл
// <collection>/[YYYY-MM/]<uuid>.json в синхронизируемой папке (iCloud Drive).
// SQLite в vault НЕ хранится (FR-SYNC-2) — это отдельный локальный индекс.
//
// Реализация оперирует сырыми байтами и не знает о доменных типах. Путь к папке
// задаётся параметром (в тестах — t.TempDir()); подключение реального
// iCloud-контейнера на iOS/macOS выполняется на этапе сборки платформы и не
// затрагивает домен/усечейсы — меняется только этот путь.
package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"advisor/internal/application/ports"
)

const (
	fileExt      = ".json"
	conflictsDir = "_conflicts"
	tmpPrefix    = ".tmp-"
	uuidLen      = 36
)

// knownCollections — коллекции с фиксированным набором (для листинга/валидации).
var knownCollections = map[string]bool{
	ports.CollectionCategories:   true,
	ports.CollectionTransactions: true,
	ports.CollectionPlans:        true,
	ports.CollectionRecurring:    true,
}

// FileVault — реализация ports.Vault поверх файловой системы.
type FileVault struct {
	root string
}

// New создаёт хранилище с корнем root, создавая папку при необходимости.
func New(root string) (*FileVault, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("vault: пустой путь хранилища")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("vault: создание корня: %w", err)
	}
	return &FileVault{root: root}, nil
}

// Path возвращает корневой путь хранилища.
func (v *FileVault) Path() string { return v.root }

// dirFor возвращает каталог для коллекции с учётом партиции.
func (v *FileVault) dirFor(collection, partition string) string {
	if partition == "" {
		return filepath.Join(v.root, collection)
	}
	return filepath.Join(v.root, collection, partition)
}

// fileFor возвращает путь к файлу записи.
func (v *FileVault) fileFor(ref ports.RecordRef) string {
	return filepath.Join(v.dirFor(ref.Collection, ref.Partition), ref.ID+fileExt)
}

// Put атомарно записывает запись: temp-файл в том же каталоге + rename.
func (v *FileVault) Put(rec ports.Record) error {
	if rec.ID == "" || rec.Collection == "" {
		return errors.New("vault: запись без collection/id")
	}
	dir := v.dirFor(rec.Collection, rec.Partition)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("vault: создание каталога %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, tmpPrefix+rec.ID+"-*"+fileExt)
	if err != nil {
		return fmt.Errorf("vault: создание temp-файла: %w", err)
	}
	tmpName := tmp.Name()
	// Гарантируем очистку temp при ошибке.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(rec.Data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("vault: запись temp-файла: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("vault: sync temp-файла: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("vault: закрытие temp-файла: %w", err)
	}
	if err := os.Rename(tmpName, v.fileFor(rec.RecordRef)); err != nil {
		return fmt.Errorf("vault: атомарный rename: %w", err)
	}
	return nil
}

// Get читает запись по ссылке.
func (v *FileVault) Get(ref ports.RecordRef) (ports.Record, error) {
	path := v.fileFor(ref)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ports.Record{}, ports.ErrRecordNotFound
		}
		return ports.Record{}, fmt.Errorf("vault: чтение %s: %w", path, err)
	}
	info, _ := os.Stat(path)
	out := ports.Record{RecordRef: ref, Data: data}
	if info != nil {
		out.ModTime = info.ModTime().UTC()
	}
	out.Hash = hashBytes(data)
	return out, nil
}

// Delete удаляет запись.
func (v *FileVault) Delete(ref ports.RecordRef) error {
	err := os.Remove(v.fileFor(ref))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("vault: удаление: %w", err)
	}
	return nil
}

// List перечисляет все записи всех известных коллекций.
func (v *FileVault) List() ([]ports.RecordRef, error) {
	var refs []ports.RecordRef
	for collection := range knownCollections {
		collDir := filepath.Join(v.root, collection)
		entries, err := os.ReadDir(collDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("vault: чтение %s: %w", collDir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				// Партиция YYYY-MM.
				partition := e.Name()
				partRefs, err := v.listPartition(collection, partition)
				if err != nil {
					return nil, err
				}
				refs = append(refs, partRefs...)
				continue
			}
			ref, ok := v.refFromFile(collection, "", e.Name())
			if ok {
				refs = append(refs, ref)
			}
		}
	}
	return refs, nil
}

func (v *FileVault) listPartition(collection, partition string) ([]ports.RecordRef, error) {
	dir := filepath.Join(v.root, collection, partition)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("vault: чтение %s: %w", dir, err)
	}
	var refs []ports.RecordRef
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ref, ok := v.refFromFile(collection, partition, e.Name()); ok {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

// refFromFile строит RecordRef из имени файла записи, отсекая temp и конфликтные копии.
func (v *FileVault) refFromFile(collection, partition, name string) (ports.RecordRef, bool) {
	if strings.HasPrefix(name, tmpPrefix) || !strings.HasSuffix(name, fileExt) {
		return ports.RecordRef{}, false
	}
	base := strings.TrimSuffix(name, fileExt)
	// Каноничная запись: имя ровно UUID (36 символов). Иные — конфликтные копии.
	if len(base) != uuidLen {
		return ports.RecordRef{}, false
	}
	ref := ports.RecordRef{Collection: collection, Partition: partition, ID: base}
	path := filepath.Join(v.dirFor(collection, partition), name)
	if info, err := os.Stat(path); err == nil {
		ref.ModTime = info.ModTime().UTC()
		if data, err := os.ReadFile(path); err == nil {
			ref.Hash = hashBytes(data)
		}
	}
	return ref, true
}

// ReadMeta читает служебный файл верхнего уровня (advisor.json, settings.json).
func (v *FileVault) ReadMeta(name string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(v.root, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, fmt.Errorf("vault: чтение meta %s: %w", name, err)
	}
	return data, nil
}

// WriteMeta атомарно пишет служебный файл верхнего уровня.
func (v *FileVault) WriteMeta(name string, data []byte) error {
	tmp, err := os.CreateTemp(v.root, tmpPrefix+name+"-*")
	if err != nil {
		return fmt.Errorf("vault: temp meta: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(v.root, name))
}

// hashBytes возвращает sha256-хеш содержимого в hex.
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// recordVersion — минимальные поля для разрешения конфликтов (общие для всех записей).
type recordVersion struct {
	Rev       int64  `json:"rev"`
	UpdatedAt string `json:"updated_at"`
}

// parseVersion извлекает (rev, updated_at) из произвольного JSON записи.
func parseVersion(data []byte) (int64, time.Time, error) {
	var rv recordVersion
	if err := json.Unmarshal(data, &rv); err != nil {
		return 0, time.Time{}, err
	}
	var ts time.Time
	if rv.UpdatedAt != "" {
		t, err := time.Parse(time.RFC3339, rv.UpdatedAt)
		if err == nil {
			ts = t.UTC()
		}
	}
	return rv.Rev, ts, nil
}
