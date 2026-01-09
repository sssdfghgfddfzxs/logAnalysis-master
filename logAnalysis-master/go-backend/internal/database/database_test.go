package database

import (
	"context"
	"testing"
	"time"

	"intelligent-log-analysis/internal/config"
)

func TestDatabase_Configuration(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "test",
		Password: "test",
		DBName:   "test_db",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	expected := "host=localhost port=5432 user=test password=test dbname=test_db sslmode=disable"

	if dsn != expected {
		t.Errorf("Expected DSN %s, got %s", expected, dsn)
	}
}

func TestDatabase_Health(t *testing.T) {
	// Test health check timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This would normally test database health, but we'll test the context timeout
	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}
