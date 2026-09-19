package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

// CollectResult agrega os indicadores de performance e contadores da execução de uma coleta.
type CollectResult struct {
	CollectorName string        `json:"collector_name"`
	TotalFound    int           `json:"total_found"`
	Normalized    int           `json:"normalized"`
	Inserted      int           `json:"inserted"`
	Updated       int           `json:"updated"`
	Duplicates    int           `json:"duplicates"`
	Failed        int           `json:"failed"`
	Duration      time.Duration `json:"duration"`
	Errors        []error       `json:"errors,omitempty"`
}

// Service orquestra o fluxo ponta a ponta de coleta, saneamento, deduplicação e persistência.
type Service struct {
	repo         job.Repository
	normalizer   *Normalizer
	deduplicator *Deduplicator
	logger       *slog.Logger
}

// NewService constrói uma nova instância do serviço de orquestração de coleta.
func NewService(repo job.Repository, normalizer *Normalizer, deduplicator *Deduplicator, logger *slog.Logger) *Service {
	if normalizer == nil {
		normalizer = NewNormalizer()
	}
	if deduplicator == nil {
		deduplicator = NewDeduplicator()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:         repo,
		normalizer:   normalizer,
		deduplicator: deduplicator,
		logger:       logger,
	}
}

// CollectFrom executa a coleta de uma fonte externa específica através do Collector fornecido.
// O fluxo é idempotente: atualiza registros existentes e descarta duplicatas lógicas.
func (s *Service) CollectFrom(ctx context.Context, c Collector, query SearchQuery) (CollectResult, error) {
	start := time.Now()
	result := CollectResult{
		CollectorName: c.Name(),
	}

	rawJobs, err := c.Collect(ctx, query)
	if err != nil {
		s.logger.ErrorContext(ctx, "falha ao executar coleta",
			slog.String("collector", c.Name()),
			slog.String("erro", err.Error()),
		)
		return result, fmt.Errorf("executar coletor %s: %w", c.Name(), err)
	}

	result.TotalFound = len(rawJobs)
	validJobs := make([]job.Job, 0, len(rawJobs))

	for _, raw := range rawJobs {
		normalized, err := s.normalizer.Normalize(raw)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
			s.logger.WarnContext(ctx, "vaga rejeitada na normalizacao",
				slog.String("collector", c.Name()),
				slog.String("external_id", raw.ExternalID),
				slog.String("erro", err.Error()),
			)
			continue
		}
		validJobs = append(validJobs, normalized)
	}

	result.Normalized = len(validJobs)
	dedupedJobs, inBatchDiscarded := s.deduplicator.DeduplicateBatch(validJobs)
	result.Duplicates += inBatchDiscarded

	now := time.Now().UTC()

	for _, j := range dedupedJobs {
		// 1. Verifica se já existe por source + external_id
		existing, err := s.repo.FindBySourceAndExternalID(ctx, j.Source, j.ExternalID)
		if err == nil {
			// Vaga já existe: atualiza dados cadastrais e last_seen_at
			j.ID = existing.ID
			j.CreatedAt = existing.CreatedAt
			j.LastSeenAt = now

			if _, updateErr := s.repo.Update(ctx, j); updateErr != nil {
				result.Failed++
				result.Errors = append(result.Errors, updateErr)
				continue
			}

			if updateSeenErr := s.repo.UpdateLastSeen(ctx, existing.ID, now); updateSeenErr != nil {
				s.logger.WarnContext(ctx, "falha ao atualizar last_seen_at da vaga",
					slog.String("id", existing.ID.String()),
					slog.String("erro", updateSeenErr.Error()),
				)
			}

			result.Updated++
			continue
		}

		if !errors.Is(err, job.ErrNotFound) {
			result.Failed++
			result.Errors = append(result.Errors, err)
			continue
		}

		// 2. Se não existe por source + external_id, verifica se já existe por fingerprint (duplicata lógica entre fontes)
		existingFP, err := s.repo.FindByFingerprint(ctx, j.Fingerprint)
		if err == nil {
			if updateSeenErr := s.repo.UpdateLastSeen(ctx, existingFP.ID, now); updateSeenErr != nil {
				s.logger.WarnContext(ctx, "falha ao atualizar last_seen_at de vaga duplicada por fingerprint",
					slog.String("id", existingFP.ID.String()),
					slog.String("erro", updateSeenErr.Error()),
				)
			}
			result.Duplicates++
			continue
		}

		if !errors.Is(err, job.ErrNotFound) {
			result.Failed++
			result.Errors = append(result.Errors, err)
			continue
		}

		// 3. Vaga totalmente nova: insere no banco
		j.LastSeenAt = now
		j.CollectedAt = now
		if _, insertErr := s.repo.Insert(ctx, j); insertErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, insertErr)
			continue
		}

		result.Inserted++
	}

	result.Duration = time.Since(start)

	s.logger.InfoContext(ctx, "coleta finalizada com sucesso",
		slog.String("collector", c.Name()),
		slog.Int("total_encontradas", result.TotalFound),
		slog.Int("inseridas", result.Inserted),
		slog.Int("atualizadas", result.Updated),
		slog.Int("duplicadas", result.Duplicates),
		slog.Int("falhas", result.Failed),
		slog.Duration("duracao", result.Duration),
	)

	return result, nil
}
