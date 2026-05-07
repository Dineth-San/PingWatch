package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/kaludineth/pingwatch/api/internal/middleware"
	"github.com/kaludineth/pingwatch/api/internal/store"
)

type MonitorHandler struct {
	store *store.Store
}

func NewMonitorHandler(s *store.Store) *MonitorHandler {
	return &MonitorHandler{store: s}
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	summaries, err := h.store.ListMonitorSummaries(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if summaries == nil {
		summaries = []store.MonitorSummary{}
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	var body struct {
		Name            string `json:"name"`
		URL             string `json:"url"`
		IntervalSeconds int    `json:"interval_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !isValidURL(body.URL) {
		writeError(w, http.StatusBadRequest, "url must be a valid http or https URL")
		return
	}
	if !isValidInterval(body.IntervalSeconds) {
		writeError(w, http.StatusBadRequest, "interval_seconds must be 60, 120, 300, or 600")
		return
	}

	m, err := h.store.CreateMonitor(r.Context(), userID, body.Name, body.URL, body.IntervalSeconds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *MonitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	m, err := h.store.GetMonitor(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	var body struct {
		Name            string `json:"name"`
		URL             string `json:"url"`
		IntervalSeconds int    `json:"interval_seconds"`
		IsActive        bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !isValidURL(body.URL) {
		writeError(w, http.StatusBadRequest, "url must be a valid http or https URL")
		return
	}
	if !isValidInterval(body.IntervalSeconds) {
		writeError(w, http.StatusBadRequest, "interval_seconds must be 60, 120, 300, or 600")
		return
	}

	m, err := h.store.UpdateMonitor(r.Context(), id, userID, body.Name, body.URL, body.IntervalSeconds, body.IsActive)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.store.DeactivateMonitor(r.Context(), id, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isValidURL returns true only for absolute http:// or https:// URLs with a non-empty host.
func isValidURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// isValidInterval returns true for the four allowed check intervals.
func isValidInterval(seconds int) bool {
	switch seconds {
	case 60, 120, 300, 600:
		return true
	}
	return false
}
