package service

import (
	"context"
	"fmt"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"
)

type AnalysisService struct {
	repo *repository.Repository
}

// NewAnalysisService creates a new analysis service
func NewAnalysisService(repo *repository.Repository) *AnalysisService {
	return &AnalysisService{
		repo: repo,
	}
}

// CreateAnalysisResult creates a new analysis result
func (s *AnalysisService) CreateAnalysisResult(ctx context.Context, result *models.AnalysisResult) error {
	// Validate analysis result
	if err := s.validateAnalysisResult(result); err != nil {
		return fmt.Errorf("invalid analysis result: %w", err)
	}

	return s.repo.Analysis.SaveAnalysisResult(ctx, result)
}

// CreateAnalysisResults creates multiple analysis results in a transaction
func (s *AnalysisService) CreateAnalysisResults(ctx context.Context, results []*models.AnalysisResult) error {
	// Validate all analysis results
	for i, result := range results {
		if err := s.validateAnalysisResult(result); err != nil {
			return fmt.Errorf("invalid analysis result at index %d: %w", i, err)
		}
	}

	return s.repo.Analysis.SaveAnalysisResults(ctx, results)
}

// GetAnalysisResultByLogID retrieves analysis result by log ID
func (s *AnalysisService) GetAnalysisResultByLogID(ctx context.Context, logID string) (*models.AnalysisResult, error) {
	return s.repo.Analysis.GetAnalysisResultByLogID(ctx, logID)
}

// GetAnalysisResults retrieves analysis results based on filter criteria
func (s *AnalysisService) GetAnalysisResults(ctx context.Context, filter repository.ResultFilter) ([]*models.AnalysisResult, error) {
	return s.repo.Analysis.GetAnalysisResults(ctx, filter)
}

// CountAnalysisResults returns the total count of results matching the filter
func (s *AnalysisService) CountAnalysisResults(ctx context.Context, filter repository.ResultFilter) (int64, error) {
	return s.repo.Analysis.CountAnalysisResults(ctx, filter)
}

// GetAnomalyStats returns anomaly statistics for dashboard
func (s *AnalysisService) GetAnomalyStats(ctx context.Context, period time.Duration) (*repository.AnomalyStats, error) {
	return s.repo.Analysis.GetAnomalyStats(ctx, period)
}

// validateAnalysisResult validates an analysis result
func (s *AnalysisService) validateAnalysisResult(result *models.AnalysisResult) error {
	if result == nil {
		return fmt.Errorf("analysis result cannot be nil")
	}

	if result.LogID == "" {
		return fmt.Errorf("log ID cannot be empty")
	}

	// Validate anomaly score range
	if result.AnomalyScore < 0 || result.AnomalyScore > 1 {
		return fmt.Errorf("anomaly score must be between 0 and 1, got: %f", result.AnomalyScore)
	}

	return nil
}
