package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"intelligent-log-analysis/internal/cache"
	"intelligent-log-analysis/internal/config"
	"intelligent-log-analysis/internal/database"
	"intelligent-log-analysis/internal/grpc"
	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/queue"
	"intelligent-log-analysis/internal/repository"
	"intelligent-log-analysis/internal/repository/postgres"
	"intelligent-log-analysis/internal/service"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config       *config.Config
	router       *gin.Engine
	db           *database.DB
	cache        *cache.RedisClient
	aiClient     *grpc.AIServiceClient
	services     *service.Services
	queueService *queue.QueueService
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		router: gin.Default(),
	}
}

// Router returns the gin router for testing purposes
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Start starts the server
func (s *Server) Start() error {
	// Initialize database
	db, err := database.New(&s.config.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	s.db = db

	// Initialize repository layer
	repo := postgres.NewRepository(s.db.DB)

	// Initialize AI client if enabled
	var aiClient *grpc.AIServiceClient
	if s.config.GRPC.AIService.Enabled {
		clientConfig := s.config.GRPC.AIService.ToClientConfig()
		var err error
		aiClient, err = grpc.NewAIServiceClient(clientConfig)
		if err != nil {
			log.Printf("Warning: Failed to initialize AI client: %v", err)
			log.Println("Server will continue without AI analysis")
		} else {
			s.aiClient = aiClient
			log.Println("AI gRPC client connected successfully")
		}
	} else {
		log.Println("AI service is disabled in configuration")
	}

	// Initialize Redis cache
	cache, err := cache.New(&s.config.Redis)
	if err != nil {
		log.Printf("Warning: Failed to initialize Redis cache: %v", err)
		log.Println("Server will continue without cache")
	} else {
		s.cache = cache
		log.Println("Redis cache connected successfully")
	}

	// Initialize queue service if both Redis and AI client are available
	if cache != nil && aiClient != nil {
		// First initialize services to get alert service
		s.services = service.NewServicesWithAlert(repo, aiClient, nil, s.config) // 先不传队列服务

		// Start alert service
		if s.services.Alert != nil {
			if err := s.services.Alert.Start(context.Background()); err != nil {
				log.Printf("Failed to start alert service: %v", err)
			} else {
				log.Println("Alert service started successfully")
			}
		}

		// Now create queue service with alert engine
		var queueService *queue.QueueService
		if s.services.Alert != nil {
			queueService = queue.NewQueueServiceWithAlert(cache, repo, aiClient, s.services.Alert.GetAlertEngine())
		} else {
			queueService = queue.NewQueueService(cache, repo, aiClient)
		}
		s.queueService = queueService

		// Start the queue service
		ctx := context.Background()
		if err := queueService.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start queue service: %v", err)
			log.Println("Server will continue without task queue")
			s.queueService = nil
		} else {
			log.Println("Task queue service started successfully")
		}

		// Update services to include queue service
		if s.queueService != nil {
			s.services = service.NewServicesWithAlert(repo, aiClient, queueService, s.config)
		}
	} else {
		log.Println("Task queue service disabled (requires both Redis and AI service)")
		// Initialize service layer with or without AI client
		if aiClient != nil {
			s.services = service.NewServicesWithAI(repo, aiClient)
		} else {
			s.services = service.NewServices(repo)
		}
	}

	// Setup routes
	s.setupRoutes()

	// Create HTTP server with appropriate timeouts
	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,  // 读取请求超时
		WriteTimeout: 200 * time.Second, // 写入响应超时，给LLM分析足够时间
		IdleTimeout:  120 * time.Second, // 空闲连接超时
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if err := s.db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	// Close Redis connection
	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	// Close AI client connection
	if s.aiClient != nil {
		if err := s.aiClient.Close(); err != nil {
			log.Printf("Error closing AI client: %v", err)
		}
	}

	// Stop queue service
	if s.queueService != nil {
		if err := s.queueService.Stop(); err != nil {
			log.Printf("Error stopping queue service: %v", err)
		}
	}

	log.Println("Server exited")
	return nil
}

// SetupRoutes configures the API routes (public for testing)
func (s *Server) SetupRoutes() {
	s.setupRoutes()
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Add CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		// Check database health
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		dbStatus := "healthy"
		if err := s.db.Health(ctx); err != nil {
			dbStatus = "unhealthy"
		}

		// Check AI service health
		aiStatus := "disabled"
		if s.aiClient != nil {
			if err := s.services.Log.HealthCheckAI(ctx); err != nil {
				aiStatus = "unhealthy"
			} else {
				aiStatus = "healthy"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "healthy",
			"time":       time.Now().Format(time.RFC3339),
			"database":   dbStatus,
			"ai_service": aiStatus,
		})
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Log management endpoints - simplified to single endpoint for filebeat
		v1.POST("/logs", s.handleLogUpload)
		v1.GET("/logs", s.handleGetLogs)
		v1.GET("/logs/analyzed", s.handleGetAnalyzedLogs)
		v1.GET("/analysis/results", s.handleGetAnalysisResults)
		v1.GET("/dashboard/stats", s.handleGetDashboardStats)
		v1.GET("/dashboard/trends", s.handleGetDashboardTrends)

		// AI service endpoints - unified LLM analysis
		v1.GET("/ai/stats", s.handleGetAIStats)
		v1.POST("/ai/analyze", s.handleAnalyzeLogs) // 统一的LLM分析接口

		// Task queue endpoints
		v1.GET("/queue/stats", s.handleGetQueueStats)
		v1.GET("/queue/tasks", s.handleGetTasks)
		v1.GET("/queue/tasks/:id", s.handleGetTask)
		v1.POST("/queue/tasks", s.handleScheduleTask)
		v1.DELETE("/queue/tasks/:id", s.handleDeleteTask)

		// Alert rule endpoints
		v1.POST("/alert-rules", s.handleCreateAlertRule)
		v1.GET("/alert-rules", s.handleGetAlertRules)
		v1.GET("/alert-rules/:id", s.handleGetAlertRule)
		v1.PUT("/alert-rules/:id", s.handleUpdateAlertRule)
		v1.DELETE("/alert-rules/:id", s.handleDeleteAlertRule)
		v1.POST("/alert-rules/:id/test", s.handleTestAlertRule)
		v1.GET("/notification-channels", s.handleGetNotificationChannels)
	}
}

// Request/Response structures

// LogUploadRequest represents the request body for log upload
type LogUploadRequest struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level" binding:"required"`
	Message   string            `json:"message" binding:"required"`
	Source    string            `json:"source" binding:"required"`
	Metadata  map[string]string `json:"metadata"`
}

// BatchLogUploadRequest represents the request body for batch log upload
type BatchLogUploadRequest struct {
	Logs []LogUploadRequest `json:"logs" binding:"required,min=1"`
}

// AnalysisResultsResponse represents the response for analysis results query
type AnalysisResultsResponse struct {
	Results []AnalysisResultWithLog `json:"results"`
	Total   int64                   `json:"total"`
	Page    int                     `json:"page"`
	Limit   int                     `json:"limit"`
}

// AnalysisResultWithLog includes the log entry with analysis result
type AnalysisResultWithLog struct {
	*models.AnalysisResult
	Log *models.LogEntry `json:"log"`
}

// DashboardStatsResponse represents the response for dashboard statistics
type DashboardStatsResponse struct {
	TotalLogs           int64                   `json:"total_logs"`
	AnomalyCount        int64                   `json:"anomaly_count"`
	AnomalyRate         float64                 `json:"anomaly_rate"`
	TopSources          []repository.SourceStat `json:"top_sources"`
	ActiveServices      int                     `json:"active_services"`
	AvgResponseTime     int                     `json:"avg_response_time"`
	AnomalyTrend        []TrendPoint            `json:"anomaly_trend"`
	ServiceDistribution []ServiceDistribution   `json:"service_distribution"`
	LevelDistribution   []LevelDistribution     `json:"level_distribution"`
}

// TrendPoint represents a point in time series data
type TrendPoint struct {
	Time  string `json:"time"`
	Count int64  `json:"count"`
}

// ServiceDistribution represents service distribution data
type ServiceDistribution struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// LevelDistribution represents log level distribution data
type LevelDistribution struct {
	Level string `json:"level"`
	Count int64  `json:"count"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// LogsResponse represents the response for logs query (with analysis status)
type LogsResponse struct {
	Results []LogWithAnalysisStatus `json:"results"`
	Total   int64                   `json:"total"`
	Page    int                     `json:"page"`
	Limit   int                     `json:"limit"`
}

// LogWithAnalysisStatus includes log entry with analysis status
type LogWithAnalysisStatus struct {
	*models.LogEntry
	IsAnalyzed      bool            `json:"is_analyzed"`
	AnalyzedAt      *time.Time      `json:"analyzed_at,omitempty"`
	IsAnomaly       bool            `json:"is_anomaly"`
	AnomalyScore    *float64        `json:"anomaly_score,omitempty"`
	RootCauses      *models.JSONMap `json:"root_causes,omitempty"`
	Recommendations *models.JSONMap `json:"recommendations,omitempty"`
}

// API Handlers

// handleLogUpload handles POST /api/v1/logs - optimized for filebeat nginx logs
func (s *Server) handleLogUpload(c *gin.Context) {
	ctx := c.Request.Context()

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST",
			fmt.Sprintf("Failed to read request body: %v", err))
		return
	}

	// Log request data for debugging
	log.Printf("INFO: handleLogUpload - Request body size: %d bytes, Content-Type: %s, Body: %s",
		len(body), c.GetHeader("Content-Type"), string(body))

	// Try to parse as single log entry (filebeat format)
	var req LogUploadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("ERROR: handleLogUpload - JSON unmarshal failed: %v, Body: %s", err, string(body))
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST",
			fmt.Sprintf("Invalid JSON format: %v", err))
		return
	}

	// Convert to log entry
	logEntry := &models.LogEntry{
		Timestamp: req.Timestamp,
		Level:     strings.ToUpper(req.Level),
		Message:   req.Message,
		Source:    req.Source,
		Metadata:  req.Metadata,
	}

	// Set timestamp if not provided
	if logEntry.Timestamp.IsZero() {
		logEntry.Timestamp = time.Now()
	}

	// Create log entry
	if err := s.services.Log.CreateLog(ctx, logEntry); err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "LOG_CREATION_FAILED",
			fmt.Sprintf("Failed to create log entry: %v", err))
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Log entry created successfully",
		"log_id":  logEntry.ID,
	})
}

// handleGetAnalysisResults handles GET /api/v1/analysis/results
func (s *Server) handleGetAnalysisResults(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetAnalysisResults - Query params: %s", c.Request.URL.RawQuery)

	// Parse query parameters
	filter, page, limit, err := s.parseAnalysisResultsQuery(c)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	// Set pagination
	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	// Get analysis results
	results, err := s.services.Analysis.GetAnalysisResults(ctx, filter)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "QUERY_FAILED",
			fmt.Sprintf("Failed to query analysis results: %v", err))
		return
	}

	// Get total count
	total, err := s.services.Analysis.CountAnalysisResults(ctx, filter)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "COUNT_FAILED",
			fmt.Sprintf("Failed to count analysis results: %v", err))
		return
	}

	// Convert to response format with log details
	responseResults := make([]AnalysisResultWithLog, len(results))
	for i, result := range results {
		// Get associated log entry
		logEntry, err := s.services.Log.GetLog(ctx, result.LogID)
		if err != nil {
			log.Printf("Warning: Failed to get log for analysis result %s: %v", result.ID, err)
		}

		responseResults[i] = AnalysisResultWithLog{
			AnalysisResult: result,
			Log:            logEntry,
		}
	}

	// Return response
	response := AnalysisResultsResponse{
		Results: responseResults,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}

	c.JSON(http.StatusOK, response)
}

// handleGetDashboardStats handles GET /api/v1/dashboard/stats
func (s *Server) handleGetDashboardStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetDashboardStats - Query params: %s", c.Request.URL.RawQuery)

	// Parse period parameter
	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_PERIOD",
			fmt.Sprintf("Invalid period format: %s", periodStr))
		return
	}

	// Get anomaly statistics
	stats, err := s.services.Analysis.GetAnomalyStats(ctx, period)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "STATS_QUERY_FAILED",
			fmt.Sprintf("Failed to get anomaly statistics: %v", err))
		return
	}

	// Generate trend data (simplified - in real implementation this would come from database)
	trendData := s.generateTrendData(ctx, period)
	serviceDistribution := s.generateServiceDistribution(ctx, period)
	levelDistribution := s.generateLevelDistribution(ctx, period)

	// Build response
	response := DashboardStatsResponse{
		TotalLogs:           stats.TotalLogs,
		AnomalyCount:        stats.AnomalyCount,
		AnomalyRate:         stats.AnomalyRate,
		TopSources:          stats.TopSources,
		ActiveServices:      stats.ActiveServices,
		AvgResponseTime:     245, // Placeholder - would be calculated from actual metrics
		AnomalyTrend:        trendData,
		ServiceDistribution: serviceDistribution,
		LevelDistribution:   levelDistribution,
	}

	c.JSON(http.StatusOK, response)
}

// handleGetDashboardTrends handles GET /api/v1/dashboard/trends
func (s *Server) handleGetDashboardTrends(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetDashboardTrends - Query params: %s", c.Request.URL.RawQuery)

	// Parse period parameter
	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_PERIOD",
			fmt.Sprintf("Invalid period format: %s", periodStr))
		return
	}

	// Generate trend data for the specified period
	trendData := s.generateTrendData(ctx, period)

	// Return trend data in the format expected by the frontend
	response := gin.H{
		"labels": make([]string, len(trendData)),
		"datasets": []gin.H{
			{
				"label":           "异常检测",
				"data":            make([]int64, len(trendData)),
				"borderColor":     "#f56c6c",
				"backgroundColor": "rgba(245, 108, 108, 0.1)",
			},
		},
	}

	// Fill in the data
	for i, point := range trendData {
		response["labels"].([]string)[i] = point.Time
		response["datasets"].([]gin.H)[0]["data"].([]int64)[i] = point.Count
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

// parseAnalysisResultsQuery parses query parameters for analysis results
func (s *Server) parseAnalysisResultsQuery(c *gin.Context) (repository.ResultFilter, int, int, error) {
	var filter repository.ResultFilter

	// Parse period parameter (convert to start_time and end_time)
	if periodStr := c.Query("period"); periodStr != "" {
		period, err := time.ParseDuration(periodStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid period format: %v", err)
		}
		now := time.Now()
		startTime := now.Add(-period)
		filter.StartTime = &startTime
		filter.EndTime = &now
	}

	// Parse time range (explicit start_time and end_time override period)
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid start_time format: %v", err)
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid end_time format: %v", err)
		}
		filter.EndTime = &endTime
	}

	// Parse anomaly filter
	if anomalyStr := c.Query("anomaly"); anomalyStr != "" {
		isAnomaly, err := strconv.ParseBool(anomalyStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid anomaly format: %v", err)
		}
		filter.IsAnomaly = &isAnomaly
	}

	// Parse anomaly_only filter (for frontend compatibility)
	if anomalyOnlyStr := c.Query("anomaly_only"); anomalyOnlyStr != "" {
		isAnomaly, err := strconv.ParseBool(anomalyOnlyStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid anomaly_only format: %v", err)
		}
		if isAnomaly {
			filter.IsAnomaly = &isAnomaly
		}
	}

	// Parse score range
	if minScoreStr := c.Query("min_score"); minScoreStr != "" {
		minScore, err := strconv.ParseFloat(minScoreStr, 64)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid min_score format: %v", err)
		}
		filter.MinScore = &minScore
	}

	if maxScoreStr := c.Query("max_score"); maxScoreStr != "" {
		maxScore, err := strconv.ParseFloat(maxScoreStr, 64)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid max_score format: %v", err)
		}
		filter.MaxScore = &maxScore
	}

	// Parse source filter
	if source := c.Query("source"); source != "" {
		filter.Source = source
	}

	// Parse pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return filter, 0, 0, fmt.Errorf("invalid page number: %s", pageStr)
		}
	}

	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			return filter, 0, 0, fmt.Errorf("invalid limit (must be 1-1000): %s", limitStr)
		}
	}

	return filter, page, limit, nil
}

// generateTrendData generates trend data for the dashboard (placeholder implementation)
func (s *Server) generateTrendData(ctx context.Context, period time.Duration) []TrendPoint {
	// This is a simplified implementation
	// In a real system, this would query the database for time-series data
	points := make([]TrendPoint, 0)

	now := time.Now()
	intervals := 24 // 24 data points for the period
	intervalDuration := period / time.Duration(intervals)

	for i := 0; i < intervals; i++ {
		pointTime := now.Add(-period + time.Duration(i)*intervalDuration)
		points = append(points, TrendPoint{
			Time:  pointTime.Format(time.RFC3339),
			Count: int64(i % 10), // Placeholder data
		})
	}

	return points
}

// generateServiceDistribution generates service distribution data (placeholder implementation)
func (s *Server) generateServiceDistribution(ctx context.Context, period time.Duration) []ServiceDistribution {
	// This would typically query the database for actual service distribution
	return []ServiceDistribution{
		{Name: "user-service", Value: 45},
		{Name: "payment-service", Value: 32},
		{Name: "order-service", Value: 23},
	}
}

// generateLevelDistribution generates log level distribution data (placeholder implementation)
func (s *Server) generateLevelDistribution(ctx context.Context, period time.Duration) []LevelDistribution {
	// This would typically query the database for actual level distribution
	return []LevelDistribution{
		{Level: "ERROR", Count: 89},
		{Level: "WARN", Count: 156},
		{Level: "INFO", Count: 1234},
		{Level: "DEBUG", Count: 567},
	}
}

// sendErrorResponse sends a standardized error response
func (s *Server) sendErrorResponse(c *gin.Context, statusCode int, errorCode, message string) {
	response := ErrorResponse{
		Error:   errorCode,
		Message: message,
		Code:    statusCode,
	}
	c.JSON(statusCode, response)
}

// handleGetAIStats handles GET /api/v1/ai/stats
func (s *Server) handleGetAIStats(c *gin.Context) {
	// Log request data for debugging
	log.Printf("INFO: handleGetAIStats - Request from: %s", c.ClientIP())

	stats := s.services.Log.GetAIStats()
	c.JSON(http.StatusOK, stats)
}

// handleAnalyzeLogs handles POST /api/v1/ai/analyze - 统一的LLM分析接口
func (s *Server) handleAnalyzeLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse request body
	var req struct {
		LogIDs []string `json:"log_ids" binding:"required,min=1,max=20"` // 限制最多20条日志
		Stream bool     `json:"stream,omitempty"`                        // 是否使用流式响应
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST",
			fmt.Sprintf("Invalid request format: %v", err))
		return
	}

	// Log request data for debugging
	log.Printf("INFO: handleAnalyzeLogs - Log IDs count: %d, IDs: %v, Stream: %v", len(req.LogIDs), req.LogIDs, req.Stream)

	// Check if AI client is available
	if s.aiClient == nil {
		s.sendErrorResponse(c, http.StatusServiceUnavailable, "AI_SERVICE_NOT_AVAILABLE",
			"AI service is not available")
		return
	}

	// Get logs from database
	logs := make([]*models.LogEntry, 0, len(req.LogIDs))
	for _, logID := range req.LogIDs {
		logEntry, err := s.services.Log.GetLog(ctx, logID)
		if err != nil {
			s.sendErrorResponse(c, http.StatusNotFound, "LOG_NOT_FOUND",
				fmt.Sprintf("Log with ID %s not found: %v", logID, err))
			return
		}
		logs = append(logs, logEntry)
	}

	// 如果请求流式响应，使用SSE
	if req.Stream {
		s.handleAnalyzeLogsStream(c, ctx, logs)
		return
	}

	// 调用AI服务进行LLM分析（统一使用LLM）
	results, err := s.services.Log.AnalyzeLogsSync(ctx, logs)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "LLM_ANALYSIS_FAILED",
			fmt.Sprintf("LLM analysis failed: %v", err))
		return
	}

	// Return analysis results
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       fmt.Sprintf("Successfully performed LLM analysis on %d logs", len(logs)),
		"results":       results,
		"analysis_type": "pure_llm_analysis",
	})
}

// handleAnalyzeLogsStream handles streaming LLM analysis with Server-Sent Events
func (s *Server) handleAnalyzeLogsStream(c *gin.Context, ctx context.Context, logs []*models.LogEntry) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Send initial status
	s.sendSSEEvent(c, "status", map[string]interface{}{
		"message": "开始LLM深度分析...",
		"stage":   "initializing",
		"total":   len(logs),
	})

	// Send progress update
	s.sendSSEEvent(c, "progress", map[string]interface{}{
		"message":   "正在调用大模型API...",
		"stage":     "analyzing",
		"completed": 0,
		"total":     len(logs),
	})

	// 调用AI服务进行LLM分析
	results, err := s.services.Log.AnalyzeLogsSync(ctx, logs)
	if err != nil {
		s.sendSSEEvent(c, "error", map[string]interface{}{
			"message": fmt.Sprintf("LLM analysis failed: %v", err),
			"stage":   "failed",
		})
		return
	}

	// Send analysis progress
	s.sendSSEEvent(c, "progress", map[string]interface{}{
		"message":   "分析完成，结果已保存",
		"stage":     "completed",
		"completed": len(logs),
		"total":     len(logs),
	})

	// Send final results
	s.sendSSEEvent(c, "result", map[string]interface{}{
		"success":       true,
		"message":       fmt.Sprintf("Successfully performed LLM analysis on %d logs", len(logs)),
		"results":       results,
		"analysis_type": "pure_llm_analysis",
		"stage":         "completed",
	})

	// Send completion event
	s.sendSSEEvent(c, "done", map[string]interface{}{
		"message": "分析完成",
		"stage":   "done",
	})
}

// sendSSEEvent sends a Server-Sent Event
func (s *Server) sendSSEEvent(c *gin.Context, eventType string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, jsonData)
	c.Writer.Flush()
}

// Queue management handlers

// handleGetQueueStats handles GET /api/v1/queue/stats
func (s *Server) handleGetQueueStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetQueueStats - Request from: %s", c.ClientIP())

	if s.queueService == nil {
		s.sendErrorResponse(c, http.StatusServiceUnavailable, "QUEUE_NOT_AVAILABLE",
			"Task queue service is not available")
		return
	}

	// Get queue statistics
	queueStats, err := s.services.Log.GetQueueStats(ctx)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "QUEUE_STATS_FAILED",
			fmt.Sprintf("Failed to get queue statistics: %v", err))
		return
	}

	// Get scheduler statistics
	schedulerStats := s.services.Log.GetSchedulerStats()

	// Combine statistics
	response := gin.H{
		"queue":     queueStats,
		"scheduler": schedulerStats,
	}

	c.JSON(http.StatusOK, response)
}

// handleGetTasks handles GET /api/v1/queue/tasks
func (s *Server) handleGetTasks(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetTasks - Query params: %s", c.Request.URL.RawQuery)

	if s.queueService == nil {
		s.sendErrorResponse(c, http.StatusServiceUnavailable, "QUEUE_NOT_AVAILABLE",
			"Task queue service is not available")
		return
	}

	// Parse query parameters
	filter, page, limit, err := s.parseTasksQuery(c)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	// Set pagination
	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	// Get tasks
	tasks, err := s.services.Log.GetTasks(ctx, filter)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "TASKS_QUERY_FAILED",
			fmt.Sprintf("Failed to query tasks: %v", err))
		return
	}

	// Return response
	response := gin.H{
		"tasks": tasks,
		"page":  page,
		"limit": limit,
	}

	c.JSON(http.StatusOK, response)
}

// handleGetTask handles GET /api/v1/queue/tasks/:id
func (s *Server) handleGetTask(c *gin.Context) {
	ctx := c.Request.Context()

	if s.queueService == nil {
		s.sendErrorResponse(c, http.StatusServiceUnavailable, "QUEUE_NOT_AVAILABLE",
			"Task queue service is not available")
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_TASK_ID",
			"Task ID is required")
		return
	}

	// Get task
	task, err := s.services.Log.GetTask(ctx, taskID)
	if err != nil {
		s.sendErrorResponse(c, http.StatusNotFound, "TASK_NOT_FOUND",
			fmt.Sprintf("Task not found: %v", err))
		return
	}

	c.JSON(http.StatusOK, task)
}

// handleScheduleTask handles POST /api/v1/queue/tasks
func (s *Server) handleScheduleTask(c *gin.Context) {
	ctx := c.Request.Context()

	if s.queueService == nil {
		s.sendErrorResponse(c, http.StatusServiceUnavailable, "QUEUE_NOT_AVAILABLE",
			"Task queue service is not available")
		return
	}

	// Parse request body
	var req struct {
		Type    string                 `json:"type" binding:"required"`
		LogIDs  []string               `json:"log_ids"`
		Payload map[string]interface{} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST",
			fmt.Sprintf("Invalid request format: %v", err))
		return
	}

	// Validate task type
	if req.Type != string(queue.TaskTypeLogAnalysis) {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_TASK_TYPE",
			fmt.Sprintf("Unsupported task type: %s", req.Type))
		return
	}

	// For log analysis tasks, use log_ids
	var logIDs []string
	if req.Type == string(queue.TaskTypeLogAnalysis) {
		if len(req.LogIDs) == 0 {
			s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_LOG_IDS",
				"Log IDs are required for log analysis tasks")
			return
		}
		logIDs = req.LogIDs
	}

	// Schedule the task
	task, err := s.queueService.ScheduleLogAnalysis(ctx, logIDs)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "TASK_SCHEDULING_FAILED",
			fmt.Sprintf("Failed to schedule task: %v", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Task scheduled successfully",
		"task":    task,
	})
}

// handleDeleteTask handles DELETE /api/v1/queue/tasks/:id
func (s *Server) handleDeleteTask(c *gin.Context) {
	ctx := c.Request.Context()

	if s.queueService == nil {
		s.sendErrorResponse(c, http.StatusServiceUnavailable, "QUEUE_NOT_AVAILABLE",
			"Task queue service is not available")
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_TASK_ID",
			"Task ID is required")
		return
	}

	// Delete task
	if err := s.queueService.DeleteTask(ctx, taskID); err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "TASK_DELETION_FAILED",
			fmt.Sprintf("Failed to delete task: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Task deleted successfully",
	})
}

// parseTasksQuery parses query parameters for tasks
func (s *Server) parseTasksQuery(c *gin.Context) (queue.TaskFilter, int, int, error) {
	var filter queue.TaskFilter

	// Parse task status
	if statusStr := c.Query("status"); statusStr != "" {
		status := queue.TaskStatus(statusStr)
		filter.Status = &status
	}

	// Parse task type
	if typeStr := c.Query("type"); typeStr != "" {
		taskType := queue.TaskType(typeStr)
		filter.Type = &taskType
	}

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid start_time format: %v", err)
		}
		filter.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return filter, 0, 0, fmt.Errorf("invalid end_time format: %v", err)
		}
		filter.EndTime = &endTime
	}

	// Parse pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return filter, 0, 0, fmt.Errorf("invalid page number: %s", pageStr)
		}
	}

	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			return filter, 0, 0, fmt.Errorf("invalid limit (must be 1-1000): %s", limitStr)
		}
	}

	return filter, page, limit, nil
}

// Alert Rule Handlers

// AlertRuleRequest represents the request body for creating/updating alert rules
type AlertRuleRequest struct {
	Name                 string                 `json:"name" binding:"required"`
	Condition            map[string]interface{} `json:"condition" binding:"required"`
	NotificationChannels []string               `json:"notification_channels" binding:"required,min=1"`
	IsActive             *bool                  `json:"is_active,omitempty"`
}

// handleCreateAlertRule handles POST /api/v1/alert-rules
func (s *Server) handleCreateAlertRule(c *gin.Context) {
	ctx := c.Request.Context()

	var req AlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST",
			fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Create alert rule model
	rule := &models.AlertRule{
		Name:                 req.Name,
		Condition:            req.Condition,
		NotificationChannels: req.NotificationChannels,
		IsActive:             true, // Default to active
	}

	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	// Create alert rule
	if err := s.services.Alert.CreateAlertRule(ctx, rule); err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "ALERT_RULE_CREATION_FAILED",
			fmt.Sprintf("Failed to create alert rule: %v", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Alert rule created successfully",
		"rule":    rule,
	})
}

// handleGetAlertRules handles GET /api/v1/alert-rules
func (s *Server) handleGetAlertRules(c *gin.Context) {
	ctx := c.Request.Context()

	// Get active filter
	activeOnly := c.Query("active") == "true"

	var rules []*models.AlertRule
	var err error

	if activeOnly {
		rules, err = s.services.Alert.GetActiveAlertRules(ctx)
	} else {
		// For now, we only have GetActiveAlertRules, so we'll use that
		// In a full implementation, you'd add GetAllAlertRules to the repository
		rules, err = s.services.Alert.GetActiveAlertRules(ctx)
	}

	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "ALERT_RULES_FETCH_FAILED",
			fmt.Sprintf("Failed to fetch alert rules: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"rules":   rules,
		"total":   len(rules),
	})
}

// handleGetAlertRule handles GET /api/v1/alert-rules/:id
func (s *Server) handleGetAlertRule(c *gin.Context) {
	ctx := c.Request.Context()

	ruleID := c.Param("id")
	if ruleID == "" {
		s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_RULE_ID",
			"Alert rule ID is required")
		return
	}

	rule, err := s.services.Alert.GetAlertRule(ctx, ruleID)
	if err != nil {
		s.sendErrorResponse(c, http.StatusNotFound, "ALERT_RULE_NOT_FOUND",
			fmt.Sprintf("Alert rule not found: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"rule":    rule,
	})
}

// handleUpdateAlertRule handles PUT /api/v1/alert-rules/:id
func (s *Server) handleUpdateAlertRule(c *gin.Context) {
	ctx := c.Request.Context()

	ruleID := c.Param("id")
	if ruleID == "" {
		s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_RULE_ID",
			"Alert rule ID is required")
		return
	}

	var req AlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST",
			fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Get existing rule
	existingRule, err := s.services.Alert.GetAlertRule(ctx, ruleID)
	if err != nil {
		s.sendErrorResponse(c, http.StatusNotFound, "ALERT_RULE_NOT_FOUND",
			fmt.Sprintf("Alert rule not found: %v", err))
		return
	}

	// Update rule fields
	existingRule.Name = req.Name
	existingRule.Condition = req.Condition
	existingRule.NotificationChannels = req.NotificationChannels

	if req.IsActive != nil {
		existingRule.IsActive = *req.IsActive
	}

	// Update alert rule
	if err := s.services.Alert.UpdateAlertRule(ctx, existingRule); err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "ALERT_RULE_UPDATE_FAILED",
			fmt.Sprintf("Failed to update alert rule: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert rule updated successfully",
		"rule":    existingRule,
	})
}

// handleDeleteAlertRule handles DELETE /api/v1/alert-rules/:id
func (s *Server) handleDeleteAlertRule(c *gin.Context) {
	ctx := c.Request.Context()

	ruleID := c.Param("id")
	if ruleID == "" {
		s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_RULE_ID",
			"Alert rule ID is required")
		return
	}

	// Delete alert rule
	if err := s.services.Alert.DeleteAlertRule(ctx, ruleID); err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "ALERT_RULE_DELETION_FAILED",
			fmt.Sprintf("Failed to delete alert rule: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert rule deleted successfully",
	})
}

// handleTestAlertRule handles POST /api/v1/alert-rules/:id/test
func (s *Server) handleTestAlertRule(c *gin.Context) {
	ctx := c.Request.Context()

	ruleID := c.Param("id")
	if ruleID == "" {
		s.sendErrorResponse(c, http.StatusBadRequest, "MISSING_RULE_ID",
			"Alert rule ID is required")
		return
	}

	// Get alert rule
	rule, err := s.services.Alert.GetAlertRule(ctx, ruleID)
	if err != nil {
		s.sendErrorResponse(c, http.StatusNotFound, "ALERT_RULE_NOT_FOUND",
			fmt.Sprintf("Alert rule not found: %v", err))
		return
	}

	// Test the alert rule
	results, err := s.services.Alert.TestAlertRule(ctx, rule)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "ALERT_TEST_FAILED",
			fmt.Sprintf("Failed to test alert rule: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test alert sent",
		"results": results,
	})
}

// handleGetNotificationChannels handles GET /api/v1/notification-channels
func (s *Server) handleGetNotificationChannels(c *gin.Context) {
	channels := []map[string]interface{}{
		{
			"name":         "email",
			"display_name": "邮件通知",
			"description":  "通过邮件发送告警通知",
			"icon":         "📧",
		},
		{
			"name":         "dingtalk",
			"display_name": "钉钉通知",
			"description":  "通过钉钉机器人发送告警通知",
			"icon":         "💬",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"channels": channels,
	})
}

// handleGetLogs handles GET /api/v1/logs - returns all logs with analysis status
func (s *Server) handleGetLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetLogs - Query params: %s", c.Request.URL.RawQuery)

	// Parse query parameters
	filter, page, limit, err := s.parseLogsQuery(c)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	// Set pagination
	filter.Limit = limit
	filter.Offset = (page - 1) * limit

	// Get logs
	logs, err := s.services.Log.GetLogs(ctx, filter)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "QUERY_FAILED",
			fmt.Sprintf("Failed to query logs: %v", err))
		return
	}

	// Get total count
	total, err := s.services.Log.CountLogs(ctx, filter)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "COUNT_FAILED",
			fmt.Sprintf("Failed to count logs: %v", err))
		return
	}

	// Convert to response format with analysis status
	responseResults := make([]LogWithAnalysisStatus, len(logs))
	for i, logEntry := range logs {
		// Check if log has analysis result
		analysisResult, err := s.services.Analysis.GetAnalysisResultByLogID(ctx, logEntry.ID)

		logWithStatus := LogWithAnalysisStatus{
			LogEntry:   logEntry,
			IsAnalyzed: err == nil && analysisResult != nil,
		}

		if analysisResult != nil {
			logWithStatus.AnalyzedAt = &analysisResult.AnalyzedAt
			logWithStatus.IsAnomaly = analysisResult.IsAnomaly
			logWithStatus.AnomalyScore = &analysisResult.AnomalyScore
			logWithStatus.RootCauses = &analysisResult.RootCauses
			logWithStatus.Recommendations = &analysisResult.Recommendations
		}

		responseResults[i] = logWithStatus
	}

	// Return response
	response := LogsResponse{
		Results: responseResults,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}

	c.JSON(http.StatusOK, response)
}

// handleGetAnalyzedLogs handles GET /api/v1/logs/analyzed - returns only analyzed logs for dashboard
func (s *Server) handleGetAnalyzedLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Log request data for debugging
	log.Printf("INFO: handleGetAnalyzedLogs - Query params: %s", c.Request.URL.RawQuery)

	// Parse query parameters for analysis results (only analyzed logs)
	filter, _, limit, err := s.parseAnalysisResultsQuery(c)
	if err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
	}

	// For dashboard, we don't need pagination, limit to recent logs
	if limit > 100 {
		limit = 100 // Dashboard should show max 100 recent analyzed logs
	}
	filter.Limit = limit
	filter.Offset = 0 // No pagination for dashboard

	// Get analysis results (which are analyzed logs)
	results, err := s.services.Analysis.GetAnalysisResults(ctx, filter)
	if err != nil {
		s.sendErrorResponse(c, http.StatusInternalServerError, "QUERY_FAILED",
			fmt.Sprintf("Failed to query analyzed logs: %v", err))
		return
	}

	// Convert to response format
	responseResults := make([]LogWithAnalysisStatus, len(results))
	for i, result := range results {
		// Get associated log entry
		logEntry, err := s.services.Log.GetLog(ctx, result.LogID)
		if err != nil {
			log.Printf("Warning: Failed to get log for analysis result %s: %v", result.ID, err)
			continue
		}

		responseResults[i] = LogWithAnalysisStatus{
			LogEntry:        logEntry,
			IsAnalyzed:      true,
			AnalyzedAt:      &result.AnalyzedAt,
			IsAnomaly:       result.IsAnomaly,
			AnomalyScore:    &result.AnomalyScore,
			RootCauses:      &result.RootCauses,
			Recommendations: &result.Recommendations,
		}
	}

	// Return response (no pagination for dashboard)
	c.JSON(http.StatusOK, gin.H{
		"results": responseResults,
		"total":   len(responseResults),
	})
}

// parseLogsQuery parses query parameters for logs
func (s *Server) parseLogsQuery(c *gin.Context) (repository.LogFilter, int, int, error) {
	var filter repository.LogFilter

	// Parse time range
	timeRange := c.DefaultQuery("time_range", "24h")
	switch timeRange {
	case "1h":
		startTime := time.Now().Add(-1 * time.Hour)
		filter.StartTime = &startTime
	case "6h":
		startTime := time.Now().Add(-6 * time.Hour)
		filter.StartTime = &startTime
	case "24h":
		startTime := time.Now().Add(-24 * time.Hour)
		filter.StartTime = &startTime
	case "7d":
		startTime := time.Now().Add(-7 * 24 * time.Hour)
		filter.StartTime = &startTime
	case "30d":
		startTime := time.Now().Add(-30 * 24 * time.Hour)
		filter.StartTime = &startTime
	case "all":
		// No time filter
	default:
		return filter, 0, 0, fmt.Errorf("invalid time_range: %s", timeRange)
	}

	// Parse log level - 移除level过滤
	// if level := c.Query("level"); level != "" && level != "all" {
	// 	filter.Level = level
	// }

	// Parse source
	if source := c.Query("source"); source != "" && source != "all" {
		filter.Source = source
	}

	// Parse analysis status filter
	if analyzedStr := c.Query("analyzed"); analyzedStr != "" {
		analyzed := analyzedStr == "true"
		filter.AnalyzedOnly = &analyzed
	}

	// Parse anomaly filter
	if anomalyStr := c.Query("anomaly_only"); anomalyStr == "true" {
		anomaly := true
		filter.AnomalyOnly = &anomaly
	}

	// Parse pagination
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return filter, 0, 0, fmt.Errorf("invalid page number: %s", pageStr)
		}
	}

	limit := 20 // Default limit for logs list
	if limitStr := c.Query("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			return filter, 0, 0, fmt.Errorf("invalid limit (must be 1-1000): %s", limitStr)
		}
	}

	return filter, page, limit, nil
}
