package collector

import (
	"net/http"
	"time"
)

const (
	defaultSaoLucasURL = "https://hospitalsaolucas.gupy.io"
)

var _ Collector = (*SaoLucasCollector)(nil)

// SaoLucasCollector implementa a extração de vagas do Hospital São Lucas da PUCRS.
type SaoLucasCollector struct {
	*GupyCollector
}

func NewSaoLucasCollector(client *http.Client, timeout time.Duration) *SaoLucasCollector {
	return NewSaoLucasCollectorWithURL(defaultSaoLucasURL, client, timeout)
}

func NewSaoLucasCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *SaoLucasCollector {
	gupy := NewGupyCollector(GupyConfig{
		Name:          "saolucas",
		CompanyName:   "Hospital São Lucas da PUCRS",
		BaseURL:       baseURL,
		PublicBaseURL: defaultSaoLucasURL,
		DefaultCity:   "Porto Alegre",
		DefaultState:  "RS",
	}, client, timeout)

	return &SaoLucasCollector{
		GupyCollector: gupy,
	}
}
