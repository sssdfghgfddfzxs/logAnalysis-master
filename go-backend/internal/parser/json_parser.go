package parser

import (
	"encoding/json"
	"fmt"
	"intelligent-log-analysis/internal/models"
	"strings"
	"time"
)

// JSONParser handles JSON-formatted log entries
type JSONParser struct {
	fieldMapper FieldMapper
}

// NewJSONParser creates a new JSON parser
func NewJSONParser() LogParser {
	return &JSONParser{
		fieldMapper: NewStandardFieldMapper(),
	}
}

// Parse parses JSON log data
func (p *JSONParser) Parse(data []byte) ([]*models.LogEntry, error) {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, fmt.Errorf("empty JSON data")
	}

	// Try to parse as single JSON object first
	var singleLog map[string]interface{}
	if err := json.Unmarshal(data, &singleLog); err == nil {
		entry, err := p.parseJSONObject(singleLog)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON object: %w", err)
		}
		return []*models.LogEntry{entry}, nil
	}

	// Try to parse as JSON array
	var logArray []map[string]interface{}
	if err := json.Unmarshal(data, &logArray); err == nil {
		entries := make([]*models.LogEntry, 0, len(logArray))
		for i, logObj := range logArray {
			entry, err := p.parseJSONObject(logObj)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON object at index %d: %w", i, err)
			}
			entries = append(entries, entry)
		}
		return entries, nil
	}

	// Try to parse as NDJSON (newline-delimited JSON)
	lines := strings.Split(string(data), "\n")
	var entries []*models.LogEntry

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var logObj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logObj); err != nil {
			return nil, fmt.Errorf("failed to parse JSON line %d: %w", i+1, err)
		}

		entry, err := p.parseJSONObject(logObj)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON object at line %d: %w", i+1, err)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid JSON log entries found")
	}

	return entries, nil
}

// parseJSONObject converts a JSON object to a LogEntry
func (p *JSONParser) parseJSONObject(obj map[string]interface{}) (*models.LogEntry, error) {
	entry, err := p.fieldMapper.MapFields(obj)
	if err != nil {
		return nil, err
	}

	// Validate required fields
	if entry.Message == "" {
		return nil, fmt.Errorf("missing required field: message")
	}

	if entry.Level == "" {
		entry.Level = "INFO" // Default level
	}

	if entry.Source == "" {
		entry.Source = "unknown" // Default source
	}

	// Normalize level to uppercase
	entry.Level = strings.ToUpper(entry.Level)

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return entry, nil
}

// CanParse determines if this parser can handle the given data
func (p *JSONParser) CanParse(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	data = []byte(strings.TrimSpace(string(data)))

	// Check if it starts with { or [
	if data[0] == '{' || data[0] == '[' {
		// Try to parse as JSON to make sure it's valid
		var obj interface{}
		if json.Unmarshal(data, &obj) == nil {
			// Check if it looks like a log entry (has message or similar fields)
			if objMap, ok := obj.(map[string]interface{}); ok {
				return p.looksLikeLogJSON(objMap)
			}
			if objArray, ok := obj.([]interface{}); ok {
				// Check first element if it's an array
				if len(objArray) > 0 {
					if firstObj, ok := objArray[0].(map[string]interface{}); ok {
						return p.looksLikeLogJSON(firstObj)
					}
				}
			}
			return true // Valid JSON, assume it's logs
		}
		return false
	}

	// Check for NDJSON format (multiple lines, each starting with {)
	lines := strings.Split(string(data), "\n")
	validJSONLines := 0
	totalLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		totalLines++

		if strings.HasPrefix(line, "{") {
			var obj map[string]interface{}
			if json.Unmarshal([]byte(line), &obj) == nil {
				if p.looksLikeLogJSON(obj) {
					validJSONLines++
				}
			}
		}
	}

	// If more than half the non-empty lines are valid log JSON, consider it NDJSON
	return totalLines > 0 && validJSONLines >= totalLines/2
}

// looksLikeLogJSON checks if a JSON object looks like a log entry
func (p *JSONParser) looksLikeLogJSON(obj map[string]interface{}) bool {
	// Check for common log fields
	logFields := []string{"message", "msg", "text", "content", "level", "severity", "timestamp", "time", "@timestamp"}

	for _, field := range logFields {
		if _, exists := obj[field]; exists {
			return true
		}
	}

	// If it has beat, agent, or host fields, it might be Filebeat (let Filebeat parser handle it)
	filebeatFields := []string{"beat", "agent", "host", "input", "ecs"}
	filebeatFieldCount := 0
	for _, field := range filebeatFields {
		if _, exists := obj[field]; exists {
			filebeatFieldCount++
		}
	}

	// If it has 2 or more Filebeat fields, let Filebeat parser handle it
	if filebeatFieldCount >= 2 {
		return false
	}

	return false
}

// GetFormat returns the format name
func (p *JSONParser) GetFormat() string {
	return "json"
}

// StandardFieldMapper implements FieldMapper for standard log fields
type StandardFieldMapper struct {
	fieldMappings map[string]LogField
}

// NewStandardFieldMapper creates a new standard field mapper
func NewStandardFieldMapper() FieldMapper {
	return &StandardFieldMapper{
		fieldMappings: map[string]LogField{
			"timestamp":  {Name: "timestamp", Type: "timestamp", Required: false},
			"time":       {Name: "timestamp", Type: "timestamp", Required: false},
			"@timestamp": {Name: "timestamp", Type: "timestamp", Required: false},
			"ts":         {Name: "timestamp", Type: "timestamp", Required: false},
			"datetime":   {Name: "timestamp", Type: "timestamp", Required: false},

			"level":    {Name: "level", Type: "string", Required: false, DefaultValue: "INFO"},
			"severity": {Name: "level", Type: "string", Required: false, DefaultValue: "INFO"},
			"priority": {Name: "level", Type: "string", Required: false, DefaultValue: "INFO"},
			"loglevel": {Name: "level", Type: "string", Required: false, DefaultValue: "INFO"},

			"message": {Name: "message", Type: "string", Required: true},
			"msg":     {Name: "message", Type: "string", Required: true},
			"text":    {Name: "message", Type: "string", Required: true},
			"content": {Name: "message", Type: "string", Required: true},

			"source":    {Name: "source", Type: "string", Required: false, DefaultValue: "unknown"},
			"service":   {Name: "source", Type: "string", Required: false, DefaultValue: "unknown"},
			"component": {Name: "source", Type: "string", Required: false, DefaultValue: "unknown"},
			"logger":    {Name: "source", Type: "string", Required: false, DefaultValue: "unknown"},
			"app":       {Name: "source", Type: "string", Required: false, DefaultValue: "unknown"},
		},
	}
}

// MapFields maps fields from source format to standard format
func (m *StandardFieldMapper) MapFields(sourceFields map[string]interface{}) (*models.LogEntry, error) {
	entry := &models.LogEntry{
		Metadata: make(map[string]string),
	}

	// Map known fields
	for sourceField, value := range sourceFields {
		lowerField := strings.ToLower(sourceField)

		if mapping, exists := m.fieldMappings[lowerField]; exists {
			switch mapping.Name {
			case "timestamp":
				if ts, err := m.parseTimestamp(value); err == nil {
					entry.Timestamp = ts
				}
			case "level":
				if str, ok := m.toString(value); ok {
					entry.Level = str
				}
			case "message":
				if str, ok := m.toString(value); ok {
					entry.Message = str
				}
			case "source":
				if str, ok := m.toString(value); ok {
					entry.Source = str
				}
			}
		} else {
			// Add unmapped fields to metadata
			if str, ok := m.toString(value); ok {
				entry.Metadata[sourceField] = str
			}
		}
	}

	// Apply defaults for missing required fields
	for _, mapping := range m.fieldMappings {
		if mapping.DefaultValue != "" {
			switch mapping.Name {
			case "level":
				if entry.Level == "" {
					entry.Level = mapping.DefaultValue
				}
			case "source":
				if entry.Source == "" {
					entry.Source = mapping.DefaultValue
				}
			}
		}
	}

	return entry, nil
}

// GetFieldMappings returns the field mapping configuration
func (m *StandardFieldMapper) GetFieldMappings() map[string]LogField {
	return m.fieldMappings
}

// parseTimestamp attempts to parse various timestamp formats
func (m *StandardFieldMapper) parseTimestamp(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return m.parseTimestampString(v)
	case float64:
		// Unix timestamp
		return time.Unix(int64(v), 0), nil
	case int64:
		// Unix timestamp
		return time.Unix(v, 0), nil
	case int:
		// Unix timestamp
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type: %T", value)
	}
}

// parseTimestampString parses timestamp strings in various formats
func (m *StandardFieldMapper) parseTimestampString(str string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006/01/02 15:04:05",
		"Jan 02 15:04:05",
		"Jan  2 15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			// If year is missing, use current year
			if t.Year() == 0 {
				now := time.Now()
				t = t.AddDate(now.Year(), 0, 0)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", str)
}

// toString converts various types to string
func (m *StandardFieldMapper) toString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case int:
		return fmt.Sprintf("%d", v), true
	case int64:
		return fmt.Sprintf("%d", v), true
	case float64:
		return fmt.Sprintf("%g", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	case nil:
		return "", false
	default:
		// Try JSON marshaling for complex types
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes), true
		}
		return fmt.Sprintf("%v", v), true
	}
}
