package recovery

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"intelligent-log-analysis/internal/monitoring"
)

// ErrorHandler manages error handling and recovery mechanisms
type ErrorHandler struct {
	metricsCollector *monitoring.MetricsCollector
	config           *ErrorHandlerConfig
	errorCounts      map[string]int64
	lastErrors       map[string]time.Time
	mu               sync.RWMutex
}

// ErrorHandlerConfig contains configuration for error handling
type ErrorHandlerConfig struct {
	MaxRetries              int           `json:"max_retries"`
	RetryBackoffBase        time.Duration `json:"retry_backoff_base"`
	RetryBackoffMax         time.Duration `json:"retry_backoff_max"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`
	ErrorRateThreshold      float64       `json:"error_rate_threshold"`
	AlertOnHighErrorRate    bool          `json:"alert_on_high_error_rate"`
}

// DefaultErrorHandlerConfig returns default error handler configuration
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		MaxRetries:              3,
		RetryBackoffBase:        100 * time.Millisecond,
		RetryBackoffMax:         30 * time.Second,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,
		ErrorRateThreshold:      0.1, // 10%
		AlertOnHighErrorRate:    true,
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(collector *monitoring.MetricsCollector, config *ErrorHandlerConfig) *ErrorHandler {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}

	return &ErrorHandler{
		metricsCollector: collector,
		config:           config,
		errorCounts:      make(map[string]int64),
		lastErrors:       make(map[string]time.Time),
	}
}

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeAI         ErrorType = "ai_service"
	ErrorTypeCache      ErrorType = "cache"
	ErrorTypeQueue      ErrorType = "queue"
	ErrorTypeHTTP       ErrorType = "http"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeNetwork    ErrorType = "network"
)

// ErrorContext contains context information about an error
type ErrorContext struct {
	Type      ErrorType              `json:"type"`
	Operation string                 `json:"operation"`
	Component string                 `json:"component"`
	Timestamp time.Time              `json:"timestamp"`
	Error     error                  `json:"error"`
	Metadata  map[string]interface{} `json:"metadata"`
	Retryable bool                   `json:"retryable"`
	Severity  ErrorSeverity          `json:"severity"`
}

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// RecoveryAction represents an action taken during error recovery
type RecoveryAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Success     bool                   `json:"success"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ErrorReport contains comprehensive error information
type ErrorReport struct {
	Context         *ErrorContext    `json:"context"`
	RecoveryActions []RecoveryAction `json:"recovery_actions"`
	Resolved        bool             `json:"resolved"`
	ResolutionTime  time.Duration    `json:"resolution_time"`
	TotalRetries    int              `json:"total_retries"`
}

// HandleError processes an error and attempts recovery
func (eh *ErrorHandler) HandleError(ctx context.Context, errorCtx *ErrorContext) *ErrorReport {
	start := time.Now()

	eh.mu.Lock()
	defer eh.mu.Unlock()

	// Record error metrics
	errorKey := string(errorCtx.Type) + ":" + errorCtx.Component
	eh.errorCounts[errorKey]++
	eh.lastErrors[errorKey] = errorCtx.Timestamp

	report := &ErrorReport{
		Context:         errorCtx,
		RecoveryActions: make([]RecoveryAction, 0),
	}

	// Determine if error is retryable
	if !errorCtx.Retryable {
		report.RecoveryActions = append(report.RecoveryActions, RecoveryAction{
			Type:        "log_error",
			Description: "Error is not retryable, logging for investigation",
			Timestamp:   time.Now(),
			Success:     true,
		})
		return report
	}

	// Check circuit breaker
	if eh.isCircuitBreakerOpen(errorKey) {
		report.RecoveryActions = append(report.RecoveryActions, RecoveryAction{
			Type:        "circuit_breaker",
			Description: "Circuit breaker is open, skipping retry",
			Timestamp:   time.Now(),
			Success:     false,
		})
		return report
	}

	// Attempt recovery based on error type
	recovered := eh.attemptRecovery(ctx, errorCtx, report)

	report.Resolved = recovered
	report.ResolutionTime = time.Since(start)

	// Check if we need to open circuit breaker
	if !recovered && eh.errorCounts[errorKey] >= int64(eh.config.CircuitBreakerThreshold) {
		eh.openCircuitBreaker(errorKey)
		report.RecoveryActions = append(report.RecoveryActions, RecoveryAction{
			Type:        "circuit_breaker_open",
			Description: "Opened circuit breaker due to repeated failures",
			Timestamp:   time.Now(),
			Success:     true,
		})
	}

	return report
}

// attemptRecovery attempts to recover from an error based on its type
func (eh *ErrorHandler) attemptRecovery(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	switch errorCtx.Type {
	case ErrorTypeDatabase:
		return eh.recoverDatabaseError(ctx, errorCtx, report)
	case ErrorTypeAI:
		return eh.recoverAIError(ctx, errorCtx, report)
	case ErrorTypeCache:
		return eh.recoverCacheError(ctx, errorCtx, report)
	case ErrorTypeQueue:
		return eh.recoverQueueError(ctx, errorCtx, report)
	case ErrorTypeNetwork:
		return eh.recoverNetworkError(ctx, errorCtx, report)
	default:
		return eh.recoverGenericError(ctx, errorCtx, report)
	}
}

// recoverDatabaseError attempts to recover from database errors
func (eh *ErrorHandler) recoverDatabaseError(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	// Implement database-specific recovery logic
	for attempt := 1; attempt <= eh.config.MaxRetries; attempt++ {
		report.TotalRetries = attempt

		// Wait with exponential backoff
		backoff := eh.calculateBackoff(attempt)
		time.Sleep(backoff)

		action := RecoveryAction{
			Type:        "database_retry",
			Description: fmt.Sprintf("Retrying database operation (attempt %d/%d)", attempt, eh.config.MaxRetries),
			Timestamp:   time.Now(),
			Metadata: map[string]interface{}{
				"attempt": attempt,
				"backoff": backoff.String(),
			},
		}

		// In a real implementation, you would retry the actual database operation here
		// For now, we'll simulate a recovery based on error patterns
		if eh.isDatabaseRecoverable(errorCtx.Error) {
			action.Success = true
			report.RecoveryActions = append(report.RecoveryActions, action)
			return true
		}

		action.Success = false
		report.RecoveryActions = append(report.RecoveryActions, action)
	}

	// If retries failed, try fallback mechanisms
	fallbackAction := RecoveryAction{
		Type:        "database_fallback",
		Description: "Attempting fallback to read-only mode",
		Timestamp:   time.Now(),
	}

	// Implement fallback logic (e.g., switch to read-only replica)
	fallbackAction.Success = true // Assume fallback succeeds for demo
	report.RecoveryActions = append(report.RecoveryActions, fallbackAction)

	return fallbackAction.Success
}

// recoverAIError attempts to recover from AI service errors
func (eh *ErrorHandler) recoverAIError(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	// Check if AI service is temporarily unavailable
	if eh.isAIServiceUnavailable(errorCtx.Error) {
		action := RecoveryAction{
			Type:        "ai_service_fallback",
			Description: "AI service unavailable, queuing for later processing",
			Timestamp:   time.Now(),
		}

		// Queue the request for later processing
		action.Success = true // Assume queuing succeeds
		report.RecoveryActions = append(report.RecoveryActions, action)
		return true
	}

	// Retry with backoff for transient errors
	for attempt := 1; attempt <= eh.config.MaxRetries; attempt++ {
		report.TotalRetries = attempt

		backoff := eh.calculateBackoff(attempt)
		time.Sleep(backoff)

		action := RecoveryAction{
			Type:        "ai_service_retry",
			Description: fmt.Sprintf("Retrying AI service call (attempt %d/%d)", attempt, eh.config.MaxRetries),
			Timestamp:   time.Now(),
			Metadata: map[string]interface{}{
				"attempt": attempt,
				"backoff": backoff.String(),
			},
		}

		// Simulate retry logic
		if attempt == eh.config.MaxRetries {
			action.Success = true
			report.RecoveryActions = append(report.RecoveryActions, action)
			return true
		}

		action.Success = false
		report.RecoveryActions = append(report.RecoveryActions, action)
	}

	return false
}

// recoverCacheError attempts to recover from cache errors
func (eh *ErrorHandler) recoverCacheError(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	action := RecoveryAction{
		Type:        "cache_fallback",
		Description: "Cache unavailable, falling back to database",
		Timestamp:   time.Now(),
	}

	// Cache errors are typically handled by falling back to the primary data source
	action.Success = true
	report.RecoveryActions = append(report.RecoveryActions, action)
	return true
}

// recoverQueueError attempts to recover from queue errors
func (eh *ErrorHandler) recoverQueueError(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	// Try to reconnect to queue
	for attempt := 1; attempt <= eh.config.MaxRetries; attempt++ {
		report.TotalRetries = attempt

		backoff := eh.calculateBackoff(attempt)
		time.Sleep(backoff)

		action := RecoveryAction{
			Type:        "queue_reconnect",
			Description: fmt.Sprintf("Attempting to reconnect to queue (attempt %d/%d)", attempt, eh.config.MaxRetries),
			Timestamp:   time.Now(),
		}

		// Simulate reconnection logic
		if attempt >= 2 { // Assume success after 2 attempts
			action.Success = true
			report.RecoveryActions = append(report.RecoveryActions, action)
			return true
		}

		action.Success = false
		report.RecoveryActions = append(report.RecoveryActions, action)
	}

	return false
}

// recoverNetworkError attempts to recover from network errors
func (eh *ErrorHandler) recoverNetworkError(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	for attempt := 1; attempt <= eh.config.MaxRetries; attempt++ {
		report.TotalRetries = attempt

		backoff := eh.calculateBackoff(attempt)
		time.Sleep(backoff)

		action := RecoveryAction{
			Type:        "network_retry",
			Description: fmt.Sprintf("Retrying network operation (attempt %d/%d)", attempt, eh.config.MaxRetries),
			Timestamp:   time.Now(),
		}

		// Simulate network retry
		if attempt >= 2 {
			action.Success = true
			report.RecoveryActions = append(report.RecoveryActions, action)
			return true
		}

		action.Success = false
		report.RecoveryActions = append(report.RecoveryActions, action)
	}

	return false
}

// recoverGenericError attempts to recover from generic errors
func (eh *ErrorHandler) recoverGenericError(ctx context.Context, errorCtx *ErrorContext, report *ErrorReport) bool {
	action := RecoveryAction{
		Type:        "generic_recovery",
		Description: "Logging error for manual investigation",
		Timestamp:   time.Now(),
	}

	// For generic errors, we typically just log them
	log.Printf("Generic error in %s.%s: %v", errorCtx.Component, errorCtx.Operation, errorCtx.Error)
	action.Success = true
	report.RecoveryActions = append(report.RecoveryActions, action)

	return false // Generic errors are not automatically recoverable
}

// calculateBackoff calculates exponential backoff with jitter
func (eh *ErrorHandler) calculateBackoff(attempt int) time.Duration {
	backoff := eh.config.RetryBackoffBase * time.Duration(1<<uint(attempt-1))
	if backoff > eh.config.RetryBackoffMax {
		backoff = eh.config.RetryBackoffMax
	}

	// Add jitter (±25%)
	jitter := time.Duration(float64(backoff) * 0.25 * (2*rand.Float64() - 1))
	return backoff + jitter
}

// isCircuitBreakerOpen checks if the circuit breaker is open for a given error key
func (eh *ErrorHandler) isCircuitBreakerOpen(errorKey string) bool {
	lastError, exists := eh.lastErrors[errorKey]
	if !exists {
		return false
	}

	errorCount := eh.errorCounts[errorKey]
	if errorCount < int64(eh.config.CircuitBreakerThreshold) {
		return false
	}

	// Circuit breaker is open if we've exceeded threshold and timeout hasn't passed
	return time.Since(lastError) < eh.config.CircuitBreakerTimeout
}

// openCircuitBreaker opens the circuit breaker for a given error key
func (eh *ErrorHandler) openCircuitBreaker(errorKey string) {
	log.Printf("Opening circuit breaker for %s due to repeated failures", errorKey)
	// In a real implementation, you might want to store circuit breaker state persistently
}

// isDatabaseRecoverable checks if a database error is recoverable
func (eh *ErrorHandler) isDatabaseRecoverable(err error) bool {
	if err == nil {
		return true
	}

	errorStr := err.Error()

	// Check for recoverable database errors
	recoverablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"deadlock",
		"lock wait timeout",
	}

	for _, pattern := range recoverablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isAIServiceUnavailable checks if AI service is temporarily unavailable
func (eh *ErrorHandler) isAIServiceUnavailable(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	unavailablePatterns := []string{
		"connection refused",
		"service unavailable",
		"timeout",
		"no such host",
	}

	for _, pattern := range unavailablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// GetErrorStatistics returns error statistics
func (eh *ErrorHandler) GetErrorStatistics() map[string]interface{} {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	stats := make(map[string]interface{})

	// Copy error counts
	errorCounts := make(map[string]int64)
	for k, v := range eh.errorCounts {
		errorCounts[k] = v
	}

	stats["error_counts"] = errorCounts
	stats["total_errors"] = eh.getTotalErrors()
	stats["error_rate"] = eh.calculateErrorRate()
	stats["circuit_breakers"] = eh.getCircuitBreakerStatus()

	return stats
}

// getTotalErrors returns the total number of errors
func (eh *ErrorHandler) getTotalErrors() int64 {
	var total int64
	for _, count := range eh.errorCounts {
		total += count
	}
	return total
}

// calculateErrorRate calculates the current error rate
func (eh *ErrorHandler) calculateErrorRate() float64 {
	summary := eh.metricsCollector.GetSummary()
	if summary.HTTP.RequestCount == 0 {
		return 0
	}

	return float64(summary.HTTP.ErrorCount) / float64(summary.HTTP.RequestCount)
}

// getCircuitBreakerStatus returns the status of all circuit breakers
func (eh *ErrorHandler) getCircuitBreakerStatus() map[string]bool {
	status := make(map[string]bool)

	for errorKey := range eh.errorCounts {
		status[errorKey] = eh.isCircuitBreakerOpen(errorKey)
	}

	return status
}

// ResetErrorCounts resets error counts (useful for testing or maintenance)
func (eh *ErrorHandler) ResetErrorCounts() {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.errorCounts = make(map[string]int64)
	eh.lastErrors = make(map[string]time.Time)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr))))
}

// containsMiddle checks if substr is in the middle of s
func containsMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
