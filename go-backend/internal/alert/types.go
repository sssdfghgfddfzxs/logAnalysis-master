package alert

import (
	"context"
	"time"
)

// Alert represents an alert notification
type Alert struct {
	RuleID          string    `json:"rule_id"`
	RuleName        string    `json:"rule_name"`
	LogID           string    `json:"log_id"`
	Source          string    `json:"source"`
	Level           string    `json:"level"`
	Message         string    `json:"message"`
	AnomalyScore    float64   `json:"anomaly_score"`
	RootCauses      []string  `json:"root_causes"`
	Recommendations []string  `json:"recommendations"`
	Timestamp       time.Time `json:"timestamp"`
}

// Notifier interface for sending alert notifications
type Notifier interface {
	SendAlert(ctx context.Context, alert *Alert) error
}

// NotificationConfig holds configuration for notification services
type NotificationConfig struct {
	Email    EmailConfig    `json:"email"`
	DingTalk DingTalkConfig `json:"dingtalk"`
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	SMTPHost    string   `json:"smtp_host"`
	SMTPPort    int      `json:"smtp_port"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	UseTLS      bool     `json:"use_tls"`
	UseStartTLS bool     `json:"use_starttls"`
}

// DingTalkConfig holds DingTalk notification configuration
type DingTalkConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"`
}
