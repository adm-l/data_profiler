package quality

import (
	"fmt"

	"github.com/example/go-data-profiler/internal/domain"
)

const completenessThreshold = 20.0

func Evaluate(p *domain.TableProfile) domain.QualityReport {
	var checks []domain.QualityCheck
	score := 100.0

	for _, c := range p.Columns {
		nullPct := percentage(c.NullCount, c.TotalRows)
		completenessPassed := nullPct < completenessThreshold
		checks = append(checks, domain.QualityCheck{
			Name:      "completeness:" + c.Name,
			Passed:    completenessPassed,
			Severity:  severity(completenessPassed, "fail"),
			Metric:    "null_percentage",
			Threshold: completenessThreshold,
			Actual:    nullPct,
			Details:   fmt.Sprintf("null %.2f%% (threshold %.2f%%)", nullPct, completenessThreshold),
		})
		if !completenessPassed {
			score -= 10
		}

		if isLikelyKey(c.Name) {
			unique := c.TotalRows > 0 && c.DistinctCount == c.TotalRows
			checks = append(checks, domain.QualityCheck{
				Name:      "uniqueness:" + c.Name,
				Passed:    unique,
				Severity:  severity(unique, "fail"),
				Metric:    "distinct_percentage",
				Threshold: 100,
				Actual:    c.DistinctPercentage,
				Details:   fmt.Sprintf("distinct=%d rows=%d", c.DistinctCount, c.TotalRows),
			})
			if !unique {
				score -= 5
			}
		}

		// A single distinct value across the table is usually a constant or near-useless column.
		if c.TotalRows > 0 && c.DistinctCount <= 1 {
			checks = append(checks, domain.QualityCheck{
				Name: "constant:" + c.Name, Passed: false, Severity: "warn",
				Metric: "distinct_count", Threshold: 1, Actual: float64(c.DistinctCount),
				Details: fmt.Sprintf("only %d distinct value(s) across %d rows", c.DistinctCount, c.TotalRows),
			})
		}

		// Very high cardinality is useful for identifiers, but can be suspicious for
		// fields expected to contain reusable categorical values.
		if c.TotalRows > 0 && c.DistinctPercentage >= 95 && !isLikelyKey(c.Name) {
			checks = append(checks, domain.QualityCheck{
				Name: "high_cardinality:" + c.Name, Passed: true, Severity: "info",
				Metric: "distinct_percentage", Threshold: 95, Actual: c.DistinctPercentage,
				Details: fmt.Sprintf("%.2f%% of rows are distinct", c.DistinctPercentage),
			})
		}

		// A dominant value can indicate a heavily imbalanced categorical field.
		if len(c.TopValues) > 0 && c.TopValues[0].Percentage >= 80 {
			checks = append(checks, domain.QualityCheck{
				Name: "dominant_value:" + c.Name, Passed: true, Severity: "info",
				Metric: "top_value_percentage", Threshold: 80, Actual: c.TopValues[0].Percentage,
				Details: fmt.Sprintf("top value represents %.2f%% of rows", c.TopValues[0].Percentage),
			})
		}

		if isTextColumn(c.DataType) {
			emptyPct := percentage(c.EmptyCount, c.TotalRows)
			checks = append(checks, domain.QualityCheck{
				Name:      "empty:" + c.Name,
				Passed:    c.EmptyCount == 0,
				Severity:  severity(c.EmptyCount == 0, "warn"),
				Metric:    "empty_percentage",
				Threshold: 0,
				Actual:    emptyPct,
				Details:   fmt.Sprintf("empty %.2f%%", emptyPct),
			})
			if c.EmptyCount > 0 {
				score -= 3
			}

			whitespacePct := percentage(c.WhitespaceCount, c.TotalRows)
			checks = append(checks, domain.QualityCheck{
				Name:      "whitespace:" + c.Name,
				Passed:    c.WhitespaceCount == 0,
				Severity:  severity(c.WhitespaceCount == 0, "warn"),
				Metric:    "whitespace_percentage",
				Threshold: 0,
				Actual:    whitespacePct,
				Details:   fmt.Sprintf("whitespace-only %.2f%%", whitespacePct),
			})
			if c.WhitespaceCount > 0 {
				score -= 2
			}
		}

		if isNumericColumn(c.DataType) && c.OutlierCount > 0 {
			checks = append(checks, domain.QualityCheck{
				Name: "outliers:" + c.Name, Passed: true, Severity: "warn",
				Metric: "outlier_percentage", Threshold: 0, Actual: c.OutlierPercentage,
				Details: fmt.Sprintf("3-sigma screening found %d outlier value(s) (%.2f%% of non-null values)", c.OutlierCount, c.OutlierPercentage),
			})
		}

		if isNumericColumn(c.DataType) && c.ZeroCount > 0 {
			zeroPct := percentage(c.ZeroCount, c.TotalRows-c.NullCount)
			checks = append(checks, domain.QualityCheck{
				Name:      "zero_values:" + c.Name,
				Passed:    true,
				Severity:  "info",
				Metric:    "zero_percentage",
				Threshold: 0,
				Actual:    zeroPct,
				Details:   fmt.Sprintf("zero values %d (%.2f%% of non-null values)", c.ZeroCount, zeroPct),
			})
		}
	}

	if p.DuplicateRowCount > 0 {
		checks = append(checks, domain.QualityCheck{Name: "duplicate_rows", Passed: false, Severity: "warn", Metric: "duplicate_row_percentage", Threshold: 0, Actual: p.DuplicateRowPercentage, Details: fmt.Sprintf("%d duplicate row(s) (%.2f%%)", p.DuplicateRowCount, p.DuplicateRowPercentage)})
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	return domain.QualityReport{Score: score, Checks: checks}
}

func percentage(count, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(count) * 100 / float64(total)
}

func severity(passed bool, failedSeverity string) string {
	if passed {
		return "pass"
	}
	return failedSeverity
}

func isLikelyKey(n string) bool {
	n = lower(n)
	return n == "id" || len(n) > 3 && (n[len(n)-3:] == "_id" || n[len(n)-2:] == "id")
}

func isTextColumn(t string) bool {
	t = lower(t)
	return containsAny(t, "char", "text", "string")
}

func isNumericColumn(t string) bool {
	t = lower(t)
	return containsAny(t, "int", "decimal", "numeric", "float", "double", "real", "money")
}

func containsAny(s string, values ...string) bool {
	for _, value := range values {
		for i := 0; i+len(value) <= len(s); i++ {
			if s[i:i+len(value)] == value {
				return true
			}
		}
	}
	return false
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
