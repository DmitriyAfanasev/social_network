#!/usr/bin/env bash

set -euo pipefail

if [[ "$#" -ne 6 ]]; then
	echo "usage: $0 PROJECT_NAME PORT MIGRATION ENV_NAME MODULE TEST_NAME" >&2
	exit 2
fi

project_name="$1"
port="$2"
migration="$3"
env_name="$4"
module="$5"
test_name="$6"
compose=(docker compose -p "$project_name" -f docker-compose.yaml)
export PG_PORT="$port"
database_url="postgres://admin:password@localhost:$port/database?sslmode=disable"

cleanup() {
	"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

"${compose[@]}" up -d pg
ready=false
for _ in $(seq 1 60); do
	if "${compose[@]}" exec -T pg pg_isready -h localhost -p 5432 -U admin -d database >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	echo "PostgreSQL did not become ready within 60 seconds" >&2
	exit 1
fi

# Небольшая задержка чтобы PostgreSQL успел создать unix socket после pg_isready
sleep 2

"${compose[@]}" exec -T pg psql -h localhost -p 5432 -U admin -d database -v ON_ERROR_STOP=1 < db/bootstrap.sql
goose -dir "services/$module/migrations" -table "$migration.goose_db_version" postgres "$database_url" up

test_output=$(env "$env_name=$database_url" \
	GOCACHE="${GOCACHE:-/tmp/general-project-go-test-cache}" \
	go -C "services/$module" test ./internal/adapters/postgres -run "^${test_name}$" -count=1 -v 2>&1) || {
	printf '%s\n' "$test_output"
	exit 1
}
printf '%s\n' "$test_output"
if printf '%s\n' "$test_output" | rg -q '\[no tests to run\]'; then
	echo "integration test $test_name did not execute" >&2
	exit 1
fi