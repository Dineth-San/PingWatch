package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Monitor struct {
	ID              string
	UserID          string
	Name            string
	URL             string
	IntervalSeconds int
	IsActive        bool
	CreatedAt       time.Time
}

func (s *Store) CreateMonitor(ctx context.Context, userID, name, url string, intervalSeconds int) (*Monitor, error) {
	m := &Monitor{}
	err := s.db.QueryRow(ctx,
		`INSERT INTO monitors (user_id, name, url, interval_seconds)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, name, url, interval_seconds, is_active, created_at`,
		userID, name, url, intervalSeconds,
	).Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.IntervalSeconds, &m.IsActive, &m.CreatedAt)
	return m, err
}

func (s *Store) ListMonitors(ctx context.Context, userID string) ([]Monitor, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, name, url, interval_seconds, is_active, created_at
		 FROM monitors WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.IntervalSeconds, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (s *Store) GetMonitor(ctx context.Context, id, userID string) (*Monitor, error) {
	m := &Monitor{}
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, name, url, interval_seconds, is_active, created_at
		 FROM monitors WHERE id = $1 AND user_id = $2 AND is_active = true`,
		id, userID,
	).Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.IntervalSeconds, &m.IsActive, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (s *Store) UpdateMonitor(ctx context.Context, id, userID, name, url string, intervalSeconds int, isActive bool) (*Monitor, error) {
	m := &Monitor{}
	err := s.db.QueryRow(ctx,
		`UPDATE monitors SET name=$3, url=$4, interval_seconds=$5, is_active=$6
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, user_id, name, url, interval_seconds, is_active, created_at`,
		id, userID, name, url, intervalSeconds, isActive,
	).Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &m.IntervalSeconds, &m.IsActive, &m.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

// DeactivateMonitor sets is_active=false, preserving all check history.
// Returns ErrNotFound if the monitor doesn't exist or belongs to a different user.
func (s *Store) DeactivateMonitor(ctx context.Context, id, userID string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE monitors SET is_active=false WHERE id=$1 AND user_id=$2`,
		id, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
