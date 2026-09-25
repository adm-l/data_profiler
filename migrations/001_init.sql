CREATE TABLE IF NOT EXISTS jobs (
 id UUID PRIMARY KEY,
 status TEXT NOT NULL,
 error TEXT,
 request JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL,
 started_at TIMESTAMPTZ,
 finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_jobs_status_created ON jobs(status,created_at);
CREATE TABLE IF NOT EXISTS profiles (
 id BIGSERIAL PRIMARY KEY,
 job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 schema_name TEXT NOT NULL,
 table_name TEXT NOT NULL,
 payload JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_profiles_job_created ON profiles(job_id,created_at DESC);
