package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaludineth/pingwatch/api/internal/middleware"
	"github.com/kaludineth/pingwatch/api/internal/store"
)

type CheckHandler struct {
	store *store.Store
}

func NewCheckHandler(s *store.Store) *CheckHandler {
	return &CheckHandler{store: s}
}

// ownsMonitor verifies the monitor exists and belongs to the authenticated user.
func (h *CheckHandler) ownsMonitor(w http.ResponseWriter, r *http.Request, monitorID string) bool {
	userID := middleware.UserIDFromCtx(r.Context())
	_, err := h.store.GetMonitor(r.Context(), monitorID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return false
	}
	return true
}

// ListChecks handles GET /api/monitors/:id/checks?from=&to=&limit=
// Defaults: from=24h ago, to=now, limit=200.
func (h *CheckHandler) ListChecks(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "id")
	if !h.ownsMonitor(w, r, monitorID) {
		return
	}

	q := r.URL.Query()
	from := parseTimeOr(q.Get("from"), time.Now().Add(-24*time.Hour))
	to := parseTimeOr(q.Get("to"), time.Now())
	limit := parseIntOr(q.Get("limit"), 200)

	checks, err := h.store.ListChecks(r.Context(), monitorID, from, to, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if checks == nil {
		checks = []store.Check{}
	}
	writeJSON(w, http.StatusOK, checks)
}

// ListIncidents handles GET /api/monitors/:id/incidents, newest first.
func (h *CheckHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "id")
	if !h.ownsMonitor(w, r, monitorID) {
		return
	}

	incidents, err := h.store.ListIncidents(r.Context(), monitorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if incidents == nil {
		incidents = []store.Incident{}
	}
	writeJSON(w, http.StatusOK, incidents)
}

// LatestCheck handles GET /api/monitors/:id/checks/latest.
// Returns the single most recent check, or 404 if no checks have run yet.
func (h *CheckHandler) LatestCheck(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "id")
	if !h.ownsMonitor(w, r, monitorID) {
		return
	}

	check, err := h.store.GetLatestCheck(r.Context(), monitorID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no checks found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, check)
}

// GetStats handles GET /api/monitors/:id/stats.
// Always returns uptime_1d, uptime_7d, uptime_30d, avg_response_ms, p95_response_ms.
func (h *CheckHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "id")
	if !h.ownsMonitor(w, r, monitorID) {
		return
	}

	stats, err := h.store.GetStats(r.Context(), monitorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func parseTimeOr(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return def
	}
	return t
}

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	return v
}
