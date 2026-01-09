package service

import (
	"context"
	"fmt"
	"time"

	"intelligent-log-analysis/internal/alert"
	"intelligent-log-analysis/internal/config"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"
)

type AlertService struct {
	repo        *repository.Repository
	alertEngine *alert.AlertEngine
	config      *config.Config
}

// NewAlertService creates a new alert service
func NewAlertService(repo *repository.Repository, cfg *config.Config) *AlertService {
	alertEngine := alert.NewAlertEngine(repo)

	// Register notifiers based on configuration
	if cfg.Alert.Email.SMTPHost != "" && cfg.Alert.Email.From != "" && len(cfg.Alert.Email.To) > 0 {
		emailConfig := alert.EmailConfig{
			SMTPHost:    cfg.Alert.Email.SMTPHost,
			SMTPPort:    cfg.Alert.Email.SMTPPort,
			Username:    cfg.Alert.Email.Username,
			Password:    cfg.Alert.Email.Password,
			From:        cfg.Alert.Email.From,
			To:          cfg.Alert.Email.To,
			UseTLS:      cfg.Alert.Email.UseTLS,
			UseStartTLS: cfg.Alert.Email.UseStartTLS,
		}
		emailNotifier := alert.NewEmailNotifier(emailConfig)
		alertEngine.RegisterNotifier("email", emailNotifier)
	}

	if cfg.Alert.DingTalk.WebhookURL != "" {
		dingTalkConfig := alert.DingTalkConfig{
			WebhookURL: cfg.Alert.DingTalk.WebhookURL,
			Secret:     cfg.Alert.DingTalk.Secret,
		}
		dingTalkNotifier := alert.NewDingTalkNotifier(dingTalkConfig)
		alertEngine.RegisterNotifier("dingtalk", dingTalkNotifier)
	}

	return &AlertService{
		repo:        repo,
		alertEngine: alertEngine,
		config:      cfg,
	}
}

// GetAlertEngine returns the alert engine instance
func (s *AlertService) GetAlertEngine() *alert.AlertEngine {
	return s.alertEngine
}

// Start starts the alert service
func (s *AlertService) Start(ctx context.Context) error {
	return s.alertEngine.Start(ctx)
}

// Stop stops the alert service
func (s *AlertService) Stop() {
	s.alertEngine.Stop()
}

// CreateAlertRule creates a new alert rule
func (s *AlertService) CreateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	// Validate alert rule
	if err := s.validateAlertRule(rule); err != nil {
		return fmt.Errorf("invalid alert rule: %w", err)
	}

	return s.repo.AlertRule.SaveAlertRule(ctx, rule)
}

// GetAlertRule retrieves an alert rule by ID
func (s *AlertService) GetAlertRule(ctx context.Context, id string) (*models.AlertRule, error) {
	return s.repo.AlertRule.GetAlertRuleByID(ctx, id)
}

// GetActiveAlertRules retrieves all active alert rules
func (s *AlertService) GetActiveAlertRules(ctx context.Context) ([]*models.AlertRule, error) {
	return s.repo.AlertRule.GetActiveAlertRules(ctx)
}

// UpdateAlertRule updates an existing alert rule
func (s *AlertService) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	// Validate alert rule
	if err := s.validateAlertRule(rule); err != nil {
		return fmt.Errorf("invalid alert rule: %w", err)
	}

	return s.repo.AlertRule.UpdateAlertRule(ctx, rule)
}

// DeleteAlertRule deletes an alert rule by ID
func (s *AlertService) DeleteAlertRule(ctx context.Context, id string) error {
	return s.repo.AlertRule.DeleteAlertRule(ctx, id)
}

// TestAlertRule tests an alert rule by sending a test notification
func (s *AlertService) TestAlertRule(ctx context.Context, rule *models.AlertRule) (map[string]string, error) {
	// Create a test alert
	testAlert := &alert.Alert{
		RuleID:          rule.ID,
		RuleName:        rule.Name,
		LogID:           "test-log-id",
		Source:          "test-source",
		Level:           "ERROR",
		Message:         "This is a test alert message for rule: " + rule.Name,
		AnomalyScore:    0.95,
		RootCauses:      []string{"Test root cause analysis", "Simulated system issue"},
		Recommendations: []string{"Test recommendation 1", "Test recommendation 2"},
		Timestamp:       time.Now(),
	}

	// Send test notifications
	results := make(map[string]string)
	for _, channel := range rule.NotificationChannels {
		if notifier, exists := s.alertEngine.GetNotifier(channel); exists {
			if err := notifier.SendAlert(ctx, testAlert); err != nil {
				results[channel] = fmt.Sprintf("Failed: %v", err)
			} else {
				results[channel] = "Success"
			}
		} else {
			results[channel] = "Notifier not configured"
		}
	}

	return results, nil
}

// validateAlertRule validates an alert rule
func (s *AlertService) validateAlertRule(rule *models.AlertRule) error {
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
	}

	for _, channel := range rule.NotificationChannels {
		if !validChannels[channel] {
			return fmt.Errorf("invalid notification channel: %s", channel)
		}
	}

	// Validate condition structure
	if err := s.validateAlertCondition(rule.Condition); err != nil {
		return fmt.Errorf("invalid alert condition: %w", err)
	}

	return nil
}

// validateAlertCondition validates the alert condition structure
func (s *AlertService) validateAlertCondition(condition map[string]interface{}) error {
	// Check required fields
	if _, ok := condition["anomaly_score_threshold"]; !ok {
		return fmt.Errorf("anomaly_score_threshold is required")
	}

	// Validate anomaly score threshold
	if threshold, ok := condition["anomaly_score_threshold"].(float64); ok {
		if threshold < 0 || threshold > 1 {
			return fmt.Errorf("anomaly_score_threshold must be between 0 and 1")
		}
	} else {
		return fmt.Errorf("anomaly_score_threshold must be a number")
	}

	// Validate optional fields
	if minCount, ok := condition["min_anomaly_count"]; ok {
		if count, ok := minCount.(float64); ok {
			if count < 1 {
				return fmt.Errorf("min_anomaly_count must be at least 1")
			}
		} else {
			return fmt.Errorf("min_anomaly_count must be a number")
		}
	}

	if timeWindow, ok := condition["time_window_minutes"]; ok {
		if window, ok := timeWindow.(float64); ok {
			if window < 1 {
				return fmt.Errorf("time_window_minutes must be at least 1")
			}
		} else {
			return fmt.Errorf("time_window_minutes must be a number")
		}
	}

	return nil
}
