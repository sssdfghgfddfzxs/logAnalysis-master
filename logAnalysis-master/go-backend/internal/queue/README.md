# Task Queue and Scheduling System

This package implements a Redis-based task queue and scheduling system for the intelligent log analysis service. It provides asynchronous task processing with retry mechanisms, status tracking, and failure recovery.

## Features

- **Redis-based Task Queue**: Persistent task storage using Redis
- **Task Scheduling**: Automatic task distribution to worker processes
- **Retry Mechanism**: Exponential backoff retry for failed tasks
- **Status Tracking**: Real-time task status monitoring
- **Priority Support**: Task prioritization for important operations
- **Statistics**: Comprehensive queue and scheduler statistics
- **Cleanup**: Automatic cleanup of expired tasks

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Log Service   │───▶│  Task Queue     │───▶│  Task Scheduler │
│                 │    │  (Redis)        │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                              ┌─────────────────┐
                                              │ Task Processors │
                                              │ - Log Analysis  │
                                              └─────────────────┘
```

## Components

### 1. Task Queue (`TaskQueue` interface)
- **RedisTaskQueue**: Redis-based implementation
- Supports task enqueueing, dequeuing, and management
- Handles task priorities and scheduling
- Provides statistics and monitoring

### 2. Task Scheduler (`TaskScheduler` interface)
- **DefaultScheduler**: Multi-worker task scheduler
- Manages worker goroutines for task processing
- Handles task distribution and load balancing
- Implements retry logic with exponential backoff

### 3. Task Processors (`TaskProcessor` interface)
- **LogAnalysisProcessor**: Processes log analysis tasks
- Integrates with AI service for log analysis
- Handles task-specific logic and error handling

### 4. Queue Service (`QueueService`)
- High-level service for task queue management
- Integrates queue, scheduler, and processors
- Provides convenient API for task operations

## Task Types

### Log Analysis Task
- **Type**: `TaskTypeLogAnalysis`
- **Payload**: `{"log_ids": ["id1", "id2", ...]}`
- **Purpose**: Analyze logs using AI service
- **Result**: Analysis results with anomaly detection

## Task States

```
┌─────────┐    ┌────────────┐    ┌───────────┐
│ PENDING │───▶│ PROCESSING │───▶│ COMPLETED │
└─────────┘    └────────────┘    └───────────┘
     │              │
     │              ▼
     │         ┌─────────┐    ┌───────────┐
     └────────▶│ FAILED  │───▶│ RETRYING  │
               └─────────┘    └───────────┘
                                    │
                                    ▼
                               ┌─────────┐
                               │ PENDING │
                               └─────────┘
```

## Usage Examples

### Basic Task Creation and Processing

```go
// Create queue service
queueService := queue.NewQueueService(redisClient, repo, aiClient)

// Start the service
ctx := context.Background()
err := queueService.Start(ctx)

// Schedule a log analysis task
logIDs := []string{"log-1", "log-2", "log-3"}
task, err := queueService.ScheduleLogAnalysis(ctx, logIDs)

// Get task status
task, err = queueService.GetTask(ctx, task.ID)
fmt.Printf("Task Status: %s\n", task.Status)
```

### Manual Task Queue Operations

```go
// Create task queue
taskQueue := queue.NewRedisTaskQueue(redisClient)

// Create and enqueue task
task := queue.NewLogAnalysisTask([]string{"log-1", "log-2"})
task.Priority = 5
err := taskQueue.Enqueue(ctx, task)

// Dequeue and process task
dequeuedTask, err := taskQueue.Dequeue(ctx)
if dequeuedTask != nil {
    // Process task...
    dequeuedTask.MarkAsCompleted(result)
    taskQueue.UpdateTask(ctx, dequeuedTask)
}
```

### Task Scheduler with Custom Processor

```go
// Create scheduler
scheduler := queue.NewDefaultScheduler(taskQueue, 3) // 3 workers

// Create and register processor
processor := queue.NewLogAnalysisProcessor(repo, aiClient)
scheduler.RegisterProcessor(processor)

// Start scheduler
err := scheduler.Start(ctx)

// Schedule tasks
task := queue.NewLogAnalysisTask(logIDs)
err = scheduler.ScheduleTask(ctx, task)
```

## Configuration

### Redis Configuration
```go
redisConfig := &config.RedisConfig{
    Host:     "localhost",
    Port:     "6379",
    Password: "",
    DB:       0,
}
```

### Scheduler Configuration
```go
// Number of worker goroutines
workers := 3

// Task retry configuration
task.MaxRetries = 3

// Task timeout (handled by processor)
timeout := 10 * time.Minute
```

## Monitoring and Statistics

### Queue Statistics
```go
stats, err := queueService.GetQueueStats(ctx)
fmt.Printf("Total Tasks: %d\n", stats.TotalTasks)
fmt.Printf("Pending: %d\n", stats.PendingTasks)
fmt.Printf("Processing: %d\n", stats.ProcessingTasks)
fmt.Printf("Completed: %d\n", stats.CompletedTasks)
fmt.Printf("Failed: %d\n", stats.FailedTasks)
```

### Scheduler Statistics
```go
stats := queueService.GetSchedulerStats()
fmt.Printf("Workers: %d\n", stats["workers_count"])
fmt.Printf("Processed: %d\n", stats["processed_tasks"])
fmt.Printf("Failed: %d\n", stats["failed_tasks"])
```

## Error Handling

### Retry Mechanism
- Tasks are automatically retried on failure
- Exponential backoff: 1min, 4min, 9min, etc.
- Maximum retry limit: 3 attempts by default
- Failed tasks are marked as permanently failed after max retries

### Failure Recovery
- Tasks stuck in processing state are detected and recovered
- Expired tasks (>24 hours) are automatically cleaned up
- Redis connection failures are handled gracefully

## API Endpoints

The task queue system exposes REST API endpoints:

- `GET /api/v1/queue/stats` - Get queue statistics
- `GET /api/v1/queue/tasks` - List tasks with filtering
- `GET /api/v1/queue/tasks/:id` - Get specific task
- `POST /api/v1/queue/tasks` - Schedule new task
- `DELETE /api/v1/queue/tasks/:id` - Delete task

## Testing

Run the test suite:
```bash
go test ./internal/queue -v
```

Run the example:
```bash
go run examples/queue_example.go
```

## Performance Considerations

- **Redis Memory**: Tasks are stored in Redis with 24-hour TTL
- **Worker Scaling**: Adjust worker count based on CPU and AI service capacity
- **Batch Processing**: Group multiple logs in single analysis task for efficiency
- **Monitoring**: Monitor queue depth and processing times

## Integration with Log Service

The task queue is integrated with the log service to automatically schedule AI analysis:

```go
// When logs are created, analysis tasks are automatically scheduled
err := logService.CreateLogs(ctx, logs)
// This triggers: queueService.ScheduleLogAnalysis(ctx, logIDs)
```

## Requirements Validation

This implementation satisfies the following requirements:

- **需求 1.4**: Task scheduling for AI analysis when logs are received
- **需求 2.4**: Asynchronous AI analysis task processing
- **Redis-based**: Uses Redis for persistent task storage
- **Retry Mechanism**: Implements exponential backoff retry
- **Status Tracking**: Comprehensive task status monitoring
- **Failure Recovery**: Handles various failure scenarios