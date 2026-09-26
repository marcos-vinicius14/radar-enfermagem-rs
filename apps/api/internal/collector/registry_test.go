package collector_test

import (
	"testing"

	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/collector"
)

func TestRegistry(t *testing.T) {
	reg := collector.NewRegistry()

	c1 := &mockCollector{name: "beta"}
	c2 := &mockCollector{name: "alpha"}

	reg.Register(c1)
	reg.Register(c2)

	// Names deve vir ordenado
	names := reg.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Fatalf("Names() inesperado: %v", names)
	}

	// Get
	if c, ok := reg.Get("alpha"); !ok || c != c2 {
		t.Errorf("Get('alpha') falhou: c=%v, ok=%v", c, ok)
	}
	if _, ok := reg.Get("inexistente"); ok {
		t.Errorf("Get('inexistente') deveria retornar false")
	}

	// All deve vir ordenado por nome
	all := reg.All()
	if len(all) != 2 || all[0].Name() != "alpha" || all[1].Name() != "beta" {
		t.Fatalf("All() inesperado: %v", all)
	}
}
