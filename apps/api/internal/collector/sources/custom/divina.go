package custom

import (
	"context"
	"crypto/sha256"
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
	defaultDivinaURL     = "https://divinaprovidencia.org.br/trabalhe-conosco/vagas/"
	defaultDivinaTimeout = 10 * time.Second
)

var (
	divinaVagaRegex  = regexp.MustCompile(`(?s)<div class="vaga">(.*?)</div>\s*</div>`)
	divinaTitleRegex = regexp.MustCompile(`(?s)<h2>(.*?)</h2>`)
	divinaBlockRegex = regexp.MustCompile(`(?s)<h3>(.*?)</h3>\s*<p>(.*?)</p>`)
)

var _ collector.Collector = (*DivinaCollector)(nil)

type DivinaCollector struct {
	baseURL string
	client  *http.Client
	timeout time.Duration
}

func NewDivinaCollector(client *http.Client, timeout time.Duration) *DivinaCollector {
	return NewDivinaCollectorWithURL(defaultDivinaURL, client, timeout)
}

func NewDivinaCollectorWithURL(baseURL string, client *http.Client, timeout time.Duration) *DivinaCollector {
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}
	if timeout <= 0 {
		timeout = defaultDivinaTimeout
	}
	return &DivinaCollector{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		timeout: timeout,
	}
}

func (c *DivinaCollector) Name() string {
	return "divina"
}

func (c *DivinaCollector) BaseURL() string {
	return c.baseURL
}

func (c *DivinaCollector) Collect(ctx context.Context, query collector.SearchQuery) ([]collector.RawJob, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("criar requisição para o portal da Divina Providência: %w", err)
	}
	req.Header.Set("User-Agent", collector.DefaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conectar ao portal da Divina Providência (%s): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal da Divina Providência retornou código HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ler resposta do portal da Divina Providência: %w", err)
	}

	return c.parseHTML(bodyBytes, query)
}

func (c *DivinaCollector) parseHTML(body []byte, query collector.SearchQuery) ([]collector.RawJob, error) {
	matches := divinaVagaRegex.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		if !strings.Contains(string(body), "vaga") && !strings.Contains(string(body), "trabalhe") {
			return nil, fmt.Errorf("não foi possível localizar as vagas no portal da Divina Providência (layout alterado)")
		}
		return []collector.RawJob{}, nil
	}

	normalizedQuery := job.NormalizeText(query.Query)
	normalizedCity := job.NormalizeText(query.City)
	results := make([]collector.RawJob, 0, len(matches))

	for _, m := range matches {
		blockHTML := m[1]

		titleMatch := divinaTitleRegex.FindSubmatch(blockHTML)
		if len(titleMatch) < 2 {
			continue
		}
		title := cleanHTMLTags(string(titleMatch[1]))
		if title == "" {
			continue
		}

		blocks := divinaBlockRegex.FindAllSubmatch(blockHTML, -1)
		meta := make(map[string]string)
		for _, b := range blocks {
			key := strings.TrimSpace(cleanHTMLTags(string(b[1])))
			val := strings.TrimSpace(cleanHTMLTags(string(b[2])))
			meta[strings.ToLower(key)] = val
		}

		hospital := meta["hospital"]
		if hospital == "" {
			hospital = "Hospital Divina"
		}

		city := "Porto Alegre"
		state := "RS"
		if rawCity := meta["cidade"]; rawCity != "" {
			parts := strings.Split(rawCity, "-")
			if len(parts) >= 1 && strings.TrimSpace(parts[0]) != "" {
				city = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
				state = strings.TrimSpace(parts[1])
			}
		}

		if normalizedQuery != "" {
			normalizedTitle := job.NormalizeText(title)
			normalizedHosp := job.NormalizeText(hospital)
			if !strings.Contains(normalizedTitle, normalizedQuery) && !strings.Contains(normalizedHosp, normalizedQuery) {
				continue
			}
		}

		if normalizedCity != "" && !strings.Contains(job.NormalizeText(city), normalizedCity) {
			continue
		}

		hash := sha256.Sum256([]byte(job.NormalizeText(title) + "|" + job.NormalizeText(hospital)))
		externalID := fmt.Sprintf("%x", hash)[:16]

		schedule := meta["carga horária"]
		contact := meta["contato"]

		results = append(results, collector.RawJob{
			ExternalID:  externalID,
			Title:       title,
			Company:     "Rede de Saúde Divina Providência",
			Description: fmt.Sprintf("Unidade: %s | Horário: %s", hospital, schedule),
			City:        city,
			State:       state,
			Source:      c.Name(),
			SourceURL:   defaultDivinaURL,
			WorkMode:    "presencial",
			RawData: map[string]any{
				"hospital": hospital,
				"schedule": schedule,
				"contact":  contact,
			},
		})

		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}

	return results, nil
}

func cleanHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(strings.ReplaceAll(s, "&nbsp;", " "))
}
