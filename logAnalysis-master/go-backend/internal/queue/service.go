package queue

import (
	"context"
	"fmt"
	"log"

	"intelligent-log-analysis/internal/cache"
	"intelligent-log-analysis/internal/grpc"
	"intelligent-log-analysis/internal/repository"
)

// QueueService manages the task queue and scheduler
type QueueService struct {
	queue     TaskQueue
	scheduler TaskScheduler
	processor TaskProcessor
}

// NewQueueService creates a new queue service
func NewQueueService(redisClient *cache.RedisClient, repo *repository.Repository, aiClient *grpc.AIServiceClient) *QueueService {
	// Create Redis-based task queue
	queue := NewRedisTaskQueue(redisClient.GetClient())

	// Create task scheduler with 3 workers
	scheduler := NewDefaultScheduler(queue, 3)

	// Create log analysis processor
	processor := NewLogAnalysisProcessor(repo, aiClient)

	// Register processor with scheduler
	scheduler.RegisterProcessor(processor)

	return &QueueService{
		queue:     queue,
		scheduler: scheduler,
		processor: processor,
	}
}

// NewQueueServiceWithAlert creates a new queue service with alert engine
func NewQueueServiceWithAlert(redisClient *cache.RedisClient, repo *repository.Repository, aiClient *grpc.AIServiceClient, alertEngine AlertEngine) *QueueService {
	// Create Redis-based task queue
	queue := NewRedisTaskQueue(redisClient.GetClient())

	// Create task scheduler with 3 workers
	scheduler := NewDefaultScheduler(queue, 3)

	// Create log analysis processor with alert engine
	processor := NewLogAnalysisProcessorWithAlert(repo, aiClient, alertEngine)

	// Register processor with scheduler
	scheduler.RegisterProcessor(processor)

	return &QueueService{
		queue:     queue,
		scheduler: scheduler,
		processor: processor,
	}
}

// Start starts the queue service
func (s *QueueService) Start(ctx context.Context) error {
	log.Println("Starting queue service...")
	return s.scheduler.Start(ctx)
}

// Stop stops the queue service
func (s *QueueService) Stop() error {
	log.Println("Stopping queue service...")
	if err := s.scheduler.Stop(); err != nil {
		return err
	}
	return s.queue.Close()
}

// ScheduleLogAnalysis schedules a log analysis task
func (s *QueueService) ScheduleLogAnalysis(ctx context.Context, logIDs []string) (*Task, error) {
	if len(logIDs) == 0 {
		return nil, fmt.Errorf("no log IDs provided")
	}

	// Create log analysis task
	task := NewLogAnalysisTask(logIDs)

	// Schedule the task
	if err := s.scheduler.ScheduleTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to schedule log analysis task: %w", err)
	}

	log.Printf("Scheduled log analysis task %s for %d logs", task.ID, len(logIDs))
	return task, nil
}

// GetTask retrieves a task by ID
func (s *QueueService) GetTask(ctx context.Context, taskID string) (*Task, error) {
	return s.queue.GetTask(ctx, taskID)
}

// GetTasks retrieves tasks based on filter criteria
func (s *QueueService) GetTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	return s.queue.GetTasks(ctx, filter)
}

// GetQueueStats returns queue statistics
func (s *QueueService) GetQueueStats(ctx context.Context) (*TaskStats, error) {
	return s.queue.GetStats(ctx)
}

// GetSchedulerStats returns scheduler statistics
func (s *QueueService) GetSchedulerStats() map[string]interface{} {
	return s.scheduler.GetSchedulerStats()
}

// DeleteTask removes a task from the queue
func (s *QueueService) DeleteTask(ctx context.Context, taskID string) error {
	return s.queue.DeleteTask(ctx, taskID)
}

// CleanupExpiredTasks removes expired tasks
func (s *QueueService) CleanupExpiredTasks(ctx context.Context) (int64, error) {
	return s.queue.CleanupExpiredTasks(ctx)
}
