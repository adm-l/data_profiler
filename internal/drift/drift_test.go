package drift

import (
	"github.com/example/go-data-profiler/internal/domain"
	"testing"
)

func TestCompare(t *testing.T) {
	a := domain.TableProfile{Columns: []domain.ColumnProfile{{Name: "id", DataType: "integer", DistinctCount: 10}}}
	b := domain.TableProfile{Columns: []domain.ColumnProfile{{Name: "id", DataType: "text", DistinctCount: 10}}}
	if !Compare(a, b).Changed {
		t.Fatal("expected change")
	}
}
