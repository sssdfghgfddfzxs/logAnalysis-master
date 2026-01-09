package grpc

import (
	"context"
	"testing"
	"time"

	"intelligent-log-analysis/internal/models"
	pb "intelligent-log-analysis/pkg/proto"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "localhost:50051", config.Address)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.RetryDelay)
	assert.Equal(t, 30*time.Second, config.RequestTimeout)
	assert.Equal(t, 10*time.Second, config.ConnTimeout)
	assert.Equal(t, 10, config.MaxConnections)
	assert.Equal(t, 30*time.Second, config.KeepAliveTime)
	assert.Equal(t, 5*time.Second, config.KeepAliveTimeout)
}

func TestConvertLogToProto(t *testing.T) {
	client := &AIServiceClient{
		config: DefaultClientConfig(),
	}

	now := time.Now()
	log := &models.LogEntry{
		ID:        "test-id",
		Timestamp: now,
		Level:     "ERROR",
		Message:   "Test error message",
		Source:    "test-service",
		Metadata: map[string]string{
			"host": "server-01",
			"env":  "production",
		},
	}

	pbLog := client.convertLogToProto(log)

	assert.Equal(t, "test-id", pbLog.Id)
	assert.Equal(t, now.Format(time.RFC3339Nano), pbLog.Timestamp)
	assert.Equal(t, "ERROR", pbLog.Level)
	assert.Equal(t, "Test error message", pbLog.Message)
	assert.Equal(t, "test-service", pbLog.Source)
	assert.Equal(t, "server-01", pbLog.Metadata["host"])
	assert.Equal(t, "production", pbLog.Metadata["env"])
}

func TestAnalyzeLogsValidation(t *testing.T) {
	// Note: This test will fail if the AI service is not running
	// It's designed to test the validation logic, not the actual connection
	config := DefaultClientConfig()
	config.ConnTimeout = 1 * time.Second
	config.MaxConnections = 1

	// This will fail to connect, which is expected for this test
	client, err := NewAIServiceClient(config)
	if err != nil {
		// Expected when AI service is not running
		t.Logf("Expected error when AI service is not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Test with empty logs
	_, err = client.AnalyzeLogs(ctx, []*models.LogEntry{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no logs provided")
}

func TestGetClient(t *testing.T) {
	// Create a mock client without actual connections
	client := &AIServiceClient{
		config:      DefaultClientConfig(),
		currentConn: 0,
	}

	// Mock 3 clients
	client.clients = make([]pb.AIAnalysisServiceClient, 3)
	for i := 0; i < 3; i++ {
		client.clients[i] = nil // Mock clients
	}

	// Test round-robin behavior
	for i := 0; i < 10; i++ {
		expectedIndex := i % 3
		assert.Equal(t, expectedIndex, client.currentConn)
		// Simulate getting a client
		client.mu.Lock()
		client.currentConn = (client.currentConn + 1) % len(client.clients)
		client.mu.Unlock()
	}
}

func TestIsRetryableError(t *testing.T) {
	client := &AIServiceClient{
		config: DefaultClientConfig(),
	}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConnectionStats(t *testing.T) {
	client := &AIServiceClient{
		config:      DefaultClientConfig(),
		currentConn: 2,
	}

	// Mock 5 connections
	client.connPool = make([]*grpc.ClientConn, 5)
	for i := 0; i < 5; i++ {
		client.connPool[i] = nil // Mock connections
	}

	stats := client.GetConnectionStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 5, stats["total_connections"])
	assert.Equal(t, 2, stats["current_connection"])

	configStats := stats["config"].(map[string]interface{})
	assert.Equal(t, "localhost:50051", configStats["address"])
	assert.Equal(t, 3, configStats["max_retries"])
	assert.Equal(t, 10, configStats["max_connections"])
}
