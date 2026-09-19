# Observability Guidelines

> If a system cannot be observed, it cannot be reliably operated.

Observability is a first-class requirement of this project.

It must not be added after implementation.

Every relevant feature must be designed with:

1. Structured logs
2. Metrics
3. Distributed tracing
4. Correlation
5. Error visibility
6. Performance visibility
7. Operational diagnostics
8. Security and privacy
9. Actionable alerts

The objective is to answer production questions without reproducing the problem locally.

---

# 1. Core Principle

Every important operation should allow answering:

```text
What happened?
When did it happen?
Where did it happen?
Which operation failed?
Why did it fail?
How long did it take?
Which dependency was involved?
Which request/job/message triggered it?
How often is it happening?
Is the problem isolated or systemic?
```

If these questions cannot be answered using telemetry, observability is incomplete.

---

# 2. Three Primary Signals

The application must use:

```text
Logs
Metrics
Traces
```

Each solves a different problem.

## Logs

Explain discrete events.

Examples:

```text
job ingestion failed
database query failed
HTTP request rejected
scraping completed
configuration loaded
```

## Metrics

Explain aggregate system behavior.

Examples:

```text
request rate
error rate
latency
jobs imported
queue depth
database pool usage
```

## Traces

Explain the path and timing of an operation across components.

Example:

```text
HTTP request
    ↓
service
    ↓
PostgreSQL
    ↓
external API
```

Do not attempt to replace all three signals with logs.

---

# 3. Logging Library

Use the Go standard library:

```go
log/slog
```

for application logging unless another implementation is explicitly required.

Prefer structured logging.

Good:

```go
logger.Info(
	"job imported",
	"job_id", job.ID,
	"institution_id", job.InstitutionID,
	"source", job.Source,
)
```

Bad:

```go
logger.Printf(
	"Job %s imported from %s for institution %s",
	job.ID,
	job.Source,
	job.InstitutionID,
)
```

Structured fields enable:

```text
filtering
aggregation
correlation
dashboards
alerts
machine processing
```

---

# 4. Production Logs Must Be Structured

Production logs should normally be emitted as JSON.

Example:

```json
{
  "time": "2026-09-19T13:30:42.321Z",
  "level": "INFO",
  "msg": "job imported",
  "service": "radar-enfermagem-rs",
  "job_id": "018f...",
  "institution_id": "0190...",
  "source": "hospital-site"
}
```

Do not use plain multiline application logs in production.

Development environments may use human-readable output.

---

# 5. Configure Logger Once

Create the root logger during application startup.

Example:

```go
func newLogger(environment string) *slog.Logger {
	var handler slog.Handler

	if environment == "production" {
		handler = slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		)
	} else {
		handler = slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		)
	}

	return slog.New(handler)
}
```

Inject it where required.

Do not repeatedly instantiate independent logger configurations.

---

# 6. Prefer Explicit Logger Dependencies

Example:

```go
type Service struct {
	logger *slog.Logger
	repo   Repository
}

func NewService(
	logger *slog.Logger,
	repo Repository,
) *Service {
	return &Service{
		logger: logger,
		repo:   repo,
	}
}
```

Avoid uncontrolled global logging configuration.

---

# 7. Log Levels

Use log levels consistently.

## DEBUG

Detailed diagnostic information.

Examples:

```text
scraper page parsed
pagination cursor generated
retry scheduled
cache lookup result
```

DEBUG logs may be high-volume.

Do not depend on DEBUG logs for critical operational visibility.

---

## INFO

Normal significant system events.

Examples:

```text
application started
scraper execution completed
job imported
worker started
scheduled task completed
```

Do not log every internal function invocation.

---

## WARN

Unexpected behavior that did not prevent the operation from continuing.

Examples:

```text
retry required
optional external field missing
fallback mechanism used
deprecated input received
```

WARN should indicate something worth investigating if frequent.

---

## ERROR

An operation failed.

Examples:

```text
database query failed
external API request failed
scraper execution aborted
message processing failed
```

ERROR should correspond to an actual f
