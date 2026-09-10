#!/usr/bin/env bash
exec bash scripts/run-postgres-integration.sh general-project-messaging-integration 55438 messaging MESSAGING_TEST_DATABASE_URL messaging TestPostgresMessagingRepository
