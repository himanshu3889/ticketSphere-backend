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
	$(MIGRATE_POSTGRES) up

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

up-db:
	docker compose up -d postgres mongodb redis

down-db:
	docker compose down postgres mongodb redis

up-mon:
	docker compose --profile monitoring up -d prometheus grafana loki

# kafka-exporter redis-exporter postgres-exporter mongodb-exporter
down-mon:
	docker compose down prometheus grafana loki

up-sys:
	docker compose up -d postgres mongodb redis kafka debezium

down-sys:
	docker compose down


cont-urls:
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:9300"


# REDIS Commands
.PHONY: migrate-redis-up migrate-redis-down

MIGRATE_REDIS_DIR = base/migrations/redis

# Applying migrations up in the increasing order
migrate-redis-up:
	@echo "Applying Redis migrations (UP)..."
	@for file in $$(ls $(MIGRATE_REDIS_DIR)/*.up.sh 2>/dev/null | sort); do \
		echo "=> Running $$file"; \
		bash "$$file" || exit 1; \
	done

# Applying migrations down in the decreasing order
migrate-redis-down:
	@echo "Rolling back Redis migrations (DOWN)..."
	@for file in $$(ls $(MIGRATE_REDIS_DIR)/*.down.sh 2>/dev/null | sort -r); do \
		echo "=> Reverting $$file"; \
		bash "$$file" || exit 1; \
	done


# KAKFA Commands
KAFKA_CONTAINER ?= ticketsphere-kafka
BOOTSTRAP_SERVER ?= localhost:9092

KAFKA_GROUP_CMD = docker exec -it $(KAFKA_CONTAINER) \
	/opt/kafka/bin/kafka-consumer-groups.sh \
	--bootstrap-server $(BOOTSTRAP_SERVER)

KAFKA_CONSUMER_CMD = docker exec -it $(KAFKA_CONTAINER) \
	/opt/kafka/bin/kafka-console-consumer.sh \
	--bootstrap-server $(BOOTSTRAP_SERVER)

KAFKA_TOPIC_CMD = docker exec -it $(KAFKA_CONTAINER) \
	/opt/kafka/bin/kafka-topics.sh \
	--bootstrap-server $(BOOTSTRAP_SERVER)

kafka-grp:
	$(KAFKA_GROUP_CMD) $(args)

kafka-grp-describe:
	$(KAFKA_GROUP_CMD) --describe --group $(group)

kafka-topics:
	$(KAFKA_TOPIC_CMD) $(args)

kafka-consumer:
	$(KAFKA_CONSUMER_CMD) $(args)

# GRPC proto builds
build-proto:
	protoc --go_out=paths=source_relative:. --go_grpc_out=paths=source_relative:. $(FILE)

air-inventory:
	air -c .air.inventory.toml

air-booking:
	air -c .air.booking.toml

# DEBEZIUM CONNECTOR

deb-status:  # windows curl.exe  ; mac/linux: curl
	curl.exe -s http://localhost:8083/connectors/ticketsphere-connector/status 


# Gateway
up-gateway:
	docker compose up -d gateway-config gateway

down-gateway:
	docker compose down gateway-config gateway

restart-gateway:  # when yaml config change then it is required
	docker compose restart gateway-config gateway
