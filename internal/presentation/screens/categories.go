package screens

import (
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
	"advisor/internal/presentation/i18n"
)

// CategoriesScreen — управление категориями и подкатегориями (FR-CAT).
type CategoriesScreen struct {
	d            *Deps
	showArchived bool
	list         *fyne.Container
	root         *fyne.Container
}

// NewCategoriesScreen создаёт экран категорий.
func NewCategoriesScreen(d *Deps) *CategoriesScreen {
	return &CategoriesScreen{d: d}
}

func (s *CategoriesScreen) Title() string       { return i18n.NavCategories }
func (s *CategoriesScreen) Icon() fyne.Resource { return theme.ListIcon() }

func (s *CategoriesScreen) Build() fyne.CanvasObject {
	archCheck := widget.NewCheck(i18n.CatShowArchived, func(v bool) {
		s.showArchived = v
		s.Refresh()
	})
	addExpense := widget.NewButtonWithIcon(i18n.CatAddTop, theme.ContentAddIcon(), func() {
		s.openCreateTop(core.Expense)
	})
	addIncome := widget.NewButton(i18n.CatIncomesSection, func() { s.openCreateTop(core.Income) })
	addIncome.SetIcon(theme.ContentAddIcon())

	header := container.NewVBox(
		container.NewHBox(addExpense, addIncome),
		archCheck,
		widget.NewSeparator(),
	)
	s.list = container.NewVBox()
	s.root = container.NewBorder(header, nil, nil, nil, container.NewVScroll(s.list))
	s.Refresh()
	return s.root
}

func (s *CategoriesScreen) Refresh() {
	if s.list == nil {
		return
	}
	cats, err := s.d.Catalog.List(s.showArchived)
	if showError(s.d.Window, err) {
		return
	}
	s.list.Objects = nil
	s.list.Add(sectionTitle(i18n.CatExpensesSection))
	s.addTypeSection(cats, core.Expense)
	s.list.Add(widget.NewSeparator())
	s.list.Add(sectionTitle(i18n.CatIncomesSection))
	s.addTypeSection(cats, core.Income)
	s.list.Refresh()
}

func (s *CategoriesScreen) addTypeSection(cats []*category.Category, typ core.EntryType) {
	tops := filterCats(cats, func(c *category.Category) bool {
		return c.Type == typ && c.IsTopLevel()
	})
	sortByName(tops)
	for _, parent := range tops {
		s.list.Add(s.catRow(parent, false))
		children := filterCats(cats, func(c *category.Category) bool {
			return c.ParentID == parent.Meta.ID
		})
		sortByName(children)
		for _, child := range children {
			s.list.Add(s.catRow(child, true))
		}
	}
}

func (s *CategoriesScreen) catRow(c *category.Category, isChild bool) fyne.CanvasObject {
	name := c.Name
	if c.IsBuiltin {
		name += "  · " + i18n.CatBuiltin
	}
	if c.IsArchived() {
		name += "  · " + i18n.CatArchived
	}
	label := widget.NewLabel(name)
	if isChild {
		label.SetText("    " + name)
	}

	id := c.Meta.ID
	var actions []fyne.CanvasObject
	rename := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() { s.openRename(c) })
	actions = append(actions, rename)

	if !isChild {
		addSub := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() { s.openCreateSub(c) })
		actions = append(actions, addSub)
	}
	if c.IsArchived() {
		restore := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
			if !showError(s.d.Window, s.d.Catalog.Unarchive(id)) {
				s.Refresh()
			}
		})
		actions = append(actions, restore)
	} else {
		archive := widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
			if !showError(s.d.Window, s.d.Catalog.Archive(id)) {
				s.Refresh()
			}
		})
		actions = append(actions, archive)
	}
	del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() { s.deleteCat(id) })
	actions = append(actions, del)

	return container.NewBorder(nil, nil, nil, container.NewHBox(actions...), label)
}

func (s *CategoriesScreen) deleteCat(id string) {
	confirm(s.d.Window, i18n.CatConfirmDelete, func() {
		if err := s.d.Catalog.Delete(id); err != nil {
			// Удаление заблокировано при наличии ссылок (FR-CAT-4).
			showError(s.d.Window, errText(i18n.CatDeleteBlocked))
			return
		}
		s.Refresh()
	})
}

func (s *CategoriesScreen) openCreateTop(typ core.EntryType) {
	nameEntry := widget.NewEntry()
	items := []*widget.FormItem{{Text: i18n.CatName, Widget: nameEntry}}
	formDialog(s.d.Window, i18n.CatAddTop, items, func() {
		if nameEntry.Text == "" {
			showError(s.d.Window, errText(i18n.MsgEmptyName))
			return
		}
		if _, err := s.d.Catalog.Create(nameEntry.Text, typ); showError(s.d.Window, err) {
			return
		}
		s.Refresh()
	})
}

func (s *CategoriesScreen) openCreateSub(parent *category.Category) {
	nameEntry := widget.NewEntry()
	items := []*widget.FormItem{{Text: i18n.CatName, Widget: nameEntry}}
	formDialog(s.d.Window, i18n.CatAddSub, items, func() {
		if nameEntry.Text == "" {
			showError(s.d.Window, errText(i18n.MsgEmptyName))
			return
		}
		if _, err := s.d.Catalog.CreateSub(nameEntry.Text, parent.Type, parent.Meta.ID); showError(s.d.Window, err) {
			return
		}
		s.Refresh()
	})
}

func (s *CategoriesScreen) openRename(c *category.Category) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(c.Name)
	items := []*widget.FormItem{{Text: i18n.CatName, Widget: nameEntry}}
	formDialog(s.d.Window, i18n.CatRenameTitle, items, func() {
		if nameEntry.Text == "" {
			showError(s.d.Window, errText(i18n.MsgEmptyName))
			return
		}
		if err := s.d.Catalog.Rename(c.Meta.ID, nameEntry.Text); showError(s.d.Window, err) {
			return
		}
		s.Refresh()
	})
}

func filterCats(cats []*category.Category, keep func(*category.Category) bool) []*category.Category {
	var out []*category.Category
	for _, c := range cats {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

func sortByName(cats []*category.Category) {
	sort.Slice(cats, func(i, j int) bool { return cats[i].Name < cats[j].Name })
}
