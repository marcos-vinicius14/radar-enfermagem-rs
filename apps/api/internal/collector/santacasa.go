package collector

import (
	"context"
	"encoding/json"
	"errors"
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
	defaultSantaCasaURL = "https://santacasa.gupy.io"
	defaultTimeout      = 10 * time.Second
	defaultUserAgent    = "RadarEnfermagemRS/1.0 (+https://github.com/marcos-vinicius14/radar-enfermagem-rs)"
)

var nextDataRegex = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

var _ Collector = (*SantaCasaCollector)(nil)

type SantaCasaCollector struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

func NewSantaCasaCollector(client *http.Client, timeout time.Duration) *SantaCasaCollector {
	return NewSantaCasaCollectorWithURL(defaultSantaCasaURL, client, timeout)
}

func NewSantaCasaCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *SantaCasaCollector {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &SantaCasaCollector{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		timeout: timeout,
	}
}

func (c *SantaCasaCollector) Name() string {
	return "santacasa"
}

func (c *SantaCasaCollector) Collect(ctx context.Context, query SearchQuery) ([]RawJob, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para a Santa Casa: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conectar ao portal da Santa Casa (%s): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal da Santa Casa retornou código HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ler resposta do portal da Santa Casa: %w", err)
	}

	return c.parseHTML(bodyBytes, query)
}

func (c *SantaCasaCollector) parseHTML(body []byte, query SearchQuery) ([]RawJob, error) {
	matches := nextDataRegex.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, errors.New("não foi possível localizar os dados de vagas no portal da Santa Casa (__NEXT_DATA__ ausente)")
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
		return nil, fmt.Errorf("deserializar dados estruturados de vagas da Santa Casa: %w", err)
	}

	companyName := "Santa Casa de Porto Alegre"
	if payload.Props.PageProps.CareerPage.Name != "" {
		companyName = payload.Props.PageProps.CareerPage.Name
	}

	normalizedQuery := job.NormalizeText(query.Query)
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
			city = "Porto Alegre"
		}

		state := strings.TrimSpace(item.Workplace.Address.StateShortName)
		if state == "" {
			state = strings.TrimSpace(item.Workplace.Address.State)
		}
		if state == "" {
			state = "RS"
		}

		externalID := strconv.FormatInt(item.ID, 10)
		sourceURL := fmt.Sprintf("https://santacasa.gupy.io/jobs/%s", externalID)

		results = append(results, RawJob{
			ExternalID:     externalID,
			Title:          item.Title,
			Company:        companyName,
			Description:    item.Department,
			City:           city,
			State:          state,
			Source:         c.Name(),
			SourceURL:      sourceURL,
			WorkMode:       item.Workplace.WorkplaceType,
			EmploymentType: item.Type,
			RawData: map[string]any{
				"department": item.Department,
				"gupy_id":    item.ID,
			},
		})
	}

	return results, nil
}
