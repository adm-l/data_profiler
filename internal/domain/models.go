package domain

import "time"

type SourceConfig struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}
type QualityRules struct {
	NullPercentageThreshold      *float64 `json:"null_percentage_threshold,omitempty"`
	DominantValuePercentage      *float64 `json:"dominant_value_percentage,omitempty"`
	HighCardinalityPercentage    *float64 `json:"high_cardinality_percentage,omitempty"`
	AllowEmpty                   *bool    `json:"allow_empty,omitempty"`
	AllowWhitespace              *bool    `json:"allow_whitespace,omitempty"`
}
type ProfileRequest struct {
	Source     SourceConfig `json:"source"`
	Schema     string       `json:"schema"`
	Table      string       `json:"table"`
	SampleSize   int          `json:"sample_size,omitempty"`
	QualityRules *QualityRules `json:"quality_rules,omitempty"`
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
	Name               string       `json:"name"`
	DataType           string       `json:"data_type"`
	Nullable           bool         `json:"nullable"`
	TotalRows          int64        `json:"total_rows"`
	NullCount          int64        `json:"null_count"`
	NullPercentage     float64      `json:"null_percentage"`
	DistinctCount      int64        `json:"distinct_count"`
	DistinctPercentage float64      `json:"distinct_percentage"`
	EmptyCount         int64        `json:"empty_count"`
	WhitespaceCount    int64        `json:"whitespace_count"`
	ZeroCount          int64        `json:"zero_count,omitempty"`
	Min                *string      `json:"min,omitempty"`
	Max                *string      `json:"max,omitempty"`
	Avg                *float64     `json:"avg,omitempty"`
	StdDev             *float64     `json:"stddev,omitempty"`
	Median             *float64     `json:"median,omitempty"`
	P50                *float64     `json:"p50,omitempty"`
	P75                *float64     `json:"p75,omitempty"`
	P90                *float64     `json:"p90,omitempty"`
	P95                *float64     `json:"p95,omitempty"`
	P99                *float64     `json:"p99,omitempty"`
	MinLength          *int64       `json:"min_length,omitempty"`
	MaxLength          *int64       `json:"max_length,omitempty"`
	AvgLength          *float64     `json:"avg_length,omitempty"`
	TopValues          []ValueCount `json:"top_values,omitempty"`
	PII                string       `json:"pii,omitempty"`
	PIIConfidence      float64      `json:"pii_confidence,omitempty"`
	OutlierCount       int64        `json:"outlier_count,omitempty"`
	OutlierPercentage  float64      `json:"outlier_percentage,omitempty"`
	Histogram          []HistogramBin `json:"histogram,omitempty"`
	Entropy             *float64       `json:"entropy,omitempty"`
	NormalizedEntropy   *float64       `json:"normalized_entropy,omitempty"`
}
type HistogramBin struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}
type ValueCount struct {
	Value      string  `json:"value"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}
type TableProfile struct {
	Schema     string          `json:"schema"`
	Table      string          `json:"table"`
	TotalRows  int64           `json:"total_rows"`
	Sampled    bool            `json:"sampled"`
	SampleSize int             `json:"sample_size,omitempty"`
	Columns    []ColumnProfile `json:"columns"`
	DuplicateRowCount       int64   `json:"duplicate_row_count,omitempty"`
	DuplicateRowPercentage  float64 `json:"duplicate_row_percentage,omitempty"`
	Quality    QualityReport   `json:"quality"`
	Drift      *DriftReport    `json:"drift,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
type QualityReport struct {
	Score  float64        `json:"score"`
	Checks []QualityCheck `json:"checks"`
}
type QualityCheck struct {
	Name      string  `json:"name"`
	Passed    bool    `json:"passed"`
	Severity  string  `json:"severity"`
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
	Actual    float64 `json:"actual"`
	Details   string  `json:"details"`
}
type DriftReport struct {
	Changed       bool     `json:"changed"`
	ScoreDelta    float64  `json:"score_delta"`
	ColumnChanges []string `json:"column_changes"`
}
