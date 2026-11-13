#!/usr/bin/env bash
# Run from the repository root. An optional version verifies published modules.
set -euo pipefail

version="${1:-}"
module="github.com/Leonard-Atorough/castrum"
flags=()
if [[ -n "${version}" ]]; then
  export GOWORK=off
  flags=(-mod=mod)
fi

while IFS= read -r -d '' module_file; do
  directory="$(dirname "${module_file}")"
  echo "Checking ${directory}"
  (
    cd "${directory}"
    if [[ -n "${version}" ]]; then
      resolved="$(go list -mod=mod -m -f '{{.Version}} {{if .Replace}}replaced{{end}}' "${module}")"
      if [[ "${resolved}" != "${version} " ]]; then
        echo "::error::${directory} must resolve published ${module}@${version}, got ${resolved}"
        exit 1
      fi
    fi
    go build "${flags[@]}" ./...
    go vet "${flags[@]}" ./...
    xvfb-run -a go test "${flags[@]}" -race ./...
  )
done < <(find examples templates/starter -name go.mod -print0)
