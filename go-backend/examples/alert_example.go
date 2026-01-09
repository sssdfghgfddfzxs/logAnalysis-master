//go:build examples
// +build examples

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"intelligent-log-analysis/internal/config"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"
	"intelligent-log-analysis/internal/service"
)

// This example demonstrates how to use the alert system
func main() {
	fmt.Println("Alert System Example")
	fmt.Println("===================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// For this example, we'll use mock repositories
	// In a real application, you'd use actual database repositories
	mockRepo := &repository.Repository{
		AlertRule: &MockAlertRuleRepo{},
	}

	// Create alert service
	alertService := service.NewAlertService(mockRepo, cfg)

	// Start alert service
	ctx := context.Background()
	if err := alertService.Start(ctx); err != nil {
		log.Fatalf("Failed to start alert service: %v", err)
	}
	defer alertService.Stop()

	// Example 1: Create an alert rule
	fmt.Println("\n1. Creating an alert rule...")

	alertRule := &models.AlertRule{
		Name: "High Anomaly Score Alert",
		Condition: map[string]interface{}{
			"anomaly_score_threshold": 0.8,
			"sources":                 []string{"user-service", "payment-service"},
			"levels":                  []string{"ERROR", "FATAL"},
		},
		NotificationChannels: []string{"email", "dingtalk"},
		IsActive:             true,
	}

	if err := alertService.CreateAlertRule(ctx, alertRule); err != nil {
		log.Printf("Failed to create alert rule: %v", err)
	} else {
		fmt.Printf("Created alert rule: %s (ID: %s)\n", alertRule.Name, alertRule.ID)
	}

	// Example 2: Simulate an analysis result that triggers an alert
	fmt.Println("\n2. Simulating analysis result that triggers alert...")

	analysisResult := &models.AnalysisResult{
		ID:           "analysis-1",
		LogID:        "log-1",
		IsAnomaly:    true,
		AnomalyScore: 0.95, // High score that should trigger alert
		RootCauses:   []string{"Database connection timeout", "High memory usage"},
		Recommendations: []string{
			"Check database connectivity",
			"Monitor memory usage",
			"Review connection pool settings",
		},
		AnalyzedAt: time.Now(),
		Log: models.LogEntry{
			ID:        "log-1",
			Timestamp: time.Now(),
			Level:     "ERROR", // Matches rule condition
			Message:   "Database connection failed after 30 seconds timeout",
			Source:    "user-service", // Matches rule condition
			Metadata: map[string]string{
				"host":      "server-01",
				"thread":    "main",
				"component": "database",
			},
		},
	}

	// Get alert engine and evaluate the result
	alertEngine := alertService.GetAlertEngine()
	if err := alertEngine.EvaluateAnalysisResult(ctx, analysisResult); err != nil {
		log.Printf("Failed to evaluate analysis result: %v", err)
	} else {
		fmt.Println("Analysis result evaluated successfully")
	}

	// Example 3: Show alert rule configuration
	fmt.Println("\n3. Alert rule configuration example:")

	exampleConditions := []map[string]interface{}{
		{
			"name": "Critical Error Alert",
			"condition": map[string]interface{}{
				"anomaly_score_threshold": 0.9,
				"min_anomaly_count":       1,
				"time_window_minutes":     5,
				"levels":                  []string{"ERROR", "FATAL"},
			},
			"notification_channels": []string{"email", "dingtalk"},
		},
		{
			"name": "Service Specific Alert",
			"condition": map[string]interface{}{
				"anomaly_score_threshold": 0.7,
				"sources":                 []string{"payment-service"},
				"min_anomaly_count":       3,
				"time_window_minutes":     10,
			},
			"notification_channels": []string{"dingtalk"},
		},
		{
			"name": "General Anomaly Alert",
			"condition": map[string]interface{}{
				"anomaly_score_threshold": 0.8,
				"time_window_minutes":     15,
			},
			"notification_channels": []string{"email"},
		},
	}

	for i, example := range exampleConditions {
		fmt.Printf("\nExample %d: %s\n", i+1, example["name"])
		conditionJSON, _ := json.MarshalIndent(example["condition"], "  ", "  ")
		fmt.Printf("  Condition: %s\n", conditionJSON)
		fmt.Printf("  Channels: %v\n", example["notification_channels"])
	}

	// Example 4: Show notification configuration
	fmt.Println("\n4. Notification configuration:")
	fmt.Println("Email configuration:")
	fmt.Printf("  SMTP Host: %s\n", cfg.Alert.Email.SMTPHost)
	fmt.Printf("  SMTP Port: %d\n", cfg.Alert.Email.SMTPPort)
	fmt.Printf("  From: %s\n", cfg.Alert.Email.From)
	fmt.Printf("  To: %v\n", cfg.Alert.Email.To)
	fmt.Printf("  Use TLS: %t\n", cfg.Alert.Email.UseTLS)
	fmt.Printf("  Use STARTTLS: %t\n", cfg.Alert.Email.UseStartTLS)

	fmt.Println("\nDingTalk configuration:")
	fmt.Printf("  Webhook URL: %s\n", cfg.Alert.DingTalk.WebhookURL)
	if cfg.Alert.DingTalk.Secret != "" {
		fmt.Printf("  Secret: %s\n", "***configured***")
	} else {
		fmt.Printf("  Secret: %s\n", "not configured")
	}

	fmt.Println("\n5. Alert suppression example:")
	fmt.Println("The alert system includes suppression to prevent alert flooding.")
	fmt.Println("When an alert is triggered, identical alerts are suppressed for 5 minutes.")
	fmt.Println("This prevents spam when the same issue generates multiple log entries.")

	fmt.Println("\nAlert system example completed!")
}

// MockAlertRuleRepo is a simple mock for demonstration
type MockAlertRuleRepo struct {
	rules []*models.AlertRule
}

func (m *MockAlertRuleRepo) SaveAlertRule(ctx context.Context, rule *models.AlertRule) error {
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", len(m.rules)+1)
	}
	rule.CreatedAt = time.Now()
	m.rules = append(m.rules, rule)
	return nil
}

func (m *MockAlertRuleRepo) GetAlertRuleByID(ctx context.Context, id string) (*models.AlertRule, error) {
	for _, rule := range m.rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return nil, fmt.Errorf("alert rule not found: %s", id)
}

func (m *MockAlertRuleRepo) GetActiveAlertRules(ctx context.Context) ([]*models.AlertRule, error) {
	var activeRules []*models.AlertRule
	for _, rule := range m.rules {
		if rule.IsActive {
			activeRules = append(activeRules, rule)
		}
	}
	return activeRules, nil
}

func (m *MockAlertRuleRepo) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	for i, existingRule := range m.rules {
		if existingRule.ID == rule.ID {
			m.rules[i] = rule
			return nil
		}
	}
	return fmt.Errorf("alert rule not found: %s", rule.ID)
}

func (m *MockAlertRuleRepo) DeleteAlertRule(ctx context.Context, id string) error {
	for i, rule := range m.rules {
		if rule.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("alert rule not found: %s", id)
}
