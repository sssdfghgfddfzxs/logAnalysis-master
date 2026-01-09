package service

import (
	"context"
	"testing"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/repository"

	"github.com/google/uuid"
)

// Mock repository for testing
type mockLogRepository struct {
	logs  map[string]*models.LogEntry
	count int64
}

func newMockLogRepository() *mockLogRepository {
	return &mockLogRepository{
		logs: make(map[string]*models.LogEntry),
	}
}

func (m *mockLogRepository) SaveLog(ctx context.Context, log *models.LogEntry) error {
	m.logs[log.ID] = log
	m.count++
	return nil
}

func (m *mockLogRepository) SaveLogs(ctx context.Context, logs []*models.LogEntry) error {
	for _, log := range logs {
		m.logs[log.ID] = log
		m.count++
	}
	return nil
}

func (m *mockLogRepository) GetLogByID(ctx context.Context, id string) (*models.LogEntry, error) {
	log, exists := m.logs[id]
	if !exists {
		return nil, nil
	}
	return log, nil
}

func (m *mockLogRepository) GetLogs(ctx context.Context, filter repository.LogFilter) ([]*models.LogEntry, error) {
	var results []*models.LogEntry
	for _, log := range m.logs {
		if filter.Level != "" && filter.Level != "all" && log.Level != filter.Level {
			continue
		}
		if filter.Source != "" && filter.Source != "all" && log.Source != filter.Source {
			continue
		}
		results = append(results, log)
	}
	return results, nil
}

func (m *mockLogRepository) CountLogs(ctx context.Context, filter repository.LogFilter) (int64, error) {
	logs, _ := m.GetLogs(ctx, filter)
	return int64(len(logs)), nil
}

func (m *mockLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	deleted := int64(0)
	for id, log := range m.logs {
		if log.Timestamp.Before(cutoff) {
			delete(m.logs, id)
			deleted++
		}
	}
	return deleted, nil
}

func TestLogService_CreateLog(t *testing.T) {
	mockRepo := &repository.Repository{
		Log: newMockLogRepository(),
	}
	service := NewLogService(mockRepo)
	ctx := context.Background()

	log := &models.LogEntry{
		ID:      uuid.New().String(),
		Level:   "ERROR",
		Message: "Test error message",
		Source:  "test-service",
	}

	err := service.CreateLog(ctx, log)
	if err != nil {
		t.Errorf("CreateLog failed: %v", err)
	}

	// Verify the log was saved
	savedLog, err := service.GetLog(ctx, log.ID)
	if err != nil {
		t.Errorf("GetLog failed: %v", err)
	}

	if savedLog == nil {
		t.Error("Expected log to be saved, but got nil")
		return
	}

	if savedLog.Message != log.Message {
		t.Errorf("Expected message %s, got %s", log.Message, savedLog.Message)
	}
}

func TestLogService_CreateLog_Validation(t *testing.T) {
	mockRepo := &repository.Repository{
		Log: newMockLogRepository(),
	}
	service := NewLogService(mockRepo)
	ctx := context.Background()

	// Test empty message
	log := &models.LogEntry{
		ID:      uuid.New().String(),
		Level:   "ERROR",
		Message: "", // Empty message should fail
		Source:  "test-service",
	}

	err := service.CreateLog(ctx, log)
	if err == nil {
		t.Error("Expected validation error for empty message, but got nil")
	}

	// Test invalid level
	log = &models.LogEntry{
		ID:      uuid.New().String(),
		Level:   "INVALID", // Invalid level should fail
		Message: "Test message",
		Source:  "test-service",
	}

	err = service.CreateLog(ctx, log)
	if err == nil {
		t.Error("Expected validation error for invalid level, but got nil")
	}
}

func TestLogService_GetLogs_WithFilter(t *testing.T) {
	mockRepo := &repository.Repository{
		Log: newMockLogRepository(),
	}
	service := NewLogService(mockRepo)
	ctx := context.Background()

	// Create test logs
	logs := []*models.LogEntry{
		{
			ID:      uuid.New().String(),
			Level:   "ERROR",
			Message: "Error 1",
			Source:  "service-1",
		},
		{
			ID:      uuid.New().String(),
			Level:   "INFO",
			Message: "Info 1",
			Source:  "service-2",
		},
	}

	// Save logs
	for _, log := range logs {
		err := service.CreateLog(ctx, log)
		if err != nil {
			t.Errorf("CreateLog failed: %v", err)
		}
	}

	// Test filtering by level
	filter := repository.LogFilter{
		Level: "ERROR",
		Limit: 10,
	}

	results, err := service.GetLogs(ctx, filter)
	if err != nil {
		t.Errorf("GetLogs failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Level != "ERROR" {
		t.Errorf("Expected ERROR level, got %s", results[0].Level)
	}
}
