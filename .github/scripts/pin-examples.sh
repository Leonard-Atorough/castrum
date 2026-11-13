#!/usr/bin/env bash
# Pin standalone modules before tagging; the workspace validates the checkout.
set -euo pipefail

version="${1:?Usage: pin-examples.sh VERSION}"
if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "::error::Invalid release version: ${version}"
  exit 1
fi

while IFS= read -r -d '' module_file; do
  directory="$(dirname "${module_file}")"
  (
    cd "${directory}"
    go mod edit "-require=github.com/Leonard-Atorough/castrum@${version}" \
      "-dropreplace=github.com/Leonard-Atorough/castrum"
  )
done < <(find examples templates/starter -name go.mod -print0)
