package http

import (
	"net/http"
	"testing"
)

func TestSMSParsingFlow(t *testing.T) {
	s := newTestServer(t)

	// 1. Категория для авто-разнесения.
	_, body := do(t, s, http.MethodPost, "/api/categories",
		createCategoryReq{Name: "Магазины", Type: "expense"}, true)
	var cat categoryDTO
	mustJSON(t, body, &cat)

	// 2. Шаблон разбора SMS: «Oplata 45.20 BYN ...» → расход в категорию.
	tmpl := smsTemplateDTO{
		Name:              "Приорбанк расход",
		Sender:            "Priorbank",
		Pattern:           `Oplata (\d+[.,]\d{2}) BYN`,
		AmountGroup:       1,
		CurrencyGroup:     0,
		FixedCurrency:     "BYN",
		Type:              "expense",
		DefaultCategoryID: cat.ID,
		Enabled:           true,
	}
	rec, body := do(t, s, http.MethodPost, "/api/sms/templates", tmpl, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: %d — %s", rec.Code, body)
	}

	// 3. Тест-эндпоинт: разбор без сохранения.
	rec, body = do(t, s, http.MethodPost, "/api/sms/test",
		testSMSReq{Sender: "Priorbank", Text: "Oplata 45,20 BYN v EUROOPT"}, true)
	var test map[string]any
	mustJSON(t, body, &test)
	if test["matched"] != true {
		t.Fatalf("test: ожидался matched=true, got %v", test)
	}

	// 4. Приём SMS от форвардера → создаётся операция.
	rec, body = do(t, s, http.MethodPost, "/api/ingest/sms", map[string]string{
		"sender": "Priorbank", "text": "Oplata 45,20 BYN v EUROOPT", "receivedAt": "2026-07-15",
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingest: %d — %s", rec.Code, body)
	}
	var out map[string]any
	mustJSON(t, body, &out)
	if out["matched"] != true || out["transactionId"] == "" {
		t.Fatalf("ingest: ожидалась операция, got %v", out)
	}

	// 5. Операция появилась в списке за месяц с суммой 45.20.
	_, body = do(t, s, http.MethodGet, "/api/transactions?ym=2026-07", nil, true)
	var txs []transactionDTO
	mustJSON(t, body, &txs)
	if len(txs) != 1 || txs[0].Amount.Amount != "45.20" {
		t.Fatalf("ожидалась 1 операция 45.20, got %+v", txs)
	}

	// 6. Нераспознанное SMS → во «входящие».
	_, body = do(t, s, http.MethodPost, "/api/ingest/sms", map[string]string{
		"sender": "Unknown", "text": "какой-то текст", "receivedAt": "2026-07-16",
	}, true)
	var out2 map[string]any
	mustJSON(t, body, &out2)
	if out2["matched"] != false || out2["draftId"] == "" {
		t.Fatalf("нераспознанное: ожидался черновик, got %v", out2)
	}

	// 7. Входящие содержат один нерешённый черновик.
	_, body = do(t, s, http.MethodGet, "/api/inbox?unresolvedOnly=true", nil, true)
	var drafts []draftDTO
	mustJSON(t, body, &drafts)
	if len(drafts) != 1 {
		t.Fatalf("ожидался 1 черновик, got %d", len(drafts))
	}

	// 8. Разобрать черновик вручную: указать категорию, сумму и тип.
	rec, body = do(t, s, http.MethodPost, "/api/inbox/"+drafts[0].ID+"/resolve",
		resolveDraftReq{CategoryID: cat.ID, Amount: &moneyDTO{Amount: "10.00", Currency: "BYN"}, Type: "expense"}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve draft: %d — %s", rec.Code, body)
	}
}
