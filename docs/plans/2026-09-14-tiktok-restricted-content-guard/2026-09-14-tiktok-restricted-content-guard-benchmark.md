# TikTok Restricted Content Guard Benchmark Ledger

## Metadata

- Date: `2026-09-14`
- Plan: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-plan.md`
- Evidence: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-evidence.md`
- Benchmark: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-benchmark.md`
- Actual status: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-actual-status.md`

## B0 - P0 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|---|---|---|---|---|---|---|
| P0 | Go Test Packages Passing | count | 8/8 | 8/8 | 8/8 | 8/8 | 0 | `E0-P0A-TEST1` |
| P0 | Svelte Check Errors | count | 0 | 0 | 0 | 0 | 0 | `E0-P0A-FE1` |
| P0 | Svelte Check Warnings | count | 0 | 0 | 0 | 0 | 0 | `E0-P0A-FE1` |

## B1 - P1 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|---|---|---|---|---|---|---|
| P1-A | Domain Unit Tests | count | 7 | 8 | 8 | >=7 | +1 | `E1-P1A-TEST1` |
| P1-B | Quarantine Lifecycle Tests | count | 0 | 1 | 1 | >=1 | +1 | `E1-P1B-TEST1` |
| P1-C | TikTok Uploader Unit Tests | count | 2 | 3 | 3 | >=2 | +1 | `E1-P1C-TEST1` |

## B2 - P2 Benchmarks

| Phase | Metric | Unit | Baseline | Latest | Final | Target | Delta | Evidence |
|---|---|---|---|---|---|---|---|---|
| P2-A | Svelte Diagnostics Errors | count | 0 | 0 | 0 | 0 | 0 | `E2-P2A-FE1` |
| P2-A | Svelte Diagnostics Warnings | count | 0 | 0 | 0 | 0 | 0 | `E2-P2A-FE1` |
| P2-B | Frontend Production Build Time | s | N/A | 7.11s | 7.11s | <15s | - | `E2-P2B-BUILD1` |
| P2-B | Wails Binary Exit Code | code | N/A | 0 | 0 | 0 | 0 | `E2-P2B-BIN1` |
| P2-B | CodeGraph Synchronization | status | N/A | up-to-date | up-to-date | up-to-date | 0 | `E2-P2B-SYNC1` |
