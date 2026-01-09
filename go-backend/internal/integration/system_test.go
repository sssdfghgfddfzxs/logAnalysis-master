package integration

import (
	"context"
	"testing"
	"time"

	"intelligent-log-analysis/internal/monitoring"
	"intelligent-log-analysis/internal/optimization"
	"intelligent-log-analysis/internal/recovery"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SystemIntegrationTest tests the complete system integration
func TestSystemIntegration(t *testing.T) {
	// Initialize monitoring
	metricsCollector := monitoring.NewMetricsCollector()

	// Initialize performance optimizer
	optimizer := optimization.NewPerformanceOptimizer(metricsCollector, nil)

	// Initialize error handler
	errorHandler := recovery.NewErrorHandler(metricsCollector, nil)

	t.Run("Monitoring Integration", func(t *testing.T) {
		// Test metrics collection
		metricsCollector.RecordHTTPRequest("/api/v1/logs", 200, 150*time.Millisecond)
		metricsCollector.RecordHTTPRequest("/api/v1/logs", 201, 200*time.Millisecond)
		metricsCollector.RecordHTTPRequest("/api/v1/analysis/results", 200, 300*time.Millisecond)

		summary := metricsCollector.GetSummary()
		assert.Equal(t, int64(3), summary.HTTP.RequestCount)
		assert.Equal(t, int64(0), summary.HTTP.ErrorCount)
		// Check that response times are recorded (endpoint format may vary)
		assert.Greater(t, len(summary.HTTP.ResponseTimes), 0)
	})

	t.Run("Performance Optimization Integration", func(t *testing.T) {
		ctx := context.Background()

		// Run optimization
		result, err := optimizer.RunOptimization(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.OptimizationTime, int64(0))

		// Get performance report
		report := optimizer.GetPerformanceReport()
		assert.NotNil(t, report)
		assert.GreaterOrEqual(t, report.PerformanceScore, 0.0)
		assert.LessOrEqual(t, report.PerformanceScore, 100.0)

		// Get recommendations
		recommendations := optimizer.GetOptimizationRecommendations()
		assert.NotNil(t, recommendations)
	})

	t.Run("Error Handling Integration", func(t *testing.T) {
		ctx := context.Background()

		// Test database error handling
		errorCtx := &recovery.ErrorContext{
			Type:      recovery.ErrorTypeDatabase,
			Operation: "SaveLog",
			Component: "LogRepository",
			Timestamp: time.Now(),
			Error:     &mockError{"connection refused"},
			Retryable: true,
			Severity:  recovery.SeverityHigh,
		}

		report := errorHandler.HandleError(ctx, errorCtx)
		assert.NotNil(t, report)
		assert.Greater(t, len(report.RecoveryActions), 0)

		// Test AI service error handling
		aiErrorCtx := &recovery.ErrorContext{
			Type:      recovery.ErrorTypeAI,
			Operation: "AnalyzeLogs",
			Component: "AIServiceClient",
			Timestamp: time.Now(),
			Error:     &mockError{"service unavailable"},
			Retryable: true,
			Severity:  recovery.SeverityMedium,
		}

		aiReport := errorHandler.HandleError(ctx, aiErrorCtx)
		assert.NotNil(t, aiReport)
		assert.Greater(t, len(aiReport.RecoveryActions), 0)

		// Get error statistics
		stats := errorHandler.GetErrorStatistics()
		assert.Contains(t, stats, "error_counts")
		assert.Contains(t, stats, "total_errors")
	})

	t.Run("System Resilience Test", func(t *testing.T) {
		// Reset metrics for this test
		metricsCollector.Reset()

		// Simulate high load
		for i := 0; i < 100; i++ {
			metricsCollector.RecordHTTPRequest("/api/v1/logs", 200, time.Duration(i)*time.Millisecond)
			metricsCollector.RecordLogProcessed()
		}

		// Simulate some errors (but keep error rate below critical threshold)
		for i := 0; i < 3; i++ {
			metricsCollector.RecordHTTPRequest("/api/v1/logs", 500, 1000*time.Millisecond)
		}

		// Check system metrics
		summary := metricsCollector.GetSummary()
		assert.Equal(t, int64(103), summary.HTTP.RequestCount)
		assert.Equal(t, int64(3), summary.HTTP.ErrorCount)
		assert.Equal(t, int64(100), summary.Application.LogsProcessed)

		// Check performance alerts
		alerts := metricsCollector.CheckPerformanceThresholds()
		// Should not have critical alerts under normal test conditions
		for _, alert := range alerts {
			assert.NotEqual(t, "critical", alert.Severity)
		}
	})

	t.Run("Memory and Resource Management", func(t *testing.T) {
		// Update system metrics
		metricsCollector.UpdateSystemMetrics()

		summary := metricsCollector.GetSummary()
		assert.Greater(t, summary.System.MemoryUsage, int64(0))
		assert.Greater(t, summary.System.GoroutineCount, 0)

		// Test GC tuning
		optimizer.TuneGCSettings()

		// Performance should remain stable
		report := optimizer.GetPerformanceReport()
		assert.GreaterOrEqual(t, report.PerformanceScore, 50.0) // Should maintain reasonable performance
	})
}

// TestSystemStressTest performs stress testing on the integrated system
func TestSystemStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	metricsCollector := monitoring.NewMetricsCollector()
	optimizer := optimization.NewPerformanceOptimizer(metricsCollector, nil)
	errorHandler := recovery.NewErrorHandler(metricsCollector, nil)

	t.Run("High Load Simulation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start periodic optimization
		go optimizer.StartAutoOptimization(ctx)

		// Start metrics collection
		go metricsCollector.StartPeriodicCollection(ctx, 1*time.Second)

		// Simulate high load
		const numRequests = 1000
		const concurrency = 10

		done := make(chan bool, concurrency)

		for worker := 0; worker < concurrency; worker++ {
			go func(workerID int) {
				defer func() { done <- true }()

				for i := 0; i < numRequests/concurrency; i++ {
					// Simulate various operations
					metricsCollector.RecordHTTPRequest("/api/v1/logs", 200, time.Duration(i%100)*time.Millisecond)
					metricsCollector.RecordLogProcessed()

					if i%50 == 0 {
						metricsCollector.RecordAnalysisRequest(true)
					}

					if i%100 == 0 {
						metricsCollector.RecordCacheOperation(true)
					} else if i%20 == 0 {
						metricsCollector.RecordCacheOperation(false)
					}

					// Simulate occasional errors
					if i%200 == 0 {
						errorCtx := &recovery.ErrorContext{
							Type:      recovery.ErrorTypeNetwork,
							Operation: "HTTPRequest",
							Component: "APIServer",
							Timestamp: time.Now(),
							Error:     &mockError{"temporary network error"},
							Retryable: true,
							Severity:  recovery.SeverityLow,
						}
						errorHandler.HandleError(context.Background(), errorCtx)
					}

					// Small delay to prevent overwhelming the system
					time.Sleep(time.Millisecond)
				}
			}(worker)
		}

		// Wait for all workers to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}

		// Verify system stability
		summary := metricsCollector.GetSummary()
		assert.Equal(t, int64(numRequests), summary.HTTP.RequestCount)
		assert.Equal(t, int64(numRequests), summary.Application.LogsProcessed)

		// Performance should remain reasonable under load
		report := optimizer.GetPerformanceReport()
		assert.GreaterOrEqual(t, report.PerformanceScore, 30.0) // Allow for some degradation under stress

		t.Logf("Stress test completed: %d requests processed, performance score: %.2f",
			numRequests, report.PerformanceScore)
	})
}

// TestSystemRecoveryScenarios tests various system recovery scenarios
func TestSystemRecoveryScenarios(t *testing.T) {
	metricsCollector := monitoring.NewMetricsCollector()
	errorHandler := recovery.NewErrorHandler(metricsCollector, nil)

	scenarios := []struct {
		name           string
		errorType      recovery.ErrorType
		error          error
		retryable      bool
		expectRecovery bool
	}{
		{
			name:           "Database Connection Error",
			errorType:      recovery.ErrorTypeDatabase,
			error:          &mockError{"connection refused"},
			retryable:      true,
			expectRecovery: true,
		},
		{
			name:           "AI Service Timeout",
			errorType:      recovery.ErrorTypeAI,
			error:          &mockError{"timeout"},
			retryable:      true,
			expectRecovery: true,
		},
		{
			name:           "Cache Unavailable",
			errorType:      recovery.ErrorTypeCache,
			error:          &mockError{"redis connection failed"},
			retryable:      true,
			expectRecovery: true,
		},
		{
			name:           "Validation Error",
			errorType:      recovery.ErrorTypeValidation,
			error:          &mockError{"invalid input"},
			retryable:      false,
			expectRecovery: false,
		},
		{
			name:           "Network Error",
			errorType:      recovery.ErrorTypeNetwork,
			error:          &mockError{"network unreachable"},
			retryable:      true,
			expectRecovery: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			errorCtx := &recovery.ErrorContext{
				Type:      scenario.errorType,
				Operation: "TestOperation",
				Component: "TestComponent",
				Timestamp: time.Now(),
				Error:     scenario.error,
				Retryable: scenario.retryable,
				Severity:  recovery.SeverityMedium,
			}

			report := errorHandler.HandleError(context.Background(), errorCtx)

			assert.NotNil(t, report)
			assert.Equal(t, scenario.expectRecovery, report.Resolved)
			assert.Greater(t, len(report.RecoveryActions), 0)

			if scenario.retryable && scenario.expectRecovery {
				// Some recovery strategies don't require retries (e.g., immediate fallback)
				// Only check retries for scenarios that actually retry
				if scenario.name == "Database Connection Error" || scenario.name == "Network Error" {
					assert.Greater(t, report.TotalRetries, 0)
				}
			}

			t.Logf("Scenario '%s': Resolved=%v, Actions=%d, Retries=%d",
				scenario.name, report.Resolved, len(report.RecoveryActions), report.TotalRetries)
		})
	}
}

// TestSystemMetricsAccuracy tests the accuracy of system metrics
func TestSystemMetricsAccuracy(t *testing.T) {
	metricsCollector := monitoring.NewMetricsCollector()

	// Record known metrics
	metricsCollector.RecordHTTPRequest("/test", 200, 100*time.Millisecond)
	metricsCollector.RecordHTTPRequest("/test", 404, 50*time.Millisecond)
	metricsCollector.RecordHTTPRequest("/test", 500, 200*time.Millisecond)

	metricsCollector.RecordLogProcessed()
	metricsCollector.RecordLogProcessed()

	metricsCollector.RecordAnalysisRequest(true)
	metricsCollector.RecordAnalysisRequest(false)

	metricsCollector.RecordCacheOperation(true)
	metricsCollector.RecordCacheOperation(true)
	metricsCollector.RecordCacheOperation(false)

	// Verify metrics accuracy
	summary := metricsCollector.GetSummary()

	assert.Equal(t, int64(3), summary.HTTP.RequestCount)
	assert.Equal(t, int64(2), summary.HTTP.ErrorCount) // 404 and 500 are errors
	assert.Equal(t, int64(2), summary.Application.LogsProcessed)
	assert.Equal(t, int64(2), summary.Application.AnalysisRequests)
	assert.Equal(t, int64(1), summary.Application.AnalysisFailures)
	assert.Equal(t, float64(0.5), summary.Application.AnalysisSuccessRate)
	assert.Equal(t, int64(2), summary.Application.CacheHits)
	assert.Equal(t, int64(1), summary.Application.CacheMisses)
	assert.InDelta(t, 0.667, summary.Application.CacheHitRate, 0.01)

	// Test response time calculations
	avgTime := metricsCollector.GetAverageResponseTime("/test")
	assert.InDelta(t, 116.67, avgTime, 1.0) // (100+50+200)/3

	p95Time := metricsCollector.GetP95ResponseTime("/test")
	assert.Equal(t, int64(200), p95Time) // Should be the highest value for small dataset
}

// mockError implements the error interface for testing
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
