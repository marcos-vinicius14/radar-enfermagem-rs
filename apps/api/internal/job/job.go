package job

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("vaga não encontrada")

	ErrInvalidJob = errors.New("dados da vaga inválidos")
)

type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusUnknown Status = "UNKNOWN"
	StatusExpired Status = "EXPIRED"
)

type WorkMode string

const (
	WorkModeOnSite WorkMode = "ON_SITE"
	WorkModeRemote WorkMode = "REMOTE"
	WorkModeHybrid WorkMode = "HYBRID"
)

type EmploymentType string

const (
	EmploymentTypeFullTime   EmploymentType = "FULL_TIME"
	EmploymentTypePartTime   EmploymentType = "PART_TIME"
	EmploymentTypeTemporary  EmploymentType = "TEMPORARY"
	EmploymentTypeInternship EmploymentType = "INTERNSHIP"
)

type Job struct {
	ID             uuid.UUID      `json:"id"`
	ExternalID     string         `json:"external_id"`
	Title          string         `json:"title"`
	Company        string         `json:"company"`
	Description    string         `json:"description"`
	City           string         `json:"city"`
	State          string         `json:"state"`
	Source         string         `json:"source"`
	SourceURL      string         `json:"source_url"`
	Fingerprint    string         `json:"fingerprint,omitempty"`
	WorkMode       WorkMode       `json:"work_mode"`
	EmploymentType EmploymentType `json:"employment_type"`
	SalaryMin      *int64         `json:"salary_min"`
	SalaryMax      *int64         `json:"salary_max"`
	PublishedAt    *time.Time     `json:"published_at"`
	CollectedAt    time.Time      `json:"collected_at"`
	LastSeenAt     time.Time      `json:"last_seen_at"`
	Status         Status         `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (j *Job) Validate() error {
	if strings.TrimSpace(j.Title) == "" {
		return errors.New("o título da vaga é obrigatório")
	}
	if strings.TrimSpace(j.Company) == "" {
		return errors.New("a instituição/empresa da vaga é obrigatória")
	}
	if strings.TrimSpace(j.Source) == "" {
		return errors.New("a fonte da vaga é obrigatória")
	}
	if strings.TrimSpace(j.SourceURL) == "" {
		return errors.New("a URL de origem da vaga é obrigatória")
	}
	if strings.TrimSpace(j.Fingerprint) == "" {
		return errors.New("o fingerprint da vaga é obrigatório")
	}
	if j.SalaryMin != nil && j.SalaryMax != nil && *j.SalaryMin > *j.SalaryMax {
		return errors.New("o salário mínimo não pode ser maior que o salário máximo")
	}
	return nil
}
