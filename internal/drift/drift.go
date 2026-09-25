package drift

import "github.com/example/go-data-profiler/internal/domain"

func Compare(a, b domain.TableProfile) domain.DriftReport {
	r := domain.DriftReport{}
	r.ScoreDelta = b.Quality.Score - a.Quality.Score
	if len(a.Columns) != len(b.Columns) {
		r.Changed = true
		r.ColumnChanges = append(r.ColumnChanges, "column count changed")
	}
	am := map[string]domain.ColumnProfile{}
	for _, c := range a.Columns {
		am[c.Name] = c
	}
	for _, c := range b.Columns {
		old, ok := am[c.Name]
		if !ok {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, "added: "+c.Name)
			continue
		}
		if old.DataType != c.DataType {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, "type: "+c.Name)
		}
		if old.DistinctCount != c.DistinctCount {
			r.Changed = true
			r.ColumnChanges = append(r.ColumnChanges, "distinct count: "+c.Name)
		}
	}
	return r
}
