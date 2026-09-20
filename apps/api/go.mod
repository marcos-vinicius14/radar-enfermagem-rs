module github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api

go 1.26.0

require (
	github.com/go-chi/chi/v5 v5.3.2
	github.com/golang-migrate/migrate/v4 v4.20.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.11.0
	github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web v0.0.0
	github.com/robfig/cron/v3 v3.0.1
	golang.org/x/sync v0.23.0
	golang.org/x/text v0.38.0
	golang.org/x/time v0.16.0
)

replace github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web => ../web

require (
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
)
