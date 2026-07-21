// Package app собирает окно приложения и навигацию поверх usecase-сервисов.
// Зависит только от presentation/screens (и через него — от application/domain);
// к инфраструктуре не обращается.
package app

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"advisor/internal/presentation/i18n"
	"advisor/internal/presentation/screens"
)

// Run создаёт окно, вкладки экранов и запускает цикл событий Fyne (блокирует).
func Run(d *screens.Deps) {
	a := fyneapp.NewWithID("com.advisor.finance")
	w := a.NewWindow(i18n.AppTitle)
	d.Window = w // диалоги экранов рисуются в этом окне

	all := []screens.Screen{
		screens.NewBalanceScreen(d),
		screens.NewTxScreen(d),
		screens.NewReportsScreen(d),
		screens.NewCategoriesScreen(d),
		screens.NewRecurringScreen(d),
		screens.NewSettingsScreen(d),
	}

	tabs := container.NewAppTabs()
	for _, sc := range all {
		tabs.Append(container.NewTabItemWithIcon(sc.Title(), sc.Icon(), sc.Build()))
	}
	tabs.SetTabLocation(container.TabLocationLeading)
	// При переключении вкладки обновляем её данные (данные могли измениться
	// на других экранах).
	tabs.OnSelected = func(item *container.TabItem) {
		idx := tabs.SelectedIndex()
		if idx >= 0 && idx < len(all) {
			all[idx].Refresh()
		}
	}

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(900, 640))
	w.ShowAndRun()
}
