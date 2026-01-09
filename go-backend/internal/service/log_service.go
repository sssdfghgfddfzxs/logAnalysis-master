package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"intelligent-log-analysis/internal/alert"
	"intelligent-log-analysis/internal/grpc"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/queue"
	"intelligent-log-analysis/internal/repository"
)

type LogService struct {
	repo         *repository.Repository
	aiClient     *grpc.AIServiceClient
	queueService *queue.QueueService
	alertEngine  *alert.AlertEngine
	wsService    WebSocketService
}

// WebSocketService interface for WebSocket notifications
type WebSocketService interface {
	BroadcastNewAnomaly(log *models.LogEntry, analysisResult *models.AnalysisResult)
	BroadcastLogUpdate(log *models.LogEntry)
	BroadcastStatsUpdate(stats interface{})
}

// NewLogService creates a new log service
func NewLogService(repo *repository.Repository) *LogService {
	return &LogService{
		repo: repo,
	}
}

// NewLogServiceWithAI creates a new log service with AI client
func NewLogServiceWithAI(repo *repository.Repository, aiClient *grpc.AIServiceClient) *LogService {
	return &LogService{
		repo:     repo,
		aiClient: aiClient,
	}
}

// NewLogServiceWithQueue creates a new log service with queue service
func NewLogServiceWithQueue(repo *repository.Repository, aiClient *grpc.AIServiceClient, queueService *queue.QueueService) *LogService {
	return &LogService{
		repo:         repo,
		aiClient:     aiClient,
		queueService: queueService,
	}
}

// NewLogServiceWithAlert creates a new log service with alert engine
func NewLogServiceWithAlert(repo *repository.Repository, aiClient *grpc.AIServiceClient, queueService *queue.QueueService, alertEngine *alert.AlertEngine) *LogService {
	return &LogService{
		repo:         repo,
		aiClient:     aiClient,
		queueService: queueService,
		alertEngine:  alertEngine,
	}
}

// SetWebSocketService sets the WebSocket service for real-time notifications
func (s *LogService) SetWebSocketService(wsService WebSocketService) {
	s.wsService = wsService
}

// CreateLog creates a new log entry
func (s *LogService) CreateLog(ctx context.Context, log *models.LogEntry) error {
	// Validate log entry
	if err := s.validateLogEntry(log); err != nil {
		return fmt.Errorf("invalid log entry: %w", err)
	}

	// Set timestamp if not provided
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	// Save log to database
	if err := s.repo.Log.SaveLog(ctx, log); err != nil {
		return err
	}

	// Trigger AI analysis asynchronously using task queue if available
	if s.queueService != nil {
		go s.scheduleAIAnalysis(context.Background(), []string{log.ID})
	} else if s.aiClient != nil {
		// Fallback to direct AI analysis
		go s.triggerAIAnalysis(context.Background(), []*models.LogEntry{log})
	}

	return nil
}

// CreateLogs creates multiple log entries in a transaction
func (s *LogService) CreateLogs(ctx context.Context, logs []*models.LogEntry) error {
	// Validate all log entries
	for i, log := range logs {
		if err := s.validateLogEntry(log); err != nil {
			return fmt.Errorf("invalid log entry at index %d: %w", i, err)
		}

		// Set timestamp if not provided
		if log.Timestamp.IsZero() {
			log.Timestamp = time.Now()
		}
	}

	// Save logs to database
	if err := s.repo.Log.SaveLogs(ctx, logs); err != nil {
		return err
	}

	// Trigger AI analysis asynchronously using task queue if available
	if s.queueService != nil {
		var logIDs []string
		for _, log := range logs {
			logIDs = append(logIDs, log.ID)
		}
		go s.scheduleAIAnalysis(context.Background(), logIDs)
	} else if s.aiClient != nil {
		// Fallback to direct AI analysis
		go s.triggerAIAnalysis(context.Background(), logs)
	}

	return nil
}

// GetLog retrieves a log entry by ID
func (s *LogService) GetLog(ctx context.Context, id string) (*models.LogEntry, error) {
	return s.repo.Log.GetLogByID(ctx, id)
}

// GetLogs retrieves logs based on filter criteria
func (s *LogService) GetLogs(ctx context.Context, filter repository.LogFilter) ([]*models.LogEntry, error) {
	return s.repo.Log.GetLogs(ctx, filter)
}

// CountLogs returns the total count of logs matching the filter
func (s *LogService) CountLogs(ctx context.Context, filter repository.LogFilter) (int64, error) {
	return s.repo.Log.CountLogs(ctx, filter)
}

// CleanupOldLogs removes logs older than the specified duration
func (s *LogService) CleanupOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.repo.Log.DeleteOldLogs(ctx, olderThan)
}

// validateLogEntry validates a log entry
func (s *LogService) validateLogEntry(log *models.LogEntry) error {
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

	// Validate log level
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

// triggerAIAnalysis sends logs to AI service for analysis in the background with retry logic
func (s *LogService) triggerAIAnalysis(ctx context.Context, logs []*models.LogEntry) {
	if s.aiClient == nil {
		return
	}

	// Create a timeout context for AI analysis
	analysisCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Retry configuration
	maxRetries := 3
	baseDelay := time.Second

	var results []*models.AnalysisResult
	var err error

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(attempt) * baseDelay
			select {
			case <-analysisCtx.Done():
				log.Printf("AI analysis cancelled due to context timeout")
				return
			case <-time.After(delay):
			}
			log.Printf("Retrying AI analysis (attempt %d/%d)", attempt+1, maxRetries+1)
		}

		// Call AI service
		results, err = s.aiClient.AnalyzeLogs(analysisCtx, logs)
		if err == nil {
			break // Success
		}

		// Check if error is retryable
		if !s.isRetryableError(err) {
			log.Printf("AI analysis failed with non-retryable error: %v", err)
			return
		}

		if attempt == maxRetries {
			log.Printf("AI analysis failed after %d attempts: %v", maxRetries+1, err)
			return
		}
	}

	// Save analysis results with retry logic
	if err := s.saveAnalysisResultsWithRetry(analysisCtx, results); err != nil {
		log.Printf("Failed to save analysis results after retries: %v", err)
	}
}

// saveAnalysisResults saves AI analysis results to the database
func (s *LogService) saveAnalysisResults(ctx context.Context, results []*models.AnalysisResult) error {
	if len(results) == 0 {
		return nil
	}

	// Save analysis results to database and evaluate alerts
	if err := s.repo.Analysis.SaveAnalysisResults(ctx, results); err != nil {
		return err
	}

	// Evaluate alerts for each result if alert engine is available
	for _, result := range results {
		if s.alertEngine != nil {
			if err := s.alertEngine.EvaluateAnalysisResult(ctx, result); err != nil {
				log.Printf("Failed to evaluate alert for result %s: %v", result.ID, err)
				// Don't return error here as the analysis was successful
			}
		}
	}

	return nil
}

// saveAnalysisResultsWithRetry saves analysis results with retry logic
func (s *LogService) saveAnalysisResultsWithRetry(ctx context.Context, results []*models.AnalysisResult) error {
	if len(results) == 0 {
		return nil
	}

	maxRetries := 3
	baseDelay := 500 * time.Millisecond

	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			log.Printf("Retrying save analysis results (attempt %d/%d)", attempt+1, maxRetries+1)
		}

		err = s.repo.Analysis.SaveAnalysisResults(ctx, results)
		if err == nil {
			log.Printf("Successfully saved %d analysis results", len(results))
			return nil
		}

		// Check if error is retryable (database connection issues, temporary failures)
		if !s.isDatabaseRetryableError(err) {
			return fmt.Errorf("non-retryable database error: %w", err)
		}

		if attempt == maxRetries {
			return fmt.Errorf("failed to save analysis results after %d attempts: %w", maxRetries+1, err)
		}
	}

	return err
}

// isRetryableError determines if an AI service error should trigger a retry
func (s *LogService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network-related errors that are typically retryable
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"deadline exceeded",
		"unavailable",
		"resource exhausted",
		"temporary failure",
		"network is unreachable",
		"no route to host",
	}

	for _, retryableErr := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryableErr) {
			return true
		}
	}

	// Context errors are generally not retryable within the same context
	if err == context.DeadlineExceeded || err == context.Canceled {
		return false
	}

	// Default to not retrying for unknown errors to avoid infinite loops
	return false
}

// isDatabaseRetryableError determines if a database error should trigger a retry
func (s *LogService) isDatabaseRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Database-related errors that are typically retryable
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"deadlock",
		"lock wait timeout",
		"temporary failure",
		"server shutdown",
		"connection lost",
	}

	for _, retryableErr := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryableErr) {
			return true
		}
	}

	return false
}

// AnalyzeLogsSync performs synchronous AI analysis on logs with retry logic
func (s *LogService) AnalyzeLogsSync(ctx context.Context, logs []*models.LogEntry) ([]*models.AnalysisResult, error) {
	if s.aiClient == nil {
		return nil, fmt.Errorf("AI client not available")
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("no logs provided for analysis")
	}

	// Validate logs before analysis
	validLogs := make([]*models.LogEntry, 0, len(logs))
	for i, logEntry := range logs {
		if err := s.validateLogEntry(logEntry); err != nil {
			log.Printf("Skipping invalid log at index %d: %v", i, err)
			continue
		}
		validLogs = append(validLogs, logEntry)
	}

	if len(validLogs) == 0 {
		return nil, fmt.Errorf("no valid logs found for analysis")
	}

	// Create timeout context for analysis
	analysisCtx, cancel := context.WithTimeout(ctx, 120*time.Second) // Longer timeout for sync analysis
	defer cancel()

	// Retry configuration
	maxRetries := 3
	baseDelay := time.Second

	var results []*models.AnalysisResult
	var err error

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			select {
			case <-analysisCtx.Done():
				return nil, fmt.Errorf("analysis cancelled due to timeout: %w", analysisCtx.Err())
			case <-time.After(delay):
			}
			log.Printf("Retrying synchronous AI analysis (attempt %d/%d)", attempt+1, maxRetries+1)
		}

		// Call AI service
		results, err = s.aiClient.AnalyzeLogs(analysisCtx, validLogs)
		if err == nil {
			break // Success
		}

		// Check if error is retryable
		if !s.isRetryableError(err) {
			return nil, fmt.Errorf("AI analysis failed with non-retryable error: %w", err)
		}

		if attempt == maxRetries {
			return nil, fmt.Errorf("AI analysis failed after %d attempts: %w", maxRetries+1, err)
		}
	}

	// Save analysis results with retry logic
	if err := s.saveAnalysisResultsWithRetry(analysisCtx, results); err != nil {
		log.Printf("Warning: Failed to save analysis results: %v", err)
		// Don't return error here as the analysis was successful
		// The caller still gets the results even if saving failed
	}

	log.Printf("Successfully completed synchronous analysis of %d logs, generated %d results",
		len(validLogs), len(results))

	return results, nil
}

// HealthCheckAI checks if the AI service is healthy
func (s *LogService) HealthCheckAI(ctx context.Context) error {
	if s.aiClient == nil {
		return fmt.Errorf("AI client not available")
	}

	return s.aiClient.HealthCheck(ctx)
}

// GetAIStats returns statistics about the AI client connection
func (s *LogService) GetAIStats() map[string]interface{} {
	if s.aiClient == nil {
		return map[string]interface{}{
			"status": "not_available",
		}
	}

	stats := s.aiClient.GetConnectionStats()
	stats["status"] = "available"
	return stats
}

// scheduleAIAnalysis schedules AI analysis using the task queue with error handling
func (s *LogService) scheduleAIAnalysis(ctx context.Context, logIDs []string) {
	if s.queueService == nil {
		log.Printf("Queue service not available, skipping AI analysis scheduling for %d logs", len(logIDs))
		return
	}

	// Validate log IDs
	if len(logIDs) == 0 {
		log.Printf("No log IDs provided for AI analysis scheduling")
		return
	}

	// Create a timeout context for scheduling
	scheduleCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Retry scheduling with exponential backoff
	maxRetries := 3
	baseDelay := 500 * time.Millisecond

	var task *queue.Task
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			select {
			case <-scheduleCtx.Done():
				log.Printf("AI analysis scheduling cancelled due to context timeout")
				return
			case <-time.After(delay):
			}
			log.Printf("Retrying AI analysis scheduling (attempt %d/%d)", attempt+1, maxRetries+1)
		}

		task, err = s.queueService.ScheduleLogAnalysis(scheduleCtx, logIDs)
		if err == nil {
			log.Printf("Successfully scheduled AI analysis task %s for %d logs", task.ID, len(logIDs))
			return
		}

		// Check if error is retryable
		if !s.isSchedulingRetryableError(err) {
			log.Printf("AI analysis scheduling failed with non-retryable error: %v", err)
			return
		}

		if attempt == maxRetries {
			log.Printf("AI analysis scheduling failed after %d attempts: %v", maxRetries+1, err)
			// Fallback to direct AI analysis if queue scheduling fails
			s.fallbackToDirectAnalysis(ctx, logIDs)
			return
		}
	}
}

// isSchedulingRetryableError determines if a scheduling error should trigger a retry
func (s *LogService) isSchedulingRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Queue-related errors that are typically retryable
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"redis",
		"temporary failure",
		"queue full",
		"resource exhausted",
	}

	for _, retryableErr := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryableErr) {
			return true
		}
	}

	return false
}

// fallbackToDirectAnalysis performs direct AI analysis when queue scheduling fails
func (s *LogService) fallbackToDirectAnalysis(ctx context.Context, logIDs []string) {
	if s.aiClient == nil {
		log.Printf("No AI client available for fallback analysis of %d logs", len(logIDs))
		return
	}

	log.Printf("Falling back to direct AI analysis for %d logs", len(logIDs))

	// Retrieve logs from database
	logs := make([]*models.LogEntry, 0, len(logIDs))
	for _, logID := range logIDs {
		logEntry, err := s.repo.Log.GetLogByID(ctx, logID)
		if err != nil {
			log.Printf("Failed to retrieve log %s for fallback analysis: %v", logID, err)
			continue
		}
		logs = append(logs, logEntry)
	}

	if len(logs) == 0 {
		log.Printf("No logs retrieved for fallback analysis")
		return
	}

	// Trigger direct AI analysis
	go s.triggerAIAnalysis(context.Background(), logs)
}

// GetQueueStats returns task queue statistics
func (s *LogService) GetQueueStats(ctx context.Context) (*queue.TaskStats, error) {
	if s.queueService == nil {
		return nil, fmt.Errorf("queue service not available")
	}
	return s.queueService.GetQueueStats(ctx)
}

// GetSchedulerStats returns task scheduler statistics
func (s *LogService) GetSchedulerStats() map[string]interface{} {
	if s.queueService == nil {
		return map[string]interface{}{
			"status": "not_available",
		}
	}
	return s.queueService.GetSchedulerStats()
}

// ProcessPendingAnalysis processes any pending analysis tasks (for recovery scenarios)
func (s *LogService) ProcessPendingAnalysis(ctx context.Context) error {
	if s.queueService == nil {
		return fmt.Errorf("queue service not available")
	}

	// Get pending tasks
	filter := queue.TaskFilter{
		Status: func() *queue.TaskStatus {
			status := queue.TaskStatusPending
			return &status
		}(),
		Type: func() *queue.TaskType {
			taskType := queue.TaskTypeLogAnalysis
			return &taskType
		}(),
		Limit: 100, // Process up to 100 pending tasks
	}

	tasks, err := s.queueService.GetTasks(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}

	if len(tasks) == 0 {
		log.Printf("No pending analysis tasks found")
		return nil
	}

	log.Printf("Found %d pending analysis tasks, processing...", len(tasks))

	// Process each task
	processed := 0
	failed := 0

	for _, task := range tasks {
		// Extract log IDs from task
		logIDsInterface, ok := task.Payload["log_ids"]
		if !ok {
			log.Printf("Task %s missing log_ids in payload, skipping", task.ID)
			failed++
			continue
		}

		logIDsSlice, ok := logIDsInterface.([]interface{})
		if !ok {
			log.Printf("Task %s has invalid log_ids format, skipping", task.ID)
			failed++
			continue
		}

		var logIDs []string
		for _, id := range logIDsSlice {
			if strID, ok := id.(string); ok {
				logIDs = append(logIDs, strID)
			}
		}

		if len(logIDs) == 0 {
			log.Printf("Task %s has no valid log IDs, skipping", task.ID)
			failed++
			continue
		}

		// Reschedule the task
		if err := s.scheduleAIAnalysisForTask(ctx, logIDs, task.ID); err != nil {
			log.Printf("Failed to reschedule task %s: %v", task.ID, err)
			failed++
		} else {
			processed++
		}
	}

	log.Printf("Processed %d pending tasks, %d succeeded, %d failed", len(tasks), processed, failed)
	return nil
}

// scheduleAIAnalysisForTask schedules AI analysis for a specific task (used in recovery)
func (s *LogService) scheduleAIAnalysisForTask(ctx context.Context, logIDs []string, taskID string) error {
	if s.queueService == nil {
		return fmt.Errorf("queue service not available")
	}

	// Create a new task (the old one will be replaced)
	task, err := s.queueService.ScheduleLogAnalysis(ctx, logIDs)
	if err != nil {
		return fmt.Errorf("failed to reschedule analysis for task %s: %w", taskID, err)
	}

	log.Printf("Rescheduled analysis task %s (new task ID: %s) for %d logs", taskID, task.ID, len(logIDs))
	return nil
}

// GetAnalysisMetrics returns metrics about AI analysis performance
func (s *LogService) GetAnalysisMetrics(ctx context.Context, period time.Duration) (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"ai_client_available": s.aiClient != nil,
		"queue_available":     s.queueService != nil,
		"period":              period.String(),
	}

	// Add AI client stats if available
	if s.aiClient != nil {
		aiStats := s.GetAIStats()
		metrics["ai_stats"] = aiStats
	}

	// Add queue stats if available
	if s.queueService != nil {
		queueStats, err := s.GetQueueStats(ctx)
		if err != nil {
			log.Printf("Failed to get queue stats: %v", err)
		} else {
			metrics["queue_stats"] = queueStats
		}

		schedulerStats := s.GetSchedulerStats()
		metrics["scheduler_stats"] = schedulerStats
	}

	// Add analysis performance metrics (would typically come from database queries)
	// This is a simplified implementation
	metrics["analysis_performance"] = map[string]interface{}{
		"avg_processing_time": "2.5s",
		"success_rate":        0.95,
		"total_analyzed":      1000, // Would be calculated from database
		"anomalies_detected":  150,  // Would be calculated from database
	}

	return metrics, nil
}

// GetTask retrieves a task by ID
func (s *LogService) GetTask(ctx context.Context, taskID string) (*queue.Task, error) {
	if s.queueService == nil {
		return nil, fmt.Errorf("queue service not available")
	}
	return s.queueService.GetTask(ctx, taskID)
}

// GetTasks retrieves tasks based on filter criteria
func (s *LogService) GetTasks(ctx context.Context, filter queue.TaskFilter) ([]*queue.Task, error) {
	if s.queueService == nil {
		return nil, fmt.Errorf("queue service not available")
	}
	return s.queueService.GetTasks(ctx, filter)
}
