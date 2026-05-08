package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	appdb "nextwork/backend/db"
	"nextwork/backend/internal/server"
	"nextwork/backend/internal/store"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = "postgres://app_user:app_password@127.0.0.1:5432/education_app?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("sql open: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		log.Fatalf("db ping: %v", err)
	}
	cancel()

	if err := appdb.MigrateUp(dsn); err != nil {
		log.Fatalf("migrate up: %v", err)
	}

	addr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}

	corsOrigins := server.ParseAllowedCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	opts := server.Options{
		AllowedOrigins: corsOrigins,
		AuthDev:        os.Getenv("AUTH_DEV") == "1",
	}

	if cid := strings.TrimSpace(os.Getenv("OAUTH_CLIENT_ID")); cid != "" {
		sessionTTL := 7 * 24 * time.Hour
		if v := strings.TrimSpace(os.Getenv("SESSION_TTL_HOURS")); v != "" {
			if h, err := strconv.Atoi(v); err == nil && h > 0 {
				sessionTTL = time.Duration(h) * time.Hour
			}
		}
		scopes := strings.Fields(strings.TrimSpace(os.Getenv("OAUTH_SCOPES")))
		opts.OAuth = &server.OAuthSettings{
			Issuer:          strings.TrimSpace(os.Getenv("OAUTH_ISSUER")),
			ClientID:        cid,
			ClientSecret:    strings.TrimSpace(os.Getenv("OAUTH_CLIENT_SECRET")),
			RedirectURL:     strings.TrimSpace(os.Getenv("OAUTH_REDIRECT_URI")),
			AppPublicOrigin: strings.TrimSpace(os.Getenv("APP_PUBLIC_ORIGIN")),
			Scopes:          scopes,
			CookieSecure:    os.Getenv("COOKIE_SECURE") == "1",
			SessionTTL:      sessionTTL,
		}
	}

	srv, err := server.New(store.New(db), opts)
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
