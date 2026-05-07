package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Class is a row from the catalog (API-facing shape).
type Class struct {
	ID                 uuid.UUID
	Title              string
	Description        sql.NullString
	Capacity           int
	Status             string
	EnrollmentOpensAt  *time.Time
	EnrollmentClosesAt *time.Time
	CreatedAt          time.Time
	ActiveEnrollments  int
}

// MyClass is an enrollment joined with class metadata.
type MyClass struct {
	Class
	EnrollmentStatus string
	EnrolledAt       time.Time
}

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) EnsureUser(ctx context.Context, idpSubject string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (idp_subject) VALUES ($1)
		ON CONFLICT (idp_subject) DO UPDATE SET idp_subject = users.idp_subject
		RETURNING id
	`, idpSubject).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

const classSelect = `
	c.id, c.title, c.description, c.capacity, c.status::text, c.enrollment_opens_at, c.enrollment_closes_at, c.created_at,
	(SELECT COUNT(*)::int FROM enrollments e WHERE e.class_id = c.id AND e.status = 'active')`

func (s *Store) ListPublishedClasses(ctx context.Context) ([]Class, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+classSelect+`
		FROM classes c
		WHERE c.status = 'published'
		ORDER BY c.title ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Class
	for rows.Next() {
		var c Class
		if err := rows.Scan(
			&c.ID, &c.Title, &c.Description, &c.Capacity, &c.Status,
			&c.EnrollmentOpensAt, &c.EnrollmentClosesAt, &c.CreatedAt, &c.ActiveEnrollments,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetPublishedClass(ctx context.Context, id uuid.UUID) (*Class, error) {
	var c Class
	err := s.db.QueryRowContext(ctx, `
		SELECT `+classSelect+`
		FROM classes c
		WHERE c.id = $1 AND c.status = 'published'
	`, id).Scan(
		&c.ID, &c.Title, &c.Description, &c.Capacity, &c.Status,
		&c.EnrollmentOpensAt, &c.EnrollmentClosesAt, &c.CreatedAt, &c.ActiveEnrollments,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrClassNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListMyClasses(ctx context.Context, userID uuid.UUID) ([]MyClass, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+classSelect+`, e.status::text, e.created_at
		FROM enrollments e
		JOIN classes c ON c.id = e.class_id
		WHERE e.user_id = $1 AND e.status = 'active'
		ORDER BY e.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MyClass
	for rows.Next() {
		var mc MyClass
		if err := rows.Scan(
			&mc.ID, &mc.Title, &mc.Description, &mc.Capacity, &mc.Status,
			&mc.EnrollmentOpensAt, &mc.EnrollmentClosesAt, &mc.CreatedAt, &mc.ActiveEnrollments,
			&mc.EnrollmentStatus, &mc.EnrolledAt,
		); err != nil {
			return nil, err
		}
		out = append(out, mc)
	}
	return out, rows.Err()
}

func (s *Store) EnrollInClass(ctx context.Context, userID, classID uuid.UUID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		capacity           int
		status             string
		enrollmentOpensAt  sql.NullTime
		enrollmentClosesAt sql.NullTime
	)
	err = tx.QueryRowContext(ctx, `
		SELECT capacity, status::text, enrollment_opens_at, enrollment_closes_at
		FROM classes
		WHERE id = $1
		FOR UPDATE
	`, classID).Scan(&capacity, &status, &enrollmentOpensAt, &enrollmentClosesAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrClassNotFound
	}
	if err != nil {
		return err
	}

	if status != "published" {
		return ErrClassNotPublished
	}

	now := time.Now().UTC()
	if enrollmentOpensAt.Valid && now.Before(enrollmentOpensAt.Time.UTC()) {
		return ErrEnrollmentNotOpenYet
	}
	if enrollmentClosesAt.Valid && now.After(enrollmentClosesAt.Time.UTC()) {
		return ErrEnrollmentClosed
	}

	var activeCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)::int FROM enrollments WHERE class_id = $1 AND status = 'active'
	`, classID).Scan(&activeCount); err != nil {
		return err
	}
	if activeCount >= capacity {
		return ErrClassFull
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO enrollments (user_id, class_id, status) VALUES ($1, $2, 'active')
	`, userID, classID)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyEnrolled
		}
		return err
	}

	return tx.Commit()
}
