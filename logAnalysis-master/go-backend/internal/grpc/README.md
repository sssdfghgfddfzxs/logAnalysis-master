# gRPC AI Service Client

This package provides a robust gRPC client for communicating with the AI analysis service. It includes connection pooling, retry mechanisms, and timeout handling as required by the specification.

## Features

- **Connection Pooling**: Maintains multiple connections to the AI service for better performance
- **Retry Logic**: Automatically retries failed requests with exponential backoff
- **Timeout Handling**: Configurable timeouts for connections and requests
- **Health Checking**: Built-in health check functionality
- **Graceful Shutdown**: Proper cleanup of connections

## Configuration

The client can be configured through environment variables or programmatically:

```go
config := &grpc.ClientConfig{
    Address:         "localhost:50051",
    MaxRetries:      3,
    RetryDelay:      time.Second,
    RequestTimeout:  30 * time.Second,
    ConnTimeout:     10 * time.Second,
    MaxConnections:  5,
    KeepAliveTime:   30 * time.Second,
    KeepAliveTimeout: 5 * time.Second,
}
```

## Usage

### Basic Usage

```go
// Create client
client, err := grpc.NewAIServiceClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Analyze logs
logs := []*models.LogEntry{
    {
        ID:        "log-1",
        Timestamp: time.Now(),
        Level:     "ERROR",
        Message:   "Database connection failed",
        Source:    "user-service",
        Metadata:  map[string]string{"host": "server-01"},
    },
}

results, err := client.AnalyzeLogs(context.Background(), logs)
if err != nil {
    log.Printf("Analysis failed: %v", err)
    return
}

for _, result := range results {
    fmt.Printf("Log %s: Anomaly=%t, Score=%.3f\n", 
        result.LogID, result.IsAnomaly, result.AnomalyScore)
}
```

### Health Checking

```go
if err := client.HealthCheck(context.Background()); err != nil {
    log.Printf("AI service is unhealthy: %v", err)
}
```

### Connection Statistics

```go
stats := client.GetConnectionStats()
fmt.Printf("Total connections: %d\n", stats["total_connections"])
fmt.Printf("Current connection: %d\n", stats["current_connection"])
```

## Integration with Services

The gRPC client is integrated into the LogService for automatic AI analysis:

```go
// Create service with AI client
logService := service.NewLogServiceWithAI(repo, aiClient)

// Logs will be automatically analyzed when created
err := logService.CreateLog(ctx, logEntry)
```

## Error Handling

The client implements intelligent retry logic for different types of errors:

- **Retryable errors**: `Unavailable`, `DeadlineExceeded`, `ResourceExhausted`, `Aborted`
- **Non-retryable errors**: `InvalidArgument`, `NotFound`, `PermissionDenied`, `Unauthenticated`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_GRPC_ADDRESS` | `localhost:50051` | AI service gRPC address |
| `AI_ENABLED` | `true` | Enable/disable AI service |
| `AI_MAX_RETRIES` | `3` | Maximum retry attempts |
| `AI_RETRY_DELAY` | `1s` | Base retry delay |
| `AI_REQUEST_TIMEOUT` | `30s` | Request timeout |
| `AI_CONN_TIMEOUT` | `10s` | Connection timeout |
| `AI_MAX_CONNECTIONS` | `5` | Maximum connections in pool |
| `AI_KEEPALIVE_TIME` | `30s` | Keep-alive time |
| `AI_KEEPALIVE_TIMEOUT` | `5s` | Keep-alive timeout |

## Testing

Run the tests with:

```bash
go test ./internal/grpc/...
```

Note: Some tests may fail if the AI service is not running, which is expected behavior for connection tests.