package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaludineth/pingwatch/api/internal/handler"
	"github.com/kaludineth/pingwatch/api/internal/middleware"
	"github.com/kaludineth/pingwatch/api/internal/store"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL env var is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET env var is required")
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	migrateURL := strings.Replace(dbURL, "postgres://", "pgx5://", 1)
	m, err := migrate.New("file://db/migrations", migrateURL)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("Migrations applied")

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("pgxpool connect: %v", err)
	}
	defer pool.Close()

	s := store.New(pool)
	authH := handler.NewAuthHandler(s, jwtSecret)
	monitorH := handler.NewMonitorHandler(s)
	checkH := handler.NewCheckHandler(s)

	r := chi.NewRouter()

	// Global middleware — applied to every request in this order:
	//   1. Logger    — outermost, captures final status and duration
	//   2. CORS      — sets headers before any short-circuit (incl. rate limit)
	//   3. RateLimit — rejects excess traffic with 429 before hitting handlers
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(frontendURL))
	r.Use(middleware.RateLimit(100))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(context.Background()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"db unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth — no JWT required
	r.Post("/api/auth/register", authH.Register)
	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/logout", authH.Logout)

	// Protected routes — all require a valid JWT cookie
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtSecret))

		r.Get("/api/monitors", monitorH.List)
		r.Post("/api/monitors", monitorH.Create)
		r.Get("/api/monitors/{id}", monitorH.Get)
		r.Put("/api/monitors/{id}", monitorH.Update)
		r.Delete("/api/monitors/{id}", monitorH.Delete)

		r.Get("/api/monitors/{id}/checks", checkH.ListChecks)
		r.Get("/api/monitors/{id}/checks/latest", checkH.LatestCheck)
		r.Get("/api/monitors/{id}/incidents", checkH.ListIncidents)
		r.Get("/api/monitors/{id}/stats", checkH.GetStats)
	})

	log.Println("API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
