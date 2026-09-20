# Diretrizes e Regras de Desenvolvimento — Radar Enfermagem RS

> Manual consolidado e estruturado de padrões de engenharia, arquitetura, boas práticas e regras para agentes e desenvolvedores do projeto **Radar Enfermagem RS**.
>
> **Fontes canônicas:** Regras detalhadas em [`.agents/rules/`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules).

---

## 🧭 Sumário das Diretrizes

1. [Visão Geral & Idioma do Projeto](#1-visão-geral--idioma-do-projeto)
2. [Estratégia de Testes: Troféu de Testes & TDD](#2-estratégia-de-testes-troféu-de-testes--tdd)
3. [Boas Práticas de Backend (Go)](#3-boas-práticas-de-backend-go)
4. [Banco de Dados, PostgreSQL 18 & sqlc](#4-banco-de-dados-postgresql-18--sqlc)
5. [Observabilidade de Primeira Classe](#5-observabilidade-de-primeira-classe)
6. [Frontend & Interface de Usuário (HTMX)](#6-frontend--interface-de-usuário-htmx)
7. [Fluxo Git & Política de Commits/Pushes](#7-fluxo-git--política-de-commitspushes)
8. [Checklist de Qualidade para PRs e Agentes](#8-checklist-de-qualidade-para-prs-e-agentes)

---

## 1. Visão Geral & Idioma do Projeto

- **Objetivo do Projeto:** O **Radar Enfermagem RS** é uma plataforma web para monitorar, centralizar e pesquisar vagas de Técnico em Enfermagem em Porto Alegre/RS e região metropolitana, coletando dados de portais hospitalares oficiais e agregadores.
- **Idioma Obrigatório (pt-BR):**
  - O aplicativo é voltado ao público brasileiro.
  - **Todas as mensagens de erro da API, erros sentinela e validações de domínio DEVEM ser em Português (pt-BR)** (ex.: `"vaga não encontrada"`, `"o título da vaga é obrigatório"`).
  - Comentários de código e documentação operacional devem priorizar clareza em português.
  - Identificadores de código (nomes de funções, variáveis e tipos) seguem a convenção idiomática em inglês técnico (`Job`, `Repository`, `Insert`, `FindByID`).
- **Arquitetura em Monorepo:**
  - `apps/api`: Backend Go (Chi, pgx/v5, sqlc, slog, golang-migrate).
  - `apps/web`: Frontend com templates HTML, CSS utilitário e HTMX.
  - `deployments`: Docker Compose, PostgreSQL 18-alpine e Dockerfiles.
  - `docs`: Documentação de arquitetura, roadmap e planejamento.

---

## 2. Estratégia de Testes: Troféu de Testes & TDD

*Baseado em [`.agents/rules/tdd.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/tdd.md)*

Adotamos a filosofia do **Troféu de Testes (Testing Trophy)** como guia estratégico:

```text
        /   E2E   \        -> Fluxos críticos completos ponta a ponta
       / INTEGRATION \   -> [FOCO PRIORITÁRIO & CORAÇÃO] Banco real, queries, contratos
      /     UNIT      \  -> Algoritmos puros, isolados e regras matemáticas
     / STATIC ANALYSIS \ -> Compilador Go, go vet, golangci-lint, validação sqlc
```

### 2.1 Priorização de Testes
- **Foco Máximo em Testes de Integração:** **Priorize testes de integração sobre testes unitários.** Testes de integração oferecem o maior retorno sobre investimento (ROI) e alta confiança contra regressões.
- **Banco de Dados Real (Sem Mocks Frágeis):** A persistência é testada contra a instância real do PostgreSQL 18 (rodando no container Docker), validando SQL real, triggers, constraints, tipos e transações. Não utilize `sqlmock` para validar persistência.
- **Testes Unitários Enxutos:** Reservados exclusivamente para algoritmos puros, sem efeitos colaterais de I/O (ex.: cálculo de hash SHA-256 em `Fingerprint`, normalização de texto Unicode NFD e validações de invariantes).
- **Ciclo TDD Obrigatório:**
  1. **Red:** Escreva o teste automatizado (de integração ou unitário) descrevendo o comportamento esperado e valide que ele falha.
  2. **Green:** Implemente o código estritamente necessário para fazer o teste passar.
  3. **Refactor:** Limpe, simplifique e formate o código mantendo toda a suíte verde.
- **Cobertura de Edge Cases:** Obrigatoriedade de testar entradas nulas, vazias, limites de paginação, caracteres especiais/acentos e violação de constraints (`UNIQUE`, foreign keys).

---

## 3. Boas Práticas de Backend (Go)

*Baseado em [`.agents/rules/golang-best-practices.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/golang-best-practices.md)*

### 3.1 Simplicidade e Design Idiomático
- **Código Simples e Óbvio:** Evite abstrações especulativas, interfaces desnecessárias ou simulação de herança.
- **Estrutura por Domínio/Responsabilidade:**
  - Prefira pacotes coesos por domínio (`internal/job`, `internal/database`, `internal/config`, `internal/http`).
  - **Evite camadas Java-style artificiais** como `controllers/`, `services/`, `repositories/`, `models/`, `utils/`.
- **Nomenclatura Curta e Contextual:**
  - Evite redundâncias: dentro de `package job`, use `job.Job`, `job.Repository`, `job.Status` (em vez de `job.JobService` ou `job.JobStatus`).
  - Preserve siglas e initialisms: `ID`, `URL`, `UUID`, `HTTP`, `API`, `SQL`, `JSON`.

### 3.2 Erros e Invariantes
- **Erros são Valores:** Trate erros explicitamente. Nunca ignore erros com `_`.
- **Error Wrapping:** Sempre contextualize erros com `%w`:
  ```go
  if err != nil {
      return fmt.Errorf("inserir vaga: %w", err)
  }
  ```
- **Erros Sentinela:** Defina sentinelas tipadas e use `errors.Is`:
  ```go
  var ErrNotFound = errors.New("vaga não encontrada")
  if errors.Is(err, job.ErrNotFound) { ... }
  ```
- **Proibição de Panic:** Nunca use `panic` para fluxo de controle comum.

### 3.3 Contexto e Concorrência
- **`context.Context`:** Deve ser sempre o primeiro parâmetro (`ctx context.Context`). Nunca armazene contextos dentro de structs. Nunca passe `nil`.
- **Concorrência Segura:**
  - Toda goroutine deve ter ciclo de vida, condição de parada e cancelamento bem definidos.
  - Sempre execute testes com o race detector: `go test -race ./...`.
  - O criador/produtor de um canal é quem tem a responsabilidade de fechá-lo.
- **Interfaces Enxutas:** Aceite interfaces pequenas onde necessário e retorne tipos concretos dos construtores (`NewJobRepository(...) *JobRepository`).

---

## 4. Banco de Dados, PostgreSQL 18 & sqlc

*Baseado em [`.agents/rules/sqlc.guideliness.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/sqlc.guideliness.md)*

### 4.1 SQL Direto com sqlc
- **SQL como Código de Produção:** Escreva SQL explícito e direto. **Não introduza ORMs.**
- **pgx/v5 & Tipos Fortes:** Utilize `sql_package: "pgx/v5"`. Configure overrides no `sqlc.yaml` para mapear tipos nativos como `uuid` diretamente para `github.com/google/uuid.UUID`.
- **PostgreSQL 18 Nativo:** Utilize recursos modernos nativos do banco, como `id UUID PRIMARY KEY DEFAULT uuidv7()`, eliminando extensões desnecessárias.

### 4.2 Regras de Consulta (Query Rules)
- **Proibido `SELECT *`:** Especifique sempre as colunas explicitamente para otimizar network, payload e index-only scans.
- **Proibição Absoluta de N+1:** Nunca execute queries dentro de loops em Go. Utilize `JOIN`, agregações em SQL ou busca em lote com `WHERE id = ANY($1)`.
- **Paginação Determinística e Bounded:**
  - Toda listagem pública deve ter limite máximo defensivo (`LIMIT 20`, max `100`).
  - A ordenação deve ser determinística: `ORDER BY published_at DESC NULLS LAST, id DESC`.
- **Operações Atômicas e Upserts:**
  - Prefira `ON CONFLICT (source, external_id) DO UPDATE ... RETURNING ...` com constraint `UNIQUE (source, external_id)`.
  - Evite corridas de leitura-verificação-escrita (*check-then-act*).
- **Tradução de Erros:** No repositório, converta `pgx.ErrNoRows` para o erro sentinela de domínio `job.ErrNotFound`.

---

## 5. Observabilidade de Primeira Classe

*Baseado em [`.agents/rules/observability-guide.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/observability-guide.md)*

### 5.1 Structured Logging com `log/slog`
- Utilize a biblioteca padrão `log/slog`.
- Produção: formato JSON estruturado (`slog.NewJSONHandler`).
- Desenvolvimento: formato texto legível (`slog.NewTextHandler`).
- **Campos Estruturados (Nunca Concatenação):**
  ```go
  logger.ErrorContext(ctx, "falha ao inserir vaga",
      slog.String("source", j.Source),
      slog.String("external_id", j.ExternalID),
      slog.String("erro", err.Error()),
  )
  ```
- **Injeção Explícita:** Repositórios e serviços recebem `*slog.Logger` injetado no construtor. Se `nil`, utilizam `slog.Default()`.

### 5.2 Níveis de Log e Rastreabilidade
- `DEBUG`: Diagnósticos granulares (duração de queries, páginas parseadas).
- `INFO`: Marcos de ciclo de vida (aplicação iniciada, lote de vagas importado com sucesso).
- `WARN`: Anomalias contornadas (tentativa de duplicação, retentativas de rede).
- `ERROR`: Falhas reais de operação (queda de banco, timeout de API externa).
- **Correlação:** Propague `request_id` e trace context através de `context.Context` para todos os logs da operação.

---

## 6. Frontend & Interface de Usuário (HTMX)

*Baseado em [`.agents/rules/frontend-guidelines.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/frontend-guidelines.md)*

### 6.1 Identidade Visual Própria (Domínio de Enfermagem)
- O design deve comunicar o domínio de **saúde e enfermagem**: tons de branco, verde hospitalar, azul suave, tipografia limpa e sem ruído visual.
- **Evite Aparência Genérica de IA:** Não use gradientes roxos/azuis genéricos, glassmorphism ou dashboards corporativos sem propósito clínico.
- **Acessibilidade & Mobile First:** A maior parte dos profissionais de enfermagem acessa via smartphone. Contraste alto, botões acessíveis e hierarquia visual clara são obrigatórios.

### 6.2 Arquitetura HTMX
- Renderização no servidor (SSR) com templates em Go e fragments HTML.
- **Sem SPA pesada:** Atualizações parciais de tela usando atributos `hx-get`, `hx-target`, `hx-swap` e `hx-indicator`.
- Estados de interface explícitos: loading indicators, cards vazios (*empty state*), tratamento amigável de erros e listagem sem reload completo da página.

---

## 7. Fluxo Git & Política de Commits/Pushes

*Baseado em [`.agents/rules/git-workflow.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/git-workflow.md)*

### 7.1 Autorização Formal Obrigatória para Commit e Push
- **Proibição de Operações Autônomas:** É **terminantemente proibido** a qualquer agente executar `git commit`, `git push`, criação ou merge de PRs sem solicitar e receber confirmação explícita do usuário (ex.: `"pode commitar"`, `"autorizo o push"`).
- **Protocolo de Entrega:** O agente finaliza as alterações no código, valida os testes e linter, exibe o `git status` e a proposta de mensagem de commit, e aguarda autorização formal antes de commitar.

### 7.2 Branches e Conventional Commits
- `main`: Branch de produção estritamente protegida (somente merges via PR com CI verde).
- `develop`: Branch de integração padrão para novos desenvolvimentos e correções.
- Padrão de commits: `<tipo>(<escopo>): <descrição>` (Conventional Commits).

---

## 8. Checklist de Qualidade para PRs e Agentes

Antes de submeter código ou considerar qualquer milestone/tarefa concluída:

- [ ] A autorização formal e explícita do usuário foi obtida ANTES de qualquer `git commit` ou `git push`?
- [ ] As mensagens de erro e validações estão em **Português (pt-BR)**?
- [ ] O ciclo **TDD** foi seguido?
- [ ] Foram implementados **testes de integração reais** contra o PostgreSQL (Troféu de Testes)?
- [ ] As queries sqlc evitam `SELECT *` e previnem **N+1**?
- [ ] A ordenação e paginação são **determinísticas** e possuem limites?
- [ ] Os logs utilizam **`slog` com campos estruturados** (sem strings concatenadas)?
- [ ] `make test` passa com **`go test -race`** em todos os pacotes?
- [ ] `go vet ./...` e `gofmt -s -w .` foram executados sem erros?
- [ ] As migrações versionadas possuem scripts `up` e `down` reversíveis e validados?
- [ ] O arquivo [`docs/roadmap.md`](file:///home/marcos/Projects/radar-enfermagem-rs/docs/roadmap.md) foi atualizado com o progresso das tarefas?
