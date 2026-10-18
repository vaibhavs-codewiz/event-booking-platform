package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type healthResponse struct {
	Status       string     `json:"status"`
	Service      string     `json:"service"`
	Time         string     `json:"time"`
	Dependencies *depsStatus `json:"dependencies,omitempty"`
}

type depsStatus struct {
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

// healthLive is a liveness probe: it only confirms the process is up and able
// to answer HTTP. It deliberately avoids touching dependencies so a transient
// datastore outage does not cause the process to be killed by an orchestrator.
func (s *Server) healthLive(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{
		Status:  statusLabel(http.StatusOK),
		Service: serviceName,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}

// healthReady is a readiness probe: it confirms the process can serve traffic
// by verifying both Postgres and Redis respond. Returns 503 if any dependency
// is unreachable, with the per-dependency status included for diagnostics.
func (s *Server) healthReady(c *gin.Context) {
	ctx := c.Request.Context()
	deps := depsStatus{
		Postgres: pingPostgres(ctx, s.pg),
		Redis:    pingRedis(ctx, s.rdb),
	}

	code := http.StatusOK
	if deps.Postgres != statusOK || deps.Redis != statusOK {
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, healthResponse{
		Status:       statusLabel(code),
		Service:      serviceName,
		Time:         time.Now().UTC().Format(time.RFC3339),
		Dependencies: &deps,
	})
}

const statusOK = "ok"

func pingPostgres(ctx context.Context, pg *pgxpool.Pool) string {
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pg.Ping(pctx); err != nil {
		return "down"
	}
	return statusOK
}

func pingRedis(ctx context.Context, rdb *redis.Client) string {
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := rdb.Ping(pctx).Err(); err != nil {
		return "down"
	}
	return statusOK
}

func statusLabel(code int) string {
	if code == http.StatusOK {
		return statusOK
	}
	return "degraded"
}
