// Package category — доменная сущность категории/подкатегории (FR-CAT).
//
// Двухуровневая структура: категория верхнего уровня (ParentID == "") и
// подкатегория (ParentID указывает на родителя). Жёсткое удаление запрещено
// правилами usecase; домен предоставляет мягкое удаление — архивацию (FR-CAT-3/4).
package category

import (
	"errors"
	"strings"
	"time"

	"advisor/internal/domain/core"
)

// Ошибки домена категорий.
var (
	ErrEmptyName       = errors.New("category: пустое название")
	ErrInvalidType     = core.ErrInvalidEntryType
	ErrAlreadyArchived = errors.New("category: категория уже архивирована")
	ErrNotArchived     = errors.New("category: категория не архивирована")
)

// Category — категория или подкатегория трат/доходов.
type Category struct {
	Meta core.Meta

	Name     string
	Type     core.EntryType
	ParentID string // "" => категория верхнего уровня; иначе ID родителя

	Color string // опц., для графиков (FR-CAT-5)
	Icon  string // опц.

	IsBuiltin  bool       // предустановленная категория (Приложение A)
	CreatedAt  time.Time  // UTC
	ArchivedAt *time.Time // nil => активна; иначе момент архивации (UTC)
}

// New создаёт категорию верхнего уровня.
func New(id, name string, typ core.EntryType, now time.Time) (*Category, error) {
	return build(id, name, typ, "", false, now)
}

// NewSub создаёт подкатегорию с указанным родителем.
func NewSub(id, name string, typ core.EntryType, parentID string, now time.Time) (*Category, error) {
	if strings.TrimSpace(parentID) == "" {
		return nil, errors.New("category: у подкатегории должен быть parentID")
	}
	return build(id, name, typ, parentID, false, now)
}

// NewBuiltin создаёт предустановленную категорию/подкатегорию (сидинг).
func NewBuiltin(id, name string, typ core.EntryType, parentID string, now time.Time) (*Category, error) {
	return build(id, name, typ, parentID, true, now)
}

func build(id, name string, typ core.EntryType, parentID string, builtin bool, now time.Time) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	if !typ.Valid() {
		return nil, ErrInvalidType
	}
	return &Category{
		Meta:      core.NewMeta(id, now),
		Name:      name,
		Type:      typ,
		ParentID:  parentID,
		IsBuiltin: builtin,
		CreatedAt: now.UTC(),
	}, nil
}

// IsTopLevel сообщает, что это категория верхнего уровня.
func (c *Category) IsTopLevel() bool {
	return c.ParentID == ""
}

// IsArchived сообщает, что категория архивирована.
func (c *Category) IsArchived() bool {
	return c.ArchivedAt != nil
}

// Rename переименовывает категорию (FR-CAT-3) и повышает ревизию.
func (c *Category) Rename(name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	c.Name = name
	c.Meta = c.Meta.Touch(now)
	return nil
}

// Archive выполняет мягкое удаление (FR-CAT-3/4).
func (c *Category) Archive(now time.Time) error {
	if c.IsArchived() {
		return ErrAlreadyArchived
	}
	t := now.UTC()
	c.ArchivedAt = &t
	c.Meta = c.Meta.Touch(now)
	return nil
}

// Unarchive возвращает категорию из архива.
func (c *Category) Unarchive(now time.Time) error {
	if !c.IsArchived() {
		return ErrNotArchived
	}
	c.ArchivedAt = nil
	c.Meta = c.Meta.Touch(now)
	return nil
}

// SetStyle задаёт цвет/иконку для отображения (FR-CAT-5).
func (c *Category) SetStyle(color, icon string, now time.Time) {
	c.Color = color
	c.Icon = icon
	c.Meta = c.Meta.Touch(now)
}
