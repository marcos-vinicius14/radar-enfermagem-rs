package collector

import (
	"net/http"
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

// DefaultRegistry retorna um registro inicializado com todos os 9 portais hospitalares monitorados usando cliente padrão.
func DefaultRegistry(timeout time.Duration) *Registry {
	return NewDefaultRegistry(nil, timeout)
}

// NewDefaultRegistry retorna um registro inicializado com todos os 9 portais hospitalares monitorados,
// utilizando o cliente HTTP fornecido (permitindo injeção de cliente resiliente com rate limit e retries).
func NewDefaultRegistry(client *http.Client, timeout time.Duration) *Registry {
	r := NewRegistry()
	r.Register(NewSantaCasaCollector(client, timeout))
	r.Register(NewMoinhosCollector(client, timeout))
	r.Register(NewSaoLucasCollector(client, timeout))
	r.Register(NewUnimedCollector(client, timeout))
	r.Register(NewDoctorClinCollector(client, timeout))
	r.Register(NewFleuryCollector(client, timeout))
	r.Register(NewHCPACollector(client, timeout))
	r.Register(NewDivinaCollector(client, timeout))
	r.Register(NewMaeDeDeusCollector(client, timeout))
	return r
}
