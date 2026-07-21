// Package auth — простая токен-авторизация для личного API (FR-AUTH).
//
// Проверка Bearer-токена в постоянном времени. Набор валидных токенов задаётся
// конфигом (env): пользовательский токен и токены устройств (Android-форвардер),
// которые можно отозвать, убрав из конфига. Полноценные аккаунты — в бэклоге.
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"
)

// Verifier хранит хеши валидных токенов и проверяет предъявленные.
type Verifier struct {
	hashes [][32]byte
}

// New создаёт верификатор из списка токенов. Пустые значения игнорируются.
func New(tokens ...string) *Verifier {
	v := &Verifier{}
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		v.hashes = append(v.hashes, sha256.Sum256([]byte(t)))
	}
	return v
}

// Enabled сообщает, задан ли хотя бы один токен. Если нет — API должен отказать
// в старте (нельзя поднимать открытый API).
func (v *Verifier) Enabled() bool { return len(v.hashes) > 0 }

// Valid проверяет токен в постоянном времени против всех известных.
func (v *Verifier) Valid(token string) bool {
	if token == "" {
		return false
	}
	got := sha256.Sum256([]byte(token))
	ok := 0
	for i := range v.hashes {
		ok |= subtle.ConstantTimeCompare(got[:], v.hashes[i][:])
	}
	return ok == 1
}

// TokenFromHeader извлекает токен из заголовка "Authorization: Bearer <token>".
func TokenFromHeader(authHeader string) string {
	const prefix = "Bearer "
	if len(authHeader) > len(prefix) && strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return strings.TrimSpace(authHeader[len(prefix):])
	}
	return ""
}
