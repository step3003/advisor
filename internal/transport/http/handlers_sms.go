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
	FixedCurrency     string `json:"fixedCurrency"`
	Type              string `json:"type"`
	DefaultCategoryID string `json:"defaultCategoryId"`
	Enabled           bool   `json:"enabled"`
	Priority          int    `json:"priority"`
}

func toSMSTemplateDTO(t *smssvc.Template) smsTemplateDTO {
	return smsTemplateDTO{
		ID: t.ID, Name: t.Name, Sender: t.Sender, Pattern: t.Pattern,
		AmountGroup: t.AmountGroup, CurrencyGroup: t.CurrencyGroup, FixedCurrency: t.FixedCurrency,
		Type: string(t.Type), DefaultCategoryID: t.DefaultCategoryID, Enabled: t.Enabled, Priority: t.Priority,
	}
}

func (d smsTemplateDTO) toTemplate() *smssvc.Template {
	return &smssvc.Template{
		Name: d.Name, Sender: d.Sender, Pattern: d.Pattern,
		AmountGroup: d.AmountGroup, CurrencyGroup: d.CurrencyGroup, FixedCurrency: d.FixedCurrency,
		Type: core.EntryType(d.Type), DefaultCategoryID: d.DefaultCategoryID,
		Enabled: d.Enabled, Priority: d.Priority,
	}
}

type draftDTO struct {
	ID         string    `json:"id"`
	RawSender  string    `json:"rawSender"`
	RawText    string    `json:"rawText"`
	ReceivedAt string    `json:"receivedAt"`
	Amount     *moneyDTO `json:"amount,omitempty"`
	Type       string    `json:"type,omitempty"`
	TemplateID string    `json:"templateId,omitempty"`
	Resolved   bool      `json:"resolved"`
}

func toDraftDTO(d *smssvc.Draft) draftDTO {
	out := draftDTO{
		ID: d.ID, RawSender: d.RawSender, RawText: d.RawText,
		ReceivedAt: d.ReceivedAt.String(), Type: string(d.ParsedType),
		TemplateID: d.TemplateID, Resolved: d.Resolved,
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
	date := core.DateOf(s.svc.Clock.Now())
	if req.ReceivedAt != "" {
		d, err := core.ParseDate(req.ReceivedAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		date = d
	}
	out, err := s.svc.SMS.Ingest(req.Sender, req.Text, date)
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

func (s *Server) handleListSMSTemplates(w http.ResponseWriter, _ *http.Request) {
	tmpls, err := s.svc.SMS.ListTemplates()
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
	var req smsTemplateDTO
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.svc.SMS.CreateTemplate(req.toTemplate())
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toSMSTemplateDTO(t))
}

func (s *Server) handleUpdateSMSTemplate(w http.ResponseWriter, r *http.Request) {
	var req smsTemplateDTO
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.svc.SMS.UpdateTemplate(r.PathValue("id"), req.toTemplate())
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toSMSTemplateDTO(t))
}

func (s *Server) handleDeleteSMSTemplate(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.SMS.DeleteTemplate(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type testSMSReq struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
}

func (s *Server) handleTestSMS(w http.ResponseWriter, r *http.Request) {
	var req testSMSReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := s.svc.SMS.Test(req.Sender, req.Text)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{"matched": res.Matched}
	if res.Matched {
		out["templateName"] = res.TemplateName
		out["amount"] = toMoney(res.Amount)
		out["type"] = string(res.Type)
		out["defaultCategoryId"] = res.DefaultCategoryID
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Входящие черновики ---

func (s *Server) handleListDrafts(w http.ResponseWriter, r *http.Request) {
	unresolvedOnly := queryTrim(r, "unresolvedOnly") == "true"
	drafts, err := s.svc.SMS.ListDrafts(unresolvedOnly)
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
	CategoryID string    `json:"categoryId"`
	Amount     *moneyDTO `json:"amount"`
	Type       string    `json:"type"`
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
	tx, err := s.svc.SMS.ResolveDraft(r.PathValue("id"), req.CategoryID, amountOverride, core.EntryType(req.Type))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTransactionDTO(tx))
}

func (s *Server) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.SMS.DeleteDraft(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
