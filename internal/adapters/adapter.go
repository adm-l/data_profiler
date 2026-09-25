package adapters

import (
	"context"
	"database/sql"
	"fmt"
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
func (PostgresDialect) CastText(c string) string   { return `CAST(` + c + ` AS TEXT)` }

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
func (MySQLDialect) CastText(c string) string   { return `CAST(` + c + ` AS CHAR)` }

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
func (SQLServerDialect) CastText(c string) string   { return `CAST(` + c + ` AS NVARCHAR(MAX))` }

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
