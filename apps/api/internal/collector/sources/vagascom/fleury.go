package vagascom

import (
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"net/http"
	"time"
)

const (
	defaultFleuryURL = "https://trabalheconosco.vagas.com.br/grupo-fleury/oportunidades"
)

var _ collector.Collector = (*FleuryCollector)(nil)

type FleuryCollector struct {
	*VagasComCollector
}

func NewFleuryCollector(client *http.Client, timeout time.Duration) *FleuryCollector {
	return NewFleuryCollectorWithURL(defaultFleuryURL, client, timeout)
}

func NewFleuryCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *FleuryCollector {
	vagas := NewVagasComCollector(VagasComConfig{
		Name:          "fleury",
		CompanyName:   "Grupo Fleury / Weinmann",
		BaseURL:       baseURL,
		PublicBaseURL: "https://trabalheconosco.vagas.com.br",
		DefaultCity:   "Porto Alegre",
		DefaultState:  "RS",
	}, client, timeout)

	return &FleuryCollector{
		VagasComCollector: vagas,
	}
}
