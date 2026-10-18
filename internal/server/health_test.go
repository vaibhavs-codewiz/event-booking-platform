package server

import (
	"net/http"
	"testing"
)

func TestStatusLabel(t *testing.T) {
	if got := statusLabel(http.StatusOK); got != "ok" {
		t.Fatalf("statusLabel(200): expected ok, got %q", got)
	}
	if got := statusLabel(http.StatusServiceUnavailable); got != "degraded" {
		t.Fatalf("statusLabel(503): expected degraded, got %q", got)
	}
}
