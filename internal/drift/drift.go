package drift

import (
	"fmt"

	"github.com/example/go-data-profiler/internal/domain"
)

func Compare(a, b domain.TableProfile) domain.DriftReport {
	r := domain.DriftReport{ScoreDelta: b.Quality.Score - a.Quality.Score, ColumnChanges: []string{}}

	if r.ScoreDelta != 0 {
		r.Changed = true
		r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("quality score changed: %.2f -> %.2f", a.Quality.Score, b.Quality.Score))
	}

	oldCols := make(map[string]domain.ColumnProfile, len(a.Columns))
	newCols := make(map[string]domain.ColumnProfile, len(b.Columns))
	for _, c := range a.Columns {
		oldCols[c.Name] = c
	}
	for _, c := range b.Columns {
		newCols[c.Name] = c
	}

	for name, old := range oldCols {
		current, ok := newCols[name]
		if !ok {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, "removed: "+name)
			continue
		}

		if old.DataType != current.DataType {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("type changed: %s (%s -> %s)", name, old.DataType, current.DataType))
		}
		if old.Nullable != current.Nullable {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, "nullability changed: "+name)
		}
		if old.DistinctCount != current.DistinctCount {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("distinct count changed: %s (%d -> %d)", name, old.DistinctCount, current.DistinctCount))
		}
		if significantDelta(old.NullPercentage, current.NullPercentage, 10) {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("null percentage changed: %s (%.2f%% -> %.2f%%)", name, old.NullPercentage, current.NullPercentage))
		}
		if significantDelta(old.DistinctPercentage, current.DistinctPercentage, 10) {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("distinct percentage changed: %s (%.2f%% -> %.2f%%)", name, old.DistinctPercentage, current.DistinctPercentage))
		}
		if significantDelta(old.OutlierPercentage, current.OutlierPercentage, 5) {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("outlier percentage changed: %s (%.2f%% -> %.2f%%)", name, old.OutlierPercentage, current.OutlierPercentage))
		}
		if old.PII != current.PII {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("PII classification changed: %s (%s -> %s)", name, old.PII, current.PII))
		}
	}

	for name := range newCols {
		if _, ok := oldCols[name]; !ok {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, "added: "+name)
		}
	}

	if a.DuplicateRowCount != b.DuplicateRowCount {
		r.Changed = true
		r.ColumnChanges = append(r.ColumnChanges, fmt.Sprintf("duplicate rows changed: %d -> %d", a.DuplicateRowCount, b.DuplicateRowCount))
	}

	return r
}

func significantDelta(old, current, threshold float64) bool {
	delta := current - old
	if delta < 0 {
		delta = -delta
	}
	return delta >= threshold
}
