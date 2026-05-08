package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// CreateSession inserts a row keyed by the returned id (stored in the HttpOnly cookie).
func (s *Store) CreateSession(ctx context.Context, userID uuid.UUID, ttl time.Duration) (uuid.UUID, error) {
	expires := time.Now().UTC().Add(ttl)
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO sessions (user_id, expires_at) VALUES ($1, $2)
		RETURNING id
	`, userID, expires).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// ResolveSession returns the internal user id and IdP subject if the session exists and is not expired.
func (s *Store) ResolveSession(ctx context.Context, sessionID uuid.UUID) (userID uuid.UUID, idpSubject string, ok bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT u.id, u.idp_subject
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > now()
	`, sessionID).Scan(&userID, &idpSubject)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, err
	}
	return userID, idpSubject, true, nil
}

// DeleteSession removes one session (logout).
func (s *Store) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}
