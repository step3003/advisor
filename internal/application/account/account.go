// Package account — аккаунты и вход по логину/паролю (FR-AUTH, ТЗ v2.0).
//
// Один аккаунт используется со всех устройств: вход выдаёт per-device токен
// (сессию), проверяемый middleware. Пароль хранится как pbkdf2-хеш (stdlib,
// без внешних зависимостей). Регистрация разрешена только на первом запуске
// (пока нет ни одного пользователя) — чтобы публичный сервер нельзя было занять.
package account

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"advisor/internal/application/ports"
)

const (
	pbkdf2Iters = 210_000
	saltLen     = 16
	keyLen      = 32
	tokenBytes  = 32
)

var (
	// ErrInvalidCredentials — неверный логин или пароль.
	ErrInvalidCredentials = errors.New("account: неверный логин или пароль")
	// ErrRegistrationClosed — регистрация закрыта (аккаунт уже существует).
	ErrRegistrationClosed = errors.New("account: регистрация закрыта — аккаунт уже создан")
	// ErrUsernameTaken — логин занят.
	ErrUsernameTaken = errors.New("account: логин уже занят")
)

// User — учётная запись.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Session — токен доступа устройства.
type Session struct {
	ID         string
	UserID     string
	TokenHash  string
	Name       string
	CreatedAt  time.Time
	LastUsedAt time.Time
	RevokedAt  *time.Time
}

// UserRepo — хранилище пользователей.
type UserRepo interface {
	Create(u *User) error
	GetByUsername(username string) (*User, error) // ports.ErrRecordNotFound, если нет
	GetByID(id string) (*User, error)
	Count() (int, error)
}

// SessionRepo — хранилище сессий.
type SessionRepo interface {
	Create(s *Session) error
	GetByTokenHash(hash string) (*Session, error) // ports.ErrRecordNotFound, если нет
	Revoke(id string) error
	TouchLastUsed(id string, t time.Time) error
}

// Service — сервис аккаунтов.
type Service struct {
	users    UserRepo
	sessions SessionRepo
	clock    ports.Clock
	ids      ports.IDGenerator
}

// New создаёт сервис.
func New(users UserRepo, sessions SessionRepo, clock ports.Clock, ids ports.IDGenerator) *Service {
	return &Service{users: users, sessions: sessions, clock: clock, ids: ids}
}

// Registered сообщает, создан ли уже аккаунт (для выбора экрана вход/регистрация).
func (s *Service) Registered() (bool, error) {
	n, err := s.users.Count()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Register создаёт первый аккаунт. Разрешено только пока нет пользователей.
func (s *Service) Register(username, password, deviceName string) (token string, u *User, err error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" || len(password) < 6 {
		return "", nil, fmt.Errorf("account: логин обязателен, пароль не короче 6 символов")
	}
	n, err := s.users.Count()
	if err != nil {
		return "", nil, err
	}
	if n > 0 {
		return "", nil, ErrRegistrationClosed
	}
	u = &User{
		ID:           s.ids.NewID(),
		Username:     username,
		PasswordHash: hashPassword(password),
		CreatedAt:    s.clock.Now().UTC(),
	}
	if err := s.users.Create(u); err != nil {
		return "", nil, err
	}
	token, err = s.issueSession(u.ID, deviceName)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// Login проверяет логин/пароль и выдаёт токен сессии.
func (s *Service) Login(username, password, deviceName string) (token string, u *User, err error) {
	username = strings.TrimSpace(strings.ToLower(username))
	u, err = s.users.GetByUsername(username)
	if err != nil {
		if errors.Is(err, ports.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if !verifyPassword(password, u.PasswordHash) {
		return "", nil, ErrInvalidCredentials
	}
	token, err = s.issueSession(u.ID, deviceName)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// Validate проверяет токен и возвращает пользователя (для middleware).
func (s *Service) Validate(token string) (*User, error) {
	if token == "" {
		return nil, ErrInvalidCredentials
	}
	sess, err := s.sessions.GetByTokenHash(tokenHash(token))
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if sess.RevokedAt != nil {
		return nil, ErrInvalidCredentials
	}
	_ = s.sessions.TouchLastUsed(sess.ID, s.clock.Now().UTC())
	return s.users.GetByID(sess.UserID)
}

// Logout отзывает сессию по токену.
func (s *Service) Logout(token string) error {
	sess, err := s.sessions.GetByTokenHash(tokenHash(token))
	if err != nil {
		return nil // уже нет — считаем успехом
	}
	return s.sessions.Revoke(sess.ID)
}

func (s *Service) issueSession(userID, name string) (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := s.clock.Now().UTC()
	sess := &Session{
		ID:         s.ids.NewID(),
		UserID:     userID,
		TokenHash:  tokenHash(token),
		Name:       strings.TrimSpace(name),
		CreatedAt:  now,
		LastUsedAt: now,
	}
	if err := s.sessions.Create(sess); err != nil {
		return "", err
	}
	return token, nil
}

// --- Хеширование ---

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// hashPassword возвращает "pbkdf2-sha256$iters$saltB64$hashB64".
func hashPassword(pw string) string {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	dk, _ := pbkdf2.Key(sha256.New, pw, salt, pbkdf2Iters, keyLen)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", pbkdf2Iters,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk))
}

func verifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iters, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, iters, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
