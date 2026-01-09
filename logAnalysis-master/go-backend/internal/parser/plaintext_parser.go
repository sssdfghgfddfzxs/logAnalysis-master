package parser

import (
	"fmt"
	"intelligent-log-analysis/internal/models"
	"regexp"
	"strings"
	"time"
)

// PlainTextParser handles plain text log entries
type PlainTextParser struct {
	patterns []LogPattern
}

// LogPattern represents a pattern for parsing plain text logs
type LogPattern struct {
	Name        string
	Regex       *regexp.Regexp
	FieldNames  []string
	Description string
}

// NewPlainTextParser creates a new plain text parser
func NewPlainTextParser() LogParser {
	return &PlainTextParser{
		patterns: getDefaultLogPatterns(),
	}
}

// getDefaultLogPatterns returns common log patterns
func getDefaultLogPatterns() []LogPattern {
	patterns := []LogPattern{
		{
			Name:        "Apache Common Log Format",
			Regex:       regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([^"]*)" (\d+) (\d+|-)`),
			FieldNames:  []string{"host", "timestamp", "request", "status", "size"},
			Description: "Apache/Nginx access log format",
		},
		{
			Name:        "Apache Combined Log Format",
			Regex:       regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([^"]*)" (\d+) (\d+|-) "([^"]*)" "([^"]*)"`),
			FieldNames:  []string{"host", "timestamp", "request", "status", "size", "referer", "user_agent"},
			Description: "Apache/Nginx combined log format",
		},
		{
			Name:        "Syslog Format",
			Regex:       regexp.MustCompile(`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([^:]+):\s*(.*)`),
			FieldNames:  []string{"timestamp", "host", "process", "message"},
			Description: "Standard syslog format",
		},
		{
			Name:        "ISO Timestamp with Level",
			Regex:       regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d{3})?(?:Z|[+-]\d{2}:\d{2})?)\s+\[?(\w+)\]?\s+(?:\[([^\]]+)\])?\s*(.*)`),
			FieldNames:  []string{"timestamp", "level", "source", "message"},
			Description: "ISO timestamp with log level",
		},
		{
			Name:        "Simple Timestamp with Level",
			Regex:       regexp.MustCompile(`^(\d{2}/\d{2}/\d{4}\s+\d{2}:\d{2}:\d{2})\s+(\w+)\s+(?:\[([^\]]+)\])?\s*(.*)`),
			FieldNames:  []string{"timestamp", "level", "source", "message"},
			Description: "Simple timestamp with log level",
		},
		{
			Name:        "Java Log Format",
			Regex:       regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2},\d{3})\s+\[([^\]]+)\]\s+(\w+)\s+([^\s]+)\s+-\s+(.*)`),
			FieldNames:  []string{"timestamp", "thread", "level", "logger", "message"},
			Description: "Java application log format",
		},
		{
			Name:        "Docker Container Log",
			Regex:       regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(\w+)\s+(.*)`),
			FieldNames:  []string{"timestamp", "stream", "message"},
			Description: "Docker container log format",
		},
		{
			Name:        "Generic Level Message",
			Regex:       regexp.MustCompile(`^(?:\[?(\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d{3})?(?:Z|[+-]\d{2}:\d{2})?)\]?)?\s*\[?(\w+)\]?\s*(.*)`),
			FieldNames:  []string{"timestamp", "level", "message"},
			Description: "Generic format with optional timestamp and level",
		},
		{
			Name:        "Simple Message",
			Regex:       regexp.MustCompile(`^(.+)$`),
			FieldNames:  []string{"message"},
			Description: "Fallback pattern for any text",
		},
	}

	return patterns
}

// Parse parses plain text log data
func (p *PlainTextParser) Parse(data []byte) ([]*models.LogEntry, error) {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, fmt.Errorf("empty plain text data")
	}

	lines := strings.Split(text, "\n")
	var entries []*models.LogEntry

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry, err := p.parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", i+1, err)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid log entries found")
	}

	return entries, nil
}

// parseLine parses a single line of plain text log
func (p *PlainTextParser) parseLine(line string) (*models.LogEntry, error) {
	// Try each pattern until one matches
	for _, pattern := range p.patterns {
		matches := pattern.Regex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		// Create log entry from matches
		entry := &models.LogEntry{
			Metadata: make(map[string]string),
		}

		// Map matches to fields
		for i, fieldName := range pattern.FieldNames {
			if i+1 < len(matches) {
				value := strings.TrimSpace(matches[i+1])
				if value == "" || value == "-" {
					continue
				}

				switch fieldName {
				case "timestamp":
					if ts, err := p.parseTimestamp(value); err == nil {
						entry.Timestamp = ts
					}
				case "level":
					entry.Level = p.normalizeLevel(value)
				case "message":
					entry.Message = value
				case "source", "host", "logger", "process":
					if entry.Source == "" {
						entry.Source = value
					} else {
						entry.Metadata[fieldName] = value
					}
				default:
					entry.Metadata[fieldName] = value
				}
			}
		}

		// Set defaults
		if entry.Message == "" {
			entry.Message = line // Use entire line as message if no specific message field
		}

		if entry.Level == "" {
			entry.Level = p.extractLevelFromMessage(entry.Message)
		}

		if entry.Source == "" {
			entry.Source = "plaintext"
		}

		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		// Store the pattern used for parsing
		entry.Metadata["parser_pattern"] = pattern.Name

		return entry, nil
	}

	return nil, fmt.Errorf("no pattern matched the log line")
}

// parseTimestamp attempts to parse various timestamp formats
func (p *PlainTextParser) parseTimestamp(str string) (time.Time, error) {
	formats := []string{
		// ISO formats
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",

		// Standard formats
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"01/02/2006 15:04:05",
		"02/01/2006 15:04:05",

		// Java log format
		"2006-01-02 15:04:05,000",

		// Syslog format
		"Jan 02 15:04:05",
		"Jan  2 15:04:05",

		// Apache log format
		"02/Jan/2006:15:04:05 -0700",
		"02/Jan/2006:15:04:05 +0000",

		// Other common formats
		"2006-01-02",
		"15:04:05",
		"15:04:05.000",
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

// normalizeLevel normalizes log level strings
func (p *PlainTextParser) normalizeLevel(level string) string {
	level = strings.ToUpper(strings.TrimSpace(level))

	// Handle common variations
	switch level {
	case "WARN", "WARNING":
		return "WARN"
	case "ERR":
		return "ERROR"
	case "CRIT", "CRITICAL":
		return "ERROR"
	case "EMERG", "EMERGENCY":
		return "FATAL"
	case "ALERT":
		return "FATAL"
	case "NOTICE":
		return "INFO"
	case "VERBOSE":
		return "DEBUG"
	}

	// Validate against known levels
	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
		"FATAL": true,
		"TRACE": true,
	}

	if validLevels[level] {
		return level
	}

	return "INFO" // Default level
}

// extractLevelFromMessage attempts to extract log level from message content
func (p *PlainTextParser) extractLevelFromMessage(message string) string {
	message = strings.ToUpper(message)

	// Look for level keywords in the message
	levels := []string{"FATAL", "ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE"}

	for _, level := range levels {
		// Look for level at word boundaries
		pattern := regexp.MustCompile(`\b` + level + `\b`)
		if pattern.MatchString(message) {
			return p.normalizeLevel(level)
		}
	}

	// Look for error indicators
	errorPatterns := []string{
		"EXCEPTION", "STACK TRACE", "FAILED", "FAILURE", "PANIC",
		"CRITICAL", "SEVERE", "ALERT", "EMERGENCY",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(message, pattern) {
			return "ERROR"
		}
	}

	// Look for warning indicators
	warningPatterns := []string{
		"DEPRECATED", "OBSOLETE", "RETRY", "TIMEOUT", "SLOW",
	}

	for _, pattern := range warningPatterns {
		if strings.Contains(message, pattern) {
			return "WARN"
		}
	}

	return "INFO" // Default level
}

// CanParse determines if this parser can handle the given data
func (p *PlainTextParser) CanParse(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	text := strings.TrimSpace(string(data))

	// Plain text parser is the fallback, so it can parse anything
	// But we should check if it's not obviously JSON or other structured format

	// Skip if it looks like JSON
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return false
	}

	// Skip if it looks like XML
	if strings.HasPrefix(text, "<") {
		return false
	}

	// Skip if it looks like CSV with headers
	if strings.Contains(text, ",") && strings.Count(text, ",") > 3 {
		lines := strings.Split(text, "\n")
		if len(lines) > 1 {
			firstLine := strings.ToLower(lines[0])
			if strings.Contains(firstLine, "timestamp") || strings.Contains(firstLine, "time") {
				return false
			}
		}
	}

	// Check if we can match at least one line with our patterns
	lines := strings.Split(text, "\n")
	matchedLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to match with any pattern except the fallback pattern
		for i, pattern := range p.patterns {
			if i == len(p.patterns)-1 {
				// Skip the fallback "Simple Message" pattern for this check
				continue
			}

			if pattern.Regex.MatchString(line) {
				matchedLines++
				break
			}
		}

		// If we've checked enough lines and found some matches, it's probably plain text logs
		if len(lines) > 10 && matchedLines > 0 {
			break
		}
	}

	// Accept if we matched at least some lines, or if it's clearly text content
	return matchedLines > 0 || p.looksLikeLogText(text)
}

// looksLikeLogText checks if the text looks like log content
func (p *PlainTextParser) looksLikeLogText(text string) bool {
	// Check for common log indicators
	logIndicators := []string{
		"error", "warning", "info", "debug", "fatal",
		"exception", "stack trace", "failed", "success",
		"started", "stopped", "connecting", "connected",
		"request", "response", "http", "tcp", "udp",
		"database", "sql", "query", "transaction",
	}

	lowerText := strings.ToLower(text)
	indicatorCount := 0

	for _, indicator := range logIndicators {
		if strings.Contains(lowerText, indicator) {
			indicatorCount++
		}
	}

	// If we find multiple log-like terms, it's probably a log
	return indicatorCount >= 2
}

// GetFormat returns the format name
func (p *PlainTextParser) GetFormat() string {
	return "plaintext"
}
