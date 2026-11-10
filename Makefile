.PHONY: help up down logs restart clean

help: ## Show this help message
	@echo "Quevia Development Management Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

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
