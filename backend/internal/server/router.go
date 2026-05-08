package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"nextwork/backend/internal/store"
)

// Server wires HTTP routes to handlers and middleware.
type Server struct {
	store *store.Store
	mux   chi.Router
	opts  Options

	oauthRT             *oauthRuntime
	sessionCookieSecure bool // aligns logout cookie flags with login when OAuth is configured
}

// New constructs the HTTP API. OAuth machinery initializes eagerly when opts.OAuth is set.
func New(st *store.Store, opts Options) (*Server, error) {
	allowedOrigins := opts.AllowedOrigins
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
		AllowedHeaders:   []string{"Accept", "Content-Type", "Origin", headerDevUserSubject},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	sessionCookieSecure := false
	var oauthRT *oauthRuntime
	if opts.OAuth != nil {
		cfg := *opts.OAuth
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		rt, err := buildOAuthRuntime(ctx, cfg)
		if err != nil {
			return nil, err
		}
		oauthRT = rt
		sessionCookieSecure = rt.cfg.CookieSecure
	}

	s := &Server{
		store:               st,
		mux:                 r,
		opts:                opts,
		oauthRT:             oauthRT,
		sessionCookieSecure: sessionCookieSecure,
	}

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.getHealth)

		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", s.oauthLogin)
			r.Get("/callback", s.oauthCallback)
			r.Post("/logout", s.oauthLogout)
			r.Get("/session", s.getAuthSession)
		})

		r.Get("/classes", s.getClasses)
		r.Get("/classes/{classID}", s.getClass)
		r.With(s.requireUser).Post("/classes/{classID}/enroll", s.postEnroll)
		r.With(s.requireUser).Get("/me/classes", s.getMyClasses)
	})

	return s, nil
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
