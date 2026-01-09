package service

import (
	"context"
	"fmt"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"
)

type AlertRuleService struct {
	repo *repository.Repository
}

// NewAlertRuleService creates a new alert rule service
func NewAlertRuleService(repo *repository.Repository) *AlertRuleService {
	return &AlertRuleService{
		repo: repo,
	}
}

// CreateAlertRule creates a new alert rule
func (s *AlertRuleService) CreateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	// Validate alert rule
	if err := s.validateAlertRule(rule); err != nil {
		return fmt.Errorf("invalid alert rule: %w", err)
	}

	return s.repo.AlertRule.SaveAlertRule(ctx, rule)
}

// GetAlertRule retrieves an alert rule by ID
func (s *AlertRuleService) GetAlertRule(ctx context.Context, id string) (*models.AlertRule, error) {
	return s.repo.AlertRule.GetAlertRuleByID(ctx, id)
}

// GetActiveAlertRules retrieves all active alert rules
func (s *AlertRuleService) GetActiveAlertRules(ctx context.Context) ([]*models.AlertRule, error) {
	return s.repo.AlertRule.GetActiveAlertRules(ctx)
}

// UpdateAlertRule updates an existing alert rule
func (s *AlertRuleService) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	// Validate alert rule
	if err := s.validateAlertRule(rule); err != nil {
		return fmt.Errorf("invalid alert rule: %w", err)
	}

	return s.repo.AlertRule.UpdateAlertRule(ctx, rule)
}

// DeleteAlertRule deletes an alert rule by ID
func (s *AlertRuleService) DeleteAlertRule(ctx context.Context, id string) error {
	return s.repo.AlertRule.DeleteAlertRule(ctx, id)
}

// TestAlertRule tests an alert rule by sending a test notification
func (s *AlertRuleService) TestAlertRule(ctx context.Context, rule *models.AlertRule) (map[string]string, error) {
	// This method would need access to the alert engine to send test notifications
	// For now, we'll return a placeholder response
	results := make(map[string]string)

	for _, channel := range rule.NotificationChannels {
		switch channel {
		case "email":
			results[channel] = "Test email notification would be sent"
		case "dingtalk":
			results[channel] = "Test DingTalk notification would be sent"
		default:
			results[channel] = fmt.Sprintf("Unknown channel: %s", channel)
		}
	}

	return results, nil
}

// validateAlertRule validates an alert rule
func (s *AlertRuleService) validateAlertRule(rule *models.AlertRule) error {
	if rule == nil {
		return fmt.Errorf("alert rule cannot be nil")
	}

	if rule.Name == "" {
		return fmt.Errorf("alert rule name cannot be empty")
	}

	if rule.Condition == nil || len(rule.Condition) == 0 {
		return fmt.Errorf("alert rule condition cannot be empty")
	}

	if len(rule.NotificationChannels) == 0 {
		return fmt.Errorf("alert rule must have at least one notification channel")
	}

	// Validate notification channels
	validChannels := map[string]bool{
		"email":    true,
		"dingtalk": true,
		"webhook":  true,
	}

	for _, channel := range rule.NotificationChannels {
		if !validChannels[channel] {
			return fmt.Errorf("invalid notification channel: %s", channel)
		}
	}

	return nil
}
