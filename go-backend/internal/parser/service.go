package parser

import (
	"context"
	"fmt"
	"intelligent-log-analysis/internal/models"
	"log"
	"strings"
	"time"
)

// LogParserService provides log parsing functionality
type LogParserService struct {
	manager ParserManager
}

// NewLogParserService creates a new log parser service
func NewLogParserService() *LogParserService {
	return &LogParserService{
		manager: NewParserManager(),
	}
}

// ParseLogData parses raw log data and returns structured log entries
func (s *LogParserService) ParseLogData(ctx context.Context, data []byte, format string) ([]*models.LogEntry, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty log data")
	}

	// If format is specified, try to use a specific parser
	if format != "" && format != "auto" {
		return s.parseWithSpecificFormat(data, format)
	}

	// Auto-detect format and parse
	parsedLogs, err := s.manager.ParseLogs(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log data: %w", err)
	}

	// Convert ParsedLog to LogEntry
	entries := make([]*models.LogEntry, len(parsedLogs))
	for i, parsedLog := range parsedLogs {
		entries[i] = parsedLog.LogEntry

		// Add parsing metadata
		if parsedLog.OriginalFormat != "" {
			if entries[i].Metadata == nil {
				entries[i].Metadata = make(map[string]string)
			}
			entries[i].Metadata["original_format"] = parsedLog.OriginalFormat
		}

		// Log any parsing errors
		if len(parsedLog.Errors) > 0 {
			log.Printf("Parsing warnings for log %s: %v", entries[i].ID, parsedLog.Errors)
		}
	}

	log.Printf("Successfully parsed %d log entries from %s format",
		len(entries), parsedLogs[0].OriginalFormat)

	return entries, nil
}

// parseWithSpecificFormat attempts to parse using a specific format
func (s *LogParserService) parseWithSpecificFormat(data []byte, format string) ([]*models.LogEntry, error) {
	var parser LogParser

	switch strings.ToLower(format) {
	case "json":
		parser = NewJSONParser()
	case "filebeat":
		parser = NewFilebeatParser()
	case "plaintext", "plain", "text":
		parser = NewPlainTextParser()
	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}

	if !parser.CanParse(data) {
		return nil, fmt.Errorf("data does not match specified format: %s", format)
	}

	return parser.Parse(data)
}

// DetectLogFormat detects the format of the given log data
func (s *LogParserService) DetectLogFormat(data []byte) string {
	return s.manager.DetectFormat(data)
}

// GetSupportedFormats returns a list of supported log formats
func (s *LogParserService) GetSupportedFormats() []string {
	return s.manager.GetSupportedFormats()
}

// ValidateLogData validates log data before parsing
func (s *LogParserService) ValidateLogData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("log data is empty")
	}

	// Check for reasonable size limits
	maxSize := 10 * 1024 * 1024 // 10MB
	if len(data) > maxSize {
		return fmt.Errorf("log data too large: %d bytes (max: %d)", len(data), maxSize)
	}

	// Check for null bytes (binary data)
	for i, b := range data {
		if b == 0 {
			return fmt.Errorf("binary data detected at position %d", i)
		}
	}

	return nil
}

// ParseMultipleFormats handles data that might contain multiple log formats
func (s *LogParserService) ParseMultipleFormats(ctx context.Context, data []byte) ([]*models.LogEntry, error) {
	if manager, ok := s.manager.(*DefaultParserManager); ok {
		parsedLogs, err := manager.ParseMultipleFormats(data)
		if err != nil {
			return nil, err
		}

		// Convert to LogEntry
		entries := make([]*models.LogEntry, len(parsedLogs))
		for i, parsedLog := range parsedLogs {
			entries[i] = parsedLog.LogEntry
		}

		return entries, nil
	}

	// Fallback to regular parsing
	return s.ParseLogData(ctx, data, "")
}

// ParseLogBatch parses multiple log entries in batch
func (s *LogParserService) ParseLogBatch(ctx context.Context, batches [][]byte) ([]*models.LogEntry, error) {
	var allEntries []*models.LogEntry

	for i, data := range batches {
		entries, err := s.ParseLogData(ctx, data, "")
		if err != nil {
			log.Printf("Failed to parse batch %d: %v", i, err)
			continue
		}

		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		return nil, fmt.Errorf("no valid log entries found in any batch")
	}

	return allEntries, nil
}

// NormalizeLogEntry normalizes a log entry to ensure consistency
func (s *LogParserService) NormalizeLogEntry(entry *models.LogEntry) error {
	if entry == nil {
		return fmt.Errorf("log entry is nil")
	}

	// Normalize level
	if entry.Level != "" {
		entry.Level = strings.ToUpper(strings.TrimSpace(entry.Level))

		// Handle common variations
		switch entry.Level {
		case "WARN", "WARNING":
			entry.Level = "WARN"
		case "ERR":
			entry.Level = "ERROR"
		case "CRIT", "CRITICAL":
			entry.Level = "ERROR"
		case "EMERG", "EMERGENCY", "PANIC":
			entry.Level = "FATAL"
		case "ALERT":
			entry.Level = "FATAL"
		case "NOTICE":
			entry.Level = "INFO"
		case "VERBOSE":
			entry.Level = "DEBUG"
		}

		// Validate level
		validLevels := map[string]bool{
			"DEBUG": true,
			"INFO":  true,
			"WARN":  true,
			"ERROR": true,
			"FATAL": true,
			"TRACE": true,
		}

		if !validLevels[entry.Level] {
			entry.Level = "INFO"
		}
	} else {
		entry.Level = "INFO"
	}

	// Normalize source
	if entry.Source == "" {
		entry.Source = "unknown"
	}
	entry.Source = strings.TrimSpace(entry.Source)

	// Normalize message
	entry.Message = strings.TrimSpace(entry.Message)
	if entry.Message == "" {
		return fmt.Errorf("log message cannot be empty")
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Initialize metadata if nil
	if entry.Metadata == nil {
		entry.Metadata = make(map[string]string)
	}

	return nil
}

// GetParsingStats returns statistics about parsing operations
func (s *LogParserService) GetParsingStats() map[string]interface{} {
	return map[string]interface{}{
		"supported_formats": s.GetSupportedFormats(),
		"parser_count":      len(s.manager.GetSupportedFormats()),
	}
}

// LogFormatInfo provides information about a log format
type LogFormatInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
	Fields      []string `json:"fields"`
}

// GetFormatInfo returns detailed information about supported formats
func (s *LogParserService) GetFormatInfo() []LogFormatInfo {
	return []LogFormatInfo{
		{
			Name:        "json",
			Description: "JSON formatted logs including NDJSON (newline-delimited JSON)",
			Examples: []string{
				`{"timestamp":"2024-01-01T10:00:00Z","level":"ERROR","message":"Database connection failed","source":"user-service"}`,
				`[{"level":"INFO","message":"Server started"},{"level":"ERROR","message":"Connection failed"}]`,
			},
			Fields: []string{"timestamp", "level", "message", "source", "metadata"},
		},
		{
			Name:        "filebeat",
			Description: "Filebeat/Elastic Beats formatted logs with ECS fields",
			Examples: []string{
				`{"@timestamp":"2024-01-01T10:00:00Z","message":"Error occurred","beat":{"name":"filebeat"},"host":{"name":"server01"}}`,
			},
			Fields: []string{"@timestamp", "message", "beat", "host", "agent", "input", "fields"},
		},
		{
			Name:        "plaintext",
			Description: "Plain text logs with automatic pattern detection",
			Examples: []string{
				`2024-01-01 10:00:00 [ERROR] [user-service] Database connection failed`,
				`Jan 01 10:00:00 server01 nginx: 192.168.1.1 - - [01/Jan/2024:10:00:00 +0000] "GET / HTTP/1.1" 200 1234`,
				`ERROR: Connection timeout occurred`,
			},
			Fields: []string{"timestamp", "level", "source", "message", "extracted_fields"},
		},
	}
}
