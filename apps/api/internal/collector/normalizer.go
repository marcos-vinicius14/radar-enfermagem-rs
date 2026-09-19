package collector

import (
	"fmt"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

func (n *Normalizer) Normalize(raw RawJob) (job.Job, error) {
	title := cleanSpaces(raw.Title)
	company := cleanSpaces(raw.Company)
	city := cleanSpaces(raw.City)
	state := normalizeState(cleanSpaces(raw.State), city)
	source := strings.ToLower(cleanSpaces(raw.Source))
	sourceURL := strings.TrimSpace(raw.SourceURL)
	description := strings.TrimSpace(raw.Description)

	now := time.Now().UTC()

	fp := job.Fingerprint(company, title, city)

	j := job.Job{
		ExternalID:     strings.TrimSpace(raw.ExternalID),
		Title:          title,
		Company:        company,
		Description:    description,
		City:           city,
		State:          state,
		Source:         source,
		SourceURL:      sourceURL,
		Fingerprint:    fp,
		WorkMode:       normalizeWorkMode(raw.WorkMode),
		EmploymentType: normalizeEmploymentType(raw.EmploymentType),
		SalaryMin:      raw.SalaryMin,
		SalaryMax:      raw.SalaryMax,
		PublishedAt:    raw.PublishedAt,
		CollectedAt:    now,
		LastSeenAt:     now,
		Status:         job.StatusActive,
	}

	if err := j.Validate(); err != nil {
		return job.Job{}, fmt.Errorf("normalizar vaga: %w", err)
	}

	return j, nil
}

func cleanSpaces(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func normalizeState(state, city string) string {
	normalized := strings.ToUpper(job.NormalizeText(state))
	if normalized == "RS" || normalized == "RIO GRANDE DO SUL" {
		return "RS"
	}

	if state == "" && isMetropolitanRegionRS(city) {
		return "RS"
	}

	if len(state) == 2 {
		return strings.ToUpper(state)
	}

	return state
}

func isMetropolitanRegionRS(city string) bool {
	normalizedCity := job.NormalizeText(city)
	switch normalizedCity {
	case "porto alegre", "canoas", "novo hamburgo", "sao leopoldo", "gravatai",
		"viamao", "alvorada", "cachoeirinha", "esteio", "sapucaia do sul",
		"guaiba", "eldorado do sul", "campo bom", "sapiranga", "dois irmaos":
		return true
	default:
		return false
	}
}

func normalizeWorkMode(raw string) job.WorkMode {
	norm := job.NormalizeText(raw)
	switch {
	case strings.Contains(norm, "on-site") || strings.Contains(norm, "presencial") || strings.Contains(norm, "on_site"):
		return job.WorkModeOnSite
	case strings.Contains(norm, "remoto") || strings.Contains(norm, "remote") || strings.Contains(norm, "home office"):
		return job.WorkModeRemote
	case strings.Contains(norm, "hibrido") || strings.Contains(norm, "hybrid"):
		return job.WorkModeHybrid
	default:
		return ""
	}
}

func normalizeEmploymentType(raw string) job.EmploymentType {
	norm := job.NormalizeText(raw)
	switch {
	case strings.Contains(norm, "efetivo") || strings.Contains(norm, "clt") || strings.Contains(norm, "effective") || strings.Contains(norm, "full"):
		return job.EmploymentTypeFullTime
	case strings.Contains(norm, "estagio") || strings.Contains(norm, "internship"):
		return job.EmploymentTypeInternship
	case strings.Contains(norm, "temporario") || strings.Contains(norm, "temporary"):
		return job.EmploymentTypeTemporary
	case strings.Contains(norm, "meio") || strings.Contains(norm, "part"):
		return job.EmploymentTypePartTime
	default:
		return ""
	}
}
