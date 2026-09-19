# Radar Enfermagem RS

Aplicação web para monitorar, centralizar e pesquisar vagas de emprego para Técnico em Enfermagem em Porto Alegre e região metropolitana.

---

## Arquitetura & Stack

- **Linguagem & Backend**: Go com Chi Router
- **Banco de Dados**: PostgreSQL 18-alpine (com suporte nativo a UUIDv7)
- **Persistência**: `pgx/v5` (pool de conexões) e `sqlc` (geração de queries tipadas)
- **Migrações**: `golang-migrate/v4`
- **Frontend**: Servido via backend com HTMX (em desenvolvimento)
- **Containerização**: Docker e Docker Compose

---

## Estrutura do Monorepo

```text
radar-enfermagem-rs/
├── apps/
│   ├── api/             # Backend Go (API, handlers, banco, scheduler)
│   │   ├── cmd/         # Entrypoints (api, migrate)
│   │   ├── internal/    # Pacotes internos (config, database, http, logger)
│   │   ├── migrations/  # Migrações SQL versionadas
│   │   └── queries/     # Queries SQL para sqlc
│   └── web/             # Frontend com templates HTMX e assets
├── deployments/         # Dockerfile e docker-compose
├── docs/                # Documentação técnica e roadmap
├── Makefile             # Comandos facilitadores
└── README.md
```

---

## Começando

### Pré-requisitos
- [Go](https://golang.org/) (versão 1.23+)
- [Docker](https://www.docker.com/) e Docker Compose
- [sqlc](https://sqlc.dev/) (para compilação de queries)

### Executando Localmente

1. **Configurar variáveis de ambiente**:
   ```bash
   cp .env.example .env
   ```

2. **Subir o banco de dados**:
   ```bash
   make up
   ```

3. **Executar as migrações**:
   ```bash
   make migrate-up
   ```

4. **Rodar os testes automatizados (TDD)**:
   ```bash
   make test
   # ou com cobertura:
   make test-cover
   ```

5. **Iniciar a API**:
   ```bash
   make run
   ```

6. **Verificar os endpoints**:
   - Healthcheck: `curl http://localhost:8080/health`
   - Readiness (com verificação de banco): `curl http://localhost:8080/ready`
