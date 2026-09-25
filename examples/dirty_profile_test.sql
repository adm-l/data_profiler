-- Deliberately dirty dataset for validating Phase 2 column profiling.
-- PostgreSQL
--
-- Covers:
--   NULL values
--   empty strings
--   whitespace-only strings
--   duplicate values
--   zero and negative numeric values
--   decimal numeric values
--   timestamps
--
-- Run:
--   psql "$DATABASE_URL" -f examples/dirty_profile_test.sql

DROP TABLE IF EXISTS public.profile_test_dirty;

CREATE TABLE public.profile_test_dirty (
    id          INTEGER,
    name        VARCHAR(50),
    amount      NUMERIC(12,2),
    score       INTEGER,
    created_at  TIMESTAMP
);

INSERT INTO public.profile_test_dirty (id, name, amount, score, created_at) VALUES
    (1,  'Alice',  100.50, 10, '2026-01-01 10:00:00'),
    (2,  'Bob',     25.00,  0, '2026-01-02 11:00:00'),
    (3,  'Alice', -10.25, -5, '2026-01-03 12:00:00'),
    (4,  '',         0.00,  0, '2026-01-04 13:00:00'),
    (5,  '   ',     50.75, 20, '2026-01-05 14:00:00'),
    (6,  NULL,       NULL, NULL, NULL),
    (7,  'Bob',      25.00, 10, '2026-01-07 16:00:00'),
    (8,  'Carol',   999.99, 30, '2026-01-08 17:00:00'),
    (9,  'Carol',     0.00,  0, '2026-01-09 18:00:00'),
    (10, NULL,       -1.50, -10, '2026-01-10 19:00:00');

-- Expected high-level checks:
-- name:
--   null_count       = 2
--   empty_count      = 1
--   whitespace_count = 1
--   distinct_count   = 4 (Alice, Bob, whitespace, Carol, plus '' is 5 non-null values;
--                          PostgreSQL COUNT(DISTINCT) therefore expects 5)
--
-- amount:
--   null_count = 1
--   zero_count = 2
--   min        = -10.25
--   max        = 999.99
--
-- score:
--   null_count = 1
--   zero_count = 3
--   min        = -10
--   max        = 30
--
-- created_at:
--   null_count = 1
--   min        = 2026-01-01 10:00:00
--   max        = 2026-01-10 19:00:00
