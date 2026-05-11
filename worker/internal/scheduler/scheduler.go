package scheduler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaludineth/pingwatch/worker/internal/mailer"
)

const refreshInterval = 60 * time.Second

// Monitor holds the fields the scheduler needs for each active monitor.
type Monitor struct {
	ID              string
	Name            string
	URL             string
	UserEmail       string
	IntervalSeconds int
}

// checkResult is the outcome of a single pingURL call.
type checkResult struct {
	IsUp           bool
	StatusCode     int    // 0 when no HTTP response received (e.g. timeout/DNS failure)
	ResponseTimeMs int    // always set, even on error
	ErrorMessage   string // empty on success
}

// runningEntry tracks a running monitor goroutine so the refresh loop can
// stop it (via cancel) or detect that its interval has changed.
type runningEntry struct {
	cancel          context.CancelFunc
	intervalSeconds int
	url             string
}

// Scheduler manages the set of running monitor goroutines.
type Scheduler struct {
	db      *pgxpool.Pool
	mailer  *mailer.Mailer // nil disables email alerts
	state   sync.Map       // map[monitorID]bool — last known is_up per monitor
	mu      sync.Mutex     // protects running
	running map[string]runningEntry
}

func New(db *pgxpool.Pool, m *mailer.Mailer) *Scheduler {
	return &Scheduler{
		db:      db,
		mailer:  m,
		running: make(map[string]runningEntry),
	}
}

// Start performs the initial monitor load, spawns goroutines, then launches
// the background refresh loop.
func (s *Scheduler) Start(ctx context.Context) error {
	monitors, err := loadActiveMonitors(ctx, s.db)
	if err != nil {
		return err
	}
	log.Printf("scheduler: starting with %d active monitors", len(monitors))

	s.mu.Lock()
	for _, m := range monitors {
		s.startLocked(ctx, m)
	}
	s.mu.Unlock()

	go s.refreshLoop(ctx)
	return nil
}

// refreshLoop re-queries active monitors every 60 seconds and reconciles the
// running goroutine set against the DB.
func (s *Scheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcile(ctx)
		}
	}
}

// reconcile diffs the DB active-monitor set against s.running and:
//   - starts a new goroutine for monitors not yet running
//   - stops the goroutine for monitors that are deactivated or deleted
//   - stops and restarts the goroutine for monitors whose interval changed
//
// Both passes run under a single mutex acquisition to prevent a concurrent
// reconcile tick from observing a partially-updated running map.
func (s *Scheduler) reconcile(ctx context.Context) {
	active, err := loadActiveMonitors(ctx, s.db)
	if err != nil {
		log.Printf("scheduler: reconcile query: %v", err)
		return
	}

	// Build an O(1)-lookup set of the current active monitors.
	activeSet := make(map[string]Monitor, len(active))
	for _, m := range active {
		activeSet[m.ID] = m
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Pass 1: stop goroutines for monitors that are gone or whose interval changed.
	for id, e := range s.running {
		m, stillActive := activeSet[id]
		if !stillActive {
			e.cancel()
			delete(s.running, id)
			s.state.Delete(id) // reset so a future re-activation starts fresh
			log.Printf("monitor %s: deactivated — goroutine stopped", id)
		} else if m.IntervalSeconds != e.intervalSeconds || m.URL != e.url {
			e.cancel()
			delete(s.running, id)
			s.state.Delete(id) // reset so the new goroutine starts from unknown state
			log.Printf("monitor %s: config changed (interval=%ds url=%s) — restarting", id, m.IntervalSeconds, m.URL)
		}
	}

	// Pass 2: start goroutines for newly added monitors and just-restarted ones.
	for _, m := range active {
		if _, running := s.running[m.ID]; !running {
			s.startLocked(ctx, m)
		}
	}
}

// startLocked registers m in s.running and spawns its goroutine.
// Caller must hold s.mu.
func (s *Scheduler) startLocked(ctx context.Context, m Monitor) {
	mCtx, cancel := context.WithCancel(ctx)
	s.running[m.ID] = runningEntry{
		cancel:          cancel,
		intervalSeconds: m.IntervalSeconds,
		url:             m.URL,
	}
	go s.runMonitor(mCtx, m)
	log.Printf("monitor %s: started (interval %ds, url %s)", m.ID, m.IntervalSeconds, m.URL)
}

// loadActiveMonitors queries all monitors with is_active=true, joining users for alert emails.
func loadActiveMonitors(ctx context.Context, db *pgxpool.Pool) ([]Monitor, error) {
	rows, err := db.Query(ctx,
		`SELECT m.id, m.name, m.url, m.interval_seconds, u.email
		 FROM monitors m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.is_active = true`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		if err := rows.Scan(&m.ID, &m.Name, &m.URL, &m.IntervalSeconds, &m.UserEmail); err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

// pingURL sends an HTTP GET with a 10-second timeout.
// is_up is true only for 2xx status codes.
// ResponseTimeMs is always recorded, including on timeout or connection error.
func pingURL(url string) checkResult {
	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()
	resp, err := client.Get(url)
	elapsed := int(time.Since(start).Milliseconds())

	if err != nil {
		return checkResult{
			IsUp:           false,
			ResponseTimeMs: elapsed,
			ErrorMessage:   err.Error(),
		}
	}
	resp.Body.Close()

	isUp := resp.StatusCode >= 200 && resp.StatusCode < 300
	result := checkResult{
		IsUp:           isUp,
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: elapsed,
	}
	if !isUp {
		result.ErrorMessage = http.StatusText(resp.StatusCode)
	}
	return result
}

// saveCheck inserts one row into the checks table.
// status_code is NULL when no HTTP response was received.
// error_message is NULL on success.
func saveCheck(ctx context.Context, db *pgxpool.Pool, monitorID string, r checkResult) error {
	var statusCode *int
	if r.StatusCode > 0 {
		statusCode = &r.StatusCode
	}
	var errMsg *string
	if r.ErrorMessage != "" {
		errMsg = &r.ErrorMessage
	}
	_, err := db.Exec(ctx,
		`INSERT INTO checks (monitor_id, status_code, response_time_ms, is_up, error_message)
		 VALUES ($1, $2, $3, $4, $5)`,
		monitorID, statusCode, r.ResponseTimeMs, r.IsUp, errMsg,
	)
	return err
}

// runMonitor is the per-monitor ticker loop. On each tick it pings the URL,
// persists the check result, and evaluates incident transitions.
func (s *Scheduler) runMonitor(ctx context.Context, m Monitor) {
	ticker := time.NewTicker(time.Duration(m.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("monitor %s: stopped", m.ID)
			return
		case <-ticker.C:
			result := pingURL(m.URL)
			if err := saveCheck(ctx, s.db, m.ID, result); err != nil {
				log.Printf("monitor %s: saveCheck: %v", m.ID, err)
			}
			s.evaluateIncident(ctx, m, result)
		}
	}
}

// evaluateIncident reads s.state to detect is_up transitions and writes to
// the incidents table accordingly:
//   - up → down: INSERT a new incident row with started_at = now()
//   - down → up: UPDATE the open incident with resolved_at and duration_seconds
func (s *Scheduler) evaluateIncident(ctx context.Context, m Monitor, r checkResult) {
	prev, known := s.state.Load(m.ID)
	s.state.Store(m.ID, r.IsUp)

	if !known {
		// First check for this monitor — no prior state, no transition possible.
		return
	}

	wasUp := prev.(bool)

	switch {
	case wasUp && !r.IsUp:
		// Transition: up → down. Open a new incident.
		if _, err := s.db.Exec(ctx,
			`INSERT INTO incidents (monitor_id, started_at) VALUES ($1, now())`,
			m.ID,
		); err != nil {
			log.Printf("monitor %s: open incident: %v", m.ID, err)
		} else {
			log.Printf("monitor %s: DOWN — incident opened", m.ID)
		}
		go s.sendAlert(m, false, r.ErrorMessage)

	case !wasUp && r.IsUp:
		// Transition: down → up. Close the open incident.
		if _, err := s.db.Exec(ctx,
			`UPDATE incidents
			 SET resolved_at      = now(),
			     duration_seconds = EXTRACT(EPOCH FROM (now() - started_at))::int
			 WHERE monitor_id = $1 AND resolved_at IS NULL`,
			m.ID,
		); err != nil {
			log.Printf("monitor %s: close incident: %v", m.ID, err)
		} else {
			log.Printf("monitor %s: UP — incident closed", m.ID)
		}
		go s.sendAlert(m, true, "")
	}
}

// sendAlert emails the monitor owner about a down/up transition.
// It is a no-op when s.mailer is nil or the monitor has no email.
func (s *Scheduler) sendAlert(m Monitor, isUp bool, errMsg string) {
	if s.mailer == nil || m.UserEmail == "" {
		return
	}

	var subject, body string
	if !isUp {
		reason := errMsg
		if reason == "" {
			reason = "non-2xx response"
		}
		subject = fmt.Sprintf("[PingWatch] DOWN: %s", m.Name)
		body = fmt.Sprintf("Your monitor \"%s\" (%s) is DOWN.\n\nReason: %s", m.Name, m.URL, reason)
	} else {
		subject = fmt.Sprintf("[PingWatch] UP: %s", m.Name)
		body = fmt.Sprintf("Your monitor \"%s\" (%s) is back UP.", m.Name, m.URL)
	}

	if err := s.mailer.Send(m.UserEmail, subject, body); err != nil {
		log.Printf("monitor %s: send alert to %s: %v", m.ID, m.UserEmail, err)
	}
}
