-include .env
export

.PHONY: help test test-cover up down logs migrate-up migrate-down sqlc run build

help: ## Exibe os comandos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

test: ## Executa testes unitários com detecção de race conditions
	cd apps/api && go test -race -v ./...

test-cover: ## Executa testes e exibe relatório de cobertura
	cd apps/api && go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

up: ## Sobe os containers no docker compose
	docker compose --env-file .env -f deployments/compose.yaml up -d db

down: ## Para os containers do docker compose
	docker compose --env-file .env -f deployments/compose.yaml down

logs: ## Exibe logs dos containers
	docker compose --env-file .env -f deployments/compose.yaml logs -f

migrate-up: ## Executa migrações pendentes no banco
	cd apps/api && go run ./cmd/migrate up

migrate-down: ## Executa rollback da última migração
	cd apps/api && go run ./cmd/migrate down

sqlc: ## Gera código Go a partir das queries SQL
	cd apps/api && sqlc generate

run: ## Executa a API localmente
	cd apps/api && go run ./cmd/api

build: ## Compila o binário da API
	cd apps/api && go build -o bin/api ./cmd/api
