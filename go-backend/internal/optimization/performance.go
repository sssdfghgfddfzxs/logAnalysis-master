package optimization

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"intelligent-log-analysis/internal/monitoring"
)

// PerformanceOptimizer handles performance optimization tasks
type PerformanceOptimizer struct {
	metricsCollector *monitoring.MetricsCollector
	config           *OptimizationConfig
	mu               sync.RWMutex
	lastOptimization time.Time
}

// OptimizationConfig contains configuration for performance optimization
type OptimizationConfig struct {
	GCTargetPercent       int           `json:"gc_target_percent"`
	MaxGoroutines         int           `json:"max_goroutines"`
	MemoryThresholdMB     int64         `json:"memory_threshold_mb"`
	OptimizationInterval  time.Duration `json:"optimization_interval"`
	EnableAutoGC          bool          `json:"enable_auto_gc"`
	EnableMemoryProfiling bool          `json:"enable_memory_profiling"`
}

// DefaultOptimizationConfig returns default optimization configuration
func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		GCTargetPercent:       100, // Default Go GC target
		MaxGoroutines:         10000,
		MemoryThresholdMB:     512,
		OptimizationInterval:  5 * time.Minute,
		EnableAutoGC:          true,
		EnableMemoryProfiling: false,
	}
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(collector *monitoring.MetricsCollector, config *OptimizationConfig) *PerformanceOptimizer {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	return &PerformanceOptimizer{
		metricsCollector: collector,
		config:           config,
		lastOptimization: time.Now(),
	}
}

// OptimizationResult contains the results of an optimization run
type OptimizationResult struct {
	Timestamp          time.Time `json:"timestamp"`
	MemoryBeforeMB     int64     `json:"memory_before_mb"`
	MemoryAfterMB      int64     `json:"memory_after_mb"`
	MemoryFreedMB      int64     `json:"memory_freed_mb"`
	GoroutinesBefore   int       `json:"goroutines_before"`
	GoroutinesAfter    int       `json:"goroutines_after"`
	GCTriggered        bool      `json:"gc_triggered"`
	OptimizationTime   int64     `json:"optimization_time_ms"`
	RecommendedActions []string  `json:"recommended_actions"`
}

// RunOptimization performs a comprehensive system optimization
func (po *PerformanceOptimizer) RunOptimization(ctx context.Context) (*OptimizationResult, error) {
	start := time.Now()

	po.mu.Lock()
	defer po.mu.Unlock()

	// Get initial metrics
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	goroutinesBefore := runtime.NumGoroutine()

	result := &OptimizationResult{
		Timestamp:        start,
		MemoryBeforeMB:   int64(memStatsBefore.Alloc) / 1024 / 1024,
		GoroutinesBefore: goroutinesBefore,
	}

	var recommendations []string

	// 1. Memory optimization
	if po.config.EnableAutoGC && result.MemoryBeforeMB > po.config.MemoryThresholdMB {
		runtime.GC()
		result.GCTriggered = true
		recommendations = append(recommendations, "Triggered garbage collection due to high memory usage")
	}

	// 2. Goroutine optimization check
	if goroutinesBefore > po.config.MaxGoroutines {
		recommendations = append(recommendations,
			"High goroutine count detected - consider implementing goroutine pooling")
	}

	// 3. GC tuning recommendations
	if memStatsBefore.NumGC > 0 {
		avgGCPause := memStatsBefore.PauseTotalNs / uint64(memStatsBefore.NumGC)
		if avgGCPause > 10*1000*1000 { // > 10ms average
			recommendations = append(recommendations,
				"High GC pause times detected - consider tuning GOGC or reducing allocation rate")
		}
	}

	// 4. Memory allocation pattern analysis
	if memStatsBefore.Mallocs > 0 {
		allocRate := float64(memStatsBefore.TotalAlloc) / float64(memStatsBefore.Mallocs)
		if allocRate > 1024*1024 { // > 1MB average allocation
			recommendations = append(recommendations,
				"Large average allocation size detected - consider object pooling")
		}
	}

	// Get final metrics
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)
	goroutinesAfter := runtime.NumGoroutine()

	result.MemoryAfterMB = int64(memStatsAfter.Alloc) / 1024 / 1024
	result.MemoryFreedMB = result.MemoryBeforeMB - result.MemoryAfterMB
	result.GoroutinesAfter = goroutinesAfter
	result.OptimizationTime = time.Since(start).Nanoseconds() / 1000000
	result.RecommendedActions = recommendations

	po.lastOptimization = time.Now()

	return result, nil
}

// StartAutoOptimization starts automatic performance optimization
func (po *PerformanceOptimizer) StartAutoOptimization(ctx context.Context) {
	ticker := time.NewTicker(po.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if result, err := po.RunOptimization(ctx); err == nil {
				// Log optimization results
				if len(result.RecommendedActions) > 0 {
					// In a real implementation, you'd log these recommendations
					// or send them to a monitoring system
					_ = result
				}
			}
		}
	}
}

// GetOptimizationRecommendations analyzes current metrics and provides recommendations
func (po *PerformanceOptimizer) GetOptimizationRecommendations() []string {
	po.mu.RLock()
	defer po.mu.RUnlock()

	recommendations := make([]string, 0) // Initialize empty slice instead of nil

	if po.metricsCollector == nil {
		return recommendations
	}

	summary := po.metricsCollector.GetSummary()

	// Memory recommendations
	if summary.System.MemoryTotal > 0 {
		memoryUsagePercent := float64(summary.System.MemoryUsage) / float64(summary.System.MemoryTotal) * 100
		if memoryUsagePercent > 80 {
			recommendations = append(recommendations,
				"Memory usage is high (%.1f%%) - consider implementing memory pooling or reducing cache sizes")
		}
	}

	// Goroutine recommendations
	if summary.System.GoroutineCount > 1000 {
		recommendations = append(recommendations,
			"High goroutine count (%d) - implement worker pools to limit concurrent goroutines")
	}

	// HTTP performance recommendations
	for endpoint, times := range summary.HTTP.ResponseTimes {
		if len(times) > 0 {
			var sum int64
			for _, t := range times {
				sum += t
			}
			avgTime := sum / int64(len(times))

			if avgTime > 1000 { // > 1 second
				recommendations = append(recommendations,
					fmt.Sprintf("Endpoint %s has high average response time (%dms) - consider optimization", endpoint, avgTime))
			}
		}
	}

	// Error rate recommendations
	if summary.HTTP.RequestCount > 0 {
		errorRate := float64(summary.HTTP.ErrorCount) / float64(summary.HTTP.RequestCount) * 100
		if errorRate > 5 {
			recommendations = append(recommendations,
				"High error rate (%.1f%%) detected - investigate error causes and implement better error handling")
		}
	}

	// Cache recommendations
	if summary.Application.CacheHitRate < 0.7 && (summary.Application.CacheHits+summary.Application.CacheMisses) > 100 {
		recommendations = append(recommendations,
			"Low cache hit rate (%.1f%%) - review cache strategy and TTL settings")
	}

	// Analysis performance recommendations
	if summary.Application.AnalysisSuccessRate < 0.9 && summary.Application.AnalysisRequests > 10 {
		recommendations = append(recommendations,
			"Low analysis success rate (%.1f%%) - check AI service connectivity and error handling")
	}

	return recommendations
}

// TuneGCSettings adjusts garbage collection settings based on current performance
func (po *PerformanceOptimizer) TuneGCSettings() {
	po.mu.Lock()
	defer po.mu.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Adjust GC target based on memory pressure

	if memStats.Alloc > uint64(po.config.MemoryThresholdMB*1024*1024) {
		// High memory usage - be more aggressive with GC
		runtime.GC()
	}
}

// GetPerformanceReport generates a comprehensive performance report
func (po *PerformanceOptimizer) GetPerformanceReport() *PerformanceReport {
	po.mu.RLock()
	defer po.mu.RUnlock()

	var summary *monitoring.MetricsSummary
	if po.metricsCollector != nil {
		summary = po.metricsCollector.GetSummary()
	} else {
		// Create empty summary if metrics collector is nil
		summary = &monitoring.MetricsSummary{
			HTTP:        &monitoring.HTTPMetrics{ResponseTimes: make(map[string][]int64), StatusCodes: make(map[int]int64)},
			System:      &monitoring.SystemMetrics{},
			Application: &monitoring.ApplicationMetrics{},
		}
	}

	recommendations := po.GetOptimizationRecommendations()

	return &PerformanceReport{
		Timestamp:          time.Now(),
		SystemMetrics:      summary.System,
		HTTPMetrics:        summary.HTTP,
		ApplicationMetrics: summary.Application,
		Recommendations:    recommendations,
		LastOptimization:   po.lastOptimization,
		OptimizationConfig: po.config,
		PerformanceScore:   po.calculatePerformanceScore(summary),
	}
}

// PerformanceReport contains a comprehensive performance analysis
type PerformanceReport struct {
	Timestamp          time.Time                      `json:"timestamp"`
	SystemMetrics      *monitoring.SystemMetrics      `json:"system_metrics"`
	HTTPMetrics        *monitoring.HTTPMetrics        `json:"http_metrics"`
	ApplicationMetrics *monitoring.ApplicationMetrics `json:"application_metrics"`
	Recommendations    []string                       `json:"recommendations"`
	LastOptimization   time.Time                      `json:"last_optimization"`
	OptimizationConfig *OptimizationConfig            `json:"optimization_config"`
	PerformanceScore   float64                        `json:"performance_score"`
}

// calculatePerformanceScore calculates an overall performance score (0-100)
func (po *PerformanceOptimizer) calculatePerformanceScore(summary *monitoring.MetricsSummary) float64 {
	score := 100.0

	// Memory usage penalty (0-20 points)
	if summary.System.MemoryTotal > 0 {
		memoryUsagePercent := float64(summary.System.MemoryUsage) / float64(summary.System.MemoryTotal) * 100
		if memoryUsagePercent > 80 {
			score -= 20
		} else if memoryUsagePercent > 60 {
			score -= 10
		}
	}

	// Error rate penalty (0-30 points)
	if summary.HTTP.RequestCount > 0 {
		errorRate := float64(summary.HTTP.ErrorCount) / float64(summary.HTTP.RequestCount) * 100
		if errorRate > 10 {
			score -= 30
		} else if errorRate > 5 {
			score -= 15
		} else if errorRate > 1 {
			score -= 5
		}
	}

	// Response time penalty (0-20 points)
	var totalResponseTime int64
	var endpointCount int
	for _, times := range summary.HTTP.ResponseTimes {
		if len(times) > 0 {
			var sum int64
			for _, t := range times {
				sum += t
			}
			totalResponseTime += sum / int64(len(times))
			endpointCount++
		}
	}

	if endpointCount > 0 {
		avgResponseTime := totalResponseTime / int64(endpointCount)
		if avgResponseTime > 2000 { // > 2 seconds
			score -= 20
		} else if avgResponseTime > 1000 { // > 1 second
			score -= 10
		} else if avgResponseTime > 500 { // > 500ms
			score -= 5
		}
	}

	// Cache performance penalty (0-15 points)
	if summary.Application.CacheHitRate < 0.5 {
		score -= 15
	} else if summary.Application.CacheHitRate < 0.7 {
		score -= 10
	} else if summary.Application.CacheHitRate < 0.8 {
		score -= 5
	}

	// Analysis success rate penalty (0-15 points)
	if summary.Application.AnalysisRequests > 10 {
		if summary.Application.AnalysisSuccessRate < 0.8 {
			score -= 15
		} else if summary.Application.AnalysisSuccessRate < 0.9 {
			score -= 10
		} else if summary.Application.AnalysisSuccessRate < 0.95 {
			score -= 5
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}
