#!/usr/bin/env bash
set -e

if [ -f .env.local ]; then
  export $(grep -v '^#' .env.local | xargs)
fi

CLICKHOUSE_HOST="${CLICKHOUSE_HOST:-localhost}"
CLICKHOUSE_HTTP_PORT="${CLICKHOUSE_HTTP_PORT:-8123}"
CLICKHOUSE_DATABASE="${CLICKHOUSE_DATABASE:-energosbyt}"

CH_URL="http://${CLICKHOUSE_HOST}:${CLICKHOUSE_HTTP_PORT}?database=${CLICKHOUSE_DATABASE}"

echo "Applying Goose migrations..."
if command -v goose &> /dev/null; then
  goose -dir migrations clickhouse "${CH_URL}" up
else
  echo "goose binary not found locally, executing migration SQL via HTTP query..."
  curl -s -f -X POST "${CH_URL}" --data-binary "@migrations/00001_create_energosbyt_tables.sql" || true
fi

echo "Loading CSV files into ClickHouse..."

curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+dim_realty+FORMAT+CSV" --data-binary "@data/dim_realty.csv"
curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+dim_gauges+FORMAT+CSV" --data-binary "@data/dim_gauges.csv"
curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+dim_tariffs+FORMAT+CSV" --data-binary "@data/dim_tariffs.csv"
curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+dim_users+FORMAT+CSV" --data-binary "@data/dim_users.csv"
curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+fact_gauge_bindings+FORMAT+CSV" --data-binary "@data/fact_gauge_bindings.csv"
curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+fact_meter_readings+FORMAT+CSV" --data-binary "@data/fact_meter_readings.csv"
curl -s -f -X POST "${CH_URL}&query=INSERT+INTO+fact_invoice+FORMAT+CSV" --data-binary "@data/fact_invoice.csv"

echo "Data loading completed successfully!"
