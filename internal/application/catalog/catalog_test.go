package catalog

import (
	"errors"
	"testing"
	"time"

	"advisor/internal/domain/core"
	"advisor/internal/infrastructure/clock"
	"advisor/internal/infrastructure/memory"
)

func newService() (*Service, *memory.Store) {
	store := memory.NewStore()
	clk := clock.Fixed{T: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)}
	return New(store.Categories(), clk, memory.NewSeqIDs()), store
}

func TestSeedDefaults(t *testing.T) {
	svc, _ := newService()
	res, err := svc.SeedDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped {
		t.Fatal("first seed must not be skipped")
	}
	// Приложение A: 8 расходных групп + подкатегории, 5 доходных.
	// Проверяем, что создано разумное число (> число групп).
	if res.Created < 20 {
		t.Errorf("seeded %d categories, expected >= 20", res.Created)
	}

	all, _ := svc.List(true)
	if len(all) != res.Created {
		t.Errorf("list=%d != created=%d", len(all), res.Created)
	}

	// Повторный сидинг идемпотентен.
	res2, err := svc.SeedDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Skipped {
		t.Error("second seed must be skipped")
	}
}

func TestCreateAndArchive(t *testing.T) {
	svc, _ := newService()
	c, err := svc.Create("Тест", core.Expense)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Archive(c.Meta.ID); err != nil {
		t.Fatal(err)
	}
	// Архивная скрыта из списка без флага.
	active, _ := svc.List(false)
	for _, x := range active {
		if x.Meta.ID == c.Meta.ID {
			t.Error("archived category must be hidden")
		}
	}
	all, _ := svc.List(true)
	found := false
	for _, x := range all {
		if x.Meta.ID == c.Meta.ID {
			found = true
		}
	}
	if !found {
		t.Error("archived category must remain in full list")
	}
}

func TestRenameAndSubAndUnarchive(t *testing.T) {
	svc, _ := newService()
	c, err := svc.Create("Еда", core.Expense)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Rename(c.Meta.ID, "Питание"); err != nil {
		t.Fatal(err)
	}
	sub, err := svc.CreateSub("Продукты", core.Expense, c.Meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sub.ParentID != c.Meta.ID {
		t.Errorf("sub parent = %s", sub.ParentID)
	}
	if err := svc.Archive(c.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Unarchive(c.Meta.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.List(false)
	found := false
	for _, x := range got {
		if x.Meta.ID == c.Meta.ID && x.Name == "Питание" {
			found = true
		}
	}
	if !found {
		t.Error("renamed & unarchived category must be active")
	}
}

func TestDeleteWithReferencesBlocked(t *testing.T) {
	svc, store := newService()
	parent, _ := svc.Create("Родитель", core.Expense)
	// Создаём подкатегорию => появляется ссылка на родителя.
	if _, err := svc.CreateSub("Ребёнок", core.Expense, parent.Meta.ID); err != nil {
		t.Fatal(err)
	}

	err := svc.Delete(parent.Meta.ID)
	if !errors.Is(err, ErrHasReferences) {
		t.Errorf("delete referenced category must be blocked, got %v", err)
	}

	// Категория без ссылок удаляется.
	lone, _ := svc.Create("Одинокая", core.Expense)
	if err := svc.Delete(lone.Meta.ID); err != nil {
		t.Errorf("delete lone category failed: %v", err)
	}
	if _, err := store.Categories().Get(lone.Meta.ID); err == nil {
		t.Error("lone category must be gone")
	}
}
