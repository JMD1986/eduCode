package server

import (
	"net/http"
)

const headerDevUserSubject = "X-User-Subject"

// DevUser extracts X-User-Subject, ensures a users row exists, and sets uuid in context.
// Replace with real OIDC middleware when auth lands.
func (s *Server) devUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub := r.Header.Get(headerDevUserSubject)
		if sub == "" {
			writeError(w, http.StatusUnauthorized, "missing "+headerDevUserSubject+" header (dev identity)")
			return
		}
		id, err := s.store.EnsureUser(r.Context(), sub)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to resolve user")
			return
		}
		next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), id)))
	})
}
