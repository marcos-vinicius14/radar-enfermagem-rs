package collector

import "context"

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
