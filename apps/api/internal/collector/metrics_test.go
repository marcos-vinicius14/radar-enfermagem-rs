package collector_test

import (
	"errors"
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
)

func TestBatchMetrics_AddCollectorResult(t *testing.T) {
	bm := collector.NewBatchMetrics("test-run-1", time.Now().UTC())

	res1 := collector.CollectResult{
		CollectorName: "santacasa",
		TotalFound:    5,
		Inserted:      3,
		Updated:       1,
		Duplicates:    1,
		Failed:        0,
		Duration:      150 * time.Millisecond,
	}
	bm.AddCollectorResult(res1, nil)

	res2 := collector.CollectResult{
		CollectorName: "moinhos",
		TotalFound:    2,
		Inserted:      1,
		Updated:       0,
		Duplicates:    0,
		Failed:        1,
		Duration:      200 * time.Millisecond,
	}
	bm.AddCollectorResult(res2, errors.New("erro de conexao"))

	bm.Finish()

	if bm.TotalFound != 7 {
		t.Errorf("TotalFound = %d, esperado 7", bm.TotalFound)
	}
	if bm.TotalInserted != 4 {
		t.Errorf("TotalInserted = %d, esperado 4", bm.TotalInserted)
	}
	if bm.TotalUpdated != 1 {
		t.Errorf("TotalUpdated = %d, esperado 1", bm.TotalUpdated)
	}
	if bm.TotalDuplicates != 1 {
		t.Errorf("TotalDuplicates = %d, esperado 1", bm.TotalDuplicates)
	}
	if bm.TotalFailed != 1 {
		t.Errorf("TotalFailed = %d, esperado 1", bm.TotalFailed)
	}
	if len(bm.ErrorsBySource) != 1 || bm.ErrorsBySource["moinhos"] == "" {
		t.Errorf("esperava erro registrado para moinhos, obteve: %v", bm.ErrorsBySource)
	}
	if len(bm.ByCollector) != 2 {
		t.Errorf("ByCollector tamanho = %d, esperado 2", len(bm.ByCollector))
	}
}

func TestMetricsTracker_ThreadSafe(t *testing.T) {
	tracker := collector.NewMetricsTracker()

	bm1 := collector.NewBatchMetrics("run-1", time.Now().UTC())
	bm1.AddCollectorResult(collector.CollectResult{
		CollectorName: "santacasa",
		TotalFound:    10,
		Inserted:      8,
		Updated:       2,
	}, nil)
	bm1.Finish()

	tracker.RecordBatch(bm1)

	last, ok := tracker.LastBatch()
	if !ok {
		t.Fatalf("esperava LastBatch() ok=true")
	}
	if last.RunID != "run-1" {
		t.Errorf("RunID = %s, esperado run-1", last.RunID)
	}

	cumulative := tracker.Cumulative()
	if cumulative.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, esperado 1", cumulative.TotalRuns)
	}
	if cumulative.TotalFound != 10 {
		t.Errorf("Cumulative.TotalFound = %d, esperado 10", cumulative.TotalFound)
	}
	if cumulative.TotalInserted != 8 {
		t.Errorf("Cumulative.TotalInserted = %d, esperado 8", cumulative.TotalInserted)
	}
}
