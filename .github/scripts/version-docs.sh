#!/usr/bin/env bash
set -euo pipefail

version="${1:?Expected a release tag}"
source_root="${2:?Expected the release checkout}"
target_root="${3:?Expected the main checkout}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Expected a release tag without a pre-release suffix: $version" >&2
  exit 1
fi

source_docs="$source_root/docs"
target_docs="$target_root/docs"
baseline="${RUNNER_TEMP:?RUNNER_TEMP must be set}/docs-versions-before.json"

if [[ -f "$source_docs/versions.json" ]]; then
  cp "$source_docs/versions.json" "$baseline"
else
  printf '[]\n' > "$baseline"
fi

(
  cd "$source_docs"
  npm run docusaurus -- docs:version "$version"
)

snapshot="version-$version"
sidebar="${snapshot}-sidebars.json"
source_snapshot="$source_docs/versioned_docs/$snapshot"
target_snapshot="$target_docs/versioned_docs/$snapshot"
source_sidebar="$source_docs/versioned_sidebars/$sidebar"
target_sidebar="$target_docs/versioned_sidebars/$sidebar"

state="$(node - "$version" "$baseline" "$source_docs/versions.json" "$target_docs/versions.json" <<'NODE'
const fs = require('node:fs');
const [version, baselinePath, generatedPath, currentPath] = process.argv.slice(2);
const read = (path) => fs.existsSync(path) ? JSON.parse(fs.readFileSync(path, 'utf8')) : [];
const baseline = read(baselinePath);
const generated = read(generatedPath);
const current = read(currentPath);
if (![baseline, generated, current].every(Array.isArray)) {
  throw new Error('Documentation version lists must be arrays');
}
if (JSON.stringify(generated) !== JSON.stringify([version, ...baseline])) {
  throw new Error('Generated version list does not match the release snapshot');
}
if (current.includes(version)) {
  console.log('existing');
} else if (JSON.stringify(current) === JSON.stringify(baseline)) {
  console.log('new');
} else {
  throw new Error('Documentation versions on main changed since the release tag');
}
NODE
)"

if [[ "$state" == existing ]]; then
  diff -rq "$source_snapshot" "$target_snapshot"
  cmp "$source_sidebar" "$target_sidebar"
  echo "Documentation snapshot $version already exists and matches the release tag."
  exit 0
fi

if [[ -e "$target_snapshot" || -e "$target_sidebar" ]]; then
  echo "A documentation snapshot path already exists for $version." >&2
  exit 1
fi

mkdir -p "$target_docs/versioned_docs" "$target_docs/versioned_sidebars"
cp -R "$source_snapshot" "$target_snapshot"
cp "$source_sidebar" "$target_sidebar"
cp "$source_docs/versions.json" "$target_docs/versions.json"
