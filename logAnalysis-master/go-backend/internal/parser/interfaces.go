package parser

import (
	"intelligent-log-analysis/internal/models"
)

// LogParser defines the interface for parsing different log formats
type LogParser interface {
	// Parse parses raw log data and returns structured log entries
	Parse(data []byte) ([]*models.LogEntry, error)

	// CanParse determines if this parser can handle the given data format
	CanParse(data []byte) bool

	// GetFormat returns the format name this parser handles
	GetFormat() string
}

// ParsedLog represents a parsed log entry with additional metadata
type ParsedLog struct {
	*models.LogEntry
	OriginalFormat string            `json:"original_format"`
	ParsedFields   map[string]string `json:"parsed_fields"`
	Errors         []string          `json:"errors,omitempty"`
}

// ParserManager manages multiple log parsers and handles format detection
type ParserManager interface {
	// RegisterParser registers a new log parser
	RegisterParser(parser LogParser)

	// ParseLogs attempts to parse logs using the appropriate parser
	ParseLogs(data []byte) ([]*ParsedLog, error)

	// DetectFormat detects the log format from the data
	DetectFormat(data []byte) string

	// GetSupportedFormats returns a list of supported log formats
	GetSupportedFormats() []string
}

// LogField represents a standardized log field mapping
type LogField struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // string, timestamp, number, boolean
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
}

// FieldMapper handles mapping between different log formats and our standard format
type FieldMapper interface {
	// MapFields maps fields from source format to standard format
	MapFields(sourceFields map[string]interface{}) (*models.LogEntry, error)

	// GetFieldMappings returns the field mapping configuration
	GetFieldMappings() map[string]LogField
}
