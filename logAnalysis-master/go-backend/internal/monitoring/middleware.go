package monitoring

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware creates a middleware that collects HTTP metrics
func MetricsMiddleware(collector *MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment active connections
		collector.UpdateActiveConnections(collector.httpMetrics.ActiveConnections + 1)

		// Process request
		c.Next()

		// Calculate response time
		duration := time.Since(start)

		// Record metrics
		endpoint := c.Request.Method + " " + c.FullPath()
		statusCode := c.Writer.Status()

		collector.RecordHTTPRequest(endpoint, statusCode, duration)

		// Decrement active connections
		collector.UpdateActiveConnections(collector.httpMetrics.ActiveConnections - 1)

		// Add metrics headers for debugging
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Request-ID", strconv.FormatInt(collector.httpMetrics.RequestCount, 10))
	}
}

// LoggingMiddleware creates a middleware that logs requests with performance data
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.ErrorMessage,
		)
	})
}

// RecoveryMiddleware creates a middleware that recovers from panics and records them
func RecoveryMiddleware(collector *MetricsCollector) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Record the panic as an error
		collector.RecordHTTPRequest(c.Request.Method+" "+c.FullPath(), 500, 0)

		// Log the panic
		gin.DefaultErrorWriter.Write([]byte(fmt.Sprintf("Panic recovered: %v\n", recovered)))

		c.AbortWithStatus(500)
	})
}

// HealthCheckMiddleware adds health check information to responses
func HealthCheckMiddleware(collector *MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add health indicators to response headers
		summary := collector.GetSummary()

		c.Header("X-System-Health", "ok")
		c.Header("X-Memory-Usage", fmt.Sprintf("%.2f%%",
			float64(summary.System.MemoryUsage)/float64(summary.System.MemoryTotal)*100))
		c.Header("X-Goroutines", strconv.Itoa(summary.System.GoroutineCount))
		c.Header("X-Uptime", summary.Uptime)

		c.Next()
	}
}
