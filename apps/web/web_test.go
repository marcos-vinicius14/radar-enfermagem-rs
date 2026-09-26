package web_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/web"
)

func TestNewViewEngine_CompilationAndIntegrity(t *testing.T) {
	engine, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("esperava compilação bem-sucedida dos templates, obteve erro: %v", err)
	}
	if engine == nil {
		t.Fatal("engine compilada retornou nil")
	}
}

func TestViewEngine_RenderIndex(t *testing.T) {
	engine, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar engine: %v", err)
	}

	pubTime := time.Now().Add(-2 * time.Hour)
	salMin := int64(3000)
	salMax := int64(4500)

	sampleJob := web.JobItem{
		ID:             uuid.New(),
		Title:          "Técnico de Enfermagem - CTI Adulto",
		Company:        "Hospital Moinhos de Vento",
		City:           "Porto Alegre",
		State:          "RS",
		Source:         "moinhos",
		SourceURL:      "https://trabalheconosco.vagas.com.br/moinhos/vaga/123",
		WorkMode:       "ON_SITE",
		EmploymentType: "FULL_TIME",
		SalaryMin:      &salMin,
		SalaryMax:      &salMax,
		PublishedAt:    &pubTime,
		Description:    "Atuação no CTI adulto prestando assistência direta ao paciente crítico.",
		Status:         "ACTIVE",
	}

	data := web.PageData{
		Title:      "Radar Enfermagem RS",
		Items:      []web.JobItem{sampleJob},
		Page:       1,
		Size:       20,
		Total:      1,
		TotalPages: 1,
		Cities: []web.CityItem{
			{City: "Porto Alegre", State: "RS", TotalJobs: 1},
		},
		Companies: []web.CompanyItem{
			{Name: "Hospital Moinhos de Vento", TotalJobs: 1},
		},
	}

	var buf bytes.Buffer
	err = engine.RenderIndex(&buf, data)
	if err != nil {
		t.Fatalf("esperava renderização com sucesso do index, obteve: %v", err)
	}

	html := buf.String()

	// Verifica presença de tags e dados essenciais
	expectedSubstrings := []string{
		"<!DOCTYPE html>",
		"Radar Enfermagem RS",
		"Hospital Moinhos de Vento",
		"Técnico de Enfermagem - CTI Adulto",
		"Porto Alegre / RS",
		"UTI / CTI", // especialidade extraída
		"R$ 3.000 a R$ 4.500",
		"id=\"search-form\"",
		"hx-indicator=\"#search-indicator\"",
		"hx-sync=\"this:replace\"",
		"badge-beta",
		"Versão Beta:",
		"GNU AGPLv3",
		"Aviso Legal &amp; Isenção de Responsabilidade",
		"Aviso a recrutadores:",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(html, substr) {
			t.Errorf("HTML renderizado não contém substring esperada: %q", substr)
		}
	}
}

func TestViewEngine_RenderJobListFragment(t *testing.T) {
	engine, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar engine: %v", err)
	}

	pubTime := time.Now().Add(-48 * time.Hour)
	sampleJob := web.JobItem{
		ID:             uuid.New(),
		Title:          "Técnico de Enfermagem - Bloco Cirúrgico",
		Company:        "Santa Casa de Porto Alegre",
		City:           "Porto Alegre",
		State:          "RS",
		Source:         "santacasa",
		SourceURL:      "https://santacasa.org.br/vaga/456",
		WorkMode:       "ON_SITE",
		EmploymentType: "FULL_TIME",
		PublishedAt:    &pubTime,
		Status:         "ACTIVE",
	}

	data := web.PageData{
		Items:      []web.JobItem{sampleJob},
		Page:       1,
		Size:       10,
		Total:      1,
		TotalPages: 1,
	}

	var buf bytes.Buffer
	err = engine.RenderJobListFragment(&buf, data)
	if err != nil {
		t.Fatalf("esperava renderização com sucesso do fragmento, obteve: %v", err)
	}

	html := buf.String()

	// Garante que é um fragmento limpo (sem tags de documento global)
	if strings.Contains(html, "<!DOCTYPE html>") || strings.Contains(html, "<html") || strings.Contains(html, "<body") {
		t.Error("o fragmento HTMX não deveria conter tags de página completa (<!DOCTYPE, <html ou <body)")
	}

	if !strings.Contains(html, "id=\"jobs-container\"") {
		t.Error("fragmento deve conter o container alvo id=\"jobs-container\"")
	}

	if !strings.Contains(html, "Santa Casa de Porto Alegre") {
		t.Error("fragmento deve conter o nome da empresa")
	}
	if !strings.Contains(html, "Centro Cirúrgico") {
		t.Error("fragmento deve conter a especialidade extraída")
	}
}

func TestViewEngine_RenderJobListFragment_EmptyState(t *testing.T) {
	engine, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar engine: %v", err)
	}

	data := web.PageData{
		Items:      []web.JobItem{},
		Page:       1,
		Size:       20,
		Total:      0,
		TotalPages: 0,
	}

	var buf bytes.Buffer
	err = engine.RenderJobListFragment(&buf, data)
	if err != nil {
		t.Fatalf("erro ao renderizar fragmento vazio: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "Nenhuma vaga encontrada") {
		t.Errorf("esperava mensagem de estado vazio amigável, obteve: %s", html)
	}
	if !strings.Contains(html, "Limpar todos os filtros") {
		t.Errorf("esperava botão de limpar filtros no estado vazio, obteve: %s", html)
	}
}

func TestViewEngine_RenderRateLimit(t *testing.T) {
	engine, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar engine: %v", err)
	}

	var buf bytes.Buffer
	err = engine.RenderRateLimit(&buf)
	if err != nil {
		t.Fatalf("erro ao renderizar tela de rate limit: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "Calma aí!") {
		t.Error("tela de rate limit deve conter mensagem acolhedora 'Calma aí!'")
	}
	if !strings.Contains(html, "Tentar Novamente") {
		t.Error("tela de rate limit deve conter botão de tentar novamente")
	}
}

func TestTemplateFuncs_Helpers(t *testing.T) {
	funcs := web.TemplateFuncs()

	// 1. timeAgo
	timeAgoFn := funcs["timeAgo"].(func(*time.Time) string)
	now := time.Now()
	if timeAgoFn(nil) != "Recentemente" {
		t.Errorf("esperava 'Recentemente' para nil, obteve: %s", timeAgoFn(nil))
	}
	h1 := now.Add(-30 * time.Minute)
	if timeAgoFn(&h1) != "Publicado há pouco" {
		t.Errorf("esperava 'Publicado há pouco', obteve: %s", timeAgoFn(&h1))
	}
	h5 := now.Add(-5 * time.Hour)
	if timeAgoFn(&h5) != "Publicado hoje" {
		t.Errorf("esperava 'Publicado hoje', obteve: %s", timeAgoFn(&h5))
	}
	d1 := now.Add(-25 * time.Hour)
	if timeAgoFn(&d1) != "Publicado ontem" {
		t.Errorf("esperava 'Publicado ontem', obteve: %s", timeAgoFn(&d1))
	}

	// 2. formatSalary
	salaryFn := funcs["formatSalary"].(func(*int64, *int64) string)
	if salaryFn(nil, nil) != "A combinar" {
		t.Errorf("esperava 'A combinar', obteve: %s", salaryFn(nil, nil))
	}
	min := int64(2500)
	max := int64(3500)
	if salaryFn(&min, &max) != "R$ 2.500 a R$ 3.500" {
		t.Errorf("esperava 'R$ 2.500 a R$ 3.500', obteve: %s", salaryFn(&min, &max))
	}
	if salaryFn(&min, nil) != "A partir de R$ 2.500" {
		t.Errorf("esperava 'A partir de R$ 2.500', obteve: %s", salaryFn(&min, nil))
	}

	// 3. extractSpecialty
	specFn := funcs["extractSpecialty"].(func(string) string)
	tests := []struct {
		title    string
		expected string
	}{
		{"Técnico de Enfermagem - CTI Adulto", "UTI / CTI"},
		{"Técnico de Enfermagem - Bloco Cirúrgico", "Centro Cirúrgico"},
		{"Técnico de Enfermagem - Pronto Atendimento", "Emergência / PA"},
		{"Técnico Enfermagem - Pediatria", "Pediatria / Neo"},
		{"Técnico de Enfermagem - Nefrologia", "Hemodiálise"},
		{"Técnico Geral de Enfermagem", ""},
	}
	for _, tc := range tests {
		got := specFn(tc.title)
		if got != tc.expected {
			t.Errorf("extractSpecialty(%q) = %q; esperava %q", tc.title, got, tc.expected)
		}
	}

	// 4. paginationRange
	pageRangeFn := funcs["paginationRange"].(func(int, int) []int)
	shortRange := pageRangeFn(1, 5)
	if len(shortRange) != 5 || shortRange[4] != 5 {
		t.Errorf("esperava range de 1 a 5, obteve: %v", shortRange)
	}

	longRange := pageRangeFn(1, 10)
	if len(longRange) != 7 || longRange[5] != -1 || longRange[6] != 10 {
		t.Errorf("esperava range com elipse para 10 páginas, obteve: %v", longRange)
	}
}

func TestStaticFS(t *testing.T) {
	fs := web.StaticFS()
	file, err := fs.Open("css/styles.css")
	if err != nil {
		t.Fatalf("esperava abrir css/styles.css com sucesso: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.Size() == 0 {
		t.Errorf("arquivo styles.css está vazio ou inacessível: %v", err)
	}
}

func TestViewEngine_RenderIndex_SpecialtyChipsActiveState(t *testing.T) {
	engine, err := web.NewViewEngine()
	if err != nil {
		t.Fatalf("erro ao criar engine: %v", err)
	}

	tests := []struct {
		name           string
		query          string
		expectedActive []string
	}{
		{
			name:           "ativa chip UTI quando query for UTI",
			query:          "UTI",
			expectedActive: []string{"UTI"},
		},
		{
			name:           "ativa múltiplos chips quando query for UTI, Cirurgico",
			query:          "UTI, Cirurgico",
			expectedActive: []string{"UTI", "Cirurgico"},
		},
		{
			name:           "ativa múltiplos chips quando query for Cirurgico, Pediatria, Hemodialise",
			query:          "Cirurgico, Pediatria, Hemodialise",
			expectedActive: []string{"Cirurgico", "Pediatria", "Hemodialise"},
		},
		{
			name:           "ativa chip Emergencia quando query for Emergencia",
			query:          "Emergencia",
			expectedActive: []string{"Emergencia"},
		},
		{
			name:           "nenhum chip ativo quando query for vazia",
			query:          "",
			expectedActive: nil,
		},
		{
			name:           "nenhum chip ativo quando query for outro termo",
			query:          "hospital",
			expectedActive: nil,
		},
	}

	allChips := []string{"UTI", "Cirurgico", "Emergencia", "Pediatria", "Hemodialise"}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := web.PageData{
				Title: "Radar Enfermagem RS",
				Params: web.FilterParams{
					Query: tc.query,
				},
			}

			var buf bytes.Buffer
			if err := engine.RenderIndex(&buf, data); err != nil {
				t.Fatalf("erro ao renderizar index: %v", err)
			}

			html := buf.String()

			expectedMap := map[string]bool{}
			for _, exp := range tc.expectedActive {
				expectedMap[exp] = true
			}

			for _, chip := range allChips {
				activePattern := fmt.Sprintf("class=\"chip-btn active\"\n                data-query=\"%s\"", chip)
				inactivePattern := fmt.Sprintf("class=\"chip-btn\"\n                data-query=\"%s\"", chip)

				if expectedMap[chip] {
					if !strings.Contains(html, activePattern) {
						t.Errorf("esperava chip %q com classe 'active', mas não foi encontrado no HTML", chip)
					}
				} else {
					if !strings.Contains(html, inactivePattern) {
						t.Errorf("esperava chip %q inativo com classe 'chip-btn', mas não foi encontrado no HTML", chip)
					}
				}
			}
		})
	}
}
