# Alert System

The alert system provides intelligent alerting capabilities for the log analysis service. It monitors analysis results and triggers notifications when anomalies meet configured criteria.

## Features

- **Rule-based alerting**: Configure custom alert rules with flexible conditions
- **Multiple notification channels**: Support for email and DingTalk notifications
- **Alert suppression**: Prevents alert flooding with configurable suppression periods
- **Real-time evaluation**: Evaluates analysis results in real-time as they are generated
- **Flexible conditions**: Support for anomaly score thresholds, source filters, level filters, and more

## Components

### AlertEngine
The core component that manages alert rules and evaluates analysis results.

```go
engine := alert.NewAlertEngine(repository)
engine.RegisterNotifier("email", emailNotifier)
engine.RegisterNotifier("dingtalk", dingTalkNotifier)
engine.Start(ctx)
```

### Notifiers
Implementations for different notification channels:

- **EmailNotifier**: Sends HTML email notifications via SMTP
- **DingTalkNotifier**: Sends markdown messages to DingTalk webhooks

### SuppressionManager
Manages alert suppression to prevent flooding:

```go
suppression := alert.NewSuppressionManager()
suppression.AddSuppression("alert-key", 5*time.Minute)
```

## Configuration

### Environment Variables

```bash
# Email Configuration
ALERT_EMAIL_SMTP_HOST=smtp.gmail.com
ALERT_EMAIL_SMTP_PORT=587
ALERT_EMAIL_USERNAME=your-email@gmail.com
ALERT_EMAIL_PASSWORD=your-app-password
ALERT_EMAIL_FROM=your-email@gmail.com
ALERT_EMAIL_TO=admin@company.com,ops@company.com
ALERT_EMAIL_USE_TLS=false
ALERT_EMAIL_USE_STARTTLS=true

# DingTalk Configuration
ALERT_DINGTALK_WEBHOOK_URL=https://oapi.dingtalk.com/robot/send?access_token=your-token
ALERT_DINGTALK_SECRET=your-secret
```

### Alert Rule Structure

```json
{
  "name": "High Anomaly Score Alert",
  "condition": {
    "anomaly_score_threshold": 0.8,
    "min_anomaly_count": 1,
    "time_window_minutes": 5,
    "sources": ["user-service", "payment-service"],
    "levels": ["ERROR", "FATAL"]
  },
  "notification_channels": ["email", "dingtalk"],
  "is_active": true
}
```

#### Condition Fields

- `anomaly_score_threshold` (required): Minimum anomaly score to trigger alert (0.0-1.0)
- `min_anomaly_count` (optional): Minimum number of anomalies in time window
- `time_window_minutes` (optional): Time window for counting anomalies
- `sources` (optional): Filter by log sources (service names)
- `levels` (optional): Filter by log levels (ERROR, WARN, INFO, DEBUG)

## API Endpoints

### Create Alert Rule
```http
POST /api/v1/alert-rules
Content-Type: application/json

{
  "name": "Critical Error Alert",
  "condition": {
    "anomaly_score_threshold": 0.9,
    "levels": ["ERROR", "FATAL"]
  },
  "notification_channels": ["email", "dingtalk"]
}
```

### Get Alert Rules
```http
GET /api/v1/alert-rules?active=true
```

### Get Alert Rule by ID
```http
GET /api/v1/alert-rules/{id}
```

### Update Alert Rule
```http
PUT /api/v1/alert-rules/{id}
Content-Type: application/json

{
  "name": "Updated Alert Rule",
  "condition": {
    "anomaly_score_threshold": 0.85
  },
  "notification_channels": ["email"],
  "is_active": false
}
```

### Delete Alert Rule
```http
DELETE /api/v1/alert-rules/{id}
```

## Usage Examples

### Basic Alert Rule
```go
rule := &models.AlertRule{
    Name: "High Anomaly Alert",
    Condition: map[string]interface{}{
        "anomaly_score_threshold": 0.8,
    },
    NotificationChannels: []string{"email"},
    IsActive: true,
}

err := alertService.CreateAlertRule(ctx, rule)
```

### Service-Specific Alert
```go
rule := &models.AlertRule{
    Name: "Payment Service Alert",
    Condition: map[string]interface{}{
        "anomaly_score_threshold": 0.7,
        "sources": []string{"payment-service"},
        "levels": []string{"ERROR", "FATAL"},
        "min_anomaly_count": 3,
        "time_window_minutes": 10,
    },
    NotificationChannels: []string{"dingtalk"},
    IsActive: true,
}
```

### Custom Notifier
```go
type CustomNotifier struct{}

func (n *CustomNotifier) SendAlert(ctx context.Context, alert *Alert) error {
    // Custom notification logic
    return nil
}

engine.RegisterNotifier("custom", &CustomNotifier{})
```

## Alert Suppression

The system automatically suppresses duplicate alerts to prevent flooding:

- **Suppression Key**: Generated from rule ID, source, and level
- **Default Duration**: 5 minutes
- **Automatic Cleanup**: Expired suppressions are automatically removed

## Testing

Run the alert system tests:

```bash
go test ./internal/alert/... -v
```

Run the alert service tests:

```bash
go test ./internal/service/... -v -run TestAlert
```

## Integration

The alert system is automatically integrated into the log service. When analysis results are saved, they are evaluated against all active alert rules:

```go
// In LogService.saveAnalysisResults()
if s.alertEngine != nil {
    for _, result := range results {
        if err := s.alertEngine.EvaluateAnalysisResult(ctx, result); err != nil {
            log.Printf("Failed to evaluate alert for result %s: %v", result.ID, err)
        }
    }
}
```

## Monitoring

The alert system provides the following monitoring capabilities:

- **Rule Updates**: Automatically reloads active rules every 5 minutes
- **Error Logging**: Logs all alert evaluation and notification errors
- **Suppression Tracking**: Tracks active suppressions and cleanup
- **Health Checks**: Validates notification channel configurations

## Security Considerations

- **SMTP Credentials**: Store email credentials securely using environment variables
- **DingTalk Secrets**: Use webhook secrets for secure DingTalk integration
- **TLS/STARTTLS**: Enable secure email transmission
- **Input Validation**: All alert rule inputs are validated before processing