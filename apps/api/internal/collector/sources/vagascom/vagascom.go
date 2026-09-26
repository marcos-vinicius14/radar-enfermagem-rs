package vagascom

import (
	"context"
	"fmt"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

const (
	defaultVagasComTimeout = 10 * time.Second
)

var (
	vagasCardRegex  = regexp.MustCompile(`(?s)<div class="[^"]*vg-card-oportunidades[^"]*">(.*?)(?:</footer>\s*</a>\s*</div>)`)
	vagasHrefRegex  = regexp.MustCompile(`href="([^"]*/oportunidade/[^"]*?/(\d+))"`)
	vagasTitleRegex = regexp.MustCompile(`(?s)<p class="box__title"[^>]*>\s*([^\n<]+)`)
	vagasCargoRegex = regexp.MustCompile(`(?s)<p class="icons-row__description vg-cargo">\s*([^\n<]+)`)
	vagasLocRegex   = regexp.MustCompile(`(?s)<p class="icons-row__description vg-localizacao">\s*([^\n<]+)`)
)

var _ collector.Collector = (*VagasComCollector)(nil)

type VagasComConfig struct {
	Name          string
	CompanyName   string
	BaseURL       string
	PublicBaseURL string
	DefaultCity   string
	DefaultState  string
}

type VagasComCollector struct {
	cfg     VagasComConfig
	client  *http.Client
	timeout time.Duration
}

func NewVagasComCollector(cfg VagasComConfig, client *http.Client, timeout time.Duration) *VagasComCollector {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}
	if timeout <= 0 {
		timeout = defaultVagasComTimeout
	}
	if cfg.DefaultCity == "" {
		cfg.DefaultCity = "Porto Alegre"
	}
	if cfg.DefaultState == "" {
		cfg.DefaultState = "RS"
	}
	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = "https://trabalheconosco.vagas.com.br"
	}
	return &VagasComCollector{
		cfg:     cfg,
		client:  client,
		timeout: timeout,
	}
}

func (c *VagasComCollector) Name() string {
	return c.cfg.Name
}

func (c *VagasComCollector) BaseURL() string {
	return c.cfg.BaseURL
}

func (c *VagasComCollector) Collect(ctx context.Context, query collector.SearchQuery) ([]collector.RawJob, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.cfg.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para o portal %s: %w", c.cfg.Name, err)
	}
	req.Header.Set("User-Agent", collector.DefaultUserAgent)
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

func (c *VagasComCollector) parseHTML(body []byte, query collector.SearchQuery) ([]collector.RawJob, error) {
	cards := vagasCardRegex.FindAllSubmatch(body, -1)
	if len(cards) == 0 {
		// Verifica se a página retornou um layout completamente sem vagas ou modificado
		if !strings.Contains(string(body), "oportunidade") && !strings.Contains(string(body), "vagas") {
			return nil, fmt.Errorf("não foi possível localizar as vagas no portal %s (layout alterado)", c.cfg.Name)
		}
		return []collector.RawJob{}, nil
	}

	normalizedQuery := job.NormalizeText(query.Query)
	normalizedCity := job.NormalizeText(query.City)
	results := make([]collector.RawJob, 0, len(cards))

	for _, card := range cards {
		cardHTML := card[1]

		hrefMatch := vagasHrefRegex.FindSubmatch(cardHTML)
		if len(hrefMatch) < 3 {
			continue
		}
		relativeURL := string(hrefMatch[1])
		externalID := string(hrefMatch[2])

		titleMatch := vagasTitleRegex.FindSubmatch(cardHTML)
		if len(titleMatch) < 2 {
			continue
		}
		title := strings.TrimSpace(string(titleMatch[1]))

		var cargo string
		cargoMatch := vagasCargoRegex.FindSubmatch(cardHTML)
		if len(cargoMatch) >= 2 {
			cargo = strings.TrimSpace(string(cargoMatch[1]))
		}

		city := c.cfg.DefaultCity
		state := c.cfg.DefaultState
		locMatch := vagasLocRegex.FindSubmatch(cardHTML)
		if len(locMatch) >= 2 {
			locText := strings.TrimSpace(string(locMatch[1]))
			parts := strings.Split(locText, "/")
			if len(parts) >= 1 && strings.TrimSpace(parts[0]) != "" {
				city = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
				state = strings.TrimSpace(parts[1])
			}
		}

		if normalizedQuery != "" {
			normalizedTitle := job.NormalizeText(title)
			normalizedCargo := job.NormalizeText(cargo)
			if !strings.Contains(normalizedTitle, normalizedQuery) && !strings.Contains(normalizedCargo, normalizedQuery) {
				continue
			}
		}

		if normalizedCity != "" && !strings.Contains(job.NormalizeText(city), normalizedCity) {
			continue
		}

		sourceURL := fmt.Sprintf("%s%s", strings.TrimRight(c.cfg.PublicBaseURL, "/"), relativeURL)

		results = append(results, collector.RawJob{
			ExternalID:     externalID,
			Title:          title,
			Company:        c.cfg.CompanyName,
			Description:    cargo,
			City:           city,
			State:          state,
			Source:         c.cfg.Name,
			SourceURL:      sourceURL,
			EmploymentType: cargo,
			RawData: map[string]any{
				"relative_url": relativeURL,
			},
		})

		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}
