package collector

import "time"

// RawJob representa a oportunidade em formato bruto extraída diretamente da fonte externa.
// É propositalmente tolerante a dados parciais ou desformatados, servindo de estágio intermediário
// antes do saneamento e validação pelo Normalizer.
type RawJob struct {
	ExternalID     string         `json:"external_id"`
	Title          string         `json:"title"`
	Company        string         `json:"company"`
	Description    string         `json:"description"`
	City           string         `json:"city"`
	State          string         `json:"state"`
	Source         string         `json:"source"`
	SourceURL      string         `json:"source_url"`
	WorkMode       string         `json:"work_mode,omitempty"`
	EmploymentType string         `json:"employment_type,omitempty"`
	SalaryMin      *int64         `json:"salary_min,omitempty"`
	SalaryMax      *int64         `json:"salary_max,omitempty"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	RawData        map[string]any `json:"raw_data,omitempty"`
}
