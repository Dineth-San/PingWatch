package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaludineth/pingwatch/worker/internal/mailer"
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

	var m *mailer.Mailer
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost != "" {
		m = mailer.New(
			smtpHost,
			os.Getenv("SMTP_PORT"),
			os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASS"),
			os.Getenv("SMTP_FROM"),
		)
		log.Printf("SMTP alerts enabled via %s", smtpHost)
	} else {
		log.Println("SMTP_HOST not set — email alerts disabled")
	}

	s := scheduler.New(pool, m)
	if err := s.Start(ctx); err != nil {
		log.Fatalf("scheduler start: %v", err)
	}

	<-ctx.Done()
	log.Println("Worker shutting down")
}
