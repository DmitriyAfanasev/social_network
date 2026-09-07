#!/usr/bin/env bash
set -euo pipefail

# Проверяем новую миграцию отдельно от несовместимого старого полного отката profiles.
gallery_project="general-project-gallery-smoke"
export PG_PORT=55433
gallery_compose=(docker compose -p "$gallery_project" -f docker-compose.yaml)
gallery_dsn="postgres://admin:password@localhost:55433/database?sslmode=disable"
cleanup() { "${gallery_compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true; }
trap cleanup EXIT
"${gallery_compose[@]}" up -d pg
ready=false
for _ in $(seq 1 60); do
    if "${gallery_compose[@]}" exec -T pg psql -U admin -d database -v ON_ERROR_STOP=1 < db/bootstrap.sql; then
        ready=true
        break
    fi
    sleep 1
done
[[ "$ready" == true ]]
goose -dir services/profiles/migrations -table profiles.goose_db_version postgres "$gallery_dsn" up
GALLERY_TEST_DATABASE_URL="$gallery_dsn" GOCACHE=/tmp/general-project-go-cache go test ./services/profiles/internal/adapters/postgres -run TestGalleryRepositoryIntegration -count=1 -v
goose -dir services/profiles/migrations -table profiles.goose_db_version postgres "$gallery_dsn" down
goose -dir services/profiles/migrations -table profiles.goose_db_version postgres "$gallery_dsn" up
