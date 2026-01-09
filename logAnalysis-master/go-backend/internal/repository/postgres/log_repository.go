package postgres

import (
	"context"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"

	"gorm.io/gorm"
)

type logRepository struct {
	db *gorm.DB
}

// NewLogRepository creates a new PostgreSQL log repository
func NewLogRepository(db *gorm.DB) repository.LogRepository {
	return &logRepository{db: db}
}

func (r *logRepository) SaveLog(ctx context.Context, log *models.LogEntry) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *logRepository) SaveLogs(ctx context.Context, logs []*models.LogEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(logs, 100).Error
	})
}

func (r *logRepository) GetLogByID(ctx context.Context, id string) (*models.LogEntry, error) {
	var log models.LogEntry
	err := r.db.WithContext(ctx).First(&log, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *logRepository) GetLogs(ctx context.Context, filter repository.LogFilter) ([]*models.LogEntry, error) {
	query := r.db.WithContext(ctx).Model(&models.LogEntry{})

	// Apply filters
	query = r.applyLogFilters(query, filter)

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Order by timestamp descending (most recent first)
	query = query.Order("timestamp DESC")

	var logs []*models.LogEntry
	err := query.Find(&logs).Error
	return logs, err
}

func (r *logRepository) CountLogs(ctx context.Context, filter repository.LogFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.LogEntry{})
	query = r.applyLogFilters(query, filter)

	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *logRepository) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result := r.db.WithContext(ctx).Where("timestamp < ?", cutoffTime).Delete(&models.LogEntry{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// applyLogFilters applies filtering criteria to the query
func (r *logRepository) applyLogFilters(query *gorm.DB, filter repository.LogFilter) *gorm.DB {
	if filter.StartTime != nil {
		query = query.Where("timestamp >= ?", *filter.StartTime)
	}

	if filter.EndTime != nil {
		query = query.Where("timestamp <= ?", *filter.EndTime)
	}

	if filter.Level != "" && filter.Level != "all" {
		query = query.Where("level = ?", filter.Level)
	}

	if filter.Source != "" && filter.Source != "all" {
		query = query.Where("source = ?", filter.Source)
	}

	// Filter by analysis status
	if filter.AnalyzedOnly != nil {
		if *filter.AnalyzedOnly {
			// Only logs that have analysis results
			query = query.Where("EXISTS (SELECT 1 FROM analysis_results WHERE analysis_results.log_id = logs.id)")
		} else {
			// Only logs that don't have analysis results
			query = query.Where("NOT EXISTS (SELECT 1 FROM analysis_results WHERE analysis_results.log_id = logs.id)")
		}
	}

	// Filter by anomaly status (requires analysis)
	if filter.AnomalyOnly != nil && *filter.AnomalyOnly {
		// Only logs that are analyzed and marked as anomalies
		query = query.Where("EXISTS (SELECT 1 FROM analysis_results WHERE analysis_results.log_id = logs.id AND analysis_results.is_anomaly = true)")
	}

	return query
}
