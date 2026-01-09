package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	GRPC     GRPCConfig
	Alert    AlertConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AlertConfig struct {
	Email    EmailNotificationConfig    `json:"email"`
	DingTalk DingTalkNotificationConfig `json:"dingtalk"`
}

type EmailNotificationConfig struct {
	SMTPHost    string   `json:"smtp_host"`
	SMTPPort    int      `json:"smtp_port"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	UseTLS      bool     `json:"use_tls"`
	UseStartTLS bool     `json:"use_starttls"`
}

type DingTalkNotificationConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret,omitempty"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "log_analysis"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		GRPC: GRPCConfig{
			AIService: AIServiceConfig{
				Address:          getEnv("AI_GRPC_ADDRESS", "localhost:50051"),
				MaxRetries:       getEnvAsInt("AI_MAX_RETRIES", 3),
				RetryDelay:       getEnvAsDuration("AI_RETRY_DELAY", time.Second),
				RequestTimeout:   getEnvAsDuration("AI_REQUEST_TIMEOUT", 30*time.Second),
				ConnTimeout:      getEnvAsDuration("AI_CONN_TIMEOUT", 10*time.Second),
				MaxConnections:   getEnvAsInt("AI_MAX_CONNECTIONS", 5),
				KeepAliveTime:    getEnvAsDuration("AI_KEEPALIVE_TIME", 30*time.Second),
				KeepAliveTimeout: getEnvAsDuration("AI_KEEPALIVE_TIMEOUT", 5*time.Second),
				Enabled:          getEnvAsBool("AI_ENABLED", true),
			},
		},
		Alert: AlertConfig{
			Email: EmailNotificationConfig{
				SMTPHost:    getEnv("ALERT_EMAIL_SMTP_HOST", ""),
				SMTPPort:    getEnvAsInt("ALERT_EMAIL_SMTP_PORT", 587),
				Username:    getEnv("ALERT_EMAIL_USERNAME", ""),
				Password:    getEnv("ALERT_EMAIL_PASSWORD", ""),
				From:        getEnv("ALERT_EMAIL_FROM", ""),
				To:          getEnvAsStringSlice("ALERT_EMAIL_TO", []string{}),
				UseTLS:      getEnvAsBool("ALERT_EMAIL_USE_TLS", false),
				UseStartTLS: getEnvAsBool("ALERT_EMAIL_USE_STARTTLS", true),
			},
			DingTalk: DingTalkNotificationConfig{
				WebhookURL: getEnv("ALERT_DINGTALK_WEBHOOK_URL", ""),
				Secret:     getEnv("ALERT_DINGTALK_SECRET", ""),
			},
		},
	}

	return cfg, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}
