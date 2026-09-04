#!/usr/bin/env bash

set -euo pipefail

project_name="general-project-goose-smoke"
compose=(docker compose -p "$project_name" -f docker-compose.yaml)
smoke_port="55432"
export PG_PORT="$smoke_port"
database_url="postgres://admin:password@localhost:$smoke_port/database?sslmode=disable"

services=(
	"identity:identity"
	"profiles:profiles"
	"social:social"
	"content:content"
	"media:media"
	"messaging:messaging"
	"analytics:analytics"
	"admin:admin"
)

cleanup() {
	"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

"${compose[@]}" up -d pg
ready=false
for _ in $(seq 1 60); do
	if "${compose[@]}" exec -T pg pg_isready -U admin -d database >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	echo "PostgreSQL did not become ready within 60 seconds" >&2
	exit 1
fi

bootstrapped=false
for _ in $(seq 1 30); do
	if "${compose[@]}" exec -T pg psql -U admin -d database < db/bootstrap.sql; then
		bootstrapped=true
		break
	fi
	sleep 1
done
if [[ "$bootstrapped" != true ]]; then
	echo "PostgreSQL bootstrap failed within 30 seconds" >&2
	exit 1
fi

for migration in "${services[@]}"; do
	service="${migration%%:*}"
	schema="${migration##*:}"
	goose -dir "services/$service/migrations" -table "$schema.goose_db_version" postgres "$database_url" up
done

for ((index = ${#services[@]} - 1; index >= 0; index--)); do
	migration="${services[$index]}"
	service="${migration%%:*}"
	schema="${migration##*:}"
	goose -dir "services/$service/migrations" -table "$schema.goose_db_version" postgres "$database_url" down-to 0
done
