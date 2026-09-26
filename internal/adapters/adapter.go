package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/example/go-data-profiler/internal/domain"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
)

type Adapter interface {
	Close() error
	DiscoverColumns(context.Context, string, string) ([]ColumnMeta, error)
	ProfileTable(context.Context, string, string, int) (*domain.TableProfile, error)
}
type ColumnMeta struct {
	Name, DataType string
	Nullable       bool
}

func Open(ctx context.Context, typ, dsn string) (Adapter, error) {
	typ = strings.ToLower(typ)

	driver := map[string]string{
		"postgres":   "pgx",
		"postgresql": "pgx",
		"mysql":      "mysql",
		"sqlserver":  "sqlserver",
	}[typ]

	if driver == "" {
		if typ == "snowflake" {
			return nil, fmt.Errorf(
				"snowflake adapter boundary is defined but driver integration is not bundled; add gosnowflake in adapters/snowflake.go",
			)
		}

		return nil, fmt.Errorf("unsupported database type %q", typ)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	var dialect Dialect

	switch typ {
	case "postgres", "postgresql":
		dialect = PostgresDialect{}

	case "mysql":
		dialect = MySQLDialect{}

	case "sqlserver":
		dialect = SQLServerDialect{}

	default:
		_ = db.Close()
		return nil, fmt.Errorf("unsupported database type %q", typ)
	}

	return &SQLAdapter{
		db:      db,
		dialect: dialect,
	}, nil
}

type Dialect interface {
	Quote(string) string
	TablesQuery() string
	ColumnsQuery() string
	CountExpr(string) string
	LengthExpr(string) string
	CastText(string) string
	NumericStatsExpr(string) string
	NumericHistogramQuery(string, string) string
	FutureDateCountExpr(string) string
	DateRangeDaysExpr(string) string
	DateDistributionQuery(string, string) string
	PatternCountsQuery(string, string) string
	NumericCorrelationQuery(string, string, string) string
	RelationshipsQuery() string
	SampleTableExpr(string, int, int64) (string, bool, error)
}
type PostgresDialect struct{}

func (PostgresDialect) Quote(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
func (PostgresDialect) TablesQuery() string {
	return `SELECT table_schema,table_name FROM information_schema.tables WHERE table_schema=$1 AND table_type='BASE TABLE'`
}
func (PostgresDialect) ColumnsQuery() string {
	return `SELECT column_name,data_type,is_nullable FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position`
}
func (PostgresDialect) CountExpr(t string) string  { return `COUNT(*) FROM ` + t }
func (PostgresDialect) LengthExpr(c string) string { return `LENGTH(` + c + `)` }
func (PostgresDialect) CastText(c string) string { return `CAST(` + c + ` AS TEXT)` }
func (PostgresDialect) NumericStatsExpr(c string) string {
	return "STDDEV_POP(" + c + "),PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY " + c + "),PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY " + c + "),PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY " + c + "),PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY " + c + "),PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY " + c + ")"
}
func (PostgresDialect) NumericHistogramQuery(c, table string) string {
	bucket := "CASE WHEN (SELECT MIN(" + c + ") FROM " + table + ") = (SELECT MAX(" + c + ") FROM " + table + ") THEN 0 ELSE LEAST(9, FLOOR((" + c + " - (SELECT MIN(" + c + ") FROM " + table + ")) / NULLIF((SELECT MAX(" + c + ") FROM " + table + ") - (SELECT MIN(" + c + ") FROM " + table + ")),0) * 10)::int) END"
	return "SELECT " + bucket + ",COUNT(*) FROM " + table + " WHERE " + c + " IS NOT NULL GROUP BY " + bucket + " ORDER BY 1"
}
func (PostgresDialect) FutureDateCountExpr(c string) string { return "COALESCE(SUM(CASE WHEN " + c + " > CURRENT_TIMESTAMP THEN 1 ELSE 0 END),0)" }
func (PostgresDialect) DateRangeDaysExpr(c string) string { return "EXTRACT(EPOCH FROM (MAX(" + c + ") - MIN(" + c + ")))/86400.0" }
func (PostgresDialect) DateDistributionQuery(c, table string) string { return "SELECT TO_CHAR(DATE_TRUNC('month',"+c+"),'YYYY-MM'),COUNT(*) FROM "+table+" WHERE "+c+" IS NOT NULL GROUP BY 1 ORDER BY 1 DESC LIMIT 12" }
func (PostgresDialect) PatternCountsQuery(c, table string) string {
	return "SELECT CASE WHEN " + c + " ~* '^[^@\\s]+@[A-Za-z0-9]+\\.[A-Za-z]{2,}$' THEN 'email' WHEN " + c + " ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$' THEN 'uuid' WHEN " + c + " ~* '^https?://[^\\s]+$' THEN 'url' WHEN " + c + " ~ '^[0-9]{4}[-/][0-9]{1,2}[-/][0-9]{1,2}$' THEN 'date' WHEN " + c + " ~ '^[+-]?[0-9]+([.,][0-9]+)?$' THEN 'numeric_string' WHEN " + c + " ~ '^[0-9]+$' THEN 'integer_string' WHEN " + c + " ~ '^[A-Za-z]+$' THEN 'alphabetic' WHEN " + c + " ~ '^[A-Za-z0-9]+$' THEN 'alphanumeric' WHEN " + c + " ~ '^[^A-Za-z0-9]+$' THEN 'symbolic' ELSE 'mixed' END,COUNT(*) FROM " + table + " WHERE " + c + " IS NOT NULL GROUP BY 1 ORDER BY 2 DESC"
}
func (PostgresDialect) NumericCorrelationQuery(x, y, table string) string { return "SELECT CORR(" + x + "," + y + ") FROM " + table + " WHERE " + x + " IS NOT NULL AND " + y + " IS NOT NULL" }
func (PostgresDialect) RelationshipsQuery() string {
	return `SELECT kcu.column_name,kcu.foreign_table_schema,kcu.foreign_table_name,kcu.foreign_column_name
FROM information_schema.key_column_usage kcu
JOIN information_schema.table_constraints tc
  ON tc.constraint_schema=kcu.constraint_schema AND tc.constraint_name=kcu.constraint_name AND tc.table_name=kcu.table_name
WHERE tc.constraint_type='FOREIGN KEY' AND kcu.table_schema=$1 AND kcu.table_name=$2
ORDER BY kcu.ordinal_position`
}
func (PostgresDialect) SampleTableExpr(table string, sample int, total int64) (string, bool, error) {
	if sample <= 0 || total <= int64(sample) {
		return table, false, nil
	}
	pct := float64(sample) * 100 / float64(total)
	return "(SELECT * FROM " + table + " TABLESAMPLE SYSTEM (" + strconv.FormatFloat(pct, 'f', 6, 64) + ") REPEATABLE (42) LIMIT " + strconv.Itoa(sample) + ") AS profile_sample", true, nil
}

type MySQLDialect struct{}

func (MySQLDialect) Quote(s string) string { return "`" + strings.ReplaceAll(s, "`", "``") + "`" }
func (MySQLDialect) TablesQuery() string {
	return `SELECT table_schema,table_name FROM information_schema.tables WHERE table_schema=? AND table_type='BASE TABLE'`
}
func (MySQLDialect) ColumnsQuery() string {
	return `SELECT column_name,data_type,is_nullable FROM information_schema.columns WHERE table_schema=? AND table_name=? ORDER BY ordinal_position`
}
func (MySQLDialect) CountExpr(t string) string  { return `COUNT(*) FROM ` + t }
func (MySQLDialect) LengthExpr(c string) string { return `CHAR_LENGTH(` + c + `)` }
func (MySQLDialect) CastText(c string) string { return `CAST(` + c + ` AS CHAR)` }
func (MySQLDialect) NumericStatsExpr(c string) string {
	return "STDDEV_POP(" + c + "),NULL,NULL,NULL,NULL,NULL"
}
func (MySQLDialect) NumericHistogramQuery(c, table string) string {
	bucket := "CASE WHEN (SELECT MIN(" + c + ") FROM " + table + ") = (SELECT MAX(" + c + ") FROM " + table + ") THEN 0 ELSE LEAST(9, FLOOR((" + c + " - (SELECT MIN(" + c + ") FROM " + table + ")) / NULLIF((SELECT MAX(" + c + ") FROM " + table + ") - (SELECT MIN(" + c + ") FROM " + table + "),0) * 10)) END"
	return "SELECT " + bucket + ",COUNT(*) FROM " + table + " WHERE " + c + " IS NOT NULL GROUP BY " + bucket + " ORDER BY 1"
}
func (MySQLDialect) FutureDateCountExpr(c string) string { return "COALESCE(SUM(CASE WHEN " + c + " > CURRENT_TIMESTAMP THEN 1 ELSE 0 END),0)" }
func (MySQLDialect) DateRangeDaysExpr(c string) string { return "TIMESTAMPDIFF(SECOND,MIN(" + c + "),MAX(" + c + "))/86400.0" }
func (MySQLDialect) DateDistributionQuery(c, table string) string { return "SELECT DATE_FORMAT("+c+",'%Y-%m'),COUNT(*) FROM "+table+" WHERE "+c+" IS NOT NULL GROUP BY 1 ORDER BY 1 DESC LIMIT 12" }
func (MySQLDialect) PatternCountsQuery(c, table string) string {
	return "SELECT CASE WHEN " + c + " REGEXP '^[^@[:space:]]+@[A-Za-z0-9]+\\.[A-Za-z]{2,}$' THEN 'email' WHEN " + c + " REGEXP '^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[1-5][0-9A-Fa-f]{3}-[89AaBb][0-9A-Fa-f]{3}-[0-9A-Fa-f]{12}$' THEN 'uuid' WHEN " + c + " REGEXP '^https?://[^[:space:]]+$' THEN 'url' WHEN " + c + " REGEXP '^[0-9]{4}[-/][0-9]{1,2}[-/][0-9]{1,2}$' THEN 'date' WHEN " + c + " REGEXP '^[+-]?[0-9]+([.,][0-9]+)?$' THEN 'numeric_string' WHEN " + c + " REGEXP '^[0-9]+$' THEN 'integer_string' WHEN " + c + " REGEXP '^[A-Za-z]+$' THEN 'alphabetic' WHEN " + c + " REGEXP '^[A-Za-z0-9]+$' THEN 'alphanumeric' WHEN " + c + " REGEXP '^[^A-Za-z0-9]+$' THEN 'symbolic' ELSE 'mixed' END,COUNT(*) FROM " + table + " WHERE " + c + " IS NOT NULL GROUP BY 1 ORDER BY 2 DESC"
}
func (MySQLDialect) NumericCorrelationQuery(x, y, table string) string { return "SELECT (COUNT(*)*SUM("+x+"*"+y+")-SUM("+x+")*SUM("+y+"))/NULLIF(SQRT((COUNT(*)*SUM("+x+"*"+x+")-SUM("+x+")*SUM("+x+"))*(COUNT(*)*SUM("+y+"*"+y+")-SUM("+y+")*SUM("+y+"))),0) FROM "+table+" WHERE "+x+" IS NOT NULL AND "+y+" IS NOT NULL" }
func (MySQLDialect) RelationshipsQuery() string {
	return `SELECT kcu.column_name,kcu.referenced_table_schema,kcu.referenced_table_name,kcu.referenced_column_name
FROM information_schema.key_column_usage kcu
JOIN information_schema.table_constraints tc
  ON tc.constraint_schema=kcu.constraint_schema AND tc.constraint_name=kcu.constraint_name AND tc.table_name=kcu.table_name
WHERE tc.constraint_type='FOREIGN KEY' AND kcu.table_schema=? AND kcu.table_name=?
ORDER BY kcu.ordinal_position`
}
func (MySQLDialect) SampleTableExpr(table string, sample int, total int64) (string, bool, error) {
	if sample <= 0 || total <= int64(sample) {
		return table, false, nil
	}
	return "", false, fmt.Errorf("sampling is not yet supported for MySQL")
}

type SQLServerDialect struct{}

func (SQLServerDialect) Quote(s string) string { return "[" + strings.ReplaceAll(s, "]", "]]") + "]" }
func (SQLServerDialect) TablesQuery() string {
	return `SELECT TABLE_SCHEMA,TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA=@p1 AND TABLE_TYPE='BASE TABLE'`
}
func (SQLServerDialect) ColumnsQuery() string {
	return `SELECT COLUMN_NAME,DATA_TYPE,IS_NULLABLE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=@p1 AND TABLE_NAME=@p2 ORDER BY ORDINAL_POSITION`
}
func (SQLServerDialect) CountExpr(t string) string  { return `COUNT_BIG(*) FROM ` + t }
func (SQLServerDialect) LengthExpr(c string) string { return `LEN(` + c + `)` }
func (SQLServerDialect) CastText(c string) string { return `CAST(` + c + ` AS NVARCHAR(MAX))` }
func (SQLServerDialect) NumericStatsExpr(c string) string {
	return "STDEV(" + c + "),PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY " + c + ") OVER (),PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY " + c + ") OVER (),PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY " + c + ") OVER (),PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY " + c + ") OVER (),PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY " + c + ") OVER ()"
}
func (SQLServerDialect) NumericHistogramQuery(c, table string) string {
	bucket := "CASE WHEN (SELECT MIN(" + c + ") FROM " + table + ") = (SELECT MAX(" + c + ") FROM " + table + ") THEN 0 ELSE CASE WHEN FLOOR((" + c + " - (SELECT MIN(" + c + ") FROM " + table + ")) / NULLIF((SELECT MAX(" + c + ") FROM " + table + ") - (SELECT MIN(" + c + ") FROM " + table + "),0) * 10) > 9 THEN 9 ELSE CAST(FLOOR((" + c + " - (SELECT MIN(" + c + ") FROM " + table + ")) / NULLIF((SELECT MAX(" + c + ") FROM " + table + ") - (SELECT MIN(" + c + ") FROM " + table + "),0) * 10) AS INT) END END"
	return "SELECT " + bucket + ",COUNT(*) FROM " + table + " WHERE " + c + " IS NOT NULL GROUP BY " + bucket + " ORDER BY 1"
}
func (SQLServerDialect) FutureDateCountExpr(c string) string { return "COALESCE(SUM(CASE WHEN " + c + " > CURRENT_TIMESTAMP THEN 1 ELSE 0 END),0)" }
func (SQLServerDialect) DateRangeDaysExpr(c string) string { return "DATEDIFF_BIG(SECOND,MIN(" + c + "),MAX(" + c + "))/86400.0" }
func (SQLServerDialect) PatternCountsQuery(c, table string) string {
	return "SELECT CASE WHEN " + c + " LIKE '%@%.%' THEN 'email' WHEN " + c + " LIKE '________-____-____-____-____________' THEN 'uuid' WHEN " + c + " LIKE 'http://%' OR " + c + " LIKE 'https://%' THEN 'url' WHEN " + c + " LIKE '[0-9][0-9][0-9][0-9][- /][0-9][0-9][- /][0-9][0-9]' THEN 'date' WHEN " + c + " NOT LIKE '%[^0-9]%' THEN 'integer_string' WHEN " + c + " NOT LIKE '%[^A-Za-z]%' THEN 'alphabetic' WHEN " + c + " NOT LIKE '%[^A-Za-z0-9]%' THEN 'alphanumeric' ELSE 'mixed' END,COUNT_BIG(*) FROM " + table + " WHERE " + c + " IS NOT NULL GROUP BY 1 ORDER BY COUNT_BIG(*) DESC"
}
func (SQLServerDialect) NumericCorrelationQuery(x, y, table string) string { return "SELECT (COUNT_BIG(*)*SUM("+x+"*"+y+")-SUM("+x+")*SUM("+y+"))/NULLIF(SQRT((COUNT_BIG(*)*SUM("+x+"*"+x+")-SUM("+x+")*SUM("+x+"))*(COUNT_BIG(*)*SUM("+y+"*"+y+")-SUM("+y+")*SUM("+y+"))),0) FROM "+table+" WHERE "+x+" IS NOT NULL AND "+y+" IS NOT NULL" }
func (SQLServerDialect) DateDistributionQuery(c, table string) string { return "SELECT CONVERT(char(7),"+c+",120),COUNT_BIG(*) FROM "+table+" WHERE "+c+" IS NOT NULL GROUP BY CONVERT(char(7),"+c+",120) ORDER BY 1 DESC OFFSET 0 ROWS FETCH NEXT 12 ROWS ONLY" }
func (SQLServerDialect) RelationshipsQuery() string {
	return `SELECT kcu.COLUMN_NAME,kcu.REFERENCED_TABLE_SCHEMA,kcu.REFERENCED_TABLE_NAME,kcu.REFERENCED_COLUMN_NAME
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
  ON tc.CONSTRAINT_SCHEMA=kcu.CONSTRAINT_SCHEMA AND tc.CONSTRAINT_NAME=kcu.CONSTRAINT_NAME AND tc.TABLE_NAME=kcu.TABLE_NAME
WHERE tc.CONSTRAINT_TYPE='FOREIGN KEY' AND kcu.TABLE_SCHEMA=@p1 AND kcu.TABLE_NAME=@p2
ORDER BY kcu.ORDINAL_POSITION`
}
func (SQLServerDialect) SampleTableExpr(table string, sample int, total int64) (string, bool, error) {
	if sample <= 0 || total <= int64(sample) {
		return table, false, nil
	}
	return "", false, fmt.Errorf("sampling is not yet supported for SQL Server")
}

type SQLAdapter struct {
	db      *sql.DB
	dialect Dialect
}

func (a *SQLAdapter) Close() error { return a.db.Close() }
func (a *SQLAdapter) DiscoverColumns(ctx context.Context, schema, table string) ([]ColumnMeta, error) {
	q := a.dialect.ColumnsQuery()
	rows, err := a.db.QueryContext(ctx, q, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ColumnMeta
	for rows.Next() {
		var c ColumnMeta
		var n string
		if err := rows.Scan(&c.Name, &c.DataType, &n); err != nil {
			return nil, err
		}
		c.Nullable = strings.EqualFold(n, "YES")
		out = append(out, c)
	}
	return out, rows.Err()
}
func (a *SQLAdapter) ProfileTable(ctx context.Context, schema, table string, sample int) (*domain.TableProfile, error) {
	return ProfileTable(ctx, a.db, a.dialect, schema, table, sample)
}
