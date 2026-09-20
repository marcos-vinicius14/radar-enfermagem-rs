package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// PruneResult contém as estatísticas da execução do expurgo.
type PruneResult struct {
	TotalChecked int64
	TotalPruned  int64
	PrunedTitles []string
	Duration     time.Duration
}

// Pruner é o serviço responsável por varrer o banco de dados e expurgar vagas
// que não pertencem ao domínio de enfermagem ou que estão fora do escopo geográfico.
type Pruner struct {
	repo   Repository
	logger *slog.Logger
}

// NewPruner instancia o serviço de saneamento e expurgo de vagas.
func NewPruner(repo Repository, logger *slog.Logger) *Pruner {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pruner{
		repo:   repo,
		logger: logger,
	}
}

// Prune executa a varredura atômica em lotes e deleta registros que violam as regras de domínio.
func (p *Pruner) Prune(ctx context.Context) (PruneResult, error) {
	start := time.Now()
	p.logger.InfoContext(ctx, "iniciando saneamento e expurgo de vagas fora do dominio de enfermagem")

	const batchSize int32 = 200
	var offset int32 = 0
	var totalChecked int64 = 0
	var totalPruned int64 = 0
	prunedTitles := make([]string, 0)

	for {
		jobs, err := p.repo.ListActiveForPruning(ctx, batchSize, offset)
		if err != nil {
			return PruneResult{
				TotalChecked: totalChecked,
				TotalPruned:  totalPruned,
				Duration:     time.Since(start),
			}, fmt.Errorf("buscar vagas para expurgo no offset %d: %w", offset, err)
		}

		if len(jobs) == 0 {
			break
		}

		invalidIDs := make([]uuid.UUID, 0)
		for _, j := range jobs {
			totalChecked++
			// Vaga inválida se não pertencer à enfermagem OU não estiver no escopo metropolitano
			if !IsNursingJob(j.Title, j.Description) || !IsTargetLocation(j.City, j.State) {
				invalidIDs = append(invalidIDs, j.ID)
				if len(prunedTitles) < 50 {
					prunedTitles = append(prunedTitles, fmt.Sprintf("%s (%s - %s)", j.Title, j.Company, j.City))
				}
			}
		}

		if len(invalidIDs) > 0 {
			deleted, err := p.repo.DeleteByIDs(ctx, invalidIDs)
			if err != nil {
				return PruneResult{
					TotalChecked: totalChecked,
					TotalPruned:  totalPruned,
					Duration:     time.Since(start),
				}, fmt.Errorf("deletar lote de vagas fora de escopo: %w", err)
			}
			totalPruned += deleted
		}

		if int32(len(jobs)) < batchSize {
			break
		}

		// Ajusta o offset considerando as linhas excluídas
		offset += int32(len(jobs) - len(invalidIDs))
	}

	duration := time.Since(start)
	p.logger.InfoContext(ctx, "saneamento e expurgo concluido com sucesso",
		slog.Int64("total_verificadas", totalChecked),
		slog.Int64("total_removidas", totalPruned),
		slog.Duration("duracao", duration),
	)

	return PruneResult{
		TotalChecked: totalChecked,
		TotalPruned:  totalPruned,
		PrunedTitles: prunedTitles,
		Duration:     duration,
	}, nil
}
