package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestServer_HandleLogUpload_RequestValidation(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test server
	server := &Server{
		router: gin.New(),
	}

	// Setup routes
	server.router.POST("/api/v1/logs", server.handleLogUpload)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Invalid log upload - missing required fields",
			requestBody: LogUploadRequest{
				Message: "Test message without level",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
		{
			name:           "Invalid JSON format",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			var jsonBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				jsonBody = []byte(str)
			} else {
				jsonBody, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			// Create HTTP request
			req, err := http.NewRequest("POST", "/api/v1/logs", bytes.NewBuffer(jsonBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			server.router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check for expected error
			if tt.expectedError != "" {
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

func TestServer_HandleGetAnalysisResults_QueryValidation(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test server
	server := &Server{
		router: gin.New(),
	}

	// Setup routes
	server.router.GET("/api/v1/analysis/results", server.handleGetAnalysisResults)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid time format",
			queryParams:    "?start_time=invalid-time",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_QUERY_PARAMS",
		},
		{
			name:           "Invalid page number",
			queryParams:    "?page=0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_QUERY_PARAMS",
		},
		{
			name:           "Invalid limit",
			queryParams:    "?limit=2000",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_QUERY_PARAMS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			req, err := http.NewRequest("GET", "/api/v1/analysis/results"+tt.queryParams, nil)
			assert.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			server.router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check for expected error
			if tt.expectedError != "" {
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

func TestServer_HandleGetDashboardStats_QueryValidation(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test server
	server := &Server{
		router: gin.New(),
	}

	// Setup routes
	server.router.GET("/api/v1/dashboard/stats", server.handleGetDashboardStats)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid period format",
			queryParams:    "?period=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_PERIOD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			req, err := http.NewRequest("GET", "/api/v1/dashboard/stats"+tt.queryParams, nil)
			assert.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			server.router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check for expected error
			if tt.expectedError != "" {
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

func TestServer_SendErrorResponse(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test server
	server := &Server{}

	// Create a test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test error response
	server.sendErrorResponse(c, http.StatusBadRequest, "TEST_ERROR", "Test error message")

	// Check status code
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Parse response
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response fields
	assert.Equal(t, "TEST_ERROR", response.Error)
	assert.Equal(t, "Test error message", response.Message)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestParseAnalysisResultsQuery(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	server := &Server{}

	tests := []struct {
		name        string
		queryParams map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid empty query",
			queryParams: map[string]string{},
			expectError: false,
		},
		{
			name: "Valid time range",
			queryParams: map[string]string{
				"start_time": "2024-01-01T00:00:00Z",
				"end_time":   "2024-01-02T00:00:00Z",
			},
			expectError: false,
		},
		{
			name: "Valid pagination",
			queryParams: map[string]string{
				"page":  "1",
				"limit": "10",
			},
			expectError: false,
		},
		{
			name: "Invalid start time",
			queryParams: map[string]string{
				"start_time": "invalid",
			},
			expectError: true,
			errorMsg:    "invalid start_time format",
		},
		{
			name: "Invalid page",
			queryParams: map[string]string{
				"page": "0",
			},
			expectError: true,
			errorMsg:    "invalid page number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test context with query parameters
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set query parameters
			req := &http.Request{URL: &url.URL{}}
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			// Call the function
			_, _, _, err := server.parseAnalysisResultsQuery(c)

			// Check results
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
