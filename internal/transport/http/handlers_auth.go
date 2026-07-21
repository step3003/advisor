package http

import (
	"errors"
	"net/http"

	accountsvc "advisor/internal/application/account"
	"advisor/internal/infrastructure/auth"
)

func (s *Server) handleAuthStatus(w http.ResponseWriter, _ *http.Request) {
	reg, err := s.svc.Accounts.Registered()
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
	token, u, err := s.svc.Accounts.Register(req.Username, req.Password, req.DeviceName)
	if err != nil {
		if errors.Is(err, accountsvc.ErrRegistrationClosed) {
			writeErr(w, http.StatusForbidden, err.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, authResp{Token: token, Username: u.Username})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	token, u, err := s.svc.Accounts.Login(req.Username, req.Password, req.DeviceName)
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
	if err := s.svc.Accounts.Logout(token); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	token := auth.TokenFromHeader(r.Header.Get("Authorization"))
	u, err := s.svc.Accounts.Validate(token)
	if err != nil {
		// Мог войти статическим токеном устройства — тогда аккаунта нет.
		writeJSON(w, http.StatusOK, map[string]string{"username": ""})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": u.Username})
}
