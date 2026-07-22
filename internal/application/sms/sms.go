// Package sms — разбор банковских SMS по настраиваемым шаблонам (FR-SMS).
//
// Парсинг выполняется на сервере: Android-приложение шлёт сырой текст SMS, а
// сервис по включённым шаблонам (regex) извлекает сумму/валюту/тип и создаёт
// операцию. Если шаблон не задал категорию или не совпал — SMS попадает во
// «входящие» (inbox_drafts) для ручного разбора в кабинете.
package sms

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	ledgersvc "advisor/internal/application/ledger"
	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
	"advisor/internal/domain/transaction"
)

// Template — настраиваемый шаблон разбора SMS.
type Template struct {
	ID                string
	Name              string
	Sender            string // подстрока отправителя (без учёта регистра); "" => любой
	Pattern           string // regex по тексту SMS
	AmountGroup       int    // номер группы суммы (>=1)
	CurrencyGroup     int    // номер группы валюты; 0 => FixedCurrency
	MerchantGroup     int    // номер группы признака (контрагент/счёт); 0 => не захватывать
	CaptureKind       string // что ловит MerchantGroup: "merchant" (контрагент) | "account" (счёт)
	FixedCurrency     string
	Type              core.EntryType
	DefaultCategoryID string // "" => операция уйдёт во «входящие» на ручную категоризацию
	Enabled           bool
	Priority          int
	CreatedAt         time.Time
}

// Draft — нераспознанная или неотнесённая к категории входящая SMS.
type Draft struct {
	ID           string
	Source       string
	RawSender    string
	RawText      string
	ReceivedAt   core.Date
	ParsedAmount *money.Money // nil => распознать не удалось
	ParsedType   core.EntryType
	Merchant     string // контрагент, если захвачен шаблоном
	TemplateID   string
	Resolved     bool
	CreatedAt    time.Time
}

// ParseResult — результат применения шаблонов к SMS (для теста в кабинете и ingest).
type ParseResult struct {
	Matched           bool
	TemplateID        string
	TemplateName      string
	Amount            money.Money
	Type              core.EntryType
	Merchant          string // захваченный признак (контрагент или счёт), если есть
	Kind              string // тип признака: "merchant" | "account" (из шаблона)
	DefaultCategoryID string
}

// TemplateRepo — хранилище шаблонов.
type TemplateRepo interface {
	Save(t *Template) error
	Get(id string) (*Template, error)
	List() ([]*Template, error)        // все, по убыванию priority
	ListEnabled() ([]*Template, error) // только включённые, по убыванию priority
	Delete(id string) error
}

// DraftRepo — хранилище входящих черновиков.
type DraftRepo interface {
	Save(d *Draft) error
	Get(id string) (*Draft, error)
	List(unresolvedOnly bool) ([]*Draft, error)
	Delete(id string) error
}

// CategoryRule — правило авто-категоризации «контрагент → категория».
type CategoryRule struct {
	ID         string
	Pattern    string // подстрока названия контрагента (без учёта регистра)
	CategoryID string
	Priority   int
	CreatedAt  time.Time
}

// RuleRepo — хранилище правил.
type RuleRepo interface {
	Create(r *CategoryRule) error
	List() ([]*CategoryRule, error) // по убыванию priority
	Delete(id string) error
}

// Тип признака распознавания.
const (
	KindMerchant = "merchant" // контрагент (мерчант из покупки) — имя читаемо
	KindAccount  = "account"  // счёт (ЕРИП/перевод) — номер, нужен ярлык
)

// normalizeKind приводит тип к допустимому ("merchant" по умолчанию).
func normalizeKind(k string) string {
	if k == KindAccount {
		return KindAccount
	}
	return KindMerchant
}

// MerchantEntry — запись строгого списка распознавания: признак из SMS
// (контрагент или счёт), сколько раз встречался, оборот, дата, тип, ярлык и
// закреплённая категория.
type MerchantEntry struct {
	Name       string // сам признак (токен из SMS): имя мерчанта или номер счёта
	Kind       string // "merchant" | "account"
	Label      string // человеческое название (для счетов), показывается в операции
	SeenCount  int
	Total      money.Money
	LastSeen   string // YYYY-MM-DD
	CategoryID string // категория, закреплённая за признаком (точная привязка)
}

// MerchantRepo — строгий список распознавания (по владельцу): точное совпадение
// по имени; тип, ярлык и категория закреплены за записью.
type MerchantRepo interface {
	// Observe регистрирует встречу признака: +1 к счётчику, +сумма к обороту.
	// kind проставляется при создании записи (из шаблона).
	Observe(name, kind string, amount money.Money, on core.Date) error
	List() ([]*MerchantEntry, error)             // по убыванию частоты
	Assign(name, categoryID, label string) error // закрепить категорию и ярлык
	Entry(name string) (*MerchantEntry, error)   // запись или nil, если нет
}

// UnknownMerchant — метка операции, когда признак из SMS не распознан.
const UnknownMerchant = "Контрагент (Неизвестно)"

// Service — сервис разбора SMS.
type Service struct {
	templates TemplateRepo
	drafts    DraftRepo
	rules     RuleRepo
	merchants MerchantRepo
	ledger    *ledgersvc.Service
	clock     ports.Clock
	ids       ports.IDGenerator
}

// New создаёт сервис.
func New(templates TemplateRepo, drafts DraftRepo, rules RuleRepo, merchants MerchantRepo, ledger *ledgersvc.Service, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{templates: templates, drafts: drafts, rules: rules, merchants: merchants, ledger: ledger, clock: clock, ids: ids}
}

// ListMerchants возвращает строгий список распознавания (контрагенты и счета);
// категория и ярлык берутся прямо из записи (точная привязка, не подстрока).
func (s *Service) ListMerchants() ([]*MerchantEntry, error) {
	return s.merchants.List()
}

// AssignMerchant закрепляет категорию и ярлык за признаком (точно). Пустой
// categoryID сбрасывает категорию; пустой label убирает название.
func (s *Service) AssignMerchant(name, categoryID, label string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("sms: пустой признак")
	}
	return s.merchants.Assign(name, categoryID, strings.TrimSpace(label))
}

// --- Правила «контрагент → категория» ---

func (s *Service) ListRules() ([]*CategoryRule, error) { return s.rules.List() }

func (s *Service) CreateRule(pattern, categoryID string, priority int) (*CategoryRule, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || categoryID == "" {
		return nil, fmt.Errorf("sms: нужны контрагент (подстрока) и категория")
	}
	r := &CategoryRule{
		ID: s.ids.NewID(), Pattern: pattern, CategoryID: categoryID,
		Priority: priority, CreatedAt: s.clock.Now().UTC(),
	}
	if err := s.rules.Create(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) DeleteRule(id string) error { return s.rules.Delete(id) }

// matchRule возвращает категорию по первому правилу, чья подстрока есть в контрагенте.
func (s *Service) matchRule(merchant string) string {
	if merchant == "" {
		return ""
	}
	rules, err := s.rules.List()
	if err != nil {
		return ""
	}
	ml := strings.ToLower(merchant)
	for _, r := range rules {
		if r.Pattern != "" && strings.Contains(ml, strings.ToLower(r.Pattern)) {
			return r.CategoryID
		}
	}
	return ""
}

// --- Шаблоны (CRUD) ---

func (s *Service) ListTemplates() ([]*Template, error) { return s.templates.List() }

func (s *Service) CreateTemplate(t *Template) (*Template, error) {
	if err := validateTemplate(t); err != nil {
		return nil, err
	}
	t.ID = s.ids.NewID()
	t.CreatedAt = s.clock.Now().UTC()
	if err := s.templates.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) UpdateTemplate(id string, patch *Template) (*Template, error) {
	if err := validateTemplate(patch); err != nil {
		return nil, err
	}
	existing, err := s.templates.Get(id)
	if err != nil {
		return nil, err
	}
	patch.ID = existing.ID
	patch.CreatedAt = existing.CreatedAt
	if err := s.templates.Save(patch); err != nil {
		return nil, err
	}
	return patch, nil
}

func (s *Service) DeleteTemplate(id string) error { return s.templates.Delete(id) }

func validateTemplate(t *Template) error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("sms: имя шаблона обязательно")
	}
	if strings.TrimSpace(t.Pattern) == "" {
		return fmt.Errorf("sms: regex-шаблон обязателен")
	}
	if _, err := regexp.Compile(t.Pattern); err != nil {
		return fmt.Errorf("sms: некорректный regex: %w", err)
	}
	if t.AmountGroup < 1 {
		return fmt.Errorf("sms: номер группы суммы должен быть ≥ 1")
	}
	if t.CurrencyGroup == 0 && strings.TrimSpace(t.FixedCurrency) == "" {
		return fmt.Errorf("sms: задайте группу валюты или фиксированную валюту")
	}
	if !t.Type.Valid() {
		return fmt.Errorf("sms: тип должен быть expense или income")
	}
	return nil
}

// --- Разбор ---

// Test применяет включённые шаблоны к SMS, не сохраняя ничего (для кабинета).
func (s *Service) Test(sender, text string) (ParseResult, error) {
	tmpls, err := s.templates.ListEnabled()
	if err != nil {
		return ParseResult{}, err
	}
	return parseWith(tmpls, sender, text)
}

// IngestOutcome — что произошло при приёме SMS.
type IngestOutcome struct {
	Matched       bool
	TransactionID string // не пусто => создана операция
	DraftID       string // не пусто => создан черновик (нужен ручной разбор)
}

// Ingest принимает сырое SMS: парсит по шаблонам и либо создаёт операцию (если
// шаблон совпал и задал категорию), либо кладёт во «входящие».
func (s *Service) Ingest(sender, text string, receivedAt core.Date) (IngestOutcome, error) {
	tmpls, err := s.templates.ListEnabled()
	if err != nil {
		return IngestOutcome{}, err
	}
	res, err := parseWith(tmpls, sender, text)
	if err != nil {
		return IngestOutcome{}, err
	}

	// Признак распознан — учитываем его в строгом списке (частота/оборот), тип из
	// шаблона. Заодно берём запись для категории и ярлыка.
	var entry *MerchantEntry
	if res.Matched && res.Merchant != "" {
		_ = s.merchants.Observe(res.Merchant, res.Kind, res.Amount, receivedAt)
		entry, _ = s.merchants.Entry(res.Merchant)
	}

	// Категория, по приоритету:
	//   1) закреплённая за признаком (точное совпадение);
	//   2) правило-подстрока (доп. слой для массовых паттернов);
	//   3) дефолт шаблона.
	category := res.DefaultCategoryID
	if entry != nil && entry.CategoryID != "" {
		category = entry.CategoryID
	} else if res.Matched && res.Merchant != "" {
		if ruleCat := s.matchRule(res.Merchant); ruleCat != "" {
			category = ruleCat
		}
	}

	// Есть категория — создаём операцию сразу. Примечание: ярлык записи, иначе сам
	// признак, иначе «Неизвестно».
	if res.Matched && category != "" {
		note := smsNote(res.Merchant, entryLabel(entry))
		tx, err := s.ledger.AddFromSMS(res.Type, receivedAt, category, res.Amount, note, res.Merchant)
		if err != nil {
			return IngestOutcome{}, err
		}
		return IngestOutcome{Matched: true, TransactionID: tx.Meta.ID}, nil
	}

	// Иначе — во «входящие» (распознанные, но без категории, либо нераспознанные).
	d := &Draft{
		ID:         s.ids.NewID(),
		Source:     "sms",
		RawSender:  sender,
		RawText:    text,
		ReceivedAt: receivedAt,
		TemplateID: res.TemplateID,
		CreatedAt:  s.clock.Now().UTC(),
	}
	if res.Matched {
		amt := res.Amount
		d.ParsedAmount = &amt
		d.ParsedType = res.Type
		d.Merchant = res.Merchant
	}
	if err := s.drafts.Save(d); err != nil {
		return IngestOutcome{}, err
	}
	return IngestOutcome{Matched: res.Matched, DraftID: d.ID}, nil
}

// smsNote формирует примечание операции: человеческий ярлык (если задан), иначе
// сам признак (имя мерчанта/номер счёта), иначе метка «Контрагент (Неизвестно)».
func smsNote(merchant, label string) string {
	if label != "" {
		return label
	}
	if merchant != "" {
		return merchant
	}
	return UnknownMerchant
}

// entryLabel безопасно достаёт ярлык записи (или "" если записи нет).
func entryLabel(e *MerchantEntry) string {
	if e == nil {
		return ""
	}
	return e.Label
}

// parseWith применяет шаблоны по порядку и возвращает первый успешный разбор.
func parseWith(tmpls []*Template, sender, text string) (ParseResult, error) {
	for _, t := range tmpls {
		if t.Sender != "" && !strings.Contains(strings.ToLower(sender), strings.ToLower(t.Sender)) {
			continue
		}
		re, err := regexp.Compile(t.Pattern)
		if err != nil {
			continue // некорректный шаблон — пропускаем
		}
		m := re.FindStringSubmatch(text)
		if m == nil || t.AmountGroup >= len(m) {
			continue
		}
		cur := strings.TrimSpace(t.FixedCurrency)
		if t.CurrencyGroup > 0 && t.CurrencyGroup < len(m) {
			captured := strings.TrimSpace(m[t.CurrencyGroup])
			if isCurrencyCode(captured) {
				cur = captured
			}
		}
		// Если валюта не распозналась (кривой шаблон захватил не то) — базовая BYN,
		// а не мусор вроде "14.70".
		if !isCurrencyCode(cur) {
			cur = money.BaseCurrency.String()
		}
		amt, err := money.Parse(normalizeAmount(m[t.AmountGroup]), money.Currency(strings.ToUpper(cur)))
		if err != nil {
			continue
		}
		merchant := ""
		if t.MerchantGroup > 0 && t.MerchantGroup < len(m) {
			merchant = strings.TrimSpace(m[t.MerchantGroup])
		}
		return ParseResult{
			Matched:           true,
			TemplateID:        t.ID,
			TemplateName:      t.Name,
			Amount:            amt,
			Type:              t.Type,
			Merchant:          merchant,
			Kind:              normalizeKind(t.CaptureKind),
			DefaultCategoryID: t.DefaultCategoryID,
		}, nil
	}
	return ParseResult{Matched: false}, nil
}

// isCurrencyCode проверяет, что строка — правдоподобный код валюты (3 латинские буквы).
func isCurrencyCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

// normalizeAmount приводит "1 234,56" → "1234.56" (убирает пробелы, запятая→точка).
func normalizeAmount(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	return s
}

// --- Входящие (drafts) ---

func (s *Service) ListDrafts(unresolvedOnly bool) ([]*Draft, error) {
	return s.drafts.List(unresolvedOnly)
}

func (s *Service) DeleteDraft(id string) error { return s.drafts.Delete(id) }

// ResolveDraft превращает черновик в операцию. Если rememberMerchant=true и у
// черновика есть контрагент — категория закрепляется прямо за контрагентом
// (точно), и будущие SMS от него будут разноситься автоматически.
func (s *Service) ResolveDraft(id, categoryID string, amountOverride *money.Money, typeOverride core.EntryType, rememberMerchant bool) (*transaction.Transaction, error) {
	d, err := s.drafts.Get(id)
	if err != nil {
		return nil, err
	}
	amount := d.ParsedAmount
	if amountOverride != nil {
		amount = amountOverride
	}
	if amount == nil {
		return nil, fmt.Errorf("sms: у черновика нет суммы — укажите её вручную")
	}
	typ := d.ParsedType
	if typeOverride != "" {
		typ = typeOverride
	}
	if !typ.Valid() {
		return nil, fmt.Errorf("sms: укажите тип операции (расход/доход)")
	}
	label := ""
	if d.Merchant != "" {
		if e, _ := s.merchants.Entry(d.Merchant); e != nil {
			label = e.Label
		}
	}
	tx, err := s.ledger.AddFromSMS(typ, d.ReceivedAt, categoryID, *amount, smsNote(d.Merchant, label), d.Merchant)
	if err != nil {
		return nil, err
	}
	d.Resolved = true
	if err := s.drafts.Save(d); err != nil {
		return nil, err
	}
	// Запомнить: закрепить категорию за признаком (точно) для будущих SMS,
	// сохранив уже заданный ярлык (если был).
	if rememberMerchant && d.Merchant != "" {
		_ = s.merchants.Assign(d.Merchant, categoryID, label)
	}
	return tx, nil
}
