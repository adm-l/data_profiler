package domain

import "time"

type SourceConfig struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}
type ProfileRequest struct {
	Source     SourceConfig `json:"source"`
	Schema     string       `json:"schema"`
	Table      string       `json:"table"`
	SampleSize int          `json:"sample_size,omitempty"`
}
type Job struct {
	ID         string         `json:"id"`
	Status     string         `json:"status"`
	Error      string         `json:"error,omitempty"`
	Request    ProfileRequest `json:"request"`
	CreatedAt  time.Time      `json:"created_at"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
}
type ColumnProfile struct {
	Name          string       `json:"name"`
	DataType      string       `json:"data_type"`
	Nullable      bool         `json:"nullable"`
	TotalRows     int64        `json:"total_rows"`
	NullCount     int64        `json:"null_count"`
	DistinctCount int64        `json:"distinct_count"`
	EmptyCount    int64        `json:"empty_count"`
	Min           *string      `json:"min,omitempty"`
	Max           *string      `json:"max,omitempty"`
	Avg           *float64     `json:"avg,omitempty"`
	MinLength     *int64       `json:"min_length,omitempty"`
	MaxLength     *int64       `json:"max_length,omitempty"`
	TopValues     []ValueCount `json:"top_values,omitempty"`
	PII           string       `json:"pii,omitempty"`
}
type ValueCount struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}
type TableProfile struct {
	Schema    string          `json:"schema"`
	Table     string          `json:"table"`
	TotalRows int64           `json:"total_rows"`
	Columns   []ColumnProfile `json:"columns"`
	Quality   QualityReport   `json:"quality"`
	CreatedAt time.Time       `json:"created_at"`
}
type QualityReport struct {
	Score  float64        `json:"score"`
	Checks []QualityCheck `json:"checks"`
}
type QualityCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details"`
}
type DriftReport struct {
	Changed       bool     `json:"changed"`
	ScoreDelta    float64  `json:"score_delta"`
	ColumnChanges []string `json:"column_changes"`
}
