#!/usr/bin/env bash
# Gate CI on benchstat output.
#
# Usage: benchstat_gate.sh <benchstat-output.txt>
#
# Rules:
#   sec/op  (time):   fail on a *significant* regression of >= NS_THRESHOLD%.
#   allocs/op:        fail on any *significant* increase (allocations are deterministic).
#   B/op:             advisory only (warned, never fails).
#
# "Significant" means benchstat printed a delta (not "~"), which already
# implies the t-test p-value crossed its threshold. We do not re-check p here.
#
# Thresholds:
NS_THRESHOLD="${NS_THRESHOLD:-10}"   # min % regression in sec/op to fail

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: benchstat_gate.sh <benchstat-output.txt>" >&2
  exit 2
fi

awk -v ns_threshold="$NS_THRESHOLD" '
# A header row contains the box-drawing border (│) and a unit token, but is
# not a data row (does not start with "Benchmark" or "geomean"). It tells us
# which metric the following data rows report.
/│/ && !/^[[:space:]]*(Benchmark|geomean)/ {
    if (/sec\/op/)      unit = "time"
    else if (/allocs\/op/) unit = "allocs"
    else if (/B\/op/)     unit = "bytes"
    next
}
# Skip the geomean summary row — it has a delta but aggregates all benchmarks.
/^[[:space:]]*geomean/ { next }

# Data row. Find the "(p=..." field; the field before it is the delta token.
{
    delta = ""
    for (i = 1; i <= NF; i++) {
        if ($i ~ /^\(p=/) {
            delta = $(i - 1)
            pval = $i; sub(/^\(p=/, "", pval)
            nval = $(i + 1); sub(/^n=/, "", nval); sub(/\)$/, "", nval)
            break
        }
    }
    if (delta == "" || delta == "~") next   # no delta, or not significant

    name = $1
    sign = substr(delta, 1, 1)
    mag  = substr(delta, 2); sub(/%$/, "", mag) + 0

    if (unit == "time") {
        if (sign == "+" && mag + 0 >= ns_threshold) {
            print "::error::Regression: " name ": sec/op " delta " (p=" pval ", n=" nval ") >= " ns_threshold "% regression"
            fail = 1
        }
    } else if (unit == "allocs") {
        if (sign == "+") {
            print "::error::Regression: " name ": allocs/op " delta " (p=" pval ", n=" nval ") increased"
            fail = 1
        }
    } else if (unit == "bytes") {
        if (sign == "+") {
            print "::warning::B/op advisory (non-fatal): " name ": B/op " delta " (p=" pval ", n=" nval ") increased"
        }
    }
}

END {
    if (fail) {
        print ""
        print "Performance regression(s) detected. See benchstat output above."
        exit 1
    }
    print "No significant regressions detected."
}
' "$1"
