#!/usr/bin/env bash

set -uo pipefail

modules=(
  libs/platform
  services/analytics
  services/admin
  services/call-signaling
  services/content
  services/gateway
  services/identity
  services/media
  services/messaging
  services/profiles
  services/social
  services/video-worker
)

test_cache="${GOCACHE:-/tmp/general-project-go-test-cache}"
passed_modules=0
failed_modules=0
total_packages=0
passed_packages=0
no_test_packages=0

printf 'Go test suite (%d modules)\n' "${#modules[@]}"
printf 'Build cache: %s\n\n' "$test_cache"

for module in "${modules[@]}"; do
  log_file="$(mktemp /tmp/general-project-test.XXXXXX)"
  if GOCACHE="$test_cache" go -C "$module" test ./... >"$log_file" 2>&1; then
    module_packages="$(awk '/^(ok|\?)[[:space:]]/{count++} END {print count+0}' "$log_file")"
    module_no_tests="$(awk '/\[no test files\]/{count++} END {print count+0}' "$log_file")"
    module_passed=$((module_packages - module_no_tests))
    total_packages=$((total_packages + module_packages))
    passed_packages=$((passed_packages + module_passed))
    no_test_packages=$((no_test_packages + module_no_tests))
    passed_modules=$((passed_modules + 1))
    printf '✓ %-32s %d passed, %d without tests\n' "$module" "$module_passed" "$module_no_tests"
  else
    failed_modules=$((failed_modules + 1))
    printf '✗ %s\n' "$module"
    printf '%s\n' '  Failure output:'
    sed 's/^/    /' "$log_file"
  fi
  rm -f "$log_file"
done

printf '\nSummary: %d/%d modules passed; %d packages passed; %d without tests\n' \
  "$passed_modules" "${#modules[@]}" "$passed_packages" "$no_test_packages"

if (( failed_modules > 0 )); then
  printf 'Result: FAILED (%d module(s))\n' "$failed_modules"
  exit 1
fi

printf '%s\n' 'Result: PASSED'
