package websocket

import (
	"context"
	"log"
	"time"

	"intelligent-log-analysis/internal/models"
)

// Service provides WebSocket functionality for real-time updates
type Service struct {
	hub *Hub
}

// NewService creates a new WebSocket service
func NewService() *Service {
	return &Service{
		hub: NewHub(),
	}
}

// Start starts the WebSocket service
func (s *Service) Start(ctx context.Context) error {
	log.Println("Starting WebSocket service...")
	go s.hub.Run()
	return nil
}

// Stop stops the WebSocket service
func (s *Service) Stop() error {
	log.Println("Stopping WebSocket service...")
	s.hub.Shutdown()
	return nil
}

// GetHub returns the WebSocket hub for HTTP handler registration
func (s *Service) GetHub() *Hub {
	return s.hub
}

// BroadcastNewAnomaly broadcasts a new anomaly detection to all connected clients
func (s *Service) BroadcastNewAnomaly(logEntry *models.LogEntry, analysisResult *models.AnalysisResult) {
	if !analysisResult.IsAnomaly {
		return
	}

	data := map[string]interface{}{
		"log": map[string]interface{}{
			"id":              logEntry.ID,
			"timestamp":       logEntry.Timestamp.Format(time.RFC3339),
			"level":           logEntry.Level,
			"message":         logEntry.Message,
			"source":          logEntry.Source,
			"isAnomaly":       analysisResult.IsAnomaly,
			"anomalyScore":    analysisResult.AnomalyScore,
			"rootCauses":      analysisResult.RootCauses,
			"recommendations": analysisResult.Recommendations,
			"metadata":        logEntry.Metadata,
		},
		"analysis": map[string]interface{}{
			"id":              analysisResult.ID,
			"anomalyScore":    analysisResult.AnomalyScore,
			"rootCauses":      analysisResult.RootCauses,
			"recommendations": analysisResult.Recommendations,
			"analyzedAt":      analysisResult.AnalyzedAt.Format(time.RFC3339),
		},
	}

	if err := s.hub.BroadcastMessage("new_anomaly", data); err != nil {
		log.Printf("Failed to broadcast new anomaly: %v", err)
	}
}

// BroadcastStatsUpdate broadcasts updated statistics to all connected clients
func (s *Service) BroadcastStatsUpdate(stats interface{}) {
	if err := s.hub.BroadcastMessage("stats_update", map[string]interface{}{
		"stats": stats,
	}); err != nil {
		log.Printf("Failed to broadcast stats update: %v", err)
	}
}

// BroadcastSystemAlert broadcasts a system alert to all connected clients
func (s *Service) BroadcastSystemAlert(alertType, message string, severity string) {
	data := map[string]interface{}{
		"alertType": alertType,
		"message":   message,
		"severity":  severity,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := s.hub.BroadcastMessage("system_alert", data); err != nil {
		log.Printf("Failed to broadcast system alert: %v", err)
	}
}

// BroadcastLogUpdate broadcasts a log update to all connected clients
func (s *Service) BroadcastLogUpdate(logEntry *models.LogEntry) {
	data := map[string]interface{}{
		"log": map[string]interface{}{
			"id":        logEntry.ID,
			"timestamp": logEntry.Timestamp.Format(time.RFC3339),
			"level":     logEntry.Level,
			"message":   logEntry.Message,
			"source":    logEntry.Source,
			"metadata":  logEntry.Metadata,
		},
	}

	if err := s.hub.BroadcastMessage("log_update", data); err != nil {
		log.Printf("Failed to broadcast log update: %v", err)
	}
}

// GetConnectionStats returns WebSocket connection statistics
func (s *Service) GetConnectionStats() map[string]interface{} {
	return map[string]interface{}{
		"connectedClients": s.hub.GetClientCount(),
		"hubStatus":        "running",
		"uptime":           time.Now().Format(time.RFC3339),
	}
}
