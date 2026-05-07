package server

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const ctxKeyUserID ctxKey = 1

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxKeyUserID).(uuid.UUID)
	return id, ok
}
