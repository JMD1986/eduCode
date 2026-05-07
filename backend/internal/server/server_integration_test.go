package server_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	appdb "nextwork/backend/db"
	"nextwork/backend/internal/server"
	"nextwork/backend/internal/store"
)

func testDSN(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("DATABASE_URL"); d != "" {
		return d
	}
	return "postgres://app_user:app_password@127.0.0.1:5432/education_app?sslmode=disable"
}

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := testDSN(t)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	return db
}

func resetSchema(t *testing.T, dsn string) {
	t.Helper()
	if err := appdb.MigrateDownAll(dsn); err != nil {
		t.Fatalf("migrate down all: %v", err)
	}
	if err := appdb.MigrateUp(dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	t.Cleanup(func() {
		if err := appdb.MigrateDownAll(dsn); err != nil {
			t.Errorf("cleanup migrate: %v", err)
		}
	})
}

func seedClass(t *testing.T, db *sql.DB, title, status string, capacity int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := db.QueryRowContext(context.Background(),
		`INSERT INTO classes (title, capacity, status) VALUES ($1, $2, $3::class_status) RETURNING id`,
		title, capacity, status,
	).Scan(&id); err != nil {
		t.Fatalf("seed class: %v", err)
	}
	return id
}

func TestAPI_catalog_and_enroll(t *testing.T) {
	db := openDB(t)
	dsn := testDSN(t)
	resetSchema(t, dsn)

	pubID := seedClass(t, db, "Public Go", "published", 2)
	_ = seedClass(t, db, "Secret Draft", "draft", 10)

	st := store.New(db)
	h := server.New(st, nil).Handler()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	res, err := ts.Client().Get(ts.URL + "/api/classes")
	if err != nil {
		t.Fatalf("get classes: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d: %s", res.StatusCode, b)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), pubID.String()) {
		t.Fatalf("expected published class id in response: %s", body)
	}
	if strings.Contains(string(body), "Secret") {
		t.Fatalf("draft class leaked into catalog")
	}

	// Enroll
	subject := "sub:test-user-1"
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/classes/"+pubID.String()+"/enroll", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-User-Subject", subject)
	res2, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(res2.Body)
		t.Fatalf("enroll status %d: %s", res2.StatusCode, b)
	}

	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/classes/"+pubID.String()+"/enroll", nil)
	req2.Header.Set("X-User-Subject", subject)
	res3, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(res3.Body)
		t.Fatalf("duplicate enroll status %d: %s", res3.StatusCode, b)
	}

	req4, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/me/classes", nil)
	req4.Header.Set("X-User-Subject", subject)
	res4, err := ts.Client().Do(req4)
	if err != nil {
		t.Fatal(err)
	}
	defer res4.Body.Close()
	if res4.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res4.Body)
		t.Fatalf("my classes %d: %s", res4.StatusCode, b)
	}
	b4, _ := io.ReadAll(res4.Body)
	if !strings.Contains(string(b4), pubID.String()) {
		t.Fatalf("my classes missing enrollment: %s", b4)
	}
}

func TestAPI_enroll_respects_capacity(t *testing.T) {
	db := openDB(t)
	dsn := testDSN(t)
	resetSchema(t, dsn)

	pubID := seedClass(t, db, "Tiny", "published", 1)

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO users (idp_subject) VALUES ($1), ($2)`, "sub:a", "sub:b")
	if err != nil {
		t.Fatalf("users: %v", err)
	}
	var uidA, uidB string
	if err := db.QueryRowContext(ctx, `SELECT id::text FROM users WHERE idp_subject = $1`, "sub:a").Scan(&uidA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT id::text FROM users WHERE idp_subject = $1`, "sub:b").Scan(&uidB); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO enrollments (user_id, class_id, status) VALUES ($1::uuid, $2::uuid, 'active')`, uidA, pubID.String()); err != nil {
		t.Fatal(err)
	}

	st := store.New(db)
	h := server.New(st, nil).Handler()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/classes/"+pubID.String()+"/enroll", nil)
	req.Header.Set("X-User-Subject", "sub:b")
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 409 when full, got %d: %s", res.StatusCode, b)
	}
}
