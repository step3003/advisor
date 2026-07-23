package sms

import (
	"testing"

	"advisor/internal/domain/core"
)

func TestSynthesizeMerchantFormat(t *testing.T) {
	spec := SampleSpec{
		Name: "Приор", Text: "Oplata 14.70 BYN. BLR YANDEX.GO. Balance: 19.26 BYN",
		AmountText: "14.70", CurrencyText: "BYN", MerchantText: "YANDEX.GO",
		CaptureKind: KindMerchant, Type: core.Expense,
	}
	tmpl, err := SynthesizeTemplate(spec)
	if err != nil {
		t.Fatalf("synth: %v", err)
	}

	// На образце извлекает ровно выделенное.
	res, _ := parseWith([]*Template{tmpl}, "", spec.Text)
	if !res.Matched || res.Amount.Decimal() != "14.70" || res.Merchant != "YANDEX.GO" || res.Kind != KindMerchant {
		t.Fatalf("образец: got %+v (pattern %q)", res, tmpl.Pattern)
	}

	// Работает на другом сообщении того же формата (другой мерчант/сумма).
	res2, _ := parseWith([]*Template{tmpl}, "", "Oplata 5.00 BYN. BLR EUROPT. Balance: 1.00 BYN")
	if !res2.Matched || res2.Amount.Decimal() != "5.00" || res2.Merchant != "EUROPT" {
		t.Fatalf("другой мерчант: got %+v (pattern %q)", res2, tmpl.Pattern)
	}
}

func TestSynthesizeAccountFormat(t *testing.T) {
	spec := SampleSpec{
		Name: "ЕРИП", Text: "<#> 22/07 10:46. Platezh s DK6310, schet platezha 2162427167000030*140. Summa 40.46 BYN.",
		AmountText: "40.46", CurrencyText: "BYN", MerchantText: "2162427167000030*140",
		CaptureKind: KindAccount, Type: core.Expense,
	}
	tmpl, err := SynthesizeTemplate(spec)
	if err != nil {
		t.Fatalf("synth: %v", err)
	}

	res, _ := parseWith([]*Template{tmpl}, "", spec.Text)
	if !res.Matched || res.Amount.Decimal() != "40.46" || res.Merchant != "2162427167000030*140" || res.Kind != KindAccount {
		t.Fatalf("образец: got %+v (pattern %q)", res, tmpl.Pattern)
	}

	// Другое время в начале + другая сумма, тот же счёт — плавающий префикс не мешает.
	res2, _ := parseWith([]*Template{tmpl}, "", "<#> 23/07 09:00. Platezh s DK6310, schet platezha 2162427167000030*140. Summa 5.00 BYN.")
	if !res2.Matched || res2.Amount.Decimal() != "5.00" || res2.Merchant != "2162427167000030*140" {
		t.Fatalf("другое время: got %+v (pattern %q)", res2, tmpl.Pattern)
	}
}

func TestSynthesizeNameWithSpaces(t *testing.T) {
	// Имя-отправитель со словами не должно ломать сборку даже если помечено «Счёт».
	text := "Zachislenie perevoda 0.50 BYN. BLR ULYANA NARONSKAYA. Balance: 25.17 BYN Tel. 7299090"
	for _, kind := range []string{KindMerchant, KindAccount} {
		spec := SampleSpec{
			Name: "Зачисление", Text: text, AmountText: "0.50", CurrencyText: "BYN",
			MerchantText: "ULYANA NARONSKAYA", CaptureKind: kind, Type: core.Income,
		}
		tmpl, err := SynthesizeTemplate(spec)
		if err != nil {
			t.Fatalf("kind=%s: %v", kind, err)
		}
		res, _ := parseWith([]*Template{tmpl}, "", text)
		if !res.Matched || res.Amount.Decimal() != "0.50" || res.Merchant != "ULYANA NARONSKAYA" {
			t.Fatalf("kind=%s: got %+v (pattern %q)", kind, res, tmpl.Pattern)
		}
	}
}

func TestSynthesizeRejectsBadSelection(t *testing.T) {
	// Сумма, которой нет в тексте — ошибка, а не кривой шаблон.
	_, err := SynthesizeTemplate(SampleSpec{
		Name: "x", Text: "Oplata 14.70 BYN", AmountText: "99.99", Type: core.Expense,
	})
	if err == nil {
		t.Fatal("ожидалась ошибка на несуществующую сумму")
	}
}
