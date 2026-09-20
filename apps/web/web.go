package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:embed static/* templates/* templates/layouts/* templates/pages/* templates/fragments/jobs/*
var embeddedFiles embed.FS

// JobItem representa a visão de uma vaga para renderização no template.
type JobItem struct {
	ID             uuid.UUID
	Title          string
	Company        string
	City           string
	State          string
	Source         string
	SourceURL      string
	WorkMode       string
	EmploymentType string
	SalaryMin      *int64
	SalaryMax      *int64
	PublishedAt    *time.Time
	Description    string
	Status         string
}

// CityItem representa a opção de filtro por cidade com contagem.
type CityItem struct {
	City      string
	State     string
	TotalJobs int64
}

// CompanyItem representa a opção de filtro por instituição com contagem.
type CompanyItem struct {
	Name      string
	TotalJobs int64
}

// FilterParams guarda os parâmetros de busca para manter o estado do formulário e paginação.
type FilterParams struct {
	Query   string
	City    string
	Company string
	Status  string
}

// PageData representa os dados injetados na tela inicial e em seus fragments.
type PageData struct {
	Title      string
	Items      []JobItem
	Page       int
	Size       int
	Total      int64
	TotalPages int
	Params     FilterParams
	DateFilter string
	Cities     []CityItem
	Companies  []CompanyItem
}

// ViewEngine encapsula a compilação e execução dos templates HTML.
type ViewEngine struct {
	indexTemplate     *template.Template
	fragmentTemplate  *template.Template
	rateLimitTemplate *template.Template
}

// TemplateFuncs retorna as funções auxiliares disponíveis nos templates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"timeAgo": func(t *time.Time) string {
			if t == nil || t.IsZero() {
				return "Recentemente"
			}
			diff := time.Since(*t)
			if diff < 0 {
				diff = 0
			}

			hours := int(diff.Hours())
			if hours < 1 {
				return "Publicado há pouco"
			}
			if hours < 24 {
				return "Publicado hoje"
			}
			days := hours / 24
			if days == 1 {
				return "Publicado ontem"
			}
			if days < 7 {
				return fmt.Sprintf("Publicado há %d dias", days)
			}
			weeks := days / 7
			if weeks == 1 {
				return "Publicado há 1 semana"
			}
			if weeks < 4 {
				return fmt.Sprintf("Publicado há %d semanas", weeks)
			}
			months := days / 30
			if months <= 1 {
				return "Publicado há 1 mês"
			}
			return fmt.Sprintf("Publicado há %d meses", months)
		},

		"formatSalary": func(min, max *int64) string {
			if min == nil && max == nil {
				return "A combinar"
			}
			if min != nil && max != nil {
				if *min == *max {
					return fmt.Sprintf("R$ %s", formatNumber(*min))
				}
				return fmt.Sprintf("R$ %s a R$ %s", formatNumber(*min), formatNumber(*max))
			}
			if min != nil {
				return fmt.Sprintf("A partir de R$ %s", formatNumber(*min))
			}
			return fmt.Sprintf("Até R$ %s", formatNumber(*max))
		},

		"formatWorkMode": func(wm string) string {
			switch strings.ToUpper(strings.TrimSpace(wm)) {
			case "REMOTE":
				return "Remoto"
			case "HYBRID":
				return "Híbrido"
			case "ON_SITE":
				return "Presencial"
			default:
				if wm == "" {
					return "Presencial"
				}
				return wm
			}
		},

		"formatEmploymentType": func(et string) string {
			switch strings.ToUpper(strings.TrimSpace(et)) {
			case "FULL_TIME":
				return "CLT"
			case "PART_TIME":
				return "Meio período"
			case "TEMPORARY":
				return "Temporário"
			case "INTERNSHIP":
				return "Estágio"
			default:
				if et == "" {
					return "CLT"
				}
				return et
			}
		},

		"formatSource": func(source string) string {
			switch strings.ToLower(strings.TrimSpace(source)) {
			case "santacasa":
				return "Santa Casa de Porto Alegre"
			case "moinhos":
				return "Hospital Moinhos de Vento"
			case "saolucas":
				return "Hospital São Lucas PUCRS"
			case "unimed":
				return "Unimed Porto Alegre"
			case "doctorclin":
				return "Doctor Clin"
			case "fleury":
				return "Grupo Fleury / Weinmann"
			case "hcpa":
				return "Hospital de Clínicas de Porto Alegre"
			case "divina":
				return "Hospital Divina Providência"
			case "maededeus":
				return "Hospital Mãe de Deus"
			case "vagascom":
				return "Vagas.com"
			case "gupy":
				return "Portal Gupy"
			default:
				if source == "" {
					return "Portal Oficial"
				}
				return source
			}
		},

		"extractSpecialty": func(title string) string {
			t := strings.ToLower(title)
			switch {
			case strings.Contains(t, "uti") || strings.Contains(t, "cti") || strings.Contains(t, "terapia intensiva"):
				return "UTI / CTI"
			case strings.Contains(t, "cirurg") || strings.Contains(t, "bloco") || strings.Contains(t, "cme"):
				return "Centro Cirúrgico"
			case strings.Contains(t, "emerg") || strings.Contains(t, "pronto at") || strings.Contains(t, "pa ") || strings.Contains(t, "urgenc"):
				return "Emergência / PA"
			case strings.Contains(t, "pediatr") || strings.Contains(t, "neonat") || strings.Contains(t, "materno") || strings.Contains(t, "parto"):
				return "Pediatria / Neo"
			case strings.Contains(t, "hemodial") || strings.Contains(t, "nefro"):
				return "Hemodiálise"
			case strings.Contains(t, "oncol"):
				return "Oncologia"
			case strings.Contains(t, "interna"):
				return "Internação"
			case strings.Contains(t, "psiquiatr") || strings.Contains(t, "saude mental") || strings.Contains(t, "saúde mental"):
				return "Saúde Mental"
			case strings.Contains(t, "sangue") || strings.Contains(t, "hemoterap"):
				return "Hemoterapia"
			default:
				return ""
			}
		},

		"add": func(a, b int) int {
			return a + b
		},

		"sub": func(a, b int) int {
			return a - b
		},

		"paginationRange": func(current, total int) []int {
			if total <= 1 {
				return nil
			}
			if total <= 7 {
				r := make([]int, total)
				for i := 0; i < total; i++ {
					r[i] = i + 1
				}
				return r
			}

			// -1 representa elipse (...)
			if current <= 4 {
				return []int{1, 2, 3, 4, 5, -1, total}
			}
			if current >= total-3 {
				return []int{1, -1, total - 4, total - 3, total - 2, total - 1, total}
			}
			return []int{1, -1, current - 1, current, current + 1, -1, total}
		},
	}
}

func formatNumber(n int64) string {
	in := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(in)+len(in)/3)
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}
	return string(out)
}

// NewViewEngine compila e prepara todos os templates a partir do sistema de arquivos embutido.
func NewViewEngine() (*ViewEngine, error) {
	funcs := TemplateFuncs()

	// 1. Template da Página Inicial Completa (Layout + Index + Fragments)
	indexTmpl, err := template.New("base.html").Funcs(funcs).ParseFS(
		embeddedFiles,
		"templates/layouts/base.html",
		"templates/pages/index.html",
		"templates/fragments/jobs/list.html",
		"templates/fragments/jobs/card.html",
		"templates/fragments/jobs/pagination.html",
		"templates/fragments/jobs/empty.html",
	)
	if err != nil {
		return nil, fmt.Errorf("compilar template index: %w", err)
	}

	// 2. Template dos Fragments de Jobs para HTMX (apenas list.html + dependências filhas)
	fragmentTmpl, err := template.New("list.html").Funcs(funcs).ParseFS(
		embeddedFiles,
		"templates/fragments/jobs/list.html",
		"templates/fragments/jobs/card.html",
		"templates/fragments/jobs/pagination.html",
		"templates/fragments/jobs/empty.html",
	)
	if err != nil {
		return nil, fmt.Errorf("compilar template fragments: %w", err)
	}

	// 3. Template de Rate Limit (HTTP 429)
	rateLimitTmpl, err := template.New("base.html").Funcs(funcs).ParseFS(
		embeddedFiles,
		"templates/layouts/base.html",
		"templates/pages/ratelimit.html",
	)
	if err != nil {
		return nil, fmt.Errorf("compilar template rate limit: %w", err)
	}

	return &ViewEngine{
		indexTemplate:     indexTmpl,
		fragmentTemplate:  fragmentTmpl,
		rateLimitTemplate: rateLimitTmpl,
	}, nil
}

// RenderIndex renderiza a página inicial completa com base no layout.
func (e *ViewEngine) RenderIndex(w io.Writer, data PageData) error {
	if data.Title == "" {
		data.Title = "Radar Enfermagem RS — Vagas de Técnico em Enfermagem em Porto Alegre e Região"
	}
	return e.indexTemplate.ExecuteTemplate(w, "base.html", data)
}

// RenderJobListFragment renderiza exclusivamente o fragmento HTML da lista de vagas para requisições HTMX.
func (e *ViewEngine) RenderJobListFragment(w io.Writer, data PageData) error {
	return e.fragmentTemplate.ExecuteTemplate(w, "list.html", data)
}

// RenderRateLimit renderiza a página de aviso de limite de requisições excedido (HTTP 429).
func (e *ViewEngine) RenderRateLimit(w io.Writer) error {
	return e.rateLimitTemplate.ExecuteTemplate(w, "base.html", nil)
}

// StaticFS retorna o subsistema de arquivos estáticos prontos para ser servidos via http.FileServer.
func StaticFS() http.FileSystem {
	sub, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic(fmt.Sprintf("falha ao montar filesystem estatico: %v", err))
	}
	return http.FS(sub)
}
