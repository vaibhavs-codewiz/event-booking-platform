// Package server wires the HTTP layer (Gin) together with its dependencies and
// exposes the routes that are implemented in each phase.
//
// Phase 1: only the health endpoints are registered. Feature routes
// (auth, users, events, bookings) are added in later phases.
package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/cactro/event-booking-platform/internal/config"
)

const serviceName = "event-booking-platform"

// Server bundles the HTTP server with the dependencies it needs.
type Server struct {
	cfg    *config.Config
	pg     *pgxpool.Pool
	rdb    *redis.Client
	router *gin.Engine
	http   *http.Server
}

// New constructs the server, registers routes and configures the underlying
// http.Server with sensible timeouts.
func New(cfg *config.Config, pg *pgxpool.Pool, rdb *redis.Client) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	s := &Server{
		cfg:    cfg,
		pg:     pg,
		rdb:    rdb,
		router: router,
	}

	s.registerRoutes()
	s.http = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTPWriteTimeoutSec) * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

// registerRoutes mounts the routes available in the current phase.
func (s *Server) registerRoutes() {
	health := s.router.Group("/health")
	{
		health.GET("", s.healthLive)
		health.GET("/ready", s.healthReady)
	}
}

// Router returns the configured Gin engine. It is exported primarily for
// testing and for future wiring of feature route groups.
func (s *Server) Router() *gin.Engine { return s.router }

// Start serves HTTP requests on the configured address.
func (s *Server) Start() error { return s.http.ListenAndServe() }

// Shutdown gracefully stops the HTTP server, completing in-flight requests
// up to the context deadline.
func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }
