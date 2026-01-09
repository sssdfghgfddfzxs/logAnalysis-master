package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"intelligent-log-analysis/internal/alert"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"

	"github.com/gorilla/mux"
)

// AlertHandler handles alert-related HTTP requests
type AlertHandler struct {
	repo   *repository.Repository
	engine *alert.AlertEngine
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(repo *repository.Repository, engine *alert.AlertEngine) *AlertHandler {
	return &AlertHandler{
		repo:   repo,
		engine: engine,
	}
}

// CreateAlertRuleRequest represents the request to create an alert rule
type CreateAlertRuleRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Condition            map[string]interface{} `json:"condition"`
	NotificationChannels []string               `json:"notification_channels"`
	IsActive             bool                   `json:"is_active"`
}

// UpdateAlertRuleRequest represents the request to update an alert rule
type UpdateAlertRuleRequest struct {
	Name                 *string                `json:"name,omitempty"`
	Description          *string                `json:"description,omitempty"`
	Condition            map[string]interface{} `json:"condition,omitempty"`
	NotificationChannels []string               `json:"notification_channels,omitempty"`
	IsActive             *bool                  `json:"is_active,omitempty"`
}

// CreateAlertRule creates a new alert rule
func (h *AlertHandler) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var req CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate notification channels
	if err := h.validateNotificationChannels(req.NotificationChannels); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create alert rule
	rule := &models.AlertRule{
		Name:                 req.Name,
		Description:          req.Description,
		Condition:            req.Condition,
		NotificationChannels: req.NotificationChannels,
		IsActive:             req.IsActive,
	}

	err := h.repo.AlertRule.SaveAlertRule(r.Context(), rule)
	if err != nil {
		http.Error(w, "Failed to create alert rule", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// GetAlertRules retrieves all alert rules
func (h *AlertHandler) GetAlertRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.repo.AlertRule.GetActiveAlertRules(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve alert rules", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// GetAlertRule retrieves a specific alert rule by ID
func (h *AlertHandler) GetAlertRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rule, err := h.repo.AlertRule.GetAlertRuleByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Alert rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// UpdateAlertRule updates an existing alert rule
func (h *AlertHandler) UpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate notification channels if provided
	if req.NotificationChannels != nil {
		if err := h.validateNotificationChannels(req.NotificationChannels); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Get existing rule
	existingRule, err := h.repo.AlertRule.GetAlertRuleByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Alert rule not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != nil {
		existingRule.Name = *req.Name
	}
	if req.Description != nil {
		existingRule.Description = *req.Description
	}
	if req.Condition != nil {
		existingRule.Condition = req.Condition
	}
	if req.NotificationChannels != nil {
		existingRule.NotificationChannels = req.NotificationChannels
	}
	if req.IsActive != nil {
		existingRule.IsActive = *req.IsActive
	}

	err = h.repo.AlertRule.UpdateAlertRule(r.Context(), existingRule)
	if err != nil {
		http.Error(w, "Failed to update alert rule", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingRule)
}

// DeleteAlertRule deletes an alert rule
func (h *AlertHandler) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := h.repo.AlertRule.DeleteAlertRule(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete alert rule", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TestAlertRule tests an alert rule with sample data
func (h *AlertHandler) TestAlertRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rule, err := h.repo.AlertRule.GetAlertRuleByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Alert rule not found", http.StatusNotFound)
		return
	}

	// Create a test alert
	testAlert := &alert.Alert{
		RuleID:          rule.ID,
		RuleName:        rule.Name,
		LogID:           "test-log-id",
		Source:          "test-source",
		Level:           "ERROR",
		Message:         "This is a test alert message",
		AnomalyScore:    0.95,
		RootCauses:      []string{"Test root cause 1", "Test root cause 2"},
		Recommendations: []string{"Test recommendation 1", "Test recommendation 2"},
		Timestamp:       time.Now(),
	}

	// Send test notifications
	results := make(map[string]string)
	for _, channel := range rule.NotificationChannels {
		if notifier, exists := h.engine.GetNotifier(channel); exists {
			if err := notifier.SendAlert(r.Context(), testAlert); err != nil {
				results[channel] = fmt.Sprintf("Failed: %v", err)
			} else {
				results[channel] = "Success"
			}
		} else {
			results[channel] = "Notifier not found"
		}
	}

	response := map[string]interface{}{
		"message": "Test alert sent",
		"results": results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetNotificationChannels returns available notification channels
func (h *AlertHandler) GetNotificationChannels(w http.ResponseWriter, r *http.Request) {
	channels := []map[string]interface{}{
		{
			"name":         "email",
			"display_name": "邮件通知",
			"description":  "通过邮件发送告警通知",
		},
		{
			"name":         "dingtalk",
			"display_name": "钉钉通知",
			"description":  "通过钉钉机器人发送告警通知",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// validateNotificationChannels validates that all notification channels are supported
func (h *AlertHandler) validateNotificationChannels(channels []string) error {
	supportedChannels := map[string]bool{
		"email":    true,
		"dingtalk": true,
	}

	for _, channel := range channels {
		if !supportedChannels[channel] {
			return fmt.Errorf("unsupported notification channel: %s", channel)
		}
	}

	return nil
}
