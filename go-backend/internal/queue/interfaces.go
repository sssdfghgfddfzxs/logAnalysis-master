package queue

import (
	"context"
	"time"
)

// TaskQueue defines the interface for task queue operations
type TaskQueue interface {
	// Enqueue adds a task to the queue
	Enqueue(ctx context.Context, task *Task) error

	// Dequeue retrieves the next available task from the queue
	Dequeue(ctx context.Context) (*Task, error)

	// DequeueWithTimeout retrieves the next available task with timeout
	DequeueWithTimeout(ctx context.Context, timeout time.Duration) (*Task, error)

	// UpdateTask updates an existing task
	UpdateTask(ctx context.Context, task *Task) error

	// GetTask retrieves a task by ID
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// GetTasks retrieves tasks based on filter criteria
	GetTasks(ctx context.Context, filter TaskFilter) ([]*Task, error)

	// DeleteTask removes a task from the queue
	DeleteTask(ctx context.Context, taskID string) error

	// GetStats returns queue statistics
	GetStats(ctx context.Context) (*TaskStats, error)

	// CleanupExpiredTasks removes expired tasks
	CleanupExpiredTasks(ctx context.Context) (int64, error)

	// Close closes the queue connection
	Close() error
}

// TaskProcessor defines the interface for processing tasks
type TaskProcessor interface {
	// ProcessTask processes a single task
	ProcessTask(ctx context.Context, task *Task) error

	// CanProcess checks if the processor can handle the task type
	CanProcess(taskType TaskType) bool

	// GetProcessorName returns the name of the processor
	GetProcessorName() string
}

// TaskScheduler defines the interface for task scheduling
type TaskScheduler interface {
	// Start starts the scheduler
	Start(ctx context.Context) error

	// Stop stops the scheduler
	Stop() error

	// RegisterProcessor registers a task processor
	RegisterProcessor(processor TaskProcessor)

	// ScheduleTask schedules a task for processing
	ScheduleTask(ctx context.Context, task *Task) error

	// GetSchedulerStats returns scheduler statistics
	GetSchedulerStats() map[string]interface{}
}
