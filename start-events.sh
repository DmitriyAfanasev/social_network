#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
"$SCRIPT_DIR/scripts/create-kafka-topics.sh"

exec faststream run backend.infra.messaging.faststream_app:app
