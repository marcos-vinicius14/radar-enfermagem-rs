package sources

import (
	"net/http"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/custom"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/gupy"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/senior"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector/sources/vagascom"
)

// RegisterAll registra todos os 9 portais hospitalares monitorados no registro de coletores fornecido.
func RegisterAll(r *collector.Registry, client *http.Client, timeout time.Duration) {
	r.Register(gupy.NewSantaCasaCollector(client, timeout))
	r.Register(gupy.NewMoinhosCollector(client, timeout))
	r.Register(gupy.NewSaoLucasCollector(client, timeout))
	r.Register(gupy.NewUnimedCollector(client, timeout))
	r.Register(gupy.NewDoctorClinCollector(client, timeout))
	r.Register(vagascom.NewFleuryCollector(client, timeout))
	r.Register(custom.NewHCPACollector(client, timeout))
	r.Register(custom.NewDivinaCollector(client, timeout))
	r.Register(senior.NewMaeDeDeusCollector(client, timeout))
}

// NewDefaultRegistry retorna um registro inicializado com todos os 9 portais hospitalares monitorados,
// utilizando o cliente HTTP fornecido (permitindo injeção de cliente resiliente com rate limit e retries).
func NewDefaultRegistry(client *http.Client, timeout time.Duration) *collector.Registry {
	r := collector.NewRegistry()
	RegisterAll(r, client, timeout)
	return r
}

// DefaultRegistry retorna um registro inicializado com todos os 9 portais hospitalares monitorados usando cliente padrão.
func DefaultRegistry(timeout time.Duration) *collector.Registry {
	return NewDefaultRegistry(nil, timeout)
}
