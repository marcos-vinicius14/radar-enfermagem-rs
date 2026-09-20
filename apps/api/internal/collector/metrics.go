package collector

import (
	"sync"
	"time"
)

// CollectorMetrics armazena os indicadores granulares de uma fonte específica em uma rodada.
type CollectorMetrics struct {
	CollectorName string        `json:"collector_name"`
	TotalFound    int           `json:"total_found"`
	Inserted      int           `json:"inserted"`
	Updated       int           `json:"updated"`
	Duplicates    int           `json:"duplicates"`
	Failed        int           `json:"failed"`
	Duration      time.Duration `json:"duration"`
	Error         string        `json:"error,omitempty"`
}

// BatchMetrics consolida os resultados agregados de uma execução completa de múltiplos coletores.
type BatchMetrics struct {
	RunID           string                      `json:"run_id"`
	StartedAt       time.Time                   `json:"started_at"`
	FinishedAt      time.Time                   `json:"finished_at"`
	Duration        time.Duration               `json:"duration"`
	TotalFound      int                         `json:"total_found"`
	TotalInserted   int                         `json:"total_inserted"`
	TotalUpdated    int                         `json:"total_updated"`
	TotalDuplicates int                         `json:"total_duplicates"`
	TotalFailed     int                         `json:"total_failed"`
	ByCollector     map[string]CollectorMetrics `json:"by_collector"`
	ErrorsBySource  map[string]string           `json:"errors_by_source,omitempty"`
}

// NewBatchMetrics inicializa a estrutura de métricas para uma nova rodada de coleta.
func NewBatchMetrics(runID string, startedAt time.Time) *BatchMetrics {
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	return &BatchMetrics{
		RunID:          runID,
		StartedAt:      startedAt,
		ByCollector:    make(map[string]CollectorMetrics),
		ErrorsBySource: make(map[string]string),
	}
}

// AddCollectorResult acumula os resultados de um coletor concluído.
func (b *BatchMetrics) AddCollectorResult(res CollectResult, err error) {
	cm := CollectorMetrics{
		CollectorName: res.CollectorName,
		TotalFound:    res.TotalFound,
		Inserted:      res.Inserted,
		Updated:       res.Updated,
		Duplicates:    res.Duplicates,
		Failed:        res.Failed,
		Duration:      res.Duration,
	}

	if err != nil {
		cm.Error = err.Error()
		b.ErrorsBySource[res.CollectorName] = err.Error()
	}

	b.TotalFound += res.TotalFound
	b.TotalInserted += res.Inserted
	b.TotalUpdated += res.Updated
	b.TotalDuplicates += res.Duplicates
	b.TotalFailed += res.Failed

	b.ByCollector[res.CollectorName] = cm
}

// Finish finaliza a contagem de tempo do lote de métricas.
func (b *BatchMetrics) Finish() {
	b.FinishedAt = time.Now().UTC()
	b.Duration = b.FinishedAt.Sub(b.StartedAt)
}

// CumulativeMetrics mantém contadores globais acumulados entre sucessivas execuções.
type CumulativeMetrics struct {
	TotalRuns       int64         `json:"total_runs"`
	TotalFound      int64         `json:"total_found"`
	TotalInserted   int64         `json:"total_inserted"`
	TotalUpdated    int64         `json:"total_updated"`
	TotalDuplicates int64         `json:"total_duplicates"`
	TotalFailed     int64         `json:"total_failed"`
	TotalDuration   time.Duration `json:"total_duration"`
}

// MetricsTracker mantém em memória o estado da última rodada e os contadores acumulados.
type MetricsTracker struct {
	mu         sync.RWMutex
	lastBatch  *BatchMetrics
	cumulative CumulativeMetrics
}

func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{}
}

// RecordBatch atualiza as métricas acumuladas e a última rodada executada.
func (m *MetricsTracker) RecordBatch(batch *BatchMetrics) {
	if batch == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastBatch = batch
	m.cumulative.TotalRuns++
	m.cumulative.TotalFound += int64(batch.TotalFound)
	m.cumulative.TotalInserted += int64(batch.TotalInserted)
	m.cumulative.TotalUpdated += int64(batch.TotalUpdated)
	m.cumulative.TotalDuplicates += int64(batch.TotalDuplicates)
	m.cumulative.TotalFailed += int64(batch.TotalFailed)
	m.cumulative.TotalDuration += batch.Duration
}

// LastBatch retorna uma cópia da última rodada de métricas registrada.
func (m *MetricsTracker) LastBatch() (BatchMetrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastBatch == nil {
		return BatchMetrics{}, false
	}
	return *m.lastBatch, true
}

// Cumulative retorna os totais acumulados de todas as coletas executadas.
func (m *MetricsTracker) Cumulative() CumulativeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cumulative
}
