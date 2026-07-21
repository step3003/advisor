// Package nbrb — HTTP-клиент курсов Национального банка РБ (раздел 7 ТЗ).
//
// Реализует ports.RateProvider. Официальный курс парсится из JSON-числа в целое
// (×10^4) без промежуточного float (NFR-2). Таймаут запроса ≤10 с; сетевые
// ошибки возвращаются наверх, где usecase делает fallback на кэш (FR-CUR-4).
//
// В unit-тестах сеть не дёргается: baseURL подменяется на httptest-сервер либо
// используется мок ports.RateProvider.
package nbrb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"advisor/internal/application/ports"
	"advisor/internal/domain/core"
	"advisor/internal/domain/money"
)

// DefaultBaseURL — публичный адрес API НБ РБ.
const DefaultBaseURL = "https://api.nbrb.by"

// DefaultTimeout — таймаут HTTP-запроса (раздел 7: ≤10 c).
const DefaultTimeout = 10 * time.Second

// Client — HTTP-клиент курсов НБ РБ.
type Client struct {
	baseURL string
	http    *http.Client
}

// New создаёт клиент с адресом по умолчанию и таймаутом 10 с.
func New() *Client {
	return NewWithBase(DefaultBaseURL)
}

// NewWithBase создаёт клиент с указанным базовым URL (для тестов).
func NewWithBase(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: DefaultTimeout},
	}
}

// rateDTO — форма ответа НБ РБ. Cur_OfficialRate — json.Number, чтобы разобрать
// десятичную дробь без float.
type rateDTO struct {
	CurID           int         `json:"Cur_ID"`
	CurAbbreviation string      `json:"Cur_Abbreviation"`
	CurScale        int64       `json:"Cur_Scale"`
	CurName         string      `json:"Cur_Name"`
	CurOfficialRate json.Number `json:"Cur_OfficialRate"`
}

func (d rateDTO) toRate(date core.Date) (ports.Rate, error) {
	scaled, err := parseRateToScaled(d.CurOfficialRate.String())
	if err != nil {
		return ports.Rate{}, fmt.Errorf("nbrb: курс %s: %w", d.CurAbbreviation, err)
	}
	return ports.Rate{
		Currency:      money.Currency(d.CurAbbreviation),
		Date:          date,
		Scale:         d.CurScale,
		RateBYNScaled: scaled,
	}, nil
}

// RatesOn возвращает все курсы на дату.
func (c *Client) RatesOn(date core.Date) ([]ports.Rate, error) {
	url := fmt.Sprintf("%s/exrates/rates?ondate=%s&periodicity=0", c.baseURL, date.String())
	var dtos []rateDTO
	if err := c.getJSON(url, &dtos); err != nil {
		return nil, err
	}
	rates := make([]ports.Rate, 0, len(dtos))
	for _, d := range dtos {
		r, err := d.toRate(date)
		if err != nil {
			return nil, err
		}
		rates = append(rates, r)
	}
	return rates, nil
}

// RateFor возвращает курс одной валюты на дату (parammode=2 — буквенный код).
func (c *Client) RateFor(currency money.Currency, date core.Date) (ports.Rate, error) {
	if currency == money.BaseCurrency {
		// BYN к самому себе — курс 1:1.
		return ports.Rate{Currency: currency, Date: date, Scale: 1, RateBYNScaled: ports.RatePrecisionFactor}, nil
	}
	url := fmt.Sprintf("%s/exrates/rates/%s?ondate=%s&parammode=2", c.baseURL, currency.String(), date.String())
	var d rateDTO
	if err := c.getJSON(url, &d); err != nil {
		return ports.Rate{}, err
	}
	return d.toRate(date)
}

func (c *Client) getJSON(url string, out any) error {
	resp, err := c.http.Get(url)
	if err != nil {
		return fmt.Errorf("nbrb: запрос %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nbrb: %s вернул статус %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("nbrb: разбор ответа %s: %w", url, err)
	}
	return nil
}

// parseRateToScaled переводит десятичный курс ("3.2567") в целое ×10^4.
// Если знаков больше 4 — округляет вверх по половине; без float.
func parseRateToScaled(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("пустой курс")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg, s = true, s[1:]
	}
	intPart := s
	fracPart := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}

	var intVal int64
	if intPart != "" {
		v, err := parseUint(intPart)
		if err != nil {
			return 0, err
		}
		intVal = v
	}

	const prec = 4 // RatePrecisionFactor = 10^4
	// Дробную часть дополняем/усечём до prec знаков с округлением.
	fracDigits := fracPart
	roundUp := false
	if len(fracDigits) > prec {
		if fracDigits[prec] >= '5' {
			roundUp = true
		}
		fracDigits = fracDigits[:prec]
	}
	fracVal := int64(0)
	if fracDigits != "" {
		v, err := parseUint(fracDigits)
		if err != nil {
			return 0, err
		}
		fracVal = v
	}
	for i := len(fracDigits); i < prec; i++ {
		fracVal *= 10
	}

	scaled := intVal*ports.RatePrecisionFactor + fracVal
	if roundUp {
		scaled++
	}
	if neg {
		scaled = -scaled
	}
	return scaled, nil
}

func parseUint(s string) (int64, error) {
	var v int64
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("недопустимый символ %q", string(s[i]))
		}
		v = v*10 + int64(s[i]-'0')
	}
	return v, nil
}

// compile-time проверка.
var _ ports.RateProvider = (*Client)(nil)
