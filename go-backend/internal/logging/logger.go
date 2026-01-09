package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Operation string                 `json:"operation,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
}

// Logger provides structured logging capabilities
type Logger struct {
	level     LogLevel
	component string
	output    io.Writer
	mu        sync.Mutex
	fields    map[string]interface{}
}

// LoggerConfig contains configuration for the logger
type LoggerConfig struct {
	Level     LogLevel `json:"level"`
	Component string   `json:"component"`
	Output    string   `json:"output"` // "stdout", "stderr", or file path
	Format    string   `json:"format"` // "json" or "text"
}

// NewLogger creates a new logger instance
func NewLogger(config *LoggerConfig) *Logger {
	if config == nil {
		config = &LoggerConfig{
			Level:     INFO,
			Component: "system",
			Output:    "stdout",
			Format:    "json",
		}
	}

	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// Try to open file
		if file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			output = file
		} else {
			output = os.Stdout
		}
	}

	return &Logger{
		level:     config.Level,
		component: config.Component,
		output:    output,
		fields:    make(map[string]interface{}),
	}
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:     l.level,
		component: l.component,
		output:    l.output,
		fields:    make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		level:     l.level,
		component: l.component,
		output:    l.output,
		fields:    make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithComponent creates a logger with a specific component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		level:     l.level,
		component: component,
		output:    l.output,
		fields:    l.fields,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(DEBUG, message, nil, "")
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...), nil, "")
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(INFO, message, nil, "")
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...), nil, "")
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(WARN, message, nil, "")
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...), nil, "")
}

// Error logs an error message
func (l *Logger) Error(message string, err error) {
	l.log(ERROR, message, err, "")
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...), nil, "")
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error) {
	l.log(FATAL, message, err, "")
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...), nil, "")
	os.Exit(1)
}

// LogOperation logs the start and end of an operation
func (l *Logger) LogOperation(ctx context.Context, operation string, fn func() error) error {
	start := time.Now()

	// Extract trace ID from context if available
	traceID := getTraceIDFromContext(ctx)

	logger := l.WithField("operation", operation).WithField("trace_id", traceID)
	logger.Info("Operation started")

	err := fn()
	duration := time.Since(start)

	if err != nil {
		logger.WithField("duration_ms", duration.Milliseconds()).Error("Operation failed", err)
	} else {
		logger.WithField("duration_ms", duration.Milliseconds()).Info("Operation completed")
	}

	return err
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, err error, operation string) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Component: l.component,
		Operation: operation,
		Fields:    make(map[string]interface{}),
	}

	// Copy fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add error information
	if err != nil {
		entry.Error = err.Error()

		// Add stack trace for errors and above
		if level >= ERROR {
			entry.Stack = getStackTrace()
		}
	}

	// Add file and line information for errors and above
	if level >= ERROR {
		file, line := getCallerInfo(3) // Skip log, Error/Fatal, and the calling function
		entry.File = file
		entry.Line = line
	}

	// Write log entry
	l.writeEntry(&entry)
}

// writeEntry writes a log entry to the output
func (l *Logger) writeEntry(entry *LogEntry) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple text output if JSON marshaling fails
		fmt.Fprintf(l.output, "[%s] %s %s: %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level.String(),
			entry.Component,
			entry.Message)
		return
	}

	fmt.Fprintln(l.output, string(jsonData))
}

// getStackTrace returns the current stack trace
func getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// getCallerInfo returns the file and line number of the caller
func getCallerInfo(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}

	// Extract just the filename from the full path
	for i := len(file) - 1; i >= 0; i-- {
		if file[i] == '/' {
			file = file[i+1:]
			break
		}
	}

	return file, line
}

// getTraceIDFromContext extracts trace ID from context
func getTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}

	return ""
}

// PerformanceLogger provides performance-specific logging
type PerformanceLogger struct {
	logger *Logger
}

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger(logger *Logger) *PerformanceLogger {
	return &PerformanceLogger{
		logger: logger.WithComponent("performance"),
	}
}

// LogSlowOperation logs operations that exceed a threshold
func (pl *PerformanceLogger) LogSlowOperation(operation string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		pl.logger.WithFields(map[string]interface{}{
			"operation":    operation,
			"duration_ms":  duration.Milliseconds(),
			"threshold_ms": threshold.Milliseconds(),
			"slow_factor":  float64(duration) / float64(threshold),
		}).Warn("Slow operation detected")
	}
}

// LogMemoryUsage logs current memory usage
func (pl *PerformanceLogger) LogMemoryUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	pl.logger.WithFields(map[string]interface{}{
		"alloc_mb":       memStats.Alloc / 1024 / 1024,
		"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
		"sys_mb":         memStats.Sys / 1024 / 1024,
		"num_gc":         memStats.NumGC,
		"goroutines":     runtime.NumGoroutine(),
	}).Info("Memory usage snapshot")
}

// LogHTTPRequest logs HTTP request details
func (pl *PerformanceLogger) LogHTTPRequest(method, path string, statusCode int, duration time.Duration, size int64) {
	level := INFO
	if statusCode >= 400 {
		level = WARN
	}
	if statusCode >= 500 {
		level = ERROR
	}

	pl.logger.WithFields(map[string]interface{}{
		"method":        method,
		"path":          path,
		"status_code":   statusCode,
		"duration_ms":   duration.Milliseconds(),
		"response_size": size,
	}).log(level, "HTTP request processed", nil, "http_request")
}

// AuditLogger provides audit logging capabilities
type AuditLogger struct {
	logger *Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger.WithComponent("audit"),
	}
}

// LogUserAction logs user actions for audit purposes
func (al *AuditLogger) LogUserAction(userID, action, resource string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"user_id":  userID,
		"action":   action,
		"resource": resource,
	}

	// Add metadata fields
	for k, v := range metadata {
		fields[k] = v
	}

	al.logger.WithFields(fields).Info("User action performed")
}

// LogSystemEvent logs system events for audit purposes
func (al *AuditLogger) LogSystemEvent(event, component string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"event":     event,
		"component": component,
	}

	// Add metadata fields
	for k, v := range metadata {
		fields[k] = v
	}

	al.logger.WithFields(fields).Info("System event occurred")
}

// LogSecurityEvent logs security-related events
func (al *AuditLogger) LogSecurityEvent(event, userID, ipAddress string, metadata map[string]interface{}) {
	fields := map[string]interface{}{
		"event":      event,
		"user_id":    userID,
		"ip_address": ipAddress,
		"severity":   "security",
	}

	// Add metadata fields
	for k, v := range metadata {
		fields[k] = v
	}

	al.logger.WithFields(fields).Warn("Security event detected")
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *LoggerConfig) {
	globalLogger = NewLogger(config)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger(nil)
	}
	return globalLogger
}

// Convenience functions for global logger
func Debug(message string) {
	GetGlobalLogger().Debug(message)
}

func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().Debugf(format, args...)
}

func Info(message string) {
	GetGlobalLogger().Info(message)
}

func Infof(format string, args ...interface{}) {
	GetGlobalLogger().Infof(format, args...)
}

func Warn(message string) {
	GetGlobalLogger().Warn(message)
}

func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().Warnf(format, args...)
}

func Error(message string, err error) {
	GetGlobalLogger().Error(message, err)
}

func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().Errorf(format, args...)
}

func Fatal(message string, err error) {
	GetGlobalLogger().Fatal(message, err)
}

func Fatalf(format string, args ...interface{}) {
	GetGlobalLogger().Fatalf(format, args...)
}
