package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"intelligent-log-analysis/internal/models"
	pb "intelligent-log-analysis/pkg/proto"
)

// ClientConfig holds configuration for the gRPC client
type ClientConfig struct {
	Address          string
	MaxRetries       int
	RetryDelay       time.Duration
	RequestTimeout   time.Duration
	ConnTimeout      time.Duration
	MaxConnections   int
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
}

// DefaultClientConfig returns a default configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Address:          "localhost:50051",
		MaxRetries:       3,
		RetryDelay:       time.Second,
		RequestTimeout:   30 * time.Second,
		ConnTimeout:      10 * time.Second,
		MaxConnections:   10,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 5 * time.Second,
	}
}

// AIServiceClient wraps the gRPC client with connection pooling and retry logic
type AIServiceClient struct {
	config      *ClientConfig
	connPool    []*grpc.ClientConn
	clients     []pb.AIAnalysisServiceClient
	currentConn int
	mu          sync.RWMutex
}

// NewAIServiceClient creates a new AI service client with connection pooling
func NewAIServiceClient(config *ClientConfig) (*AIServiceClient, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	client := &AIServiceClient{
		config:   config,
		connPool: make([]*grpc.ClientConn, 0, config.MaxConnections),
		clients:  make([]pb.AIAnalysisServiceClient, 0, config.MaxConnections),
	}

	// Create connection pool
	for i := 0; i < config.MaxConnections; i++ {
		conn, err := client.createConnection()
		if err != nil {
			// Close any existing connections before returning error
			client.Close()
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}

		client.connPool = append(client.connPool, conn)
		client.clients = append(client.clients, pb.NewAIAnalysisServiceClient(conn))
	}

	return client, nil
}

// createConnection creates a new gRPC connection with proper configuration
func (c *AIServiceClient) createConnection() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConnTimeout)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.config.KeepAliveTime,
			Timeout:             c.config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	conn, err := grpc.DialContext(ctx, c.config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", c.config.Address, err)
	}

	return conn, nil
}

// getClient returns the next available client using round-robin
func (c *AIServiceClient) getClient() pb.AIAnalysisServiceClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	client := c.clients[c.currentConn]
	c.currentConn = (c.currentConn + 1) % len(c.clients)
	return client
}

// AnalyzeLogs sends logs to the AI service for analysis with retry logic (now uses pure LLM)
func (c *AIServiceClient) AnalyzeLogs(ctx context.Context, logs []*models.LogEntry) ([]*models.AnalysisResult, error) {
	if len(logs) == 0 {
		return nil, fmt.Errorf("no logs provided for analysis")
	}

	// Convert domain models to protobuf messages
	pbLogs := make([]*pb.LogEntry, len(logs))
	for i, log := range logs {
		pbLogs[i] = c.convertLogToProto(log)
	}

	request := &pb.LogAnalysisRequest{
		Logs: pbLogs,
	}

	var response *pb.LogAnalysisResponse
	var err error

	// 使用更长的超时时间，因为LLM分析需要更多时间
	longTimeout := 120 * time.Second
	if c.config.RequestTimeout > longTimeout {
		longTimeout = c.config.RequestTimeout
	}

	// Retry logic with exponential backoff (减少重试次数，因为LLM调用很慢)
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 更长的重试延迟
			delay := time.Duration(attempt*3) * c.config.RetryDelay
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		response, err = c.analyzeLogsWithTimeout(ctx, request, longTimeout)
		if err == nil {
			break
		}

		// Check if error is retryable
		if !c.isRetryableError(err) {
			break
		}

		if attempt == maxRetries {
			return nil, fmt.Errorf("LLM analysis failed after %d attempts: %w", maxRetries+1, err)
		}
	}

	if err != nil {
		return nil, err
	}

	// Check response status
	if response.Status != "success" && response.Status != "" {
		return nil, fmt.Errorf("AI service returned error: %s - %s", response.Status, response.ErrorMessage)
	}

	// Convert protobuf response to domain models
	results := make([]*models.AnalysisResult, len(response.Results))
	for i, result := range response.Results {
		results[i] = c.convertProtoToAnalysisResult(result)
	}

	return results, nil
}

// analyzeLogsWithTimeout performs the actual gRPC call with timeout
func (c *AIServiceClient) analyzeLogsWithTimeout(ctx context.Context, request *pb.LogAnalysisRequest, timeout time.Duration) (*pb.LogAnalysisResponse, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get a client from the pool
	client := c.getClient()

	// Make the gRPC call (now uses pure LLM)
	response, err := client.AnalyzeLogs(timeoutCtx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM gRPC call failed: %w", err)
	}

	return response, nil
}

// isRetryableError determines if an error should trigger a retry
func (c *AIServiceClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for gRPC status codes that are retryable
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
			return true
		case codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated:
			return false
		default:
			return true // Retry on unknown errors
		}
	}

	// Retry on context timeout errors
	if err == context.DeadlineExceeded {
		return true
	}

	return true // Default to retry for other errors
}

// convertLogToProto converts a domain log entry to protobuf format
func (c *AIServiceClient) convertLogToProto(log *models.LogEntry) *pb.LogEntry {
	return &pb.LogEntry{
		Id:        log.ID,
		Timestamp: log.Timestamp.Format(time.RFC3339Nano),
		Level:     log.Level,
		Message:   log.Message,
		Source:    log.Source,
		Metadata:  log.Metadata,
	}
}

// convertProtoToAnalysisResult converts protobuf analysis result to domain model
func (c *AIServiceClient) convertProtoToAnalysisResult(result *pb.AnalysisResult) *models.AnalysisResult {
	// Convert []string to JSONMap for root causes
	var rootCauses models.JSONMap
	if len(result.RootCauses) > 0 {
		rootCauses = models.JSONMap{
			"causes": result.RootCauses,
		}
	}

	// Convert []string to JSONMap for recommendations
	var recommendations models.JSONMap
	if len(result.Recommendations) > 0 {
		recommendations = models.JSONMap{
			"recommendations": result.Recommendations,
		}
	}

	return &models.AnalysisResult{
		LogID:           result.LogId,
		IsAnomaly:       result.IsAnomaly,
		AnomalyScore:    result.AnomalyScore,
		RootCauses:      rootCauses,
		Recommendations: recommendations,
		AnalyzedAt:      time.Now(), // Set current time as analyzed time
	}
}

// HealthCheck checks if the AI service is healthy
func (c *AIServiceClient) HealthCheck(ctx context.Context) error {
	// Create a simple test request
	testLog := &models.LogEntry{
		ID:        "health-check",
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Health check",
		Source:    "health-checker",
		Metadata:  map[string]string{"type": "health-check"},
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.AnalyzeLogs(timeoutCtx, []*models.LogEntry{testLog})
	return err
}

// Close closes all connections in the pool
func (c *AIServiceClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error
	for i, conn := range c.connPool {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection %d: %w", i, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// GetConnectionStats returns statistics about the connection pool
func (c *AIServiceClient) GetConnectionStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections":  len(c.connPool),
		"current_connection": c.currentConn,
		"config": map[string]interface{}{
			"address":         c.config.Address,
			"max_retries":     c.config.MaxRetries,
			"retry_delay":     c.config.RetryDelay.String(),
			"request_timeout": c.config.RequestTimeout.String(),
			"max_connections": c.config.MaxConnections,
		},
	}

	return stats
}
