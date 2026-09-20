# ROADMAP — Radar Enfermagem RS

Este roadmap organiza o desenvolvimento do **Radar Enfermagem RS** em milestones e tasks estruturadas como checklist para acompanhamento de progresso.

---

## 📊 Visão Geral das Milestones

| Milestone | Descrição | Status |
| :--- | :--- | :---: |
| [Milestone 1](#milestone-1--fundação-do-projeto) | Fundação do Projeto (Monorepo, Go, Chi, Postgres 18, Migrations, Docker) | Concluída |
| [Milestone 2](#milestone-2--domínio-e-persistência-de-vagas) | Domínio e Persistência de Vagas (`jobs`, UUIDv7, sqlc, Repository) | Concluída |
| [Milestone 3](#milestone-3--pipeline-de-coleta) | Pipeline de Coleta (Contrato Collector, RawJob, Normalizer, Deduplicator) | Concluída |
| [Milestone 4](#milestone-4--portais-oficiais) | Portais Oficiais (Santa Casa, Moinhos, São Lucas, Unimed, etc.) | Concluída |
| [Milestone 5](#milestone-5--scheduler-e-resiliência) | Scheduler e Resiliência (Cron, Errgroup, Rate Limiting, Retries) | Concluída |
| [Milestone 6](#milestone-6--api-de-consulta) | API de Consulta (Endpoints REST paginados e filtráveis) | A Fazer |
| [Milestone 7](#milestone-7--frontend-htmx) | Frontend HTMX (Layout clean, cards, listagem e filtros dinâmicos) | A Fazer |
| [Milestone 8](#milestone-8--full-text-search) | Full Text Search (Busca textual nativa com PostgreSQL GIN) | A Fazer |
| [Milestone 9](#milestone-9--barra-de-pesquisa) | Barra de Pesquisa (Debounce, ranking por relevância com HTMX) | A Fazer |
| [Milestone 10](#milestone-10--agregadores) | Agregadores (LinkedIn, Indeed, InfoJobs, Jobbol) | A Fazer |
| [Milestone 11](#milestone-11--qualidade-e-operação) | Qualidade e Operação (Testes ampliados, CI pipeline, Métricas) | A Fazer |
| [Milestone 12](#milestone-12--pós-mvp) | Pós-MVP (Favoritos, alertas Telegram/Email, expansão de cidades) | A Fazer |

---

## Milestone 1 — Fundação do Projeto

> **Objetivo:** Criar a base técnica do monorepo e garantir que backend, frontend e banco possam ser executados localmente.

- [x] **Task 1: Criar estrutura inicial do monorepo**
  - [x] Repositório inicializado
  - [x] Estrutura de diretórios criada (`apps/api`, `apps/web`, `deployments`, `docs`)
  - [x] `.gitignore` configurado
  - [x] `README.md` inicial criado
  - [x] `Makefile` para orquestração de tarefas

- [x] **Task 2: Inicializar aplicação Go**
  - [x] `go.mod` e `go.work` criados
  - [x] Aplicação inicia sem erros
  - [x] Endpoint de health check disponível
  - [x] Encerramento gracioso configurado

- [x] **Task 3: Configurar servidor HTTP com Chi**
  - [x] Chi configurado
  - [x] Rota `/health` implementada
  - [x] Middleware de recovery
  - [x] Middleware de request logging estruturado (`slog`)

- [x] **Task 4: Configurar PostgreSQL**
  - [x] Container PostgreSQL 18-alpine funcionando
  - [x] Aplicação consegue abrir conexão (`pgxpool`)
  - [x] Configuração por variáveis de ambiente (`.env`)
  - [x] Timeout de conexão configurado

- [x] **Task 5: Configurar migrations**
  - [x] Adicionar `golang-migrate` e estruturar migrations versionadas
  - [x] Diretório de migrations criado (`apps/api/migrations`)
  - [x] Migrations executadas via comando CLI (`cmd/migrate`)
  - [x] Rollback testado e validado

- [x] **Task 6: Configurar pgx e sqlc**
  - [x] Pool `pgx` configurado
  - [x] `sqlc.yaml` configurado
  - [x] Primeira query gerada com sucesso

- [x] **Task 7: Configurar Docker Compose**
  - [x] PostgreSQL sobe via Docker Compose com healthcheck
  - [x] API consegue conectar ao banco
  - [x] Variáveis documentadas em `.env.example`

- [x] **Task 8: Configurar logging estruturado**
  - [x] Utilizar `log/slog`
  - [x] Logs possuem nível (debug, info, warn, error)
  - [x] Logs possuem contexto (`request_id`, status, latência, IP)
  - [x] Erros externos são registrados corretamente

---

## Milestone 2 — Domínio e Persistência de Vagas

> **Objetivo:** Modelar a entidade de vaga e implementar sua persistência.

- [x] **Task 9: Modelar entidade Job**
  - [x] Campos mínimos: `ID` (UUIDv7), `ExternalID`, título, empresa, descrição, cidade, estado, fonte, URL, datas, status, fingerprint
- [x] **Task 10: Criar migration da tabela jobs**
  - [x] Tabela `jobs` criada com `id UUID DEFAULT uuidv7()` (nativo PG 18)
  - [x] Constraint `UNIQUE(source, external_id)`
  - [x] Índices criados (`title`, `city`, `company`, `published_at`, `fingerprint`)
  - [x] Timestamps definidos (`created_at`, `updated_at`, `last_seen_at`)
- [x] **Task 11: Criar JobRepository**
  - [x] Inserir vaga
  - [x] Atualizar vaga
  - [x] Buscar por ID
  - [x] Buscar por fonte + external ID
  - [x] Listar vagas
  - [x] Atualizar `last_seen_at`
- [x] **Task 12: Implementar JobRepository PostgreSQL**
  - [x] Implementar repository usando queries geradas pelo `sqlc`
  - [x] Testes de integração cobrindo inserção, busca e atualização
- [x] **Task 13: Implementar geração de fingerprint**
  - [x] Gerar fingerprint com: empresa normalizada + título normalizado + cidade
  - [x] SHA-256 utilizado
  - [x] Testes cobrindo caixa alta/baixa, múltiplos espaços e acentos

---

## Milestone 3 — Pipeline de Coleta

> **Objetivo:** Criar a abstração de collectors e validar todo o pipeline com uma primeira fonte.

- [x] **Task 14: Criar contrato Collector**
  ```go
  type Collector interface {
      Name() string
      Collect(ctx context.Context, query SearchQuery) ([]RawJob, error)
  }
  ```
- [x] **Task 15: Criar modelo RawJob**
  - [x] Representação intermediária que aceita dados incompletos das fontes e desacoplada da entidade `Job`
- [x] **Task 16: Criar Normalizer**
  - [x] Converter `RawJob` em `Job`
  - [x] Normalizações: título, empresa, cidade, estado, URL e data de publicação
- [x] **Task 17: Criar Deduplicator**
  - [x] Deduplicação exata por `source + external_id`
  - [x] Deduplicação lógica por `fingerprint`
- [x] **Task 18: Criar serviço CollectJobs**
  - [x] Orquestração: `Collector` → `RawJob` → `Normalizer` → `Deduplicator` → `Repository`
  - [x] Fluxo idempotente
  - [x] Atualização de vagas existentes
  - [x] Inserção de novas vagas
  - [x] Atualização de `last_seen_at`
- [x] **Task 19: Implementar primeiro collector (referência)**
  - [x] Coleta vagas reais
  - [x] Retorna `[]RawJob`
  - [x] Possui timeout configurado
  - [x] Possui testes de parsing
  - [x] Não persiste diretamente no banco

---

## Milestone 4 — Portais Oficiais

> **Objetivo:** Adicionar todas as instituições inicialmente monitoradas. Cada collector deve respeitar o contrato, timeout, rate limit e testes de parsing.

- [x] **Task 20: Implementar collector da Santa Casa**
- [x] **Task 21: Implementar collector do Moinhos de Vento**
- [x] **Task 22: Implementar collector do Hospital São Lucas**
- [x] **Task 23: Implementar collector da Unimed**
- [x] **Task 24: Implementar collector da Doctor Clin**
- [x] **Task 25: Implementar collector do Fleury / Weinmann**
- [x] **Task 26: Implementar collector do HCPA**
- [x] **Task 27: Implementar collector da Divina Providência**
- [x] **Task 28: Implementar collector do Hospital Mãe de Deus**
- [x] **Task 29: Criar collectors reutilizáveis por provedor**
  - [x] Abstrair integrações compartilhadas (Gupy, Senior Sistemas, Vagas.com) evitando duplicação

---

## Milestone 5 — Scheduler e Resiliência

> **Objetivo:** Automatizar a coleta e tornar o processo resiliente.

- [x] **Task 30: Configurar scheduler**
  - [x] Frequência inicial: a cada 2 horas (`robfig/cron`)
- [x] **Task 31: Executar collectors concorrentemente**
  - [x] Usar `golang.org/x/sync/errgroup` com limite de concorrência
- [x] **Task 32: Implementar rate limiting**
  - [x] Usar `golang.org/x/time/rate` com limites configuráveis por domínio
- [x] **Task 33: Implementar política de retry**
  - [x] Retry apenas para falhas transitórias (timeout, 5xx, falha temporária de rede)
  - [x] Não retentar erros definitivos de 4xx ou parsing inválido
- [x] **Task 34: Implementar controle de status das vagas**
  - [x] Ciclo de vida: `ACTIVE` → `UNKNOWN` → `EXPIRED`
- [x] **Task 35: Adicionar métricas básicas de coleta**
  - [x] Contadores: vagas encontradas, novas, atualizadas, ignoradas, erros por fonte e tempo de execução

---

## Milestone 6 — API de Consulta

> **Objetivo:** Expor as vagas armazenadas de forma paginada e filtrável.

- [ ] **Task 36: Implementar `GET /api/v1/jobs`**
  - [ ] Filtros: `query`, `city`, `state`, `company`, `status`, `date`, `page`, `size`
- [ ] **Task 37: Implementar `GET /api/v1/jobs/{id}`**
- [ ] **Task 38: Implementar `GET /api/v1/companies`**
- [ ] **Task 39: Implementar `GET /api/v1/cities`**
- [ ] **Task 40: Implementar `GET /api/v1/sources`**
- [ ] **Task 41: Implementar paginação**
  - [ ] Limite máximo configurado
  - [ ] Metadados de paginação (`items`, `page`, `size`, `total`)
  - [ ] Ordenação determinística

---

## Milestone 7 — Frontend HTMX

> **Objetivo:** Criar uma interface clean, minimalista e focada na busca rápida de vagas.

- [ ] **Task 42: Criar layout base**
  - [ ] Header, conteúdo principal, footer e responsividade básica
- [ ] **Task 43: Criar página inicial**
  - [ ] Título do projeto, barra de pesquisa, filtros, lista de vagas e contador
- [ ] **Task 44: Criar card de vaga**
  - [ ] Exibir título, instituição, cidade, data de publicação, especialidade, fonte e link de candidatura
- [ ] **Task 45: Implementar listagem com HTMX**
  - [ ] Atualização dinâmica apenas do fragment da listagem
- [ ] **Task 46: Implementar filtros com HTMX**
  - [ ] Filtros por cidade, instituição, especialidade, status e período
- [ ] **Task 47: Implementar paginação com HTMX**
  - [ ] Navegação de páginas sem reload completo
- [ ] **Task 48: Implementar estados de interface**
  - [ ] Estados visuais: loading, vazio, erro, sem resultados e dados carregados

---

## Milestone 8 — Full Text Search

> **Objetivo:** Implementar busca textual nativa utilizando PostgreSQL.

- [ ] **Task 49: Adicionar coluna `search_vector`**
  - [ ] Baseado em: título, instituição, cidade e descrição
- [ ] **Task 50: Criar índice GIN**
  ```sql
  CREATE INDEX idx_jobs_search_vector ON jobs USING GIN (search_vector);
  ```
- [ ] **Task 51: Implementar query Full Text Search**
  - [ ] Utilizar `websearch_to_tsquery` ou `plainto_tsquery`
- [ ] **Task 52: Implementar ranking de relevância**
  - [ ] Utilizar `ts_rank`
- [ ] **Task 53: Integrar FTS aos filtros existentes**

---

## Milestone 9 — Barra de Pesquisa

> **Objetivo:** Adicionar pesquisa textual global usando Full Text Search e HTMX.

- [ ] **Task 54: Criar componente da barra de pesquisa**
- [ ] **Task 55: Integrar pesquisa com HTMX**
  - [ ] Fluxo: `Input` → `HTMX` → `Handler Go` → `PostgreSQL FTS` → `Fragment HTML`
- [ ] **Task 56: Adicionar debounce no input**
  - [ ] Evitar requisições excessivas a cada caractere digitado
- [ ] **Task 57: Combinar pesquisa textual com filtros**
- [ ] **Task 58: Implementar ordenação por relevância**
  - [ ] Com busca textual: ordenar por `relevância DESC`
  - [ ] Sem busca textual: ordenar por `published_at DESC`

---

## Milestone 10 — Agregadores

> **Objetivo:** Expandir as fontes após a estabilização dos portais oficiais.

- [ ] **Task 59: Avaliar viabilidade técnica do LinkedIn**
- [ ] **Task 60: Avaliar viabilidade técnica do Indeed**
- [ ] **Task 61: Avaliar viabilidade técnica do InfoJobs**
- [ ] **Task 62: Avaliar viabilidade técnica do Jobbol**
- [ ] **Task 63: Implementar agregadores tecnicamente viáveis**
- [ ] **Task 64: Ajustar deduplicação entre portal oficial e agregadores**
  - [ ] Regra: Sempre que possível, priorizar a vaga publicada na fonte oficial

---

## Milestone 11 — Qualidade e Operação

> **Objetivo:** Preparar o projeto para execução e monitoramento contínuo.

- [ ] **Task 65: Adicionar testes unitários amplos**
  - [ ] Normalização, fingerprint, deduplicação, regras de status e parsing dos collectors
- [ ] **Task 66: Adicionar testes de integração**
  - [ ] Repository, queries, Full Text Search e API
- [x] **Task 67: Criar pipeline de CI**
  - [x] Executar `go test -race`, `golangci-lint`, `build` e validação de migrações
- [ ] **Task 68: Configurar health checks detalhados**
  - [ ] Endpoints `/health` e `/ready` com checagem de dependências
- [ ] **Task 69: Criar documentação operacional**
  - [ ] Guia de variáveis, execução local, migrations, scheduler, collectors e troubleshooting

---

## Milestone 12 — Pós-MVP

> **Objetivo:** Funcionalidades complementares após a validação do produto inicial.

- [ ] **Task 70: Favoritos de vagas**
- [ ] **Task 71: Filtros salvos**
- [ ] **Task 72: Alertas por Telegram**
- [ ] **Task 73: Alertas por e-mail**
- [ ] **Task 74: Notificações push**
- [ ] **Task 75: Histórico de vagas e métricas de mercado**
- [ ] **Task 76: Autenticação de usuários**
- [ ] **Task 77: Expansão para outras cidades do RS**
- [ ] **Task 78: Expansão para outras profissões da saúde**

---

## 🏷️ Labels Sugeridas no GitHub

`backend`, `frontend`, `database`, `collector`, `scraping`, `api`, `htmx`, `search`, `postgres`, `infra`, `testing`, `documentation`, `bug`, `enhancement`, `tech-debt`

---

## 🎯 Prioridades

| Nível | Foco | Componentes |
| :--- | :--- | :--- |
| **P0 — Essencial (MVP)** | Validação do produto principal | Fundação, Banco, Domínio, Collectors Oficiais, Scheduler, API, Listagem, Filtros, Full Text Search, Barra de Pesquisa |
| **P1 — Estabilização** | Confiabilidade e escala | Métricas, Melhorias de UX, Agregadores, Testes de Integração, CI/CD |
| **P2 — Pós-MVP** | Recursos adicionais | Autenticação, Favoritos, Alertas, Notificações, Expansão Geográfica e Profissional |
