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
	ID             uuid.UUID
	ExternalID     string
	Title          string
	Company        string
	Description    string
	City           string
	State          string
	Source         string
	SourceURL      string
	Fingerprint    string
	WorkMode       WorkMode
	EmploymentType EmploymentType
	SalaryMin      *int64
	SalaryMax      *int64
	PublishedAt    *time.Time
	CollectedAt    time.Time
	LastSeenAt     time.Time
	Status         Status
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
