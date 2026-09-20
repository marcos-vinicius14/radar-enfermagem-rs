package collector_test

import (
	"testing"
	"time"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
)

func TestDefaultRegistry(t *testing.T) {
	reg := collector.DefaultRegistry(5 * time.Second)

	expectedCollectors := []string{
		"divina",
		"doctorclin",
		"fleury",
		"hcpa",
		"maededeus",
		"moinhos",
		"santacasa",
		"saolucas",
		"unimed",
	}

	names := reg.Names()
	if len(names) != len(expectedCollectors) {
		t.Fatalf("esperava %d coletores registrados, obteve %d: %v", len(expectedCollectors), len(names), names)
	}

	for i, expected := range expectedCollectors {
		if names[i] != expected {
			t.Errorf("coletor na posição %d = %q, esperado %q", i, names[i], expected)
		}
		c, ok := reg.Get(expected)
		if !ok || c == nil {
			t.Errorf("não encontrou coletor %q no registro", expected)
		} else if c.Name() != expected {
			t.Errorf("coletor %q tem nome incorreto %q", expected, c.Name())
		}
	}

	all := reg.All()
	if len(all) != len(expectedCollectors) {
		t.Fatalf("All() retornou %d coletores, esperado %d", len(all), len(expectedCollectors))
	}
}
