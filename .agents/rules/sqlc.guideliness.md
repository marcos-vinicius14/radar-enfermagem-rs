# SQL & sqlc Query Guidelines

> Standards for writing efficient, predictable, maintainable, and production-ready PostgreSQL queries using sqlc.

This document defines the SQL standards for this project.

The priorities are:

1. Correctness
2. Predictable performance
3. Query efficiency
4. Avoiding N+1 queries
5. Proper indexing
6. Explicit SQL
7. Type safety through sqlc
8. Transaction safety
9. Maintainability
10. Measurable optimization

The database is not an implementation detail.

SQL must be treated as production code.

---

# 1. Core Principle

Every query must be evaluated considering:

```text
rows scanned
rows returned
indexes used
join strategy
sorting
memory usage
network round trips
locking
transaction duration
future table growth
```

A query that performs well with 100 rows may perform poorly with 10 million rows.

Always reason about scale.

---

# 2. SQL Is the Source of Truth

The project uses sqlc.

Write SQL directly.

Do not recreate ORM behavior around sqlc.

Preferred flow:

```text
SQL query
↓
sqlc
↓
generated type-safe Go code
↓
application/service layer
```

Do not create generic repository abstractions merely to hide generated sqlc methods.

---

# 3. Prefer pgx/v5

Recommended sqlc configuration:

```yaml
version: "2"

sql:
  - engine: "postgresql"
    schema: "db/migrations"
    queries: "db/queries"
    gen:
      go:
        package: "database"
        out: "internal/database"
        sql_package: "pgx/v5"
```

This provides strong PostgreSQL integration.

---

# 4. Query Organization

Organize queries by domain.

Example:

```text
db/
├── migrations/
└── queries/
    ├── users.sql
    ├── jobs.sql
    ├── institutions.sql
    └── applications.sql
```

Avoid:

```text
query.sql
queries2.sql
misc.sql
utils.sql
```

---

# 5. Query Naming

sqlc query names must describe intent.

Good:

```sql
-- name: GetJobByID :one
```

```sql
-- name: ListPublishedJobs :many
```

```sql
-- name: CreateJob :one
```

Bad:

```sql
-- name: GetData :many
```

```sql
-- name: Query1 :many
```

```sql
-- name: Find :many
```

---

# 6. Use Correct sqlc Cardinality

Use:

```text
:one
```

when exactly zero or one logical row is expected.

Use:

```text
:many
```

for collections.

Use:

```text
:exec
```

when no result is required.

Use returning variants when generated IDs or updated rows are required.

Do not use `:many` when only one row is needed.

---

# 7. Avoid SELECT *

Do not use:

```sql
SELECT *
FROM jobs;
```

Prefer:

```sql
SELECT
    id,
    title,
    institution_id,
    city,
    published_at
FROM jobs;
```

Reasons:

```text
smaller network payloads
stable sqlc-generated structs
better API boundaries
better index-only scan opportunities
lower memory usage
less coupling to schema changes
```

Exceptions may be acceptable for extremely small internal tables when all columns are genuinely required.

Default rule remains:

```text
explicit columns
```

---

# 8. Avoid N+1 Queries

N+1 is forbidden unless explicitly justified.

Bad:

```go
jobs, err := queries.ListJobs(ctx)
if err != nil {
	return err
}

for _, job := range jobs {
	institution, err := queries.GetInstitutionByID(
		ctx,
		job.InstitutionID,
	)

	// ...
}
```

For:

```text
100 jobs
```

this generates approximately:

```text
1 query for jobs
+
100 queries for institutions
=
101 queries
```

Use a JOIN instead.

```sql
-- name: ListJobs :many
SELECT
    j.id,
    j.title,
    j.city,
    j.published_at,
    i.id AS institution_id,
    i.name AS institution_name
FROM jobs j
JOIN institutions i
    ON i.id = j.institution_id
ORDER BY j.published_at DESC;
```

One query replaces N+1 requests.

---

# 9. Detect N+1 During Code Review

Whenever code contains:

```go
for _, item := range items {
	queries.SomeQuery(...)
}
```

stop and inspect it.

This is a potential N+1 query.

Ask:

```text
Can this be solved with JOIN?
Can this be solved with WHERE id = ANY($1)?
Can this be solved with batch?
Can this be preloaded in one query?
Can the query return everything required?
```

---

# 10. Batch Fetching

When a JOIN is not appropriate, fetch related rows in one query.

Bad:

```sql
SELECT id, name
FROM institutions
WHERE id = $1;
```

executed repeatedly.

Prefer:

```sql
-- name: ListInstitutionsByIDs :many
SELECT
    id,
    name
FROM institutions
WHERE id = ANY($1::uuid[]);
```

Then construct the lookup map in Go.

Example:

```go
institutions, err := queries.ListInstitutionsByIDs(
	ctx,
	institutionIDs,
)
if err != nil {
	return err
}

byID := make(map[uuid.UUID]database.Institution, len(institutions))

for _, institution := range institutions {
	byID[institution.ID] = institution
}
```

---

# 11. JOIN Only What Is Needed

Bad:

```sql
SELECT
    ...
FROM jobs j
JOIN institutions i
    ON i.id = j.institution_id
JOIN users u
    ON u.id = j.created_by
JOIN cities c
    ON c.id = j.city_id
JOIN states s
    ON s.id = c.state_id
```

when only institution information is required.

Every JOIN has a cost.

Join only tables required by the use case.

---

# 12. Prefer Explicit JOIN Syntax

Use:

```sql
SELECT
    j.id,
    i.name
FROM jobs j
JOIN institutions i
    ON i.id = j.institution_id;
```

Avoid:

```sql
SELECT
    j.id,
    i.name
FROM jobs j, institutions i
WHERE j.institution_id = i.id;
```

Explicit joins are clearer and safer.

---

# 13. Understand INNER JOIN

Use:

```sql
JOIN
```

or:

```sql
INNER JOIN
```

when related data must exist.

Example:

```sql
SELECT
    j.id,
    i.name
FROM jobs j
JOIN institutions i
    ON i.id = j.institution_id;
```

Rows without an institution are excluded.

---

# 14. Understand LEFT JOIN

Use `LEFT JOIN` when the related row is optional.

```sql
SELECT
    j.id,
    j.title,
    s.name AS specialty
FROM jobs j
LEFT JOIN specialties s
    ON s.id = j.specialty_id;
```

Do not use `LEFT JOIN` automatically.

If the relationship is mandatory, use `JOIN`.

---

# 15. Be Careful Filtering LEFT JOINs

This:

```sql
SELECT
    j.id,
    s.name
FROM jobs j
LEFT JOIN specialties s
    ON s.id = j.specialty_id
WHERE s.active = true;
```

effectively behaves like an `INNER JOIN`.

If rows without specialties must remain:

```sql
SELECT
    j.id,
    s.name
FROM jobs j
LEFT JOIN specialties s
    ON s.id = j.specialty_id
   AND s.active = true;
```

Understand the semantic difference.

---

# 16. Index Foreign Keys Used for Lookups

PostgreSQL does not automatically create indexes for every foreign key.

If queries frequently use:

```sql
WHERE institution_id = $1
```

consider:

```sql
CREATE INDEX idx_jobs_institution_id
    ON jobs (institution_id);
```

Do not create indexes mechanically.

Create them based on actual query patterns.

---

# 17. Primary Keys Are Already Indexed

Do not create redundant indexes.

Bad:

```sql
CREATE TABLE jobs (
    id uuid PRIMARY KEY
);

CREATE INDEX idx_jobs_id
    ON jobs (id);
```

The primary key already provides an index.

---

# 18. Unique Constraints Create Indexes

Avoid redundant indexes such as:

```sql
CREATE UNIQUE INDEX users_email_key
    ON users (email);

CREATE INDEX idx_users_email
    ON users (email);
```

The unique index already supports lookups.

---

# 19. Index Based on Access Patterns

Do not create indexes because a column "looks important".

Create indexes based on real queries.

Given:

```sql
SELECT
    id,
    title
FROM jobs
WHERE institution_id = $1
  AND status = 'published'
ORDER BY published_at DESC
LIMIT 20;
```

a possible useful index may be:

```sql
CREATE INDEX idx_jobs_institution_status_published
    ON jobs (
        institution_id,
        status,
        published_at DESC
    );
```

But verify with the actual query planner.

---

# 20. Composite Index Column Order Matters

For B-tree indexes, leading columns are important.

An index:

```sql
CREATE INDEX idx_jobs_status_city
    ON jobs (status, city_id);
```

is particularly suitable for queries beginning with:

```sql
WHERE status = ...
```

Do not assume:

```text
INDEX(a, b)
```

is identical to:

```text
INDEX(b, a)
```

It is not.

Design indexes according to actual filter and ordering patterns.

---

# 21. Equality Columns Usually Come First

For queries such as:

```sql
WHERE status = 'published'
  AND published_at < $1
ORDER BY published_at DESC
```

a common index structure is:

```sql
CREATE INDEX idx_jobs_status_published_at
    ON jobs (
        status,
        published_at DESC
    );
```

Equality conditions commonly precede range conditions.

Verify using `EXPLAIN`.

---

# 22. Avoid Indexing Everything

Indexes improve reads but cost:

```text
disk space
INSERT performance
UPDATE performance
DELETE performance
VACUUM work
memory/cache
write amplification
```

Every index must justify its existence.

---

# 23. Partial Indexes

Partial indexes are useful when queries repeatedly access a subset.

Example:

```sql
SELECT
    id,
    title
FROM jobs
WHERE status = 'published'
ORDER BY published_at DESC;
```

Instead of indexing every row:

```sql
CREATE INDEX idx_jobs_published_at
    ON jobs (published_at DESC)
    WHERE status = 'published';
```

This may produce a smaller, more focused index.

Use only when the query predicate aligns with the partial-index predicate.

---

# 24. Indexes for Soft Delete

If using:

```text
deleted_at
```

and most application queries use:

```sql
WHERE deleted_at IS NULL
```

consider partial indexes:

```sql
CREATE INDEX idx_jobs_active_published_at
    ON jobs (published_at DESC)
    WHERE deleted_at IS NULL;
```

Do not blindly index historical/deleted rows when they are rarely queried.

---

# 25. Do Not Hide Indexed Columns Behind Functions

Potential problem:

```sql
WHERE LOWER(email) = LOWER($1)
```

A normal index on:

```text
email
```

may not satisfy the access pattern efficiently.

If case-insensitive lookup is required, consider an expression index:

```sql
CREATE UNIQUE INDEX idx_users_email_lower
    ON users (LOWER(email));
```

Then query consistently:

```sql
WHERE LOWER(email) = LOWER($1);
```

---

# 26. Avoid Casting Indexed Columns

Potentially problematic:

```sql
WHERE id::text = $1;
```

Prefer casting the parameter:

```sql
WHERE id = $1::uuid;
```

Keep indexed columns in their native type whenever possible.

---

# 27. Avoid Functions on Filtering Columns

Potentially inefficient:

```sql
WHERE DATE(created_at) = $1;
```

Prefer range queries:

```sql
WHERE created_at >= $1
  AND created_at < $1 + INTERVAL '1 day';
```

This generally gives indexes a better opportunity to be used.

---

# 28. Avoid Leading Wildcards

This:

```sql
WHERE title LIKE '%nurse%'
```

cannot normally use a standard B-tree prefix lookup effectively.

For real search functionality use PostgreSQL features designed for search:

```text
Full Text Search
GIN indexes
pg_trgm where appropriate
```

Do not abuse `LIKE '%...%'` for large datasets.

---

# 29. Full Text Search

For application search, prefer PostgreSQL Full Text Search over multiple broad `ILIKE` predicates when appropriate.

Example concept:

```sql
to_tsvector(
    'portuguese',
    title || ' ' || description
)
```

with a GIN index.

Do not recompute expensive search vectors unnecessarily if the use case is heavily queried.

Consider generated/stored search columns when appropriate.

---

# 30. Avoid Large OFFSET Pagination

Basic pagination:

```sql
SELECT
    id,
    title,
    published_at
FROM jobs
ORDER BY published_at DESC
LIMIT 20
OFFSET 200000;
```

requires PostgreSQL to skip a large number of rows.

Large offsets become increasingly expensive.

For large or frequently accessed collections, prefer keyset pagination.

---

# 31. Keyset Pagination

Prefer:

```sql
-- name: ListJobsAfterCursor :many
SELECT
    id,
    title,
    published_at
FROM jobs
WHERE
    (
        published_at,
        id
    ) < (
        sqlc.arg(cursor_published_at),
        sqlc.arg(cursor_id)
    )
ORDER BY
    published_at DESC,
    id DESC
LIMIT sqlc.arg(page_size);
```

Supporting index:

```sql
CREATE INDEX idx_jobs_published_cursor
    ON jobs (
        published_at DESC,
        id DESC
    );
```

Benefits:

```text
stable performance
no large OFFSET scans
stable navigation with new inserts
```

---

# 32. Pagination Requires Deterministic Ordering

Bad:

```sql
ORDER BY published_at DESC
```

if many rows may share the same timestamp.

Prefer:

```sql
ORDER BY
    published_at DESC,
    id DESC
```

The ordering must be deterministic.

---

# 33. LIMIT Without ORDER BY Is Unstable

Do not write:

```sql
SELECT
    id,
    title
FROM jobs
LIMIT 20;
```

when result order matters.

Use explicit ordering:

```sql
ORDER BY published_at DESC, id DESC
LIMIT 20;
```

---

# 34. Never Fetch Unlimited Collections by Default

Bad:

```sql
-- name: ListJobs :many
SELECT
    id,
    title
FROM jobs;
```

for an unbounded user-facing list.

Prefer:

```sql
LIMIT $1
```

or cursor pagination.

Every public collection must have a defined maximum result size.

---

# 35. Enforce Maximum Page Size

Do not trust:

```text
?limit=1000000
```

Application code should enforce a maximum.

Example:

```go
const (
	defaultPageSize = 20
	maxPageSize     = 100
)
```

Database queries should never accidentally return millions of rows to satisfy a single request.

---

# 36. EXISTS Instead of COUNT When Checking Existence

Bad:

```sql
SELECT COUNT(*)
FROM users
WHERE email = $1;
```

followed by:

```text
count > 0
```

Prefer:

```sql
-- name: UserExistsByEmail :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE email = $1
);
```

Express the intent directly.

---

# 37. Avoid COUNT(*) When It Is Not Needed

Do not count an entire result set just to determine whether any rows exist.

Use `EXISTS`.

Use exact counts only when exact counts are actually required.

---

# 38. Be Careful With Expensive Total Counts

Pagination often does:

```sql
SELECT COUNT(*)
FROM jobs
WHERE ...;
```

on every request.

For large filtered tables, this can be expensive.

Ask whether the UI genuinely requires:

```text
12,482 results
```

or whether:

```text
next page exists
```

is enough.

Fetching:

```text
pageSize + 1
```

rows is often sufficient to determine `hasNext`.

---

# 39. Fetch One Extra Row for hasNext

Example:

Requested page size:

```text
20
```

Database query:

```text
LIMIT 21
```

If 21 rows are returned:

```text
hasNext = true
```

Return only the first 20.

This can eliminate an expensive count query.

---

# 40. Avoid Application-Side Filtering

Bad:

```go
jobs, err := queries.ListAllJobs(ctx)

filtered := make([]Job, 0)

for _, job := range jobs {
	if job.City == city {
		filtered = append(filtered, job)
	}
}
```

Prefer:

```sql
WHERE city_id = $1
```

Databases are built for filtering.

Move data only when necessary.

---

# 41. Avoid Application-Side Sorting

Bad:

```go
sort.Slice(jobs, ...)
```

after retrieving thousands of database rows.

Prefer:

```sql
ORDER BY published_at DESC;
```

unless sorting genuinely depends on application-only data.

---

# 42. Avoid Application-Side Joins

Do not fetch:

```text
jobs
institutions
cities
```

independently and manually join large collections in Go when SQL can perform the join efficiently.

Use the database.

---

# 43. Use Database Aggregation

Prefer:

```sql
SELECT
    institution_id,
    COUNT(*) AS job_count
FROM jobs
WHERE status = 'published'
GROUP BY institution_id;
```

instead of loading all jobs and counting them in Go.

---

# 44. Use RETURNING

PostgreSQL supports:

```sql
RETURNING
```

Use it to avoid unnecessary round trips.

Bad:

```text
INSERT job
SELECT job
```

Prefer:

```sql
-- name: CreateJob :one
INSERT INTO jobs (
    title,
    institution_id
)
VALUES (
    $1,
    $2
)
RETURNING
    id,
    title,
    institution_id,
    created_at;
```

---

# 45. UPDATE RETURNING

Instead of:

```text
UPDATE
SELECT
```

prefer:

```sql
-- name: UpdateJob :one
UPDATE jobs
SET
    title = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING
    id,
    title,
    updated_at;
```

when the updated entity is required.

---

# 46. DELETE RETURNING

If the application needs information about the removed row:

```sql
DELETE FROM jobs
WHERE id = $1
RETURNING id;
```

Avoid a preceding `SELECT` merely to retrieve the same identifier.

---

# 47. Upsert

Prefer PostgreSQL atomic upserts:

```sql
INSERT INTO institutions (
    external_id,
    name
)
VALUES (
    $1,
    $2
)
ON CONFLICT (external_id)
DO UPDATE
SET
    name = EXCLUDED.name,
    updated_at = NOW()
RETURNING id;
```

Avoid:

```text
SELECT existence
IF exists UPDATE
ELSE INSERT
```

This produces extra queries and race conditions.

---

# 48. Avoid Check-Then-Act Races

Bad:

```text
SELECT user
if missing:
    INSERT user
```

Two concurrent requests may both pass the existence check.

Prefer database constraints and atomic operations:

```text
UNIQUE
ON CONFLICT
transaction
row locking
```

depending on the requirement.

---

# 49. Constraints Are Part of Correctness

Use the database to enforce invariants where appropriate.

Examples:

```sql
NOT NULL
UNIQUE
CHECK
FOREIGN KEY
PRIMARY KEY
```

Do not depend exclusively on Go validation.

Application validation improves UX.

Database constraints guarantee integrity.

---

# 50. Prefer NOT NULL

If a value is required, define:

```sql
title text NOT NULL
```

Do not allow nullability accidentally.

Nullability affects generated sqlc types and increases application complexity.

---

# 51. Avoid Meaningless NULL

Do not use `NULL` for multiple concepts such as:

```text
unknown
not applicable
not loaded
empty
deleted
```

Define semantics clearly.

---

# 52. Use Correct Data Types

Use database types representing actual data.

Prefer:

```text
uuid
boolean
date
timestamp with time zone
integer
bigint
numeric
jsonb
text
```

instead of storing everything as:

```text
varchar
text
```

Correct types improve:

```text
validation
indexing
storage
query planning
generated Go types
```

---

# 53. Timestamps

For actual instants in time, generally use:

```sql
timestamptz
```

rather than:

```sql
timestamp
```

Store instants consistently.

Convert presentation timezone at application/UI boundaries.

---

# 54. Avoid VARCHAR Length Without Domain Reason

Prefer:

```sql
name text NOT NULL
```

unless:

```text
VARCHAR(100)
```

represents a genuine business invariant.

Do not add arbitrary database limits merely from habit.

---

# 55. UUIDs

Use native PostgreSQL:

```sql
uuid
```

Do not store UUID values as:

```text
varchar(36)
text
```

unless required for compatibility.

---

# 56. JSONB

Use `jsonb` only when the structure is genuinely flexible.

Do not use JSON as an escape hatch to avoid relational modeling.

Bad:

```text
jobs.metadata
```

containing important searchable relational fields.

If data is frequently:

```text
filtered
joined
ordered
constrained
```

it usually deserves real columns/tables.

---

# 57. Transactions

Use transactions when multiple database operations must succeed or fail together.

Example:

```go
tx, err := db.Begin(ctx)
if err != nil {
	return fmt.Errorf("begin transaction: %w", err)
}

defer tx.Rollback(ctx)

qtx := queries.WithTx(tx)

if err := qtx.CreateUser(...); err != nil {
	return fmt.Errorf("create user: %w", err)
}

if err := qtx.CreateProfile(...); err != nil {
	return fmt.Errorf("create profile: %w", err)
}

if err := tx.Commit(ctx); err != nil {
	return fmt.Errorf("commit transaction: %w", err)
}

return nil
```

---

# 58. Keep Transactions Short

Do not perform slow external operations inside transactions.

Bad:

```text
BEGIN
INSERT
HTTP request
send email
wait for remote API
UPDATE
COMMIT
```

Long transactions increase:

```text
lock duration
contention
deadlock probability
connection occupancy
MVCC overhead
```

Perform only required database work inside the transaction.

---

# 59. Never Call External Services Inside a DB Transaction

Avoid:

```text
database transaction starts
↓
external HTTP call
↓
RabbitMQ publish
↓
transaction commits
```

unless implementing a specific consistency strategy.

Prefer patterns such as:

```text
transactional outbox
```

when database state and messaging must coordinate reliably.

---

# 60. Transactions Must Have One Owner

The application/service layer starting a transaction should control:

```text
commit
rollback
```

Do not allow nested layers to commit unexpectedly.

---

# 61. Row Locking

For read-modify-write operations requiring serialization:

```sql
SELECT
    id,
    balance
FROM accounts
WHERE id = $1
FOR UPDATE;
```

Use only when locking semantics are required.

Do not add `FOR UPDATE` casually.

Locks reduce concurrency.

---

# 62. Prefer Atomic Updates

Instead of:

```text
SELECT counter
counter++
UPDATE counter
```

prefer:

```sql
UPDATE counters
SET value = value + 1
WHERE id = $1
RETURNING value;
```

One atomic operation avoids race conditions and round trips.

---

# 63. Avoid Read-Modify-Write When SQL Can Do It

Bad:

```go
job, _ := queries.GetJob(...)
job.ViewCount++
queries.UpdateViewCount(...)
```

Prefer:

```sql
UPDATE jobs
SET view_count = view_count + 1
WHERE id = $1;
```

---

# 64. Batch Inserts

Do not repeatedly execute:

```text
INSERT
INSERT
INSERT
INSERT
```

for thousands of rows when batch operations are available.

Prefer:

```text
COPY
batch
multi-row INSERT
```

depending on the use case.

For large ingestion workloads, prefer PostgreSQL `COPY` / pgx copy support when appropriate.

---

# 65. Multi-Row INSERT

For reasonably sized collections:

```sql
INSERT INTO jobs (
    title,
    institution_id
)
VALUES
    (...),
    (...),
    (...);
```

can reduce network round trips.

For very large ingestion, use bulk mechanisms.

---

# 66. Avoid Giant IN Lists

Do not dynamically generate:

```sql
WHERE id IN (
    $1, $2, $3, ...
)
```

for thousands of values.

With PostgreSQL and sqlc, prefer arrays:

```sql
WHERE id = ANY($1::uuid[]);
```

This keeps the SQL shape stable.

---

# 67. Query Parameterization

Always use query parameters.

Bad:

```go
query := fmt.Sprintf(
	"SELECT * FROM jobs WHERE city = '%s'",
	city,
)
```

Never interpolate external input into SQL.

Use sqlc parameters.

---

# 68. Dynamic Filtering

Avoid constructing raw SQL from user-controlled strings.

If filters are known:

```sql
WHERE
    (sqlc.narg(city_id)::uuid IS NULL
        OR city_id = sqlc.narg(city_id))
```

may be acceptable for small optional filter sets.

However, optional-filter patterns can negatively affect planning.

For complex search systems, consider separate query shapes for major access patterns.

Do not create one universal query handling every possible case.

---

# 69. Avoid Mega Queries

Bad:

```text
ListEverythingWithAllPossibleFiltersAndJoins
```

containing:

```text
20 optional filters
12 joins
multiple CTEs
dynamic sorting
aggregations
```

Prefer focused queries representing real use cases.

Query specialization often produces:

```text
simpler SQL
better indexes
better plans
easier maintenance
```

---

# 70. CTEs

Use CTEs when they improve clarity.

Example:

```sql
WITH published_jobs AS (
    SELECT
        id,
        institution_id
    FROM jobs
    WHERE status = 'published'
)
SELECT ...
```

Do not use CTEs merely to make SQL look sophisticated.

Prefer direct SQL when clearer.

---

# 71. Subqueries

Subqueries are not inherently bad.

Use them when they clearly express the operation.

Example:

```sql
SELECT EXISTS (
    SELECT 1
    FROM jobs
    WHERE institution_id = $1
);
```

Optimization should be based on the execution plan, not prejudice against specific syntax.

---

# 72. EXPLAIN

Performance-sensitive queries must be inspected with:

```sql
EXPLAIN
SELECT ...;
```

This reveals the planned operations.

---

# 73. EXPLAIN ANALYZE

For realistic testing:

```sql
EXPLAIN ANALYZE
SELECT ...;
```

This executes the query and reports actual execution statistics.

Do not run destructive statements with `EXPLAIN ANALYZE` against production without understanding the consequences.

For DML, use a safe environment or transaction strategy.

---

# 74. Prefer EXPLAIN ANALYZE BUFFERS

For deeper investigation:

```sql
EXPLAIN (
    ANALYZE,
    BUFFERS
)
SELECT ...;
```

Inspect:

```text
execution time
actual rows
estimated rows
loops
buffer hits
buffer reads
sort operations
scan types
join types
```

---

# 75. Watch for Sequential Scans

A:

```text
Seq Scan
```

is not automatically bad.

For a small table, it may be optimal.

For a large table returning very few rows, investigate.

Never assume:

```text
Seq Scan = bug
```

Use scale and execution statistics.

---

# 76. Watch Estimated vs Actual Rows

Example:

```text
estimated rows: 100
actual rows: 500000
```

Large estimation errors may cause poor plans.

Investigate:

```text
statistics
data distribution
expressions
correlated columns
stale ANALYZE data
```

---

# 77. ANALYZE

PostgreSQL relies on statistics for query planning.

Make sure normal database maintenance keeps statistics current.

Do not optimize queries based on unrealistic empty or tiny development databases.

---

# 78. Test With Realistic Data

Performance testing with:

```text
50 rows
```

proves little when production will contain:

```text
5 million rows
```

Use representative data volume and distributions.

---

# 79. Measure Before Optimizing

Do not rewrite SQL because it "looks slow".

Measure:

```text
EXPLAIN ANALYZE
query duration
rows processed
buffers
database metrics
```

Then optimize.

---

# 80. Optimize the Correct Bottleneck

Possible bottlenecks include:

```text
database CPU
disk I/O
network latency
too many round trips
missing index
bad join order
sorting
lock contention
connection pool exhaustion
large payload
application processing
```

Do not assume every performance issue requires another index.

---

# 81. Query Round Trips Matter

Ten queries taking:

```text
2 ms each
```

can still be worse than one query taking:

```text
8 ms
```

because each database call adds:

```text
network latency
pool acquisition
protocol overhead
query processing
```

Minimize unnecessary round trips.

---

# 82. Avoid Queries Inside Nested Loops

This is especially dangerous:

```go
for _, institution := range institutions {
	jobs, _ := queries.ListJobsByInstitution(...)

	for _, job := range jobs {
		applications, _ := queries.ListApplicationsByJob(...)
	}
}
```

This produces multiplicative query counts.

Refactor into:

```text
JOIN
batched queries
aggregated queries
```

---

# 83. Avoid Duplicate Queries in One Request

Do not repeatedly execute:

```sql
GetUserByID
```

for the same ID during a single operation.

Pass already-loaded data when ownership is clear.

Avoid introducing caching complexity prematurely, but do not knowingly duplicate identical queries.

---

# 84. Select Only Required Rows

Always filter as early as logically possible.

Bad:

```sql
SELECT ...
FROM jobs;
```

then filter in Go.

Prefer:

```sql
WHERE
    status = 'published'
    AND city_id = $1
```

---

# 85. Select Only Required Columns

For a result list:

```sql
SELECT
    id,
    title,
    institution_name,
    city,
    published_at
```

Do not fetch:

```text
full description
large JSON
audit metadata
raw scraped HTML
```

if the list page does not display them.

Fetch detailed fields in the detail query.

---

# 86. List Query vs Detail Query

Keep them separate.

Example:

```sql
-- name: ListJobs :many
SELECT
    id,
    title,
    city,
    published_at
FROM jobs
...
```

and:

```sql
-- name: GetJobByID :one
SELECT
    id,
    title,
    description,
    requirements,
    institution_id,
    city,
    source_url,
    published_at
FROM jobs
WHERE id = $1;
```

Do not use the detailed entity query for lists.

---

# 87. Avoid Large TEXT/JSON Fields in Lists

Large fields increase:

```text
disk I/O
buffer usage
network payload
Go allocations
GC pressure
```

Do not fetch them unless required.

---

# 88. Ordering Must Match Index Strategy

Given:

```sql
WHERE status = 'published'
ORDER BY published_at DESC, id DESC
```

consider an index matching the access pattern:

```sql
CREATE INDEX idx_jobs_published_order
    ON jobs (
        status,
        published_at DESC,
        id DESC
    );
```

Do not assume PostgreSQL can efficiently use any unrelated index for ordering.

---

# 89. Avoid Random Ordering for Large Tables

Be careful with:

```sql
ORDER BY RANDOM()
```

on large tables.

It generally requires generating random values and sorting many rows.

Use alternative sampling strategies when large-scale random selection is required.

---

# 90. Avoid Unnecessary DISTINCT

Do not add:

```sql
DISTINCT
```

merely because a JOIN produces duplicates.

Understand why duplicates exist.

Often they indicate:

```text
incorrect join
wrong relationship
missing aggregation
wrong query model
```

`DISTINCT` can hide logic problems and introduce sorting/hashing cost.

---

# 91. GROUP BY Intentionally

Do not use:

```sql
GROUP BY
```

as another form of `DISTINCT`.

Use it when performing actual aggregation.

---

# 92. Avoid Repeated Correlated Subqueries

Potentially expensive:

```sql
SELECT
    j.id,
    (
        SELECT COUNT(*)
        FROM applications a
        WHERE a.job_id = j.id
    ) AS applications
FROM jobs j;
```

This may effectively create repeated work.

Depending on the plan and workload, prefer aggregation:

```sql
SELECT
    j.id,
    COUNT(a.id) AS applications
FROM jobs j
LEFT JOIN applications a
    ON a.job_id = j.id
GROUP BY j.id;
```

Always inspect the execution plan.

---

# 93. Use LATERAL Only Intentionally

`LATERAL` can be useful for:

```text
top N children per parent
dependent subqueries
structured correlated lookups
```

but should not be introduced without understanding its execution implications.

---

# 94. Index-Only Opportunities

If a hot query requires:

```sql
SELECT
    id,
    title
FROM jobs
WHERE institution_id = $1;
```

a covering index may sometimes help:

```sql
CREATE INDEX idx_jobs_institution_cover
    ON jobs (institution_id)
    INCLUDE (id, title);
```

Do this only after measurement.

Covering indexes increase index size and write cost.

---

# 95. Concurrency

Assume concurrent requests.

A query correct under a single-user test may fail under concurrency.

Consider:

```text
unique constraints
atomic updates
transactions
locking
isolation
idempotency
```

---

# 96. Idempotency

Operations triggered by:

```text
workers
HTTP retries
scrapers
message queues
```

may execute more than once.

Database writes should be designed accordingly.

Use:

```text
unique business keys
ON CONFLICT
deduplication keys
state checks
```

where appropriate.

---

# 97. Scraping Imports

For ingestion from external job sources:

Avoid:

```text
SELECT existence
INSERT/UPDATE
```

for every scraped item.

Prefer stable external identifiers and upserts.

Example:

```sql
INSERT INTO jobs (
    source,
    external_id,
    title,
    source_url
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
ON CONFLICT (
    source,
    external_id
)
DO UPDATE
SET
    title = EXCLUDED.title,
    source_url = EXCLUDED.source_url,
    updated_at = NOW();
```

Back this with:

```sql
UNIQUE (source, external_id)
```

---

# 98. Deduplication

Do not rely only on application memory for deduplication.

Whenever a stable uniqueness rule exists, enforce it in the database.

Example:

```sql
UNIQUE (
    institution_id,
    external_id
)
```

The database is the final concurrency-safe enforcement layer.

---

# 99. Connection Pool

Do not create one database connection per query.

Use the pgx connection pool.

The application should normally own one:

```text
pgxpool.Pool
```

and sqlc queries should use it.

---

# 100. Do Not Oversize the Pool

More connections do not automatically mean more throughput.

Too many connections can increase:

```text
memory
context switching
database contention
lock contention
```

Pool size must reflect:

```text
database capacity
number of application instances
query latency
workload
```

---

# 101. Context and Queries

Always propagate context.

sqlc methods should be called using the request/job context:

```go
job, err := queries.GetJobByID(ctx, id)
```

Do not replace it unnecessarily with:

```go
context.Background()
```

inside repositories/services.

---

# 102. Query Timeouts

Slow or external-facing operations should have bounded execution.

Example:

```go
ctx, cancel := context.WithTimeout(
	ctx,
	3*time.Second,
)
defer cancel()

jobs, err := queries.ListJobs(ctx, params)
```

Do not apply arbitrary timeout values without understanding the workload.

---

# 103. Query Errors

Wrap sqlc/database errors with context.

Example:

```go
job, err := queries.GetJobByID(ctx, id)
if err != nil {
	return Job{}, fmt.Errorf(
		"get job %s: %w",
		id,
		err,
	)
}
```

Do not log and return the same error at every layer.

---

# 104. pgx ErrNoRows

When a missing row is expected behavior, translate it into a domain/application error.

Example concept:

```go
if errors.Is(err, pgx.ErrNoRows) {
	return Job{}, ErrJobNotFound
}
```

Do not expose driver-specific errors throughout the application.

---

# 105. Generated Code

Never manually modify generated sqlc files.

Generated directory:

```text
internal/database
```

should be regenerated using:

```bash
sqlc generate
```

Any changes must originate from:

```text
SQL
schema
sqlc configuration
```

---

# 106. sqlc Verification

After modifying:

```text
schema
queries
migrations
```

run:

```bash
sqlc generate
```

and ensure compilation succeeds.

---

# 107. sqlc.arg

Use named arguments when they improve generated APIs.

Example:

```sql
WHERE institution_id = sqlc.arg(institution_id)
```

instead of relying on unclear parameter numbering for complex queries.

---

# 108. sqlc.narg

Use nullable named arguments intentionally.

Do not introduce nullable parameters merely to create a universal query.

Nullable query parameters should represent a genuine optional input.

---

# 109. Keep Generated Param Structs Clear

Prefer query structures producing:

```go
type ListJobsParams struct {
	CityID uuid.UUID
	Status JobStatus
	Limit  int32
}
```

rather than dozens of poorly named generic parameters.

Query API quality matters.

---

# 110. Database Migrations and Queries Must Evolve Together

When schema changes:

```text
migration
query
generated sqlc code
tests
```

must be changed together.

Do not deploy queries expecting columns that have not yet been migrated.

---

# 111. Index Migrations

Indexes belong in migrations.

Example:

```sql
CREATE INDEX idx_jobs_status_published_at
    ON jobs (
        status,
        published_at DESC
    );
```

Do not create production indexes manually and leave them undocumented.

---

# 112. Production Index Creation

For large production tables, understand locking implications before creating indexes.

PostgreSQL supports:

```sql
CREATE INDEX CONCURRENTLY
```

for scenarios where blocking writes during index creation is unacceptable.

Use intentionally.

Do not blindly add `CONCURRENTLY` to every migration because transaction behavior differs.

---

# 113. Query Review Checklist

Every new significant query must answer:

```text
What rows can this scan?
What rows does it return?
What is the expected cardinality?
Which index supports it?
Does ORDER BY match the index?
Could it become N+1?
Can related data be joined?
Is pagination bounded?
Can OFFSET become large?
Does SELECT return unnecessary columns?
Does the query perform unnecessary COUNT?
Are there redundant round trips?
Does it work under concurrency?
Does it require a transaction?
Can SQL perform the operation atomically?
```

---

# 114. Performance Review Checklist

For performance-sensitive queries:

```text
[ ] EXPLAIN inspected
[ ] EXPLAIN ANALYZE tested using realistic data
[ ] Actual vs estimated rows checked
[ ] Sequential scans reviewed
[ ] Index strategy reviewed
[ ] Sort operations reviewed
[ ] Join strategy reviewed
[ ] Number of loops reviewed
[ ] Rows removed by filter reviewed
[ ] Buffers reviewed when necessary
[ ] Query result size measured
[ ] Number of DB round trips measured
```

---

# 115. N+1 Checklist

Before completing database-related code:

```text
[ ] No query is executed inside a loop without explicit justification
[ ] Related entities are loaded using JOIN when appropriate
[ ] Bulk ID lookups use ANY(array) or equivalent
[ ] Aggregations are performed in SQL
[ ] Duplicate queries within one operation are avoided
[ ] Nested loops do not generate nested database calls
```

---

# 116. Index Checklist

Before adding an index:

```text
[ ] A real query requires it
[ ] Existing indexes were inspected
[ ] It is not redundant with PRIMARY KEY
[ ] It is not redundant with UNIQUE
[ ] Composite column order matches query usage
[ ] Write overhead is acceptable
[ ] Table cardinality justifies it
[ ] EXPLAIN validates its usefulness
[ ] A partial index was considered where appropriate
[ ] Index size and maintenance cost were considered
```

---

# 117. Pagination Checklist

For every paginated endpoint:

```text
[ ] LIMIT is bounded
[ ] ORDER BY exists
[ ] ORDER BY is deterministic
[ ] Large OFFSET behavior was considered
[ ] Keyset pagination is used for large datasets where appropriate
[ ] Supporting index exists
[ ] COUNT(*) is only used when genuinely necessary
```

---

# 118. Transaction Checklist

Before using a transaction:

```text
[ ] Multiple operations actually require atomicity
[ ] Transaction duration is minimal
[ ] No external HTTP request occurs inside
[ ] No email sending occurs inside
[ ] No unnecessary message publishing occurs inside
[ ] Locks are understood
[ ] Rollback is guaranteed
[ ] Commit has a single owner
[ ] Context cancellation is propagated
```

---

# 119. AI Coding Agent Rules

When generating SQL/sqlc code for this repository:

## MUST

* Write explicit SQL.
* Use sqlc-generated code.
* Use PostgreSQL-native functionality where appropriate.
* Avoid N+1 queries.
* Inspect loops containing database calls.
* Prefer JOINs for directly related data.
* Prefer batched ID retrieval when JOIN is inappropriate.
* Select only required columns.
* Select only required rows.
* Bound all user-facing list queries.
* Use deterministic ordering.
* Prefer keyset pagination for large datasets.
* Use parameterized SQL.
* Design indexes around actual access patterns.
* Consider composite index ordering.
* Use database constraints for integrity.
* Use atomic SQL operations when possible.
* Use `RETURNING` to avoid unnecessary round trips.
* Use `EXISTS` for existence checks.
* Use transactions only when atomicity requires them.
* Keep transactions short.
* Propagate `context.Context`.
* Evaluate significant queries using `EXPLAIN ANALYZE`.
* Optimize based on evidence.

## MUST NOT

* Introduce an ORM over sqlc.
* Execute queries inside loops without explicit justification.
* Use `SELECT *` by default.
* Load an entire table and filter in Go.
* Load an entire table and sort in Go.
* perform large application-side joins.
* use unbounded collection queries.
* use large OFFSET pagination when keyset pagination fits.
* interpolate user input into SQL.
* add redundant indexes.
* create indexes for every column.
* use `DISTINCT` to hide broken joins.
* use `COUNT(*)` merely to check existence.
* perform check-then-insert when a database constraint/upsert can solve it.
* perform read-modify-write when an atomic UPDATE can solve it.
* store strongly relational searchable data inside JSON without justification.
* use functions or casts on indexed columns unnecessarily.
* execute network operations inside database transactions.
* manually edit sqlc-generated code.
* optimize based solely on intuition.

---

# 120. Preferred List Query

```sql
-- name: ListPublishedJobs :many
SELECT
    j.id,
    j.title,
    j.city,
    j.published_at,
    i.id AS institution_id,
    i.name AS institution_name
FROM jobs j
JOIN institutions i
    ON i.id = j.institution_id
WHERE j.status = 'published'
  AND (
      j.published_at,
      j.id
  ) < (
      sqlc.arg(cursor_published_at),
      sqlc.arg(cursor_id)
  )
ORDER BY
    j.published_at DESC,
    j.id DESC
LIMIT sqlc.arg(page_size);
```

Potential supporting index:

```sql
CREATE INDEX idx_jobs_published_cursor
    ON jobs (
        status,
        published_at DESC,
        id DESC
    );
```

Properties:

```text
one database round trip
no N+1
deterministic pagination
bounded result set
keyset pagination
explicit columns
index-friendly access pattern
```

---

# 121. Preferred Detail Query

```sql
-- name: GetJobByID :one
SELECT
    j.id,
    j.title,
    j.description,
    j.requirements,
    j.city,
    j.source_url,
    j.published_at,
    i.id AS institution_id,
    i.name AS institution_name
FROM jobs j
JOIN institutions i
    ON i.id = j.institution_id
WHERE j.id = $1;
```

Do not run:

```text
GetJob
+
GetInstitution
```

when one query can retrieve the required data.

---

# 122. Preferred Existence Query

```sql
-- name: JobExistsByExternalID :one
SELECT EXISTS (
    SELECT 1
    FROM jobs
    WHERE source = $1
      AND external_id = $2
);
```

But when immediately inserting afterward, prefer a unique constraint and `ON CONFLICT` rather than performing this query first.

---

# 123. Preferred Atomic Update

```sql
-- name: IncrementJobViews :one
UPDATE jobs
SET
    view_count = view_count + 1
WHERE id = $1
RETURNING view_count;
```

Avoid:

```text
SELECT view_count
increment in Go
UPDATE
```

---

# 124. Preferred Upsert

```sql
-- name: UpsertJob :one
INSERT INTO jobs (
    source,
    external_id,
    institution_id,
    title,
    description,
    source_url,
    published_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
ON CONFLICT (
    source,
    external_id
)
DO UPDATE
SET
    institution_id = EXCLUDED.institution_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    source_url = EXCLUDED.source_url,
    published_at = EXCLUDED.published_at,
    updated_at = NOW()
RETURNING
    id,
    source,
    external_id,
    institution_id,
    title;
```

Supporting constraint:

```sql
ALTER TABLE jobs
ADD CONSTRAINT uq_jobs_source_external_id
UNIQUE (
    source,
    external_id
);
```

This should be preferred for scraper ingestion when the external source provides a stable identifier.

---

# 125. Final Philosophy

Database code should follow:

```text
one good query > N small queries

JOIN > N+1

batch > query loop

database filtering > application filtering

database aggregation > application aggregation

atomic SQL > read-modify-write

RETURNING > write + SELECT

EXISTS > COUNT for existence

keyset pagination > large OFFSET

explicit columns > SELECT *

constraints > application-only assumptions

measured optimization > guessed optimization

few deliberate indexes > index everything

short transactions > long transactions

SQL designed for scale > SQL designed only for development data
```

The goal is not merely to make SQL work.

The goal is to make database behavior remain predictable as the amount of data grows.
