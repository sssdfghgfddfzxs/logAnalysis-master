package parser

import (
	"encoding/json"
	"fmt"
	"intelligent-log-analysis/internal/models"
	"strings"
	"time"
)

// FilebeatParser handles Filebeat-formatted log entries
type FilebeatParser struct {
	fieldMapper FieldMapper
}

// NewFilebeatParser creates a new Filebeat parser
func NewFilebeatParser() LogParser {
	return &FilebeatParser{
		fieldMapper: NewFilebeatFieldMapper(),
	}
}

// FilebeatLog represents the structure of a Filebeat log entry
type FilebeatLog struct {
	Timestamp time.Time              `json:"@timestamp"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Beat      FilebeatInfo           `json:"beat,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Offset    int64                  `json:"offset,omitempty"`
	Type      string                 `json:"type,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Host      FilebeatHost           `json:"host,omitempty"`
	Input     FilebeatInput          `json:"input,omitempty"`
	Log       FilebeatLogInfo        `json:"log,omitempty"`
	Agent     FilebeatAgent          `json:"agent,omitempty"`
	ECS       FilebeatECS            `json:"ecs,omitempty"`
}

// FilebeatInfo contains beat information
type FilebeatInfo struct {
	Name     string `json:"name,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Version  string `json:"version,omitempty"`
}

// FilebeatHost contains host information
type FilebeatHost struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// FilebeatInput contains input information
type FilebeatInput struct {
	Type string `json:"type,omitempty"`
}

// FilebeatLogInfo contains log file information
type FilebeatLogInfo struct {
	File   FilebeatFile `json:"file,omitempty"`
	Offset int64        `json:"offset,omitempty"`
}

// FilebeatFile contains file information
type FilebeatFile struct {
	Path string `json:"path,omitempty"`
}

// FilebeatAgent contains agent information
type FilebeatAgent struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Version  string `json:"version,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// FilebeatECS contains ECS information
type FilebeatECS struct {
	Version string `json:"version,omitempty"`
}

// Parse parses Filebeat log data
func (p *FilebeatParser) Parse(data []byte) ([]*models.LogEntry, error) {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, fmt.Errorf("empty Filebeat data")
	}

	// Try to parse as single Filebeat log entry
	var singleLog FilebeatLog
	if err := json.Unmarshal(data, &singleLog); err == nil {
		entry, err := p.parseFilebeatLog(&singleLog)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Filebeat log: %w", err)
		}
		return []*models.LogEntry{entry}, nil
	}

	// Try to parse as array of Filebeat logs
	var logArray []FilebeatLog
	if err := json.Unmarshal(data, &logArray); err == nil {
		entries := make([]*models.LogEntry, 0, len(logArray))
		for i, fbLog := range logArray {
			entry, err := p.parseFilebeatLog(&fbLog)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Filebeat log at index %d: %w", i, err)
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

		var fbLog FilebeatLog
		if err := json.Unmarshal([]byte(line), &fbLog); err != nil {
			return nil, fmt.Errorf("failed to parse Filebeat JSON line %d: %w", i+1, err)
		}

		entry, err := p.parseFilebeatLog(&fbLog)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Filebeat log at line %d: %w", i+1, err)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid Filebeat log entries found")
	}

	return entries, nil
}

// parseFilebeatLog converts a FilebeatLog to a LogEntry
func (p *FilebeatParser) parseFilebeatLog(fbLog *FilebeatLog) (*models.LogEntry, error) {
	// Convert Filebeat log to generic map for field mapping
	fbData := make(map[string]interface{})

	// Add standard fields
	fbData["@timestamp"] = fbLog.Timestamp
	fbData["message"] = fbLog.Message
	fbData["type"] = fbLog.Type
	fbData["source"] = fbLog.Source
	fbData["offset"] = fbLog.Offset
	fbData["tags"] = fbLog.Tags

	// Add beat information
	if fbLog.Beat.Name != "" {
		fbData["beat.name"] = fbLog.Beat.Name
		fbData["beat.hostname"] = fbLog.Beat.Hostname
		fbData["beat.version"] = fbLog.Beat.Version
	}

	// Add host information
	if fbLog.Host.Name != "" {
		fbData["host.name"] = fbLog.Host.Name
		fbData["host.id"] = fbLog.Host.ID
	}

	// Add input information
	if fbLog.Input.Type != "" {
		fbData["input.type"] = fbLog.Input.Type
	}

	// Add log file information
	if fbLog.Log.File.Path != "" {
		fbData["log.file.path"] = fbLog.Log.File.Path
		fbData["log.offset"] = fbLog.Log.Offset
	}

	// Add agent information
	if fbLog.Agent.Name != "" {
		fbData["agent.name"] = fbLog.Agent.Name
		fbData["agent.type"] = fbLog.Agent.Type
		fbData["agent.version"] = fbLog.Agent.Version
		fbData["agent.hostname"] = fbLog.Agent.Hostname
	}

	// Add ECS information
	if fbLog.ECS.Version != "" {
		fbData["ecs.version"] = fbLog.ECS.Version
	}

	// Add custom fields
	for key, value := range fbLog.Fields {
		fbData[fmt.Sprintf("fields.%s", key)] = value
	}

	// Use field mapper to convert to LogEntry
	entry, err := p.fieldMapper.MapFields(fbData)
	if err != nil {
		return nil, err
	}

	// Validate and set defaults
	if entry.Message == "" {
		return nil, fmt.Errorf("missing required field: message")
	}

	if entry.Level == "" {
		// Try to extract level from message or tags
		entry.Level = p.extractLogLevel(fbLog)
	}

	if entry.Source == "" {
		entry.Source = p.extractSource(fbLog)
	}

	// Set timestamp
	if !fbLog.Timestamp.IsZero() {
		entry.Timestamp = fbLog.Timestamp
	} else if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return entry, nil
}

// extractLogLevel attempts to extract log level from Filebeat log
func (p *FilebeatParser) extractLogLevel(fbLog *FilebeatLog) string {
	// Check tags for log level
	for _, tag := range fbLog.Tags {
		level := strings.ToUpper(tag)
		if p.isValidLogLevel(level) {
			return level
		}
	}

	// Check message for log level patterns
	message := strings.ToUpper(fbLog.Message)
	levels := []string{"ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE", "FATAL"}

	for _, level := range levels {
		if strings.Contains(message, level) {
			return level
		}
	}

	// Check fields for log level
	for key, value := range fbLog.Fields {
		if strings.Contains(strings.ToLower(key), "level") {
			if str, ok := value.(string); ok {
				level := strings.ToUpper(str)
				if p.isValidLogLevel(level) {
					return level
				}
			}
		}
	}

	return "INFO" // Default level
}

// extractSource attempts to extract source from Filebeat log
func (p *FilebeatParser) extractSource(fbLog *FilebeatLog) string {
	// Priority order for source extraction
	sources := []string{
		fbLog.Source,
		fbLog.Beat.Name,
		fbLog.Host.Name,
		fbLog.Agent.Name,
		fbLog.Type,
	}

	for _, source := range sources {
		if source != "" {
			return source
		}
	}

	// Check log file path
	if fbLog.Log.File.Path != "" {
		// Extract filename from path
		parts := strings.Split(fbLog.Log.File.Path, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			// Remove extension
			if dotIndex := strings.LastIndex(filename, "."); dotIndex > 0 {
				filename = filename[:dotIndex]
			}
			return filename
		}
	}

	return "filebeat"
}

// isValidLogLevel checks if a string is a valid log level
func (p *FilebeatParser) isValidLogLevel(level string) bool {
	validLevels := map[string]bool{
		"DEBUG":   true,
		"INFO":    true,
		"WARN":    true,
		"WARNING": true,
		"ERROR":   true,
		"FATAL":   true,
		"TRACE":   true,
	}
	return validLevels[level]
}

// CanParse determines if this parser can handle the given data
func (p *FilebeatParser) CanParse(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	data = []byte(strings.TrimSpace(string(data)))

	// Quick check for JSON structure
	if !strings.HasPrefix(string(data), "{") && !strings.HasPrefix(string(data), "[") {
		return false
	}

	// Try to parse as JSON and check for Filebeat-specific fields
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		// Try NDJSON
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				break
			}
		}

		if len(obj) == 0 {
			return false
		}
	}

	// Check for Filebeat-specific fields
	filebeatFields := []string{"@timestamp", "beat", "input", "agent", "ecs"}
	filebeatFieldCount := 0

	for _, field := range filebeatFields {
		if _, exists := obj[field]; exists {
			filebeatFieldCount++
		}
	}

	// If we have at least 2 Filebeat-specific fields, consider it Filebeat format
	return filebeatFieldCount >= 2
}

// GetFormat returns the format name
func (p *FilebeatParser) GetFormat() string {
	return "filebeat"
}

// FilebeatFieldMapper implements FieldMapper for Filebeat logs
type FilebeatFieldMapper struct {
	*StandardFieldMapper
}

// NewFilebeatFieldMapper creates a new Filebeat field mapper
func NewFilebeatFieldMapper() FieldMapper {
	base := NewStandardFieldMapper().(*StandardFieldMapper)

	// Add Filebeat-specific field mappings
	filebeatMappings := map[string]LogField{
		"@timestamp": {Name: "timestamp", Type: "timestamp", Required: false},
		"beat.name":  {Name: "source", Type: "string", Required: false},
		"host.name":  {Name: "source", Type: "string", Required: false},
		"agent.name": {Name: "source", Type: "string", Required: false},
		"type":       {Name: "source", Type: "string", Required: false},
	}

	// Merge with base mappings
	for key, value := range filebeatMappings {
		base.fieldMappings[key] = value
	}

	return &FilebeatFieldMapper{
		StandardFieldMapper: base,
	}
}
