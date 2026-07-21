// Package http — транспортный слой: HTTP JSON API поверх usecase-сервисов (ТЗ v2.0, раздел 8).
//
// Зависит только от application (+ domain через DTO); БД/сеть скрыты за сервисами.
package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	accountsvc "advisor/internal/application/account"
	catalogsvc "advisor/internal/application/catalog"
	currencysvc "advisor/internal/application/currency"
	iosvc "advisor/internal/application/io"
	ledgersvc "advisor/internal/application/ledger"
	planningsvc "advisor/internal/application/planning"
	"advisor/internal/application/ports"
	recurringsvc "advisor/internal/application/recurring"
	reportingsvc "advisor/internal/application/reporting"
	settingssvc "advisor/internal/application/settings"
	smssvc "advisor/internal/application/sms"
	"advisor/internal/infrastructure/auth"
)

// UserServices — usecase-сервисы, привязанные к данным конкретного пользователя.
type UserServices struct {
	Catalog   *catalogsvc.Service
	Ledger    *ledgersvc.Service
	Planning  *planningsvc.Service
	Recurring *recurringsvc.Service
	Reporting *reportingsvc.Service
	Settings  *settingssvc.Service
	IO        *iosvc.Service
	SMS       *smssvc.Service
}

// Global — глобальные сервисы и фабрика user-scoped сервисов (собирает main).
type Global struct {
	Accounts *accountsvc.Service
	Currency *currencysvc.Service
	Clock    ports.Clock
	// ForUser строит сервисы, работающие с данными пользователя userID.
	ForUser func(userID string) UserServices
}

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxRole
)

// Server — HTTP-обработчик API.
type Server struct {
	g       Global
	auth    *auth.Verifier
	handler http.Handler
	cors    string // разрешённый Origin для dev ("" => без CORS-заголовков)
}

// NewServer собирает роутер и middleware. Если webDir не пуст, по путям вне
// /api/ раздаётся собранный SPA (single-origin, без CORS в проде).
func NewServer(g Global, verifier *auth.Verifier, corsOrigin, webDir string) *Server {
	s := &Server{g: g, auth: verifier, cors: corsOrigin}
	mux := http.NewServeMux()
	s.routes(mux)
	apiChain := recoverMW(loggingMW(s.corsMW(s.authMW(mux))))

	if webDir != "" {
		root := http.NewServeMux()
		root.Handle("/api/", apiChain)
		root.Handle("/", spaFileServer(webDir))
		s.handler = root
	} else {
		s.handler = apiChain
	}
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

// user строит сервисы для пользователя текущего запроса.
func (s *Server) user(r *http.Request) UserServices {
	return s.g.ForUser(s.userID(r))
}

func (s *Server) userID(r *http.Request) string {
	uid, _ := r.Context().Value(ctxUserID).(string)
	return uid
}

func (s *Server) isAdmin(r *http.Request) bool {
	role, _ := r.Context().Value(ctxRole).(string)
	return role == accountsvc.RoleAdmin
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Аккаунты (вход по логину/паролю). status/register/login — без авторизации.
	mux.HandleFunc("GET /api/auth/status", s.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)

	// Управление пользователями (только админ).
	mux.HandleFunc("GET /api/admin/users", s.handleListUsers)
	mux.HandleFunc("POST /api/admin/users", s.handleCreateUser)

	// Категории
	mux.HandleFunc("GET /api/categories", s.handleListCategories)
	mux.HandleFunc("POST /api/categories", s.handleCreateCategory)
	mux.HandleFunc("PATCH /api/categories/{id}", s.handlePatchCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", s.handleDeleteCategory)

	// Планы
	mux.HandleFunc("GET /api/plans", s.handleListPlans)
	mux.HandleFunc("PUT /api/plans", s.handleSetPlan)
	mux.HandleFunc("POST /api/plans/copy-previous", s.handleCopyPrevious)

	// Операции
	mux.HandleFunc("GET /api/transactions", s.handleListTransactions)
	mux.HandleFunc("POST /api/transactions", s.handleCreateTransaction)
	mux.HandleFunc("PATCH /api/transactions/{id}", s.handleUpdateTransaction)
	mux.HandleFunc("DELETE /api/transactions/{id}", s.handleDeleteTransaction)

	// Отчёты
	mux.HandleFunc("GET /api/reports/plan-vs-fact", s.handlePlanVsFact)
	mux.HandleFunc("GET /api/reports/period", s.handlePeriodReport)
	mux.HandleFunc("GET /api/reports/dynamics", s.handleDynamics)

	// Повторяющиеся
	mux.HandleFunc("GET /api/recurring", s.handleListRecurring)
	mux.HandleFunc("POST /api/recurring", s.handleCreateRecurring)
	mux.HandleFunc("PATCH /api/recurring/{id}", s.handleUpdateRecurring)
	mux.HandleFunc("POST /api/recurring/{id}/pause", s.handlePauseRecurring)
	mux.HandleFunc("POST /api/recurring/{id}/resume", s.handleResumeRecurring)
	mux.HandleFunc("DELETE /api/recurring/{id}", s.handleDeleteRecurring)
	mux.HandleFunc("POST /api/recurring/generate", s.handleGenerateRecurring)

	// Валюты
	mux.HandleFunc("POST /api/currency/refresh", s.handleRefreshRates)
	mux.HandleFunc("GET /api/currency/currencies", s.handleListCurrencies)

	// Настройки
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", s.handlePatchSettings)

	// Захват SMS (Android-форвардер, FR-SMS): сырой текст → разбор по шаблонам.
	mux.HandleFunc("POST /api/ingest/sms", s.handleIngestSMS)

	// Настройка разбора SMS в кабинете (шаблоны + тест + входящие).
	mux.HandleFunc("GET /api/sms/templates", s.handleListSMSTemplates)
	mux.HandleFunc("POST /api/sms/templates", s.handleCreateSMSTemplate)
	mux.HandleFunc("PATCH /api/sms/templates/{id}", s.handleUpdateSMSTemplate)
	mux.HandleFunc("DELETE /api/sms/templates/{id}", s.handleDeleteSMSTemplate)
	mux.HandleFunc("POST /api/sms/test", s.handleTestSMS)
	mux.HandleFunc("GET /api/inbox", s.handleListDrafts)
	mux.HandleFunc("POST /api/inbox/{id}/resolve", s.handleResolveDraft)
	mux.HandleFunc("DELETE /api/inbox/{id}", s.handleDeleteDraft)

	// Правила «продавец → категория».
	mux.HandleFunc("GET /api/sms/rules", s.handleListRules)
	mux.HandleFunc("POST /api/sms/rules", s.handleCreateRule)
	mux.HandleFunc("DELETE /api/sms/rules/{id}", s.handleDeleteRule)

	// Экспорт
	mux.HandleFunc("GET /api/export/json", s.handleExportJSON)
	mux.HandleFunc("GET /api/export/csv", s.handleExportCSV)
}

// --- middleware ---

// publicPaths — эндпоинты без авторизации.
var publicPaths = map[string]bool{
	"/api/health":        true,
	"/api/auth/status":   true,
	"/api/auth/register": true,
	"/api/auth/login":    true,
}

func (s *Server) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPaths[r.URL.Path] || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		token := auth.TokenFromHeader(r.Header.Get("Authorization"))
		var userID, role string
		switch {
		case s.auth.Valid(token):
			// Статический токен = админ (обратная совместимость: форвардер/скрипты).
			id, err := s.g.Accounts.AdminUserID()
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "нет админ-аккаунта — сначала регистрация")
				return
			}
			userID, role = id, accountsvc.RoleAdmin
		default:
			u, err := s.g.Accounts.Validate(token)
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "требуется вход")
				return
			}
			userID, role = u.ID, u.Role
		}
		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		ctx = context.WithValue(ctx, ctxRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) corsMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cors != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.cors)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				writeErr(w, http.StatusInternalServerError, "внутренняя ошибка")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- JSON-хелперы ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type errResponse struct {
	Error string `json:"error"`
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errResponse{Error: msg})
}

func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// queryTrim возвращает обрезанный query-параметр.
func queryTrim(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}
