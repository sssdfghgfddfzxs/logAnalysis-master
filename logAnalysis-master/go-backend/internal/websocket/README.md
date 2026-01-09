# WebSocket Real-time Push Service

This package provides WebSocket functionality for real-time notifications in the intelligent log analysis system.

## Features

- **Real-time Anomaly Notifications**: Instantly notify connected clients when anomalies are detected
- **Live Log Updates**: Stream new log entries to connected clients
- **Statistics Updates**: Push updated dashboard statistics in real-time
- **System Alerts**: Broadcast system-wide alerts and notifications
- **Connection Management**: Automatic client registration/unregistration with heartbeat support
- **Graceful Shutdown**: Clean shutdown with proper client disconnection

## Architecture

### Components

1. **Hub** (`hub.go`): Central message broadcaster that manages client connections
2. **Client** (`client.go`): Individual WebSocket connection handler with read/write pumps
3. **Service** (`service.go`): High-level service interface for broadcasting different message types

### Message Flow

```
Log Entry → LogService → WebSocket Service → Hub → All Connected Clients
```

## Usage

### Starting the WebSocket Service

```go
import "intelligent-log-analysis/internal/websocket"

// Create and start WebSocket service
wsService := websocket.NewService()
err := wsService.Start(context.Background())
if err != nil {
    log.Fatal("Failed to start WebSocket service:", err)
}
defer wsService.Stop()
```

### Adding WebSocket Endpoint to HTTP Server

```go
// Add WebSocket endpoint to Gin router
router.GET("/ws", func(c *gin.Context) {
    websocket.ServeWS(wsService.GetHub(), c.Writer, c.Request)
})
```

### Broadcasting Messages

```go
// Broadcast new anomaly
wsService.BroadcastNewAnomaly(logEntry, analysisResult)

// Broadcast log update
wsService.BroadcastLogUpdate(logEntry)

// Broadcast stats update
wsService.BroadcastStatsUpdate(stats)

// Broadcast system alert
wsService.BroadcastSystemAlert("database_error", "Database connection failed", "critical")
```

## Message Types

### 1. Connection Established
Sent when a client successfully connects.

```json
{
  "type": "connection_established",
  "data": {
    "status": "connected"
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

### 2. New Anomaly
Sent when an anomaly is detected in log analysis.

```json
{
  "type": "new_anomaly",
  "data": {
    "log": {
      "id": "log-123",
      "timestamp": "2024-01-01T10:00:00Z",
      "level": "ERROR",
      "message": "Database connection failed",
      "source": "user-service",
      "isAnomaly": true,
      "anomalyScore": 0.85,
      "rootCauses": ["Database timeout", "Network issue"],
      "recommendations": ["Check database", "Verify network"],
      "metadata": {"host": "server-01"}
    },
    "analysis": {
      "id": "analysis-456",
      "anomalyScore": 0.85,
      "rootCauses": ["Database timeout"],
      "recommendations": ["Check database connectivity"],
      "analyzedAt": "2024-01-01T10:00:05Z"
    }
  },
  "timestamp": "2024-01-01T10:00:05Z"
}
```

### 3. Log Update
Sent when a new log entry is created.

```json
{
  "type": "log_update",
  "data": {
    "log": {
      "id": "log-789",
      "timestamp": "2024-01-01T10:01:00Z",
      "level": "INFO",
      "message": "User login successful",
      "source": "auth-service",
      "metadata": {"userId": "user-123"}
    }
  },
  "timestamp": "2024-01-01T10:01:00Z"
}
```

### 4. Stats Update
Sent when dashboard statistics are updated.

```json
{
  "type": "stats_update",
  "data": {
    "stats": {
      "totalLogs": 10000,
      "anomalyCount": 150,
      "anomalyRate": 0.015,
      "activeServices": 5
    }
  },
  "timestamp": "2024-01-01T10:02:00Z"
}
```

### 5. System Alert
Sent for system-wide alerts and notifications.

```json
{
  "type": "system_alert",
  "data": {
    "alertType": "database_error",
    "message": "Database connection failed",
    "severity": "critical",
    "timestamp": "2024-01-01T10:03:00Z"
  },
  "timestamp": "2024-01-01T10:03:00Z"
}
```

### 6. Heartbeat Messages
Clients can send ping messages to keep the connection alive.

**Client → Server (Ping):**
```json
{
  "type": "ping"
}
```

**Server → Client (Pong):**
```json
{
  "type": "pong",
  "data": {
    "timestamp": "2024-01-01T10:04:00Z"
  },
  "timestamp": "2024-01-01T10:04:00Z"
}
```

## Client Connection Management

### Connection Lifecycle

1. **Connection**: Client connects to `/ws` endpoint
2. **Registration**: Client is registered with the hub
3. **Welcome Message**: Server sends connection_established message
4. **Message Exchange**: Bidirectional message communication
5. **Heartbeat**: Periodic ping/pong to keep connection alive
6. **Disconnection**: Client disconnects or connection is lost
7. **Cleanup**: Client is unregistered from the hub

### Heartbeat Mechanism

- Server sends ping messages every 30 seconds
- Client should respond with pong messages
- Connection is closed if no pong is received within 60 seconds
- Clients can also send ping messages to test connectivity

## Configuration

### WebSocket Upgrader Settings

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Allow connections from any origin in development
        // In production, implement proper origin checking
        return true
    },
}
```

### Timeouts

- **Write Timeout**: 10 seconds
- **Pong Timeout**: 60 seconds  
- **Ping Period**: 54 seconds (90% of pong timeout)
- **Max Message Size**: 512 bytes for client messages

## Integration with Log Service

The WebSocket service integrates with the LogService through the `WebSocketService` interface:

```go
type WebSocketService interface {
    BroadcastNewAnomaly(log *models.LogEntry, analysisResult *models.AnalysisResult)
    BroadcastLogUpdate(log *models.LogEntry)
    BroadcastStatsUpdate(stats interface{})
}
```

### Setting WebSocket Service in LogService

```go
// In server initialization
wsService := websocket.NewService()
logService.SetWebSocketService(wsService)
```

## Testing

### Running Tests

```bash
go test ./internal/websocket -v
```

### Test Coverage

- Hub functionality and client management
- Service lifecycle (start/stop)
- Message broadcasting with no clients
- WebSocket connection establishment
- Heartbeat mechanism
- Message parsing and error handling

### Example Server

Run the example server to test WebSocket functionality:

```bash
go run examples/websocket_example.go
```

Then open http://localhost:8080 in your browser to interact with the WebSocket service.

## Frontend Integration

The frontend WebSocket client (`vue-frontend/src/services/websocket.ts`) automatically:

- Connects to the WebSocket endpoint
- Handles reconnection with exponential backoff
- Processes different message types
- Updates the dashboard store with real-time data
- Shows notifications for anomalies and alerts

### Frontend Usage

```typescript
import { realtimeService } from '@/services/websocket'

// Connect to WebSocket
await realtimeService.connect()

// Listen for connection status changes
realtimeService.onConnectionChange((connected) => {
  console.log('WebSocket connection status:', connected)
})

// Disconnect when done
realtimeService.disconnect()
```

## Security Considerations

### Production Deployment

1. **Origin Checking**: Implement proper CORS origin validation
2. **Authentication**: Add WebSocket authentication/authorization
3. **Rate Limiting**: Implement connection and message rate limits
4. **SSL/TLS**: Use secure WebSocket connections (wss://)
5. **Message Validation**: Validate all incoming client messages

### Example Production Configuration

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        return isAllowedOrigin(origin)
    },
}

func isAllowedOrigin(origin string) bool {
    allowedOrigins := []string{
        "https://yourdomain.com",
        "https://app.yourdomain.com",
    }
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

## Performance Considerations

- **Connection Limits**: Monitor and limit concurrent connections
- **Message Queuing**: Hub uses buffered channels to prevent blocking
- **Memory Usage**: Automatic cleanup of disconnected clients
- **CPU Usage**: Efficient message broadcasting to multiple clients
- **Network Usage**: JSON message compression for large payloads

## Monitoring

### Connection Statistics

```go
stats := wsService.GetConnectionStats()
// Returns:
// {
//   "connectedClients": 42,
//   "hubStatus": "running",
//   "uptime": "2024-01-01T10:00:00Z"
// }
```

### Health Checks

The WebSocket service status is included in the main health check endpoint:

```bash
curl http://localhost:8080/health
```

### Metrics to Monitor

- Number of connected clients
- Message broadcast rate
- Connection/disconnection rate
- Failed message deliveries
- Average connection duration