package senior

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

const (
	defaultSeniorVacanciesURL = "https://platform.senior.com.br/t/senior.com.br/bridge/1.0/anonymous/rest/hcm/careersmanagercandidate/queries/searchVacancies"
	defaultSeniorTimeout      = 10 * time.Second
)

var _ collector.Collector = (*SeniorCollector)(nil)

type SeniorConfig struct {
	Name          string
	CompanyName   string
	CompanyID     string
	EndpointURL   string
	PortalBaseURL string
	DefaultCity   string
	DefaultState  string
}

type SeniorCollector struct {
	cfg     SeniorConfig
	client  *http.Client
	timeout time.Duration
}

func NewSeniorCollector(cfg SeniorConfig, client *http.Client, timeout time.Duration) *SeniorCollector {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}
	if timeout <= 0 {
		timeout = defaultSeniorTimeout
	}
	if cfg.EndpointURL == "" {
		cfg.EndpointURL = defaultSeniorVacanciesURL
	}
	if cfg.DefaultCity == "" {
		cfg.DefaultCity = "Porto Alegre"
	}
	if cfg.DefaultState == "" {
		cfg.DefaultState = "RS"
	}
	return &SeniorCollector{
		cfg:     cfg,
		client:  client,
		timeout: timeout,
	}
}

func (c *SeniorCollector) Name() string {
	return c.cfg.Name
}

func (c *SeniorCollector) EndpointURL() string {
	return c.cfg.EndpointURL
}

func (c *SeniorCollector) Collect(ctx context.Context, query collector.SearchQuery) ([]collector.RawJob, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	requestBody := map[string]any{
		"page":   0,
		"size":   50,
		"filter": "",
		"match": map[string]any{
			"localizations": []any{},
			"companies": []map[string]string{
				{"id": c.cfg.CompanyID},
			},
		},
	}

	payloadBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("serializar requisição para o portal %s: %w", c.cfg.Name, err)
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.cfg.EndpointURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("criar requisição para o portal %s: %w", c.cfg.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", collector.DefaultUserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conectar à API do portal %s (%s): %w", c.cfg.Name, c.cfg.EndpointURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal %s retornou código HTTP %d", c.cfg.Name, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ler resposta do portal %s: %w", c.cfg.Name, err)
	}

	return c.parseJSON(bodyBytes, query)
}

func (c *SeniorCollector) parseJSON(body []byte, query collector.SearchQuery) ([]collector.RawJob, error) {
	var payload struct {
		TotalPages    int `json:"totalPages"`
		TotalElements int `json:"totalElements"`
		Contents      []struct {
			Vacancy struct {
				ID           string   `json:"id"`
				Title        string   `json:"title"`
				JobModel     []string `json:"jobModel"`
				Description  string   `json:"description"`
				Localization struct {
					City     string `json:"city"`
					Province string `json:"province"`
					Country  string `json:"country"`
				} `json:"localization"`
				Publication struct {
					StartDate string `json:"startDate"`
					EndDate   string `json:"endDate"`
				} `json:"publication"`
			} `json:"vacancy"`
			Company struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Tenant string `json:"tenant"`
			} `json:"company"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("deserializar dados estruturados de vagas do portal %s: %w", c.cfg.Name, err)
	}

	normalizedQuery := job.NormalizeText(query.Query)
	normalizedCity := job.NormalizeText(query.City)
	results := make([]collector.RawJob, 0, len(payload.Contents))

	for _, item := range payload.Contents {
		v := item.Vacancy
		if normalizedQuery != "" {
			normalizedTitle := job.NormalizeText(v.Title)
			normalizedDesc := job.NormalizeText(v.Description)
			if !strings.Contains(normalizedTitle, normalizedQuery) && !strings.Contains(normalizedDesc, normalizedQuery) {
				continue
			}
		}

		city := strings.TrimSpace(v.Localization.City)
		if city == "" {
			city = c.cfg.DefaultCity
		}

		if normalizedCity != "" && !strings.Contains(job.NormalizeText(city), normalizedCity) {
			continue
		}

		state := strings.TrimSpace(v.Localization.Province)
		if state == "" || state == "BR" {
			state = c.cfg.DefaultState
		}

		companyName := c.cfg.CompanyName
		if companyName == "" && item.Company.Name != "" {
			companyName = item.Company.Name
		}

		sourceURL := fmt.Sprintf("%s/vacancy/%s", strings.TrimRight(c.cfg.PortalBaseURL, "/"), v.ID)
		if c.cfg.PortalBaseURL == "" {
			sourceURL = fmt.Sprintf("https://%s.portaldetalentos.senior.com.br/vacancy/%s", item.Company.Tenant, v.ID)
		}

		var workMode string
		if len(v.JobModel) > 0 {
			switch v.JobModel[0] {
			case "IN_PERSON":
				workMode = "presencial"
			case "REMOTE":
				workMode = "remoto"
			case "HYBRID":
				workMode = "híbrido"
			default:
				workMode = v.JobModel[0]
			}
		}

		var publishedAt *time.Time
		if v.Publication.StartDate != "" {
			if t, err := time.Parse("2006-01-02", v.Publication.StartDate); err == nil {
				publishedAt = &t
			}
		}

		results = append(results, collector.RawJob{
			ExternalID:  v.ID,
			Title:       v.Title,
			Company:     companyName,
			Description: v.Description,
			City:        city,
			State:       state,
			Source:      c.cfg.Name,
			SourceURL:   sourceURL,
			WorkMode:    workMode,
			PublishedAt: publishedAt,
			RawData: map[string]any{
				"senior_vacancy_id": v.ID,
				"company_id":        item.Company.ID,
				"tenant":            item.Company.Tenant,
			},
		})

		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}
