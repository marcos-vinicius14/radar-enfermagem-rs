package gupy

import (
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"net/http"
	"time"
)

const (
	defaultMoinhosURL = "https://hospitalmoinhos.gupy.io"
)

var _ collector.Collector = (*MoinhosCollector)(nil)

type MoinhosCollector struct {
	*GupyCollector
}

func NewMoinhosCollector(client *http.Client, timeout time.Duration) *MoinhosCollector {
	return NewMoinhosCollectorWithURL(defaultMoinhosURL, client, timeout)
}

func NewMoinhosCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *MoinhosCollector {
	gupy := NewGupyCollector(GupyConfig{
		Name:          "moinhos",
		CompanyName:   "Hospital Moinhos de Vento",
		BaseURL:       baseURL,
		PublicBaseURL: defaultMoinhosURL,
		DefaultCity:   "Porto Alegre",
		DefaultState:  "RS",
	}, client, timeout)

	return &MoinhosCollector{
		GupyCollector: gupy,
	}
}
