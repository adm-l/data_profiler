package adapters

import (
	"context"
	"database/sql"
	"github.com/example/go-data-profiler/internal/domain"
	"math"
	"regexp"
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
	result := &domain.TableProfile{Schema: schema, Table: table, TotalRows: total, Sampled: sampled, SampleSize: sample, Columns: out}
	if len(cols) > 0 {
		groupCols := make([]string, len(cols))
		for i, col := range cols { groupCols[i] = d.Quote(col.Name) }
		qdup := "SELECT COALESCE(SUM(n-1),0) FROM (SELECT COUNT(*) AS n FROM " + sampledFrom + " GROUP BY " + strings.Join(groupCols, ",") + ") duplicates"
		var dup sql.NullInt64
		if err := db.QueryRowContext(ctx, qdup).Scan(&dup); err == nil && dup.Valid {
			result.DuplicateRowCount = dup.Int64
			if profileRows > 0 { result.DuplicateRowPercentage = float64(dup.Int64) * 100 / float64(profileRows) }
		}
	}
	return result, nil
}
func profileColumn(ctx context.Context, db *sql.DB, d Dialect, qt string, c ColumnMeta, total int64) (domain.ColumnProfile, error) {
	qc := d.Quote(c.Name)
	text := d.CastText(qc)
	p := domain.ColumnProfile{Name: c.Name, DataType: c.DataType, Nullable: c.Nullable, TotalRows: total}

	// Combine common aggregates into one scan per column. Nullable SQL aggregates
	// are scanned through sql.Null* so empty/all-NULL columns do not fail profiling.
	q := "SELECT COUNT(*)-COUNT(" + qc + "),COUNT(DISTINCT " + qc + ")"
	if isText(c.DataType) {
		q += ",COALESCE(MIN(" + d.LengthExpr(qc) + "),0),COALESCE(MAX(" + d.LengthExpr(qc) + "),0),AVG(" + d.LengthExpr(qc) + "),COALESCE(SUM(CASE WHEN " + qc + " IS NOT NULL AND " + qc + "='' THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN " + qc + " IS NOT NULL AND " + qc + "<>'' AND REGEXP_REPLACE(" + text + ", '[[:space:]]', '', 'g')='' THEN 1 ELSE 0 END),0)"
	} else {
		q += ",0,0,0,0,0"
	}
	if isNumeric(c.DataType) {
		q += "," + d.CastText("MIN("+qc+")") + "," + d.CastText("MAX("+qc+")") + ",AVG(" + qc + "),COALESCE(SUM(CASE WHEN " + qc + "=0 THEN 1 ELSE 0 END),0)"
	} else if isDateTime(c.DataType) {
		q += "," + d.CastText("MIN("+qc+")") + "," + d.CastText("MAX("+qc+")") + ",NULL,0," + d.FutureDateCountExpr(qc) + "," + d.DateRangeDaysExpr(qc)
	} else {
		q += ",NULL,NULL,NULL,0,0,0"
	}

	var minLen, maxLen sql.NullInt64
	var avgLen sql.NullFloat64
	var minValue, maxValue sql.NullString
	var avgNum sql.NullFloat64
	var futureDateCount sql.NullInt64
	var dateRangeDays sql.NullFloat64
	if err := db.QueryRowContext(ctx, q+" FROM "+qt).Scan(
		&p.NullCount, &p.DistinctCount,
		&minLen, &maxLen, &avgLen, &p.EmptyCount, &p.WhitespaceCount,
		&minValue, &maxValue, &avgNum, &p.ZeroCount, &futureDateCount, &dateRangeDays,
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
	if isDateTime(c.DataType) {
		if futureDateCount.Valid {
			p.FutureDateCount = futureDateCount.Int64
			nonNull := total - p.NullCount
			if nonNull > 0 { p.FutureDatePercentage = float64(p.FutureDateCount) * 100 / float64(nonNull) }
		}
		if dateRangeDays.Valid && !math.IsNaN(dateRangeDays.Float64) {
			p.DateRangeDays = &dateRangeDays.Float64
		}
		addDateDistribution(ctx, db, d, qt, qc, total, &p)
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

	if isText(c.DataType) {
		addStringPatterns(ctx, db, d, qt, qc, total, &p)
	}

	if isText(c.DataType) && p.DistinctCount > 1 && p.DistinctCount <= 10000 {
		addTextEntropy(ctx, db, qt, qc, total, p.DistinctCount, &p)
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
			addNumericHistogram(ctx, db, d, qt, qc, total, &p)
			qOut := "SELECT COUNT(*) FROM " + qt + " WHERE " + qc + " IS NOT NULL AND ABS(" + qc + " - (SELECT AVG(" + qc + ") FROM " + qt + ")) > 3 * (SELECT STDDEV_POP(" + qc + ") FROM " + qt + ")"
			var outliers sql.NullInt64
			if err := db.QueryRowContext(ctx, qOut).Scan(&outliers); err == nil && outliers.Valid {
				p.OutlierCount = outliers.Int64
				nonNull := total - p.NullCount
				if nonNull > 0 { p.OutlierPercentage = float64(p.OutlierCount) * 100 / float64(nonNull) }
			}
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



func nullableFloat(v sql.NullFloat64) *float64 {
	if !v.Valid || math.IsNaN(v.Float64) {
		return nil
	}
	return &v.Float64
}

func addNumericHistogram(ctx context.Context, db *sql.DB, d Dialect, table, column string, total int64, p *domain.ColumnProfile) {
	if p.Min == nil || p.Max == nil || total-p.NullCount <= 0 { return }
	min, errMin := strconv.ParseFloat(*p.Min, 64)
	max, errMax := strconv.ParseFloat(*p.Max, 64)
	if errMin != nil || errMax != nil { return }
	nonNull := total - p.NullCount
	if min == max {
		p.Histogram = []domain.HistogramBin{{Lower:min, Upper:max, Count:nonNull, Percentage:100}}
		return
	}
	counts := make([]int64, 10)
	rows, err := db.QueryContext(ctx, d.NumericHistogramQuery(column, table))
	if err != nil { return }
	defer rows.Close()
	for rows.Next() {
		var bucket int
		var count int64
		if err := rows.Scan(&bucket, &count); err == nil && bucket >= 0 && bucket < 10 { counts[bucket] = count }
	}
	width := (max-min)/10
	for i,count := range counts {
		if count == 0 { continue }
		lower := min+float64(i)*width
		upper := min+float64(i+1)*width
		if i == 9 { upper=max }
		p.Histogram=append(p.Histogram, domain.HistogramBin{Lower:lower,Upper:upper,Count:count,Percentage:float64(count)*100/float64(nonNull)})
	}
}


func addTextEntropy(ctx context.Context, db *sql.DB, table, column string, total, distinct int64, p *domain.ColumnProfile) {
	if total-p.NullCount <= 0 || distinct <= 1 {
		return
	}
	rows, err := db.QueryContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE "+column+" IS NOT NULL GROUP BY "+column)
	if err != nil {
		return
	}
	defer rows.Close()

	nonNull := total - p.NullCount
	var entropy float64
	for rows.Next() {
		var count int64
		if err := rows.Scan(&count); err != nil || count <= 0 {
			continue
		}
		prob := float64(count) / float64(nonNull)
		entropy -= prob * math.Log2(prob)
	}
	if err := rows.Err(); err != nil {
		return
	}
	p.Entropy = &entropy
	maxEntropy := math.Log2(float64(distinct))
	if maxEntropy > 0 {
		normalized := entropy / maxEntropy
		p.NormalizedEntropy = &normalized
	}
}

func addDateDistribution(ctx context.Context, db *sql.DB, d Dialect, table, column string, total int64, p *domain.ColumnProfile) {
	rows, err := db.QueryContext(ctx, d.DateDistributionQuery(column, table))
	if err != nil { return }
	defer rows.Close()
	for rows.Next() {
		var period string
		var count int64
		if err := rows.Scan(&period, &count); err != nil { return }
		pct := 0.0
		if total > 0 { pct = float64(count) * 100 / float64(total) }
		p.DateDistribution = append(p.DateDistribution, domain.DateDistribution{Period: period, Count: count, Percentage: pct})
	}
}


var (
 patternEmailRE = regexp.MustCompile(`(?i)^[^@\\s]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$`)
 patternUUIDRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
 patternURLRE = regexp.MustCompile(`(?i)^https?://[^\\s]+$`)
 patternDateRE = regexp.MustCompile(`^\\d{4}[-/]\\d{1,2}[-/]\\d{1,2}$`)
 patternNumericRE = regexp.MustCompile(`^[+-]?\\d+(?:[.,]\\d+)?$`)
)

func addStringPatterns(p *domain.ColumnProfile) {
 if len(p.TopValues) == 0 || p.TotalRows <= 0 { return }
 counts := map[string]int64{}
 for _, v := range p.TopValues {
  label := classifyStringPattern(v.Value)
  if label != "" { counts[label] += v.Count }
 }
 var patterns []domain.PatternCount
 for label, count := range counts {
  patterns = append(patterns, domain.PatternCount{Pattern: label, Count: count, Percentage: float64(count)*100/float64(p.TotalRows)})
 }
 // Stable order makes API output deterministic.
 for i := 0; i < len(patterns); i++ {
  for j := i+1; j < len(patterns); j++ {
   if patterns[j].Count > patterns[i].Count { patterns[i], patterns[j] = patterns[j], patterns[i] }
  }
 }
 p.Patterns = patterns
}

func classifyStringPattern(s string) string {
 s = strings.TrimSpace(s)
 switch {
 case s == "": return "empty"
 case patternEmailRE.MatchString(s): return "email"
 case patternUUIDRE.MatchString(s): return "uuid"
 case patternURLRE.MatchString(s): return "url"
 case patternDateRE.MatchString(s): return "date"
 case patternNumericRE.MatchString(s): return "numeric_string"
 }
 hasLetter, hasDigit, hasOther := false, false, false
 for _, r := range s {
  switch {
  case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z': hasLetter = true
  case r >= '0' && r <= '9': hasDigit = true
  default: hasOther = true
  }
 }
 switch {
 case hasLetter && hasDigit && !hasOther: return "alphanumeric"
 case hasLetter && !hasDigit && !hasOther: return "alphabetic"
 case hasDigit && !hasLetter && !hasOther: return "integer_string"
 case hasOther && !hasLetter && !hasDigit: return "symbolic"
 default: return "mixed"
 }
}
