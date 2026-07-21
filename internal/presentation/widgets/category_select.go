package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
)

// noneSubLabel — пункт «нет подкатегории»: операция относится прямо к категории.
const noneSubLabel = "(нет)"

// CatOption — вариант выбора категории: стабильный ID и подпись.
type CatOption struct {
	ID    string
	Label string
}

// TopLevelOptions возвращает активные категории верхнего уровня заданного типа.
func TopLevelOptions(cats []*category.Category, typ core.EntryType) []CatOption {
	var out []CatOption
	for _, c := range cats {
		if c.IsArchived() || !c.IsTopLevel() || c.Type != typ {
			continue
		}
		out = append(out, CatOption{ID: c.Meta.ID, Label: c.Name})
	}
	return out
}

// ChildOptions возвращает активные подкатегории указанного родителя.
func ChildOptions(cats []*category.Category, parentID string) []CatOption {
	var out []CatOption
	for _, c := range cats {
		if c.IsArchived() || c.ParentID != parentID {
			continue
		}
		out = append(out, CatOption{ID: c.Meta.ID, Label: c.Name})
	}
	return out
}

// CategorySelect — составной виджет выбора «категория → подкатегория».
//
// Стеклянная фильтрация вынесена в TopLevelOptions/ChildOptions (тестируемые
// функции); сам виджет только связывает два выпадающих списка.
type CategorySelect struct {
	cats []*category.Category
	typ  core.EntryType

	parent *widget.Select
	child  *widget.Select

	topByLabel   map[string]string
	childByLabel map[string]string

	// Container — корневой контейнер для вставки в форму.
	Container *fyne.Container
}

// NewCategorySelect создаёт виджет выбора категории для заданного типа операции.
func NewCategorySelect(cats []*category.Category, typ core.EntryType) *CategorySelect {
	cs := &CategorySelect{cats: cats, typ: typ}

	cs.child = widget.NewSelect(nil, nil)
	cs.child.PlaceHolder = noneSubLabel

	cs.parent = widget.NewSelect(nil, func(string) { cs.reloadChildren() })
	cs.parent.PlaceHolder = "—"

	cs.reloadParents()
	cs.Container = container.NewVBox(cs.parent, cs.child)
	return cs
}

func (cs *CategorySelect) reloadParents() {
	opts := TopLevelOptions(cs.cats, cs.typ)
	cs.topByLabel = map[string]string{}
	labels := make([]string, 0, len(opts))
	for _, o := range opts {
		cs.topByLabel[o.Label] = o.ID
		labels = append(labels, o.Label)
	}
	cs.parent.Options = labels
	cs.parent.Refresh()
	cs.reloadChildren()
}

func (cs *CategorySelect) reloadChildren() {
	parentID := cs.parentID()
	cs.childByLabel = map[string]string{}
	labels := []string{noneSubLabel}
	for _, o := range ChildOptions(cs.cats, parentID) {
		cs.childByLabel[o.Label] = o.ID
		labels = append(labels, o.Label)
	}
	cs.child.Options = labels
	cs.child.SetSelected(noneSubLabel)
	cs.child.Refresh()
}

func (cs *CategorySelect) parentID() string {
	if cs.parent.Selected == "" {
		return ""
	}
	return cs.topByLabel[cs.parent.Selected]
}

// SelectedID возвращает выбранную подкатегорию, иначе категорию верхнего уровня,
// иначе пустую строку, если ничего не выбрано.
func (cs *CategorySelect) SelectedID() string {
	if cs.child.Selected != "" && cs.child.Selected != noneSubLabel {
		if id, ok := cs.childByLabel[cs.child.Selected]; ok {
			return id
		}
	}
	return cs.parentID()
}

// SetByID предустанавливает выбор по идентификатору категории/подкатегории
// (для формы редактирования).
func (cs *CategorySelect) SetByID(id string) {
	target := categoryByID(cs.cats, id)
	if target == nil {
		return
	}
	if target.IsTopLevel() {
		cs.parent.SetSelected(target.Name)
		return
	}
	if parent := categoryByID(cs.cats, target.ParentID); parent != nil {
		cs.parent.SetSelected(parent.Name)
		cs.reloadChildren()
		cs.child.SetSelected(target.Name)
	}
}

func categoryByID(cats []*category.Category, id string) *category.Category {
	for _, c := range cats {
		if c.Meta.ID == id {
			return c
		}
	}
	return nil
}
