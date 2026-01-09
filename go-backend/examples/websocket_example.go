//go:build examples
// +build examples

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"intelligent-log-analysis/internal/models"
	"intelligent-log-analysis/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create WebSocket service
	wsService := websocket.NewService()

	// Start WebSocket service
	if err := wsService.Start(context.Background()); err != nil {
		log.Fatal("Failed to start WebSocket service:", err)
	}
	defer wsService.Stop()

	// Create Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		websocket.ServeWS(wsService.GetHub(), c.Writer, c.Request)
	})

	// Test endpoint to simulate log creation
	router.POST("/test/log", func(c *gin.Context) {
		// Create a test log entry
		logEntry := &models.LogEntry{
			ID:        fmt.Sprintf("log-%d", time.Now().Unix()),
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Test error message from WebSocket example",
			Source:    "websocket-example",
			Metadata:  map[string]string{"test": "true"},
		}

		// Broadcast log update
		wsService.BroadcastLogUpdate(logEntry)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Log broadcasted via WebSocket",
			"log":     logEntry,
		})
	})

	// Test endpoint to simulate anomaly detection
	router.POST("/test/anomaly", func(c *gin.Context) {
		// Create a test log entry
		logEntry := &models.LogEntry{
			ID:        fmt.Sprintf("anomaly-log-%d", time.Now().Unix()),
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   "Critical database connection failure detected",
			Source:    "database-service",
			Metadata:  map[string]string{"severity": "critical"},
		}

		// Create analysis result
		analysisResult := &models.AnalysisResult{
			ID:              fmt.Sprintf("analysis-%d", time.Now().Unix()),
			LogID:           logEntry.ID,
			IsAnomaly:       true,
			AnomalyScore:    0.95,
			RootCauses:      []string{"Database connection timeout", "Network connectivity issue"},
			Recommendations: []string{"Check database server status", "Verify network connectivity", "Review connection pool settings"},
			AnalyzedAt:      time.Now(),
		}

		// Broadcast anomaly
		wsService.BroadcastNewAnomaly(logEntry, analysisResult)

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "Anomaly broadcasted via WebSocket",
			"log":      logEntry,
			"analysis": analysisResult,
		})
	})

	// Test endpoint to simulate stats update
	router.POST("/test/stats", func(c *gin.Context) {
		// Create test stats
		stats := map[string]interface{}{
			"totalLogs":      1000 + time.Now().Unix()%1000,
			"anomalyCount":   50 + time.Now().Unix()%50,
			"anomalyRate":    0.05 + float64(time.Now().Unix()%10)/1000,
			"activeServices": 5,
		}

		// Broadcast stats update
		wsService.BroadcastStatsUpdate(stats)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Stats update broadcasted via WebSocket",
			"stats":   stats,
		})
	})

	// WebSocket stats endpoint
	router.GET("/ws/stats", func(c *gin.Context) {
		stats := wsService.GetConnectionStats()
		c.JSON(http.StatusOK, stats)
	})

	// Serve static HTML for testing
	router.GET("/", func(c *gin.Context) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Test</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        .status { padding: 10px; margin: 10px 0; border-radius: 5px; }
        .connected { background-color: #d4edda; color: #155724; }
        .disconnected { background-color: #f8d7da; color: #721c24; }
        .message { background-color: #f8f9fa; padding: 10px; margin: 5px 0; border-left: 3px solid #007bff; }
        .anomaly { border-left-color: #dc3545; }
        button { padding: 10px 15px; margin: 5px; background-color: #007bff; color: white; border: none; border-radius: 3px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        #messages { max-height: 400px; overflow-y: auto; border: 1px solid #ddd; padding: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>WebSocket Real-time Push Test</h1>
        
        <div id="status" class="status disconnected">Disconnected</div>
        
        <div>
            <button onclick="connect()">Connect</button>
            <button onclick="disconnect()">Disconnect</button>
            <button onclick="testLog()">Test Log</button>
            <button onclick="testAnomaly()">Test Anomaly</button>
            <button onclick="testStats()">Test Stats</button>
        </div>
        
        <h3>Messages:</h3>
        <div id="messages"></div>
    </div>

    <script>
        let ws = null;
        const statusDiv = document.getElementById('status');
        const messagesDiv = document.getElementById('messages');

        function updateStatus(connected) {
            if (connected) {
                statusDiv.textContent = 'Connected';
                statusDiv.className = 'status connected';
            } else {
                statusDiv.textContent = 'Disconnected';
                statusDiv.className = 'status disconnected';
            }
        }

        function addMessage(message, isAnomaly = false) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message' + (isAnomaly ? ' anomaly' : '');
            messageDiv.innerHTML = '<strong>' + new Date().toLocaleTimeString() + '</strong>: ' + JSON.stringify(message, null, 2);
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function connect() {
            if (ws) {
                ws.close();
            }

            ws = new WebSocket('ws://localhost:8080/ws');
            
            ws.onopen = function() {
                updateStatus(true);
                addMessage({type: 'connection', status: 'Connected to WebSocket'});
            };

            ws.onmessage = function(event) {
                try {
                    const message = JSON.parse(event.data);
                    addMessage(message, message.type === 'new_anomaly');
                } catch (e) {
                    addMessage({type: 'error', message: 'Failed to parse message: ' + event.data});
                }
            };

            ws.onclose = function() {
                updateStatus(false);
                addMessage({type: 'connection', status: 'Disconnected from WebSocket'});
            };

            ws.onerror = function(error) {
                addMessage({type: 'error', message: 'WebSocket error: ' + error});
            };
        }

        function disconnect() {
            if (ws) {
                ws.close();
                ws = null;
            }
        }

        function testLog() {
            fetch('/test/log', {method: 'POST'})
                .then(response => response.json())
                .then(data => addMessage({type: 'api_response', data: data}))
                .catch(error => addMessage({type: 'error', message: 'API error: ' + error}));
        }

        function testAnomaly() {
            fetch('/test/anomaly', {method: 'POST'})
                .then(response => response.json())
                .then(data => addMessage({type: 'api_response', data: data}))
                .catch(error => addMessage({type: 'error', message: 'API error: ' + error}));
        }

        function testStats() {
            fetch('/test/stats', {method: 'POST'})
                .then(response => response.json())
                .then(data => addMessage({type: 'api_response', data: data}))
                .catch(error => addMessage({type: 'error', message: 'API error: ' + error}));
        }

        // Auto-connect on page load
        window.onload = function() {
            connect();
        };
    </script>
</body>
</html>
        `
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	})

	// Start server
	fmt.Println("WebSocket example server starting on :8080")
	fmt.Println("Open http://localhost:8080 in your browser to test WebSocket functionality")

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
