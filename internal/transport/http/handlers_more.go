package http

import (
	"net/http"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// --- Повторяющиеся ---

func (s *Server) handleListRecurring(w http.ResponseWriter, r *http.Request) {
	activeOnly := queryTrim(r, "activeOnly") == "true"
	ts, err := s.svc.Recurring.List(activeOnly)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toRecurringDTOs(ts))
}

type recurringReq struct {
	Type           string   `json:"type"`
	CategoryID     string   `json:"categoryId"`
	Amount         moneyDTO `json:"amount"`
	DayOfMonth     int      `json:"dayOfMonth"`
	StartDate      string   `json:"startDate"`
	EndDate        string   `json:"endDate"`
	AutoCreateFact bool     `json:"autoCreateFact"`
}

func (req recurringReq) parse() (typ core.EntryType, amt money.Money, start core.Date, end *core.Date, err error) {
	typ, err = parseEntryType(req.Type)
	if err != nil {
		return
	}
	amt, err = req.Amount.parse()
	if err != nil {
		return
	}
	start, err = core.ParseDate(req.StartDate)
	if err != nil {
		return
	}
	if req.EndDate != "" {
		var e core.Date
		e, err = core.ParseDate(req.EndDate)
		if err != nil {
			return
		}
		end = &e
	}
	return
}

func (s *Server) handleCreateRecurring(w http.ResponseWriter, r *http.Request) {
	var req recurringReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	typ, amt, start, end, err := req.parse()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.svc.Recurring.Create(typ, req.CategoryID, amt, req.DayOfMonth, start, end, req.AutoCreateFact)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toRecurringDTO(t))
}

func (s *Server) handleUpdateRecurring(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req recurringReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	typ, amt, start, end, err := req.parse()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.svc.Recurring.Update(id, typ, req.CategoryID, amt, req.DayOfMonth, start, end, req.AutoCreateFact)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toRecurringDTO(t))
}

func (s *Server) handlePauseRecurring(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Recurring.Pause(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleResumeRecurring(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Recurring.Resume(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteRecurring(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Recurring.Delete(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type generateReq struct {
	Period string `json:"period"`
}

func (s *Server) handleGenerateRecurring(w http.ResponseWriter, r *http.Request) {
	var req generateReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ym, err := parseYearMonth(req.Period)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	stats, err := s.svc.Recurring.GenerateForMonth(ym)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"plansCreated": stats.PlansCreated,
		"factsCreated": stats.FactsCreated,
	})
}

// --- Валюты ---

type refreshRatesReq struct {
	Date string `json:"date"`
}

func (s *Server) handleRefreshRates(w http.ResponseWriter, r *http.Request) {
	var req refreshRatesReq
	// Тело опционально: пустое => сегодняшняя дата.
	_ = readJSON(r, &req)
	date := core.DateOf(s.svc.Clock.Now())
	if req.Date != "" {
		d, err := core.ParseDate(req.Date)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		date = d
	}
	n, err := s.svc.Currency.RefreshRates(date)
	if err != nil {
		// Сеть может отсутствовать — приложение работает на кэше (FR-CUR-5).
		writeErr(w, http.StatusBadGateway, "не удалось обновить курсы: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"updated": n})
}

type currencyDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (s *Server) handleListCurrencies(w http.ResponseWriter, _ *http.Request) {
	infos, err := s.svc.Settings.ListCurrencies()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]currencyDTO, 0, len(infos))
	for _, ci := range infos {
		out = append(out, currencyDTO{Code: ci.Code.String(), Name: ci.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Настройки ---

type settingsDTO struct {
	DefaultCurrency string        `json:"defaultCurrency"`
	Currencies      []currencyDTO `json:"currencies"`
}

func (s *Server) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	cur, err := s.svc.Settings.DefaultCurrency()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	infos, err := s.svc.Settings.ListCurrencies()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := settingsDTO{DefaultCurrency: cur.String()}
	for _, ci := range infos {
		out.Currencies = append(out.Currencies, currencyDTO{Code: ci.Code.String(), Name: ci.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

type patchSettingsReq struct {
	DefaultCurrency *string `json:"defaultCurrency"`
}

func (s *Server) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var req patchSettingsReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.DefaultCurrency != nil {
		if err := s.svc.Settings.SetDefaultCurrency(money.Currency(*req.DefaultCurrency)); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	s.handleGetSettings(w, r)
}
