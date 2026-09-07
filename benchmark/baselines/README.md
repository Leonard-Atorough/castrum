# Benchmark Baselines

This directory stores performance baselines for regression tracking and historical comparison.

## Files

- `latest.txt` — Current main branch baseline (updated on every merge to main)
- `v*.txt` — Tagged release baselines for version comparison

## Workflow

### Establishing a new baseline (on main branch merge)

```bash
cd benchmark
go test -bench=. -benchmem -benchtime=1s -run=^$ ./... > baselines/latest.txt
git add baselines/latest.txt
git commit -m "perf: update benchmark baseline"
```

### Comparing PR against baseline

```bash
# In CI/CD
benchstat baselines/latest.txt pr-results.txt
```

### Tagging release baselines

```bash
# After releasing v1.0.0
cp baselines/latest.txt baselines/v1.0.0.txt
```

## Interpreting Results

- **~0%** — No performance change (expected variance ±2%)
- **+3-5%** — Potential regression (watch for trends)
- **+5-10%** — Meaningful regression (should be documented)
- **>+10%** — Critical regression (PR should not merge on critical paths)
- **-5% or better** — Optimization success! 🎉

## Historical Performance Tracking

### v1.0.0 (Phase 2 Complete)
- GameLoopSimple: 67,378 ns/op (41% improvement vs Phase 1)
- QueryIterateNoAccess: 13 allocs/op (155x improvement)
- All 40+ tests passing
- Zero regressions

### Future versions
- v1.1.0 — Phase 3: Archetype optimization
- v1.2.0 — Phase 4: Infrastructure & monitoring
