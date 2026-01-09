package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// Redis keys
	taskQueueKey      = "task:queue"
	taskDataKey       = "task:data:%s"
	taskStatusKey     = "task:status"
	taskScheduleKey   = "task:schedule"
	taskProcessingKey = "task:processing"
	taskStatsKey      = "task:stats"
)

// RedisTaskQueue implements TaskQueue using Redis
type RedisTaskQueue struct {
	client *redis.Client
}

// NewRedisTaskQueue creates a new Redis-based task queue
func NewRedisTaskQueue(client *redis.Client) *RedisTaskQueue {
	return &RedisTaskQueue{
		client: client,
	}
}

// Enqueue adds a task to the queue
func (q *RedisTaskQueue) Enqueue(ctx context.Context, task *Task) error {
	// Serialize task data
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Use Redis pipeline for atomic operations
	pipe := q.client.Pipeline()

	// Store task data
	taskKey := fmt.Sprintf(taskDataKey, task.ID)
	pipe.Set(ctx, taskKey, taskData, 24*time.Hour) // TTL of 24 hours

	// Add task to appropriate queue based on status
	switch task.Status {
	case TaskStatusPending:
		// Add to main queue with priority (higher score = higher priority)
		pipe.ZAdd(ctx, taskQueueKey, &redis.Z{
			Score:  float64(task.Priority),
			Member: task.ID,
		})
	case TaskStatusRetrying:
		// Add to scheduled queue with timestamp
		pipe.ZAdd(ctx, taskScheduleKey, &redis.Z{
			Score:  float64(task.ScheduledAt.Unix()),
			Member: task.ID,
		})
	}

	// Update task status tracking
	pipe.HSet(ctx, taskStatusKey, task.ID, string(task.Status))

	// Update statistics
	pipe.HIncrBy(ctx, taskStatsKey, "total_tasks", 1)
	pipe.HIncrBy(ctx, taskStatsKey, string(task.Status)+"_tasks", 1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	return nil
}

// Dequeue retrieves the next available task from the queue
func (q *RedisTaskQueue) Dequeue(ctx context.Context) (*Task, error) {
	return q.DequeueWithTimeout(ctx, 0)
}

// DequeueWithTimeout retrieves the next available task with timeout
func (q *RedisTaskQueue) DequeueWithTimeout(ctx context.Context, timeout time.Duration) (*Task, error) {
	// First, move any scheduled tasks that are ready to the main queue
	if err := q.moveScheduledTasks(ctx); err != nil {
		return nil, fmt.Errorf("failed to move scheduled tasks: %w", err)
	}

	// Get the highest priority task from the queue
	var taskID string
	var err error

	if timeout > 0 {
		// Blocking pop with timeout
		result, err := q.client.BZPopMax(ctx, timeout, taskQueueKey).Result()
		if err != nil {
			if err == redis.Nil {
				return nil, nil // No task available
			}
			return nil, fmt.Errorf("failed to dequeue task: %w", err)
		}
		taskID = result.Member.(string)
	} else {
		// Non-blocking pop
		result, err := q.client.ZPopMax(ctx, taskQueueKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to dequeue task: %w", err)
		}
		if len(result) == 0 {
			return nil, nil // No task available
		}
		taskID = result[0].Member.(string)
	}

	// Get task data
	task, err := q.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Mark task as processing
	task.MarkAsProcessing()

	// Update task in Redis
	if err := q.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Add to processing set with timeout
	q.client.ZAdd(ctx, taskProcessingKey, &redis.Z{
		Score:  float64(time.Now().Add(10 * time.Minute).Unix()), // 10 minute timeout
		Member: taskID,
	})

	return task, nil
}

// UpdateTask updates an existing task
func (q *RedisTaskQueue) UpdateTask(ctx context.Context, task *Task) error {
	// Serialize task data
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	pipe := q.client.Pipeline()

	// Update task data
	taskKey := fmt.Sprintf(taskDataKey, task.ID)
	pipe.Set(ctx, taskKey, taskData, 24*time.Hour)

	// Update status tracking
	oldStatus, err := q.client.HGet(ctx, taskStatusKey, task.ID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get old task status: %w", err)
	}

	pipe.HSet(ctx, taskStatusKey, task.ID, string(task.Status))

	// Update statistics
	if oldStatus != "" && oldStatus != string(task.Status) {
		pipe.HIncrBy(ctx, taskStatsKey, oldStatus+"_tasks", -1)
		pipe.HIncrBy(ctx, taskStatsKey, string(task.Status)+"_tasks", 1)
	}

	// Remove from processing set if completed or failed
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		pipe.ZRem(ctx, taskProcessingKey, task.ID)
	}

	// Add to scheduled queue if retrying
	if task.Status == TaskStatusRetrying {
		pipe.ZAdd(ctx, taskScheduleKey, &redis.Z{
			Score:  float64(task.ScheduledAt.Unix()),
			Member: task.ID,
		})
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID
func (q *RedisTaskQueue) GetTask(ctx context.Context, taskID string) (*Task, error) {
	taskKey := fmt.Sprintf(taskDataKey, taskID)
	taskData, err := q.client.Get(ctx, taskKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// GetTasks retrieves tasks based on filter criteria
func (q *RedisTaskQueue) GetTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	var taskIDs []string

	if filter.Status != nil {
		// Get tasks by status from status tracking
		allTaskIDs, err := q.client.HKeys(ctx, taskStatusKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get task IDs: %w", err)
		}

		for _, taskID := range allTaskIDs {
			status, err := q.client.HGet(ctx, taskStatusKey, taskID).Result()
			if err != nil {
				continue
			}
			if status == string(*filter.Status) {
				taskIDs = append(taskIDs, taskID)
			}
		}
	} else {
		// Get all task IDs
		var err error
		taskIDs, err = q.client.HKeys(ctx, taskStatusKey).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get task IDs: %w", err)
		}
	}

	// Apply pagination
	start := filter.Offset
	end := start + filter.Limit
	if filter.Limit == 0 {
		end = len(taskIDs)
	}
	if start >= len(taskIDs) {
		return []*Task{}, nil
	}
	if end > len(taskIDs) {
		end = len(taskIDs)
	}

	taskIDs = taskIDs[start:end]

	// Retrieve task data
	var tasks []*Task
	for _, taskID := range taskIDs {
		task, err := q.GetTask(ctx, taskID)
		if err != nil {
			continue // Skip tasks that can't be retrieved
		}

		// Apply additional filters
		if filter.Type != nil && task.Type != *filter.Type {
			continue
		}
		if filter.StartTime != nil && task.CreatedAt.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && task.CreatedAt.After(*filter.EndTime) {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// DeleteTask removes a task from the queue
func (q *RedisTaskQueue) DeleteTask(ctx context.Context, taskID string) error {
	pipe := q.client.Pipeline()

	// Remove task data
	taskKey := fmt.Sprintf(taskDataKey, taskID)
	pipe.Del(ctx, taskKey)

	// Remove from all queues and sets
	pipe.ZRem(ctx, taskQueueKey, taskID)
	pipe.ZRem(ctx, taskScheduleKey, taskID)
	pipe.ZRem(ctx, taskProcessingKey, taskID)

	// Get current status for statistics update
	status, err := q.client.HGet(ctx, taskStatusKey, taskID).Result()
	if err == nil {
		pipe.HIncrBy(ctx, taskStatsKey, status+"_tasks", -1)
		pipe.HIncrBy(ctx, taskStatsKey, "total_tasks", -1)
	}

	// Remove from status tracking
	pipe.HDel(ctx, taskStatusKey, taskID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// GetStats returns queue statistics
func (q *RedisTaskQueue) GetStats(ctx context.Context) (*TaskStats, error) {
	stats, err := q.client.HGetAll(ctx, taskStatsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	taskStats := &TaskStats{}

	if val, ok := stats["total_tasks"]; ok {
		taskStats.TotalTasks, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := stats["pending_tasks"]; ok {
		taskStats.PendingTasks, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := stats["processing_tasks"]; ok {
		taskStats.ProcessingTasks, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := stats["completed_tasks"]; ok {
		taskStats.CompletedTasks, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := stats["failed_tasks"]; ok {
		taskStats.FailedTasks, _ = strconv.ParseInt(val, 10, 64)
	}
	if val, ok := stats["retrying_tasks"]; ok {
		taskStats.RetryingTasks, _ = strconv.ParseInt(val, 10, 64)
	}

	return taskStats, nil
}

// CleanupExpiredTasks removes expired tasks
func (q *RedisTaskQueue) CleanupExpiredTasks(ctx context.Context) (int64, error) {
	// Get all task IDs
	taskIDs, err := q.client.HKeys(ctx, taskStatusKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get task IDs: %w", err)
	}

	var expiredCount int64
	for _, taskID := range taskIDs {
		task, err := q.GetTask(ctx, taskID)
		if err != nil {
			continue
		}

		if task.IsExpired() {
			if err := q.DeleteTask(ctx, taskID); err == nil {
				expiredCount++
			}
		}
	}

	return expiredCount, nil
}

// Close closes the queue connection
func (q *RedisTaskQueue) Close() error {
	// Redis client is managed externally, so we don't close it here
	return nil
}

// moveScheduledTasks moves scheduled tasks that are ready to the main queue
func (q *RedisTaskQueue) moveScheduledTasks(ctx context.Context) error {
	now := float64(time.Now().Unix())

	// Get tasks that are ready to be processed
	readyTasks, err := q.client.ZRangeByScore(ctx, taskScheduleKey, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil {
		return err
	}

	if len(readyTasks) == 0 {
		return nil
	}

	pipe := q.client.Pipeline()

	for _, taskID := range readyTasks {
		// Get task to determine priority
		task, err := q.GetTask(ctx, taskID)
		if err != nil {
			continue
		}

		// Move to main queue
		pipe.ZAdd(ctx, taskQueueKey, &redis.Z{
			Score:  float64(task.Priority),
			Member: taskID,
		})

		// Remove from scheduled queue
		pipe.ZRem(ctx, taskScheduleKey, taskID)

		// Update status
		task.Status = TaskStatusPending
		task.UpdatedAt = time.Now()

		taskData, err := json.Marshal(task)
		if err != nil {
			continue
		}

		taskKey := fmt.Sprintf(taskDataKey, taskID)
		pipe.Set(ctx, taskKey, taskData, 24*time.Hour)
		pipe.HSet(ctx, taskStatusKey, taskID, string(TaskStatusPending))

		// Update statistics
		pipe.HIncrBy(ctx, taskStatsKey, "retrying_tasks", -1)
		pipe.HIncrBy(ctx, taskStatsKey, "pending_tasks", 1)
	}

	_, err = pipe.Exec(ctx)
	return err
}
