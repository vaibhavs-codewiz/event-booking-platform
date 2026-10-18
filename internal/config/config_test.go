package config

import "testing"

func TestGetEnvDefaultsToFallback(t *testing.T) {
	if got := getEnv("MISSING_VAR_xyz_nope", "fallback"); got != "fallback" {
		t.Fatalf("getEnv: expected fallback, got %q", got)
	}
}

func TestGetEnvIntDefaultsToFallback(t *testing.T) {
	if got := getEnvInt("MISSING_INT_xyz_nope", 42); got != 42 {
		t.Fatalf("getEnvInt: expected 42, got %d", got)
	}
}

func TestGetEnvReadsRealEnvironment(t *testing.T) {
	t.Setenv("TEST_VAR_xyz_real", "real")
	if got := getEnv("TEST_VAR_xyz_real", "fallback"); got != "real" {
		t.Fatalf("getEnv: expected real, got %q", got)
	}

	t.Setenv("TEST_INT_xyz_real", "7")
	if got := getEnvInt("TEST_INT_xyz_real", 42); got != 7 {
		t.Fatalf("getEnvInt: expected 7, got %d", got)
	}
}

func TestGetEnvIntIgnoresInvalidValue(t *testing.T) {
	t.Setenv("TEST_INT_xyz_bad", "not-a-number")
	if got := getEnvInt("TEST_INT_xyz_bad", 42); got != 42 {
		t.Fatalf("getEnvInt: expected fallback 42, got %d", got)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	// Ensure required vars are unset/empty for this test. t.Setenv restores
	// their prior values automatically when the test ends.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
	}
}

func TestLoadRequiresRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when REDIS_URL is missing, got nil")
	}
}

func TestLoadSucceedsWithRequiredVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" {
		t.Fatal("required vars not populated")
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}
}
