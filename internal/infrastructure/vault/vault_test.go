package vault

import (
	"os"
	"path/filepath"
	"testing"

	"advisor/internal/application/ports"
)

func newVault(t *testing.T) *FileVault {
	t.Helper()
	v, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestPutGetDelete(t *testing.T) {
	v := newVault(t)
	ref := ports.RecordRef{Collection: ports.CollectionCategories, ID: "11111111-1111-4111-8111-111111111111"}
	rec := ports.Record{RecordRef: ref, Data: []byte(`{"rev":1,"updated_at":"2026-07-20T10:00:00Z"}`)}

	if err := v.Put(rec); err != nil {
		t.Fatal(err)
	}
	got, err := v.Get(ref)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Data) != string(rec.Data) {
		t.Errorf("data mismatch: %s", got.Data)
	}
	if got.Hash == "" {
		t.Error("hash must be set")
	}
	if err := v.Delete(ref); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get(ref); err != ports.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestPutPartitioned(t *testing.T) {
	v := newVault(t)
	ref := ports.RecordRef{
		Collection: ports.CollectionTransactions,
		Partition:  "2026-07",
		ID:         "22222222-2222-4222-8222-222222222222",
	}
	rec := ports.Record{RecordRef: ref, Data: []byte(`{"rev":1,"updated_at":"2026-07-20T10:00:00Z"}`)}
	if err := v.Put(rec); err != nil {
		t.Fatal(err)
	}
	// Проверяем физическое расположение файла.
	want := filepath.Join(v.Path(), "transactions", "2026-07", ref.ID+".json")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected file at %s: %v", want, err)
	}
}

func TestList(t *testing.T) {
	v := newVault(t)
	records := []ports.Record{
		{RecordRef: ports.RecordRef{Collection: ports.CollectionCategories, ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}, Data: []byte(`{"rev":1}`)},
		{RecordRef: ports.RecordRef{Collection: ports.CollectionTransactions, Partition: "2026-07", ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}, Data: []byte(`{"rev":1}`)},
		{RecordRef: ports.RecordRef{Collection: ports.CollectionTransactions, Partition: "2026-08", ID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc"}, Data: []byte(`{"rev":1}`)},
		{RecordRef: ports.RecordRef{Collection: ports.CollectionRecurring, ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd"}, Data: []byte(`{"rev":1}`)},
	}
	for _, r := range records {
		if err := v.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	refs, err := v.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 4 {
		t.Fatalf("List = %d refs, want 4", len(refs))
	}
	// Meta-файлы не должны попадать в листинг.
	if err := v.WriteMeta("advisor.json", []byte(`{"format_version":1}`)); err != nil {
		t.Fatal(err)
	}
	refs2, _ := v.List()
	if len(refs2) != 4 {
		t.Errorf("meta must not be listed: got %d", len(refs2))
	}
}

func TestMetaRoundTrip(t *testing.T) {
	v := newVault(t)
	if _, err := v.ReadMeta("settings.json"); err != ports.ErrRecordNotFound {
		t.Errorf("missing meta must be ErrRecordNotFound, got %v", err)
	}
	if err := v.WriteMeta("settings.json", []byte(`{"default_currency":"BYN"}`)); err != nil {
		t.Fatal(err)
	}
	data, err := v.ReadMeta("settings.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"default_currency":"BYN"}` {
		t.Errorf("meta mismatch: %s", data)
	}
}

func TestResolveConflicts(t *testing.T) {
	v := newVault(t)
	uuid := "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	dir := filepath.Join(v.Path(), "categories")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Каноничный файл rev=2, конфликтная копия iCloud rev=3 (должна победить).
	canonical := filepath.Join(dir, uuid+".json")
	conflict := filepath.Join(dir, uuid+" 2.json")
	if err := os.WriteFile(canonical, []byte(`{"rev":2,"updated_at":"2026-07-20T10:00:00Z","name":"старое"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conflict, []byte(`{"rev":3,"updated_at":"2026-07-20T11:00:00Z","name":"новое"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts, err := v.ResolveConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	// Каноничный файл теперь содержит победителя (rev=3).
	data, _ := os.ReadFile(canonical)
	if !contains(string(data), `"rev":3`) {
		t.Errorf("winner not written to canonical: %s", data)
	}
	// Конфликтная копия убрана из коллекции.
	if _, err := os.Stat(conflict); !os.IsNotExist(err) {
		t.Error("conflict copy must be removed from collection dir")
	}
	// Проигравшая версия сохранена в _conflicts/ (данные не теряются).
	loserDir := filepath.Join(v.Path(), "_conflicts", "categories")
	entries, _ := os.ReadDir(loserDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 loser in _conflicts, got %d", len(entries))
	}
	loserData, _ := os.ReadFile(filepath.Join(loserDir, entries[0].Name()))
	if !contains(string(loserData), `"rev":2`) {
		t.Errorf("loser (rev=2) not preserved: %s", loserData)
	}
}

func TestResolveConflictsNoConflict(t *testing.T) {
	v := newVault(t)
	ref := ports.RecordRef{Collection: ports.CollectionCategories, ID: "ffffffff-ffff-4fff-8fff-ffffffffffff"}
	_ = v.Put(ports.Record{RecordRef: ref, Data: []byte(`{"rev":1,"updated_at":"2026-07-20T10:00:00Z"}`)})
	conflicts, err := v.ResolveConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
