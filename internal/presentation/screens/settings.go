package screens

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iosvc "advisor/internal/application/io"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/presentation/i18n"
)

// SettingsScreen — настройки: валюта, курсы, экспорт/импорт, синхронизация (FR-SET).
type SettingsScreen struct {
	d       *Deps
	syncBox *fyne.Container
	sync    SyncInfo
	curSel  *widget.Select
	root    fyne.CanvasObject
}

// NewSettingsScreen создаёт экран настроек.
func NewSettingsScreen(d *Deps) *SettingsScreen {
	return &SettingsScreen{d: d, sync: d.Sync}
}

func (s *SettingsScreen) Title() string       { return i18n.NavSettings }
func (s *SettingsScreen) Icon() fyne.Resource { return theme.SettingsIcon() }

func (s *SettingsScreen) Build() fyne.CanvasObject {
	s.curSel = newCurrencySelect(s.d, defaultCurrency(s.d))
	saveCur := widget.NewButtonWithIcon(i18n.BtnSave, theme.ConfirmIcon(), func() {
		if err := s.d.Settings.SetDefaultCurrency(money.Currency(s.curSel.Selected)); showError(s.d.Window, err) {
			return
		}
		showInfo(s.d.Window, i18n.MsgDone, i18n.SetCurrencySaved)
	})
	currencyGroup := widget.NewCard(i18n.SetDefaultCurrency, "", container.NewHBox(s.curSel, saveCur))

	ratesBtn := widget.NewButtonWithIcon(i18n.SetUpdateRates, theme.ViewRefreshIcon(), s.updateRates)
	ratesGroup := widget.NewCard(i18n.SetRates, "", ratesBtn)

	exportJSON := widget.NewButtonWithIcon(i18n.SetExportJSON, theme.DocumentSaveIcon(), s.exportJSON)
	importJSON := widget.NewButtonWithIcon(i18n.SetImportJSON, theme.FolderOpenIcon(), s.importJSON)
	exportCSV := widget.NewButtonWithIcon(i18n.SetExportCSV, theme.DocumentSaveIcon(), s.exportCSV)
	ioGroup := widget.NewCard(i18n.SetExportImport, "", container.NewVBox(exportJSON, importJSON, exportCSV))

	s.syncBox = container.NewVBox()
	rescanBtn := widget.NewButtonWithIcon(i18n.SetRescan, theme.ViewRefreshIcon(), s.rescan)
	syncGroup := widget.NewCard(i18n.SetSync, "", container.NewVBox(s.syncBox, rescanBtn))

	s.renderSync()
	s.root = container.NewVScroll(container.NewVBox(currencyGroup, ratesGroup, ioGroup, syncGroup))
	return s.root
}

func (s *SettingsScreen) Refresh() { s.renderSync() }

func (s *SettingsScreen) renderSync() {
	if s.syncBox == nil {
		return
	}
	s.syncBox.Objects = nil
	s.syncBox.Add(totalLine(i18n.SetVaultPath, s.sync.VaultPath))
	s.syncBox.Add(totalLine(i18n.SetSyncLastRebuild, s.sync.LastRun))
	counts := strconv.Itoa(s.sync.Categories+s.sync.Transactions+s.sync.Plans+s.sync.Recurring) +
		" (кат " + strconv.Itoa(s.sync.Categories) + ", оп " + strconv.Itoa(s.sync.Transactions) +
		", план " + strconv.Itoa(s.sync.Plans) + ", повт " + strconv.Itoa(s.sync.Recurring) + ")"
	s.syncBox.Add(totalLine(i18n.SetSyncCounts, counts))
	s.syncBox.Add(totalLine(i18n.SetSyncConflicts, strconv.Itoa(s.sync.Conflicts)))
	s.syncBox.Add(totalLine(i18n.SetSyncErrors, strconv.Itoa(s.sync.Errors)))
	s.syncBox.Refresh()
}

func (s *SettingsScreen) updateRates() {
	_, err := s.d.Currency.RefreshRates(s.d.Today())
	if err != nil {
		// Сеть может отсутствовать — приложение работает на кэше (FR-CUR-5).
		showInfo(s.d.Window, i18n.MsgError, i18n.SetRatesOffline)
		return
	}
	showInfo(s.d.Window, i18n.MsgDone, i18n.SetRatesUpdated)
}

func (s *SettingsScreen) rescan() {
	if s.d.Rescan == nil {
		return
	}
	info, err := s.d.Rescan()
	if showError(s.d.Window, err) {
		return
	}
	s.sync = info
	s.renderSync()
	showInfo(s.d.Window, i18n.MsgDone, i18n.SetRescanDone)
}

func (s *SettingsScreen) exportJSON() {
	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		defer w.Close()
		if showError(s.d.Window, s.d.IO.Export(w)) {
			return
		}
		showInfo(s.d.Window, i18n.MsgDone, i18n.SetExportDone)
	}, s.d.Window)
}

func (s *SettingsScreen) importJSON() {
	modeSel := widget.NewSelect([]string{i18n.SetImportMerge, i18n.SetImportReplace}, nil)
	modeSel.SetSelected(i18n.SetImportMerge)
	dialog.ShowForm(i18n.SetImportMode, i18n.BtnApply, i18n.BtnCancel,
		[]*widget.FormItem{{Text: i18n.SetImportMode, Widget: modeSel}},
		func(ok bool) {
			if !ok {
				return
			}
			mode := iosvc.ModeMerge
			if modeSel.Selected == i18n.SetImportReplace {
				mode = iosvc.ModeReplace
			}
			dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
				if err != nil || r == nil {
					return
				}
				defer r.Close()
				if showError(s.d.Window, s.d.IO.Import(r, mode)) {
					return
				}
				showInfo(s.d.Window, i18n.MsgDone, i18n.SetImportDone)
			}, s.d.Window)
		}, s.d.Window)
}

func (s *SettingsScreen) exportCSV() {
	fromEntry := widget.NewEntry()
	toEntry := widget.NewEntry()
	ym := s.d.CurrentMonth()
	fromEntry.SetText(ym.FirstDay().String())
	toEntry.SetText(lastDay(ym).String())
	items := []*widget.FormItem{
		{Text: i18n.ReportFrom, Widget: fromEntry},
		{Text: i18n.ReportTo, Widget: toEntry},
	}
	dialog.ShowForm(i18n.SetPeriodForCSV, i18n.BtnApply, i18n.BtnCancel, items, func(ok bool) {
		if !ok {
			return
		}
		from, err := core.ParseDate(fromEntry.Text)
		if err != nil {
			showError(s.d.Window, errText(i18n.ReportBadDates))
			return
		}
		to, err := core.ParseDate(toEntry.Text)
		if err != nil || to.Before(from) {
			showError(s.d.Window, errText(i18n.ReportBadDates))
			return
		}
		dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
			if err != nil || w == nil {
				return
			}
			defer w.Close()
			if showError(s.d.Window, s.d.IO.ExportTransactionsCSV(w, from, to)) {
				return
			}
			showInfo(s.d.Window, i18n.MsgDone, i18n.SetExportDone)
		}, s.d.Window)
	}, s.d.Window)
}
