-include .env
export

.PHONY: help test test-unit test-integration test-e2e test-cover lint up down logs migrate-up migrate-down sqlc collect collect-persist collect-json run build deploy release dev

help: ## Exibe os comandos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

test: ## Executa todos os testes com detecção de race conditions (sequencial entre pacotes)
	cd apps/api && go test -race -p 1 -v ./...
	cd apps/web && go test -race -v ./...

test-unit: ## Executa apenas testes unitários com detecção de race conditions
	cd apps/api && go test -short -race -v ./...
	cd apps/web && go test -race -v ./...

test-integration: ## Executa testes de integração reais contra o PostgreSQL
	cd apps/api && go test -race -p 1 -v ./internal/database/... ./internal/collector/...

test-e2e: ## Executa testes E2E reais contra todos os portais externos monitorados
	cd apps/api && go test -v -tags=e2e ./internal/collector/...

test-cover: ## Executa testes e exibe relatório de cobertura
	cd apps/api && go test -race -p 1 -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

lint: ## Executa validação de formato e análise estática (go vet e gofmt)
	cd apps/api && go vet ./... && test -z "$$(gofmt -s -l .)"
	cd apps/web && go vet ./... && test -z "$$(gofmt -s -l .)"

up: ## Sobe os containers no docker compose
	docker compose --env-file .env -f deployments/compose.local.yaml up -d db

down: ## Para os containers do docker compose
	docker compose --env-file .env -f deployments/compose.local.yaml down

logs: ## Exibe logs dos containers
	docker compose --env-file .env -f deployments/compose.local.yaml logs -f

migrate-up: ## Executa migrações pendentes no banco
	cd apps/api && go run ./cmd/migrate up

migrate-down: ## Executa rollback da última migração
	cd apps/api && go run ./cmd/migrate down

sqlc: ## Gera código Go a partir das queries SQL
	cd apps/api && sqlc generate

collect: ## Executa a coleta ao vivo no terminal (dry-run com visualização formatada)
	cd apps/api && go run ./cmd/collector -query="enfermagem"

collect-persist: ## Executa a coleta de todos os hospitais e persiste no PostgreSQL
	cd apps/api && go run ./cmd/collector -collector=all -persist

collect-json: ## Executa a coleta ao vivo no terminal e exibe em formato JSON
	cd apps/api && go run ./cmd/collector -query="enfermagem" -json

run: ## Executa a API localmente
	cd apps/api && go run ./cmd/api

dev: up ## Sobe o banco de dados, executa migrações pendentes e inicia a API com Frontend integrado
	@echo "==> Aguardando banco de dados PostgreSQL estar pronto..."
	@until docker compose --env-file .env -f deployments/compose.local.yaml exec -T db pg_isready -U $${DB_USER:-postgres} -d $${DB_NAME:-radar_enfermagem} >/dev/null 2>&1; do sleep 1; done
	@echo "==> Executando migrações pendentes..."
	$(MAKE) migrate-up
	@echo "==> Iniciando aplicação (API + Frontend HTMX) em http://localhost:$${PORT:-8080}..."
	$(MAKE) run

build: ## Compila o binário da API
	cd apps/api && go build -o bin/api ./cmd/api

deploy: up migrate-up ## Sobe o banco, aplica migrações e popula as vagas
	@echo "==> Executando coleta e persistência inicial..."
	cd apps/api && go run ./cmd/collector -collector=all -persist
	@echo "==> Deploy concluído com sucesso!"

release: ## Cria tag SemVer e publica GitHub Release (uso: make release VERSION=v1.0.0 [TITLE="..."])
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Erro: informe a versão. Exemplo: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if ! echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$$'; then \
		echo "❌ Erro: formato de versão inválido '$(VERSION)'. Use o padrão SemVer (ex: v1.0.0, v1.0.1-rc.1)"; \
		exit 1; \
	fi
	@current_branch=$$(git branch --show-current); \
	if [ "$$current_branch" != "main" ]; then \
		echo "⚠️  Aviso: você está na branch '$$current_branch'. Releases para produção devem preferencialmente ser publicadas a partir da branch 'main'."; \
	fi
	@echo "🔍 Executando validação de linter e testes antes de gerar a release..."
	$(MAKE) lint
	$(MAKE) test
	@echo "🚀 Criando tag $(VERSION) e publicando GitHub Release..."
	gh release create $(VERSION) --title "$(if $(TITLE),$(TITLE),$(VERSION))" --generate-notes
	@echo "✅ Release $(VERSION) publicada com sucesso! O workflow de deploy foi disparado no GitHub Actions."
