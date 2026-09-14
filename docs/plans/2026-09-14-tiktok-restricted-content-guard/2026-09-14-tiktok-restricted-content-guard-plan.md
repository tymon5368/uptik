# TikTok Restricted Content Guard & De-duplication Architecture Plan

## Metadata

- Date: `2026-09-14`
- Status: `draft`
- Plan: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-plan.md`
- Evidence: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-evidence.md`
- Benchmark: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-benchmark.md`
- Actual status: `docs/plans/2026-09-14-tiktok-restricted-content-guard/2026-09-14-tiktok-restricted-content-guard-actual-status.md`

## Goal

Triển khai cơ chế xử lý hoàn chỉnh cho trường hợp TikTok Studio cảnh báo nội dung bị hạn chế ("Content may be restricted" do unoriginal, low-quality, QR code) với tùy chọn cấu hình linh hoạt (`skip` vs `post_anyway`), tự động xử lý/đóng modal cảnh báo bằng CDP, cách ly video vi phạm sang thư mục `restricted/` để bảo vệ kênh, và bổ sung cơ chế nhận diện xuất bản thành công thông minh (Smart Confirmation) nhằm loại bỏ triệt để lỗi không ghi nhận video đã đăng và ngăn ngừa đăng trùng lặp ở các slot tiếp theo.

## Rules

- Complete P0 actual status before implementation work.
- Update each checklist item immediately when it is completed.
- Record evidence as work completes with exact numeric data and reproducible commands.
- Zero speculation: every assertion must be backed by file contents, compiler checks, or runtime test execution logs.
- Strict adherence to AGENTS.md, docs/CP.md (CP-1, CP-2, CP-3, CP-6, CP-7, CP-8), and docs/IB.md.
- Bun only for frontend tooling (`bun run check`, `bun run build`).
- Lucide icons only for frontend UI, zero emoji.
- Each implementation slice must have its own QA gate before moving to the next slice.

## Problem

Khi upload video lên TikTok Studio:
1. TikTok phát hiện nội dung không nguyên bản (unoriginal/low quality) và tự động bật popup modal `"Content may be restricted"`.
2. Modal này có backdrop overlay che khuất nút "Post", khiến lệnh click Post của CDP bị chặn.
3. Mã nguồn hiện tại trong `uploader.go` chỉ tìm các dialog chứa chữ "copyright", "tiếp tục đăng", nên hoàn toàn bỏ lỡ modal "Content may be restricted", dẫn đến việc app chờ 90s rồi báo lỗi Timeout.
4. Khi app báo Timeout, `SafeArchive` không được gọi, video không được chuyển vào `uploaded/`. Khi user can thiệp thủ công bấm Post trên Chrome, video đã đăng lên TikTok nhưng UpTik không biết, dẫn đến slot hẹn giờ tiếp theo lại bốc đúng file cũ để đăng tiếp, gây ra việc 1 video bị đăng trùng 3 lần.

## Scope

- Backend domain model `domain.Settings` và default settings: thêm `TikTokRestrictedPolicy` (`skip` vs `post_anyway`).
- Use case `UploadPipelineUseCase`: thêm `SafeQuarantine` để di chuyển video bị hạn chế sang `restricted/` khi policy là `skip`, ngăn việc lặp lại.
- Adapter `TikTokUploader`:
  - Mở rộng bước kiểm tra song song `Content checks` và `Music copyright`.
  - Nhận diện modal `"Content may be restricted"` (đa ngôn ngữ En/Vi).
  - Tự động đóng modal nếu policy là `post_anyway` và click Post; hoặc dừng an toàn và kích hoạt quarantine nếu policy là `skip`.
  - Tăng cường Smart Confirmation: quét liên tục URL (`/content`, `/manage`) và thông báo xuất bản kể cả khi user can thiệp bấm tay để luôn ghi nhận thành công và dọn dẹp file.
- Frontend Settings UI: thêm control cấu hình chính sách Restricted Content (Lucide icons, Paraglide i18n, Netflix theme).
- Unit tests & integration assertions cho toàn bộ luồng.

## Non-Goals

- Không can thiệp vào logic đăng của YouTube Shorts và Facebook Reels (giữ nguyên isolation theo CP-1 & CP-3).
- Không tự động sửa/render lại nội dung video (nội dung do user chuẩn bị).

## Requirements

1. **R1 (Domain & Policy)**: `domain.Settings` hỗ trợ `TikTokRestrictedPolicy` với giá trị `"skip"` (mặc định) và `"post_anyway"`.
2. **R2 (CDP Modal Handling)**: `TikTokUploader` có thể phát hiện modal "Content may be restricted", thực hiện đóng modal khi `post_anyway`, hoặc return typed error `ErrTikTokContentRestricted` khi `skip`.
3. **R3 (Quarantine & Safe Move)**: Khi video bị `ErrTikTokContentRestricted`, pipeline tự động chuyển video vào thư mục `restricted/`, giải phóng thư mục chờ để slot sau không bốc lại file này.
4. **R4 (Smart Confirmation & Anti-Duplicate)**: Sau khi bấm Post, nếu trang chuyển sang `/content` hoặc có thông báo thành công thì luôn đánh dấu hoàn tất và gọi `SafeArchive`, tránh tình trạng "video đã lên TikTok nhưng app báo thất bại".
5. **R5 (UI & i18n)**: Tab Cài Đặt hiển thị cấu hình chính sách rõ ràng, thân thiện, 100% Lucide icons và hỗ trợ đa ngôn ngữ.

## Acceptance Criteria

- AC-1: `go test ./...` pass 100% bao gồm cả test case mới cho `SafeQuarantine`, `TikTokRestrictedPolicy`, và dialog handling.
- AC-2: `bun run check` pass 0 errors, 0 warnings.
- AC-3: `bun run build` build production frontend thành công.
- AC-4: `go build -tags "webkit2_41"` build binary ứng dụng Wails thành công.
- AC-5: `codegraph sync` cập nhật chỉ mục kiến trúc đầy đủ.

## Checklist

- [x] P0-A: Complete actual status before implementation work.
  - Goal: Thu thập đầy đủ hiện trạng source code, symbols liên quan, file relationships và metric baseline.
  - Work Steps: Khám phá `internal/domain`, `internal/adapters/platforms/tiktok`, `internal/usecases`, `frontend/src`.
  - Implementation Gate: Kiểm tra tests và baseline build pass trước khi viết code mới.

### P1: Backend Domain, Quarantine & Smart CDP Handler

- [x] P1-A: Cập nhật Domain Model & Settings Repository với `TikTokRestrictedPolicy`.
  - Work Steps: Thêm field trong `internal/domain/models.go`, thiết lập default trong `settings.go`, bổ sung helper và tests.
  - QA Gate: `go test ./internal/domain/...` pass.

- [x] P1-B: Bổ sung `SafeQuarantine` vào `UploadPipelineUseCase` và xử lý typed error `ErrTikTokContentRestricted`.
  - Work Steps: Cập nhật `internal/usecases/upload_pipeline.go`, đảm bảo video bị hạn chế được chuyển vào `restricted/`, không kẹt ở root.
  - QA Gate: `go test ./internal/usecases/...` pass 100%.

- [x] P1-C: Nâng cấp `TikTokUploader` với Content Check & Modal Handler & Smart Confirmation.
  - Work Steps: Cập nhật `internal/adapters/platforms/tiktok/uploader.go`, thêm detection cho Content checks, auto dismiss modal khi `post_anyway`, quarantine error khi `skip`, và smart detection cho URL `/content`.
  - QA Gate: `go test ./internal/adapters/platforms/tiktok/...` pass.

### P2: Frontend UI Settings & End-to-End Build Verification

- [x] P2-A: Cập nhật Frontend Types, Paraglide Messages và Settings UI trong `App.svelte`.
  - Work Steps: Thêm `tiktokRestrictedPolicy` vào `frontend/src/lib/types.ts`, thêm i18n keys trong `messages/en.json` và `messages/vi.json`, thêm radio toggle trong tab Cài Đặt.
  - QA Gate: `bun run check` pass 0 errors, 0 warnings.

- [x] P2-B: Full System Build, CodeGraph Sync & Evidence Ledger Closure.
  - Work Steps: Chạy `bun run build`, `go test ./...`, `go build -tags "webkit2_41"`, `codegraph sync`.
  - QA Gate: Toàn bộ build pass, bằng chứng số liệu thật được ghi đầy đủ vào benchmark & evidence.
