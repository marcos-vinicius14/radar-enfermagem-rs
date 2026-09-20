package collector

import (
	"net/http"
	"time"
)

const (
	defaultMaeDeDeusCompanyID = "5ddfb2e5-3a03-4dd9-8d35-ae8d5538185b"
	defaultMaeDeDeusPortalURL = "https://somosaesc.portaldetalentos.senior.com.br"
)

var _ Collector = (*MaeDeDeusCollector)(nil)

type MaeDeDeusCollector struct {
	*SeniorCollector
}

func NewMaeDeDeusCollector(client *http.Client, timeout time.Duration) *MaeDeDeusCollector {
	return NewMaeDeDeusCollectorWithURL("", client, timeout)
}

func NewMaeDeDeusCollectorWithURL(endpointURL string, client *http.Client, timeout time.Duration) *MaeDeDeusCollector {
	senior := NewSeniorCollector(SeniorConfig{
		Name:          "maededeus",
		CompanyName:   "Hospital Mãe de Deus",
		CompanyID:     defaultMaeDeDeusCompanyID,
		EndpointURL:   endpointURL,
		PortalBaseURL: defaultMaeDeDeusPortalURL,
		DefaultCity:   "Porto Alegre",
		DefaultState:  "RS",
	}, client, timeout)

	return &MaeDeDeusCollector{
		SeniorCollector: senior,
	}
}
