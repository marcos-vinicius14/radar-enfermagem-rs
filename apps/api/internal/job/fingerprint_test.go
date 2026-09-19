package job_test

import (
	"testing"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/job"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converte caixa alta para caixa baixa",
			input:    "HOSPITAL MOINHOS DE VENTO",
			expected: "hospital moinhos de vento",
		},
		{
			name:     "remove acentos graficos simples e compostos",
			input:    "Técnico de Enfermagem - Atenção Básica e Saúde",
			expected: "tecnico de enfermagem - atencao basica e saude",
		},
		{
			name:     "remove acentos em nomes de instituicoes",
			input:    "Hospital São Lucas da PUCRS & Mãe de Deus",
			expected: "hospital sao lucas da pucrs & mae de deus",
		},
		{
			name:     "colapsa multiplos espacos, tabulacoes e quebras de linha",
			input:    "  Hospital   \t  Santa   \n  Casa   ",
			expected: "hospital santa casa",
		},
		{
			name:     "string vazia",
			input:    "",
			expected: "",
		},
		{
			name:     "string apenas com espacos em branco",
			input:    "   \t  \n  ",
			expected: "",
		},
		{
			name:     "caracteres com tremas e circunflexos",
			input:    "Ambulatório Müller e Pôr do Sol",
			expected: "ambulatorio muller e por do sol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := job.NormalizeText(tt.input)
			if got != tt.expected {
				t.Fatalf("NormalizeText(%q) = %q, esperado %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFingerprint(t *testing.T) {
	t.Run("gera hash sha256 de 64 caracteres hexadecimais", func(t *testing.T) {
		fp := job.Fingerprint("Hospital Moinhos de Vento", "Técnico de Enfermagem - UTI", "Porto Alegre")
		if len(fp) != 64 {
			t.Fatalf("Fingerprint length = %d, esperado 64", len(fp))
		}
	})

	t.Run("e deterministico e invariante a caixa, acentos e espacos", func(t *testing.T) {
		fp1 := job.Fingerprint(
			"Hospital São Lucas",
			"Técnico de Enfermagem",
			"Porto Alegre",
		)

		fp2 := job.Fingerprint(
			"  HOSPITAL SAO LUCAS  ",
			"técnico   de   enfermagem",
			"porto alegre",
		)

		if fp1 != fp2 {
			t.Fatalf("Fingerprint mismatch para entradas semanticamente equivalentes:\nfp1: %s\nfp2: %s", fp1, fp2)
		}
	})

	t.Run("diferentes combinacoes geram diferentes fingerprints", func(t *testing.T) {
		fp1 := job.Fingerprint("Hospital Moinhos de Vento", "Técnico de Enfermagem", "Porto Alegre")
		fp2 := job.Fingerprint("Hospital Moinhos de Vento", "Enfermeiro", "Porto Alegre")
		fp3 := job.Fingerprint("Santa Casa", "Técnico de Enfermagem", "Porto Alegre")
		fp4 := job.Fingerprint("Hospital Moinhos de Vento", "Técnico de Enfermagem", "Canoas")

		hashes := map[string]string{
			"fp1": fp1,
			"fp2": fp2,
			"fp3": fp3,
			"fp4": fp4,
		}

		visited := make(map[string]string)
		for name, h := range hashes {
			if existing, ok := visited[h]; ok {
				t.Fatalf("colisão inesperada entre %s e %s: %s", name, existing, h)
			}
			visited[h] = name
		}
	})
}
