#!/usr/bin/env bash

set -euo pipefail

project_name="general-project-e2e"
compose=(docker compose -p "$project_name" -f docker-compose.yaml -f docker-compose.e2e.yaml)
port="55440"
database_url="postgres://admin:password@localhost:$port/database?sslmode=disable"

cleanup() {
	status=$?
	if [[ "$status" -ne 0 ]]; then
		"${compose[@]}" logs --no-color --tail=80 identity gateway messaging profiles social content media >&2 || true
	fi
	"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
	return "$status"
}
trap cleanup EXIT

"${compose[@]}" up -d pg redis kafka mailpit minio
ready=false
for _ in $(seq 1 90); do
	if "${compose[@]}" exec -T pg pg_isready -U admin -d database >/dev/null 2>&1 && \
		"${compose[@]}" exec -T redis redis-cli ping >/dev/null 2>&1 && \
		curl --fail --silent http://127.0.0.1:58025/api/v1/info >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	echo "E2E infrastructure did not become ready within 90 seconds" >&2
	exit 1
fi

"${compose[@]}" exec -T pg psql -U admin -d database -v ON_ERROR_STOP=1 < db/bootstrap.sql
for migration in identity profiles social content media messaging analytics admin; do
	goose -dir "services/$migration/migrations" -table "$migration.goose_db_version" postgres "$database_url" up
done

"${compose[@]}" up -d --build identity profiles social content media messaging analytics admin call-signaling gateway

ready=false
for _ in $(seq 1 120); do
	if curl --fail --silent http://127.0.0.1:58000/healthz >/dev/null 2>&1 && \
		curl --fail --silent http://127.0.0.1:58101/healthz >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done
if [[ "$ready" != true ]]; then
	echo "E2E services did not become ready within 120 seconds" >&2
	"${compose[@]}" ps
	exit 1
fi

run_e2e_test() {
	local test_name="$1"
	E2E_BASE_URL="http://127.0.0.1:58000" \
		E2E_MAILPIT_URL="http://127.0.0.1:58025" \
		GOCACHE="${GOCACHE:-/tmp/general-project-go-test-cache}" \
		go -C services/gateway test ./e2e -run "^${test_name}$" -count=1 -v
}

if [[ "${E2E_TEST_SUITE:-}" == "all" ]]; then
	for test_name in \
		TestAuthLifecycle \
		TestMessagingLifecycle \
		TestCallSignalingLifecycle \
		TestContentLifecycle \
		TestGatewayRoutingAndErrorEnvelope \
		TestSocialFriendRequestAndPrivacy; do
		# Each scenario owns its users; clear only ephemeral Redis state so the
		# intentional registration rate limit does not couple independent tests.
		"${compose[@]}" exec -T redis redis-cli FLUSHDB >/dev/null
		run_e2e_test "$test_name"
	done
else
	run_e2e_test "${E2E_TEST_REGEX:-TestAuthLifecycle}"
fi
