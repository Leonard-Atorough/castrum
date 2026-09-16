#!/usr/bin/env python3
"""Gate CI on benchstat output.

Usage: benchstat_gate.py <benchstat-output.txt>

Rules:
  - sec/op (time): fail on a *significant* regression of >= NS_THRESHOLD%.
  - allocs/op:     fail on any *significant* increase (allocations are deterministic).
  - B/op:          advisory only (reported, never fails).

"Significant" means benchstat printed a delta (not "~"), which already implies
the t-test p-value crossed its threshold. We do not re-check p here.
"""
import re
import sys

NS_THRESHOLD = 10.0

# benchstat metric-unit header tokens -> category.
UNITS = {
    "sec/op": "time",
    "allocs/op": "allocs",
    "B/op": "bytes",
}

# Matches the delta token immediately before "(p=... n=...)".
# Delta is "~" (not significant) or "+12.34%" / "-7.8%".
DELTA_RE = re.compile(r"(~|[+-][0-9]+(?:\.[0-9]+)?%)\s*\(p=([0-9.]+)\s+n=([0-9]+)\)")


def parse(path):
    failures = []
    advisories = []
    current_unit = None
    with open(path, encoding="utf-8") as f:
        for line in f:
            # Unit header lines contain a column border and a unit token, but
            # are not data rows.
            if "\u2502" in line and not line.lstrip().startswith(("Benchmark", "geomean")):
                for tok, cat in UNITS.items():
                    if tok in line:
                        current_unit = cat
                        break
                continue
            m = DELTA_RE.search(line)
            if not m or line.lstrip().startswith("geomean"):
                continue
            delta = m.group(1)
            if delta == "~":
                continue  # not statistically significant
            sign = delta[0]
            mag = float(delta[1:].rstrip("%"))
            name = line.split()[0]
            if current_unit == "time":
                if sign == "+" and mag >= NS_THRESHOLD:
                    failures.append(
                        f"{name}: sec/op {delta} (p={m.group(2)}, n={m.group(3)}) "
                        f">= {NS_THRESHOLD:.0f}% regression"
                    )
            elif current_unit == "allocs":
                if sign == "+":
                    failures.append(
                        f"{name}: allocs/op {delta} (p={m.group(2)}, n={m.group(3)}) increased"
                    )
            elif current_unit == "bytes":
                if sign == "+":
                    advisories.append(
                        f"{name}: B/op {delta} (p={m.group(2)}, n={m.group(3)}) increased"
                    )
    return failures, advisories


def main():
    if len(sys.argv) != 2:
        print("usage: benchstat_gate.py <benchstat-output.txt>", file=sys.stderr)
        sys.exit(2)
    failures, advisories = parse(sys.argv[1])
    for a in advisories:
        print(f"::warning::B/op advisory (non-fatal): {a}")
    if failures:
        for f in failures:
            print(f"::error::Regression: {f}")
        print("\nPerformance regression(s) detected. See benchstat output above.")
        sys.exit(1)
    print("No significant regressions detected.")


if __name__ == "__main__":
    main()
