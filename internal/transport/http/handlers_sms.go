package http

import (
	"net/http"

	smssvc "advisor/internal/application/sms"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// --- DTO ---

type smsTemplateDTO struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Sender            string `json:"sender"`
	Pattern           string `json:"pattern"`
	AmountGroup       int    `json:"amountGroup"`
	CurrencyGroup     int    `json:"currencyGroup"`
	MerchantGroup     int    `json:"merchantGroup"`
	CaptureKind       string `json:"captureKind"` // "merchant" | "account"
	FixedCurrency     string `json:"fixedCurrency"`
	Type              string `json:"type"`
	DefaultCategoryID string `json:"defaultCategoryId"`
	Enabled           bool   `json:"enabled"`
	Priority          int    `json:"priority"`
}

func toSMSTemplateDTO(t *smssvc.Template) smsTemplateDTO {
	return smsTemplateDTO{
		ID: t.ID, Name: t.Name, Sender: t.Sender, Pattern: t.Pattern,
		AmountGroup: t.AmountGroup, CurrencyGroup: t.CurrencyGroup, MerchantGroup: t.MerchantGroup,
		CaptureKind: t.CaptureKind, FixedCurrency: t.FixedCurrency, Type: string(t.Type),
		DefaultCategoryID: t.DefaultCategoryID, Enabled: t.Enabled, Priority: t.Priority,
	}
}

func (d smsTemplateDTO) toTemplate() *smssvc.Template {
	return &smssvc.Template{
		Name: d.Name, Sender: d.Sender, Pattern: d.Pattern,
		AmountGroup: d.AmountGroup, CurrencyGroup: d.CurrencyGroup, MerchantGroup: d.MerchantGroup,
		CaptureKind: d.CaptureKind, FixedCurrency: d.FixedCurrency, Type: core.EntryType(d.Type),
		DefaultCategoryID: d.DefaultCategoryID, Enabled: d.Enabled, Priority: d.Priority,
	}
}

type draftDTO struct {
	ID         string    `json:"id"`
	RawSender  string    `json:"rawSender"`
	RawText    string    `json:"rawText"`
	ReceivedAt string    `json:"receivedAt"`
	Amount     *moneyDTO `json:"amount,omitempty"`
	Type       string    `json:"type,omitempty"`
	Merchant   string    `json:"merchant,omitempty"`
	TemplateID string    `json:"templateId,omitempty"`
	Resolved   bool      `json:"resolved"`
}

func toDraftDTO(d *smssvc.Draft) draftDTO {
	out := draftDTO{
		ID: d.ID, RawSender: d.RawSender, RawText: d.RawText,
		ReceivedAt: d.ReceivedAt.String(), Type: string(d.ParsedType),
		Merchant: d.Merchant, TemplateID: d.TemplateID, Resolved: d.Resolved,
	}
	if d.ParsedAmount != nil {
		m := toMoney(*d.ParsedAmount)
		out.Amount = &m
	}
	return out
}

// --- Приём SMS от Android-форвардера ---

type ingestSMSReq struct {
	Sender     string `json:"sender"`
	Text       string `json:"text"`
	ReceivedAt string `json:"receivedAt"` // YYYY-MM-DD; пусто => сегодня
}

func (s *Server) handleIngestSMS(w http.ResponseWriter, r *http.Request) {
	var req ingestSMSReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	date := core.DateOf(s.g.Clock.Now())
	if req.ReceivedAt != "" {
		d, err := core.ParseDate(req.ReceivedAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		date = d
	}
	out, err := s.user(r).SMS.Ingest(req.Sender, req.Text, date)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"matched":       out.Matched,
		"transactionId": out.TransactionID,
		"draftId":       out.DraftID,
	})
}

// --- Шаблоны ---

func (s *Server) handleListSMSTemplates(w http.ResponseWriter, r *http.Request) {
	tmpls, err := s.user(r).SMS.ListTemplates()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]smsTemplateDTO, 0, len(tmpls))
	for _, t := range tmpls {
		out = append(out, toSMSTemplateDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateSMSTemplate(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeErr(w, http.StatusForbidden, "шаблоны разбора — общие; менять может только админ")
		return
	}
	var req smsTemplateDTO
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.user(r).SMS.CreateTemplate(req.toTemplate())
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toSMSTemplateDTO(t))
}

func (s *Server) handleUpdateSMSTemplate(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeErr(w, http.StatusForbidden, "шаблоны разбора — общие; менять может только админ")
		return
	}
	var req smsTemplateDTO
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.user(r).SMS.UpdateTemplate(r.PathValue("id"), req.toTemplate())
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toSMSTemplateDTO(t))
}

func (s *Server) handleDeleteSMSTemplate(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeErr(w, http.StatusForbidden, "шаблоны разбора — общие; менять может только админ")
		return
	}
	if err := s.user(r).SMS.DeleteTemplate(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// fromSampleReq — сборка шаблона «по образцу» из выделенных полей реального SMS.
type fromSampleReq struct {
	DraftID       string `json:"draftId"`
	Name          string `json:"name"`
	Sender        string `json:"sender"`
	Text          string `json:"text"`
	AmountText    string `json:"amountText"`
	CurrencyText  string `json:"currencyText"`
	FixedCurrency string `json:"fixedCurrency"`
	MerchantText  string `json:"merchantText"`
	CaptureKind   string `json:"captureKind"`
	Type          string `json:"type"`
	CategoryID    string `json:"categoryId"`
}

func (s *Server) handleTemplateFromSample(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeErr(w, http.StatusForbidden, "шаблоны разбора — общие; менять может только админ")
		return
	}
	var req fromSampleReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	spec := smssvc.SampleSpec{
		Name: req.Name, Sender: req.Sender, Text: req.Text,
		AmountText: req.AmountText, CurrencyText: req.CurrencyText, FixedCurrency: req.FixedCurrency,
		MerchantText: req.MerchantText, CaptureKind: req.CaptureKind, Type: core.EntryType(req.Type),
	}
	t, txID, err := s.user(r).SMS.CreateTemplateFromSample(spec, req.CategoryID, req.DraftID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": toSMSTemplateDTO(t), "transactionId": txID})
}

// --- Входящие черновики ---

func (s *Server) handleListDrafts(w http.ResponseWriter, r *http.Request) {
	unresolvedOnly := queryTrim(r, "unresolvedOnly") == "true"
	drafts, err := s.user(r).SMS.ListDrafts(unresolvedOnly)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]draftDTO, 0, len(drafts))
	for _, d := range drafts {
		out = append(out, toDraftDTO(d))
	}
	writeJSON(w, http.StatusOK, out)
}

type resolveDraftReq struct {
	CategoryID       string    `json:"categoryId"`
	Amount           *moneyDTO `json:"amount"`
	Type             string    `json:"type"`
	RememberMerchant bool      `json:"rememberMerchant"`
}

func (s *Server) handleResolveDraft(w http.ResponseWriter, r *http.Request) {
	var req resolveDraftReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	var amountOverride *money.Money
	if req.Amount != nil {
		m, err := req.Amount.parse()
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		amountOverride = &m
	}
	tx, err := s.user(r).SMS.ResolveDraft(r.PathValue("id"), req.CategoryID, amountOverride, core.EntryType(req.Type), req.RememberMerchant)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTransactionDTO(tx))
}

// --- Справочник контрагентов ---

type merchantDTO struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"` // "merchant" | "account"
	Label      string   `json:"label,omitempty"`
	SeenCount  int      `json:"seenCount"`
	Total      moneyDTO `json:"total"`
	LastSeen   string   `json:"lastSeen"`
	CategoryID string   `json:"categoryId,omitempty"`
}

func (s *Server) handleListMerchants(w http.ResponseWriter, r *http.Request) {
	list, err := s.user(r).SMS.ListMerchants()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]merchantDTO, 0, len(list))
	for _, m := range list {
		out = append(out, merchantDTO{
			Name: m.Name, Kind: m.Kind, Label: m.Label, SeenCount: m.SeenCount,
			Total: toMoney(m.Total), LastSeen: m.LastSeen, CategoryID: m.CategoryID,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type assignMerchantReq struct {
	Name       string `json:"name"`
	CategoryID string `json:"categoryId"` // "" => сбросить категорию
	Label      string `json:"label"`      // "" => убрать название
}

func (s *Server) handleAssignMerchant(w http.ResponseWriter, r *http.Request) {
	var req assignMerchantReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.user(r).SMS.AssignMerchant(req.Name, req.CategoryID, req.Label); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	if err := s.user(r).SMS.DeleteDraft(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
