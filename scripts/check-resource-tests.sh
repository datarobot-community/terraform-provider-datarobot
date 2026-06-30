#!/usr/bin/env bash
# check-resource-tests.sh — Verify each resource/data-source has acceptance tests.
#
# For every pkg/provider/*_resource.go and *_data_source.go:
#   1. A corresponding *_test.go file must exist.
#   2. That file must contain at least one TestAcc* or TestIntegration* function.
#
# This is a static check — no API credentials required.
# Run it in CI as part of the build/lint stage to catch missing tests early.
#
# Exceptions:
#   Add source files to EXCEPTIONS below when their acceptance tests live inside
#   another resource's test file (e.g. a sub-resource sharing its parent's tests).
#   Document the reason so the exception can be revisited later.
set -euo pipefail

###############################################################################
# Known exceptions — resource files intentionally without a dedicated TestAcc*
# function in their own test file. Keep this list short.
###############################################################################
declare -A EXCEPTIONS
# covered by TestAccCustomModelFromLlmBlueprintResource and the LLM blueprint tests
EXCEPTIONS["pkg/provider/custom_model_from_vector_database_resource.go"]="covered by llm_blueprint_resource_test.go + vector_database_resource_test.go"

###############################################################################
# Check
###############################################################################
MISSING_FILE=()
MISSING_TESTS=()

for src in pkg/provider/*_resource.go pkg/provider/*_data_source.go; do
  [[ -f "$src" ]] || continue

  if [[ -n "${EXCEPTIONS[$src]+x}" ]]; then
    echo "  skip (exception): $src — ${EXCEPTIONS[$src]}"
    continue
  fi

  base=$(basename "$src" .go)
  test_file="pkg/provider/${base}_test.go"

  if [[ ! -f "$test_file" ]]; then
    MISSING_FILE+=("$src")
  elif ! grep -qE 'func Test(Acc|Integration)[A-Za-z0-9_]+' "$test_file"; then
    MISSING_TESTS+=("$test_file")
  fi
done

###############################################################################
# Report
###############################################################################
FAILED=false

if [[ ${#MISSING_FILE[@]} -gt 0 ]]; then
  echo ""
  echo "ERROR: No test file found for:"
  for f in "${MISSING_FILE[@]}"; do
    echo "  $f  →  expected: pkg/provider/$(basename "$f" .go)_test.go"
  done
  FAILED=true
fi

if [[ ${#MISSING_TESTS[@]} -gt 0 ]]; then
  echo ""
  echo "ERROR: Test file exists but contains no TestAcc* or TestIntegration* functions:"
  for f in "${MISSING_TESTS[@]}"; do
    echo "  $f"
  done
  echo ""
  echo "  Add at least one acceptance test or, if the resource is tested via another"
  echo "  resource's test file, add an entry to EXCEPTIONS in scripts/check-resource-tests.sh."
  FAILED=true
fi

if $FAILED; then
  exit 1
fi

TOTAL=$(ls pkg/provider/*_resource.go pkg/provider/*_data_source.go 2>/dev/null | wc -l | tr -d ' ')
SKIPPED=${#EXCEPTIONS[@]}
echo "OK: $((TOTAL - SKIPPED)) resource/data-source files checked, ${SKIPPED} skipped (exceptions)."
