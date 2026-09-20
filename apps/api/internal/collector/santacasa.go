package collector

import (
	"net/http"
	"time"
)

const (
	defaultSantaCasaURL = "https://santacasa.gupy.io"
)

var _ Collector = (*SantaCasaCollector)(nil)

type SantaCasaCollector struct {
	*GupyCollector
}

func NewSantaCasaCollector(client *http.Client, timeout time.Duration) *SantaCasaCollector {
	return NewSantaCasaCollectorWithURL(defaultSantaCasaURL, client, timeout)
}

func NewSantaCasaCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *SantaCasaCollector {
	gupy := NewGupyCollector(GupyConfig{
		Name:          "santacasa",
		CompanyName:   "Santa Casa de Porto Alegre",
		BaseURL:       baseURL,
		PublicBaseURL: defaultSantaCasaURL,
		DefaultCity:   "Porto Alegre",
		DefaultState:  "RS",
	}, client, timeout)

	return &SantaCasaCollector{
		GupyCollector: gupy,
	}
}
