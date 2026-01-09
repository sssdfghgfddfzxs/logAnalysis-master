package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DefaultScheduler implements TaskScheduler
type DefaultScheduler struct {
	queue      TaskQueue
	processors map[TaskType]TaskProcessor
	workers    int
	running    bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	stats      *SchedulerStats
}

// SchedulerStats holds scheduler statistics
type SchedulerStats struct {
	WorkersCount    int                    `json:"workers_count"`
	ProcessedTasks  int64                  `json:"processed_tasks"`
	FailedTasks     int64                  `json:"failed_tasks"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	LastProcessedAt *time.Time             `json:"last_processed_at,omitempty"`
	ProcessorStats  map[string]interface{} `json:"processor_stats"`
	mu              sync.RWMutex
}

// NewDefaultScheduler creates a new task scheduler
func NewDefaultScheduler(queue TaskQueue, workers int) *DefaultScheduler {
	if workers <= 0 {
		workers = 1
	}

	return &DefaultScheduler{
		queue:      queue,
		processors: make(map[TaskType]TaskProcessor),
		workers:    workers,
		stopCh:     make(chan struct{}),
		stats: &SchedulerStats{
			WorkersCount:   workers,
			ProcessorStats: make(map[string]interface{}),
		},
	}
}

// Start starts the scheduler
func (s *DefaultScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.running = true
	log.Printf("Starting task scheduler with %d workers", s.workers)

	// Start worker goroutines
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	// Start cleanup goroutine
	s.wg.Add(1)
	go s.cleanupWorker(ctx)

	return nil
}

// Stop stops the scheduler
func (s *DefaultScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	log.Println("Stopping task scheduler...")
	s.running = false
	close(s.stopCh)
	s.wg.Wait()
	log.Println("Task scheduler stopped")

	return nil
}

// RegisterProcessor registers a task processor
func (s *DefaultScheduler) RegisterProcessor(processor TaskProcessor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find which task types this processor can handle
	for _, taskType := range []TaskType{TaskTypeLogAnalysis} {
		if processor.CanProcess(taskType) {
			s.processors[taskType] = processor
			log.Printf("Registered processor %s for task type %s",
				processor.GetProcessorName(), taskType)
		}
	}
}

// ScheduleTask schedules a task for processing
func (s *DefaultScheduler) ScheduleTask(ctx context.Context, task *Task) error {
	// Check if we have a processor for this task type
	s.mu.RLock()
	_, hasProcessor := s.processors[task.Type]
	s.mu.RUnlock()

	if !hasProcessor {
		return fmt.Errorf("no processor registered for task type: %s", task.Type)
	}

	// Enqueue the task
	return s.queue.Enqueue(ctx, task)
}

// GetSchedulerStats returns scheduler statistics
func (s *DefaultScheduler) GetSchedulerStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	stats := map[string]interface{}{
		"workers_count":     s.stats.WorkersCount,
		"processed_tasks":   s.stats.ProcessedTasks,
		"failed_tasks":      s.stats.FailedTasks,
		"processing_time":   s.stats.ProcessingTime.String(),
		"last_processed_at": s.stats.LastProcessedAt,
		"processor_stats":   s.stats.ProcessorStats,
		"running":           s.running,
	}

	// Add queue stats if available
	if queueStats, err := s.queue.GetStats(context.Background()); err == nil {
		stats["queue_stats"] = queueStats
	}

	return stats
}

// worker is the main worker goroutine that processes tasks
func (s *DefaultScheduler) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()

	log.Printf("Worker %d started", workerID)
	defer log.Printf("Worker %d stopped", workerID)

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// Try to dequeue a task with timeout
			task, err := s.queue.DequeueWithTimeout(ctx, 5*time.Second)
			if err != nil {
				log.Printf("Worker %d: failed to dequeue task: %v", workerID, err)
				continue
			}

			if task == nil {
				// No task available, continue
				continue
			}

			// Process the task
			s.processTask(ctx, workerID, task)
		}
	}
}

// processTask processes a single task
func (s *DefaultScheduler) processTask(ctx context.Context, workerID int, task *Task) {
	startTime := time.Now()
	log.Printf("Worker %d: processing task %s (type: %s)", workerID, task.ID, task.Type)

	// Get the appropriate processor
	s.mu.RLock()
	processor, exists := s.processors[task.Type]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Worker %d: no processor found for task type %s", workerID, task.Type)
		task.MarkAsFailed(fmt.Sprintf("no processor found for task type: %s", task.Type))
		s.queue.UpdateTask(ctx, task)
		s.updateFailedStats()
		return
	}

	// Create a timeout context for task processing
	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Process the task
	err := processor.ProcessTask(taskCtx, task)
	processingTime := time.Since(startTime)

	if err != nil {
		log.Printf("Worker %d: task %s failed: %v", workerID, task.ID, err)

		// Check if task should be retried
		if task.ShouldRetry() {
			// Calculate retry delay with exponential backoff
			retryDelay := time.Duration(task.RetryCount*task.RetryCount) * time.Minute
			if retryDelay > 30*time.Minute {
				retryDelay = 30 * time.Minute
			}

			task.MarkAsRetrying(retryDelay)
			log.Printf("Worker %d: task %s scheduled for retry %d/%d in %v",
				workerID, task.ID, task.RetryCount, task.MaxRetries, retryDelay)
		} else {
			task.MarkAsFailed(err.Error())
			log.Printf("Worker %d: task %s failed permanently after %d retries",
				workerID, task.ID, task.RetryCount)
		}

		s.updateFailedStats()
	} else {
		// Task completed successfully
		task.MarkAsCompleted(task.Result)
		log.Printf("Worker %d: task %s completed successfully in %v",
			workerID, task.ID, processingTime)
		s.updateProcessedStats(processingTime)
	}

	// Update task in queue
	if err := s.queue.UpdateTask(ctx, task); err != nil {
		log.Printf("Worker %d: failed to update task %s: %v", workerID, task.ID, err)
	}
}

// cleanupWorker periodically cleans up expired tasks
func (s *DefaultScheduler) cleanupWorker(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Clean up expired tasks
			if count, err := s.queue.CleanupExpiredTasks(ctx); err != nil {
				log.Printf("Cleanup worker: failed to cleanup expired tasks: %v", err)
			} else if count > 0 {
				log.Printf("Cleanup worker: cleaned up %d expired tasks", count)
			}
		}
	}
}

// updateProcessedStats updates processed task statistics
func (s *DefaultScheduler) updateProcessedStats(processingTime time.Duration) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	s.stats.ProcessedTasks++
	s.stats.ProcessingTime += processingTime
	now := time.Now()
	s.stats.LastProcessedAt = &now
}

// updateFailedStats updates failed task statistics
func (s *DefaultScheduler) updateFailedStats() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	s.stats.FailedTasks++
}
