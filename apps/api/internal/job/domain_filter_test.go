package job_test

import (
	"testing"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

func TestIsNursingJob(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		want        bool
	}{
		// Casos Positivos Válidos de Enfermagem
		{
			name:  "Tecnico de Enfermagem direto",
			title: "Técnico de Enfermagem",
			want:  true,
		},
		{
			name:  "Tecnica em Enfermagem - UTI Adulto",
			title: "Técnica em Enfermagem - UTI Adulto",
			want:  true,
		},
		{
			name:  "Auxiliar de Enfermagem",
			title: "Auxiliar de Enfermagem - Pronto Atendimento",
			want:  true,
		},
		{
			name:  "Enfermeiro(a) - Bloco Cirurgico",
			title: "Enfermeiro(a) - Bloco Cirúrgico",
			want:  true,
		},
		{
			name:  "Enfermeira Obstetra",
			title: "Enfermeira Obstetra",
			want:  true,
		},
		{
			name:  "Instrumentacao Cirurgica",
			title: "Instrumentador Cirúrgico",
			want:  true,
		},
		{
			name:  "Tecnico em Instrumentacao Cirurgica",
			title: "Técnico em Instrumentação Cirúrgica",
			want:  true,
		},
		{
			name:  "Tecnico de Enfermagem - CME",
			title: "Técnico de Enfermagem - Centro de Material e Esterilização (CME)",
			want:  true,
		},
		{
			name:  "Flebotomista Hospitalar",
			title: "Flebotomista - Coleta Laboratorial",
			want:  true,
		},
		{
			name:  "Tecnico de Enfermagem - Hemodialise",
			title: "Técnico de Enfermagem - Hemodiálise",
			want:  true,
		},
		{
			name:  "Tecnico de Enfermagem - Quimioterapia Oncologia",
			title: "Técnico de Enfermagem - Quimioterapia / Oncologia",
			want:  true,
		},
		{
			name:  "Enfermeiro Triagem e Acolhimento",
			title: "Enfermeiro - Triagem e Acolhimento",
			want:  true,
		},
		{
			name:  "Tec. Enfermagem abreviado",
			title: "Tec. Enfermagem - Internação Clínica",
			want:  true,
		},
		{
			name:  "Vacinador / Imunizacao",
			title: "Técnico de Enfermagem - Imunização e Vacinas",
			want:  true,
		},
		{
			name:        "Titulo generico com descricao de enfermagem",
			title:       "Profissional de Apoio Assistencial",
			description: "Requisitos: Curso Técnico em Enfermagem completo e registro ativo no COREN-RS.",
			want:        true,
		},

		// Casos Negativos de Exclusão Estrita (Bugs Reais de Produção)
		{
			name:  "Arquiteto PUCRS (Bug Issue #15)",
			title: "BANCO DE TALENTOS - Arquiteto/a - Projetos Executivos e Infraestrutura Pública - Contrato por prazo determinado - PUCRS Consulting",
			want:  false,
		},
		{
			name:  "Jovem Aprendiz Colegio Sao Carlos (Bug Issue #15)",
			title: "Jovem Aprendiz - Colégio São Carlos (Santa Vitória do Palmar)",
			want:  false,
		},
		{
			name:  "Auxiliar de Hidraulica",
			title: "Auxiliar de Hidráulica",
			want:  false,
		},
		{
			name:  "Atendente de Museu",
			title: "Atendente de Museu",
			want:  false,
		},
		{
			name:  "Advogado",
			title: "Advogado Pleno - Contencioso Cível",
			want:  false,
		},
		{
			name:  "Engenheiro Civil",
			title: "Engenheiro Civil de Obras",
			want:  false,
		},
		{
			name:  "Cozinheiro Hospitalar",
			title: "Cozinheiro Hospitalar",
			want:  false,
		},
		{
			name:  "Auxiliar de Cozinha",
			title: "Auxiliar de Cozinha",
			want:  false,
		},
		{
			name:  "Analista de TI",
			title: "Analista de Sistemas / TI",
			want:  false,
		},
		{
			name:  "Desenvolvedor Go",
			title: "Desenvolvedor Backend Golang",
			want:  false,
		},
		{
			name:  "Professor",
			title: "Professor de Ensino Médio - Biologia",
			want:  false,
		},
		{
			name:  "Secretaria Escolar",
			title: "Secretária Escolar - Colégio",
			want:  false,
		},
		{
			name:  "Porteiro",
			title: "Porteiro / Vigia Noturno",
			want:  false,
		},
		{
			name:  "Telefonista",
			title: "Telefonista / Central de Atendimento",
			want:  false,
		},
		{
			name:  "Medico (outra categoria profissional)",
			title: "Médico Clínico Geral - Plantonista",
			want:  false,
		},
		{
			name:  "Fisioterapeuta (outra categoria profissional)",
			title: "Fisioterapeuta Respiratório - UTI",
			want:  false,
		},
		{
			name:  "Psicologo (outra categoria profissional)",
			title: "Psicólogo Hospitalar",
			want:  false,
		},
		{
			name:  "Vazio",
			title: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := job.IsNursingJob(tt.title, tt.description)
			if got != tt.want {
				t.Errorf("IsNursingJob(%q, %q) = %v, want %v", tt.title, tt.description, got, tt.want)
			}
		})
	}
}

func TestIsTargetLocation(t *testing.T) {
	tests := []struct {
		name  string
		city  string
		state string
		want  bool
	}{
		// Cidades válidas da Região Metropolitana de Porto Alegre / RS
		{"Porto Alegre RS", "Porto Alegre", "RS", true},
		{"Porto Alegre Rio Grande do Sul", "Porto Alegre", "Rio Grande do Sul", true},
		{"Canoas RS", "Canoas", "RS", true},
		{"Novo Hamburgo RS", "Novo Hamburgo", "RS", true},
		{"Sao Leopoldo RS", "São Leopoldo", "RS", true},
		{"Gravatai RS", "Gravataí", "RS", true},
		{"Viamao RS", "Viamão", "RS", true},
		{"Alvorada RS", "Alvorada", "RS", true},
		{"Cachoeirinha RS", "Cachoeirinha", "RS", true},
		{"Esteio RS", "Esteio", "RS", true},
		{"Sapucaia do Sul RS", "Sapucaia do Sul", "RS", true},
		{"Guaiba RS", "Guaíba", "RS", true},
		{"Eldorado do Sul RS", "Eldorado do Sul", "RS", true},
		{"Campo Bom RS", "Campo Bom", "RS", true},
		{"Sapiranga RS", "Sapiranga", "RS", true},
		{"Dois Irmaos RS", "Dois Irmãos", "RS", true},

		// Cidades do RS fora da Região Metropolitana (devem ser rejeitadas)
		{"Santa Vitoria do Palmar RS (Bug AESC)", "Santa Vitória do Palmar", "RS", false},
		{"Passo Fundo RS", "Passo Fundo", "RS", false},
		{"Pelotas RS", "Pelotas", "RS", false},
		{"Caxias do Sul RS", "Caxias do Sul", "RS", false},
		{"Santa Maria RS", "Santa Maria", "RS", false},

		// Outros Estados
		{"Sao Paulo SP", "São Paulo", "SP", false},
		{"Curitiba PR", "Curitiba", "PR", false},
		{"Porto Alegre SC (Estado errado)", "Porto Alegre", "SC", false},

		// Vazios
		{"Vazio", "", "", false},
		{"Apenas Estado", "", "RS", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := job.IsTargetLocation(tt.city, tt.state)
			if got != tt.want {
				t.Errorf("IsTargetLocation(%q, %q) = %v, want %v", tt.city, tt.state, got, tt.want)
			}
		})
	}
}
