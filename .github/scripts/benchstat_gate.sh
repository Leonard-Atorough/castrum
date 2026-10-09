#!/usr/bin/env bash
# benchstat_gate.sh annotates regressions from a benchstat comparison.
# It is warn-only by design: this workflow never blocks a merge, and a
# red X on a non-blocking job is noise. Regressions surface as workflow
# annotations and in the step summary for a human to judge.
set -euo pipefail

file="${1:?usage: benchstat_gate.sh <benchstat output>}"
if [ ! -f "$file" ]; then
  echo "::warning::no benchstat output at $file"
  exit 0
fi

# benchstat prints one delta column per metric; a regression shows as a
# "+12.34%" in that column. Lines with "~" deltas are statistically
# insignificant and never match.
regressions=$(grep -E '\+[0-9]+(\.[0-9]+)?%' "$file" || true)
if [ -n "$regressions" ]; then
  echo "::warning::Benchmark regressions detected (see step summary for the full benchstat output)"
  echo "$regressions"
else
  echo "No benchmark regressions detected."
fi
exit 0
