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
	MerchantGroup     int    // номер группы продавца; 0 => не захватывать
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
	Merchant     string // продавец, если захвачен шаблоном
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
	Merchant          string // название продавца, если захвачено (MerchantGroup)
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

// Service — сервис разбора SMS.
type Service struct {
	templates TemplateRepo
	drafts    DraftRepo
	ledger    *ledgersvc.Service
	clock     ports.Clock
	ids       ports.IDGenerator
}

// New создаёт сервис.
func New(templates TemplateRepo, drafts DraftRepo, ledger *ledgersvc.Service, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{templates: templates, drafts: drafts, ledger: ledger, clock: clock, ids: ids}
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

	// Совпал шаблон и задана категория — создаём операцию сразу.
	if res.Matched && res.DefaultCategoryID != "" {
		tx, err := s.ledger.Add(res.Type, receivedAt, res.DefaultCategoryID, res.Amount, smsNote(res.Merchant, text))
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

// smsNote формирует примечание операции из SMS: продавец, если захвачен, иначе
// укороченный текст сообщения.
func smsNote(merchant, rawText string) string {
	if merchant != "" {
		return merchant
	}
	return "SMS: " + truncate(rawText, 120)
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
		cur := t.FixedCurrency
		if t.CurrencyGroup > 0 && t.CurrencyGroup < len(m) {
			cur = strings.TrimSpace(m[t.CurrencyGroup])
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
			DefaultCategoryID: t.DefaultCategoryID,
		}, nil
	}
	return ParseResult{Matched: false}, nil
}

// normalizeAmount приводит "1 234,56" → "1234.56" (убирает пробелы, запятая→точка).
func normalizeAmount(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	return s
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// --- Входящие (drafts) ---

func (s *Service) ListDrafts(unresolvedOnly bool) ([]*Draft, error) {
	return s.drafts.List(unresolvedOnly)
}

func (s *Service) DeleteDraft(id string) error { return s.drafts.Delete(id) }

// ResolveDraft превращает черновик в операцию: назначается категория; сумма/тип
// берутся из разобранных полей, либо из переданных override (для нераспознанных).
func (s *Service) ResolveDraft(id, categoryID string, amountOverride *money.Money, typeOverride core.EntryType) (*transaction.Transaction, error) {
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
	tx, err := s.ledger.Add(typ, d.ReceivedAt, categoryID, *amount, smsNote(d.Merchant, d.RawText))
	if err != nil {
		return nil, err
	}
	d.Resolved = true
	if err := s.drafts.Save(d); err != nil {
		return nil, err
	}
	return tx, nil
}
