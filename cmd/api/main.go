// Command api is the HTTP entry point for the Event Booking Platform.
//
// Phase 1 responsibility: load configuration, connect to PostgreSQL and Redis,
// start the Gin HTTP server and expose the health endpoints. It shuts down
// gracefully on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cactro/event-booking-platform/internal/config"
	"github.com/cactro/event-booking-platform/internal/db"
	"github.com/cactro/event-booking-platform/internal/server"
)

func main() {
	logger := log.New(os.Stdout, "api ", log.LstdFlags|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("config: %v", err)
	}

	// rootCtx is cancelled on SIGINT/SIGTERM, triggering graceful shutdown.
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pg, err := db.NewPostgres(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("postgres: %v", err)
	}
	defer pg.Close()
	logger.Println("connected to postgres")

	rdb, err := db.NewRedis(rootCtx, cfg.RedisURL)
	if err != nil {
		logger.Fatalf("redis: %v", err)
	}
	defer rdb.Close()
	logger.Println("connected to redis")

	srv := server.New(cfg, pg, rdb)

	go func() {
		logger.Printf("listening on :%s", cfg.Port)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server: %v", err)
		}
	}()

	<-rootCtx.Done()
	logger.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("graceful shutdown: %v", err)
	}
	logger.Println("server stopped")
}
