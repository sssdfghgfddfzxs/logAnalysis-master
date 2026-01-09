package alert

import (
	"os"
	"strconv"
	"strings"
)

// NotificationManager manages notification configurations and notifiers
type NotificationManager struct {
	config    NotificationConfig
	notifiers map[string]Notifier
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	config := loadNotificationConfig()

	manager := &NotificationManager{
		config:    config,
		notifiers: make(map[string]Notifier),
	}

	// Initialize notifiers based on configuration
	manager.initializeNotifiers()

	return manager
}

// GetNotifiers returns all configured notifiers
func (nm *NotificationManager) GetNotifiers() map[string]Notifier {
	return nm.notifiers
}

// GetNotifier returns a specific notifier by name
func (nm *NotificationManager) GetNotifier(name string) (Notifier, bool) {
	notifier, exists := nm.notifiers[name]
	return notifier, exists
}

// initializeNotifiers initializes notifiers based on configuration
func (nm *NotificationManager) initializeNotifiers() {
	// Initialize email notifier if configured
	if nm.isEmailConfigured() {
		emailNotifier := NewEmailNotifier(nm.config.Email)
		nm.notifiers["email"] = emailNotifier
	}

	// Initialize DingTalk notifier if configured
	if nm.isDingTalkConfigured() {
		dingTalkNotifier := NewDingTalkNotifier(nm.config.DingTalk)
		nm.notifiers["dingtalk"] = dingTalkNotifier
	}
}

// isEmailConfigured checks if email configuration is complete
func (nm *NotificationManager) isEmailConfigured() bool {
	return nm.config.Email.SMTPHost != "" &&
		nm.config.Email.Username != "" &&
		nm.config.Email.Password != "" &&
		nm.config.Email.From != "" &&
		len(nm.config.Email.To) > 0
}

// isDingTalkConfigured checks if DingTalk configuration is complete
func (nm *NotificationManager) isDingTalkConfigured() bool {
	return nm.config.DingTalk.WebhookURL != ""
}

// loadNotificationConfig loads notification configuration from environment variables
func loadNotificationConfig() NotificationConfig {
	config := NotificationConfig{}

	// Load email configuration
	config.Email = EmailConfig{
		SMTPHost:    getEnv("EMAIL_SMTP_HOST", ""),
		SMTPPort:    getEnvAsInt("EMAIL_SMTP_PORT", 587),
		Username:    getEnv("EMAIL_USERNAME", ""),
		Password:    getEnv("EMAIL_PASSWORD", ""),
		From:        getEnv("EMAIL_FROM", ""),
		To:          getEnvAsSlice("EMAIL_TO", ","),
		UseTLS:      getEnvAsBool("EMAIL_USE_TLS", false),
		UseStartTLS: getEnvAsBool("EMAIL_USE_STARTTLS", true),
	}

	// Load DingTalk configuration
	config.DingTalk = DingTalkConfig{
		WebhookURL: getEnv("DINGTALK_WEBHOOK_URL", ""),
		Secret:     getEnv("DINGTALK_SECRET", ""),
	}

	return config
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key, separator string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, separator)
		var result []string
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return []string{}
}
