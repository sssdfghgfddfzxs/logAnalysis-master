package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"intelligent-log-analysis/internal/config"
	"intelligent-log-analysis/internal/logging"
	"intelligent-log-analysis/internal/monitoring"
	"intelligent-log-analysis/internal/optimization"
	"intelligent-log-analysis/internal/recovery"

	"github.com/gin-gonic/gin"
)

// EnhancedServer extends the basic server with monitoring, optimization, and error handling
type EnhancedServer struct {
	*Server
	metricsCollector  *monitoring.MetricsCollector
	optimizer         *optimization.PerformanceOptimizer
	errorHandler      *recovery.ErrorHandler
	logger            *logging.Logger
	performanceLogger *logging.PerformanceLogger
	auditLogger       *logging.AuditLogger
}

// NewEnhancedServer creates a new enhanced server with monitoring and optimization
func NewEnhancedServer(cfg *config.Config) *EnhancedServer {
	// Initialize logging
	logConfig := &logging.LoggerConfig{
		Level:     logging.INFO,
		Component: "server",
		Output:    "stdout",
		Format:    "json",
	}
	logger := logging.NewLogger(logConfig)

	// Initialize base server
	baseServer := New(cfg)

	// Initialize monitoring
	metricsCollector := monitoring.NewMetricsCollector()

	// Initialize performance optimizer
	optimizer := optimization.NewPerformanceOptimizer(metricsCollector, nil)

	// Initialize error handler
	errorHandler := recovery.NewErrorHandler(metricsCollector, nil)

	// Create specialized loggers
	performanceLogger := logging.NewPerformanceLogger(logger)
	auditLogger := logging.NewAuditLogger(logger)

	enhanced := &EnhancedServer{
		Server:            baseServer,
		metricsCollector:  metricsCollector,
		optimizer:         optimizer,
		errorHandler:      errorHandler,
		logger:            logger,
		performanceLogger: performanceLogger,
		auditLogger:       auditLogger,
	}

	// Setup enhanced middleware
	enhanced.setupEnhancedMiddleware()

	// Setup monitoring endpoints
	enhanced.setupMonitoringEndpoints()

	return enhanced
}

// setupEnhancedMiddleware adds monitoring and error handling middleware
func (es *EnhancedServer) setupEnhancedMiddleware() {
	// Add metrics middleware
	es.router.Use(monitoring.MetricsMiddleware(es.metricsCollector))

	// Add recovery middleware with error handling
	es.router.Use(monitoring.RecoveryMiddleware(es.metricsCollector))

	// Add health check middleware
	es.router.Use(monitoring.HealthCheckMiddleware(es.metricsCollector))

	// Add logging middleware
	es.router.Use(es.loggingMiddleware())
}

// setupMonitoringEndpoints adds monitoring and management endpoints
func (es *EnhancedServer) setupMonitoringEndpoints() {
	// Monitoring endpoints
	monitoring := es.router.Group("/monitoring")
	{
		monitoring.GET("/metrics", es.handleGetMetrics)
		monitoring.GET("/performance", es.handleGetPerformanceReport)
		monitoring.GET("/errors", es.handleGetErrorStatistics)
		monitoring.POST("/optimize", es.handleRunOptimization)
		monitoring.GET("/health/detailed", es.handleDetailedHealthCheck)
		monitoring.GET("/alerts", es.handleGetAlerts)
	}

	// Admin endpoints
	admin := es.router.Group("/admin")
	{
		admin.POST("/gc", es.handleTriggerGC)
		admin.POST("/reset-metrics", es.handleResetMetrics)
		admin.POST("/reset-errors", es.handleResetErrors)
		admin.GET("/system-info", es.handleGetSystemInfo)
	}
}

// Start starts the enhanced server with monitoring and optimization
func (es *EnhancedServer) Start() error {
	ctx := context.Background()

	// Start periodic metrics collection
	go es.metricsCollector.StartPeriodicCollection(ctx, 10*time.Second)

	// Start automatic performance optimization
	go es.optimizer.StartAutoOptimization(ctx)

	// Log server startup
	es.auditLogger.LogSystemEvent("server_startup", "enhanced_server", map[string]interface{}{
		"host": es.config.Server.Host,
		"port": es.config.Server.Port,
	})

	// Start the base server
	return es.Server.Start()
}

// loggingMiddleware creates a custom logging middleware
func (es *EnhancedServer) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log request details
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := int64(c.Writer.Size())

		es.performanceLogger.LogHTTPRequest(method, path, statusCode, duration, responseSize)

		// Log slow requests
		es.performanceLogger.LogSlowOperation(
			fmt.Sprintf("%s %s", method, path),
			duration,
			1*time.Second, // 1 second threshold
		)
	}
}

// Monitoring endpoint handlers

// handleGetMetrics returns comprehensive system metrics
func (es *EnhancedServer) handleGetMetrics(c *gin.Context) {
	summary := es.metricsCollector.GetSummary()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"metrics": summary,
	})
}

// handleGetPerformanceReport returns a detailed performance report
func (es *EnhancedServer) handleGetPerformanceReport(c *gin.Context) {
	report := es.optimizer.GetPerformanceReport()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  report,
	})
}

// handleGetErrorStatistics returns error statistics
func (es *EnhancedServer) handleGetErrorStatistics(c *gin.Context) {
	stats := es.errorHandler.GetErrorStatistics()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"errors":  stats,
	})
}

// handleRunOptimization manually triggers system optimization
func (es *EnhancedServer) handleRunOptimization(c *gin.Context) {
	ctx := c.Request.Context()

	result, err := es.optimizer.RunOptimization(ctx)
	if err != nil {
		es.sendErrorResponse(c, http.StatusInternalServerError, "OPTIMIZATION_FAILED",
			fmt.Sprintf("Failed to run optimization: %v", err))
		return
	}

	es.auditLogger.LogSystemEvent("manual_optimization", "enhanced_server", map[string]interface{}{
		"memory_freed_mb":   result.MemoryFreedMB,
		"optimization_time": result.OptimizationTime,
		"gc_triggered":      result.GCTriggered,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// handleDetailedHealthCheck returns detailed health information
func (es *EnhancedServer) handleDetailedHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Check database health
	if es.db != nil {
		if err := es.db.Health(ctx); err != nil {
			health["database"] = gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			health["database"] = gin.H{
				"status": "healthy",
			}
		}
	}

	// Check AI service health
	if es.aiClient != nil {
		if err := es.services.Log.HealthCheckAI(ctx); err != nil {
			health["ai_service"] = gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			health["ai_service"] = gin.H{
				"status": "healthy",
			}
		}
	} else {
		health["ai_service"] = gin.H{
			"status": "disabled",
		}
	}

	// Add performance metrics
	summary := es.metricsCollector.GetSummary()
	health["performance"] = gin.H{
		"memory_usage_mb": summary.System.MemoryUsage / 1024 / 1024,
		"goroutines":      summary.System.GoroutineCount,
		"uptime":          summary.Uptime,
		"request_count":   summary.HTTP.RequestCount,
		"error_rate":      float64(summary.HTTP.ErrorCount) / float64(summary.HTTP.RequestCount),
	}

	// Add performance score
	report := es.optimizer.GetPerformanceReport()
	health["performance_score"] = report.PerformanceScore

	// Check for performance alerts
	alerts := es.metricsCollector.CheckPerformanceThresholds()
	if len(alerts) > 0 {
		health["alerts"] = alerts
		health["status"] = "degraded"
	}

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// handleGetAlerts returns current performance alerts
func (es *EnhancedServer) handleGetAlerts(c *gin.Context) {
	alerts := es.metricsCollector.CheckPerformanceThresholds()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"alerts":  alerts,
		"count":   len(alerts),
	})
}

// Admin endpoint handlers

// handleTriggerGC manually triggers garbage collection
func (es *EnhancedServer) handleTriggerGC(c *gin.Context) {
	es.optimizer.TuneGCSettings()

	es.auditLogger.LogSystemEvent("manual_gc", "enhanced_server", map[string]interface{}{
		"triggered_by": "admin_endpoint",
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Garbage collection triggered",
	})
}

// handleResetMetrics resets all metrics
func (es *EnhancedServer) handleResetMetrics(c *gin.Context) {
	es.metricsCollector.Reset()

	es.auditLogger.LogSystemEvent("metrics_reset", "enhanced_server", map[string]interface{}{
		"triggered_by": "admin_endpoint",
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Metrics reset successfully",
	})
}

// handleResetErrors resets error counts
func (es *EnhancedServer) handleResetErrors(c *gin.Context) {
	es.errorHandler.ResetErrorCounts()

	es.auditLogger.LogSystemEvent("error_counts_reset", "enhanced_server", map[string]interface{}{
		"triggered_by": "admin_endpoint",
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Error counts reset successfully",
	})
}

// handleGetSystemInfo returns detailed system information
func (es *EnhancedServer) handleGetSystemInfo(c *gin.Context) {
	summary := es.metricsCollector.GetSummary()
	report := es.optimizer.GetPerformanceReport()
	errorStats := es.errorHandler.GetErrorStatistics()

	systemInfo := gin.H{
		"server": gin.H{
			"version":   "1.0.0",
			"build":     "development",
			"uptime":    summary.Uptime,
			"timestamp": time.Now().Format(time.RFC3339),
		},
		"metrics":     summary,
		"performance": report,
		"errors":      errorStats,
		"config": gin.H{
			"host": es.config.Server.Host,
			"port": es.config.Server.Port,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"system_info": systemInfo,
	})
}

// HandleError provides a centralized error handling mechanism
func (es *EnhancedServer) HandleError(ctx context.Context, errorType recovery.ErrorType, component, operation string, err error) *recovery.ErrorReport {
	errorCtx := &recovery.ErrorContext{
		Type:      errorType,
		Operation: operation,
		Component: component,
		Timestamp: time.Now(),
		Error:     err,
		Retryable: es.isRetryableError(errorType, err),
		Severity:  es.determineSeverity(errorType, err),
	}

	report := es.errorHandler.HandleError(ctx, errorCtx)

	// Log the error handling result
	es.logger.WithFields(map[string]interface{}{
		"error_type":       string(errorType),
		"component":        component,
		"operation":        operation,
		"resolved":         report.Resolved,
		"recovery_actions": len(report.RecoveryActions),
		"total_retries":    report.TotalRetries,
	}).Error("Error handled", err)

	return report
}

// isRetryableError determines if an error is retryable based on type and content
func (es *EnhancedServer) isRetryableError(errorType recovery.ErrorType, err error) bool {
	if err == nil {
		return false
	}

	switch errorType {
	case recovery.ErrorTypeValidation:
		return false
	case recovery.ErrorTypeDatabase, recovery.ErrorTypeAI, recovery.ErrorTypeNetwork, recovery.ErrorTypeQueue:
		return true
	default:
		return false
	}
}

// determineSeverity determines the severity of an error
func (es *EnhancedServer) determineSeverity(errorType recovery.ErrorType, err error) recovery.ErrorSeverity {
	switch errorType {
	case recovery.ErrorTypeValidation:
		return recovery.SeverityLow
	case recovery.ErrorTypeCache:
		return recovery.SeverityMedium
	case recovery.ErrorTypeDatabase, recovery.ErrorTypeAI:
		return recovery.SeverityHigh
	case recovery.ErrorTypeInternal:
		return recovery.SeverityCritical
	default:
		return recovery.SeverityMedium
	}
}
