package http

import (
	"net/http"

	iosvc "advisor/internal/application/io"
	"advisor/internal/domain/category"
	"advisor/internal/domain/core"
)

// --- Категории ---

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	includeArchived := queryTrim(r, "includeArchived") == "true"
	cats, err := s.user(r).Catalog.List(includeArchived)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toCategoryDTOs(cats))
}

type createCategoryReq struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID string `json:"parentId"`
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req createCategoryReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	typ, err := parseEntryType(req.Type)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var c *category.Category
	if req.ParentID == "" {
		c, err = s.user(r).Catalog.Create(req.Name, typ)
	} else {
		c, err = s.user(r).Catalog.CreateSub(req.Name, typ, req.ParentID)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toCategoryDTO(c))
}

type patchCategoryReq struct {
	Name     *string `json:"name"`
	Archived *bool   `json:"archived"`
}

func (s *Server) handlePatchCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req patchCategoryReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		if err := s.user(r).Catalog.Rename(id, *req.Name); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.Archived != nil {
		var err error
		if *req.Archived {
			err = s.user(r).Catalog.Archive(id)
		} else {
			err = s.user(r).Catalog.Unarchive(id)
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	c, err := s.findCategory(r, id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "категория не найдена")
		return
	}
	writeJSON(w, http.StatusOK, toCategoryDTO(c))
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.user(r).Catalog.Delete(id); err != nil {
		// Удаление заблокировано при наличии операций/планов (FR-CAT-4).
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) findCategory(r *http.Request, id string) (*category.Category, error) {
	cats, err := s.user(r).Catalog.List(true)
	if err != nil {
		return nil, err
	}
	for _, c := range cats {
		if c.Meta.ID == id {
			return c, nil
		}
	}
	return nil, ports_ErrNotFound
}

// --- Планы ---

func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	ym, err := parseYearMonth(queryTrim(r, "ym"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := s.user(r).Planning.ListMonth(ym)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPlanItemDTOs(items))
}

type setPlanReq struct {
	Period     string   `json:"period"`
	CategoryID string   `json:"categoryId"`
	Amount     moneyDTO `json:"amount"`
	Note       string   `json:"note"`
}

func (s *Server) handleSetPlan(w http.ResponseWriter, r *http.Request) {
	var req setPlanReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ym, err := parseYearMonth(req.Period)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	amt, err := req.Amount.parse()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := s.user(r).Planning.SetPlan(ym, req.CategoryID, amt, req.Note)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPlanItemDTO(p))
}

type copyPreviousReq struct {
	Period string `json:"period"`
}

func (s *Server) handleCopyPrevious(w http.ResponseWriter, r *http.Request) {
	var req copyPreviousReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ym, err := parseYearMonth(req.Period)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	n, err := s.user(r).Planning.CopyFromPreviousMonth(ym)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"copied": n})
}

// --- Операции ---

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	ym, err := parseYearMonth(queryTrim(r, "ym"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	txs, err := s.user(r).Ledger.ListMonth(ym)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTransactionDTOs(txs))
}

type txReq struct {
	Date       string   `json:"date"`
	Type       string   `json:"type"`
	CategoryID string   `json:"categoryId"`
	Amount     moneyDTO `json:"amount"`
	Note       string   `json:"note"`
}

func (s *Server) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req txReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	typ, err := parseEntryType(req.Type)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	date, err := core.ParseDate(req.Date)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	amt, err := req.Amount.parse()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.user(r).Ledger.Add(typ, date, req.CategoryID, amt, req.Note)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toTransactionDTO(t))
}

func (s *Server) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req txReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	date, err := core.ParseDate(req.Date)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	amt, err := req.Amount.parse()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.user(r).Ledger.Edit(id, date, req.CategoryID, amt, req.Note)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTransactionDTO(t))
}

func (s *Server) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	if err := s.user(r).Ledger.Delete(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Отчёты ---

func (s *Server) handlePlanVsFact(w http.ResponseWriter, r *http.Request) {
	ym, err := parseYearMonth(queryTrim(r, "ym"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	pvf, err := s.user(r).Planning.PlanVsFact(ym)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPlanVsFactDTO(pvf))
}

func (s *Server) handlePeriodReport(w http.ResponseWriter, r *http.Request) {
	from, err := core.ParseDate(queryTrim(r, "from"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "параметр from: "+err.Error())
		return
	}
	to, err := core.ParseDate(queryTrim(r, "to"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "параметр to: "+err.Error())
		return
	}
	sum, err := s.user(r).Reporting.PeriodSummary(from, to)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPeriodSummaryDTO(sum))
}

func (s *Server) handleDynamics(w http.ResponseWriter, r *http.Request) {
	from, err := parseYearMonth(queryTrim(r, "from"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "параметр from: "+err.Error())
		return
	}
	to, err := parseYearMonth(queryTrim(r, "to"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "параметр to: "+err.Error())
		return
	}
	points, err := s.user(r).Reporting.MonthlyDynamics(from, to)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toMonthPointDTOs(points))
}

// --- Экспорт ---

func (s *Server) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="advisor-export.json"`)
	if err := s.user(r).IO.Export(w); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	from, err := core.ParseDate(queryTrim(r, "from"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "параметр from: "+err.Error())
		return
	}
	to, err := core.ParseDate(queryTrim(r, "to"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "параметр to: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="advisor-transactions.csv"`)
	if err := s.user(r).IO.ExportTransactionsCSV(w, from, to); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}

// parseEntryType валидирует тип операции.
func parseEntryType(s string) (core.EntryType, error) {
	t := core.EntryType(s)
	if !t.Valid() {
		return "", errBadType
	}
	return t, nil
}

var (
	errBadType        = &apiError{"тип операции должен быть expense или income"}
	ports_ErrNotFound = &apiError{"не найдено"}
)

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }

// ensure iosvc import used (Export/ExportTransactionsCSV via s.user(r).IO)
var _ = iosvc.ModeMerge
