package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nextwork/backend/internal/store"
)

type errorBody struct {
	Error string `json:"error"`
}

type classResponse struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description,omitempty"`
	Capacity           int        `json:"capacity"`
	Status             string     `json:"status"`
	EnrollmentOpensAt  *time.Time `json:"enrollment_opens_at,omitempty"`
	EnrollmentClosesAt *time.Time `json:"enrollment_closes_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	ActiveEnrollments  int        `json:"active_enrollments"`
}

type myClassResponse struct {
	classResponse
	EnrollmentStatus string    `json:"enrollment_status"`
	EnrolledAt       time.Time `json:"enrolled_at"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

func classToJSON(c store.Class) classResponse {
	out := classResponse{
		ID:                c.ID.String(),
		Title:             c.Title,
		Capacity:          c.Capacity,
		Status:            c.Status,
		CreatedAt:         c.CreatedAt.UTC(),
		ActiveEnrollments: c.ActiveEnrollments,
	}
	if c.Description.Valid {
		s := c.Description.String
		out.Description = &s
	}
	if c.EnrollmentOpensAt != nil {
		t := c.EnrollmentOpensAt.UTC()
		out.EnrollmentOpensAt = &t
	}
	if c.EnrollmentClosesAt != nil {
		t := c.EnrollmentClosesAt.UTC()
		out.EnrollmentClosesAt = &t
	}
	return out
}

func (s *Server) getHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getClasses(w http.ResponseWriter, r *http.Request) {
	classes, err := s.store.ListPublishedClasses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list classes")
		return
	}
	out := make([]classResponse, 0, len(classes))
	for _, c := range classes {
		out = append(out, classToJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"classes": out})
}

func (s *Server) getClass(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "classID")
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}
	c, err := s.store.GetPublishedClass(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrClassNotFound) {
			writeError(w, http.StatusNotFound, "class not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load class")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"class": classToJSON(*c)})
}

func (s *Server) postEnroll(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	raw := chi.URLParam(r, "classID")
	classID, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}
	err = s.store.EnrollInClass(r.Context(), userID, classID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrClassNotFound):
			writeError(w, http.StatusNotFound, "class not found")
		case errors.Is(err, store.ErrClassNotPublished),
			errors.Is(err, store.ErrEnrollmentNotOpenYet),
			errors.Is(err, store.ErrEnrollmentClosed):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, store.ErrClassFull),
			errors.Is(err, store.ErrAlreadyEnrolled):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to enroll")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getMyClasses(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	list, err := s.store.ListMyClasses(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list enrollments")
		return
	}
	out := make([]myClassResponse, 0, len(list))
	for _, mc := range list {
		out = append(out, myClassResponse{
			classResponse:    classToJSON(mc.Class),
			EnrollmentStatus: mc.EnrollmentStatus,
			EnrolledAt:       mc.EnrolledAt.UTC(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"classes": out})
}
