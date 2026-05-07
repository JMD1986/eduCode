package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"nextwork/backend/internal/store"
)

type Server struct {
	store *store.Store
	mux   chi.Router
}

func New(st *store.Store, allowedOrigins []string) *Server {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", headerDevUserSubject},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	s := &Server{store: st, mux: r}

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.getHealth)
		r.Get("/classes", s.getClasses)
		r.Get("/classes/{classID}", s.getClass)
		r.With(s.devUser).Post("/classes/{classID}/enroll", s.postEnroll)
		r.With(s.devUser).Get("/me/classes", s.getMyClasses)
	})

	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

// ParseAllowedCORSOrigins splits a comma-separated env value. Empty items are skipped.
func ParseAllowedCORSOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}
