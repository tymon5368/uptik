package usecases

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uptik/internal/adapters/platforms"
	"uptik/internal/adapters/queue"
	"uptik/internal/adapters/storage/sqlite"
	"uptik/internal/domain"
	"uptik/internal/ports"

	"github.com/go-rod/rod"
)

type mockPlatform struct {
	id   string
	name string
	fail bool
}

func (m *mockPlatform) ID() string          { return m.id }
func (m *mockPlatform) DisplayName() string { return m.name }
func (m *mockPlatform) LoginURL() string    { return "https://example.com" }
func (m *mockPlatform) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	return true, nil
}
func (m *mockPlatform) UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error {
	if m.fail {
		return os.ErrInvalid
	}
	return nil
}

func TestUseCases(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "usecase_test.db")

	storage, err := sqlite.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize sqlite storage: %v", err)
	}
	defer storage.Close()

	// 1. Test ScanVideosUseCase
	videoFolder := filepath.Join(tmpDir, "videos")
	_ = os.MkdirAll(videoFolder, 0755)

	f1 := filepath.Join(videoFolder, "movie1 (1080p).mp4")
	f2 := filepath.Join(videoFolder, "movie2.mp4")
	_ = os.WriteFile(f1, []byte("fake video 1"), 0644)
	_ = os.WriteFile(f2, []byte("fake video 2"), 0644)

	scanUC := NewScanVideosUseCase(storage)
	scanned, err := scanUC.Execute(videoFolder, "#phim")
	if err != nil {
		t.Fatalf("ScanVideosUseCase failed: %v", err)
	}
	if len(scanned) != 2 {
		t.Fatalf("Expected 2 scanned videos, got %d", len(scanned))
	}
	if scanned[0].CleanTitle != "movie1 #phim" {
		t.Errorf("Clean title wrong: %s", scanned[0].CleanTitle)
	}

	// 2. Test ScheduleSlotsUseCase
	_ = storage.Append(domain.HistoryRecord{
		Filename:      "old.mp4",
		Title:         "Old Video",
		ScheduledDate: "2026-10-01",
		ScheduledTime: "11:30",
		Channels:      []string{"tiktok"},
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	})

	schedUC := NewScheduleSlotsUseCase(storage)
	startDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	scheduled := schedUC.Execute(scanned, []string{"11:30", "18:30"}, 30, startDate)

	if len(scheduled) != 2 {
		t.Fatalf("Expected 2 scheduled videos, got %d", len(scheduled))
	}
	// Slot 1 should avoid 11:30 (taken) and take 18:30
	if scheduled[0].ScheduledDate != "2026-10-01" || scheduled[0].ScheduledTime != "18:30" {
		t.Errorf("Expected 2026-10-01 18:30, got %s %s", scheduled[0].ScheduledDate, scheduled[0].ScheduledTime)
	}

	// 3. Test UploadPipelineUseCase
	reg := platforms.NewRegistry()
	reg.Register(&mockPlatform{id: "tiktok", name: "TikTok Studio"})
	reg.Register(&mockPlatform{id: "youtube", name: "YouTube Shorts"})

	jobQueue := queue.NewPersistentJobQueue(storage)
	pipeUC := NewUploadPipelineUseCase(reg, storage, jobQueue, nil)

	ctx := context.Background()
	enqueued, err := jobQueue.Enqueue(ctx, scheduled[0], []string{"tiktok", "youtube"})
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	err = pipeUC.ProcessJob(ctx, nil, enqueued)
	if err != nil {
		t.Fatalf("ProcessJob failed: %v", err)
	}

	// Verify job is marked completed
	savedJob, err := storage.GetJob(enqueued.ID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if savedJob.State != ports.JobStateCompleted {
		t.Errorf("Expected job completed, got: %s", savedJob.State)
	}

	// Verify history updated
	history, err := storage.GetAll()
	if err != nil {
		t.Fatalf("GetAll history failed: %v", err)
	}
	if len(history) < 2 {
		t.Errorf("Expected history to include newly uploaded video, got %d", len(history))
	}

	// 4. Test BackgroundSchedulerService
	st := domain.Settings{
		VideoFolder:       videoFolder,
		DefaultTag:        "#phim",
		GoldenHours:       []string{"11:30", "18:30"},
		EnabledChannels:   []string{"tiktok"},
		AutoUploadEnabled: true,
	}
	_ = storage.Save(st)

	triggeredItems := []domain.VideoItem{}
	triggerUploadMock := func(items []domain.VideoItem, channels []string) error {
		triggeredItems = append(triggeredItems, items...)
		return nil
	}

	schedulerSvc := NewBackgroundSchedulerService(
		storage,
		scanUC,
		jobQueue,
		nil,
		nil,
		triggerUploadMock,
		func() bool { return false },
	)

	// Test GetNextSlotStatus
	nextD, nextT, rem, label := schedulerSvc.GetNextSlotStatus(st)
	if nextD == "" || nextT == "" || rem <= 0 || label == "" {
		t.Errorf("GetNextSlotStatus returned invalid values: %s %s %d %s", nextD, nextT, rem, label)
	}

	// Test TriggerManual
	err = schedulerSvc.TriggerManual(st)
	if err != nil {
		t.Fatalf("TriggerManual failed: %v", err)
	}
	if len(triggeredItems) != 1 {
		t.Fatalf("Expected 1 triggered video, got %d", len(triggeredItems))
	}
	if triggeredItems[0].PublishMode != domain.PublishModePublishNow {
		t.Errorf("Expected PublishMode = %s, got %s", domain.PublishModePublishNow, triggeredItems[0].PublishMode)
	}
}

type mockRestrictedPlatform struct {
	id   string
	name string
}

func (m *mockRestrictedPlatform) ID() string          { return m.id }
func (m *mockRestrictedPlatform) DisplayName() string { return m.name }
func (m *mockRestrictedPlatform) LoginURL() string    { return "https://example.com" }
func (m *mockRestrictedPlatform) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	return true, nil
}
func (m *mockRestrictedPlatform) UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error {
	return domain.ErrContentRestricted
}

func TestUploadPipelineQuarantine(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "quarantine_test.db")

	storage, err := sqlite.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize sqlite storage: %v", err)
	}
	defer storage.Close()

	videoFolder := filepath.Join(tmpDir, "videos")
	_ = os.MkdirAll(videoFolder, 0755)

	videoFile := filepath.Join(videoFolder, "restricted_movie.mp4")
	_ = os.WriteFile(videoFile, []byte("fake video content flagged as unoriginal"), 0644)

	reg := platforms.NewRegistry()
	reg.Register(&mockRestrictedPlatform{id: "tiktok", name: "TikTok Studio"})

	jobQueue := queue.NewPersistentJobQueue(storage)
	pipeUC := NewUploadPipelineUseCase(reg, storage, jobQueue, nil)

	ctx := context.Background()
	item := domain.VideoItem{
		Filename:    "restricted_movie.mp4",
		FullPath:    videoFile,
		CustomTitle: "Restricted Movie",
		Channels:    map[string]domain.ChannelStatus{},
	}

	enqueued, err := jobQueue.Enqueue(ctx, item, []string{"tiktok"})
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	err = pipeUC.ProcessJob(ctx, nil, enqueued)
	if err == nil {
		t.Fatalf("Expected ProcessJob to return quarantine error, got nil")
	}

	// Verify original file is gone from root folder
	if _, err := os.Stat(videoFile); !os.IsNotExist(err) {
		t.Errorf("Expected original file to be moved, but it still exists at %s", videoFile)
	}

	// Verify file is moved to restricted/ subdirectory
	quarantinedPath := filepath.Join(videoFolder, "restricted", "restricted_movie.mp4")
	if _, err := os.Stat(quarantinedPath); err != nil {
		t.Errorf("Expected file in quarantined path %s, stat error: %v", quarantinedPath, err)
	}

	// Verify job state
	savedJob, err := storage.GetJob(enqueued.ID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if savedJob.State != ports.JobStateFailed {
		t.Errorf("Expected job state failed, got %s", savedJob.State)
	}
}

func TestUploadPipelineQuarantine_CollisionAndPartial(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uptik_quarantine_collision_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage, err := sqlite.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize sqlite storage: %v", err)
	}
	defer storage.Close()

	videoFolder := filepath.Join(tmpDir, "videos")
	_ = os.MkdirAll(videoFolder, 0755)
	restrictedFolder := filepath.Join(videoFolder, "restricted")
	_ = os.MkdirAll(restrictedFolder, 0755)

	// Pre-create existing file in restricted folder to trigger collision guard
	existingInRestricted := filepath.Join(restrictedFolder, "sample.mp4")
	_ = os.WriteFile(existingInRestricted, []byte("pre-existing quarantined video"), 0644)

	// New video with identical name in input folder
	videoFile := filepath.Join(videoFolder, "sample.mp4")
	_ = os.WriteFile(videoFile, []byte("second video with same name"), 0644)

	reg := platforms.NewRegistry()
	reg.Register(&mockRestrictedPlatform{id: "tiktok", name: "TikTok Studio"})
	reg.Register(&mockPlatform{id: "youtube", name: "YouTube Shorts"})

	jobQueue := queue.NewPersistentJobQueue(storage)
	pipeUC := NewUploadPipelineUseCase(reg, storage, jobQueue, nil)

	item := domain.VideoItem{
		Filename:    "sample.mp4",
		FullPath:    videoFile,
		CustomTitle: "Sample Video",
		Channels:    map[string]domain.ChannelStatus{},
	}

	// Test collision-safe SafeQuarantine directly
	err = pipeUC.SafeQuarantine(item)
	if err != nil {
		t.Fatalf("SafeQuarantine failed with collision: %v", err)
	}

	// Pre-existing file must be intact
	content, err := os.ReadFile(existingInRestricted)
	if err != nil || string(content) != "pre-existing quarantined video" {
		t.Errorf("Pre-existing file was overwritten or corrupted!")
	}

	// Root file must be moved
	if _, err := os.Stat(videoFile); !os.IsNotExist(err) {
		t.Errorf("Original file was not moved from root folder")
	}

	// Test partial success with restricted channel
	partialFile := filepath.Join(videoFolder, "partial.mp4")
	_ = os.WriteFile(partialFile, []byte("partial content"), 0644)
	partialItem := domain.VideoItem{
		Filename:    "partial.mp4",
		FullPath:    partialFile,
		CustomTitle: "Partial Video",
		Channels:    map[string]domain.ChannelStatus{},
	}

	ctx := context.Background()
	enqueued, err := jobQueue.Enqueue(ctx, partialItem, []string{"youtube", "tiktok"})
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	err = pipeUC.ProcessJob(ctx, nil, enqueued)
	if err == nil {
		t.Fatalf("Expected partial error, got nil")
	}

	// Verify that partial restricted video was quarantined
	if _, err := os.Stat(partialFile); !os.IsNotExist(err) {
		t.Errorf("Partial restricted video was not moved from root folder")
	}
}
