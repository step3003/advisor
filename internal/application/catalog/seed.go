package catalog

import (
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
)

// presetGroup — предустановленная категория с подкатегориями (Приложение A ТЗ).
type presetGroup struct {
	name string
	subs []string
}

var presetExpenses = []presetGroup{
	{"Еда", []string{"Продукты", "Кафе и рестораны"}},
	{"Транспорт", []string{"Общественный", "Такси", "Топливо"}},
	{"Жильё", []string{"Аренда", "Коммунальные", "Интернет/связь"}},
	{"Здоровье", []string{"Аптека", "Врачи"}},
	{"Развлечения", []string{"Подписки", "Хобби"}},
	{"Одежда", nil},
	{"Образование", nil},
	{"Прочее", nil},
}

var presetIncomes = []presetGroup{
	{"Зарплата", nil},
	{"Подработка", nil},
	{"Подарки", nil},
	{"Проценты/инвестиции", nil},
	{"Прочее", nil},
}

// SeedResult — итог сидинга.
type SeedResult struct {
	Created int
	Skipped bool // true => категории уже были, сидинг пропущен
}

// SeedDefaults создаёт предустановленный набор категорий при первом запуске
// (FR-CAT-2). Идемпотентно: если категории уже есть — ничего не делает.
func (s *Service) SeedDefaults() (SeedResult, error) {
	existing, err := s.cats.List(true)
	if err != nil {
		return SeedResult{}, err
	}
	if len(existing) > 0 {
		return SeedResult{Skipped: true}, nil
	}

	created := 0
	now := s.clock.Now()

	seedGroup := func(groups []presetGroup, typ core.EntryType) error {
		for _, g := range groups {
			parent, err := category.NewBuiltin(s.ids.NewID(), g.name, typ, "", now)
			if err != nil {
				return err
			}
			if err := s.cats.Save(parent); err != nil {
				return err
			}
			created++
			for _, sub := range g.subs {
				child, err := category.NewBuiltin(s.ids.NewID(), sub, typ, parent.Meta.ID, now)
				if err != nil {
					return err
				}
				if err := s.cats.Save(child); err != nil {
					return err
				}
				created++
			}
		}
		return nil
	}

	if err := seedGroup(presetExpenses, core.Expense); err != nil {
		return SeedResult{}, err
	}
	if err := seedGroup(presetIncomes, core.Income); err != nil {
		return SeedResult{}, err
	}
	return SeedResult{Created: created}, nil
}
