package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"advisor/internal/application/ports"
)

// conflictFile — один файл-кандидат в группе с общим UUID.
type conflictFile struct {
	path        string
	name        string
	isCanonical bool
	rev         int64
	tsUnix      int64
	data        []byte
}

// ResolveConflicts обнаруживает iCloud-конфликтные копии и разрешает их (FR-SYNC-5).
//
// iCloud создаёт копии вида "<uuid> 2.json" рядом с каноничным "<uuid>.json".
// Для каждой группы с общим UUID выбирается победитель по (rev, updated_at),
// его содержимое кладётся в каноничный файл, проигравшие переносятся в
// _conflicts/ — данные не теряются.
func (v *FileVault) ResolveConflicts() ([]ports.Conflict, error) {
	var conflicts []ports.Conflict
	for collection := range knownCollections {
		collDir := filepath.Join(v.root, collection)
		entries, err := os.ReadDir(collDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		// Плоские файлы коллекции (partition "").
		if c, err := v.resolveDir(collection, ""); err != nil {
			return nil, err
		} else {
			conflicts = append(conflicts, c...)
		}
		// Партиции YYYY-MM.
		for _, e := range entries {
			if e.IsDir() {
				if c, err := v.resolveDir(collection, e.Name()); err != nil {
					return nil, err
				} else {
					conflicts = append(conflicts, c...)
				}
			}
		}
	}
	return conflicts, nil
}

// resolveDir разрешает конфликты в одном каталоге коллекции/партиции.
func (v *FileVault) resolveDir(collection, partition string) ([]ports.Conflict, error) {
	dir := v.dirFor(collection, partition)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Группируем файлы по ведущему UUID.
	groups := map[string][]conflictFile{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, tmpPrefix) || !strings.HasSuffix(name, fileExt) {
			continue
		}
		base := strings.TrimSuffix(name, fileExt)
		var uuid string
		var canonical bool
		switch {
		case len(base) == uuidLen && isUUID(base):
			uuid, canonical = base, true
		case len(base) > uuidLen && isUUID(base[:uuidLen]) && isConflictMarker(base[uuidLen]):
			uuid, canonical = base[:uuidLen], false
		default:
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rev, ts, _ := parseVersion(data)
		groups[uuid] = append(groups[uuid], conflictFile{
			path: path, name: name, isCanonical: canonical,
			rev: rev, tsUnix: ts.Unix(), data: data,
		})
	}

	var result []ports.Conflict
	for uuid, files := range groups {
		if len(files) < 2 {
			continue // конфликта нет
		}
		winner := pickWinner(files)
		canonicalPath := filepath.Join(dir, uuid+fileExt)

		// Порядок важен: проигравшие (в т.ч. каноничный файл с тем же именем)
		// переносим в _conflicts И удаляем ПЕРЕД записью победителя — иначе
		// удаление одноимённого проигравшего затрёт свежий каноничный файл.
		for _, f := range files {
			if f.path == winner.path {
				continue
			}
			loserDest, err := v.moveToConflicts(collection, partition, f.name, f.data)
			if err != nil {
				return nil, err
			}
			result = append(result, ports.Conflict{
				Ref:         ports.RecordRef{Collection: collection, Partition: partition, ID: uuid},
				WinnerPath:  canonicalPath,
				LoserPath:   loserDest,
				Description: fmt.Sprintf("конфликт %s/%s: победила rev=%d", collection, uuid, winner.rev),
			})
		}

		// Удаляем исходный файл победителя, если это была конфликтная копия.
		if !winner.isCanonical {
			_ = os.Remove(winner.path)
		}
		// Пишем содержимое победителя в каноничный файл.
		if err := v.writeAtomic(dir, uuid+fileExt, winner.data); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// pickWinner выбирает победителя: большая rev, при равенстве — более позднее
// updated_at, при полном равенстве — каноничный файл.
func pickWinner(files []conflictFile) conflictFile {
	best := files[0]
	for _, f := range files[1:] {
		if f.rev != best.rev {
			if f.rev > best.rev {
				best = f
			}
			continue
		}
		if f.tsUnix != best.tsUnix {
			if f.tsUnix > best.tsUnix {
				best = f
			}
			continue
		}
		if f.isCanonical && !best.isCanonical {
			best = f
		}
	}
	return best
}

// moveToConflicts переносит проигравшую версию в _conflicts/, сохраняя данные.
func (v *FileVault) moveToConflicts(collection, partition, name string, data []byte) (string, error) {
	destDir := filepath.Join(v.root, conflictsDir, collection, partition)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, name)
	// Избегаем перезаписи ранее сохранённых конфликтов.
	for i := 1; ; i++ {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			break
		}
		dest = filepath.Join(destDir, fmt.Sprintf("%s.%d", name, i))
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return "", err
	}
	// Удаляем исходный проигравший файл из коллекции.
	_ = os.Remove(filepath.Join(v.dirFor(collection, partition), name))
	return dest, nil
}

// writeAtomic пишет файл в каталог атомарно (temp + rename).
func (v *FileVault) writeAtomic(dir, name string, data []byte) error {
	tmp, err := os.CreateTemp(dir, tmpPrefix+"resolve-*")
	if err != nil {
		return err
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
	return os.Rename(tmpName, filepath.Join(dir, name))
}

// isConflictMarker сообщает, что символ после UUID начинает суффикс конфликтной копии.
func isConflictMarker(c byte) bool {
	return c == ' ' || c == '(' || c == '-' || c == '_'
}

// isUUID выполняет мягкую проверку канонической формы UUID (8-4-4-4-12 hex).
func isUUID(s string) bool {
	if len(s) != uuidLen {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
