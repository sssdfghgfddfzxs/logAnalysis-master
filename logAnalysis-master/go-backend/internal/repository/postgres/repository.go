package postgres

import (
	"intelligent-log-analysis/internal/repository"

	"gorm.io/gorm"
)

// NewRepository creates a new repository instance with PostgreSQL implementations
func NewRepository(db *gorm.DB) *repository.Repository {
	return &repository.Repository{
		Log:       NewLogRepository(db),
		Analysis:  NewAnalysisRepository(db),
		AlertRule: NewAlertRuleRepository(db),
	}
}
