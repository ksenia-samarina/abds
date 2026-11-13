.PHONY: run rund services stop clean gen load setup teardown
.DEFAULT_GOAL = run

# Runs all services in detached mode.
run:
	docker compose --env-file .env.local up -d --build

# Runs all services without detached mode (for debugging).
rund:
	docker compose --env-file .env.local up --build

# Shows all service statuses.
services:
	docker compose ps

# Stops all running services.
stop:
	docker compose down

# Cleans up all resources including volumes.
clean:
	docker compose down -v

# Generates dataset CSV files using Go binary generator
gen:
	go run ./cmd/generator/main.go

# Loads CSV datasets into ClickHouse
load:
	chmod +x ./scripts/load.sh && ./scripts/load.sh

# Complete setup: generates datasets and loads them into ClickHouse
setup: gen load

# Drops all tables in the database
teardown:
	@if [ -f .env.local ]; then export $$(grep -v '^#' .env.local | xargs); fi; \
	curl -s -X POST "http://$${CLICKHOUSE_HOST:-localhost}:$${CLICKHOUSE_HTTP_PORT:-8123}?database=$${CLICKHOUSE_DATABASE:-energosbyt}&query=DROP+DATABASE+IF+EXISTS+$${CLICKHOUSE_DATABASE:-energosbyt}" ; \
	curl -s -X POST "http://$${CLICKHOUSE_HOST:-localhost}:$${CLICKHOUSE_HTTP_PORT:-8123}?query=CREATE+DATABASE+IF+NOT+EXISTS+$${CLICKHOUSE_DATABASE:-energosbyt}"
