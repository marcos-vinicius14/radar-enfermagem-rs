-include .env
export

.PHONY: help test test-unit test-integration test-e2e test-cover lint up down logs migrate-up migrate-down sqlc collect collect-persist collect-json run build deploy

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

build: ## Compila o binário da API
	cd apps/api && go build -o bin/api ./cmd/api

deploy: up migrate-up ## Sobe o banco, aplica migrações e popula as vagas
	@echo "==> Executando coleta e persistência inicial..."
	cd apps/api && go run ./cmd/collector -collector=all -persist
	@echo "==> Deploy concluído com sucesso!"
