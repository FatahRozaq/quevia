.PHONY: help up down logs restart clean test-backend build-backend build-frontend

help: ## Show this help message
	@echo "Quevia Development Management Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

up: ## Start PostgreSQL, Redis, and Mailpit in background
	docker compose up -d

down: ## Stop all development docker services
	docker compose down

logs: ## Follow docker services logs
	docker compose logs -f

restart: ## Restart development docker services
	docker compose restart

clean: ## Stop services and delete volumes (warning: resets database!)
	docker compose down -v

test-backend: ## Run backend unit tests
	cd backend && go test -v ./...

build-backend: ## Build Go API and Worker binaries
	cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker

build-frontend: ## Build Next.js frontend production bundle
	cd frontend && npm run build
