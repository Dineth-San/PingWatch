package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"

// UserIDFromCtx retrieves the authenticated user's ID from the request context.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// RequireAuth validates the JWT httpOnly cookie and injects user_id into ctx.
// Returns 401 JSON if the cookie is missing or invalid.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			userID, _ := claims["user_id"].(string)
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS sets cross-origin headers for the configured allowed origin.
// Always sets credentials and allows OPTIONS preflights before any short-circuit.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowedOrigin == "*" || strings.EqualFold(origin, allowedOrigin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- Rate limiting ---

type ipBucket struct {
	mu       sync.Mutex
	tokens   float64
	lastSeen time.Time
}

// RateLimit enforces requestsPerMinute per client IP using a token bucket.
// Exceeded requests receive 429 with a Retry-After header (seconds).
func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	var buckets sync.Map
	capacity := float64(requestsPerMinute)
	rate := capacity / 60.0 // tokens restored per second

	// Background goroutine: sweep buckets idle for more than 2 minutes.
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			buckets.Range(func(k, v any) bool {
				b := v.(*ipBucket)
				b.mu.Lock()
				idle := time.Since(b.lastSeen) > 2*time.Minute
				b.mu.Unlock()
				if idle {
					buckets.Delete(k)
				}
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			val, _ := buckets.LoadOrStore(ip, &ipBucket{tokens: capacity, lastSeen: time.Now()})
			b := val.(*ipBucket)

			b.mu.Lock()
			now := time.Now()
			elapsed := now.Sub(b.lastSeen).Seconds()
			b.tokens = min(capacity, b.tokens+elapsed*rate)
			b.lastSeen = now

			if b.tokens < 1 {
				retryAfter := int(math.Ceil((1 - b.tokens) / rate))
				b.mu.Unlock()
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}
			b.tokens--
			b.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the real client IP, respecting common proxy headers.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- Request logging ---

// statusRecorder wraps ResponseWriter to capture the written status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) statusCode() int {
	if sr.status == 0 {
		return http.StatusOK // never explicitly written → implicit 200
	}
	return sr.status
}

// Logger logs each request as a JSON line to stdout:
// {"time":"…","method":"GET","path":"/api/…","status":200,"duration_ms":12}
func Logger() func(http.Handler) http.Handler {
	type entry struct {
		Time       string `json:"time"`
		Method     string `json:"method"`
		Path       string `json:"path"`
		Status     int    `json:"status"`
		DurationMs int64  `json:"duration_ms"`
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			e := entry{
				Time:       start.UTC().Format(time.RFC3339),
				Method:     r.Method,
				Path:       r.URL.Path,
				Status:     rec.statusCode(),
				DurationMs: time.Since(start).Milliseconds(),
			}
			b, _ := json.Marshal(e)
			fmt.Fprintln(os.Stdout, string(b))
		})
	}
}

// writeJSON is shared by all middleware that need to write JSON error responses.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
