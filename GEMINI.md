# Guia do Agente Gemini — Radar Enfermagem RS

> Instruções de contexto, arquitetura, regras de engenharia e comandos para o agente **Gemini / Antigravity** no repositório **Radar Enfermagem RS**.
>
> Baseado integralmente nas regras canônicas em [`.agents/rules/`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules) e no [`AGENTS.md`](file:///home/marcos/Projects/radar-enfermagem-rs/AGENTS.md).

---

## 1. Visão Geral do Projeto & Diretrizes Gerais

- **Contexto:** Aplicação web para monitorar, centralizar e pesquisar vagas de Técnico em Enfermagem em Porto Alegre/RS e região metropolitana.
- **Idioma Obrigatório (pt-BR):**
  - Todas as mensagens de erro (retornos da API, validações de domínio, erros sentinela) **DEVEM ser em Português (pt-BR)** (ex.: `"vaga não encontrada"`, `"o título da vaga é obrigatório"`).
  - Nomes de símbolos de código permanecem em inglês técnico (`Job`, `Repository`, `Fingerprint`, `Insert`).
- **Arquitetura:**
  - Monorepo: backend Go (`apps/api`), frontend HTMX/templates (`apps/web`), Docker Compose e infraestrutura (`deployments`), documentação (`docs`).

---

## 2. Comandos Frequentes de Desenvolvimento

```bash
# Executar todos os testes com detecção de concorrência (-race)
make test

# Executar testes e gerar relatório de cobertura
make test-cover

# Subir banco de dados PostgreSQL 18 em container
make up

# Parar containers
make down

# Aplicar migrações pendentes
make migrate-up

# Rollback da última migração
make migrate-down

# Gerar código Go a partir das queries SQL com sqlc
make sqlc

# Compilar binário da API
make build

# Formatação e análise estática
cd apps/api && go vet ./... && gofmt -s -w .
```

---

## 3. Resumo Estruturado das Regras (`.agents/rules`)

### 3.1 Estratégia de Testes: Troféu de Testes & TDD ([`tdd.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/tdd.md))
- **Priorize Testes de Integração:** O coração da suíte de testes são testes de integração reais contra o PostgreSQL 18 em container Docker. Não use mocks de banco (`sqlmock`).
- **Testes Unitários Enxutos:** Restritos a algoritmos puros sem I/O (ex.: cálculo SHA-256 de `Fingerprint`, normalização Unicode NFD de texto e validações de invariantes).
- **Ciclo TDD:** Red (escreva o teste e observe falhar) -> Green (implemente o mínimo para passar) -> Refactor (limpe e formate).
- **Edge Cases Obrigatórios:** Teste valores nulos, vazios, violação de constraints `UNIQUE`, ordenação determinística e paginação.

### 3.2 Boas Práticas de Backend Go ([`golang-best-practices.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/golang-best-practices.md))
- **Simplicidade:** Código óbvio e direto sem abstrações prematuras ou camadas Java-style (`controllers/`, `services/`, `models/`).
- **Pacotes por Domínio:** `internal/job`, `internal/database`, `internal/config`, `internal/logger`, `internal/http`.
- **Nomenclatura:** Curta, contextual e sem redundâncias (`job.Job`, `job.Repository`, `job.Status`). Preservar siglas: `ID`, `URL`, `UUID`, `HTTP`, `API`, `SQL`.
- **Erros:** Sempre explicitamente tratados e envelopados com `%w` (`fmt.Errorf("inserir vaga: %w", err)`). Erros sentinela tipados com `errors.Is`.
- **Context:** Sempre como primeiro argumento (`ctx context.Context`). Nunca guardar em structs nem passar `nil`.
- **Concorrência:** Sem vazamento de goroutines, canais controlados pelo produtor, testes com `-race`.

### 3.3 Banco de Dados, PostgreSQL 18 & sqlc ([`sqlc.guideliness.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/sqlc.guideliness.md))
- **SQL Direto:** Use `sqlc` com `sql_package: "pgx/v5"`. Sem ORMs.
- **PostgreSQL 18:** Chave primária nativa `id UUID PRIMARY KEY DEFAULT uuidv7()`.
- **Override no sqlc:** Mapeie `uuid` diretamente para `github.com/google/uuid.UUID`.
- **Proibido `SELECT *`:** Especifique colunas explicitamente.
- **Prevenção de N+1:** Proibido executar queries em loops em Go. Use `JOIN`, agregações em SQL ou `WHERE id = ANY($1)`.
- **Paginação Bounded e Determinística:** Toda listagem pública deve ter `LIMIT` defensivo e ordenação determinística (`ORDER BY published_at DESC NULLS LAST, id DESC`).
- **Operações Atômicas:** Prefira `ON CONFLICT (source, external_id) DO UPDATE ... RETURNING ...`.
- **Tradução de Erros:** Mapeie `pgx.ErrNoRows` para `job.ErrNotFound` ("vaga não encontrada").

### 3.4 Observabilidade de Primeira Classe ([`observability-guide.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/observability-guide.md))
- **`log/slog`:** Padrão do projeto. JSON em produção, texto legível em desenvolvimento.
- **Campos Estruturados:** Nunca use concatenação de strings para dados dinâmicos. Use atributos `slog.String`, `slog.Int`, etc.
- **Injeção de Logger:** Construtores recebem `*slog.Logger` injetado explicitamente (fallback para `slog.Default()`).
- **Rastreabilidade:** Propague `request_id` e contexto de correlação via `ctx`.

### 3.5 Frontend & Interface HTMX ([`frontend-guidelines.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/frontend-guidelines.md))
- **Identidade de Enfermagem:** Visual sóbrio e profissional de saúde (branco, verde hospitalar, azul suave). Sem layouts genéricos de IA ou dashboards desnecessários.
- **HTMX:** Renderização no servidor (SSR), atualizações parciais com fragments, mínimo JavaScript.
- **Acessibilidade & Mobile First:** Foco em leitura rápida e usabilidade em smartphones para profissionais de enfermagem.
- **Estados Visuais:** Loading indicators, cards vazios (*empty state*) e feedback de erro claro em português.

### 3.6 Fluxo Git & Política de Commits/Pushes ([`git-workflow.md`](file:///home/marcos/Projects/radar-enfermagem-rs/.agents/rules/git-workflow.md))
- **Autorização Formal Obrigatória:** É **terminantemente proibido** executar `git commit`, `git push`, abertura ou merge de PRs sem confirmação expressa do usuário.
- **Protocolo de Parada:** Ao concluir edições e testes, o agente deve exibir o `git status`, resumo das mudanças e sugerir a mensagem de commit, aguardando aprovação explícita antes de prosseguir.
- **Branches e Conventional Commits:** `main` estritamente protegida; desenvolvimento em `develop` ou feature branches; padrão Conventional Commits (`feat:`, `fix:`, `chore:`, etc.).

---

## 4. Checklist do Agente Gemini Antes de Concluir Tarefas

- [ ] Autorização formal e explícita do usuário obtida ANTES de qualquer `git commit` ou `git push`?
- [ ] Mensagens de erro e validações escritas em **Português (pt-BR)**?
- [ ] Testes de integração reais adicionados contra o PostgreSQL 18 (Troféu de Testes)?
- [ ] Queries sqlc sem `SELECT *`, livres de **N+1** e com ordenação determinística?
- [ ] Injeção de dependência de `*slog.Logger` e logs estruturados utilizados?
- [ ] `make test` passa com detecção de races (`-race`)?
- [ ] `cd apps/api && go vet ./... && gofmt -s -w .` executados?
- [ ] Atualização do checklist em [`docs/roadmap.md`](file:///home/marcos/Projects/radar-enfermagem-rs/docs/roadmap.md)?
