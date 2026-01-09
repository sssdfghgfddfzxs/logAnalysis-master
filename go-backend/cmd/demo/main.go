package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"intelligent-log-analysis/internal/logging"
	"intelligent-log-analysis/internal/monitoring"
	"intelligent-log-analysis/internal/optimization"
	"intelligent-log-analysis/internal/recovery"
)

func main() {
	fmt.Println("=== Intelligent Log Analysis System - Integration Demo ===")

	// Initialize logging
	logConfig := &logging.LoggerConfig{
		Level:     logging.INFO,
		Component: "demo",
		Output:    "stdout",
		Format:    "json",
	}
	logger := logging.NewLogger(logConfig)

	// Initialize monitoring
	metricsCollector := monitoring.NewMetricsCollector()

	// Initialize performance optimizer
	optimizer := optimization.NewPerformanceOptimizer(metricsCollector, nil)

	// Initialize error handler
	errorHandler := recovery.NewErrorHandler(metricsCollector, nil)

	ctx := context.Background()

	// Start periodic metrics collection
	go metricsCollector.StartPeriodicCollection(ctx, 2*time.Second)

	logger.Info("System components initialized successfully")

	// Simulate some system activity
	fmt.Println("\n1. Simulating HTTP requests...")
	for i := 0; i < 10; i++ {
		endpoint := "/api/v1/logs"
		statusCode := 200
		if i%5 == 0 {
			statusCode = 500 // Simulate some errors
		}

		responseTime := time.Duration(50+i*10) * time.Millisecond
		metricsCollector.RecordHTTPRequest(endpoint, statusCode, responseTime)

		if i%3 == 0 {
			metricsCollector.RecordLogProcessed()
			metricsCollector.RecordAnalysisRequest(true)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Get and display metrics
	fmt.Println("\n2. Current System Metrics:")
	summary := metricsCollector.GetSummary()
	fmt.Printf("   - Total HTTP Requests: %d\n", summary.HTTP.RequestCount)
	fmt.Printf("   - Error Count: %d\n", summary.HTTP.ErrorCount)
	fmt.Printf("   - Logs Processed: %d\n", summary.Application.LogsProcessed)
	fmt.Printf("   - Memory Usage: %d MB\n", summary.System.MemoryUsage/1024/1024)
	fmt.Printf("   - Goroutines: %d\n", summary.System.GoroutineCount)

	// Run performance optimization
	fmt.Println("\n3. Running Performance Optimization...")
	result, err := optimizer.RunOptimization(ctx)
	if err != nil {
		log.Printf("Optimization failed: %v", err)
	} else {
		fmt.Printf("   - Memory Before: %d MB\n", result.MemoryBeforeMB)
		fmt.Printf("   - Memory After: %d MB\n", result.MemoryAfterMB)
		fmt.Printf("   - Memory Freed: %d MB\n", result.MemoryFreedMB)
		fmt.Printf("   - GC Triggered: %v\n", result.GCTriggered)
		fmt.Printf("   - Optimization Time: %d ms\n", result.OptimizationTime)
		fmt.Printf("   - Recommendations: %d\n", len(result.RecommendedActions))

		for _, rec := range result.RecommendedActions {
			fmt.Printf("     * %s\n", rec)
		}
	}

	// Get performance report
	fmt.Println("\n4. Performance Report:")
	report := optimizer.GetPerformanceReport()
	fmt.Printf("   - Performance Score: %.2f/100\n", report.PerformanceScore)
	fmt.Printf("   - Recommendations: %d\n", len(report.Recommendations))

	for _, rec := range report.Recommendations {
		fmt.Printf("     * %s\n", rec)
	}

	// Test error handling
	fmt.Println("\n5. Testing Error Handling and Recovery...")

	errorScenarios := []struct {
		name      string
		errorType recovery.ErrorType
		error     error
	}{
		{"Database Connection", recovery.ErrorTypeDatabase, fmt.Errorf("connection refused")},
		{"AI Service Timeout", recovery.ErrorTypeAI, fmt.Errorf("timeout")},
		{"Cache Unavailable", recovery.ErrorTypeCache, fmt.Errorf("redis connection failed")},
		{"Network Error", recovery.ErrorTypeNetwork, fmt.Errorf("network unreachable")},
	}

	for _, scenario := range errorScenarios {
		errorCtx := &recovery.ErrorContext{
			Type:      scenario.errorType,
			Operation: "TestOperation",
			Component: "DemoComponent",
			Timestamp: time.Now(),
			Error:     scenario.error,
			Retryable: true,
			Severity:  recovery.SeverityMedium,
		}

		report := errorHandler.HandleError(ctx, errorCtx)
		fmt.Printf("   - %s: Resolved=%v, Actions=%d, Retries=%d\n",
			scenario.name, report.Resolved, len(report.RecoveryActions), report.TotalRetries)
	}

	// Check performance alerts
	fmt.Println("\n6. Performance Alerts:")
	alerts := metricsCollector.CheckPerformanceThresholds()
	if len(alerts) == 0 {
		fmt.Println("   - No performance alerts")
	} else {
		for _, alert := range alerts {
			fmt.Printf("   - %s: %s (%.2f > %.2f)\n",
				alert.Type, alert.Message, alert.MetricValue, alert.Threshold)
		}
	}

	// Get error statistics
	fmt.Println("\n7. Error Statistics:")
	errorStats := errorHandler.GetErrorStatistics()
	fmt.Printf("   - Total Errors: %v\n", errorStats["total_errors"])
	fmt.Printf("   - Error Rate: %.4f\n", errorStats["error_rate"])

	// Final system status
	fmt.Println("\n8. Final System Status:")
	finalSummary := metricsCollector.GetSummary()
	finalReport := optimizer.GetPerformanceReport()

	fmt.Printf("   - Uptime: %s\n", finalSummary.Uptime)
	fmt.Printf("   - Total Requests Processed: %d\n", finalSummary.HTTP.RequestCount)
	fmt.Printf("   - Overall Performance Score: %.2f/100\n", finalReport.PerformanceScore)

	if finalReport.PerformanceScore >= 80 {
		fmt.Println("   - System Status: EXCELLENT ✅")
	} else if finalReport.PerformanceScore >= 60 {
		fmt.Println("   - System Status: GOOD ✅")
	} else if finalReport.PerformanceScore >= 40 {
		fmt.Println("   - System Status: FAIR ⚠️")
	} else {
		fmt.Println("   - System Status: NEEDS ATTENTION ❌")
	}

	logger.Info("Demo completed successfully")
	fmt.Println("\n=== Demo Complete ===")
}
