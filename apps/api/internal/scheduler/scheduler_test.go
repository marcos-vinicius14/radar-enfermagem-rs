package scheduler_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/scheduler"
)

type mockRepo struct {
	mu           sync.Mutex
	reconcileCnt int
}

func (m *mockRepo) Insert(ctx context.Context, j job.Job) (job.Job, error) {
	return j, nil
}
func (m *mockRepo) Update(ctx context.Context, j job.Job) (job.Job, error) {
	return j, nil
}
func (m *mockRepo) FindByID(ctx context.Context, id uuid.UUID) (job.Job, error) {
	return job.Job{}, job.ErrNotFound
}
func (m *mockRepo) FindBySourceAndExternalID(ctx context.Context, source, externalID string) (job.Job, error) {
	return job.Job{}, job.ErrNotFound
}
func (m *mockRepo) FindByFingerprint(ctx context.Context, fingerprint string) (job.Job, error) {
	return job.Job{}, job.ErrNotFound
}
func (m *mockRepo) List(ctx context.Context, params job.ListParams) ([]job.Job, error) {
	return nil, nil
}
func (m *mockRepo) UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeenAt time.Time) error {
	return nil
}
func (m *mockRepo) ReconcileStatuses(ctx context.Context, unknownBefore, expiredBefore time.Time) (job.StatusReconciliationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reconcileCnt++
	return job.StatusReconciliationResult{MarkedUnknown: 2, MarkedExpired: 1}, nil
}

type mockCollector struct {
	name  string
	delay time.Duration
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) Collect(ctx context.Context, query collector.SearchQuery) ([]collector.RawJob, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return []collector.RawJob{
		{
			ExternalID: "mock-1",
			Title:      "Técnico em Enfermagem",
			Company:    "Hospital Teste",
			Source:     m.name,
			SourceURL:  "https://teste.com/1",
		},
	}, nil
}

func TestScheduler_InvalidCronExpression(t *testing.T) {
	repo := &mockRepo{}
	svc := collector.NewService(repo, nil, nil, nil)
	reg := collector.NewRegistry()

	_, err := scheduler.NewScheduler(scheduler.Config{
		CronSchedule: "expressao_invalida_de_cron",
	}, svc, reg, nil)

	if err == nil {
		t.Fatalf("esperava erro para expressao cron invalida, obteve nil")
	}
}

func TestScheduler_TriggerNow(t *testing.T) {
	repo := &mockRepo{}
	svc := collector.NewService(repo, nil, nil, nil)
	reg := collector.NewRegistry()
	reg.Register(&mockCollector{name: "portal-teste"})

	sched, err := scheduler.NewScheduler(scheduler.Config{
		CronSchedule: "0 */2 * * *",
		Concurrency:  2,
	}, svc, reg, nil)
	if err != nil {
		t.Fatalf("NewScheduler() erro: %v", err)
	}

	ctx := context.Background()
	metrics, err := sched.TriggerNow(ctx)
	if err != nil {
		t.Fatalf("TriggerNow() erro: %v", err)
	}

	if metrics.TotalFound != 1 {
		t.Errorf("TotalFound = %d, esperado 1", metrics.TotalFound)
	}

	lastMetrics, ok := sched.LastMetrics()
	if !ok {
		t.Fatalf("esperava LastMetrics() ok=true")
	}
	if lastMetrics.TotalFound != 1 {
		t.Errorf("lastMetrics.TotalFound = %d, esperado 1", lastMetrics.TotalFound)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.reconcileCnt != 1 {
		t.Errorf("ReconcileStatuses chamadas = %d, esperado 1", repo.reconcileCnt)
	}
}

func TestScheduler_PreventionsOverlap(t *testing.T) {
	repo := &mockRepo{}
	svc := collector.NewService(repo, nil, nil, nil)
	reg := collector.NewRegistry()
	// Coletor com delay de 100ms
	reg.Register(&mockCollector{name: "portal-lento", delay: 100 * time.Millisecond})

	sched, err := scheduler.NewScheduler(scheduler.Config{
		CronSchedule: "0 */2 * * *",
		Concurrency:  1,
	}, svc, reg, nil)
	if err != nil {
		t.Fatalf("NewScheduler() erro: %v", err)
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)

	var firstErr, secondErr error
	go func() {
		defer wg.Done()
		_, firstErr = sched.TriggerNow(ctx)
	}()

	// Pequeno delay para garantir que a primeira iniciou
	time.Sleep(20 * time.Millisecond)

	// Segunda chamada enquanto a primeira está ativa deve retornar ErrAlreadyRunning
	_, secondErr = sched.TriggerNow(ctx)

	wg.Wait()

	if firstErr != nil {
		t.Errorf("primeira execucao falhou: %v", firstErr)
	}
	if secondErr == nil || !errors.Is(secondErr, scheduler.ErrAlreadyRunning) {
		t.Errorf("esperava scheduler.ErrAlreadyRunning na segunda execucao, obteve: %v", secondErr)
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	repo := &mockRepo{}
	svc := collector.NewService(repo, nil, nil, nil)
	reg := collector.NewRegistry()

	sched, err := scheduler.NewScheduler(scheduler.Config{
		CronSchedule: "@every 1s",
	}, svc, reg, nil)
	if err != nil {
		t.Fatalf("NewScheduler() erro: %v", err)
	}

	ctx := context.Background()
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start() erro: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	sched.Stop()
}
