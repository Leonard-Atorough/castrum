#!/usr/bin/env bash
# Computes the next semver tag from conventional commits since the last v* tag.
# Usage: scripts/bump.sh [auto|major|minor|patch]
set -euo pipefail

MODE="${1:-auto}"
case "${MODE}" in
  auto|major|minor|patch) ;;
  *) echo "Error: invalid bump mode '${MODE}'. Usage: bump.sh [auto|major|minor|patch]" >&2; exit 1 ;;
esac

LATEST="$(git tag --list 'v*' --sort=-v:refname | head -n1 || true)"

if [[ -z "${LATEST}" ]]; then
  RANGE=""
  MAJOR=0; MINOR=0; PATCH=0
else
  PREV="${LATEST#v}"
  RANGE="${LATEST}..HEAD"
  IFS='.' read -r MAJOR MINOR PATCH <<<"${PREV}"
fi

SUBJECTS="$(git log --pretty=format:%s ${RANGE} 2>/dev/null || true)"
BODIES="$(git log --pretty=format:%B ${RANGE} 2>/dev/null || true)"

LEVEL="patch"
# Conventional type analysis (subject prefixes + trailer detection)
if   printf '%s\n' "${SUBJECTS}" | grep -Eq '^[a-zA-Z]+(\(.+\))?!:' ||
     grep -qiE '^BREAKING[- ]CHANGE:' <<<"${BODIES}"; then
  LEVEL="major"
elif grep -Eq '^feat(\(|:|!)' <<<"${SUBJECTS}"; then
  LEVEL="minor"
fi

# Pre-1.0 rule: breaking changes bump the minor, never the major
if [[ "${MAJOR}" == "0" && "${LEVEL}" == "major" ]]; then
  LEVEL="minor"
fi

# Honor explicit override
if [[ "${MODE}" != "auto" ]]; then
  LEVEL="${MODE}"
fi

case "${LEVEL}" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

echo "v${MAJOR}.${MINOR}.${PATCH}"