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

func TestSMSCurrencyGuard(t *testing.T) {
	s := newTestServer(t)
	_, body := do(t, s, http.MethodPost, "/api/categories",
		createCategoryReq{Name: "Такси", Type: "expense"}, true)
	var cat categoryDTO
	mustJSON(t, body, &cat)

	// Кривой шаблон: группа валюты указывает на сумму (=1) — валюта захватит число.
	tmpl := smsTemplateDTO{
		Name: "Кривая валюта", Pattern: `Oplata ([0-9]+[.,][0-9]{2}) BYN`,
		AmountGroup: 1, CurrencyGroup: 1, Type: "expense",
		DefaultCategoryID: cat.ID, Enabled: true,
	}
	rec, body := do(t, s, http.MethodPost, "/api/sms/templates", tmpl, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: %d — %s", rec.Code, body)
	}

	do(t, s, http.MethodPost, "/api/ingest/sms", map[string]string{
		"sender": "Bank", "text": "Oplata 14.70 BYN. BLR YANDEX.GO", "receivedAt": "2026-07-21",
	}, true)
	_, body = do(t, s, http.MethodGet, "/api/transactions?ym=2026-07", nil, true)
	var txs []transactionDTO
	mustJSON(t, body, &txs)
	if len(txs) != 1 {
		t.Fatalf("ожидалась 1 операция, got %d", len(txs))
	}
	// Валюта должна быть BYN (а не "14.70"), сумма — 14.70.
	if txs[0].Amount.Currency != "BYN" || txs[0].Amount.Amount != "14.70" {
		t.Fatalf("ожидалось 14.70 BYN, got %s %s", txs[0].Amount.Amount, txs[0].Amount.Currency)
	}
}

func TestSMSMerchantRule(t *testing.T) {
	s := newTestServer(t)
	_, body := do(t, s, http.MethodPost, "/api/categories",
		createCategoryReq{Name: "Такси", Type: "expense"}, true)
	var taxi categoryDTO
	mustJSON(t, body, &taxi)

	// Правило: контрагент содержит "YANDEX" → категория Такси.
	rec, body := do(t, s, http.MethodPost, "/api/sms/rules",
		ruleDTO{Pattern: "YANDEX", CategoryID: taxi.ID}, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create rule: %d — %s", rec.Code, body)
	}

	// Шаблон БЕЗ категории, но с захватом контрагента.
	tmpl := smsTemplateDTO{
		Name: "Оплата", Pattern: `Oplata ([0-9]+[.,][0-9]{2}) ([A-Z]{3})\. BLR (.+?)\. Balance`,
		AmountGroup: 1, CurrencyGroup: 2, MerchantGroup: 3, Type: "expense", Enabled: true,
	}
	rec, body = do(t, s, http.MethodPost, "/api/sms/templates", tmpl, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: %d — %s", rec.Code, body)
	}

	// Приходит SMS от Yandex Go — правило должно авто-разнести в Такси.
	rec, body = do(t, s, http.MethodPost, "/api/ingest/sms", map[string]string{
		"sender": "Bank", "text": "Oplata 14.70 BYN. BLR YANDEX.GO. Balance: 19.26 BYN", "receivedAt": "2026-07-21",
	}, true)
	var out map[string]any
	mustJSON(t, body, &out)
	if out["matched"] != true || out["transactionId"] == "" {
		t.Fatalf("ожидалась авто-операция по правилу, got %v", out)
	}
	_, body = do(t, s, http.MethodGet, "/api/transactions?ym=2026-07", nil, true)
	var txs []transactionDTO
	mustJSON(t, body, &txs)
	if len(txs) != 1 || txs[0].CategoryID != taxi.ID || txs[0].Amount.Amount != "14.70" {
		t.Fatalf("ожидалась операция 14.70 в Такси, got %+v", txs)
	}
}

func TestSMSMerchantDirectory(t *testing.T) {
	s := newTestServer(t)

	// Шаблон с захватом контрагента, но БЕЗ категории → операции идут во «входящие»,
	// а контрагенты копятся в справочнике.
	tmpl := smsTemplateDTO{
		Name: "Оплата", Pattern: `Oplata ([0-9]+[.,][0-9]{2}) ([A-Z]{3})\. BLR (.+?)\. Balance`,
		AmountGroup: 1, CurrencyGroup: 2, MerchantGroup: 3, Type: "expense", Enabled: true,
	}
	rec, body := do(t, s, http.MethodPost, "/api/sms/templates", tmpl, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: %d — %s", rec.Code, body)
	}

	// Две SMS от одного контрагента (14.70 + 5.30) и одна от другого.
	for _, txt := range []string{
		"Oplata 14.70 BYN. BLR YANDEX.GO. Balance: 19.26 BYN",
		"Oplata 5.30 BYN. BLR YANDEX.GO. Balance: 13.96 BYN",
		"Oplata 42.00 BYN. BLR EUROPT. Balance: 100.00 BYN",
	} {
		do(t, s, http.MethodPost, "/api/ingest/sms", map[string]string{
			"sender": "Priorbank", "text": txt, "receivedAt": "2026-07-20",
		}, true)
	}

	_, body = do(t, s, http.MethodGet, "/api/sms/merchants", nil, true)
	var merchants []merchantDTO
	mustJSON(t, body, &merchants)
	if len(merchants) != 2 {
		t.Fatalf("ожидалось 2 контрагента, got %d: %+v", len(merchants), merchants)
	}
	// Первым — самый частый (YANDEX.GO: 2 раза, оборот 20.00).
	top := merchants[0]
	if top.Name != "YANDEX.GO" || top.SeenCount != 2 || top.Total.Amount != "20.00" {
		t.Fatalf("ожидался YANDEX.GO ×2 на 20.00, got %+v", top)
	}
}

func TestSMSMerchantCapture(t *testing.T) {
	s := newTestServer(t)
	_, body := do(t, s, http.MethodPost, "/api/categories",
		createCategoryReq{Name: "Топливо", Type: "expense"}, true)
	var cat categoryDTO
	mustJSON(t, body, &cat)

	// Шаблон захватывает сумму(1), валюту(2) и контрагента(3).
	tmpl := smsTemplateDTO{
		Name:              "Оплата с продавцом",
		Pattern:           `Oplata ([0-9]+[.,][0-9]{2}) ([A-Z]{3})\. BLR (.+?)\. Balance`,
		AmountGroup:       1,
		CurrencyGroup:     2,
		MerchantGroup:     3,
		Type:              "expense",
		DefaultCategoryID: cat.ID,
		Enabled:           true,
	}
	rec, body := do(t, s, http.MethodPost, "/api/sms/templates", tmpl, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create template: %d — %s", rec.Code, body)
	}

	real := "Karta 4***2021 20-07-26 19:08:53. Oplata 14.90 BYN. BLR PRIRODNAYA ZAPRA. Balance: 18.51 BYN Tel. 7299090"

	// Тест-эндпоинт должен вернуть контрагента.
	_, body = do(t, s, http.MethodPost, "/api/sms/test",
		testSMSReq{Text: real}, true)
	var test map[string]any
	mustJSON(t, body, &test)
	if test["merchant"] != "PRIRODNAYA ZAPRA" {
		t.Fatalf("тест: ожидался контрагент PRIRODNAYA ZAPRA, got %v", test["merchant"])
	}

	// Ingest создаёт операцию, контрагент — в примечании.
	do(t, s, http.MethodPost, "/api/ingest/sms", map[string]string{
		"sender": "Priorbank", "text": real, "receivedAt": "2026-07-20",
	}, true)
	_, body = do(t, s, http.MethodGet, "/api/transactions?ym=2026-07", nil, true)
	var txs []transactionDTO
	mustJSON(t, body, &txs)
	if len(txs) != 1 || txs[0].Note != "PRIRODNAYA ZAPRA" || txs[0].Amount.Amount != "14.90" {
		t.Fatalf("ожидалась операция 14.90 с примечанием-продавцом, got %+v", txs)
	}
}
