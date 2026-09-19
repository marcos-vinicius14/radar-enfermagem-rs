package collector

import (
	"fmt"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

type Deduplicator struct{}

func NewDeduplicator() *Deduplicator {
	return &Deduplicator{}
}

func (d *Deduplicator) DeduplicateBatch(jobs []job.Job) ([]job.Job, int) {
	if len(jobs) == 0 {
		return []job.Job{}, 0
	}

	seenKeys := make(map[string]struct{}, len(jobs))
	seenFingerprints := make(map[string]struct{}, len(jobs))
	result := make([]job.Job, 0, len(jobs))
	discarded := 0

	for _, j := range jobs {
		key := fmt.Sprintf("%s:%s", j.Source, j.ExternalID)
		if _, exists := seenKeys[key]; exists {
			discarded++
			continue
		}

		if j.Fingerprint != "" {
			if _, exists := seenFingerprints[j.Fingerprint]; exists {
				discarded++
				continue
			}
			seenFingerprints[j.Fingerprint] = struct{}{}
		}

		seenKeys[key] = struct{}{}
		result = append(result, j)
	}

	return result, discarded
}
