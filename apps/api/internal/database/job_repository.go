package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/database/db"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

var _ job.Repository = (*JobRepository)(nil)

type JobRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

func NewJobRepository(pool *pgxpool.Pool, logger *slog.Logger) *JobRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &JobRepository{
		queries: db.New(pool),
		pool:    pool,
		logger:  logger,
	}
}

func (r *JobRepository) Insert(ctx context.Context, j job.Job) (job.Job, error) {
	if err := j.Validate(); err != nil {
		return job.Job{}, fmt.Errorf("validar vaga: %w", err)
	}

	now := time.Now()
	collectedAt := j.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = now
	}
	lastSeenAt := j.LastSeenAt
	if lastSeenAt.IsZero() {
		lastSeenAt = now
	}

	params := db.CreateJobParams{
		ID:             j.ID,
		ExternalID:     j.ExternalID,
		Title:          j.Title,
		Company:        j.Company,
		Description:    j.Description,
		City:           j.City,
		State:          j.State,
		Source:         j.Source,
		SourceUrl:      j.SourceURL,
		Fingerprint:    j.Fingerprint,
		WorkMode:       string(j.WorkMode),
		EmploymentType: string(j.EmploymentType),
		SalaryMin:      toNullableInt8(j.SalaryMin),
		SalaryMax:      toNullableInt8(j.SalaryMax),
		PublishedAt:    toNullableTimestamptz(j.PublishedAt),
		CollectedAt:    toTimestamptz(collectedAt),
		LastSeenAt:     toTimestamptz(lastSeenAt),
		Status:         string(j.Status),
	}

	row, err := r.queries.CreateJob(ctx, params)
	if err != nil {
		r.logger.ErrorContext(ctx, "falha ao inserir vaga no banco de dados",
			slog.String("source", j.Source),
			slog.String("external_id", j.ExternalID),
			slog.String("erro", err.Error()),
		)
		return job.Job{}, fmt.Errorf("inserir vaga: %w", err)
	}

	return toDomainJob(row), nil
}

func (r *JobRepository) Update(ctx context.Context, j job.Job) (job.Job, error) {
	if err := j.Validate(); err != nil {
		return job.Job{}, fmt.Errorf("validar vaga: %w", err)
	}

	params := db.UpdateJobParams{
		ID:             j.ID,
		Title:          j.Title,
		Company:        j.Company,
		Description:    j.Description,
		City:           j.City,
		State:          j.State,
		SourceUrl:      j.SourceURL,
		Fingerprint:    j.Fingerprint,
		WorkMode:       string(j.WorkMode),
		EmploymentType: string(j.EmploymentType),
		SalaryMin:      toNullableInt8(j.SalaryMin),
		SalaryMax:      toNullableInt8(j.SalaryMax),
		PublishedAt:    toNullableTimestamptz(j.PublishedAt),
		Status:         string(j.Status),
	}

	row, err := r.queries.UpdateJob(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Job{}, job.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "falha ao atualizar vaga no banco de dados",
			slog.String("id", j.ID.String()),
			slog.String("erro", err.Error()),
		)
		return job.Job{}, fmt.Errorf("atualizar vaga: %w", err)
	}

	return toDomainJob(row), nil
}

func (r *JobRepository) FindByID(ctx context.Context, id uuid.UUID) (job.Job, error) {
	row, err := r.queries.GetJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Job{}, job.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "falha ao buscar vaga por id",
			slog.String("id", id.String()),
			slog.String("erro", err.Error()),
		)
		return job.Job{}, fmt.Errorf("buscar vaga por id: %w", err)
	}

	return toDomainJob(row), nil
}

func (r *JobRepository) FindBySourceAndExternalID(ctx context.Context, source, externalID string) (job.Job, error) {
	params := db.GetJobBySourceAndExternalIDParams{
		Source:     source,
		ExternalID: externalID,
	}

	row, err := r.queries.GetJobBySourceAndExternalID(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Job{}, job.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "falha ao buscar vaga por fonte e id externo",
			slog.String("source", source),
			slog.String("external_id", externalID),
			slog.String("erro", err.Error()),
		)
		return job.Job{}, fmt.Errorf("buscar vaga por fonte e id externo: %w", err)
	}

	return toDomainJob(row), nil
}

func (r *JobRepository) FindByFingerprint(ctx context.Context, fingerprint string) (job.Job, error) {
	row, err := r.queries.GetJobByFingerprint(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.Job{}, job.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "falha ao buscar vaga por fingerprint",
			slog.String("fingerprint", fingerprint),
			slog.String("erro", err.Error()),
		)
		return job.Job{}, fmt.Errorf("buscar vaga por fingerprint: %w", err)
	}

	return toDomainJob(row), nil
}

func (r *JobRepository) List(ctx context.Context, params job.ListParams) ([]job.Job, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := r.queries.ListJobs(ctx, db.ListJobsParams{
		LimitCount:  limit,
		OffsetCount: offset,
	})
	if err != nil {
		r.logger.ErrorContext(ctx, "falha ao listar vagas",
			slog.Int("limit", int(limit)),
			slog.Int("offset", int(offset)),
			slog.String("erro", err.Error()),
		)
		return nil, fmt.Errorf("listar vagas: %w", err)
	}

	jobs := make([]job.Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, toDomainJob(row))
	}

	return jobs, nil
}

func (r *JobRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeenAt time.Time) error {
	params := db.UpdateJobLastSeenParams{
		ID:         id,
		LastSeenAt: toTimestamptz(lastSeenAt),
	}

	if _, err := r.queries.UpdateJobLastSeen(ctx, params); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return job.ErrNotFound
		}

		r.logger.ErrorContext(ctx, "falha ao atualizar último visto da vaga",
			slog.String("id", id.String()),
			slog.String("erro", err.Error()),
		)
		return fmt.Errorf("atualizar último visto da vaga: %w", err)
	}

	return nil
}

func (r *JobRepository) ReconcileStatuses(ctx context.Context, unknownBefore, expiredBefore time.Time) (job.StatusReconciliationResult, error) {
	var result job.StatusReconciliationResult

	markedUnknown, err := r.queries.MarkJobsUnknown(ctx, toTimestamptz(unknownBefore))
	if err != nil {
		r.logger.ErrorContext(ctx, "falha ao marcar vagas como desconhecidas",
			slog.Time("unknown_before", unknownBefore),
			slog.String("erro", err.Error()),
		)
		return result, fmt.Errorf("marcar vagas como desconhecidas: %w", err)
	}
	result.MarkedUnknown = markedUnknown

	markedExpired, err := r.queries.MarkJobsExpired(ctx, toTimestamptz(expiredBefore))
	if err != nil {
		r.logger.ErrorContext(ctx, "falha ao marcar vagas como expiradas",
			slog.Time("expired_before", expiredBefore),
			slog.String("erro", err.Error()),
		)
		return result, fmt.Errorf("marcar vagas como expiradas: %w", err)
	}
	result.MarkedExpired = markedExpired

	r.logger.InfoContext(ctx, "reconciliacao de status de vagas executada com sucesso",
		slog.Int64("marcadas_unknown", result.MarkedUnknown),
		slog.Int64("marcadas_expired", result.MarkedExpired),
	)

	return result, nil
}

func toDomainJob(m db.Job) job.Job {
	return job.Job{
		ID:             m.ID,
		ExternalID:     m.ExternalID,
		Title:          m.Title,
		Company:        m.Company,
		Description:    m.Description,
		City:           m.City,
		State:          m.State,
		Source:         m.Source,
		SourceURL:      m.SourceUrl,
		Fingerprint:    m.Fingerprint,
		WorkMode:       job.WorkMode(m.WorkMode),
		EmploymentType: job.EmploymentType(m.EmploymentType),
		SalaryMin:      fromNullableInt8(m.SalaryMin),
		SalaryMax:      fromNullableInt8(m.SalaryMax),
		PublishedAt:    fromNullableTimestamptz(m.PublishedAt),
		CollectedAt:    m.CollectedAt.Time,
		LastSeenAt:     m.LastSeenAt.Time,
		Status:         job.Status(m.Status),
		CreatedAt:      m.CreatedAt.Time,
		UpdatedAt:      m.UpdatedAt.Time,
	}
}

func toNullableInt8(val *int64) pgtype.Int8 {
	if val == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *val, Valid: true}
}

func fromNullableInt8(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	val := v.Int64
	return &val
}

func toNullableTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromNullableTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	val := t.Time
	return &val
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
