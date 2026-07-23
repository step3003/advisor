package sms

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// SampleSpec — задание на сборку шаблона «по образцу»: реальный текст SMS и
// выделенные пользователем значения полей (как подстроки этого текста).
type SampleSpec struct {
	Name              string
	Sender            string
	Text              string
	Action            string // "operation" | "discard"
	SignatureText     string // для discard: слово/фраза-признак мусора
	AmountText        string // выделенная сумма (обязательно для operation)
	CurrencyText      string // выделенная валюта ("" => FixedCurrency)
	FixedCurrency     string
	MerchantText      string // выделенный признак ("" => не захватывать)
	CaptureKind       string // merchant|account (если MerchantText задан)
	Type              core.EntryType
	DefaultCategoryID string
}

type synthField struct {
	start, end int
	kind       string // "amount" | "currency" | "merchant"
}

var accountValueRe = regexp.MustCompile(`^[0-9*]+$`)

// SynthesizeTemplate собирает regex-шаблон по образцу: якорится на литеральном
// тексте вокруг выделенных полей, обобщает сами значения. Проверяет, что
// собранный шаблон извлекает из образца ровно выделенные значения.
func SynthesizeTemplate(spec SampleSpec) (*Template, error) {
	s := spec.Text
	if strings.TrimSpace(spec.Name) == "" {
		return nil, fmt.Errorf("sms: задайте название типа")
	}

	// Мусор: паттерн — литеральная фраза-признак; извлекать нечего.
	if normalizeAction(spec.Action) == ActionDiscard {
		sig := strings.TrimSpace(spec.SignatureText)
		if sig == "" {
			return nil, fmt.Errorf("sms: отметьте слово-признак мусора")
		}
		if !strings.Contains(s, sig) {
			return nil, fmt.Errorf("sms: признак не найден в тексте")
		}
		return &Template{
			Name: strings.TrimSpace(spec.Name), Sender: strings.TrimSpace(spec.Sender),
			Pattern: regexp.QuoteMeta(sig), Action: ActionDiscard, Enabled: true,
			Type: core.Expense,
		}, nil
	}

	amountText := strings.TrimSpace(spec.AmountText)
	if amountText == "" {
		return nil, fmt.Errorf("sms: выделите сумму в сообщении")
	}
	aStart := strings.Index(s, amountText)
	if aStart < 0 {
		return nil, fmt.Errorf("sms: выделенная сумма не найдена в тексте")
	}
	fields := []synthField{{aStart, aStart + len(amountText), "amount"}}

	currencyText := strings.TrimSpace(spec.CurrencyText)
	if currencyText != "" {
		// Валюту ищем после суммы — так снимается неоднозначность (напр. два "BYN").
		from := aStart + len(amountText)
		idx := strings.Index(s[from:], currencyText)
		if idx < 0 {
			from, idx = 0, strings.Index(s, currencyText)
		}
		if idx < 0 {
			return nil, fmt.Errorf("sms: выделенная валюта не найдена в тексте")
		}
		cs := from + idx
		fields = append(fields, synthField{cs, cs + len(currencyText), "currency"})
	}

	merchantText := strings.TrimSpace(spec.MerchantText)
	if merchantText != "" {
		idx := strings.Index(s, merchantText)
		if idx < 0 {
			return nil, fmt.Errorf("sms: выделенный признак не найден в тексте")
		}
		fields = append(fields, synthField{idx, idx + len(merchantText), "merchant"})
	}

	sort.Slice(fields, func(i, j int) bool { return fields[i].start < fields[j].start })
	if overlaps(fields) {
		return nil, fmt.Errorf("sms: выделенные поля пересекаются — выделите их раздельно")
	}

	var b strings.Builder
	b.WriteString(regexp.QuoteMeta(leadingAnchor(s, fields[0].start)))
	amountGroup, currencyGroup, merchantGroup := 0, 0, 0
	for i, f := range fields {
		switch f.kind {
		case "amount":
			amountGroup = i + 1
			b.WriteString(`([0-9]+[.,][0-9]{2})`)
		case "currency":
			currencyGroup = i + 1
			b.WriteString(`([A-Z]{3})`)
		case "merchant":
			merchantGroup = i + 1
			b.WriteString(merchantPattern(spec.CaptureKind, merchantText))
		}
		if i < len(fields)-1 {
			b.WriteString(regexp.QuoteMeta(s[f.end:fields[i+1].start]))
		} else if f.kind == "merchant" && needsTrailingAnchor(spec.CaptureKind, merchantText) {
			b.WriteString(regexp.QuoteMeta(trailingAnchor(s, f.end)))
		}
	}
	pattern := b.String()

	fixed := strings.TrimSpace(spec.FixedCurrency)
	if currencyGroup == 0 && fixed == "" {
		fixed = money.BaseCurrency.String()
	}
	t := &Template{
		Name: strings.TrimSpace(spec.Name), Sender: strings.TrimSpace(spec.Sender), Pattern: pattern,
		Action:      ActionOperation,
		AmountGroup: amountGroup, CurrencyGroup: currencyGroup, MerchantGroup: merchantGroup,
		CaptureKind: normalizeKind(spec.CaptureKind), FixedCurrency: fixed,
		Type: spec.Type, DefaultCategoryID: spec.DefaultCategoryID, Enabled: true,
	}

	// Самопроверка: собранный шаблон должен извлечь ровно выделенные значения.
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("sms: не удалось собрать корректный шаблон: %w", err)
	}
	mm := re.FindStringSubmatch(s)
	if mm == nil {
		return nil, fmt.Errorf("sms: не удалось собрать шаблон по этим выделениям — попробуйте выделить поля иначе")
	}
	if got := strings.TrimSpace(mm[amountGroup]); got != amountText {
		return nil, fmt.Errorf("sms: сумма собралась как «%s», а выделена «%s» — возможно, выделен остаток (Balance), а не сумма операции", got, amountText)
	}
	if currencyGroup > 0 {
		if got := strings.TrimSpace(mm[currencyGroup]); got != currencyText {
			return nil, fmt.Errorf("sms: валюта собралась как «%s», а выделена «%s» — выделите валюту рядом с суммой", got, currencyText)
		}
	}
	if merchantGroup > 0 {
		if got := strings.TrimSpace(mm[merchantGroup]); got != merchantText {
			return nil, fmt.Errorf("sms: контрагент собрался как «%s», а выделен «%s» — выделите его целиком", got, merchantText)
		}
	}
	return t, nil
}

func overlaps(fs []synthField) bool {
	for i := 1; i < len(fs); i++ {
		if fs[i].start < fs[i-1].end {
			return true
		}
	}
	return false
}

// merchantPattern обобщает признак по его СОДЕРЖИМОМУ, а не по типу: чистый
// номер (цифры/звёздочка) — узкий класс; всё остальное (имена со словами) —
// «любой текст» с якорем-концом. Так имя со словами не ломает сборку.
func merchantPattern(_, value string) string {
	if accountValueRe.MatchString(value) {
		return `([0-9*]+)`
	}
	return `(.+?)`
}

func needsTrailingAnchor(_, value string) bool {
	// Ограниченный класс [0-9*]+ сам останавливается — якорь не нужен.
	return !accountValueRe.MatchString(value)
}

// leadingAnchor возвращает литеральный якорь перед полем: ближайшее слово слева
// + разделители (напр. "Oplata ", "Summa ", "platezha ").
func leadingAnchor(s string, pos int) string {
	j := pos
	for j > 0 && isSpaceByte(s[j-1]) {
		j--
	}
	k := j
	for k > 0 && !isSpaceByte(s[k-1]) {
		k--
	}
	return s[k:pos]
}

// trailingAnchor возвращает литеральный якорь после поля: разделители + ближайшее
// слово справа (напр. ". Balance"), чтобы ограничить «жадные» признаки.
func trailingAnchor(s string, pos int) string {
	j := pos
	for j < len(s) && !isAlnumByte(s[j]) {
		j++
	}
	for j < len(s) && isAlnumByte(s[j]) {
		j++
	}
	return s[pos:j]
}

func isSpaceByte(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

func isAlnumByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
