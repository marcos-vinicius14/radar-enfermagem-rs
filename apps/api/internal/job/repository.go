package job

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ListParams struct {
	Limit  int32
	Offset int32
}

type FilterParams struct {
	Query          string
	City           string
	State          string
	Company        string
	Source         string
	Status         string
	PublishedSince *time.Time
	PublishedUntil *time.Time
	Page           int
	Size           int
}

type PaginatedJobs struct {
	Items      []Job `json:"items"`
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type CompanyStat struct {
	Name      string `json:"name"`
	TotalJobs int64  `json:"total_jobs"`
}

type CityStat struct {
	City      string `json:"city"`
	State     string `json:"state"`
	TotalJobs int64  `json:"total_jobs"`
}

type SourceStat struct {
	Source    string `json:"source"`
	TotalJobs int64  `json:"total_jobs"`
}

type StatusReconciliationResult struct {
	MarkedUnknown int64
	MarkedExpired int64
}

type Repository interface {
	Insert(ctx context.Context, j Job) (Job, error)

	Update(ctx context.Context, j Job) (Job, error)

	FindByID(ctx context.Context, id uuid.UUID) (Job, error)

	FindBySourceAndExternalID(ctx context.Context, source, externalID string) (Job, error)

	FindByFingerprint(ctx context.Context, fingerprint string) (Job, error)

	List(ctx context.Context, params ListParams) ([]Job, error)

	Search(ctx context.Context, params FilterParams) (PaginatedJobs, error)

	ListCompanies(ctx context.Context, status string) ([]CompanyStat, error)

	ListCities(ctx context.Context, status string) ([]CityStat, error)

	ListSources(ctx context.Context, status string) ([]SourceStat, error)

	UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeenAt time.Time) error

	ReconcileStatuses(ctx context.Context, unknownBefore, expiredBefore time.Time) (StatusReconciliationResult, error)

	DeleteByIDs(ctx context.Context, ids []uuid.UUID) (int64, error)

	ListActiveForPruning(ctx context.Context, limit, offset int32) ([]Job, error)
}
