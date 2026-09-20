package collector

import (
	"sort"
	"sync"
	"time"
)

// Registry gerencia a coleção de coletores de vagas registrados na aplicação.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register adiciona ou substitui um coletor no registro.
func (r *Registry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[c.Name()] = c
}

func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

func (r *Registry) All() []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	sort.Strings(names)

	list := make([]Collector, 0, len(names))
	for _, name := range names {
		list = append(list, r.collectors[name])
	}
	return list
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultRegistry retorna um registro inicializado com todos os 9 portais hospitalares monitorados.
func DefaultRegistry(timeout time.Duration) *Registry {
	r := NewRegistry()
	r.Register(NewSantaCasaCollector(nil, timeout))
	r.Register(NewMoinhosCollector(nil, timeout))
	r.Register(NewSaoLucasCollector(nil, timeout))
	r.Register(NewUnimedCollector(nil, timeout))
	r.Register(NewDoctorClinCollector(nil, timeout))
	r.Register(NewFleuryCollector(nil, timeout))
	r.Register(NewHCPACollector(nil, timeout))
	r.Register(NewDivinaCollector(nil, timeout))
	r.Register(NewMaeDeDeusCollector(nil, timeout))
	return r
}
