package postgres

import (
	"context"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"

	"gorm.io/gorm"
)

type alertRuleRepository struct {
	db *gorm.DB
}

// NewAlertRuleRepository creates a new PostgreSQL alert rule repository
func NewAlertRuleRepository(db *gorm.DB) repository.AlertRuleRepository {
	return &alertRuleRepository{db: db}
}

func (r *alertRuleRepository) SaveAlertRule(ctx context.Context, rule *models.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *alertRuleRepository) GetAlertRuleByID(ctx context.Context, id string) (*models.AlertRule, error) {
	var rule models.AlertRule
	err := r.db.WithContext(ctx).First(&rule, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *alertRuleRepository) GetActiveAlertRules(ctx context.Context) ([]*models.AlertRule, error) {
	var rules []*models.AlertRule
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&rules).Error
	return rules, err
}

func (r *alertRuleRepository) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *alertRuleRepository) DeleteAlertRule(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.AlertRule{}, "id = ?", id).Error
}
