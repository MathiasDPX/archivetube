package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "archivetube_session"

// sessionStore holds valid session tokens in memory.
var (
	sessions   = map[string]struct{}{}
	sessionsMu sync.RWMutex
)

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func addSession(token string) {
	sessionsMu.Lock()
	sessions[token] = struct{}{}
	sessionsMu.Unlock()
}

func validSession(token string) bool {
	sessionsMu.RLock()
	_, ok := sessions[token]
	sessionsMu.RUnlock()
	return ok
}

func deleteSession(token string) {
	sessionsMu.Lock()
	delete(sessions, token)
	sessionsMu.Unlock()
}

// isLoggedIn checks whether the request carries a valid session cookie.
func isLoggedIn(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return validSession(c.Value)
}

// requireAuth is middleware that redirects to /login if not authenticated.
// If no password is configured, all requests are allowed through.
func (h *handlers) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.config.PasswordHash == "" {
			next(w, r)
			return
		}
		if !isLoggedIn(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// requireAuthAPI is middleware that returns 401 for unauthenticated API requests.
// If no password is configured, all requests are allowed through.
func (h *handlers) requireAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.config.PasswordHash == "" {
			next(w, r)
			return
		}
		if !isLoggedIn(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (h *handlers) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if h.config.PasswordHash == "" || isLoggedIn(r) {
		http.Redirect(w, r, "/archive", http.StatusSeeOther)
		return
	}
	h.renderWithRequest(w, r, "login.tmpl", LoginData{})
}

type LoginData struct {
	Error string
}

func (h *handlers) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.serverError(w, err)
		return
	}

	password := r.FormValue("password")
	if err := bcrypt.CompareHashAndPassword([]byte(h.config.PasswordHash), []byte(password)); err != nil {
		h.renderWithRequest(w, r, "login.tmpl", LoginData{Error: "Invalid password."})
		return
	}

	token, err := newSessionToken()
	if err != nil {
		h.serverError(w, err)
		return
	}
	addSession(token)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/archive", http.StatusSeeOther)
}

func (h *handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		deleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
