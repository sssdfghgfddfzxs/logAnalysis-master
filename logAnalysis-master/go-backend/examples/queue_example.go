//go:build examples
// +build examples

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"intelligent-log-analysis/internal/queue"

	"github.com/go-redis/redis/v8"
)

// This example demonstrates how to use the task queue system
func main() {
	fmt.Println("Task Queue System Example")
	fmt.Println("========================")

	// Create a Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Create task queue
	taskQueue := queue.NewRedisTaskQueue(redisClient)

	fmt.Println("✓ Connected to Redis")

	// Example 1: Create and enqueue a task
	fmt.Println("\n1. Creating and enqueueing a task...")

	task := queue.NewLogAnalysisTask([]string{"log-1", "log-2", "log-3"})
	task.Priority = 5

	if err := taskQueue.Enqueue(ctx, task); err != nil {
		log.Fatalf("Failed to enqueue task: %v", err)
	}

	fmt.Printf("✓ Task %s enqueued successfully\n", task.ID)
	fmt.Printf("  - Type: %s\n", task.Type)
	fmt.Printf("  - Status: %s\n", task.Status)
	fmt.Printf("  - Priority: %d\n", task.Priority)
	fmt.Printf("  - Log IDs: %v\n", task.Payload["log_ids"])

	// Example 2: Get queue statistics
	fmt.Println("\n2. Getting queue statistics...")

	stats, err := taskQueue.GetStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}

	fmt.Printf("✓ Queue Statistics:\n")
	fmt.Printf("  - Total Tasks: %d\n", stats.TotalTasks)
	fmt.Printf("  - Pending Tasks: %d\n", stats.PendingTasks)
	fmt.Printf("  - Processing Tasks: %d\n", stats.ProcessingTasks)
	fmt.Printf("  - Completed Tasks: %d\n", stats.CompletedTasks)
	fmt.Printf("  - Failed Tasks: %d\n", stats.FailedTasks)

	// Example 3: Dequeue and process a task
	fmt.Println("\n3. Dequeuing and processing a task...")

	dequeuedTask, err := taskQueue.Dequeue(ctx)
	if err != nil {
		log.Fatalf("Failed to dequeue task: %v", err)
	}

	if dequeuedTask != nil {
		fmt.Printf("✓ Task %s dequeued successfully\n", dequeuedTask.ID)
		fmt.Printf("  - Status: %s\n", dequeuedTask.Status)
		fmt.Printf("  - Started At: %v\n", dequeuedTask.StartedAt)

		// Simulate task processing
		fmt.Println("  - Simulating task processing...")
		time.Sleep(2 * time.Second)

		// Mark task as completed
		dequeuedTask.MarkAsCompleted(map[string]interface{}{
			"processed_logs":  3,
			"anomalies_found": 1,
		})

		if err := taskQueue.UpdateTask(ctx, dequeuedTask); err != nil {
			log.Fatalf("Failed to update task: %v", err)
		}

		fmt.Printf("✓ Task %s completed successfully\n", dequeuedTask.ID)
		fmt.Printf("  - Status: %s\n", dequeuedTask.Status)
		fmt.Printf("  - Result: %v\n", dequeuedTask.Result)
	} else {
		fmt.Println("No tasks available in queue")
	}

	// Example 4: Create a task that will be retried
	fmt.Println("\n4. Creating a task that will be retried...")

	retryTask := queue.NewLogAnalysisTask([]string{"log-4"})
	if err := taskQueue.Enqueue(ctx, retryTask); err != nil {
		log.Fatalf("Failed to enqueue retry task: %v", err)
	}

	// Dequeue and simulate failure
	failedTask, err := taskQueue.Dequeue(ctx)
	if err != nil {
		log.Fatalf("Failed to dequeue retry task: %v", err)
	}

	if failedTask != nil {
		fmt.Printf("✓ Task %s dequeued for retry example\n", failedTask.ID)

		// Simulate task failure and retry
		if failedTask.ShouldRetry() {
			failedTask.MarkAsRetrying(5 * time.Second)
			if err := taskQueue.UpdateTask(ctx, failedTask); err != nil {
				log.Fatalf("Failed to update retry task: %v", err)
			}

			fmt.Printf("✓ Task %s marked for retry\n", failedTask.ID)
			fmt.Printf("  - Retry Count: %d/%d\n", failedTask.RetryCount, failedTask.MaxRetries)
			fmt.Printf("  - Scheduled At: %v\n", failedTask.ScheduledAt)
		}
	}

	// Example 5: Get updated statistics
	fmt.Println("\n5. Getting updated queue statistics...")

	updatedStats, err := taskQueue.GetStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get updated stats: %v", err)
	}

	fmt.Printf("✓ Updated Queue Statistics:\n")
	fmt.Printf("  - Total Tasks: %d\n", updatedStats.TotalTasks)
	fmt.Printf("  - Pending Tasks: %d\n", updatedStats.PendingTasks)
	fmt.Printf("  - Processing Tasks: %d\n", updatedStats.ProcessingTasks)
	fmt.Printf("  - Completed Tasks: %d\n", updatedStats.CompletedTasks)
	fmt.Printf("  - Failed Tasks: %d\n", updatedStats.FailedTasks)
	fmt.Printf("  - Retrying Tasks: %d\n", updatedStats.RetryingTasks)

	fmt.Println("\n✓ Task Queue System Example completed successfully!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- Task creation and enqueueing")
	fmt.Println("- Task dequeuing and processing")
	fmt.Println("- Task status tracking")
	fmt.Println("- Retry mechanism with exponential backoff")
	fmt.Println("- Queue statistics and monitoring")
	fmt.Println("- Redis-based persistence")

	// Clean up
	redisClient.Close()
}
