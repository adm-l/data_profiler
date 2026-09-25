package quality

import (
	"github.com/example/go-data-profiler/internal/domain"
	"testing"
)

func TestEvaluate(t *testing.T) {
	p := &domain.TableProfile{TotalRows: 100, Columns: []domain.ColumnProfile{{Name: "id", TotalRows: 100, NullCount: 0, DistinctCount: 100}, {Name: "email", TotalRows: 100, NullCount: 50, DistinctCount: 90}}}
	r := Evaluate(p)
	if r.Score >= 100 {
		t.Fatal("expected deductions")
	}
}
