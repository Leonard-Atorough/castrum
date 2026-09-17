#!/usr/bin/env bash
# Computes the next semver tag from conventional commits since the last v* tag.
# Usage: bump.sh [auto|major|minor|patch] [pre-release-suffix]
#   e.g. bump.sh auto        -> v0.3.0
#        bump.sh auto rc     -> v0.3.0-rc.1
#        bump.sh minor beta  -> v0.4.0-beta.1
set -euo pipefail

MODE="${1:-auto}"
PRE="${2:-}"
case "${MODE}" in
  auto|major|minor|patch) ;;
  *) echo "Error: invalid bump mode '${MODE}'. Usage: bump.sh [auto|major|minor|patch] [pre-release-suffix]" >&2; exit 1 ;;
esac

LATEST="$(git tag --list 'v*' --sort=-v:refname | head -n1 || true)"

if [[ -z "${LATEST}" ]]; then
  RANGE=""
  MAJOR=0; MINOR=0; PATCH=0
else
  # Use the latest stable tag (no pre-release suffix) as the base version,
  # so that v0.2.1-rc.1 -> rc.2 keeps the same base (0.2.1) instead of
  # bumping the patch again. The range is still from the latest tag (pre-release
  # or stable) so the "nothing to release" check catches no-op dispatches.
  BASE_TAG="$(git tag --list 'v*' --sort=-v:refname | grep -v -- '-' | head -n1 || true)"
  [[ -z "${BASE_TAG}" ]] && BASE_TAG="${LATEST}"
  PREV="${BASE_TAG#v}"
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

VERSION="v${MAJOR}.${MINOR}.${PATCH}"

# Append a pre-release suffix (e.g. -rc.1, -beta.2) if requested.
# The number is incremented from the highest existing tag for this version+suffix.
if [[ -n "${PRE}" ]]; then
  PREFIX="${VERSION}-${PRE}."
  LAST_NUM="$(git tag --list "${PREFIX}*" --sort=-v:refname | head -n1 | sed "s|.*${PREFIX}||" || true)"
  if [[ -n "${LAST_NUM}" && "${LAST_NUM}" =~ ^[0-9]+$ ]]; then
    NUM=$((LAST_NUM + 1))
  else
    NUM=1
  fi
  VERSION="${PREFIX}${NUM}"
fi

echo "${VERSION}"