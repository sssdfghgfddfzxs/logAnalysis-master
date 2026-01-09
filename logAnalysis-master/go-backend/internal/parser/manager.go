package parser

import (
	"fmt"
	"log"
	"strings"
	"unicode/utf8"
)

// DefaultParserManager implements the ParserManager interface
type DefaultParserManager struct {
	parsers []LogParser
}

// NewParserManager creates a new parser manager with default parsers
func NewParserManager() ParserManager {
	manager := &DefaultParserManager{
		parsers: make([]LogParser, 0),
	}

	// Register default parsers in order of preference
	manager.RegisterParser(NewJSONParser())
	manager.RegisterParser(NewFilebeatParser())
	manager.RegisterParser(NewPlainTextParser())

	return manager
}

// RegisterParser registers a new log parser
func (m *DefaultParserManager) RegisterParser(parser LogParser) {
	if parser == nil {
		return
	}

	m.parsers = append(m.parsers, parser)
	log.Printf("Registered log parser for format: %s", parser.GetFormat())
}

// ParseLogs attempts to parse logs using the appropriate parser
func (m *DefaultParserManager) ParseLogs(data []byte) ([]*ParsedLog, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty log data")
	}

	// Validate UTF-8 encoding and handle encoding issues
	cleanData, err := m.sanitizeEncoding(data)
	if err != nil {
		return nil, fmt.Errorf("encoding error: %w", err)
	}

	// Try each parser until one succeeds
	var lastErr error
	for _, parser := range m.parsers {
		if parser.CanParse(cleanData) {
			entries, err := parser.Parse(cleanData)
			if err != nil {
				lastErr = err
				log.Printf("Parser %s failed: %v", parser.GetFormat(), err)
				continue
			}

			// Convert to ParsedLog format
			parsedLogs := make([]*ParsedLog, len(entries))
			for i, entry := range entries {
				parsedLogs[i] = &ParsedLog{
					LogEntry:       entry,
					OriginalFormat: parser.GetFormat(),
					ParsedFields:   make(map[string]string),
				}
			}

			log.Printf("Successfully parsed %d log entries using %s parser",
				len(parsedLogs), parser.GetFormat())
			return parsedLogs, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("no parser could handle the data format: %w", lastErr)
	}

	return nil, fmt.Errorf("no parser could handle the data format")
}

// DetectFormat detects the log format from the data
func (m *DefaultParserManager) DetectFormat(data []byte) string {
	if len(data) == 0 {
		return "unknown"
	}

	// Clean data first
	cleanData, err := m.sanitizeEncoding(data)
	if err != nil {
		return "unknown"
	}

	// Check each parser
	for _, parser := range m.parsers {
		if parser.CanParse(cleanData) {
			return parser.GetFormat()
		}
	}

	return "unknown"
}

// GetSupportedFormats returns a list of supported log formats
func (m *DefaultParserManager) GetSupportedFormats() []string {
	formats := make([]string, len(m.parsers))
	for i, parser := range m.parsers {
		formats[i] = parser.GetFormat()
	}
	return formats
}

// sanitizeEncoding handles character encoding issues and special characters
func (m *DefaultParserManager) sanitizeEncoding(data []byte) ([]byte, error) {
	// Check if data is valid UTF-8
	if utf8.Valid(data) {
		return data, nil
	}

	// Try to convert invalid UTF-8 sequences
	str := string(data)

	// Replace invalid UTF-8 sequences with replacement character
	cleanStr := strings.ToValidUTF8(str, "�")

	// Handle common encoding issues
	cleanStr = m.handleCommonEncodingIssues(cleanStr)

	return []byte(cleanStr), nil
}

// handleCommonEncodingIssues handles common character encoding problems
func (m *DefaultParserManager) handleCommonEncodingIssues(str string) string {
	// Handle common problematic characters
	replacements := map[string]string{
		"\x00": "",  // Null bytes
		"\x01": "",  // Start of heading
		"\x02": "",  // Start of text
		"\x03": "",  // End of text
		"\x04": "",  // End of transmission
		"\x05": "",  // Enquiry
		"\x06": "",  // Acknowledge
		"\x07": "",  // Bell
		"\x08": "",  // Backspace
		"\x0B": " ", // Vertical tab -> space
		"\x0C": " ", // Form feed -> space
		"\x0E": "",  // Shift out
		"\x0F": "",  // Shift in
		"\x10": "",  // Data link escape
		"\x11": "",  // Device control 1
		"\x12": "",  // Device control 2
		"\x13": "",  // Device control 3
		"\x14": "",  // Device control 4
		"\x15": "",  // Negative acknowledge
		"\x16": "",  // Synchronous idle
		"\x17": "",  // End of transmission block
		"\x18": "",  // Cancel
		"\x19": "",  // End of medium
		"\x1A": "",  // Substitute
		"\x1B": "",  // Escape
		"\x1C": "",  // File separator
		"\x1D": "",  // Group separator
		"\x1E": "",  // Record separator
		"\x1F": "",  // Unit separator
	}

	for old, new := range replacements {
		str = strings.ReplaceAll(str, old, new)
	}

	// Handle multiple consecutive spaces
	for strings.Contains(str, "  ") {
		str = strings.ReplaceAll(str, "  ", " ")
	}

	return str
}

// ParseMultipleFormats attempts to parse data that might contain multiple formats
func (m *DefaultParserManager) ParseMultipleFormats(data []byte) ([]*ParsedLog, error) {
	// Split data by lines and try to parse each line separately
	lines := strings.Split(string(data), "\n")
	var allParsedLogs []*ParsedLog
	var errors []string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parsedLogs, err := m.ParseLogs([]byte(line))
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: %v", i+1, err))
			continue
		}

		allParsedLogs = append(allParsedLogs, parsedLogs...)
	}

	if len(allParsedLogs) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to parse any lines: %s", strings.Join(errors, "; "))
	}

	// Add parsing errors to the first parsed log if any
	if len(errors) > 0 && len(allParsedLogs) > 0 {
		allParsedLogs[0].Errors = errors
	}

	return allParsedLogs, nil
}
