package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Check struct {
	ID             string
	MonitorID      string
	CheckedAt      time.Time
	StatusCode     *int
	ResponseTimeMs *int
	IsUp           bool
	ErrorMessage   *string
}

type Incident struct {
	ID              string
	MonitorID       string
	StartedAt       time.Time
	ResolvedAt      *time.Time
	DurationSeconds *int
}

// Stats holds per-monitor uptime and response-time metrics across three windows.
// Field names match the JSON keys expected by the spec.
type Stats struct {
	Uptime1d      float64 `json:"uptime_1d"`
	Uptime7d      float64 `json:"uptime_7d"`
	Uptime30d     float64 `json:"uptime_30d"`
	AvgResponseMs float64 `json:"avg_response_ms"`
	P95ResponseMs float64 `json:"p95_response_ms"`
}

// MonitorSummary is returned by ListMonitorSummaries. It includes the monitor
// fields plus the latest check status and 30-day uptime so the dashboard list
// view needs only a single request.
type MonitorSummary struct {
	ID              string
	UserID          string
	Name            string
	URL             string
	IntervalSeconds int
	IsActive        bool
	CreatedAt       time.Time
	IsUp            *bool    // nil when no checks have run yet
	ResponseTimeMs  *int     // nil when no checks have run yet
	Uptime30d       float64
}

// GetLatestCheck returns the most recent check for a monitor.
// Returns ErrNotFound when no checks have run yet.
func (s *Store) GetLatestCheck(ctx context.Context, monitorID string) (*Check, error) {
	c := &Check{}
	err := s.db.QueryRow(ctx,
		`SELECT id, monitor_id, checked_at, status_code, response_time_ms, is_up, error_message
		 FROM checks WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 1`,
		monitorID,
	).Scan(&c.ID, &c.MonitorID, &c.CheckedAt, &c.StatusCode, &c.ResponseTimeMs, &c.IsUp, &c.ErrorMessage)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *Store) ListChecks(ctx context.Context, monitorID string, from, to time.Time, limit int) ([]Check, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, monitor_id, checked_at, status_code, response_time_ms, is_up, error_message
		 FROM checks
		 WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at <= $3
		 ORDER BY checked_at DESC
		 LIMIT $4`,
		monitorID, from, to, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.MonitorID, &c.CheckedAt, &c.StatusCode, &c.ResponseTimeMs, &c.IsUp, &c.ErrorMessage); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (s *Store) ListIncidents(ctx context.Context, monitorID string) ([]Incident, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, monitor_id, started_at, resolved_at, duration_seconds
		 FROM incidents WHERE monitor_id = $1 ORDER BY started_at DESC`,
		monitorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var i Incident
		if err := rows.Scan(&i.ID, &i.MonitorID, &i.StartedAt, &i.ResolvedAt, &i.DurationSeconds); err != nil {
			return nil, err
		}
		incidents = append(incidents, i)
	}
	return incidents, rows.Err()
}

// GetStats computes uptime_1d, uptime_7d, uptime_30d and response-time metrics
// for a monitor in a single query. avg and p95 are over the last 24 hours.
func (s *Store) GetStats(ctx context.Context, monitorID string) (*Stats, error) {
	stats := &Stats{}
	err := s.db.QueryRow(ctx,
		`SELECT
			COALESCE(
				COUNT(*) FILTER (WHERE is_up AND checked_at >= now() - interval '1 day') * 100.0
				/ NULLIF(COUNT(*) FILTER (WHERE checked_at >= now() - interval '1 day'), 0),
				0
			) AS uptime_1d,
			COALESCE(
				COUNT(*) FILTER (WHERE is_up AND checked_at >= now() - interval '7 days') * 100.0
				/ NULLIF(COUNT(*) FILTER (WHERE checked_at >= now() - interval '7 days'), 0),
				0
			) AS uptime_7d,
			COALESCE(
				COUNT(*) FILTER (WHERE is_up AND checked_at >= now() - interval '30 days') * 100.0
				/ NULLIF(COUNT(*) FILTER (WHERE checked_at >= now() - interval '30 days'), 0),
				0
			) AS uptime_30d,
			COALESCE(
				AVG(response_time_ms) FILTER (
					WHERE response_time_ms IS NOT NULL
					  AND checked_at >= now() - interval '1 day'
				), 0
			) AS avg_ms,
			COALESCE(
				PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time_ms)
					FILTER (WHERE response_time_ms IS NOT NULL AND checked_at >= now() - interval '1 day'),
				0
			) AS p95_ms
		FROM checks
		WHERE monitor_id = $1
		  AND checked_at >= now() - interval '30 days'`,
		monitorID,
	).Scan(&stats.Uptime1d, &stats.Uptime7d, &stats.Uptime30d, &stats.AvgResponseMs, &stats.P95ResponseMs)
	return stats, err
}

// ListMonitorSummaries returns monitors for a user with the latest check status
// and 30-day uptime pre-joined, so the dashboard list needs only one request.
func (s *Store) ListMonitorSummaries(ctx context.Context, userID string) ([]MonitorSummary, error) {
	rows, err := s.db.Query(ctx,
		`SELECT
			m.id, m.user_id, m.name, m.url, m.interval_seconds, m.is_active, m.created_at,
			c.is_up,
			c.response_time_ms,
			COALESCE(
				(SELECT COUNT(*) FILTER (WHERE is_up) * 100.0
				        / NULLIF(COUNT(*), 0)
				 FROM checks
				 WHERE monitor_id = m.id
				   AND checked_at >= now() - interval '30 days'),
				0
			) AS uptime_30d
		FROM monitors m
		LEFT JOIN LATERAL (
			SELECT is_up, response_time_ms
			FROM checks
			WHERE monitor_id = m.id
			ORDER BY checked_at DESC
			LIMIT 1
		) c ON true
		WHERE m.user_id = $1
		ORDER BY m.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []MonitorSummary
	for rows.Next() {
		var ms MonitorSummary
		if err := rows.Scan(
			&ms.ID, &ms.UserID, &ms.Name, &ms.URL, &ms.IntervalSeconds, &ms.IsActive, &ms.CreatedAt,
			&ms.IsUp, &ms.ResponseTimeMs, &ms.Uptime30d,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, ms)
	}
	return summaries, rows.Err()
}
