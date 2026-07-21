package category

import (
	"testing"
	"time"

	"advisor/internal/domain/core"
)

var now = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func TestNewCategory(t *testing.T) {
	c, err := New("cat-1", "Еда", core.Expense, now)
	if err != nil {
		t.Fatal(err)
	}
	if !c.IsTopLevel() {
		t.Error("must be top level")
	}
	if c.Meta.Rev != 1 {
		t.Errorf("rev = %d", c.Meta.Rev)
	}
	if c.IsArchived() {
		t.Error("new category must not be archived")
	}
}

func TestNewCategoryValidation(t *testing.T) {
	if _, err := New("id", "  ", core.Expense, now); err == nil {
		t.Error("empty name must error")
	}
	if _, err := New("id", "X", core.EntryType("bad"), now); err == nil {
		t.Error("bad type must error")
	}
	if _, err := NewSub("id", "Sub", core.Expense, "", now); err == nil {
		t.Error("sub without parent must error")
	}
}

func TestSubCategory(t *testing.T) {
	c, err := NewSub("sub-1", "Продукты", core.Expense, "cat-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if c.IsTopLevel() {
		t.Error("sub must not be top level")
	}
	if c.ParentID != "cat-1" {
		t.Errorf("parent = %s", c.ParentID)
	}
}

func TestArchiveLifecycle(t *testing.T) {
	c, _ := New("cat-1", "Еда", core.Expense, now)
	later := now.Add(time.Hour)
	if err := c.Archive(later); err != nil {
		t.Fatal(err)
	}
	if !c.IsArchived() {
		t.Error("must be archived")
	}
	if c.Meta.Rev != 2 {
		t.Errorf("rev after archive = %d, want 2", c.Meta.Rev)
	}
	// Повторная архивация запрещена.
	if err := c.Archive(later); err == nil {
		t.Error("double archive must error")
	}
	// Разархивация.
	if err := c.Unarchive(later); err != nil {
		t.Fatal(err)
	}
	if c.IsArchived() {
		t.Error("must be active after unarchive")
	}
	if err := c.Unarchive(later); err == nil {
		t.Error("unarchive active must error")
	}
}

func TestRename(t *testing.T) {
	c, _ := New("cat-1", "Еда", core.Expense, now)
	if err := c.Rename("Питание", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.Name != "Питание" {
		t.Errorf("name = %s", c.Name)
	}
	if c.Meta.Rev != 2 {
		t.Errorf("rev = %d", c.Meta.Rev)
	}
	if err := c.Rename("", now); err == nil {
		t.Error("empty rename must error")
	}
}
