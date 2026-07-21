package sqlite

import (
	"database/sql"
	"errors"
	"time"

	accountsvc "advisor/internal/application/account"
	"advisor/internal/application/ports"
)

// UserRepo реализует account.UserRepo поверх таблицы users.
type UserRepo struct{ idx *Index }

// Users возвращает репозиторий пользователей.
func (i *Index) Users() *UserRepo { return &UserRepo{idx: i} }

const userCols = `SELECT id,username,password_hash,role,created_at FROM users`

func (r *UserRepo) Create(u *accountsvc.User) error {
	role := u.Role
	if role == "" {
		role = accountsvc.RoleUser
	}
	_, err := r.idx.db.Exec(`INSERT INTO users(id,username,password_hash,role,created_at) VALUES(?,?,?,?,?)`,
		u.ID, u.Username, u.PasswordHash, role, u.CreatedAt.UTC().Format(rfc3339))
	return err
}

func (r *UserRepo) GetByUsername(username string) (*accountsvc.User, error) {
	return scanUser(r.idx.db.QueryRow(userCols+` WHERE username=?`, username))
}

func (r *UserRepo) GetByID(id string) (*accountsvc.User, error) {
	return scanUser(r.idx.db.QueryRow(userCols+` WHERE id=?`, id))
}

func (r *UserRepo) FirstAdmin() (*accountsvc.User, error) {
	return scanUser(r.idx.db.QueryRow(userCols + ` WHERE role='admin' ORDER BY created_at ASC LIMIT 1`))
}

func (r *UserRepo) Count() (int, error) {
	var n int
	err := r.idx.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepo) All() ([]*accountsvc.User, error) {
	rows, err := r.idx.db.Query(userCols + ` ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*accountsvc.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func scanUser(sc scanner) (*accountsvc.User, error) {
	var u accountsvc.User
	var created string
	if err := sc.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, err
	}
	u.CreatedAt, _ = parseTime(created)
	return &u, nil
}

// SessionRepo реализует account.SessionRepo поверх таблицы auth_sessions.
type SessionRepo struct{ idx *Index }

// Sessions возвращает репозиторий сессий.
func (i *Index) Sessions() *SessionRepo { return &SessionRepo{idx: i} }

func (r *SessionRepo) Create(s *accountsvc.Session) error {
	_, err := r.idx.db.Exec(`INSERT INTO auth_sessions(id,user_id,token_hash,name,created_at,last_used_at)
		VALUES(?,?,?,?,?,?)`,
		s.ID, s.UserID, s.TokenHash, s.Name, s.CreatedAt.UTC().Format(rfc3339), s.LastUsedAt.UTC().Format(rfc3339))
	return err
}

func (r *SessionRepo) GetByTokenHash(hash string) (*accountsvc.Session, error) {
	row := r.idx.db.QueryRow(`SELECT id,user_id,token_hash,name,created_at,last_used_at,revoked_at
		FROM auth_sessions WHERE token_hash=?`, hash)
	var s accountsvc.Session
	var created, lastUsed string
	var revoked sql.NullString
	if err := row.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.Name, &created, &lastUsed, &revoked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrRecordNotFound
		}
		return nil, err
	}
	s.CreatedAt, _ = parseTime(created)
	s.LastUsedAt, _ = parseTime(lastUsed)
	if revoked.Valid {
		t, _ := parseTime(revoked.String)
		s.RevokedAt = &t
	}
	return &s, nil
}

func (r *SessionRepo) Revoke(id string) error {
	_, err := r.idx.db.Exec(`UPDATE auth_sessions SET revoked_at=? WHERE id=?`,
		time.Now().UTC().Format(rfc3339), id)
	return err
}

func (r *SessionRepo) TouchLastUsed(id string, t time.Time) error {
	_, err := r.idx.db.Exec(`UPDATE auth_sessions SET last_used_at=? WHERE id=?`,
		t.UTC().Format(rfc3339), id)
	return err
}
