package server

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const headerDevUserSubject = "X-User-Subject"

// userFromSession resolves the opaque nw_session cookie to an internal user id.
func (s *Server) userFromSession(r *http.Request) (uuid.UUID, bool) {
	c, err := r.Cookie(cookieSession)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		return uuid.Nil, false
	}
	sid, err := uuid.Parse(c.Value)
	if err != nil {
		return uuid.Nil, false
	}
	userID, _, ok, err := s.store.ResolveSession(r.Context(), sid)
	if err != nil || !ok {
		return uuid.Nil, false
	}
	return userID, true
}

// requireUser prefers a DB-backed session cookie created after OAuth login.
// When opts.AuthDev is true (never in production), X-User-Subject is accepted for integration tests.
func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := s.userFromSession(r); ok {
			next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), uid)))
			return
		}

		if s.opts.AuthDev {
			sub := strings.TrimSpace(r.Header.Get(headerDevUserSubject))
			if sub != "" {
				id, err := s.store.EnsureUser(r.Context(), sub)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "failed to resolve user")
					return
				}
				next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), id)))
				return
			}
		}

		writeError(w, http.StatusUnauthorized, "authentication required")
	})
}
