package queue

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *redis.Client {
	// Use Redis database 15 for testing to avoid conflicts
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	// Test connection
	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up test database
	client.FlushDB(ctx)

	return client
}

func TestRedisTaskQueue_EnqueueDequeue(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	queue := NewRedisTaskQueue(client)
	ctx := context.Background()

	// Create a test task
	task := NewLogAnalysisTask([]string{"log1", "log2"})
	task.Priority = 5

	// Enqueue the task
	err := queue.Enqueue(ctx, task)
	require.NoError(t, err)

	// Dequeue the task
	dequeuedTask, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	require.NotNil(t, dequeuedTask)

	// Verify task properties
	assert.Equal(t, task.ID, dequeuedTask.ID)
	assert.Equal(t, task.Type, dequeuedTask.Type)
	assert.Equal(t, TaskStatusProcessing, dequeuedTask.Status)
	assert.Equal(t, task.Priority, dequeuedTask.Priority)
	assert.NotNil(t, dequeuedTask.StartedAt)
}

func TestRedisTaskQueue_TaskRetry(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	queue := NewRedisTaskQueue(client)
	ctx := context.Background()

	// Create a test task
	task := NewLogAnalysisTask([]string{"log1"})

	// Enqueue the task
	err := queue.Enqueue(ctx, task)
	require.NoError(t, err)

	// Dequeue and mark as failed for retry
	dequeuedTask, err := queue.Dequeue(ctx)
	require.NoError(t, err)

	// Mark for retry
	dequeuedTask.MarkAsRetrying(1 * time.Second)
	err = queue.UpdateTask(ctx, dequeuedTask)
	require.NoError(t, err)

	// Verify task is in retrying status
	retrievedTask, err := queue.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusRetrying, retrievedTask.Status)
	assert.Equal(t, 1, retrievedTask.RetryCount)

	// Wait for retry time to pass
	time.Sleep(2 * time.Second)

	// Task should be available for dequeue again
	retriedTask, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	require.NotNil(t, retriedTask)
	assert.Equal(t, task.ID, retriedTask.ID)
	assert.Equal(t, TaskStatusProcessing, retriedTask.Status)
}

func TestRedisTaskQueue_GetStats(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	queue := NewRedisTaskQueue(client)
	ctx := context.Background()

	// Create and enqueue multiple tasks
	for i := 0; i < 3; i++ {
		task := NewLogAnalysisTask([]string{"log1"})
		err := queue.Enqueue(ctx, task)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := queue.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(3), stats.TotalTasks)
	assert.Equal(t, int64(3), stats.PendingTasks)
	assert.Equal(t, int64(0), stats.ProcessingTasks)
	assert.Equal(t, int64(0), stats.CompletedTasks)
}

func TestRedisTaskQueue_CleanupExpiredTasks(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	queue := NewRedisTaskQueue(client)
	ctx := context.Background()

	// Create an expired task (simulate by setting old creation time)
	task := NewLogAnalysisTask([]string{"log1"})
	task.CreatedAt = time.Now().Add(-25 * time.Hour) // 25 hours ago

	err := queue.Enqueue(ctx, task)
	require.NoError(t, err)

	// Clean up expired tasks
	count, err := queue.CleanupExpiredTasks(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify task is deleted
	_, err = queue.GetTask(ctx, task.ID)
	assert.Error(t, err)
}
