package gupy

import (
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"net/http"
	"time"
)

const (
	defaultUnimedURL = "https://unimedpoa.gupy.io"
)

var _ collector.Collector = (*UnimedCollector)(nil)

// UnimedCollector implementa a extração de vagas da Unimed Porto Alegre.
type UnimedCollector struct {
	*GupyCollector
}

func NewUnimedCollector(client *http.Client, timeout time.Duration) *UnimedCollector {
	return NewUnimedCollectorWithURL(defaultUnimedURL, client, timeout)
}

func NewUnimedCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *UnimedCollector {
	gupy := NewGupyCollector(GupyConfig{
		Name:          "unimed",
		CompanyName:   "Unimed Porto Alegre",
		BaseURL:       baseURL,
		PublicBaseURL: defaultUnimedURL,
		DefaultCity:   "Porto Alegre",
		DefaultState:  "RS",
	}, client, timeout)

	return &UnimedCollector{
		GupyCollector: gupy,
	}
}
