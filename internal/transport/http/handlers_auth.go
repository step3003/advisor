package http

import (
	"errors"
	"net/http"

	accountsvc "advisor/internal/application/account"
	"advisor/internal/infrastructure/auth"
)

func (s *Server) handleAuthStatus(w http.ResponseWriter, _ *http.Request) {
	reg, err := s.g.Accounts.Registered()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"registered": reg})
}

type authReq struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DeviceName string `json:"deviceName"`
}

type authResp struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	token, u, err := s.g.Accounts.Register(req.Username, req.Password, req.DeviceName)
	if err != nil {
		if errors.Is(err, accountsvc.ErrRegistrationClosed) {
			writeErr(w, http.StatusForbidden, err.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	// Засеваем предустановленные категории новому пользователю (Приложение A).
	_, _ = s.g.ForUser(u.ID).Catalog.SeedDefaults()
	writeJSON(w, http.StatusCreated, authResp{Token: token, Username: u.Username})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	token, u, err := s.g.Accounts.Login(req.Username, req.Password, req.DeviceName)
	if err != nil {
		if errors.Is(err, accountsvc.ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, authResp{Token: token, Username: u.Username})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := auth.TokenFromHeader(r.Header.Get("Authorization"))
	if err := s.g.Accounts.Logout(token); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.g.Accounts.User(s.userID(r))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "не найден пользователь")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"username": u.Username,
		"role":     u.Role,
		"isAdmin":  u.IsAdmin(),
	})
}

// --- Управление пользователями (только админ) ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeErr(w, http.StatusForbidden, "только для админа")
		return
	}
	users, err := s.g.Accounts.ListUsers()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{"id": u.ID, "username": u.Username, "role": u.Role})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		writeErr(w, http.StatusForbidden, "только для админа")
		return
	}
	var req authReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := s.g.Accounts.CreateUser(req.Username, req.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	// Засеваем категории новому пользователю.
	_, _ = s.g.ForUser(u.ID).Catalog.SeedDefaults()
	writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "username": u.Username, "role": u.Role})
}
