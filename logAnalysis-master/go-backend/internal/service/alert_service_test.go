package service

import (
	"context"
	"testing"

	"intelligent-log-analysis/internal/config"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAlertRuleRepository for testing
type MockAlertRuleRepository struct {
	mock.Mock
}

func (m *MockAlertRuleRepository) SaveAlertRule(ctx context.Context, rule *models.AlertRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAlertRuleRepository) GetAlertRuleByID(ctx context.Context, id string) (*models.AlertRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AlertRule), args.Error(1)
}

func (m *MockAlertRuleRepository) GetActiveAlertRules(ctx context.Context) ([]*models.AlertRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.AlertRule), args.Error(1)
}

func (m *MockAlertRuleRepository) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAlertRuleRepository) DeleteAlertRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAlertService_CreateAlertRule(t *testing.T) {
	// Setup
	mockAlertRepo := &MockAlertRuleRepository{}
	mockRepo := &repository.Repository{
		AlertRule: mockAlertRepo,
	}

	cfg := &config.Config{
		Alert: config.AlertConfig{
			Email: config.EmailNotificationConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
				From:     "test@example.com",
				To:       []string{"admin@example.com"},
			},
			DingTalk: config.DingTalkNotificationConfig{
				WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=test",
			},
		},
	}

	alertService := NewAlertService(mockRepo, cfg)

	// Test data
	rule := &models.AlertRule{
		Name: "Test Alert Rule",
		Condition: map[string]interface{}{
			"anomaly_score_threshold": 0.8,
			"sources":                 []string{"test-service"},
		},
		NotificationChannels: []string{"email", "dingtalk"},
		IsActive:             true,
	}

	// Mock expectations
	mockAlertRepo.On("SaveAlertRule", mock.Anything, rule).Return(nil)

	// Test
	ctx := context.Background()
	err := alertService.CreateAlertRule(ctx, rule)

	// Assertions
	assert.NoError(t, err)
	mockAlertRepo.AssertExpectations(t)
}

func TestAlertService_ValidateAlertRule(t *testing.T) {
	cfg := &config.Config{}
	mockRepo := &repository.Repository{}
	alertService := NewAlertService(mockRepo, cfg)

	tests := []struct {
		name        string
		rule        *models.AlertRule
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid rule",
			rule: &models.AlertRule{
				Name: "Valid Rule",
				Condition: map[string]interface{}{
					"anomaly_score_threshold": 0.8,
				},
				NotificationChannels: []string{"email"},
				IsActive:             true,
			},
			expectError: false,
		},
		{
			name:        "Nil rule",
			rule:        nil,
			expectError: true,
			errorMsg:    "alert rule cannot be nil",
		},
		{
			name: "Empty name",
			rule: &models.AlertRule{
				Name: "",
				Condition: map[string]interface{}{
					"anomaly_score_threshold": 0.8,
				},
				NotificationChannels: []string{"email"},
			},
			expectError: true,
			errorMsg:    "alert rule name cannot be empty",
		},
		{
			name: "Empty condition",
			rule: &models.AlertRule{
				Name:                 "Test Rule",
				Condition:            map[string]interface{}{},
				NotificationChannels: []string{"email"},
			},
			expectError: true,
			errorMsg:    "alert rule condition cannot be empty",
		},
		{
			name: "No notification channels",
			rule: &models.AlertRule{
				Name: "Test Rule",
				Condition: map[string]interface{}{
					"anomaly_score_threshold": 0.8,
				},
				NotificationChannels: []string{},
			},
			expectError: true,
			errorMsg:    "alert rule must have at least one notification channel",
		},
		{
			name: "Invalid notification channel",
			rule: &models.AlertRule{
				Name: "Test Rule",
				Condition: map[string]interface{}{
					"anomaly_score_threshold": 0.8,
				},
				NotificationChannels: []string{"invalid_channel"},
			},
			expectError: true,
			errorMsg:    "invalid notification channel: invalid_channel",
		},
		{
			name: "Missing anomaly score threshold",
			rule: &models.AlertRule{
				Name: "Test Rule",
				Condition: map[string]interface{}{
					"sources": []string{"test-service"},
				},
				NotificationChannels: []string{"email"},
			},
			expectError: true,
			errorMsg:    "anomaly_score_threshold is required",
		},
		{
			name: "Invalid anomaly score threshold - too high",
			rule: &models.AlertRule{
				Name: "Test Rule",
				Condition: map[string]interface{}{
					"anomaly_score_threshold": 1.5,
				},
				NotificationChannels: []string{"email"},
			},
			expectError: true,
			errorMsg:    "anomaly_score_threshold must be between 0 and 1",
		},
		{
			name: "Invalid anomaly score threshold - negative",
			rule: &models.AlertRule{
				Name: "Test Rule",
				Condition: map[string]interface{}{
					"anomaly_score_threshold": -0.1,
				},
				NotificationChannels: []string{"email"},
			},
			expectError: true,
			errorMsg:    "anomaly_score_threshold must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := alertService.validateAlertRule(tt.rule)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertService_GetActiveAlertRules(t *testing.T) {
	// Setup
	mockAlertRepo := &MockAlertRuleRepository{}
	mockRepo := &repository.Repository{
		AlertRule: mockAlertRepo,
	}

	cfg := &config.Config{}
	alertService := NewAlertService(mockRepo, cfg)

	// Test data
	expectedRules := []*models.AlertRule{
		{
			ID:   "rule-1",
			Name: "Rule 1",
			Condition: map[string]interface{}{
				"anomaly_score_threshold": 0.8,
			},
			NotificationChannels: []string{"email"},
			IsActive:             true,
		},
		{
			ID:   "rule-2",
			Name: "Rule 2",
			Condition: map[string]interface{}{
				"anomaly_score_threshold": 0.9,
			},
			NotificationChannels: []string{"dingtalk"},
			IsActive:             true,
		},
	}

	// Mock expectations
	mockAlertRepo.On("GetActiveAlertRules", mock.Anything).Return(expectedRules, nil)

	// Test
	ctx := context.Background()
	rules, err := alertService.GetActiveAlertRules(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, rules, 2)
	assert.Equal(t, expectedRules, rules)
	mockAlertRepo.AssertExpectations(t)
}
