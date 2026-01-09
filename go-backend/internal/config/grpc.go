package config

import (
	"time"

	"intelligent-log-analysis/internal/grpc"
)

// GRPCConfig holds gRPC client configuration
type GRPCConfig struct {
	AIService AIServiceConfig `mapstructure:"ai_service"`
}

// AIServiceConfig holds AI service specific configuration
type AIServiceConfig struct {
	Address          string        `mapstructure:"address"`
	MaxRetries       int           `mapstructure:"max_retries"`
	RetryDelay       time.Duration `mapstructure:"retry_delay"`
	RequestTimeout   time.Duration `mapstructure:"request_timeout"`
	ConnTimeout      time.Duration `mapstructure:"conn_timeout"`
	MaxConnections   int           `mapstructure:"max_connections"`
	KeepAliveTime    time.Duration `mapstructure:"keep_alive_time"`
	KeepAliveTimeout time.Duration `mapstructure:"keep_alive_timeout"`
	Enabled          bool          `mapstructure:"enabled"`
}

// DefaultGRPCConfig returns default gRPC configuration
func DefaultGRPCConfig() *GRPCConfig {
	return &GRPCConfig{
		AIService: AIServiceConfig{
			Address:          "localhost:50051",
			MaxRetries:       3,
			RetryDelay:       time.Second,
			RequestTimeout:   30 * time.Second,
			ConnTimeout:      10 * time.Second,
			MaxConnections:   5,
			KeepAliveTime:    30 * time.Second,
			KeepAliveTimeout: 5 * time.Second,
			Enabled:          true,
		},
	}
}

// ToClientConfig converts AIServiceConfig to grpc.ClientConfig
func (c *AIServiceConfig) ToClientConfig() *grpc.ClientConfig {
	return &grpc.ClientConfig{
		Address:          c.Address,
		MaxRetries:       c.MaxRetries,
		RetryDelay:       c.RetryDelay,
		RequestTimeout:   c.RequestTimeout,
		ConnTimeout:      c.ConnTimeout,
		MaxConnections:   c.MaxConnections,
		KeepAliveTime:    c.KeepAliveTime,
		KeepAliveTimeout: c.KeepAliveTimeout,
	}
}
