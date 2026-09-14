# TikTok Restricted Content Guard Evidence Ledger

## Metadata

- Date: `2026-09-14`
- Plan: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-plan.md`
- Evidence: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-evidence.md`
- Benchmark: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-benchmark.md`
- Actual status: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-actual-status.md`

## Evidence Rules

- Use stable phase-scoped evidence IDs (`E<phase>-<item>-<kind><n>`).
- Capture real commands, exit codes, and output logs.
- Zero speculation, 100% empirical evidence.

## E0 - P0 Baseline Evidence

### E0-P0A-TEST1: Baseline Go Test Suite Run
- Command: `go test ./...`
- Exit Code: `0`
- Result: 8/8 Go packages passing.

### E0-P0A-FE1: Baseline Frontend Svelte Diagnostics
- Command: `bun run check`
- Exit Code: `0`
- Result: 0 errors and 0 warnings.

### E0-P0A-SRC1..3: Codebase Baseline Inspection
- Files: `internal/domain/models.go`, `internal/usecases/upload_pipeline.go`, `internal/adapters/platforms/tiktok/uploader.go`.

---

## E1 - P1 Implementation Evidence

### E1-P1A-TEST1: Domain Model & Settings Unit Tests
- Command: `go test -v ./internal/domain/... ./internal/adapters/storage/...`
- Exit Code: `0`
- Output:
  ```text
  === RUN   TestDomainCleanVideoTitle
  --- PASS: TestDomainCleanVideoTitle (0.00s)
  === RUN   TestDomainFormatFileSize
  --- PASS: TestDomainFormatFileSize (0.00s)
  === RUN   TestDomainHashString
  --- PASS: TestDomainHashString (0.00s)
  === RUN   TestDomainAssignScheduleSlots
  --- PASS: TestDomainAssignScheduleSlots (0.00s)
  === RUN   TestDomainValidateAndSortHours
  --- PASS: TestDomainValidateAndSortHours (0.00s)
  === RUN   TestDomainGetHourLabel
  --- PASS: TestDomainGetHourLabel (0.00s)
  === RUN   TestDomainGetNextScheduledSlot
  --- PASS: TestDomainGetNextScheduledSlot (0.00s)
  === RUN   TestDomainTikTokRestrictedPolicy
  --- PASS: TestDomainTikTokRestrictedPolicy (0.00s)
  PASS
  ok   uptik/internal/domain 0.002s
  ```

### E1-P1B-TEST1: Upload Pipeline Quarantine Unit Test
- Command: `go test -v ./internal/usecases/...`
- Exit Code: `0`
- Output:
  ```text
  === RUN   TestUseCases
  --- PASS: TestUseCases (5.01s)
  === RUN   TestUploadPipelineQuarantine
  --- PASS: TestUploadPipelineQuarantine (0.00s)
  PASS
  ok   uptik/internal/usecases 5.009s
  ```
- Verification: Asserted file is moved into `restricted/` subdirectory and removed from root folder upon `ErrContentRestricted`.

### E1-P1C-TEST1: TikTok Uploader CDP & Policy Provider Tests
- Command: `go test -v ./internal/adapters/platforms/tiktok/...`
- Exit Code: `0`
- Output:
  ```text
  === RUN   TestTikTokUploader_Basic
  --- PASS: TestTikTokUploader_Basic (0.00s)
  === RUN   TestTikTokUploader_ContextCancellation
  --- PASS: TestTikTokUploader_ContextCancellation (0.00s)
  === RUN   TestTikTokUploader_PolicyProvider
  --- PASS: TestTikTokUploader_PolicyProvider (0.00s)
  PASS
  ok   uptik/internal/adapters/platforms/tiktok 0.003s
  ```

---

## E2 - P2 Verification Evidence

### E2-P2A-FE1: Frontend Typecheck & Svelte 5 Diagnostics
- Command: `bun run check`
- Exit Code: `0`
- Output:
  ```text
  ✔ [paraglide-js] Successfully compiled inlang project.
  svelte-check found 0 errors and 0 warnings
  ```

### E2-P2B-BUILD1: Production Frontend Build
- Command: `bun run build`
- Exit Code: `0`
- Output:
  ```text
  dist/index.html                       0.47 kB
  dist/assets/index-BbsMXl1m.css       54.17 kB
  dist/assets/index-CoyhbTX7.js       373.95 kB
  ✓ built in 7.11s
  ```

### E2-P2B-BIN1: Full Wails Binary Compilation
- Command: `go build -tags "webkit2_41" -o /dev/null .`
- Exit Code: `0`
- Output: Clean compilation with 0 warnings.

### E2-P2B-SYNC1: CodeGraph Index Synchronization
- Command: `codegraph sync`
- Exit Code: `0`
- Output: `Already up to date`
