# Make sure install migrate cli tool: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

SHELL := bash

# Safely check if .env exists, include it, and export its variables
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Migrate commands

MIGRATE_POSTGRES = migrate -path base/migrations/postgres -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable"

# Ignore the file system. These are action words, not files.
.PHONY: migrate-post-up migrate-post-down migrate-post-version migrate-post-new

migrate-post-up:
	$(MIGRATE_POSTGRES) up 1

migrate-post-down:
	$(MIGRATE_POSTGRES) down 1

migrate-post-down-force:
	$(MIGRATE_POSTGRES) force 1
	$(MIGRATE_POSTGRES) down 1

migrate-post-version:
	$(MIGRATE_POSTGRES) version

migrate-post-new:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir internal/modules/core/migrations -seq $${name}


# Docker project commands
DB_SERVICES = postgres mongodb redis
MONITORING_SERVICES = prometheus prometheus-config grafana loki redis-exporter postgres-exporter otel-collector tempo
QUEUE_SERVICES = rabbitmq

up-db:
	docker compose rm -f prometheus-config
	docker compose up -d $(DB_SERVICES) 

down-db:
	docker compose down $(DB_SERVICES)

up-mon:
	docker compose rm -f prometheus-config
	docker compose --profile monitoring up -d $(MONITORING_SERVICES)

down-mon:
	docker compose down --remove-orphans $(MONITORING_SERVICES) prometheus-config


up-queue:
	docker compose up -d $(QUEUE_SERVICES)

down-queue:
	docker compose down $(QUEUE_SERVICES)

up-sys: up-db up-mon up-queue

down-sys: down-queue down-mon down-db


cont-urls:
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:9300"


# REDIS Commands
.PHONY: migrate-redis-up migrate-redis-down

MIGRATE_REDIS_DIR = base/migrations/redis

# Applying migrations up in the increasing order
migrate-redis-up-all:
	@echo "Applying Redis migrations (UP)..."
	@for file in $$(ls $(MIGRATE_REDIS_DIR)/*.up.sh 2>/dev/null | sort); do \
		echo "=> Running $$file"; \
		bash "$$file" || exit 1; \
	done

# Applying migrations down in the decreasing order
migrate-redis-down-all:
	@echo "Rolling back Redis migrations (DOWN)..."
	@for file in $$(ls $(MIGRATE_REDIS_DIR)/*.down.sh 2>/dev/null | sort -r); do \
		echo "=> Reverting $$file"; \
		bash "$$file" || exit 1; \
	done

migrate-redis-single:
	@if [ -z "$(FILE)" ]; then echo "Error: FILE argument is required."; exit 1; fi
	@echo "Reverting single file: $(FILE)..."
	@export $$(grep -v '^#' .env | xargs) && bash $(FILE)


# GRPC proto builds
build-proto:
	protoc --go_out=paths=source_relative:. --go_grpc_out=paths=source_relative:. $(FILE)

# Air commands
air-inventory:
	air -c .air.inventory.toml

air-booking:
	air -c .air.booking.toml

# Gateway
up-gateway:
	docker compose up -d gateway-config gateway

down-gateway:
	docker compose down gateway-config gateway

restart-gateway:  # when yaml config change then it is required
	docker compose restart gateway-config gateway
