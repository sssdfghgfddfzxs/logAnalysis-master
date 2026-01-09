package repository

import (
	"context"
	"time"

	"intelligent-log-analysis/internal/models"
)

// LogFilter represents filtering criteria for log queries
type LogFilter struct {
	StartTime    *time.Time
	EndTime      *time.Time
	Level        string
	Source       string
	AnalyzedOnly *bool // Filter by analysis status
	AnomalyOnly  *bool // Filter by anomaly status (requires analysis)
	Limit        int
	Offset       int
}

// ResultFilter represents filtering criteria for analysis result queries
type ResultFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
	IsAnomaly *bool
	MinScore  *float64
	MaxScore  *float64
	Source    string
	Limit     int
	Offset    int
}

// LogRepository defines the interface for log data access operations
type LogRepository interface {
	// SaveLog stores a single log entry
	SaveLog(ctx context.Context, log *models.LogEntry) error

	// SaveLogs stores multiple log entries in a transaction
	SaveLogs(ctx context.Context, logs []*models.LogEntry) error

	// GetLogByID retrieves a log entry by its ID
	GetLogByID(ctx context.Context, id string) (*models.LogEntry, error)

	// GetLogs retrieves logs based on filter criteria
	GetLogs(ctx context.Context, filter LogFilter) ([]*models.LogEntry, error)

	// CountLogs returns the total count of logs matching the filter
	CountLogs(ctx context.Context, filter LogFilter) (int64, error)

	// DeleteOldLogs removes logs older than the specified duration
	DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error)
}

// AnalysisRepository defines the interface for analysis result data access operations
type AnalysisRepository interface {
	// SaveAnalysisResult stores a single analysis result
	SaveAnalysisResult(ctx context.Context, result *models.AnalysisResult) error

	// SaveAnalysisResults stores multiple analysis results in a transaction
	SaveAnalysisResults(ctx context.Context, results []*models.AnalysisResult) error

	// GetAnalysisResultByLogID retrieves analysis result by log ID
	GetAnalysisResultByLogID(ctx context.Context, logID string) (*models.AnalysisResult, error)

	// GetAnalysisResults retrieves analysis results based on filter criteria
	GetAnalysisResults(ctx context.Context, filter ResultFilter) ([]*models.AnalysisResult, error)

	// CountAnalysisResults returns the total count of results matching the filter
	CountAnalysisResults(ctx context.Context, filter ResultFilter) (int64, error)

	// GetAnomalyStats returns anomaly statistics for dashboard
	GetAnomalyStats(ctx context.Context, period time.Duration) (*AnomalyStats, error)
}

// AlertRuleRepository defines the interface for alert rule data access operations
type AlertRuleRepository interface {
	// SaveAlertRule stores a single alert rule
	SaveAlertRule(ctx context.Context, rule *models.AlertRule) error

	// GetAlertRuleByID retrieves an alert rule by its ID
	GetAlertRuleByID(ctx context.Context, id string) (*models.AlertRule, error)

	// GetActiveAlertRules retrieves all active alert rules
	GetActiveAlertRules(ctx context.Context) ([]*models.AlertRule, error)

	// UpdateAlertRule updates an existing alert rule
	UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error

	// DeleteAlertRule deletes an alert rule by ID
	DeleteAlertRule(ctx context.Context, id string) error
}

// AnomalyStats represents anomaly statistics for dashboard
type AnomalyStats struct {
	TotalLogs      int64        `json:"total_logs"`
	AnomalyCount   int64        `json:"anomaly_count"`
	AnomalyRate    float64      `json:"anomaly_rate"`
	TopSources     []SourceStat `json:"top_sources"`
	ActiveServices int          `json:"active_services"`
}

// SourceStat represents statistics for a specific log source
type SourceStat struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

// Repository aggregates all repository interfaces
type Repository struct {
	Log       LogRepository
	Analysis  AnalysisRepository
	AlertRule AlertRuleRepository
}
