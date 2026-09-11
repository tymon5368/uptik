package usecases

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uptik/internal/domain"
	"uptik/internal/ports"

	"github.com/go-rod/rod"
)

type UploadPipelineUseCase struct {
	registry    ports.PlatformRegistry
	historyRepo ports.HistoryRepository
	jobQueue    ports.JobQueue
	logFn       func(level, msg string)
}

func NewUploadPipelineUseCase(
	registry ports.PlatformRegistry,
	historyRepo ports.HistoryRepository,
	jobQueue ports.JobQueue,
	logFn func(level, msg string),
) *UploadPipelineUseCase {
	return &UploadPipelineUseCase{
		registry:    registry,
		historyRepo: historyRepo,
		jobQueue:    jobQueue,
		logFn:       logFn,
	}
}

func (uc *UploadPipelineUseCase) Log(level, msg string) {
	if uc.logFn != nil {
		uc.logFn(level, msg)
	}
}

// ProcessJob uploads a single job across all its target channels sequentially
func (uc *UploadPipelineUseCase) ProcessJob(ctx context.Context, b *rod.Browser, job *ports.JobItem) error {
	item := &job.Video
	targetChannels := job.TargetChannels
	if len(targetChannels) == 0 {
		targetChannels = []string{"tiktok"}
	}

	if item.Channels == nil {
		item.Channels = make(map[string]domain.ChannelStatus)
	}

	uc.Log("info", fmt.Sprintf("🚀 [PIPELINE] Bắt đầu xử lý: %s (%d kênh: %s)", item.CustomTitle, len(targetChannels), strings.Join(targetChannels, ", ")))

	// Mark state to uploading in queue
	_ = uc.jobQueue.MarkState(ctx, job.ID, ports.JobStateUploading, "")

	var successfulChannels []string
	var failedChannels []string

	for idx, ch := range targetChannels {
		select {
		case <-ctx.Done():
			_ = uc.jobQueue.MarkState(ctx, job.ID, ports.JobStateCancelled, "Người dùng đã hủy tiến trình")
			return ctx.Err()
		default:
		}

		platform, err := uc.registry.Get(ch)
		if err != nil {
			uc.Log("error", fmt.Sprintf("Bỏ qua nền tảng không hợp lệ %s: %v", ch, err))
			continue
		}

		uc.Log("info", fmt.Sprintf("▶️ [%d/%d] Đang upload lên %s...", idx+1, len(targetChannels), platform.DisplayName()))
		item.Channels[ch] = domain.ChannelStatus{Status: "uploading"}

		uploadErr := platform.UploadVideo(ctx, b, item, uc.logFn)
		if uploadErr != nil {
			uc.Log("error", fmt.Sprintf("❌ Lỗi upload %s: %v", platform.DisplayName(), uploadErr))
			item.Channels[ch] = domain.ChannelStatus{
				Status:   "error",
				ErrorMsg: uploadErr.Error(),
			}
			failedChannels = append(failedChannels, ch)
		} else {
			uc.Log("success", fmt.Sprintf("✅ Upload thành công lên %s!", platform.DisplayName()))
			item.Channels[ch] = domain.ChannelStatus{
				Status:     "scheduled",
				UploadedAt: time.Now().Format("15:04:05"),
			}
			successfulChannels = append(successfulChannels, ch)
		}

		// Courtesy delay between platforms to free resources & prevent spam detection
		if idx < len(targetChannels)-1 {
			uc.Log("info", "⏳ Nghỉ 5 giây giữa các nền tảng để giải phóng tài nguyên...")
			time.Sleep(5 * time.Second)
		}
	}

	// Record to history if at least one succeeded
	if len(successfulChannels) > 0 && uc.historyRepo != nil {
		_ = uc.historyRepo.Append(domain.HistoryRecord{
			Filename:      item.Filename,
			Title:         item.CustomTitle,
			ScheduledDate: item.ScheduledDate,
			ScheduledTime: item.ScheduledTime,
			Channels:      successfulChannels,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		})
	}

	// If all channels succeeded: safe move to uploaded/ and mark completed
	if len(failedChannels) == 0 && len(successfulChannels) > 0 {
		item.Status = "scheduled"
		item.UploadedAt = time.Now().Format("15:04:05")

		_ = uc.SafeArchive(*item)
		_ = uc.jobQueue.MarkState(ctx, job.ID, ports.JobStateCompleted, "")
		return nil
	}

	// Partial success
	if len(successfulChannels) > 0 && len(failedChannels) > 0 {
		item.Status = "partial"
		item.UploadedAt = time.Now().Format("15:04:05")
		errMsg := fmt.Sprintf("Thành công %s, thất bại trên: %s", strings.Join(successfulChannels, ", "), strings.Join(failedChannels, ", "))
		_ = uc.jobQueue.MarkState(ctx, job.ID, ports.JobStatePartial, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Total failure
	item.Status = "error"
	errMsg := fmt.Sprintf("Thất bại trên toàn bộ các kênh: %s", strings.Join(failedChannels, ", "))
	_ = uc.jobQueue.MarkState(ctx, job.ID, ports.JobStateFailed, errMsg)
	return fmt.Errorf("%s", errMsg)
}

// SafeArchive moves an uploaded video to uploaded/ subfolder atomically
func (uc *UploadPipelineUseCase) SafeArchive(video domain.VideoItem) error {
	folder := filepath.Dir(video.FullPath)
	uploadedDir := filepath.Join(folder, "uploaded")
	_ = os.MkdirAll(uploadedDir, 0755)

	destPath := filepath.Join(uploadedDir, video.Filename)
	err := os.Rename(video.FullPath, destPath)
	if err != nil {
		uc.Log("warn", fmt.Sprintf("Không thể di chuyển file: %v", err))
		return err
	}
	uc.Log("success", fmt.Sprintf("📁 Đã di chuyển an toàn video vào: uploaded/%s", video.Filename))
	return nil
}
