package service

import (
	"context"
	"testing"
	"time"

	"intelligent-log-analysis/internal/models"
)

// TestLogServiceAIIntegration tests the integration between LogService and AI module
func TestLogServiceAIIntegration(t *testing.T) {
	// Create a mock log service (without actual AI client for testing)
	service := &LogService{}

	// Test error handling methods
	t.Run("TestRetryableErrorDetection", func(t *testing.T) {
		testCases := []struct {
			name     string
			err      error
			expected bool
		}{
			{"nil error", nil, false},
			{"connection refused", &mockError{"connection refused"}, true},
			{"timeout error", &mockError{"timeout"}, true},
			{"invalid argument", &mockError{"invalid argument"}, false},
			{"unknown error", &mockError{"some random error"}, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := service.isRetryableError(tc.err)
				if result != tc.expected {
					t.Errorf("Expected %v, got %v for error: %v", tc.expected, result, tc.err)
				}
			})
		}
	})

	t.Run("TestDatabaseRetryableErrorDetection", func(t *testing.T) {
		testCases := []struct {
			name     string
			err      error
			expected bool
		}{
			{"nil error", nil, false},
			{"connection refused", &mockError{"connection refused"}, true},
			{"deadlock", &mockError{"deadlock detected"}, true},
			{"syntax error", &mockError{"syntax error"}, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := service.isDatabaseRetryableError(tc.err)
				if result != tc.expected {
					t.Errorf("Expected %v, got %v for error: %v", tc.expected, result, tc.err)
				}
			})
		}
	})

	t.Run("TestSchedulingRetryableErrorDetection", func(t *testing.T) {
		testCases := []struct {
			name     string
			err      error
			expected bool
		}{
			{"nil error", nil, false},
			{"redis connection error", &mockError{"redis connection failed"}, true},
			{"queue full", &mockError{"queue full"}, true},
			{"invalid task type", &mockError{"invalid task type"}, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := service.isSchedulingRetryableError(tc.err)
				if result != tc.expected {
					t.Errorf("Expected %v, got %v for error: %v", tc.expected, result, tc.err)
				}
			})
		}
	})
}

// TestLogValidation tests log entry validation
func TestLogValidation(t *testing.T) {
	service := &LogService{}

	testCases := []struct {
		name        string
		log         *models.LogEntry
		expectError bool
	}{
		{
			name:        "nil log",
			log:         nil,
			expectError: true,
		},
		{
			name: "empty message",
			log: &models.LogEntry{
				Level:  "ERROR",
				Source: "test-service",
			},
			expectError: true,
		},
		{
			name: "empty level",
			log: &models.LogEntry{
				Message: "Test message",
				Source:  "test-service",
			},
			expectError: true,
		},
		{
			name: "empty source",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "ERROR",
			},
			expectError: true,
		},
		{
			name: "invalid level",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "INVALID",
				Source:  "test-service",
			},
			expectError: true,
		},
		{
			name: "valid log",
			log: &models.LogEntry{
				Message: "Test message",
				Level:   "ERROR",
				Source:  "test-service",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.validateLogEntry(tc.log)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestAnalysisMetrics tests the analysis metrics functionality
func TestAnalysisMetrics(t *testing.T) {
	service := &LogService{}

	ctx := context.Background()
	period := 24 * time.Hour

	metrics, err := service.GetAnalysisMetrics(ctx, period)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that basic metrics are present
	expectedKeys := []string{"ai_client_available", "queue_available", "period", "analysis_performance"}
	for _, key := range expectedKeys {
		if _, exists := metrics[key]; !exists {
			t.Errorf("Expected metric key %s not found", key)
		}
	}

	// Check that AI client is marked as not available (since we don't have one in test)
	if aiAvailable, ok := metrics["ai_client_available"].(bool); !ok || aiAvailable {
		t.Error("Expected ai_client_available to be false")
	}

	// Check that queue is marked as not available (since we don't have one in test)
	if queueAvailable, ok := metrics["queue_available"].(bool); !ok || queueAvailable {
		t.Error("Expected queue_available to be false")
	}
}

// mockError is a simple error implementation for testing
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
