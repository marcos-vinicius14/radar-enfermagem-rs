Radar Enfermagem RS

Objetivo

Construir o Radar Enfermagem RS, uma aplicação web gratuita para monitorar, centralizar e pesquisar vagas de Técnico em Enfermagem em Porto Alegre/RS e região metropolitana.

A primeira versão irá monitorar diretamente os portais de carreira das seguintes instituições:

Santa Casa de Porto Alegre

Hospital Moinhos de Vento

Hospital São Lucas da PUCRS

Unimed

Doctor Clin

Grupo Fleury / Weinmann

HCPA

Hospital Divina Providência

Hospital Mãe de Deus

Em uma segunda etapa, serão adicionados agregadores como LinkedIn, Indeed, InfoJobs e similares.

Stack

Backend

Go

Chi

PostgreSQL

pgx

sqlc

golang-migrate

goquery

robfig/cron

slog

golang.org/x/sync/errgroup

golang.org/x/time/rate

Frontend

HTMX

HTML renderizado pelo backend

CSS utilitário ou stylesheet próprio

JavaScript mínimo, apenas quando necessário

Infraestrutura

Docker

Docker Compose

PostgreSQL

API e frontend containerizados

Arquitetura

                       ┌──────────────────────┐
                       │      Scheduler       │
                       └──────────┬───────────┘
                                  │
                                  v
                    ┌─────────────────────────┐
                    │       Collectors        │
                    │                         │
                    │ SantaCasaCollector      │
                    │ MoinhosCollector        │
                    │ SaoLucasCollector       │
                    │ UnimedCollector         │
                    │ DoctorClinCollector     │
                    │ FleuryCollector         │
                    │ HCPACollector           │
                    │ DivinaCollector         │
                    │ MaeDeDeusCollector      │
                    └───────────┬─────────────┘
                                │
                                v
                     ┌────────────────────────┐
                     │     Normalization      │
                     └──────────┬─────────────┘
                                │
                                v
                     ┌────────────────────────┐
                     │     Deduplication      │
                     └──────────┬─────────────┘
                                │
                                v
                         ┌─────────────┐
                         │ PostgreSQL  │
                         └──────┬──────┘
                                │
                                v
                          ┌───────────┐
                          │  Go API   │
                          └─────┬─────┘
                                │
                                v
                            ┌───────┐
                            │ HTMX  │
                            └───────┘

A coleta será executada em background por um scheduler. O frontend nunca deverá disparar scraping diretamente.

Fluxo de Coleta

Portal da instituição
        ↓
Collector
        ↓
RawJob
        ↓
Normalizer
        ↓
Job
        ↓
Deduplicator
        ↓
PostgreSQL

Cada instituição deverá possuir um collector independente.

Contrato dos Collectors

type Collector interface {
    Name() string

    Collect(
        ctx context.Context,
        query SearchQuery,
    ) ([]RawJob, error)
}

Exemplo:

type RawJob struct {
    ExternalID string
    Title      string
    Company    string
    Location   string
    URL        string
    Content    string
    Published  string
}

O collector não deve persistir diretamente a entidade de domínio.

Entidade Job

type Job struct {
    ID          uuid.UUID

    ExternalID  string
    Title       string
    Company     string
    Description string

    City        string
    State       string

    Source      string
    SourceURL   string

    WorkMode    WorkMode
    Employment  EmploymentType

    SalaryMin   *int64
    SalaryMax   *int64

    PublishedAt *time.Time
    CollectedAt time.Time
    LastSeenAt  time.Time

    Status      JobStatus
}

Status possíveis:

const (
    JobStatusActive  JobStatus = "ACTIVE"
    JobStatusUnknown JobStatus = "UNKNOWN"
    JobStatusExpired JobStatus = "EXPIRED"
)

Deduplicação

A mesma vaga pode aparecer mais de uma vez ou ser republicada.

Deduplicação exata

source + external_id

Constraint:

UNIQUE(source, external_id)

Deduplicação lógica

Gerar fingerprint utilizando:

empresa normalizada
+
título normalizado
+
cidade

Exemplo:

func Fingerprint(job Job) string {
    value := strings.Join([]string{
        normalize(job.Company),
        normalize(job.Title),
        normalize(job.City),
    }, "|")

    sum := sha256.Sum256([]byte(value))

    return hex.EncodeToString(sum[:])
}

Banco de Dados

Tabela principal:

CREATE TABLE jobs (
    id UUID PRIMARY KEY,

    external_id VARCHAR(255),

    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    description TEXT,

    city VARCHAR(100),
    state CHAR(2),

    source VARCHAR(100) NOT NULL,
    source_url TEXT NOT NULL,

    fingerprint VARCHAR(64),

    work_mode VARCHAR(30),
    employment_type VARCHAR(30),

    salary_min BIGINT,
    salary_max BIGINT,

    published_at TIMESTAMPTZ,
    collected_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,

    status VARCHAR(30) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    UNIQUE(source, external_id)
);

Índices iniciais:

CREATE INDEX idx_jobs_title
ON jobs(title);

CREATE INDEX idx_jobs_city
ON jobs(city);

CREATE INDEX idx_jobs_company
ON jobs(company);

CREATE INDEX idx_jobs_published_at
ON jobs(published_at DESC);

CREATE INDEX idx_jobs_fingerprint
ON jobs(fingerprint);

Scheduler

As fontes serão atualizadas periodicamente.

Exemplo:

cron.AddFunc(
    "0 */2 * * *",
    func() {
        service.CollectJobs(ctx)
    },
)

Frequência inicial sugerida:

a cada 2 horas

Os collectors poderão executar em paralelo com limite de concorrência.

group, ctx := errgroup.WithContext(ctx)
group.SetLimit(5)

Rate Limiting

Cada domínio deverá possuir seu próprio limite de requisições.

Exemplo:

gupy.io             -> limite específico
hospital.com.br     -> limite específico
unimed.com.br       -> limite específico

Utilizar:

golang.org/x/time/rate

O objetivo é evitar carga excessiva, bloqueios e comportamento agressivo de scraping.

API

Endpoint principal:

GET /api/v1/jobs

Filtros previstos:

?query=tecnico enfermagem
&city=porto-alegre
&state=RS
&company=moinhos-de-vento
&source=gupy
&page=1
&size=20

Endpoints iniciais:

GET /api/v1/jobs
GET /api/v1/jobs/{id}

GET /api/v1/companies
GET /api/v1/cities
GET /api/v1/sources

Resposta paginada:

{
  "items": [],
  "page": 1,
  "size": 20,
  "total": 134
}

Frontend

O frontend será implementado com HTMX, priorizando HTML renderizado no servidor e interações incrementais sem a necessidade de uma SPA.

A interface deverá seguir um design clean, minimalista e funcional, com baixa poluição visual e foco na leitura rápida das vagas.

A identidade visual deverá utilizar uma combinação de cores, tipografia e ícones que remetam à área de enfermagem e saúde, mantendo aparência profissional e discreta. A paleta pode explorar tons como branco, verde, azul e variações suaves, evitando excesso de cores ou elementos decorativos.

Os ícones deverão reforçar visualmente conceitos como:

enfermagem;

hospital;

localização;

calendário;

turno;

especialidade;

candidatura;

instituição.

A prioridade do design será:

legibilidade;

hierarquia visual clara;

navegação simples;

boa experiência em dispositivos móveis;

destaque para informações essenciais da vaga;

consistência visual entre páginas e fragments HTMX.

A API e a camada web poderão compartilhar o mesmo backend em Go, mantendo handlers separados para:

endpoints JSON;

páginas HTML;

fragments HTML utilizados pelo HTMX.

Tela inicial:

┌─────────────────────────────────────────────┐
│ Técnico de Enfermagem                      │
│                                             │
│ [cargo................] [cidade........]   │
│                                             │
│ [Buscar]                                    │
└─────────────────────────────────────────────┘

134 vagas encontradas

[ Porto Alegre ] [ Canoas ] [ Remoto ]

-----------------------------------------------
Técnico de Enfermagem — UTI
Hospital Moinhos de Vento

Porto Alegre / RS
Publicado há 2 dias

Fonte: Portal oficial

[Ver vaga]
-----------------------------------------------

Filtros principais:

cargo

cidade

instituição

modalidade

especialidade

data de publicação

Estrutura do Backend

internal/
├── job/
│   ├── domain/
│   │   ├── job.go
│   │   ├── repository.go
│   │   └── collector.go
│   │
│   ├── application/
│   │   ├── collect_jobs.go
│   │   ├── search_jobs.go
│   │   └── deduplicate_jobs.go
│   │
│   └── infrastructure/
│       ├── postgres/
│       └── collectors/
│           ├── santacasa/
│           ├── moinhos/
│           ├── saolucas/
│           ├── unimed/
│           ├── doctorclin/
│           ├── fleury/
│           ├── hcpa/
│           ├── divina/
│           └── maededeus/
│
├── http/
│   ├── handlers/
│   ├── middleware/
│   └── router.go
│
├── scheduler/
└── config/

Estrutura do Frontend

Como o frontend utilizará HTMX, a estrutura será orientada a templates e fragments renderizados pelo backend:

apps/web/
├── templates/
│   ├── layouts/
│   ├── pages/
│   │   └── jobs/
│   └── fragments/
│       └── jobs/
├── static/
│   ├── css/
│   ├── js/
│   └── images/
└── handlers/

Os fragments serão usados para atualizar partes da interface via HTMX, como:

lista de vagas;

paginação;

filtros;

contagem de resultados;

detalhes resumidos da vaga.

Estrutura do Repositório

O projeto será mantido como monorepo, concentrando backend, frontend HTMX, migrations, infraestrutura e documentação em um único repositório.

radar-enfermagem-rs/
├── apps/
│   ├── api/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── migrations/
│   │   └── go.mod
│   │
│   └── web/
│       ├── templates/
│       │   ├── layouts/
│       │   ├── pages/
│       │   └── fragments/
│       ├── static/
│       │   ├── css/
│       │   ├── js/
│       │   └── images/
│       └── handlers/
│
├── deployments/
│   ├── docker-compose.yml
│   └── Dockerfile
│
├── docs/
└── README.md

MVP

Fase 1 — Fundação

Nesta fase será criada a base técnica do projeto e a estrutura inicial do monorepo.

Serão configurados:

estrutura do monorepo;

aplicação Go;

frontend com HTMX;

PostgreSQL;

migrations;

configuração por variáveis de ambiente;

Docker e Docker Compose;

logging estruturado;

tratamento básico de erros.

Também será feita a modelagem inicial da entidade Job, definição dos contratos de repository e criação das primeiras queries com sqlc.

Ao final desta fase, a aplicação já deverá subir localmente com banco, backend e interface web funcionando, mesmo que ainda sem coleta automática de vagas.

Fase 2 — Primeiro Collector e Pipeline de Coleta

Será implementado o primeiro collector utilizando uma das instituições como referência arquitetural para as demais.

O objetivo é validar todo o fluxo:

Portal da instituição
        ↓
Collector
        ↓
RawJob
        ↓
Normalizer
        ↓
Job
        ↓
Deduplicator
        ↓
Repository
        ↓
PostgreSQL

Nesta fase serão definidos:

contrato comum dos collectors;

modelo RawJob;

normalização dos dados recebidos;

persistência idempotente;

identificação da vaga pela fonte;

geração de fingerprint;

atualização de last_seen_at.

O primeiro collector servirá como padrão para as próximas integrações.

Fase 3 — Demais Portais Oficiais

Após validar o pipeline, serão implementados os demais collectors dos portais oficiais:

Santa Casa

Moinhos de Vento

Hospital São Lucas

Unimed

Doctor Clin

Fleury / Weinmann

HCPA

Divina Providência

Mãe de Deus

Cada integração deverá ficar isolada em seu próprio pacote.

Mudanças em uma fonte não devem impactar as demais.

Quando diferentes instituições utilizarem o mesmo provedor de recrutamento, como Gupy ou outra plataforma comum, será criado um collector reutilizável baseado no provedor em vez de duplicar lógica.

Fase 4 — Coleta Automática e Controle de Estado

A coleta deixará de ser manual e passará a ser executada periodicamente por um scheduler.

Inicialmente, a coleta poderá rodar a cada duas horas.

Nesta fase serão adicionados:

scheduler;

execução concorrente dos collectors;

limite de concorrência;

timeout por requisição;

retry apenas para falhas transitórias;

rate limiting por domínio;

logs por execução;

métricas básicas de quantidade encontrada, criada e atualizada;

controle de last_seen_at;

atualização de status das vagas.

Uma vaga encontrada novamente terá sua informação atualizada e seu last_seen_at renovado.

Vagas que deixarem de aparecer por determinado período poderão passar de:

ACTIVE
  ↓
UNKNOWN
  ↓
EXPIRED

Nenhuma vaga deverá ser removida fisicamente apenas por deixar de aparecer no portal.

Fase 5 — API e Interface de Listagem

A aplicação disponibilizará os dados coletados através da API Go e da interface web renderizada no servidor.

A página principal exibirá:

título da vaga;

instituição;

cidade;

data de publicação;

origem;

status;

link para candidatura.

Os filtros iniciais serão:

cidade;

instituição;

especialidade;

status;

data de publicação.

As interações de filtro e paginação serão feitas com HTMX.

Somente os fragments necessários serão atualizados, evitando reload completo da página.

Fase 6 — Qualidade da Busca

A busca textual será implementada utilizando o Full Text Search nativo do PostgreSQL.

Os principais campos pesquisáveis serão combinados em um tsvector, por exemplo:

título;

instituição;

cidade;

descrição.

O vetor será indexado com GIN para evitar scans completos da tabela.

As consultas poderão utilizar websearch_to_tsquery ou plainto_tsquery, com ordenação por relevância via ts_rank.

Nesta fase também serão refinados:

normalização textual;

deduplicação;

fingerprint;

filtros avançados;

ordenação por data;

relevância dos resultados.

Fase 7 — Barra de Pesquisa

Será adicionada uma barra de pesquisa global na página principal.

O usuário poderá digitar termos livres, como:

técnico enfermagem uti
centro cirúrgico
moinhos
canoas
emergência

Fluxo:

Barra de pesquisa
        ↓
HTMX
        ↓
Handler Go
        ↓
PostgreSQL Full Text Search
        ↓
Resultados ordenados por relevância
        ↓
Fragment HTML
        ↓
Atualização da lista

A pesquisa poderá ser combinada com os filtros existentes, como cidade, instituição, especialidade, status e data.

A interface deverá atualizar os resultados sem recarregar a página inteira.

Fase 8 — Agregadores

Somente depois dos portais oficiais estarem estáveis serão adicionadas fontes agregadoras.

Inicialmente:

LinkedIn;

Indeed;

InfoJobs;

Jobbol;

outros agregadores relevantes.

Essas fontes serão tratadas como complementares.

O sistema continuará priorizando os portais oficiais como fonte principal.

Os mesmos contratos de collector, normalização e deduplicação serão reutilizados.

Fase 9 — Funcionalidades Futuras

Após o MVP estar estável, poderão ser adicionadas:

autenticação;

favoritos;

filtros salvos;

alertas;

Telegram;

e-mail;

notificações push;

histórico de vagas;

ranking de relevância;

expansão para outras profissões;

expansão para outras regiões do Rio Grande do Sul.

Prioridade Técnica

Ordem recomendada de implementação:

1. Modelagem do domínio
2. Schema PostgreSQL
3. API REST
4. Primeiro collector
5. Pipeline RawJob -> Job
6. Persistência
7. Scheduler
8. Deduplicação
9. Frontend HTMX
10. Demais instituições
11. Alertas
12. Agregadores

O primeiro collector deve servir como referência arquitetural para todos os demais.

Princípios do Projeto

Collectors isolados por fonte.

Nenhuma regra de scraping dentro da camada HTTP.

Normalização separada da coleta.

Persistência separada dos collectors.

Rate limiting por domínio.

Timeouts obrigatórios em chamadas externas.

Retry apenas para erros transitórios.

Idempotência na persistência.

Vagas antigas não devem ser apagadas.

Manter histórico de disponibilidade.

Preferir portais oficiais antes de agregadores.

Não acoplar o domínio ao formato retornado por sites externos.

Toda nova fonte deve ser adicionada através do mesmo contrato de collector.
