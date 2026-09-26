package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/example/go-data-profiler/internal/domain"
)

type Store interface {
	CreateJob(context.Context, *domain.Job) error
	UpdateJob(context.Context, *domain.Job) error
	GetJob(context.Context, string) (*domain.Job, error)
	GetJobByIdempotencyKey(context.Context, string) (*domain.Job, error)
	ListQueuedJobs(context.Context, int) ([]*domain.Job, error)
	SaveProfile(context.Context, string, *domain.TableProfile) error
	GetLatestProfile(context.Context, string) (*domain.TableProfile, error)
	GetPreviousProfile(context.Context, string, string) (*domain.TableProfile, error)
	GetProfileHistory(context.Context, string, string, int) ([]domain.TableProfile, error)
}
type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }
func (s *PostgresStore) CreateJob(ctx context.Context, j *domain.Job) error {
	b, _ := json.Marshal(j.Request)
	_, e := s.db.ExecContext(ctx, `INSERT INTO jobs(id,status,request,created_at,idempotency_key) VALUES($1,$2,$3,$4,$5)`, j.ID, j.Status, b, j.CreatedAt, nullString(j.IdempotencyKey))
	return e
}
func (s *PostgresStore) UpdateJob(ctx context.Context, j *domain.Job) error {
	_, e := s.db.ExecContext(ctx, `UPDATE jobs SET status=$2,error=$3,started_at=$4,finished_at=$5 WHERE id=$1`, j.ID, j.Status, j.Error, j.StartedAt, j.FinishedAt)
	return e
}
func (s *PostgresStore) ListQueuedJobs(ctx context.Context, limit int) ([]*domain.Job, error) {
	if limit <= 0 || limit > 1000 { limit = 1000 }
	rows, err := s.db.QueryContext(ctx, `SELECT id,status,COALESCE(error,''),request,created_at,started_at,finished_at FROM jobs WHERE status='queued' ORDER BY created_at LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]*domain.Job, 0, limit)
	for rows.Next() {
		var j domain.Job
		var b []byte
		if err := rows.Scan(&j.ID,&j.Status,&j.Error,&b,&j.CreatedAt,&j.StartedAt,&j.FinishedAt); err != nil { return nil, err }
		if err := json.Unmarshal(b,&j.Request); err != nil { return nil, err }
		out = append(out,&j)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetJobByIdempotencyKey(ctx context.Context, key string) (*domain.Job, error) {
	var j domain.Job
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,status,COALESCE(error,''),request,created_at,started_at,finished_at FROM jobs WHERE idempotency_key=$1`, key).Scan(&j.ID, &j.Status, &j.Error, &b, &j.CreatedAt, &j.StartedAt, &j.FinishedAt)
	if err != nil { return nil, err }
	if err = json.Unmarshal(b, &j.Request); err != nil { return nil, err }
	j.IdempotencyKey = key
	return &j, nil
}

func (s *PostgresStore) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	var j domain.Job
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,status,COALESCE(error,''),request,created_at,started_at,finished_at FROM jobs WHERE id=$1`, id).Scan(&j.ID, &j.Status, &j.Error, &b, &j.CreatedAt, &j.StartedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &j.Request); err != nil {
		return nil, err
	}
	return &j, nil
}
func (s *PostgresStore) SaveProfile(ctx context.Context, id string, p *domain.TableProfile) error {
	b, _ := json.Marshal(p)
	_, e := s.db.ExecContext(ctx, `INSERT INTO profiles(job_id,schema_name,table_name,payload,created_at) VALUES($1,$2,$3,$4,$5)`, id, p.Schema, p.Table, b, p.CreatedAt)
	return e
}
func (s *PostgresStore) GetLatestProfile(ctx context.Context, id string) (*domain.TableProfile, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM profiles WHERE job_id=$1 ORDER BY created_at DESC LIMIT 1`, id).Scan(&b)
	if err != nil {
		return nil, err
	}
	var p domain.TableProfile
	err = json.Unmarshal(b, &p)
	return &p, err
}


func (s *PostgresStore) GetPreviousProfile(ctx context.Context, schema, table string) (*domain.TableProfile, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM profiles WHERE schema_name=$1 AND table_name=$2 ORDER BY created_at DESC LIMIT 1`, schema, table).Scan(&b)
	if err != nil {
		return nil, err
	}
	var p domain.TableProfile
	err = json.Unmarshal(b, &p)
	return &p, err
}


func (s *PostgresStore) GetProfileHistory(ctx context.Context, schema, table string, limit int) ([]domain.TableProfile, error) {
	if limit <= 0 || limit > 100 { limit = 20 }
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM profiles WHERE schema_name=$1 AND table_name=$2 ORDER BY created_at DESC LIMIT $3", schema, table, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	out := make([]domain.TableProfile, 0, limit)
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil { return nil, err }
		var p domain.TableProfile
		if err := json.Unmarshal(b, &p); err != nil { return nil, err }
		out = append(out, p)
	}
	return out, rows.Err()
}

func nullString(v string) any {
	if v == "" { return nil }
	return v
}
