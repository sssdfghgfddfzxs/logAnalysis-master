package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// MetricsCollector collects and manages system metrics
type MetricsCollector struct {
	mu                 sync.RWMutex
	httpMetrics        *HTTPMetrics
	systemMetrics      *SystemMetrics
	applicationMetrics *ApplicationMetrics
	startTime          time.Time
}

// HTTPMetrics tracks HTTP-related metrics
type HTTPMetrics struct {
	RequestCount      int64              `json:"request_count"`
	ResponseTimes     map[string][]int64 `json:"response_times"` // endpoint -> response times in ms
	StatusCodes       map[int]int64      `json:"status_codes"`
	ErrorCount        int64              `json:"error_count"`
	ActiveConnections int64              `json:"active_connections"`
}

// SystemMetrics tracks system-level metrics
type SystemMetrics struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    int64   `json:"memory_usage"`
	MemoryTotal    int64   `json:"memory_total"`
	GoroutineCount int     `json:"goroutine_count"`
	GCPauseTime    int64   `json:"gc_pause_time_ns"`
	GCCount        int64   `json:"gc_count"`
	HeapSize       int64   `json:"heap_size"`
	HeapInUse      int64   `json:"heap_in_use"`
}

// ApplicationMetrics tracks application-specific metrics
type ApplicationMetrics struct {
	LogsProcessed        int64   `json:"logs_processed"`
	LogsPerSecond        float64 `json:"logs_per_second"`
	AnalysisRequests     int64   `json:"analysis_requests"`
	AnalysisSuccessRate  float64 `json:"analysis_success_rate"`
	AnalysisFailures     int64   `json:"analysis_failures"`
	QueueSize            int64   `json:"queue_size"`
	QueueProcessingTime  int64   `json:"queue_processing_time_ms"`
	CacheHitRate         float64 `json:"cache_hit_rate"`
	CacheHits            int64   `json:"cache_hits"`
	CacheMisses          int64   `json:"cache_misses"`
	DatabaseConnections  int     `json:"database_connections"`
	AlertsSent           int64   `json:"alerts_sent"`
	WebSocketConnections int     `json:"websocket_connections"`
}

// MetricsSummary provides a comprehensive view of all metrics
type MetricsSummary struct {
	Timestamp   time.Time           `json:"timestamp"`
	Uptime      string              `json:"uptime"`
	HTTP        *HTTPMetrics        `json:"http"`
	System      *SystemMetrics      `json:"system"`
	Application *ApplicationMetrics `json:"application"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		httpMetrics: &HTTPMetrics{
			ResponseTimes: make(map[string][]int64),
			StatusCodes:   make(map[int]int64),
		},
		systemMetrics:      &SystemMetrics{},
		applicationMetrics: &ApplicationMetrics{},
		startTime:          time.Now(),
	}
}

// RecordHTTPRequest records an HTTP request metric
func (mc *MetricsCollector) RecordHTTPRequest(endpoint string, statusCode int, responseTime time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.httpMetrics.RequestCount++
	mc.httpMetrics.StatusCodes[statusCode]++

	if statusCode >= 400 {
		mc.httpMetrics.ErrorCount++
	}

	// Store response time in milliseconds
	responseTimeMs := responseTime.Nanoseconds() / 1000000
	if mc.httpMetrics.ResponseTimes[endpoint] == nil {
		mc.httpMetrics.ResponseTimes[endpoint] = make([]int64, 0)
	}

	// Keep only last 100 response times per endpoint to prevent memory growth
	times := mc.httpMetrics.ResponseTimes[endpoint]
	if len(times) >= 100 {
		times = times[1:]
	}
	mc.httpMetrics.ResponseTimes[endpoint] = append(times, responseTimeMs)
}

// UpdateActiveConnections updates the active HTTP connections count
func (mc *MetricsCollector) UpdateActiveConnections(count int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.httpMetrics.ActiveConnections = count
}

// UpdateSystemMetrics updates system-level metrics
func (mc *MetricsCollector) UpdateSystemMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mc.systemMetrics.MemoryUsage = int64(memStats.Alloc)
	mc.systemMetrics.MemoryTotal = int64(memStats.Sys)
	mc.systemMetrics.GoroutineCount = runtime.NumGoroutine()
	mc.systemMetrics.GCPauseTime = int64(memStats.PauseNs[(memStats.NumGC+255)%256])
	mc.systemMetrics.GCCount = int64(memStats.NumGC)
	mc.systemMetrics.HeapSize = int64(memStats.HeapSys)
	mc.systemMetrics.HeapInUse = int64(memStats.HeapInuse)
}

// RecordLogProcessed records a processed log
func (mc *MetricsCollector) RecordLogProcessed() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.applicationMetrics.LogsProcessed++
}

// RecordAnalysisRequest records an analysis request
func (mc *MetricsCollector) RecordAnalysisRequest(success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.applicationMetrics.AnalysisRequests++
	if !success {
		mc.applicationMetrics.AnalysisFailures++
	}

	// Calculate success rate
	if mc.applicationMetrics.AnalysisRequests > 0 {
		successCount := mc.applicationMetrics.AnalysisRequests - mc.applicationMetrics.AnalysisFailures
		mc.applicationMetrics.AnalysisSuccessRate = float64(successCount) / float64(mc.applicationMetrics.AnalysisRequests)
	}
}

// UpdateQueueMetrics updates queue-related metrics
func (mc *MetricsCollector) UpdateQueueMetrics(size int64, processingTime time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.applicationMetrics.QueueSize = size
	mc.applicationMetrics.QueueProcessingTime = processingTime.Nanoseconds() / 1000000 // Convert to ms
}

// RecordCacheOperation records a cache operation
func (mc *MetricsCollector) RecordCacheOperation(hit bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if hit {
		mc.applicationMetrics.CacheHits++
	} else {
		mc.applicationMetrics.CacheMisses++
	}

	// Calculate hit rate
	total := mc.applicationMetrics.CacheHits + mc.applicationMetrics.CacheMisses
	if total > 0 {
		mc.applicationMetrics.CacheHitRate = float64(mc.applicationMetrics.CacheHits) / float64(total)
	}
}

// UpdateDatabaseConnections updates database connection count
func (mc *MetricsCollector) UpdateDatabaseConnections(count int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.applicationMetrics.DatabaseConnections = count
}

// RecordAlertSent records an alert being sent
func (mc *MetricsCollector) RecordAlertSent() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.applicationMetrics.AlertsSent++
}

// UpdateWebSocketConnections updates WebSocket connection count
func (mc *MetricsCollector) UpdateWebSocketConnections(count int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.applicationMetrics.WebSocketConnections = count
}

// GetSummary returns a comprehensive metrics summary
func (mc *MetricsCollector) GetSummary() *MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Update system metrics before returning summary
	mc.mu.RUnlock()
	mc.UpdateSystemMetrics()
	mc.mu.RLock()

	// Calculate logs per second
	uptime := time.Since(mc.startTime)
	if uptime.Seconds() > 0 {
		mc.applicationMetrics.LogsPerSecond = float64(mc.applicationMetrics.LogsProcessed) / uptime.Seconds()
	}

	// Deep copy metrics to avoid race conditions
	httpMetrics := &HTTPMetrics{
		RequestCount:      mc.httpMetrics.RequestCount,
		ResponseTimes:     make(map[string][]int64),
		StatusCodes:       make(map[int]int64),
		ErrorCount:        mc.httpMetrics.ErrorCount,
		ActiveConnections: mc.httpMetrics.ActiveConnections,
	}

	for endpoint, times := range mc.httpMetrics.ResponseTimes {
		httpMetrics.ResponseTimes[endpoint] = make([]int64, len(times))
		copy(httpMetrics.ResponseTimes[endpoint], times)
	}

	for code, count := range mc.httpMetrics.StatusCodes {
		httpMetrics.StatusCodes[code] = count
	}

	return &MetricsSummary{
		Timestamp:   time.Now(),
		Uptime:      uptime.String(),
		HTTP:        httpMetrics,
		System:      &(*mc.systemMetrics),      // Copy struct
		Application: &(*mc.applicationMetrics), // Copy struct
	}
}

// GetAverageResponseTime calculates average response time for an endpoint
func (mc *MetricsCollector) GetAverageResponseTime(endpoint string) float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	times, exists := mc.httpMetrics.ResponseTimes[endpoint]
	if !exists || len(times) == 0 {
		return 0
	}

	var sum int64
	for _, time := range times {
		sum += time
	}

	return float64(sum) / float64(len(times))
}

// GetP95ResponseTime calculates 95th percentile response time for an endpoint
func (mc *MetricsCollector) GetP95ResponseTime(endpoint string) int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	times, exists := mc.httpMetrics.ResponseTimes[endpoint]
	if !exists || len(times) == 0 {
		return 0
	}

	// Simple implementation - in production, use a proper percentile calculation
	if len(times) == 1 {
		return times[0]
	}

	// Sort times (simple bubble sort for small arrays)
	sortedTimes := make([]int64, len(times))
	copy(sortedTimes, times)

	for i := 0; i < len(sortedTimes); i++ {
		for j := 0; j < len(sortedTimes)-1-i; j++ {
			if sortedTimes[j] > sortedTimes[j+1] {
				sortedTimes[j], sortedTimes[j+1] = sortedTimes[j+1], sortedTimes[j]
			}
		}
	}

	// Calculate 95th percentile index
	index := int(float64(len(sortedTimes)) * 0.95)
	if index >= len(sortedTimes) {
		index = len(sortedTimes) - 1
	}

	return sortedTimes[index]
}

// Reset resets all metrics (useful for testing)
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.httpMetrics = &HTTPMetrics{
		ResponseTimes: make(map[string][]int64),
		StatusCodes:   make(map[int]int64),
	}
	mc.systemMetrics = &SystemMetrics{}
	mc.applicationMetrics = &ApplicationMetrics{}
	mc.startTime = time.Now()
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	MetricValue float64   `json:"metric_value"`
	Threshold   float64   `json:"threshold"`
}

// CheckPerformanceThresholds checks if any performance thresholds are exceeded
func (mc *MetricsCollector) CheckPerformanceThresholds() []PerformanceAlert {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var alerts []PerformanceAlert
	now := time.Now()

	// Check memory usage (alert if > 80% of total)
	if mc.systemMetrics.MemoryTotal > 0 {
		memoryUsagePercent := float64(mc.systemMetrics.MemoryUsage) / float64(mc.systemMetrics.MemoryTotal) * 100
		if memoryUsagePercent > 80 {
			alerts = append(alerts, PerformanceAlert{
				Type:        "memory_usage",
				Message:     "High memory usage detected",
				Severity:    "warning",
				Timestamp:   now,
				MetricValue: memoryUsagePercent,
				Threshold:   80,
			})
		}
	}

	// Check goroutine count (alert if > 1000)
	if mc.systemMetrics.GoroutineCount > 1000 {
		alerts = append(alerts, PerformanceAlert{
			Type:        "goroutine_count",
			Message:     "High goroutine count detected",
			Severity:    "warning",
			Timestamp:   now,
			MetricValue: float64(mc.systemMetrics.GoroutineCount),
			Threshold:   1000,
		})
	}

	// Check error rate (alert if > 5%)
	if mc.httpMetrics.RequestCount > 0 {
		errorRate := float64(mc.httpMetrics.ErrorCount) / float64(mc.httpMetrics.RequestCount) * 100
		if errorRate > 5 {
			alerts = append(alerts, PerformanceAlert{
				Type:        "error_rate",
				Message:     "High error rate detected",
				Severity:    "critical",
				Timestamp:   now,
				MetricValue: errorRate,
				Threshold:   5,
			})
		}
	}

	// Check analysis success rate (alert if < 90%)
	if mc.applicationMetrics.AnalysisRequests > 10 && mc.applicationMetrics.AnalysisSuccessRate < 0.9 {
		alerts = append(alerts, PerformanceAlert{
			Type:        "analysis_success_rate",
			Message:     "Low analysis success rate detected",
			Severity:    "warning",
			Timestamp:   now,
			MetricValue: mc.applicationMetrics.AnalysisSuccessRate * 100,
			Threshold:   90,
		})
	}

	// Check cache hit rate (alert if < 70%)
	total := mc.applicationMetrics.CacheHits + mc.applicationMetrics.CacheMisses
	if total > 100 && mc.applicationMetrics.CacheHitRate < 0.7 {
		alerts = append(alerts, PerformanceAlert{
			Type:        "cache_hit_rate",
			Message:     "Low cache hit rate detected",
			Severity:    "warning",
			Timestamp:   now,
			MetricValue: mc.applicationMetrics.CacheHitRate * 100,
			Threshold:   70,
		})
	}

	return alerts
}

// StartPeriodicCollection starts periodic metrics collection
func (mc *MetricsCollector) StartPeriodicCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.UpdateSystemMetrics()
		}
	}
}
