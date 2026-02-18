package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/devaloi/shrink/internal/config"
	"github.com/devaloi/shrink/internal/handler"
	"github.com/devaloi/shrink/internal/middleware"
	"github.com/devaloi/shrink/internal/repository"
	"github.com/devaloi/shrink/internal/service"
)

// Server timeout constants.
const (
	ReadTimeout     = 15 * time.Second
	WriteTimeout    = 15 * time.Second
	IdleTimeout     = 60 * time.Second
	ShutdownTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log.Printf("Starting shrink server...")
	log.Printf("Port: %d", cfg.Port)
	log.Printf("Database: %s", cfg.DatabaseURL)
	log.Printf("Base URL: %s", cfg.BaseURL)
	log.Printf("Rate limit: %.0f req/s, burst: %d", cfg.RateLimit, cfg.RateBurst)

	db, err := sql.Open("sqlite3", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("Error closing database: %v", cerr)
		}
	}()

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: could not enable WAL mode: %v", err)
	}

	repo := repository.NewSQLite(db)
	if err := repo.Migrate(); err != nil {
		return err
	}

	svc := service.NewURLService(repo, cfg.BaseURL)
	h := handler.New(svc, db)

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateBurst)

	chain := middleware.Chain(
		middleware.RequestID,
		middleware.Logging,
		middleware.Recovery,
		middleware.CORS(middleware.DefaultCORSConfig()),
		rateLimiter.Middleware,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/shorten", h.CreateShortURL)
	mux.HandleFunc("GET /api/health", h.HealthCheck)
	mux.HandleFunc("GET /api/stats", h.GlobalStats)
	mux.HandleFunc("GET /api/urls/{code}", h.GetStats)
	mux.HandleFunc("GET /{code}", h.Redirect)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      chain(mux),
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	go func() {
		log.Printf("Server listening on %s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server stopped")
	return nil
}
