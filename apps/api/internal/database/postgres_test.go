package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database"
)

func TestNew_InvalidConfig(t *testing.T) {
	cfg := &config.Config{
		DBHost:        "invalid host with spaces @#$",
		DBPort:        5432,
		DBUser:        "user",
		DBPassword:    "pass",
		DBName:        "db",
		DBSSLMode:     "disable",
		DBMaxConns:    10,
		DBMinConns:    2,
		DBConnTimeout: 100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := database.New(ctx, cfg)
	if err == nil {
		t.Fatalf("expected error with invalid host config, got nil")
	}
}

func TestNew_ContextTimeoutOnUnreachableHost(t *testing.T) {
	// Points to non-routable test IP
	cfg := &config.Config{
		DBHost:        "192.0.2.1", // TEST-NET-1 (non-routable)
		DBPort:        5432,
		DBUser:        "user",
		DBPassword:    "pass",
		DBName:        "db",
		DBSSLMode:     "disable",
		DBMaxConns:    2,
		DBMinConns:    1,
		DBConnTimeout: 50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := database.New(ctx, cfg)
	duration := time.Since(start)

	if err == nil {
		t.Fatalf("expected connection timeout error, got nil")
	}
	if duration > 1*time.Second {
		t.Errorf("expected connection attempt to fail quickly within timeout, took %v", duration)
	}
}
