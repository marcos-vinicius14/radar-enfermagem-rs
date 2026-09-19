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
