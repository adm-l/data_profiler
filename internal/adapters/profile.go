package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/example/go-data-profiler/internal/domain"
	"math"
	"strings"
	"sync"
)

func ProfileTable(ctx context.Context, db *sql.DB, d Dialect, schema, table string, sample int) (*domain.TableProfile, error) {
	qt := d.Quote(schema) + "." + d.Quote(table)
	var total int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+qt).Scan(&total); err != nil {
		return nil, err
	}
	sampledFrom, sampled, err := d.SampleTableExpr(qt, sample, total)
	if err != nil {
		return nil, err
	}
	profileRows := total
	if sampled {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+sampledFrom).Scan(&profileRows); err != nil {
			return nil, err
		}
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
			p, e := profileColumn(ctx, db, d, sampledFrom, c, profileRows)
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
	return &domain.TableProfile{Schema: schema, Table: table, TotalRows: total, Sampled: sampled, SampleSize: sample, Columns: out}, nil
}
func profileColumn(ctx context.Context, db *sql.DB, d Dialect, qt string, c ColumnMeta, total int64) (domain.ColumnProfile, error) {
	qc := d.Quote(c.Name)
	text := d.CastText(qc)
	p := domain.ColumnProfile{Name: c.Name, DataType: c.DataType, Nullable: c.Nullable, TotalRows: total}

	// Combine common aggregates into one scan per column. Nullable SQL aggregates
	// are scanned through sql.Null* so empty/all-NULL columns do not fail profiling.
	q := "SELECT COUNT(*)-COUNT(" + qc + "),COUNT(DISTINCT " + qc + ")"
	if isText(c.DataType) {
		q += ",COALESCE(MIN(" + d.LengthExpr(qc) + "),0),COALESCE(MAX(" + d.LengthExpr(qc) + "),0),AVG(" + d.LengthExpr(qc) + "),COALESCE(SUM(CASE WHEN " + qc + " IS NOT NULL AND " + qc + "='' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN " + qc + " IS NOT NULL AND " + qc + "<>'' AND TRIM(" + qc + ")='' THEN 1 ELSE 0 END),0)"
	} else {
		q += ",0,0,0,0,0"
	}
	if isNumeric(c.DataType) {
		q += "," + d.CastText("MIN("+qc+")") + "," + d.CastText("MAX("+qc+")") + ",AVG(" + qc + "),COALESCE(SUM(CASE WHEN " + qc + "=0 THEN 1 ELSE 0 END),0)"
	} else if isDateTime(c.DataType) {
		q += "," + d.CastText("MIN("+qc+")") + "," + d.CastText("MAX("+qc+")") + ",NULL,0"
	} else {
		q += ",NULL,NULL,NULL,0"
	}

	var minLen, maxLen sql.NullInt64
	var avgLen sql.NullFloat64
	var minValue, maxValue sql.NullString
	var avgNum sql.NullFloat64
	if err := db.QueryRowContext(ctx, q+" FROM "+qt).Scan(
		&p.NullCount, &p.DistinctCount,
		&minLen, &maxLen, &avgLen, &p.EmptyCount, &p.WhitespaceCount,
		&minValue, &maxValue, &avgNum, &p.ZeroCount,
	); err != nil {
		return p, err
	}

	if total > 0 {
		p.NullPercentage = float64(p.NullCount) * 100 / float64(total)
		p.DistinctPercentage = float64(p.DistinctCount) * 100 / float64(total)
	}
	if minLen.Valid {
		p.MinLength = &minLen.Int64
	}
	if maxLen.Valid {
		p.MaxLength = &maxLen.Int64
	}
	if avgLen.Valid && !math.IsNaN(avgLen.Float64) {
		p.AvgLength = &avgLen.Float64
	}
	if minValue.Valid {
		p.Min = &minValue.String
	}
	if maxValue.Valid {
		p.Max = &maxValue.String
	}
	if avgNum.Valid && !math.IsNaN(avgNum.Float64) {
		p.Avg = &avgNum.Float64
	}

	q3 := "SELECT " + text + ",COUNT(*) FROM " + qt + " WHERE " + qc + " IS NOT NULL GROUP BY " + text + " ORDER BY COUNT(*) DESC LIMIT 10"
	rows, err := db.QueryContext(ctx, q3)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
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
		var stddev, median, p75, p90, p95, p99 sql.NullFloat64
		q5 := "SELECT " + d.NumericStatsExpr(qc) + " FROM " + qt
		if err := db.QueryRowContext(ctx, q5).Scan(&stddev, &median, &p75, &p90, &p95, &p99); err == nil {
			if stddev.Valid && !math.IsNaN(stddev.Float64) {
				p.StdDev = &stddev.Float64
			}
			p.Median = nullableFloat(median)
			p.P50 = p.Median
			p.P75 = nullableFloat(p75)
			p.P90 = nullableFloat(p90)
			p.P95 = nullableFloat(p95)
			p.P99 = nullableFloat(p99)
		}
	}
	return p, nil
}
func isText(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "char") || strings.Contains(t, "text") || strings.Contains(t, "string")
}
func isDateTime(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "date") || strings.Contains(t, "time")
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


func nullableFloat(v sql.NullFloat64) *float64 {
	if !v.Valid || math.IsNaN(v.Float64) {
		return nil
	}
	return &v.Float64
}
