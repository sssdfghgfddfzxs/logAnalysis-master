package queue

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusRetrying   TaskStatus = "retrying"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeLogAnalysis TaskType = "log_analysis"
)

// Task represents a task in the queue
type Task struct {
	ID          string                 `json:"id"`
	Type        TaskType               `json:"type"`
	Status      TaskStatus             `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    int                    `json:"priority"`
	MaxRetries  int                    `json:"max_retries"`
	RetryCount  int                    `json:"retry_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
}

// NewTask creates a new task with default values
func NewTask(taskType TaskType, payload map[string]interface{}) *Task {
	now := time.Now()
	return &Task{
		ID:          uuid.New().String(),
		Type:        taskType,
		Status:      TaskStatusPending,
		Payload:     payload,
		Priority:    0,
		MaxRetries:  3,
		RetryCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: now,
	}
}

// NewLogAnalysisTask creates a new log analysis task
func NewLogAnalysisTask(logIDs []string) *Task {
	payload := map[string]interface{}{
		"log_ids": logIDs,
	}
	return NewTask(TaskTypeLogAnalysis, payload)
}

// MarkAsProcessing marks the task as processing
func (t *Task) MarkAsProcessing() {
	now := time.Now()
	t.Status = TaskStatusProcessing
	t.StartedAt = &now
	t.UpdatedAt = now
}

// MarkAsCompleted marks the task as completed
func (t *Task) MarkAsCompleted(result map[string]interface{}) {
	now := time.Now()
	t.Status = TaskStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now
	t.Result = result
}

// MarkAsFailed marks the task as failed
func (t *Task) MarkAsFailed(errorMsg string) {
	t.Status = TaskStatusFailed
	t.UpdatedAt = time.Now()
	t.ErrorMsg = errorMsg
}

// MarkAsRetrying marks the task for retry
func (t *Task) MarkAsRetrying(delay time.Duration) {
	t.Status = TaskStatusRetrying
	t.RetryCount++
	t.UpdatedAt = time.Now()
	t.ScheduledAt = time.Now().Add(delay)
}

// ShouldRetry checks if the task should be retried
func (t *Task) ShouldRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// IsExpired checks if the task has expired (older than 24 hours)
func (t *Task) IsExpired() bool {
	return time.Since(t.CreatedAt) > 24*time.Hour
}

// TaskFilter represents filtering criteria for task queries
type TaskFilter struct {
	Status    *TaskStatus
	Type      *TaskType
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// TaskStats represents task statistics
type TaskStats struct {
	TotalTasks      int64 `json:"total_tasks"`
	PendingTasks    int64 `json:"pending_tasks"`
	ProcessingTasks int64 `json:"processing_tasks"`
	CompletedTasks  int64 `json:"completed_tasks"`
	FailedTasks     int64 `json:"failed_tasks"`
	RetryingTasks   int64 `json:"retrying_tasks"`
}
