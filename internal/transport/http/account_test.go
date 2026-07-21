package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// reqTok выполняет запрос с произвольным Bearer-токеном (для тестов сессий).
func reqTok(t *testing.T, s *Server, method, path string, body any, token string) (*httptest.ResponseRecorder, []byte) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, path, &buf)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, r)
	return rec, rec.Body.Bytes()
}

func TestAccountFlow(t *testing.T) {
	s := newTestServer(t)

	// 1. Статус до регистрации — аккаунта нет.
	_, body := reqTok(t, s, http.MethodGet, "/api/auth/status", nil, "")
	var st map[string]bool
	mustJSON(t, body, &st)
	if st["registered"] {
		t.Fatal("ожидалось registered=false до регистрации")
	}

	// 2. Регистрация первого аккаунта.
	rec, body := reqTok(t, s, http.MethodPost, "/api/auth/register",
		authReq{Username: "Alex", Password: "secret123", DeviceName: "mac"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d — %s", rec.Code, body)
	}
	var reg authResp
	mustJSON(t, body, &reg)
	if reg.Token == "" || reg.Username != "alex" {
		t.Fatalf("ожидались токен и username=alex, got %+v", reg)
	}

	// 3. Повторная регистрация закрыта.
	rec, _ = reqTok(t, s, http.MethodPost, "/api/auth/register",
		authReq{Username: "bob", Password: "secret123"}, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("вторая регистрация: ожидался 403, got %d", rec.Code)
	}

	// 4. Токен сессии даёт доступ к защищённым эндпоинтам.
	rec, _ = reqTok(t, s, http.MethodGet, "/api/categories", nil, reg.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("доступ по сессии: ожидался 200, got %d", rec.Code)
	}

	// 5. Без токена — 401.
	rec, _ = reqTok(t, s, http.MethodGet, "/api/categories", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("без токена: ожидался 401, got %d", rec.Code)
	}

	// 6. Вход с неверным паролем — 401.
	rec, _ = reqTok(t, s, http.MethodPost, "/api/auth/login",
		authReq{Username: "alex", Password: "wrong"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("неверный пароль: ожидался 401, got %d", rec.Code)
	}

	// 7. Вход с верным паролем (регистр логина не важен) → новый токен.
	rec, body = reqTok(t, s, http.MethodPost, "/api/auth/login",
		authReq{Username: "ALEX", Password: "secret123", DeviceName: "phone"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d — %s", rec.Code, body)
	}
	var lg authResp
	mustJSON(t, body, &lg)

	// 8. Logout отзывает токен → далее доступа нет.
	rec, _ = reqTok(t, s, http.MethodPost, "/api/auth/logout", nil, lg.Token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: ожидался 204, got %d", rec.Code)
	}
	rec, _ = reqTok(t, s, http.MethodGet, "/api/categories", nil, lg.Token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("после logout: ожидался 401, got %d", rec.Code)
	}

	// 9. Статический токен устройства всё ещё работает (обратная совместимость).
	rec, _ = reqTok(t, s, http.MethodGet, "/api/categories", nil, testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("статический токен: ожидался 200, got %d", rec.Code)
	}
}
