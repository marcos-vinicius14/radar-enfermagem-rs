package collector

import "context"

const (
	// DefaultUserAgent identifica o coletor perante os servidores web externos dos portais hospitalares.
	DefaultUserAgent = "RadarEnfermagemRS/1.0 (+https://github.com/marcos-vinicius14/radar-enfermagem-rs)"
)

type SearchQuery struct {
	Query string
	City  string
	State string
	Limit int
}

type Collector interface {
	Name() string
	Collect(ctx context.Context, query SearchQuery) ([]RawJob, error)
}
