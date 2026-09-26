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

func TestCompareNoChangeHasZeroScoreDeltaAndEmptyChanges(t *testing.T) {
	a := domain.TableProfile{Quality: domain.QualityReport{Score: 98}, Columns: []domain.ColumnProfile{{Name: "id", DataType: "integer", DistinctCount: 10}}}
	b := domain.TableProfile{Quality: domain.QualityReport{Score: 98}, Columns: []domain.ColumnProfile{{Name: "id", DataType: "integer", DistinctCount: 10}}}
	r := Compare(a, b)
	if r.Changed || r.ScoreDelta != 0 || len(r.ColumnChanges) != 0 {
		t.Fatalf("expected no drift, got changed=%v score_delta=%v changes=%v", r.Changed, r.ScoreDelta, r.ColumnChanges)
	}
}

func TestCompareQualityScoreChangeIsDrift(t *testing.T) {
	a := domain.TableProfile{Quality: domain.QualityReport{Score: 98}}
	b := domain.TableProfile{Quality: domain.QualityReport{Score: 78}}
	r := Compare(a, b)
	if !r.Changed || r.ScoreDelta != -20 || len(r.ColumnChanges) != 1 {
		t.Fatalf("expected quality drift, got changed=%v score_delta=%v changes=%v", r.Changed, r.ScoreDelta, r.ColumnChanges)
	}
}
