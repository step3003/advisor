// Package catalog — usecase управления категориями (FR-CAT).
//
// Реализует создание/переименование/архивацию и правило запрета жёсткого
// удаления при наличии ссылок (FR-CAT-4), а также сидинг предустановленного
// набора категорий (Приложение A) при первом запуске.
package catalog

import (
	"errors"

	"advisor/internal/application/ports"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
)

// ErrHasReferences — жёсткое удаление запрещено: есть транзакции/планы/подкатегории.
var ErrHasReferences = errors.New("catalog: нельзя удалить категорию со ссылками — используйте архивацию")

// Service — сервис управления категориями.
type Service struct {
	cats  ports.CategoryRepository
	clock ports.Clock
	ids   ports.IDGenerator
}

// New собирает сервис.
func New(cats ports.CategoryRepository, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{cats: cats, clock: clock, ids: ids}
}

// Create создаёт категорию верхнего уровня (FR-CAT-3).
func (s *Service) Create(name string, typ core.EntryType) (*category.Category, error) {
	c, err := category.New(s.ids.NewID(), name, typ, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.cats.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

// CreateSub создаёт подкатегорию (FR-CAT-1).
func (s *Service) CreateSub(name string, typ core.EntryType, parentID string) (*category.Category, error) {
	c, err := category.NewSub(s.ids.NewID(), name, typ, parentID, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.cats.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Rename переименовывает категорию (FR-CAT-3).
func (s *Service) Rename(id, name string) error {
	c, err := s.cats.Get(id)
	if err != nil {
		return err
	}
	if err := c.Rename(name, s.clock.Now()); err != nil {
		return err
	}
	return s.cats.Save(c)
}

// Archive выполняет мягкое удаление (FR-CAT-3).
func (s *Service) Archive(id string) error {
	c, err := s.cats.Get(id)
	if err != nil {
		return err
	}
	if err := c.Archive(s.clock.Now()); err != nil {
		return err
	}
	return s.cats.Save(c)
}

// Unarchive возвращает категорию из архива.
func (s *Service) Unarchive(id string) error {
	c, err := s.cats.Get(id)
	if err != nil {
		return err
	}
	if err := c.Unarchive(s.clock.Now()); err != nil {
		return err
	}
	return s.cats.Save(c)
}

// Delete жёстко удаляет категорию, только если на неё нет ссылок (FR-CAT-4).
func (s *Service) Delete(id string) error {
	has, err := s.cats.HasReferences(id)
	if err != nil {
		return err
	}
	if has {
		return ErrHasReferences
	}
	return s.cats.Delete(id)
}

// List возвращает категории (архивные — по флагу).
func (s *Service) List(includeArchived bool) ([]*category.Category, error) {
	return s.cats.List(includeArchived)
}
