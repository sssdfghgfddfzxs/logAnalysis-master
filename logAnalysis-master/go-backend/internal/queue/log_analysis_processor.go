package queue

import (
	"context"
	"fmt"
	"log"

	"intelligent-log-analysis/internal/grpc"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"
)

// LogAnalysisProcessor processes log analysis tasks
type LogAnalysisProcessor struct {
	repo        *repository.Repository
	aiClient    *grpc.AIServiceClient
	alertEngine AlertEngine // 添加告警引擎接口
}

// AlertEngine interface for alert evaluation
type AlertEngine interface {
	EvaluateAnalysisResult(ctx context.Context, result *models.AnalysisResult) error
}

// NewLogAnalysisProcessor creates a new log analysis processor
func NewLogAnalysisProcessor(repo *repository.Repository, aiClient *grpc.AIServiceClient) *LogAnalysisProcessor {
	return &LogAnalysisProcessor{
		repo:     repo,
		aiClient: aiClient,
	}
}

// NewLogAnalysisProcessorWithAlert creates a new log analysis processor with alert engine
func NewLogAnalysisProcessorWithAlert(repo *repository.Repository, aiClient *grpc.AIServiceClient, alertEngine AlertEngine) *LogAnalysisProcessor {
	return &LogAnalysisProcessor{
		repo:        repo,
		aiClient:    aiClient,
		alertEngine: alertEngine,
	}
}

// ProcessTask processes a log analysis task
func (p *LogAnalysisProcessor) ProcessTask(ctx context.Context, task *Task) error {
	if task.Type != TaskTypeLogAnalysis {
		return fmt.Errorf("unsupported task type: %s", task.Type)
	}

	// Extract log IDs from task payload
	logIDsInterface, ok := task.Payload["log_ids"]
	if !ok {
		return fmt.Errorf("missing log_ids in task payload")
	}

	logIDsSlice, ok := logIDsInterface.([]interface{})
	if !ok {
		return fmt.Errorf("invalid log_ids format in task payload")
	}

	var logIDs []string
	for _, id := range logIDsSlice {
		if strID, ok := id.(string); ok {
			logIDs = append(logIDs, strID)
		}
	}

	if len(logIDs) == 0 {
		return fmt.Errorf("no valid log IDs found in task payload")
	}

	// Retrieve logs from database
	logs, err := p.getLogsByIDs(ctx, logIDs)
	if err != nil {
		return fmt.Errorf("failed to retrieve logs: %w", err)
	}

	if len(logs) == 0 {
		return fmt.Errorf("no logs found for the given IDs")
	}

	// Call AI service for analysis
	results, err := p.aiClient.AnalyzeLogs(ctx, logs)
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	// Save analysis results to database
	if err := p.repo.Analysis.SaveAnalysisResults(ctx, results); err != nil {
		return fmt.Errorf("failed to save analysis results: %w", err)
	}

	// Evaluate alerts for each result if alert engine is available
	if p.alertEngine != nil {
		for _, result := range results {
			if err := p.alertEngine.EvaluateAnalysisResult(ctx, result); err != nil {
				log.Printf("Failed to evaluate alert for result %s: %v", result.ID, err)
				// Don't return error here as the analysis was successful
			}
		}
	}

	// Update task result
	task.Result = map[string]interface{}{
		"analyzed_logs":    len(logs),
		"analysis_results": len(results),
		"anomalies_found":  p.countAnomalies(results),
	}

	log.Printf("Successfully processed log analysis task %s: analyzed %d logs, found %d anomalies",
		task.ID, len(logs), p.countAnomalies(results))

	return nil
}

// CanProcess checks if the processor can handle the task type
func (p *LogAnalysisProcessor) CanProcess(taskType TaskType) bool {
	return taskType == TaskTypeLogAnalysis
}

// GetProcessorName returns the name of the processor
func (p *LogAnalysisProcessor) GetProcessorName() string {
	return "LogAnalysisProcessor"
}

// getLogsByIDs retrieves logs by their IDs
func (p *LogAnalysisProcessor) getLogsByIDs(ctx context.Context, logIDs []string) ([]*models.LogEntry, error) {
	var logs []*models.LogEntry

	for _, logID := range logIDs {
		logEntry, err := p.repo.Log.GetLogByID(ctx, logID)
		if err != nil {
			// Log the error but continue with other logs
			log.Printf("Failed to retrieve log %s: %v", logID, err)
			continue
		}
		logs = append(logs, logEntry)
	}

	return logs, nil
}

// countAnomalies counts the number of anomalies in the analysis results
func (p *LogAnalysisProcessor) countAnomalies(results []*models.AnalysisResult) int {
	count := 0
	for _, result := range results {
		if result.IsAnomaly {
			count++
		}
	}
	return count
}
