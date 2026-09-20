package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

const (
	defaultGupyTimeout = 10 * time.Second
	defaultUserAgent   = "RadarEnfermagemRS/1.0 (+https://github.com/marcos-vinicius14/radar-enfermagem-rs)"
)

var nextDataRegex = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

var _ Collector = (*GupyCollector)(nil)

type GupyConfig struct {
	Name          string
	CompanyName   string
	BaseURL       string
	PublicBaseURL string
	DefaultCity   string
	DefaultState  string
}

type GupyCollector struct {
	cfg     GupyConfig
	client  *http.Client
	timeout time.Duration
}

func NewGupyCollector(cfg GupyConfig, client *http.Client, timeout time.Duration) *GupyCollector {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}
	if timeout <= 0 {
		timeout = defaultGupyTimeout
	}
	if cfg.DefaultCity == "" {
		cfg.DefaultCity = "Porto Alegre"
	}
	if cfg.DefaultState == "" {
		cfg.DefaultState = "RS"
	}
	return &GupyCollector{
		cfg:     cfg,
		client:  client,
		timeout: timeout,
	}
}

func (c *GupyCollector) Name() string {
	return c.cfg.Name
}

func (c *GupyCollector) BaseURL() string {
	return c.cfg.BaseURL
}

func (c *GupyCollector) CompanyName() string {
	return c.cfg.CompanyName
}

func (c *GupyCollector) Collect(ctx context.Context, query SearchQuery) ([]RawJob, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.cfg.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para o portal %s: %w", c.cfg.Name, err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conectar ao portal %s (%s): %w", c.cfg.Name, c.cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal %s retornou código HTTP %d", c.cfg.Name, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ler resposta do portal %s: %w", c.cfg.Name, err)
	}

	return c.parseHTML(bodyBytes, query)
}

func (c *GupyCollector) parseHTML(body []byte, query SearchQuery) ([]RawJob, error) {
	matches := nextDataRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("não foi possível localizar os dados de vagas no portal %s (__NEXT_DATA__ ausente)", c.cfg.Name)
	}

	var payload struct {
		Props struct {
			PageProps struct {
				CareerPage struct {
					Name string `json:"name"`
				} `json:"careerPage"`
				Jobs []struct {
					ID         int64  `json:"id"`
					Title      string `json:"title"`
					Type       string `json:"type"`
					Department string `json:"department"`
					Workplace  struct {
						Address struct {
							City           string `json:"city"`
							State          string `json:"state"`
							StateShortName string `json:"stateShortName"`
						} `json:"address"`
						WorkplaceType string `json:"workplaceType"`
					} `json:"workplace"`
				} `json:"jobs"`
			} `json:"pageProps"`
		} `json:"props"`
	}

	if err := json.Unmarshal(matches[1], &payload); err != nil {
		return nil, fmt.Errorf("deserializar dados estruturados de vagas do portal %s: %w", c.cfg.Name, err)
	}

	companyName := c.cfg.CompanyName
	if payload.Props.PageProps.CareerPage.Name != "" {
		companyName = payload.Props.PageProps.CareerPage.Name
	}

	normalizedQuery := job.NormalizeText(query.Query)
	normalizedCity := job.NormalizeText(query.City)
	jobsList := payload.Props.PageProps.Jobs
	results := make([]RawJob, 0, len(jobsList))

	for _, item := range jobsList {
		if normalizedQuery != "" {
			normalizedTitle := job.NormalizeText(item.Title)
			normalizedDept := job.NormalizeText(item.Department)
			if !strings.Contains(normalizedTitle, normalizedQuery) && !strings.Contains(normalizedDept, normalizedQuery) {
				continue
			}
		}

		city := strings.TrimSpace(item.Workplace.Address.City)
		if city == "" {
			city = c.cfg.DefaultCity
		}

		if normalizedCity != "" && !strings.Contains(job.NormalizeText(city), normalizedCity) {
			continue
		}

		state := strings.TrimSpace(item.Workplace.Address.StateShortName)
		if state == "" {
			state = strings.TrimSpace(item.Workplace.Address.State)
		}
		if state == "" {
			state = c.cfg.DefaultState
		}

		externalID := strconv.FormatInt(item.ID, 10)
		publicBaseURL := c.cfg.PublicBaseURL
		if publicBaseURL == "" {
			publicBaseURL = c.cfg.BaseURL
		}
		sourceURL := fmt.Sprintf("%s/jobs/%s", strings.TrimRight(publicBaseURL, "/"), externalID)

		results = append(results, RawJob{
			ExternalID:     externalID,
			Title:          item.Title,
			Company:        companyName,
			Description:    item.Department,
			City:           city,
			State:          state,
			Source:         c.cfg.Name,
			SourceURL:      sourceURL,
			WorkMode:       item.Workplace.WorkplaceType,
			EmploymentType: item.Type,
			RawData: map[string]any{
				"department": item.Department,
				"gupy_id":    item.ID,
			},
		})

		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	if len(results) == 0 && len(jobsList) == 0 {
		return results, nil
	}

	return results, nil
}
