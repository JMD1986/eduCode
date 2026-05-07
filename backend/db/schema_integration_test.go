package db_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"

	appdb "nextwork/backend/db"
)

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://app_user:app_password@127.0.0.1:5432/education_app?sslmode=disable"
}

func openPostgres(t *testing.T) *sql.DB {
	t.Helper()
	dsn := testDatabaseURL(t)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("postgres not reachable at %q (start docker compose or set DATABASE_URL): %v", dsn, err)
	}
	return db
}

func resetMigrations(t *testing.T, dsn string) {
	t.Helper()
	if err := appdb.MigrateDownAll(dsn); err != nil {
		t.Fatalf("migrate down all: %v", err)
	}
	if err := appdb.MigrateUp(dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	t.Cleanup(func() {
		if err := appdb.MigrateDownAll(dsn); err != nil {
			t.Errorf("cleanup migrate down all: %v", err)
		}
	})
}

func TestSchema_initial_migration_creates_expected_tables(t *testing.T) {
	sqlDB := openPostgres(t)
	dsn := testDatabaseURL(t)
	resetMigrations(t, dsn)

	ctx := context.Background()
	for _, table := range []string{"users", "classes", "enrollments"} {
		var name string
		q := `SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'r' AND n.nspname = 'public' AND c.relname = $1`
		if err := sqlDB.QueryRowContext(ctx, q, table).Scan(&name); err != nil {
			t.Fatalf("table %q: %v", table, err)
		}
		if name != table {
			t.Fatalf("unexpected table name %q for %q", name, table)
		}
	}
}

func TestSchema_enrollments_user_class_is_unique(t *testing.T) {
	sqlDB := openPostgres(t)
	dsn := testDatabaseURL(t)
	resetMigrations(t, dsn)
	ctx := context.Background()

	var userID, classID string
	if err := sqlDB.QueryRowContext(ctx, `INSERT INTO users (idp_subject, display_name) VALUES ($1, $2) RETURNING id`, "sub:alice", "Alice").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, `INSERT INTO classes (title, capacity, status) VALUES ($1, $2, 'published') RETURNING id`, "Go 101", 10).Scan(&classID); err != nil {
		t.Fatalf("insert class: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO enrollments (user_id, class_id, status) VALUES ($1, $2, 'active')`, userID, classID); err != nil {
		t.Fatalf("first enrollment: %v", err)
	}
	_, err := sqlDB.ExecContext(ctx, `INSERT INTO enrollments (user_id, class_id, status) VALUES ($1, $2, 'active')`, userID, classID)
	if err == nil {
		t.Fatal("expected duplicate enrollment to fail")
	}
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("expected unique_violation (23505), got %v", err)
	}
}

func TestSchema_enrollments_require_valid_foreign_keys(t *testing.T) {
	sqlDB := openPostgres(t)
	dsn := testDatabaseURL(t)
	resetMigrations(t, dsn)
	ctx := context.Background()

	var userID string
	if err := sqlDB.QueryRowContext(ctx, `INSERT INTO users (idp_subject) VALUES ($1) RETURNING id`, "sub:bob").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	fakeClassID := "00000000-0000-0000-0000-000000000001"
	_, err := sqlDB.ExecContext(ctx, `INSERT INTO enrollments (user_id, class_id, status) VALUES ($1, $2, 'active')`, userID, fakeClassID)
	if err == nil {
		t.Fatal("expected FK violation for class_id")
	}
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("expected foreign_key_violation (23503), got %v", err)
	}
}
