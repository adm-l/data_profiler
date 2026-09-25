package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/example/go-data-profiler/internal/domain"
	"math"
	"strconv"
	"strings"
	"sync"
)

func ProfileTable(ctx context.Context, db *sql.DB, d Dialect, schema, table string, sample int) (*domain.TableProfile, error) {
	qt := d.Quote(schema) + "." + d.Quote(table)
	var total int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+qt).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, d.ColumnsQuery(), schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []ColumnMeta
	for rows.Next() {
		var c ColumnMeta
		var n string
		if err := rows.Scan(&c.Name, &c.DataType, &n); err != nil {
			return nil, err
		}
		c.Nullable = strings.EqualFold(n, "YES")
		cols = append(cols, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out := make([]domain.ColumnProfile, len(cols))
	workers := 4
	if workers > len(cols) {
		workers = len(cols)
	}
	if workers == 0 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var first error
	var mu sync.Mutex
	for i, c := range cols {
		i, c := i, c
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			p, e := profileColumn(ctx, db, d, qt, c, total)
			mu.Lock()
			defer mu.Unlock()
			if e != nil && first == nil {
				first = e
			} else {
				out[i] = p
			}
		}()
	}
	wg.Wait()
	if first != nil {
		return nil, first
	}
	return &domain.TableProfile{Schema: schema, Table: table, TotalRows: total, Columns: out}, nil
}
func profileColumn(ctx context.Context, db *sql.DB, d Dialect, qt string, c ColumnMeta, total int64) (domain.ColumnProfile, error) {
	qc := d.Quote(c.Name)
	text := d.CastText(qc)
	p := domain.ColumnProfile{Name: c.Name, DataType: c.DataType, Nullable: c.Nullable, TotalRows: total}
	q := "SELECT COUNT(*)-COUNT(" + qc + "), COUNT(DISTINCT " + text + ") FROM " + qt
	if err := db.QueryRowContext(ctx, q).Scan(&p.NullCount, &p.DistinctCount); err != nil {
		return p, err
	}
	if total > 0 {
		p.NullPercentage = float64(p.NullCount) * 100 / float64(total)
		p.DistinctPercentage = float64(p.DistinctCount) * 100 / float64(total)
	}
	if isText(c.DataType) {
		var min, max int64
		var avg sql.NullFloat64
		q2 := "SELECT COALESCE(MIN(" + d.LengthExpr(qc) + "),0),COALESCE(MAX(" + d.LengthExpr(qc) + "),0),AVG(" + d.LengthExpr(qc) + "),SUM(CASE WHEN " + qc + " IS NOT NULL AND " + qc + "='' THEN 1 ELSE 0 END) FROM " + qt
		if err := db.QueryRowContext(ctx, q2).Scan(&min, &max, &avg, &p.EmptyCount); err == nil {
			p.MinLength = &min
			p.MaxLength = &max
			if avg.Valid && !math.IsNaN(avg.Float64) {
				p.Avg = &avg.Float64
			}
		}
	}
	q3 := "SELECT " + text + ",COUNT(*) FROM " + qt + " WHERE " + qc + " IS NOT NULL GROUP BY " + text + " ORDER BY COUNT(*) DESC"
	rows, err := db.QueryContext(ctx, q3)
	if err == nil {
		defer rows.Close()
		for rows.Next() && len(p.TopValues) < 10 {
			var v string
			var n int64
			if rows.Scan(&v, &n) == nil {
				pct := 0.0
				if total > 0 {
					pct = float64(n) * 100 / float64(total)
				}
				p.TopValues = append(p.TopValues, domain.ValueCount{Value: v, Count: n, Percentage: pct})
			}
		}
	}
	if isNumeric(c.DataType) {
		var min, max, avg sql.NullFloat64
		q4 := "SELECT MIN(" + qc + "),MAX(" + qc + "),AVG(" + qc + ") FROM " + qt
		if err := db.QueryRowContext(ctx, q4).Scan(&min, &max, &avg); err == nil {
			if min.Valid {
				s := strconv.FormatFloat(min.Float64, 'f', -1, 64)
				p.Min = &s
			}
			if max.Valid {
				s := strconv.FormatFloat(max.Float64, 'f', -1, 64)
				p.Max = &s
			}
			if avg.Valid && !math.IsNaN(avg.Float64) {
				p.Avg = &avg.Float64
			}
		}
	}
	return p, nil
}
func isText(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "char") || strings.Contains(t, "text") || strings.Contains(t, "string")
}
func isNumeric(t string) bool {
	t = strings.ToLower(t)
	for _, x := range []string{"int", "decimal", "numeric", "float", "double", "real", "money"} {
		if strings.Contains(t, x) {
			return true
		}
	}
	return false
}

var _ = fmt.Sprintf
