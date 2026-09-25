package quality

import (
	"fmt"
	"github.com/example/go-data-profiler/internal/domain"
)

func Evaluate(p *domain.TableProfile) domain.QualityReport {
	var checks []domain.QualityCheck
	score := 100.0
	for _, c := range p.Columns {
		nullPct := 0.0
		if c.TotalRows > 0 {
			nullPct = float64(c.NullCount) * 100 / float64(c.TotalRows)
		}
		passed := nullPct < 20
		checks = append(checks, domain.QualityCheck{Name: "completeness:" + c.Name, Passed: passed, Details: fmt.Sprintf("null %.2f%%", nullPct)})
		if !passed {
			score -= 10
		}
		unique := c.DistinctCount == c.TotalRows && c.TotalRows > 0
		checks = append(checks, domain.QualityCheck{Name: "uniqueness:" + c.Name, Passed: !isLikelyKey(c.Name) || unique, Details: fmt.Sprintf("distinct=%d rows=%d", c.DistinctCount, c.TotalRows)})
		if isLikelyKey(c.Name) && !unique {
			score -= 5
		}
	}
	if score < 0 {
		score = 0
	}
	return domain.QualityReport{Score: score, Checks: checks}
}
func isLikelyKey(n string) bool {
	n = lower(n)
	return n == "id" || len(n) > 3 && (n[len(n)-3:] == "_id" || n[len(n)-2:] == "id")
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
