package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"github.com/MathiasDPX/archivetube/internal/config"
)

const sessionCookieName = "archivetube_session"

// store valid session tokens in memory
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

func isLoggedIn(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return validSession(c.Value)
}

func (h *handlers) getRealIp(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return r.RemoteAddr
	}

	if h.config.Server.RealIPHeader == "" {
		return host
	}

	realip := r.Header.Get(h.config.Server.RealIPHeader)

	if realip == "" {
		return host
	}

	return realip
}

func (h *handlers) authEnabled() bool {
	switch h.config.Auth.Mode {
	case "password":
		return h.config.Auth.PasswordHash != ""
	case "oidc":
		return true
	default:
		return h.config.Auth.PasswordHash != ""
	}
}

// middleware that redirects to /login if not authenticated
func (h *handlers) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.authEnabled() {
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

// middleware for unauthenticated API requests
func (h *handlers) requireAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.authEnabled() {
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

// --- Password auth handlers ---

func (h *handlers) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if h.config.Auth.Mode == "oidc" {
		h.handleOIDCLogin(w, r)
		return
	}

	if h.config.Auth.PasswordHash == "" || isLoggedIn(r) {
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
	if err := bcrypt.CompareHashAndPassword([]byte(h.config.Auth.PasswordHash), []byte(password)); err != nil {
		log.Printf("WARN: failed login attempt from %s", h.getRealIp(r))
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

// --- OIDC auth ---

type oidcAuth struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
}

func newOIDCAuth(cfg *config.AuthConfig) *oidcAuth {
	provider, err := oidc.NewProvider(context.Background(), cfg.OIDCIssuer)
	if err != nil {
		log.Fatalf("oidc: failed to create provider: %v", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDCClientID,
	})

	return &oidcAuth{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
	}
}

func (h *handlers) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if isLoggedIn(r) {
		http.Redirect(w, r, "/archive", http.StatusSeeOther)
		return
	}

	state, err := newSessionToken()
	if err != nil {
		h.serverError(w, err)
		return
	}
	nonce, err := newSessionToken()
	if err != nil {
		h.serverError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_nonce",
		Value:    nonce,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	http.Redirect(w, r, h.oidc.oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (h *handlers) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stateCookie, err := r.Cookie("oidc_state")
	if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	oauth2Token, err := h.oidc.oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("oidc: token exchange failed: %v", err)
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in response", http.StatusInternalServerError)
		return
	}

	idToken, err := h.oidc.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("oidc: id_token verification failed: %v", err)
		http.Error(w, "id_token verification failed", http.StatusInternalServerError)
		return
	}

	nonceCookie, err := r.Cookie("oidc_nonce")
	if err != nil || idToken.Nonce != nonceCookie.Value {
		http.Error(w, "nonce mismatch", http.StatusBadRequest)
		return
	}

	// Clear OIDC cookies
	for _, name := range []string{"oidc_state", "oidc_nonce"} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
	}

	// Create a session
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
