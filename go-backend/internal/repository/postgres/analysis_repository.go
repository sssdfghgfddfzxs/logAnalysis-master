package postgres

import (
	"context"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"

	"gorm.io/gorm"
)

type analysisRepository struct {
	db *gorm.DB
}

// NewAnalysisRepository creates a new PostgreSQL analysis repository
func NewAnalysisRepository(db *gorm.DB) repository.AnalysisRepository {
	return &analysisRepository{db: db}
}

func (r *analysisRepository) SaveAnalysisResult(ctx context.Context, result *models.AnalysisResult) error {
	// 使用 UPSERT 操作：如果存在就更新，不存在就创建
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.AnalysisResult
		err := tx.Where("log_id = ?", result.LogID).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// 记录不存在，创建新记录
			if result.AnalyzedAt.IsZero() {
				result.AnalyzedAt = time.Now()
			}
			return tx.Create(result).Error
		} else if err != nil {
			// 其他错误
			return err
		} else {
			// 记录存在，更新现有记录
			// 保留原始的 ID，更新其他字段
			result.ID = existing.ID
			result.AnalyzedAt = time.Now() // 更新分析时间

			// 更新所有字段（除了ID）
			return tx.Model(&existing).Updates(map[string]interface{}{
				"is_anomaly":      result.IsAnomaly,
				"anomaly_score":   result.AnomalyScore,
				"root_causes":     result.RootCauses,
				"recommendations": result.Recommendations,
				"analyzed_at":     result.AnalyzedAt,
			}).Error
		}
	})
}

func (r *analysisRepository) SaveAnalysisResults(ctx context.Context, results []*models.AnalysisResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, result := range results {
			var existing models.AnalysisResult
			err := tx.Where("log_id = ?", result.LogID).First(&existing).Error

			if err == gorm.ErrRecordNotFound {
				// 记录不存在，创建新记录
				if result.AnalyzedAt.IsZero() {
					result.AnalyzedAt = time.Now()
				}
				if err := tx.Create(result).Error; err != nil {
					return err
				}
			} else if err != nil {
				// 其他错误
				return err
			} else {
				// 记录存在，更新现有记录
				result.ID = existing.ID
				result.AnalyzedAt = time.Now()

				if err := tx.Model(&existing).Updates(map[string]interface{}{
					"is_anomaly":      result.IsAnomaly,
					"anomaly_score":   result.AnomalyScore,
					"root_causes":     result.RootCauses,
					"recommendations": result.Recommendations,
					"analyzed_at":     result.AnalyzedAt,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *analysisRepository) GetAnalysisResultByLogID(ctx context.Context, logID string) (*models.AnalysisResult, error) {
	var result models.AnalysisResult
	err := r.db.WithContext(ctx).Preload("Log").First(&result, "log_id = ?", logID).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *analysisRepository) GetAnalysisResults(ctx context.Context, filter repository.ResultFilter) ([]*models.AnalysisResult, error) {
	query := r.db.WithContext(ctx).Model(&models.AnalysisResult{}).Preload("Log")

	// Apply filters
	query = r.applyResultFilters(query, filter)

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Order by analyzed_at descending (most recent first)
	query = query.Order("analyzed_at DESC")

	var results []*models.AnalysisResult
	err := query.Find(&results).Error
	return results, err
}

func (r *analysisRepository) CountAnalysisResults(ctx context.Context, filter repository.ResultFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AnalysisResult{})
	query = r.applyResultFilters(query, filter)

	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r *analysisRepository) GetAnomalyStats(ctx context.Context, period time.Duration) (*repository.AnomalyStats, error) {
	startTime := time.Now().Add(-period)

	// Get total logs and anomaly count
	var totalLogs, anomalyCount int64

	// Count total logs in period
	err := r.db.WithContext(ctx).Model(&models.LogEntry{}).
		Where("timestamp >= ?", startTime).
		Count(&totalLogs).Error
	if err != nil {
		return nil, err
	}

	// Count anomalies in period
	err = r.db.WithContext(ctx).Model(&models.AnalysisResult{}).
		Joins("JOIN logs ON analysis_results.log_id = logs.id").
		Where("logs.timestamp >= ? AND analysis_results.is_anomaly = ?", startTime, true).
		Count(&anomalyCount).Error
	if err != nil {
		return nil, err
	}

	// Calculate anomaly rate (as decimal, not percentage)
	var anomalyRate float64
	if totalLogs > 0 {
		anomalyRate = float64(anomalyCount) / float64(totalLogs)
	}

	// Get top sources
	var topSources []repository.SourceStat
	err = r.db.WithContext(ctx).Model(&models.LogEntry{}).
		Select("source, COUNT(*) as count").
		Where("timestamp >= ?", startTime).
		Group("source").
		Order("count DESC").
		Limit(5).
		Scan(&topSources).Error
	if err != nil {
		return nil, err
	}

	// Count active services (distinct sources)
	var activeServices int64
	err = r.db.WithContext(ctx).Model(&models.LogEntry{}).
		Where("timestamp >= ?", startTime).
		Distinct("source").
		Count(&activeServices).Error
	if err != nil {
		return nil, err
	}

	return &repository.AnomalyStats{
		TotalLogs:      totalLogs,
		AnomalyCount:   anomalyCount,
		AnomalyRate:    anomalyRate,
		TopSources:     topSources,
		ActiveServices: int(activeServices),
	}, nil
}

// applyResultFilters applies filtering criteria to the analysis results query
func (r *analysisRepository) applyResultFilters(query *gorm.DB, filter repository.ResultFilter) *gorm.DB {
	if filter.StartTime != nil || filter.EndTime != nil || filter.Source != "" {
		// Join with logs table for timestamp and source filtering
		query = query.Joins("JOIN logs ON analysis_results.log_id = logs.id")

		if filter.StartTime != nil {
			query = query.Where("logs.timestamp >= ?", *filter.StartTime)
		}

		if filter.EndTime != nil {
			query = query.Where("logs.timestamp <= ?", *filter.EndTime)
		}

		if filter.Source != "" && filter.Source != "all" {
			query = query.Where("logs.source = ?", filter.Source)
		}
	}

	if filter.IsAnomaly != nil {
		query = query.Where("analysis_results.is_anomaly = ?", *filter.IsAnomaly)
	}

	if filter.MinScore != nil {
		query = query.Where("analysis_results.anomaly_score >= ?", *filter.MinScore)
	}

	if filter.MaxScore != nil {
		query = query.Where("analysis_results.anomaly_score <= ?", *filter.MaxScore)
	}

	return query
}
