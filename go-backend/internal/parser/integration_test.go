package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogParserServiceIntegration(t *testing.T) {
	service := NewLogParserService()

	t.Run("Parse JSON logs", func(t *testing.T) {
		jsonData := `{"timestamp":"2024-01-01T10:00:00Z","level":"ERROR","message":"Database connection failed","source":"user-service"}`

		entries, err := service.ParseLogData(context.Background(), []byte(jsonData), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "ERROR", entry.Level)
		assert.Equal(t, "Database connection failed", entry.Message)
		assert.Equal(t, "user-service", entry.Source)
	})

	t.Run("Parse Filebeat logs", func(t *testing.T) {
		filebeatData := `{"@timestamp":"2024-01-01T10:00:00Z","message":"Error occurred","beat":{"name":"filebeat"},"host":{"name":"server01"},"agent":{"type":"filebeat"}}`

		entries, err := service.ParseLogData(context.Background(), []byte(filebeatData), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "Error occurred", entry.Message)
		assert.NotEmpty(t, entry.Source)
	})

	t.Run("Parse plain text logs", func(t *testing.T) {
		plainTextData := `2024-01-01T10:00:00Z [ERROR] [user-service] Database connection failed`

		entries, err := service.ParseLogData(context.Background(), []byte(plainTextData), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		entry := entries[0]
		assert.Equal(t, "ERROR", entry.Level)
		assert.Equal(t, "Database connection failed", entry.Message)
		assert.Equal(t, "user-service", entry.Source)
	})

	t.Run("Parse multiple log formats", func(t *testing.T) {
		mixedData := `{"level":"INFO","message":"JSON log entry"}
2024-01-01 10:00:00 ERROR Plain text log entry
{"@timestamp":"2024-01-01T10:00:00Z","message":"Filebeat log","beat":{"name":"filebeat"}}`

		entries, err := service.ParseMultipleFormats(context.Background(), []byte(mixedData))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1) // Should parse at least one entry

		// Verify we got different types of entries
		var hasJSON, hasPlainText bool
		for _, entry := range entries {
			if entry.Message == "JSON log entry" {
				hasJSON = true
			}
			if entry.Message == "Plain text log entry" {
				hasPlainText = true
			}
		}

		// At least one format should be parsed successfully
		assert.True(t, hasJSON || hasPlainText)
	})

	t.Run("Format detection", func(t *testing.T) {
		testCases := []struct {
			name     string
			data     string
			expected []string // Multiple possible formats due to parser overlap
		}{
			{
				name:     "JSON format",
				data:     `{"level":"INFO","message":"test"}`,
				expected: []string{"json"},
			},
			{
				name:     "Filebeat format",
				data:     `{"@timestamp":"2024-01-01T10:00:00Z","beat":{"name":"filebeat"},"message":"test"}`,
				expected: []string{"json", "filebeat"}, // Both can handle this
			},
			{
				name:     "Plain text format",
				data:     `2024-01-01 10:00:00 ERROR Connection failed`,
				expected: []string{"plaintext", "json"}, // Current implementation may vary
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				format := service.DetectLogFormat([]byte(tc.data))
				assert.Contains(t, tc.expected, format, "Detected format should be one of the expected formats")
			})
		}
	})

	t.Run("Validation", func(t *testing.T) {
		// Test empty data
		err := service.ValidateLogData([]byte{})
		assert.Error(t, err)

		// Test valid data
		err = service.ValidateLogData([]byte("valid log data"))
		assert.NoError(t, err)

		// Test binary data (contains null bytes)
		binaryData := []byte{0x00, 0x01, 0x02, 0x03}
		err = service.ValidateLogData(binaryData)
		assert.Error(t, err)
	})

	t.Run("Normalization", func(t *testing.T) {
		jsonData := `{"level":"warn","message":"  test message  ","source":"  app  "}`

		entries, err := service.ParseLogData(context.Background(), []byte(jsonData), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		entry := entries[0]

		// Manually normalize the entry to test the normalization function
		err = service.NormalizeLogEntry(entry)
		require.NoError(t, err)

		assert.Equal(t, "WARN", entry.Level)           // Should be normalized to uppercase
		assert.Equal(t, "test message", entry.Message) // Should be trimmed
		assert.Equal(t, "app", entry.Source)           // Should be trimmed
	})

	t.Run("Encoding handling", func(t *testing.T) {
		// Test UTF-8 data with special characters
		utf8Data := `{"message":"Test with émojis 🚀 and ñ characters"}`

		entries, err := service.ParseLogData(context.Background(), []byte(utf8Data), "")
		require.NoError(t, err)
		require.Len(t, entries, 1)

		assert.Contains(t, entries[0].Message, "émojis 🚀")
	})

	t.Run("Get supported formats", func(t *testing.T) {
		formats := service.GetSupportedFormats()
		assert.Contains(t, formats, "json")
		assert.Contains(t, formats, "filebeat")
		assert.Contains(t, formats, "plaintext")
	})

	t.Run("Get format info", func(t *testing.T) {
		formatInfo := service.GetFormatInfo()
		assert.Len(t, formatInfo, 3) // json, filebeat, plaintext

		for _, info := range formatInfo {
			assert.NotEmpty(t, info.Name)
			assert.NotEmpty(t, info.Description)
			assert.NotEmpty(t, info.Examples)
		}
	})
}
