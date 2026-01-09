package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringMap is a custom type for handling string map in JSONB
type StringMap map[string]string

// Value implements the driver.Valuer interface for database storage
func (s StringMap) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *StringMap) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into StringMap", value)
	}

	return json.Unmarshal(bytes, s)
}

// StringArray is a custom type for handling string arrays in PostgreSQL
type StringArray []string

// Value implements the driver.Valuer interface for database storage
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}

	// Convert to PostgreSQL array format: {"item1","item2","item3"}
	if len(s) == 0 {
		return "{}", nil
	}

	result := "{"
	for i, item := range s {
		if i > 0 {
			result += ","
		}
		// Escape quotes in the item
		escaped := fmt.Sprintf(`"%s"`, item)
		result += escaped
	}
	result += "}"

	return result, nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}

	// Handle empty array
	if str == "{}" || str == "" {
		*s = StringArray{}
		return nil
	}

	// Parse PostgreSQL array format: {item1,item2,item3}
	if len(str) < 2 || str[0] != '{' || str[len(str)-1] != '}' {
		return fmt.Errorf("invalid array format: %s", str)
	}

	// Remove braces and split by comma
	content := str[1 : len(str)-1]
	if content == "" {
		*s = StringArray{}
		return nil
	}

	// Simple split by comma (this assumes no commas in the values)
	// For more complex parsing, you might need a proper parser
	parts := strings.Split(content, ",")
	result := make(StringArray, len(parts))

	for i, part := range parts {
		// Remove quotes if present
		trimmed := strings.TrimSpace(part)
		if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			trimmed = trimmed[1 : len(trimmed)-1]
		}
		result[i] = trimmed
	}

	*s = result
	return nil
}

type LogEntry struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`
	Level     string    `json:"level" gorm:"not null;size:20;index"`
	Message   string    `json:"message" gorm:"not null;type:text"`
	Source    string    `json:"source" gorm:"not null;size:100;index"`
	Metadata  StringMap `json:"metadata" gorm:"type:jsonb"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for LogEntry
func (LogEntry) TableName() string {
	return "logs"
}

func (l *LogEntry) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

type AnalysisResult struct {
	ID              string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	LogID           string    `json:"log_id" gorm:"not null;index"`
	IsAnomaly       bool      `json:"is_anomaly" gorm:"not null;index"`
	AnomalyScore    float64   `json:"anomaly_score" gorm:"type:decimal(5,4);index"`
	RootCauses      JSONMap   `json:"root_causes" gorm:"type:jsonb"`
	Recommendations JSONMap   `json:"recommendations" gorm:"type:jsonb"`
	AnalyzedAt      time.Time `json:"analyzed_at" gorm:"autoCreateTime"`

	// Relationship
	Log LogEntry `json:"log" gorm:"foreignKey:LogID;references:ID"`
}

func (a *AnalysisResult) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// JSONMap is a custom type for handling JSON data in database
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database retrieval
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}

	return json.Unmarshal(bytes, j)
}

type AlertRule struct {
	ID                   string      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name                 string      `json:"name" gorm:"not null;size:100"`
	Description          string      `json:"description" gorm:"size:500"`
	Condition            JSONMap     `json:"condition" gorm:"type:jsonb;not null"`
	NotificationChannels StringArray `json:"notification_channels" gorm:"type:text[]"`
	IsActive             bool        `json:"is_active" gorm:"default:true"`
	CreatedAt            time.Time   `json:"created_at" gorm:"autoCreateTime"`
}

func (a *AlertRule) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
