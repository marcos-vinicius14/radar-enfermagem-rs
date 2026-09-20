-- name: CreateJob :one
INSERT INTO jobs (
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status
) VALUES (
    COALESCE(NULLIF(sqlc.arg(id)::uuid, '00000000-0000-0000-0000-000000000000'::uuid), uuidv7()),
    sqlc.arg(external_id),
    sqlc.arg(title),
    sqlc.arg(company),
    sqlc.arg(description),
    sqlc.arg(city),
    sqlc.arg(state),
    sqlc.arg(source),
    sqlc.arg(source_url),
    sqlc.arg(fingerprint),
    sqlc.arg(work_mode),
    sqlc.arg(employment_type),
    sqlc.arg(salary_min),
    sqlc.arg(salary_max),
    sqlc.arg(published_at),
    sqlc.arg(collected_at),
    sqlc.arg(last_seen_at),
    sqlc.arg(status)
)
RETURNING
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at;

-- name: UpdateJob :one
UPDATE jobs
SET
    title = sqlc.arg(title),
    company = sqlc.arg(company),
    description = sqlc.arg(description),
    city = sqlc.arg(city),
    state = sqlc.arg(state),
    source_url = sqlc.arg(source_url),
    fingerprint = sqlc.arg(fingerprint),
    work_mode = sqlc.arg(work_mode),
    employment_type = sqlc.arg(employment_type),
    salary_min = sqlc.arg(salary_min),
    salary_max = sqlc.arg(salary_max),
    published_at = sqlc.arg(published_at),
    status = sqlc.arg(status),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at;

-- name: GetJobByID :one
SELECT
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at
FROM jobs
WHERE id = $1;

-- name: GetJobBySourceAndExternalID :one
SELECT
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at
FROM jobs
WHERE source = $1 AND external_id = $2;

-- name: GetJobByFingerprint :one
SELECT
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at
FROM jobs
WHERE fingerprint = $1
LIMIT 1;

-- name: ListJobs :many
SELECT
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at
FROM jobs
ORDER BY published_at DESC NULLS LAST, id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: UpdateJobLastSeen :one
UPDATE jobs
SET
    last_seen_at = sqlc.arg(last_seen_at),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, last_seen_at, updated_at;

-- name: CountJobs :one
SELECT COUNT(*) FROM jobs;

-- name: MarkJobsUnknown :execrows
UPDATE jobs
SET status = 'UNKNOWN', updated_at = NOW()
WHERE status = 'ACTIVE' AND last_seen_at < sqlc.arg(before_time);

-- name: MarkJobsExpired :execrows
UPDATE jobs
SET status = 'EXPIRED', updated_at = NOW()
WHERE status = 'UNKNOWN' AND last_seen_at < sqlc.arg(before_time);

-- name: SearchJobs :many
SELECT
    id,
    external_id,
    title,
    company,
    description,
    city,
    state,
    source,
    source_url,
    fingerprint,
    work_mode,
    employment_type,
    salary_min,
    salary_max,
    published_at,
    collected_at,
    last_seen_at,
    status,
    created_at,
    updated_at
FROM jobs
WHERE
    (sqlc.narg('query')::text IS NULL OR (
        title ILIKE '%' || sqlc.narg('query')::text || '%'
        OR company ILIKE '%' || sqlc.narg('query')::text || '%'
        OR description ILIKE '%' || sqlc.narg('query')::text || '%'
    ))
    AND (sqlc.narg('city')::text IS NULL OR city ILIKE '%' || sqlc.narg('city')::text || '%')
    AND (sqlc.narg('state')::text IS NULL OR state ILIKE sqlc.narg('state')::text)
    AND (sqlc.narg('company')::text IS NULL OR company ILIKE '%' || sqlc.narg('company')::text || '%')
    AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source')::text)
    AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
    AND (sqlc.narg('published_since')::timestamptz IS NULL OR published_at >= sqlc.narg('published_since')::timestamptz)
    AND (sqlc.narg('published_until')::timestamptz IS NULL OR published_at <= sqlc.narg('published_until')::timestamptz)
ORDER BY published_at DESC NULLS LAST, id DESC
LIMIT sqlc.arg('limit_count') OFFSET sqlc.arg('offset_count');

-- name: CountSearchJobs :one
SELECT COUNT(*)
FROM jobs
WHERE
    (sqlc.narg('query')::text IS NULL OR (
        title ILIKE '%' || sqlc.narg('query')::text || '%'
        OR company ILIKE '%' || sqlc.narg('query')::text || '%'
        OR description ILIKE '%' || sqlc.narg('query')::text || '%'
    ))
    AND (sqlc.narg('city')::text IS NULL OR city ILIKE '%' || sqlc.narg('city')::text || '%')
    AND (sqlc.narg('state')::text IS NULL OR state ILIKE sqlc.narg('state')::text)
    AND (sqlc.narg('company')::text IS NULL OR company ILIKE '%' || sqlc.narg('company')::text || '%')
    AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source')::text)
    AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
    AND (sqlc.narg('published_since')::timestamptz IS NULL OR published_at >= sqlc.narg('published_since')::timestamptz)
    AND (sqlc.narg('published_until')::timestamptz IS NULL OR published_at <= sqlc.narg('published_until')::timestamptz);

-- name: ListCompanies :many
SELECT
    company AS name,
    COUNT(*)::bigint AS total_jobs
FROM jobs
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND company != ''
GROUP BY company
ORDER BY company ASC;

-- name: ListCities :many
SELECT
    city,
    state,
    COUNT(*)::bigint AS total_jobs
FROM jobs
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND city != ''
GROUP BY city, state
ORDER BY city ASC, state ASC;

-- name: ListSources :many
SELECT
    source,
    COUNT(*)::bigint AS total_jobs
FROM jobs
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND source != ''
GROUP BY source
ORDER BY source ASC;

-- name: DeleteJobsByIDs :execrows
DELETE FROM jobs
WHERE id = ANY($1::uuid[]);

-- name: ListActiveJobsForPruning :many
SELECT id, title, description, city, state, company, source
FROM jobs
WHERE status = 'ACTIVE'
ORDER BY id ASC
LIMIT $1 OFFSET $2;


