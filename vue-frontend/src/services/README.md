# Real-time Services Documentation

## WebSocket Service

The `RealtimeService` class provides WebSocket connectivity for real-time data updates in the intelligent log analysis system.

### Features

- **Automatic Connection Management**: Handles WebSocket connection lifecycle
- **Reconnection Logic**: Automatic reconnection with exponential backoff
- **Heartbeat Mechanism**: Keeps connection alive with periodic ping messages
- **Connection Status Tracking**: Provides connection status and listeners
- **Error Handling**: Graceful error handling and user notifications
- **Message Processing**: Handles different types of real-time messages

### Usage

```typescript
import { realtimeService } from '@/services/websocket'

// Connect to WebSocket
await realtimeService.connect()

// Listen for connection changes
realtimeService.onConnectionChange((connected) => {
  console.log('Connection status:', connected)
})

// Check connection status
const isConnected = realtimeService.isConnected()

// Get detailed status
const status = realtimeService.getConnectionStatus()

// Disconnect
realtimeService.disconnect()
```

### Message Types

The service handles the following real-time message types:

#### 1. New Anomaly (`new_anomaly`)
Triggered when a new anomaly is detected in the logs.

```json
{
  "type": "new_anomaly",
  "data": {
    "log": {
      "id": "uuid",
      "timestamp": "2024-01-01T10:00:00Z",
      "level": "ERROR",
      "message": "Database connection failed",
      "source": "user-service",
      "isAnomaly": true,
      "anomalyScore": 0.85,
      "rootCauses": ["Database timeout"],
      "recommendations": ["Check database connectivity"]
    }
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

#### 2. Stats Update (`stats_update`)
Provides updated dashboard statistics.

```json
{
  "type": "stats_update",
  "data": {
    "stats": {
      "totalLogs": 10500,
      "anomalyCount": 155,
      "anomalyRate": 0.0147,
      "activeServices": 5
    }
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

#### 3. System Alert (`system_alert`)
System-level alerts and notifications.

```json
{
  "type": "system_alert",
  "data": {
    "message": "AI analysis service is experiencing high load",
    "severity": "warning"
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

### Integration with Dashboard Store

The WebSocket service integrates seamlessly with the Pinia dashboard store:

```typescript
// In the WebSocket message handler
private handleRealtimeUpdate(message: RealtimeMessage) {
  const dashboardStore = useDashboardStore()

  switch (message.type) {
    case 'new_anomaly':
      dashboardStore.addRecentLog(message.data.log)
      dashboardStore.updateStats({ 
        anomalyCount: dashboardStore.stats.anomalyCount + 1 
      })
      break

    case 'stats_update':
      dashboardStore.updateStats(message.data.stats)
      break
  }
}
```

### Error Handling

The service provides comprehensive error handling:

- **Connection Errors**: Automatic retry with exponential backoff
- **Message Parsing Errors**: Graceful handling of malformed messages
- **Network Issues**: Reconnection attempts with user notifications
- **Service Unavailable**: Fallback to polling or manual refresh

### Configuration

Default configuration can be customized:

```typescript
const service = new RealtimeService('ws://localhost:8080/ws')

// Configuration options (internal)
- maxReconnectAttempts: 5
- reconnectDelay: 1000ms (with exponential backoff)
- heartbeatInterval: 30000ms
```

### Testing

The service includes comprehensive tests covering:

- Connection establishment and teardown
- Message handling and parsing
- Error scenarios and recovery
- Connection status management
- Listener management

Run tests with:
```bash
npm test
```

### Browser Compatibility

The WebSocket service is compatible with all modern browsers that support:
- WebSocket API
- ES6+ features (Promises, Classes)
- JSON parsing

### Performance Considerations

- **Memory Management**: Recent logs are limited to 100 items to prevent memory leaks
- **Connection Pooling**: Single WebSocket connection shared across components
- **Efficient Updates**: Only updates affected UI components
- **Heartbeat Optimization**: Minimal overhead with 30-second intervals