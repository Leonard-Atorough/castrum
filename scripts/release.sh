#!/bin/bash
set -euo pipefail

# Release script for semantic versioning
# Usage: ./scripts/release.sh [patch|minor|major] [--prepare]
# --prepare: only update files, don't commit/tag (for CI)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION_FILE="$PROJECT_DIR/VERSION"
CHANGELOG_FILE="$PROJECT_DIR/CHANGELOG.md"

PREPARE_ONLY=false
if [[ "${2:-}" == "--prepare" ]]; then
    PREPARE_ONLY=true
fi

# Parse current version
if [[ ! -f "$VERSION_FILE" ]]; then
    echo "Error: VERSION file not found at $VERSION_FILE"
    exit 1
fi

CURRENT_VERSION=$(cat "$VERSION_FILE" | tr -d '\n' | tr -d ' ')

# Remove 'v' prefix if present for processing
VERSION_NUM="${CURRENT_VERSION#v}"

# Validate semver format (X.Y.Z)
if ! [[ "$VERSION_NUM" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: VERSION must be in semver format (e.g., 1.0.0), got: $VERSION_NUM"
    exit 1
fi

IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION_NUM"

# Determine bump type from argument
BUMP_TYPE="${1:-patch}"
case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "Error: BUMP_TYPE must be 'major', 'minor', or 'patch', got: $BUMP_TYPE"
        exit 1
        ;;
esac

NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

echo "Bumping version: $CURRENT_VERSION → $NEW_VERSION ($BUMP_TYPE)"

# Update VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"

# Generate changelog entry from git log since last tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -z "$LAST_TAG" ]]; then
    COMMIT_RANGE="HEAD"
else
    COMMIT_RANGE="${LAST_TAG}..HEAD"
fi

# Extract conventional commit messages
FEATURES=$(git log "$COMMIT_RANGE" --pretty=format:"%s" 2>/dev/null | grep "^feat" | sed 's/^feat[:(].*): /- /' | head -20 || true)
FIXES=$(git log "$COMMIT_RANGE" --pretty=format:"%s" 2>/dev/null | grep "^fix" | sed 's/^fix[:(].*): /- /' | head -20 || true)
PERF=$(git log "$COMMIT_RANGE" --pretty=format:"%s" 2>/dev/null | grep "^perf" | sed 's/^perf[:(].*): /- /' | head -20 || true)

# Build changelog entry
CHANGELOG_ENTRY="## [$NEW_VERSION] - $(date +%Y-%m-%d)

"

if [[ -n "$FEATURES" ]]; then
    CHANGELOG_ENTRY+="### Features

${FEATURES}

"
fi

if [[ -n "$FIXES" ]]; then
    CHANGELOG_ENTRY+="### Bug Fixes

${FIXES}

"
fi

if [[ -n "$PERF" ]]; then
    CHANGELOG_ENTRY+="### Performance

${PERF}

"
fi

# Prepend to CHANGELOG if it exists
if [[ -f "$CHANGELOG_FILE" ]]; then
    {
        echo "$CHANGELOG_ENTRY"
        cat "$CHANGELOG_FILE"
    } > "$CHANGELOG_FILE.tmp"
    mv "$CHANGELOG_FILE.tmp" "$CHANGELOG_FILE"
else
    echo "$CHANGELOG_ENTRY" > "$CHANGELOG_FILE"
fi

if [[ "$PREPARE_ONLY" == true ]]; then
    echo "✓ Files prepared for release: $NEW_VERSION"
    echo "  - VERSION file updated"
    echo "  - CHANGELOG.md updated"
    exit 0
fi

# Commit and tag (only if not in prepare mode)
git add "$VERSION_FILE" "$CHANGELOG_FILE"
git commit -m "chore(release): $NEW_VERSION"
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

echo "✓ Released $NEW_VERSION"
echo "  - VERSION file updated"
echo "  - CHANGELOG.md updated"
echo "  - Git tag created: $NEW_VERSION"
echo ""
echo "Next step: git push origin main && git push origin $NEW_VERSION"

