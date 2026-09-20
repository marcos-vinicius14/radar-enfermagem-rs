package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
	"github.com/robfig/cron/v3"
)

var (
	ErrAlreadyRunning = errors.New("execução anterior de coleta ainda em andamento")
	ErrInvalidCron    = errors.New("expressão cron inválida")
)

// Config define os parâmetros de agendamento e execução periódica de coleta.
type Config struct {
	CronSchedule       string
	Concurrency        int
	UnknownThreshold   time.Duration
	ExpiredThreshold   time.Duration
	Query              collector.SearchQuery
	PrunerCronSchedule string
	Pruner             *job.Pruner
}

// Scheduler gerencia a execução periódica e automática dos coletores de vagas.
type Scheduler struct {
	cfg         Config
	svc         *collector.Service
	registry    *collector.Registry
	pruner      *job.Pruner
	tracker     *collector.MetricsTracker
	logger      *slog.Logger
	cron        *cron.Cron
	entryID     cron.EntryID
	isRunning   atomic.Bool
	activeRunMu sync.RWMutex
	activeRunID string
}

// NewScheduler instancia um novo agendador com validação rigorosa de expressão cron e injeção de logger.
func NewScheduler(cfg Config, svc *collector.Service, registry *collector.Registry, logger *slog.Logger) (*Scheduler, error) {
	if cfg.CronSchedule == "" {
		cfg.CronSchedule = "0 */2 * * *" // A cada 2 horas por padrão
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 3
	}
	if cfg.UnknownThreshold <= 0 {
		cfg.UnknownThreshold = 24 * time.Hour
	}
	if cfg.ExpiredThreshold <= 0 {
		cfg.ExpiredThreshold = 7 * 24 * time.Hour
	}
	if logger == nil {
		logger = slog.Default()
	}

	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := cronParser.Parse(cfg.CronSchedule)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %v", ErrInvalidCron, cfg.CronSchedule, err)
	}

	c := cron.New(cron.WithParser(cronParser))

	s := &Scheduler{
		cfg:      cfg,
		svc:      svc,
		registry: registry,
		pruner:   cfg.Pruner,
		tracker:  collector.NewMetricsTracker(),
		logger:   logger,
		cron:     c,
	}

	entryID := c.Schedule(schedule, cron.FuncJob(func() {
		ctx := context.Background()
		_, runErr := s.TriggerNow(ctx)
		if runErr != nil && !errors.Is(runErr, ErrAlreadyRunning) {
			s.logger.ErrorContext(ctx, "falha durante execucao agendada do cron",
				slog.String("erro", runErr.Error()),
			)
		}
	}))
	s.entryID = entryID

	if cfg.Pruner != nil {
		prunerCron := cfg.PrunerCronSchedule
		if prunerCron == "" {
			prunerCron = "0 */12 * * *"
		}
		prunerSched, pErr := cronParser.Parse(prunerCron)
		if pErr != nil {
			return nil, fmt.Errorf("%w no pruner cron %q: %v", ErrInvalidCron, prunerCron, pErr)
		}
		c.Schedule(prunerSched, cron.FuncJob(func() {
			ctx := context.Background()
			res, err := s.pruner.Prune(ctx)
			if err != nil {
				s.logger.ErrorContext(ctx, "falha durante execucao agendada do pruner",
					slog.String("erro", err.Error()),
				)
			} else {
				s.logger.InfoContext(ctx, "execucao agendada do pruner concluida com sucesso",
					slog.Int64("total_verificadas", res.TotalChecked),
					slog.Int64("total_removidas", res.TotalPruned),
					slog.Duration("duracao", res.Duration),
				)
			}
		}))
	}

	return s, nil
}

// Start inicia o loop do agendador em background.
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "iniciando scheduler de coleta de vagas",
		slog.String("cron_schedule", s.cfg.CronSchedule),
		slog.Int("concorrencia", s.cfg.Concurrency),
		slog.Duration("unknown_threshold", s.cfg.UnknownThreshold),
		slog.Duration("expired_threshold", s.cfg.ExpiredThreshold),
	)

	if s.pruner != nil {
		prunerCron := s.cfg.PrunerCronSchedule
		if prunerCron == "" {
			prunerCron = "0 */12 * * *"
		}
		s.logger.InfoContext(ctx, "iniciando scheduler de saneamento/expurgo de vagas fora de dominio",
			slog.String("pruner_cron", prunerCron),
		)
	}

	s.cron.Start()
	return nil
}

// TriggerPruneNow executa o saneamento e expurgo de vagas imediatamente sob demanda.
func (s *Scheduler) TriggerPruneNow(ctx context.Context) (job.PruneResult, error) {
	if s.pruner == nil {
		return job.PruneResult{}, errors.New("pruner não configurado no scheduler")
	}
	return s.pruner.Prune(ctx)
}

// Stop finaliza graciosamente o agendador aguardando a conclusão de tarefas em andamento.
func (s *Scheduler) Stop() {
	s.logger.Info("encerrando scheduler de coleta de vagas...")
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
	s.logger.Info("scheduler de coleta de vagas encerrado com sucesso")
}

// TriggerNow executa uma rodada imediata de coleta e reconciliação de status.
// Previne execuções sobrepostas caso uma coleta anterior ainda esteja ativa.
func (s *Scheduler) TriggerNow(ctx context.Context) (collector.BatchMetrics, error) {
	if !s.isRunning.CompareAndSwap(false, true) {
		s.activeRunMu.RLock()
		activeID := s.activeRunID
		s.activeRunMu.RUnlock()

		s.logger.WarnContext(ctx, "coleta ignorada: execucao anterior ainda em andamento",
			slog.String("run_id_ativo", activeID),
		)
		return collector.BatchMetrics{}, ErrAlreadyRunning
	}
	defer s.isRunning.Store(false)

	runID := uuid.Must(uuid.NewV7()).String()

	s.activeRunMu.Lock()
	s.activeRunID = runID
	s.activeRunMu.Unlock()

	defer func() {
		s.activeRunMu.Lock()
		s.activeRunID = ""
		s.activeRunMu.Unlock()
	}()

	s.logger.InfoContext(ctx, "disparando rodada de coleta de vagas",
		slog.String("run_id", runID),
	)

	targets := s.registry.All()
	metrics, err := s.svc.CollectAll(ctx, targets, s.cfg.Query, s.cfg.Concurrency)
	if err != nil {
		s.logger.ErrorContext(ctx, "falha no ciclo de coleta concorrente",
			slog.String("run_id", runID),
			slog.String("erro", err.Error()),
		)
		return metrics, fmt.Errorf("executar coleta concorrente: %w", err)
	}

	// Reconcilia status de vagas não vistas
	reconcileRes, recErr := s.svc.ReconcileJobStatuses(ctx, s.cfg.UnknownThreshold, s.cfg.ExpiredThreshold)
	if recErr != nil {
		s.logger.ErrorContext(ctx, "falha na reconciliacao de status apos coleta",
			slog.String("run_id", runID),
			slog.String("erro", recErr.Error()),
		)
	} else {
		s.logger.InfoContext(ctx, "reconciliacao de status concluida no scheduler",
			slog.String("run_id", runID),
			slog.Int64("marcadas_unknown", reconcileRes.MarkedUnknown),
			slog.Int64("marcadas_expired", reconcileRes.MarkedExpired),
		)
	}

	s.tracker.RecordBatch(&metrics)

	return metrics, nil
}

// LastMetrics retorna as métricas consolidadas da última rodada de coleta.
func (s *Scheduler) LastMetrics() (collector.BatchMetrics, bool) {
	return s.tracker.LastBatch()
}

// CumulativeMetrics retorna os totais acumulados pelo agendador.
func (s *Scheduler) CumulativeMetrics() collector.CumulativeMetrics {
	return s.tracker.Cumulative()
}
