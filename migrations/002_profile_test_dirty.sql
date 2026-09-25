-- Demo/source data used to exercise Phase 2 profiling.
-- This migration is intentionally separate from the profiler's internal
-- jobs/profiles tables so the source data can be profiled through the API.

CREATE TABLE IF NOT EXISTS public.profile_test_dirty (
    id          INTEGER,
    name        VARCHAR(50),
    amount      NUMERIC(12,2),
    score       INTEGER,
    created_at  TIMESTAMP
);

INSERT INTO public.profile_test_dirty (id, name, amount, score, created_at)
SELECT *
FROM (VALUES
    (1,  'Alice'::VARCHAR(50),  100.50::NUMERIC(12,2), 10,  '2026-01-01 10:00:00'::TIMESTAMP),
    (2,  'Bob'::VARCHAR(50),     25.00::NUMERIC(12,2),  0,  '2026-01-02 11:00:00'::TIMESTAMP),
    (3,  'Alice'::VARCHAR(50),  -10.25::NUMERIC(12,2), -5,  '2026-01-03 12:00:00'::TIMESTAMP),
    (4,  ''::VARCHAR(50),         0.00::NUMERIC(12,2),  0,  '2026-01-04 13:00:00'::TIMESTAMP),
    (5,  '   '::VARCHAR(50),     50.75::NUMERIC(12,2), 20,  '2026-01-05 14:00:00'::TIMESTAMP),
    (6,  NULL::VARCHAR(50),       NULL::NUMERIC(12,2), NULL, NULL::TIMESTAMP),
    (7,  'Bob'::VARCHAR(50),      25.00::NUMERIC(12,2), 10,  '2026-01-07 16:00:00'::TIMESTAMP),
    (8,  'Carol'::VARCHAR(50),   999.99::NUMERIC(12,2), 30,  '2026-01-08 17:00:00'::TIMESTAMP),
    (9,  'Carol'::VARCHAR(50),     0.00::NUMERIC(12,2),  0,  '2026-01-09 18:00:00'::TIMESTAMP),
    (10, NULL::VARCHAR(50),       -1.50::NUMERIC(12,2), -10, '2026-01-10 19:00:00'::TIMESTAMP)
) AS seed(id, name, amount, score, created_at)
WHERE NOT EXISTS (SELECT 1 FROM public.profile_test_dirty);
