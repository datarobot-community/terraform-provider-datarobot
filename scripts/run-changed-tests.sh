#!/usr/bin/env bash
#
# run-changed-tests.sh — Selective acceptance test runner for Harness CI.
#
# Diffs the current branch against BASE_REF, identifies changed files in
# pkg/provider/ and internal/client/, and runs only the relevant TestAcc*
# acceptance tests. Produces a JUnit XML report compatible with Harness.
#
# Environment variables:
#   BASE_REF        — target branch to diff against (required, e.g. "main")
#   REPORT_FILE     — JUnit XML output path    (default: /harness/report.xml)
#   TEST_TIMEOUT    — go test timeout           (default: 60m)
#   TEST_PARALLEL   — go test parallelism       (default: 4)
#
# Usage:
#   BASE_REF=main bash scripts/run-changed-tests.sh
#
set -euo pipefail

###############################################################################
# Configuration
###############################################################################
BASE_REF="${BASE_REF:-main}"
REPORT_FILE="${REPORT_FILE:-/harness/report.xml}"
TEST_TIMEOUT="${TEST_TIMEOUT:-60m}"
TEST_PARALLEL="${TEST_PARALLEL:-4}"

# Shared helper files in pkg/provider/ that are used across all resources.
# Any change to these triggers the full test suite.
SHARED_HELPERS=(
  "provider.go"
  "models.go"
  "utils.go"
  "validators.go"
  "path_utils.go"
  "version_label.go"
  "user_mcp_common.go"
  # Shared test infrastructure: init(), testAccProtoV6ProviderFactories,
  # testAccPreCheck, env config variables — used by every acceptance test.
  "provider_test.go"
  "test_env_config_test.go"
)

###############################################################################
# Helpers
###############################################################################

# Write an empty passing JUnit report so Harness doesn't fail on missing file.
write_empty_report() {
  local dir
  dir="$(dirname "$REPORT_FILE")"
  if [[ -d "$dir" ]] || mkdir -p "$dir" 2>/dev/null; then
    cat > "$REPORT_FILE" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="no-tests" tests="0" failures="0" errors="0" time="0">
  </testsuite>
</testsuites>
XML
    echo "Wrote empty JUnit report to ${REPORT_FILE}"
  else
    echo "Warning: could not write report to ${REPORT_FILE} (directory does not exist)"
  fi
}

# Extract TestAcc* function names from a Go test file.
# Outputs one function name per line.
extract_test_names() {
  local file="$1"
  if [[ -f "$file" ]]; then
    grep -ohE 'func TestAcc[A-Za-z0-9_]+' "$file" | sed 's/func //' || true
  fi
}

# Map a resource/data-source source file to its corresponding test file.
# e.g. use_case_resource.go → use_case_resource_test.go
source_to_test_file() {
  local src="$1"
  local base
  base="$(basename "$src" .go)"
  echo "pkg/provider/${base}_test.go"
}

###############################################################################
# 1. Fetch target branch & compute diff
###############################################################################
echo "==> Fetching origin/${BASE_REF}..."
git fetch origin "$BASE_REF" --depth=1 2>/dev/null || git fetch origin "$BASE_REF" || true

echo "==> Computing diff against origin/${BASE_REF}..."
CHANGED_FILES=$(git diff --name-only --diff-filter=d "origin/${BASE_REF}...HEAD" 2>/dev/null || \
                git diff --name-only --diff-filter=d "origin/${BASE_REF}" 2>/dev/null || true)

if [[ -z "$CHANGED_FILES" ]]; then
  echo "No changed files detected."
  write_empty_report
  exit 0
fi

echo "Changed files:"
echo "$CHANGED_FILES" | sed 's/^/  /'

###############################################################################
# 2. Categorize changed files
###############################################################################
RUN_ALL=false
TEST_FILES_TO_SCAN=()

while IFS= read -r file; do
  # Skip non-Go files
  [[ "$file" != *.go ]] && continue

  # Bucket A: internal/client/ changes → run all
  if [[ "$file" == internal/client/* ]]; then
    echo "  [client change] $file → will run ALL tests"
    RUN_ALL=true
    continue
  fi

  # Only process pkg/provider/ files from here on
  [[ "$file" != pkg/provider/* ]] && continue

  filename="$(basename "$file")"

  # Bucket D: shared helper changes → run all
  for helper in "${SHARED_HELPERS[@]}"; do
    if [[ "$filename" == "$helper" ]]; then
      echo "  [shared helper] $file → will run ALL tests"
      RUN_ALL=true
      continue 2
    fi
  done

  # Bucket C: test file changes → extract TestAcc* directly
  if [[ "$filename" == *_test.go ]]; then
    echo "  [test file] $file"
    TEST_FILES_TO_SCAN+=("$file")
    continue
  fi

  # Bucket B: resource/data-source source files → map to test file
  if [[ "$filename" == *_resource.go ]] || [[ "$filename" == *_data_source.go ]]; then
    test_file="$(source_to_test_file "$file")"
    echo "  [source file] $file → $test_file"
    TEST_FILES_TO_SCAN+=("$test_file")
    continue
  fi

  # Any other .go file in pkg/provider/ we don't recognize → run all to be safe
  echo "  [unknown provider file] $file → will run ALL tests"
  RUN_ALL=true

done <<< "$CHANGED_FILES"

###############################################################################
# 3. Build test regex
###############################################################################
if $RUN_ALL; then
  echo ""
  echo "==> Running ALL acceptance tests (shared code changed)"
  TEST_RUN_ARG=""
elif [[ ${#TEST_FILES_TO_SCAN[@]} -eq 0 ]]; then
  echo ""
  echo "==> No relevant changes detected in pkg/provider/ or internal/client/. Skipping tests."
  write_empty_report
  exit 0
else
  # Collect unique TestAcc* function names from all identified test files
  declare -A SEEN_TESTS
  TEST_NAMES=()

  for tf in "${TEST_FILES_TO_SCAN[@]}"; do
    while IFS= read -r name; do
      [[ -z "$name" ]] && continue
      if [[ -z "${SEEN_TESTS[$name]+x}" ]]; then
        SEEN_TESTS[$name]=1
        TEST_NAMES+=("$name")
      fi
    done < <(extract_test_names "$tf")
  done

  if [[ ${#TEST_NAMES[@]} -eq 0 ]]; then
    echo ""
    echo "==> Changed test files contain no TestAcc* functions. Skipping tests."
    write_empty_report
    exit 0
  fi

  # Build regex: TestAccFoo|TestAccBar|...
  REGEX=$(IFS='|'; echo "${TEST_NAMES[*]}")
  TEST_RUN_ARG="-run ${REGEX}"
  echo ""
  echo "==> Running selective tests: ${REGEX}"
fi

###############################################################################
# 4. Run tests
###############################################################################
echo "==> Executing: TF_ACC=1 go test -v -cover ${TEST_RUN_ARG} -timeout ${TEST_TIMEOUT} -parallel ${TEST_PARALLEL} ./pkg/provider/"
echo ""

set +e
# shellcheck disable=SC2086
TF_ACC=1 go test -v -cover ${TEST_RUN_ARG} -timeout "${TEST_TIMEOUT}" -parallel "${TEST_PARALLEL}" ./pkg/provider/ 2>&1 | tee report.out
TEST_EXIT_CODE=${PIPESTATUS[0]}
set -e

###############################################################################
# 5. Generate JUnit report
###############################################################################
if command -v go-junit-report &>/dev/null; then
  cat report.out | go-junit-report -set-exit-code > "$REPORT_FILE" 2>/dev/null || true
  echo "==> JUnit report written to ${REPORT_FILE}"
else
  echo "Warning: go-junit-report not found, skipping JUnit report generation"
fi

rm -f report.out
exit "$TEST_EXIT_CODE"
