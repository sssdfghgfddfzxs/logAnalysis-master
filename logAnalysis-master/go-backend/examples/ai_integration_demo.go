//go:build examples
// +build examples

package main

import (
	"context"
	"fmt"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/service"
)

// Example demonstrating Go backend integration with AI module
func main() {
	fmt.Println("=== Go Backend AI Integration Example ===")

	// Create a log service (in real usage, this would be initialized with actual dependencies)
	logService := &service.LogService{}

	// Example 1: Log validation
	fmt.Println("\n1. Testing log validation...")

	validLog := &models.LogEntry{
		Message:   "Database connection established successfully",
		Level:     "INFO",
		Source:    "user-service",
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"host":     "server-01",
			"database": "users_db",
		},
	}

	invalidLog := &models.LogEntry{
		Message: "Invalid log entry",
		Level:   "INVALID_LEVEL", // This will fail validation
		Source:  "test-service",
	}

	// Test validation
	if err := validateLogEntry(logService, validLog); err != nil {
		fmt.Printf("❌ Valid log failed validation: %v\n", err)
	} else {
		fmt.Printf("✅ Valid log passed validation\n")
	}

	if err := validateLogEntry(logService, invalidLog); err != nil {
		fmt.Printf("✅ Invalid log correctly rejected: %v\n", err)
	} else {
		fmt.Printf("❌ Invalid log incorrectly accepted\n")
	}

	// Example 2: Error handling demonstration
	fmt.Println("\n2. Testing error handling...")

	testErrors := []error{
		fmt.Errorf("connection refused"),
		fmt.Errorf("timeout exceeded"),
		fmt.Errorf("invalid argument"),
		fmt.Errorf("deadlock detected"),
		fmt.Errorf("redis connection failed"),
	}

	for _, err := range testErrors {
		retryable := isRetryableError(logService, err)
		dbRetryable := isDatabaseRetryableError(logService, err)
		schedRetryable := isSchedulingRetryableError(logService, err)

		fmt.Printf("Error: %v\n", err)
		fmt.Printf("  - AI Retryable: %v\n", retryable)
		fmt.Printf("  - DB Retryable: %v\n", dbRetryable)
		fmt.Printf("  - Scheduling Retryable: %v\n", schedRetryable)
	}

	// Example 3: Analysis metrics
	fmt.Println("\n3. Getting analysis metrics...")

	ctx := context.Background()
	period := 24 * time.Hour

	metrics, err := logService.GetAnalysisMetrics(ctx, period)
	if err != nil {
		fmt.Printf("❌ Failed to get metrics: %v\n", err)
	} else {
		fmt.Printf("✅ Analysis metrics retrieved:\n")
		for key, value := range metrics {
			fmt.Printf("  - %s: %v\n", key, value)
		}
	}

	fmt.Println("\n=== Integration Example Complete ===")
}

// Helper functions to access private methods for demonstration

func validateLogEntry(service *service.LogService, log *models.LogEntry) error {
	// This would normally be called internally by CreateLog
	// For demonstration, we'll simulate the validation logic
	if log == nil {
		return fmt.Errorf("log entry cannot be nil")
	}
	if log.Message == "" {
		return fmt.Errorf("log message cannot be empty")
	}
	if log.Level == "" {
		return fmt.Errorf("log level cannot be empty")
	}
	if log.Source == "" {
		return fmt.Errorf("log source cannot be empty")
	}

	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
	}

	if !validLevels[log.Level] {
		return fmt.Errorf("invalid log level: %s", log.Level)
	}

	return nil
}

func isRetryableError(service *service.LogService, err error) bool {
	// Simulate the retry logic for AI errors
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"deadline exceeded",
		"unavailable",
		"resource exhausted",
		"temporary failure",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	return false
}

func isDatabaseRetryableError(service *service.LogService, err error) bool {
	// Simulate the retry logic for database errors
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"deadlock",
		"lock wait timeout",
		"temporary failure",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	return false
}

func isSchedulingRetryableError(service *service.LogService, err error) bool {
	// Simulate the retry logic for scheduling errors
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"redis",
		"temporary failure",
		"queue full",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
