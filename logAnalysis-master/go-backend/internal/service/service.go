package service

import (
	"intelligent-log-analysis/internal/config"
	"intelligent-log-analysis/internal/grpc"
	"intelligent-log-analysis/internal/queue"
	"intelligent-log-analysis/internal/repository"
)

// Services aggregates all service interfaces
type Services struct {
	Log       *LogService
	Analysis  *AnalysisService
	AlertRule *AlertRuleService
	Alert     *AlertService
	Queue     *queue.QueueService
}

// NewServices creates a new services instance
func NewServices(repo *repository.Repository) *Services {
	return &Services{
		Log:       NewLogService(repo),
		Analysis:  NewAnalysisService(repo),
		AlertRule: NewAlertRuleService(repo),
	}
}

// NewServicesWithAI creates a new services instance with AI client
func NewServicesWithAI(repo *repository.Repository, aiClient *grpc.AIServiceClient) *Services {
	return &Services{
		Log:       NewLogServiceWithAI(repo, aiClient),
		Analysis:  NewAnalysisService(repo),
		AlertRule: NewAlertRuleService(repo),
	}
}

// NewServicesWithQueue creates a new services instance with queue service
func NewServicesWithQueue(repo *repository.Repository, aiClient *grpc.AIServiceClient, queueService *queue.QueueService) *Services {
	return &Services{
		Log:       NewLogServiceWithQueue(repo, aiClient, queueService),
		Analysis:  NewAnalysisService(repo),
		AlertRule: NewAlertRuleService(repo),
		Queue:     queueService,
	}
}

// NewServicesWithAlert creates a new services instance with alert service
func NewServicesWithAlert(repo *repository.Repository, aiClient *grpc.AIServiceClient, queueService *queue.QueueService, cfg *config.Config) *Services {
	alertService := NewAlertService(repo, cfg)

	return &Services{
		Log:       NewLogServiceWithAlert(repo, aiClient, queueService, alertService.GetAlertEngine()),
		Analysis:  NewAnalysisService(repo),
		AlertRule: NewAlertRuleService(repo),
		Alert:     alertService,
		Queue:     queueService,
	}
}
