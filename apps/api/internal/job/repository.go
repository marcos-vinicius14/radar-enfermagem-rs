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

	UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeenAt time.Time) error

	ReconcileStatuses(ctx context.Context, unknownBefore, expiredBefore time.Time) (StatusReconciliationResult, error)
}
