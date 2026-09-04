#!/usr/bin/env bash

set -euo pipefail

mapfile -t postgres_volumes < <(
	docker volume ls --quiet --filter label=com.docker.compose.volume=postgres_data
)

if ((${#postgres_volumes[@]} == 0)); then
	echo "PostgreSQL volume postgres_data was not found; nothing to reset."
	exit 0
fi

if ((${#postgres_volumes[@]} != 1)); then
	echo "Expected exactly one Compose postgres_data volume, found: ${postgres_volumes[*]}" >&2
	exit 1
fi

docker compose rm --force --stop pg
docker volume rm "${postgres_volumes[0]}"
echo "Removed local PostgreSQL volume ${postgres_volumes[0]}"
