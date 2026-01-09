package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"
)

// AlertEngine manages alert rules and triggers notifications
type AlertEngine struct {
	repo         *repository.Repository
	notifiers    map[string]Notifier
	suppression  *SuppressionManager
	mu           sync.RWMutex
	activeRules  []*models.AlertRule
	lastUpdate   time.Time
	updateTicker *time.Ticker
	stopCh       chan struct{}
}

// AlertCondition represents the structure of alert conditions
type AlertCondition struct {
	AnomalyScoreThreshold float64  `json:"anomaly_score_threshold"`
	MinAnomalyCount       int      `json:"min_anomaly_count"`
	TimeWindowMinutes     int      `json:"time_window_minutes"`
	Sources               []string `json:"sources,omitempty"`
	Levels                []string `json:"levels,omitempty"`
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine(repo *repository.Repository) *AlertEngine {
	engine := &AlertEngine{
		repo:        repo,
		notifiers:   make(map[string]Notifier),
		suppression: NewSuppressionManager(),
		stopCh:      make(chan struct{}),
	}

	// Initialize notification manager and register notifiers
	notificationManager := NewNotificationManager()
	for name, notifier := range notificationManager.GetNotifiers() {
		engine.RegisterNotifier(name, notifier)
	}

	// Start rule update ticker (every 5 minutes)
	engine.updateTicker = time.NewTicker(5 * time.Minute)
	go engine.ruleUpdateLoop()

	return engine
}

// RegisterNotifier registers a notification service
func (e *AlertEngine) RegisterNotifier(name string, notifier Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifiers[name] = notifier
}

// GetNotifier returns a notifier by name
func (e *AlertEngine) GetNotifier(name string) (Notifier, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	notifier, exists := e.notifiers[name]
	return notifier, exists
}

// Start starts the alert engine
func (e *AlertEngine) Start(ctx context.Context) error {
	// Load initial rules
	if err := e.loadActiveRules(ctx); err != nil {
		return fmt.Errorf("failed to load active rules: %w", err)
	}

	log.Printf("Alert engine started with %d active rules", len(e.activeRules))
	return nil
}

// Stop stops the alert engine
func (e *AlertEngine) Stop() {
	close(e.stopCh)
	if e.updateTicker != nil {
		e.updateTicker.Stop()
	}
}

// EvaluateAnalysisResult evaluates an analysis result against all active rules
func (e *AlertEngine) EvaluateAnalysisResult(ctx context.Context, result *models.AnalysisResult) error {
	e.mu.RLock()
	rules := make([]*models.AlertRule, len(e.activeRules))
	copy(rules, e.activeRules)
	e.mu.RUnlock()

	for _, rule := range rules {
		if e.shouldTriggerAlert(ctx, rule, result) {
			if err := e.triggerAlert(ctx, rule, result); err != nil {
				log.Printf("Failed to trigger alert for rule %s: %v", rule.Name, err)
			}
		}
	}

	return nil
}

// shouldTriggerAlert checks if an alert should be triggered for a given rule and result
func (e *AlertEngine) shouldTriggerAlert(ctx context.Context, rule *models.AlertRule, result *models.AnalysisResult) bool {
	// Parse condition
	var condition AlertCondition
	conditionBytes, err := json.Marshal(rule.Condition)
	if err != nil {
		log.Printf("Failed to marshal condition for rule %s: %v", rule.Name, err)
		return false
	}

	if err := json.Unmarshal(conditionBytes, &condition); err != nil {
		log.Printf("Failed to parse condition for rule %s: %v", rule.Name, err)
		return false
	}

	// Check if result meets the condition
	if !result.IsAnomaly {
		return false
	}

	// Check anomaly score threshold
	if result.AnomalyScore < condition.AnomalyScoreThreshold {
		return false
	}

	// Check source filter
	if len(condition.Sources) > 0 {
		sourceMatch := false
		for _, source := range condition.Sources {
			if result.Log.Source == source {
				sourceMatch = true
				break
			}
		}
		if !sourceMatch {
			return false
		}
	}

	// Check level filter
	if len(condition.Levels) > 0 {
		levelMatch := false
		for _, level := range condition.Levels {
			if result.Log.Level == level {
				levelMatch = true
				break
			}
		}
		if !levelMatch {
			return false
		}
	}

	// Check suppression
	alertKey := fmt.Sprintf("%s:%s:%s", rule.ID, result.Log.Source, result.Log.Level)
	if e.suppression.IsSuppressed(alertKey) {
		return false
	}

	return true
}

// triggerAlert triggers an alert for a given rule and result
func (e *AlertEngine) triggerAlert(ctx context.Context, rule *models.AlertRule, result *models.AnalysisResult) error {
	// Convert JSONMap to []string for root causes
	var rootCauses []string
	if result.RootCauses != nil {
		if causes, ok := result.RootCauses["causes"]; ok {
			if causesSlice, ok := causes.([]interface{}); ok {
				for _, cause := range causesSlice {
					if causeStr, ok := cause.(string); ok {
						rootCauses = append(rootCauses, causeStr)
					}
				}
			}
		}
	}

	// Convert JSONMap to []string for recommendations
	var recommendations []string
	if result.Recommendations != nil {
		if recs, ok := result.Recommendations["recommendations"]; ok {
			if recsSlice, ok := recs.([]interface{}); ok {
				for _, rec := range recsSlice {
					if recStr, ok := rec.(string); ok {
						recommendations = append(recommendations, recStr)
					}
				}
			}
		}
	}

	alert := &Alert{
		RuleID:          rule.ID,
		RuleName:        rule.Name,
		LogID:           result.LogID,
		Source:          result.Log.Source,
		Level:           result.Log.Level,
		Message:         result.Log.Message,
		AnomalyScore:    result.AnomalyScore,
		RootCauses:      rootCauses,
		Recommendations: recommendations,
		Timestamp:       time.Now(),
	}

	// Send notifications
	for _, channel := range rule.NotificationChannels {
		if notifier, exists := e.notifiers[channel]; exists {
			if err := notifier.SendAlert(ctx, alert); err != nil {
				log.Printf("Failed to send alert via %s: %v", channel, err)
			}
		} else {
			log.Printf("Unknown notification channel: %s", channel)
		}
	}

	// Add to suppression
	alertKey := fmt.Sprintf("%s:%s:%s", rule.ID, result.Log.Source, result.Log.Level)
	e.suppression.AddSuppression(alertKey, 5*time.Minute) // 5 minutes suppression

	log.Printf("Alert triggered for rule %s: %s", rule.Name, alert.Message)
	return nil
}

// loadActiveRules loads active alert rules from the repository
func (e *AlertEngine) loadActiveRules(ctx context.Context) error {
	rules, err := e.repo.AlertRule.GetActiveAlertRules(ctx)
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.activeRules = rules
	e.lastUpdate = time.Now()
	e.mu.Unlock()

	return nil
}

// ruleUpdateLoop periodically updates active rules
func (e *AlertEngine) ruleUpdateLoop() {
	for {
		select {
		case <-e.updateTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := e.loadActiveRules(ctx); err != nil {
				log.Printf("Failed to update active rules: %v", err)
			}
			cancel()
		case <-e.stopCh:
			return
		}
	}
}
