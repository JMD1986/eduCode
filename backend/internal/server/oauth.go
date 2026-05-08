package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	cookieOAuthState = "nw_oauth_state"
	cookieOAuthPKCE  = "nw_oauth_pkce"
	cookieSession    = "nw_session"
	pathOAuthCookies = "/api/auth"
)

type oauthRuntime struct {
	cfg      OAuthSettings
	provider *oidc.Provider
	oauth2   oauth2.Config
}

func normalizeOAuth(cfg *OAuthSettings) {
	cfg.AppPublicOrigin = strings.TrimSpace(cfg.AppPublicOrigin)
	cfg.RedirectURL = strings.TrimSpace(cfg.RedirectURL)
	cfg.Issuer = strings.TrimSpace(cfg.Issuer)
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 7 * 24 * time.Hour
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
}

func buildOAuthRuntime(ctx context.Context, cfg OAuthSettings) (*oauthRuntime, error) {
	normalizeOAuth(&cfg)
	switch {
	case cfg.Issuer == "":
		return nil, errors.New("oauth: missing issuer")
	case cfg.ClientID == "":
		return nil, errors.New("oauth: missing client id")
	case cfg.RedirectURL == "":
		return nil, errors.New("oauth: missing redirect URL")
	case cfg.AppPublicOrigin == "":
		return nil, errors.New("oauth: missing app public origin")
	}

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}

	o2 := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	return &oauthRuntime{cfg: cfg, provider: provider, oauth2: o2}, nil
}

func randomURLSafe(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Server) oauthLogin(w http.ResponseWriter, r *http.Request) {
	if s.oauthRT == nil {
		writeError(w, http.StatusServiceUnavailable, "oauth is not configured")
		return
	}

	verifier, err := randomURLSafe(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare login")
		return
	}
	state, err := randomURLSafe(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to prepare login")
		return
	}

	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	sec := s.oauthRT.cfg.CookieSecure
	http.SetCookie(w, &http.Cookie{
		Name:     cookieOAuthPKCE,
		Value:    verifier,
		Path:     pathOAuthCookies,
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sec,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     cookieOAuthState,
		Value:    state,
		Path:     pathOAuthCookies,
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sec,
	})

	authURL := s.oauthRT.oauth2.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func clearOAuthCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{cookieOAuthState, cookieOAuthPKCE} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     pathOAuthCookies,
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
		})
	}
}

func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
	if s.oauthRT == nil {
		writeError(w, http.StatusServiceUnavailable, "oauth is not configured")
		return
	}

	q := r.URL.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		desc := q.Get("error_description")
		msg := "oauth error: " + errMsg
		if desc != "" {
			msg += ": " + desc
		}
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	code := q.Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing code")
		return
	}
	stateParam := q.Get("state")

	stateCookie, err := r.Cookie(cookieOAuthState)
	if err != nil || stateCookie.Value == "" {
		writeError(w, http.StatusBadRequest, "missing oauth state cookie")
		return
	}
	if stateParam != stateCookie.Value {
		writeError(w, http.StatusBadRequest, "oauth state mismatch")
		return
	}
	pkceCookie, err := r.Cookie(cookieOAuthPKCE)
	if err != nil || pkceCookie.Value == "" {
		writeError(w, http.StatusBadRequest, "missing oauth pkce cookie")
		return
	}

	ctx := r.Context()
	tok, err := s.oauthRT.oauth2.Exchange(ctx, code, oauth2.VerifierOption(pkceCookie.Value))
	if err != nil {
		log.Printf("oauth token exchange: %v", err)
		writeError(w, http.StatusUnauthorized, "token exchange failed")
		return
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		writeError(w, http.StatusBadRequest, "missing id_token; ensure openid scope is configured")
		return
	}

	oidcVerifier := s.oauthRT.provider.Verifier(&oidc.Config{ClientID: s.oauthRT.oauth2.ClientID})
	idToken, err := oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("id_token verify: %v", err)
		writeError(w, http.StatusUnauthorized, "invalid id_token")
		return
	}

	sub := strings.TrimSpace(idToken.Subject)
	if sub == "" {
		writeError(w, http.StatusUnauthorized, "missing subject")
		return
	}

	userID, err := s.store.EnsureUser(ctx, sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist user")
		return
	}

	sessionID, err := s.store.CreateSession(ctx, userID, s.oauthRT.cfg.SessionTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	sec := s.oauthRT.cfg.CookieSecure
	clearOAuthCookies(w, sec)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieSession,
		Value:    sessionID.String(),
		Path:     "/",
		MaxAge:   int(s.oauthRT.cfg.SessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sec,
	})

	target := strings.TrimRight(s.oauthRT.cfg.AppPublicOrigin, "/") + "/"
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) oauthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if c, err := r.Cookie(cookieSession); err == nil && c.Value != "" {
		if sid, err := uuid.Parse(c.Value); err == nil {
			if delErr := s.store.DeleteSession(r.Context(), sid); delErr != nil {
				log.Printf("session delete: %v", delErr)
			}
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieSession,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.sessionCookieSecure,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getAuthSession(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieSession)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	sid, err := uuid.Parse(c.Value)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	_, sub, ok, err := s.store.ResolveSession(r.Context(), sid)
	if err != nil || !ok {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"subject":       sub,
	})
}
