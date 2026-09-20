package collector

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

const (
	defaultHCPAURL     = "https://www.hcpa.edu.br/venha-para-o-hcpa/processo-seletivo-publico"
	defaultHCPATimeout = 10 * time.Second
)

var (
	hcpaLinkRegex = regexp.MustCompile(`(?s)<a[^>]+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
)

var _ Collector = (*HCPACollector)(nil)

// HCPACollector implementa a extração de editais e seleções públicas do Hospital de Clínicas de Porto Alegre.
type HCPACollector struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

func NewHCPACollector(client *http.Client, timeout time.Duration) *HCPACollector {
	return NewHCPACollectorWithURL(defaultHCPAURL, client, timeout)
}

func NewHCPACollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *HCPACollector {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}
	if timeout <= 0 {
		timeout = defaultHCPATimeout
	}
	return &HCPACollector{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		timeout: timeout,
	}
}

func (c *HCPACollector) Name() string {
	return "hcpa"
}

func (c *HCPACollector) BaseURL() string {
	return c.baseURL
}

func (c *HCPACollector) Collect(ctx context.Context, query SearchQuery) ([]RawJob, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para o portal do HCPA: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conectar ao portal do HCPA (%s): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal do HCPA retornou código HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ler resposta do portal do HCPA: %w", err)
	}

	return c.parseHTML(bodyBytes, query)
}

func (c *HCPACollector) parseHTML(body []byte, query SearchQuery) ([]RawJob, error) {
	matches := hcpaLinkRegex.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("não foi possível localizar os editais no portal do HCPA (layout alterado)")
	}

	normalizedQuery := job.NormalizeText(query.Query)
	normalizedCity := job.NormalizeText(query.City)
	results := make([]RawJob, 0)
	seen := make(map[string]bool)

	for _, m := range matches {
		link := strings.TrimSpace(string(m[1]))
		text := strings.TrimSpace(cleanHTMLTags(string(m[2])))

		lowerText := strings.ToLower(text)
		lowerLink := strings.ToLower(link)

		// Filtra apenas links relacionados a editais, seleções públicas ou concursos do HCPA
		isEdital := strings.Contains(lowerText, "edital") || strings.Contains(lowerText, "pss") || strings.Contains(lowerText, "processo seletivo")
		isRelevantLink := strings.Contains(lowerLink, "faurgs") || strings.Contains(lowerLink, "concursos") || strings.Contains(lowerLink, "processo-seletivo")

		if (!isEdital && !isRelevantLink) || text == "" || len(text) < 4 {
			continue
		}

		if strings.Contains(lowerText, "encerrados") || strings.Contains(lowerText, "fale conosco") {
			continue
		}

		if normalizedQuery != "" {
			normalizedTitle := job.NormalizeText(text)
			if !strings.Contains(normalizedTitle, normalizedQuery) {
				continue
			}
		}

		city := "Porto Alegre"
		state := "RS"
		if normalizedCity != "" && !strings.Contains(job.NormalizeText(city), normalizedCity) {
			continue
		}

		// Resolve link relativo
		fullURL := link
		if strings.HasPrefix(link, "/") {
			fullURL = fmt.Sprintf("https://www.hcpa.edu.br%s", link)
		}

		// Garante unicidade por link + título
		key := fmt.Sprintf("%s|%s", link, text)
		if seen[key] {
			continue
		}
		seen[key] = true

		hash := sha256.Sum256([]byte(key))
		externalID := fmt.Sprintf("hcpa-%x", hash)[:16]

		results = append(results, RawJob{
			ExternalID:     externalID,
			Title:          text,
			Company:        "Hospital de Clínicas de Porto Alegre",
			Description:    "Processo Seletivo Público / Edital FAURGS - HCPA",
			City:           city,
			State:          state,
			Source:         c.Name(),
			SourceURL:      fullURL,
			WorkMode:       "presencial",
			EmploymentType: "concurso_clt",
			RawData: map[string]any{
				"raw_title": text,
				"raw_link":  link,
			},
		})

		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	if len(results) == 0 && len(matches) > 0 {
		// Se havia tags <a> mas nenhuma foi qualificada como processo seletivo válido
		if !strings.Contains(string(body), "Processo Seletivo") && !strings.Contains(string(body), "Edital") {
			return nil, fmt.Errorf("não foi possível localizar os dados de editais no portal do HCPA")
		}
	}

	return results, nil
}
