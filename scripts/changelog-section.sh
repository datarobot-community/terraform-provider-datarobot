#!/usr/bin/env bash
# changelog-section.sh — Print one version's section from CHANGELOG.md.
#
# Used by .github/workflows/release.yml to turn the tagged version's changelog
# entry into the GitHub release body and the Slack release message.
#
# Usage:
#   bash scripts/changelog-section.sh <version-or-tag> [changelog-path]
#
# Examples:
#   bash scripts/changelog-section.sh v0.10.45      # tag form, leading v stripped
#   bash scripts/changelog-section.sh 0.10.43       # bare version form
#
# CHANGELOG.md headings are '## [X.Y.Z] - YYYY-MM-DD' (no 'v' prefix) while git
# tags are 'vX.Y.Z', so the leading 'v' is stripped before matching. A section
# runs from its heading to the next line starting with '## ' (or EOF); only the
# body is printed, with surrounding blank lines trimmed.
#
# Exits non-zero if the version has no heading or its section is empty.
set -euo pipefail

usage() {
  echo "Usage: bash scripts/changelog-section.sh <version-or-tag> [changelog-path]" >&2
}

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  usage
  exit 2
fi

version="${1#v}"
changelog="${2:-CHANGELOG.md}"

if [ -z "$version" ]; then
  echo "changelog-section.sh: empty version argument" >&2
  usage
  exit 2
fi

if [ ! -f "$changelog" ]; then
  echo "changelog-section.sh: no such file: $changelog" >&2
  exit 1
fi

# Each '## ' heading is reduced to its bare version token and compared for
# equality, so '0.10.4' never matches '## [0.10.45]'. Both the current
# '## [X.Y.Z] - date' form and the older unbracketed '## X.Y.Z' form (see the
# 0.0.x entries at the end of the file) are recognized.
#
# Blank lines are buffered and only flushed when a non-blank line follows, which
# trims leading and trailing blank lines from the body.
section="$(
  awk -v ver="$version" '
    /^## / {
      heading = substr($0, 4)
      sub(/^\[/, "", heading)
      sub(/\].*$/, "", heading)
      sub(/[[:space:]].*$/, "", heading)
      if (heading == ver) { in_section = 1; next }
      if (in_section) exit
      next
    }
    in_section {
      if ($0 ~ /^[[:space:]]*$/) {
        if (started) pending = pending "\n"
        next
      }
      if (started) printf "%s", pending
      pending = ""
      started = 1
      print
    }
  ' "$changelog"
)"

if [ -z "$section" ]; then
  echo "changelog-section.sh: no changelog entry found for version $version in $changelog" >&2
  echo "changelog-section.sh: expected a heading like '## [$version] - YYYY-MM-DD'" >&2
  exit 1
fi

printf '%s\n' "$section"
