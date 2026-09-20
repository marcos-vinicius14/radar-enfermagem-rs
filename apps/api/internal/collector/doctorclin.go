package collector

import (
	"net/http"
	"time"
)

const (
	defaultDoctorClinURL = "https://dcgroup.gupy.io"
)

var _ Collector = (*DoctorClinCollector)(nil)

type DoctorClinCollector struct {
	*GupyCollector
}

func NewDoctorClinCollector(client *http.Client, timeout time.Duration) *DoctorClinCollector {
	return NewDoctorClinCollectorWithURL(defaultDoctorClinURL, client, timeout)
}

func NewDoctorClinCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *DoctorClinCollector {
	gupy := NewGupyCollector(GupyConfig{
		Name:          "doctorclin",
		CompanyName:   "Doctor Clin",
		BaseURL:       baseURL,
		PublicBaseURL: defaultDoctorClinURL,
		DefaultCity:   "Novo Hamburgo",
		DefaultState:  "RS",
	}, client, timeout)

	return &DoctorClinCollector{
		GupyCollector: gupy,
	}
}
