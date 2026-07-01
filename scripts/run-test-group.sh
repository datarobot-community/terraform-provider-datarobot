#!/usr/bin/env bash
# run-test-group.sh — Run or validate acceptance test groups.
#
# Groups are defined by file name prefixes. A prefix matches all
# pkg/provider/<prefix>*_test.go files, and their TestAcc*/TestIntegration*
# functions are extracted and passed as the -run regex to go test.
#
# Usage:
#   bash scripts/run-test-group.sh <group>   # run a group: fast|models|deployments|apps
#   bash scripts/run-test-group.sh --check   # verify every TestAcc*/TestIntegration* is covered
#
# Environment variables:
#   TEST_PARALLEL — go test -parallel value  (default: 6)
#   REPORT_FILE   — write JUnit XML here     (optional, requires go-junit-report in PATH)
#
# Adding a new resource:
#   1. Put its file name prefix in the right group in the GROUP_PREFIXES section below.
#   2. Run 'make testacc-check-groups' to confirm full coverage.
set -euo pipefail

TEST_PARALLEL="${TEST_PARALLEL:-6}"
REPORT_FILE="${REPORT_FILE:-}"

###############################################################################
# Group definitions
#
# Each entry is a space-separated list of file name prefixes.
# A prefix matches pkg/provider/<prefix>*_test.go via shell globbing.
# Use the start of the file name up to (but not including) the first unique
# differentiator — e.g. "custom_model" covers custom_model_resource_test.go,
# custom_model_llm_validation_resource_test.go, etc.
###############################################################################

# Group: fast — credentials, datasets, datastores, use cases, infra primitives
FAST_PREFIXES="api_token_credential app_oauth aws_credential azure_credential basic_credential batch_prediction_job data_source_resource data_store dataset_from google_cloud_service_account prediction_environment quota remote_repository use_case"
FAST_TIMEOUT="30m"

# Group: models — custom models, registered models, LLM blueprints, vector DB, playground
MODELS_PREFIXES="artifact custom_model llm_blueprint_resource playground registered_model vector_database"
MODELS_TIMEOUT="60m"

# Group: deployments — deployments, retraining policies, custom metrics, workloads
DEPLOYMENTS_PREFIXES="custom_metric deployment global_model llm_blueprint_deployment memory_space qa_application workload"
DEPLOYMENTS_TIMEOUT="60m"

# Group: apps — custom apps, application sources, notebooks, custom jobs, execution envs
APPS_PREFIXES="application_source custom_application custom_job execution_environment notebook notification_channel notification_policy user_mcp"
APPS_TIMEOUT="45m"

###############################################################################
# Helpers
###############################################################################

# Print TestAcc*/TestIntegration* names from files whose names start with any
# of the given prefixes. Outputs one name per line, deduplicated.
names_for_prefixes() {
  local prefixes=("$@")
  local seen_file
  seen_file=$(mktemp)
  for prefix in "${prefixes[@]}"; do
    for f in pkg/provider/${prefix}*_test.go; do
      [[ -f "$f" ]] || continue
      grep -ohE 'func Test(Acc|Integration)[A-Za-z0-9_]+' "$f" \
        | sed 's/func //' || true
    done
  done | sort -u > "$seen_file"
  cat "$seen_file"
  rm -f "$seen_file"
}

# Map a group name to its prefix list variable.
group_prefixes() {
  case "$1" in
    fast)        echo "$FAST_PREFIXES";;
    models)      echo "$MODELS_PREFIXES";;
    deployments) echo "$DEPLOYMENTS_PREFIXES";;
    apps)        echo "$APPS_PREFIXES";;
    *)           echo "Unknown group: $1. Valid: fast|models|deployments|apps" >&2; return 1;;
  esac
}

group_timeout() {
  case "$1" in
    fast)        echo "$FAST_TIMEOUT";;
    models)      echo "$MODELS_TIMEOUT";;
    deployments) echo "$DEPLOYMENTS_TIMEOUT";;
    apps)        echo "$APPS_TIMEOUT";;
  esac
}

###############################################################################
# --check mode
###############################################################################
if [[ "${1:-}" == "--check" ]]; then
  echo "==> Checking test group coverage..."
  ok=true

  # All TestAcc*/TestIntegration* functions that exist in the codebase
  ALL=$(grep -rh "^func Test\(Acc\|Integration\)" pkg/provider/*_test.go \
        | sed 's/func \(Test[A-Za-z0-9_]*\)(.*$/\1/' | sort -u)

  # All functions covered by any group
  COVERED_FILE=$(mktemp)
  for group in fast models deployments apps; do
    read -ra prefixes <<< "$(group_prefixes "$group")"

    # Warn about prefixes that don't match any file
    for prefix in "${prefixes[@]}"; do
      matched=false
      for f in pkg/provider/${prefix}*_test.go; do
        [[ -f "$f" ]] && matched=true && break
      done
      if ! $matched; then
        echo "  WARNING [group=$group]: prefix '$prefix' matches no files — typo?" >&2
      fi
    done

    names_for_prefixes "${prefixes[@]}" >> "$COVERED_FILE"
  done
  COVERED=$(sort -u < "$COVERED_FILE")
  rm -f "$COVERED_FILE"

  MISSING=$(comm -23 <(echo "$ALL") <(echo "$COVERED"))
  if [[ -n "$MISSING" ]]; then
    echo ""
    echo "ERROR: These tests are not assigned to any group:"
    echo "$MISSING" | sed 's/^/  /'
    echo ""
    echo "Add the test file's prefix to the matching group in scripts/run-test-group.sh."
    ok=false
  fi

  if $ok; then
    echo "OK: all $(echo "$ALL" | wc -l | tr -d ' ') TestAcc*/TestIntegration* functions are covered."
    exit 0
  else
    exit 1
  fi
fi

###############################################################################
# Run mode
###############################################################################
GROUP="${1:-}"
if [[ -z "$GROUP" ]]; then
  echo "Usage: $0 <group> | --check" >&2
  echo "Groups: fast | models | deployments | apps" >&2
  exit 1
fi

PREFIXES_STR="$(group_prefixes "$GROUP")"
TIMEOUT="$(group_timeout "$GROUP")"
read -ra PREFIXES <<< "$PREFIXES_STR"

mapfile -t TEST_NAMES < <(names_for_prefixes "${PREFIXES[@]}")

if [[ ${#TEST_NAMES[@]} -eq 0 ]]; then
  echo "ERROR: no TestAcc*/TestIntegration* functions found for group '$GROUP'" >&2
  exit 1
fi

REGEX=$(IFS='|'; echo "${TEST_NAMES[*]}")
echo "==> Group '$GROUP': ${#TEST_NAMES[@]} tests, timeout ${TIMEOUT}, parallel ${TEST_PARALLEL}"
echo ""

set +e
TF_ACC=1 go test ./pkg/provider/ -v -timeout "${TIMEOUT}" -parallel "${TEST_PARALLEL}" \
  -run "${REGEX}" 2>&1 | tee report.out
EXIT_CODE=${PIPESTATUS[0]}
set -e

if [[ -n "$REPORT_FILE" ]] && command -v go-junit-report &>/dev/null; then
  mkdir -p "$(dirname "$REPORT_FILE")"
  cat report.out | go-junit-report -set-exit-code > "$REPORT_FILE" 2>/dev/null || true
  echo "==> JUnit report written to ${REPORT_FILE}"
fi

rm -f report.out
exit "$EXIT_CODE"
