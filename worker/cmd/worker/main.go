package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaludineth/pingwatch/worker/internal/scheduler"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL env var is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("pgxpool connect: %v", err)
	}
	defer pool.Close()

	s := scheduler.New(pool)
	if err := s.Start(ctx); err != nil {
		log.Fatalf("scheduler start: %v", err)
	}

	<-ctx.Done()
	log.Println("Worker shutting down")
}
