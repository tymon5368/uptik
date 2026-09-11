package usecases

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"
	"uptik/internal/domain"
	"uptik/internal/ports"
)

type SchedulerNotificationFunc func(title, message string)

// BackgroundSchedulerService manages automated video upload at scheduled time slots
type BackgroundSchedulerService struct {
	settingsRepo  ports.SettingsRepository
	scanUC        *ScanVideosUseCase
	jobQueue      ports.JobQueue
	logFn         func(level, msg string)
	notifyFn      SchedulerNotificationFunc
	triggerUpload func(items []domain.VideoItem, channels []string) error
	isUploadingFn func() bool

	mu                sync.Mutex
	isRunning         bool
	lastTriggeredSlot string
	cancel            context.CancelFunc
}

func NewBackgroundSchedulerService(
	settingsRepo ports.SettingsRepository,
	scanUC *ScanVideosUseCase,
	jobQueue ports.JobQueue,
	logFn func(level, msg string),
	notifyFn SchedulerNotificationFunc,
	triggerUpload func(items []domain.VideoItem, channels []string) error,
	isUploadingFn func() bool,
) *BackgroundSchedulerService {
	return &BackgroundSchedulerService{
		settingsRepo:  settingsRepo,
		scanUC:        scanUC,
		jobQueue:      jobQueue,
		logFn:         logFn,
		notifyFn:      notifyFn,
		triggerUpload: triggerUpload,
		isUploadingFn: isUploadingFn,
	}
}

func (s *BackgroundSchedulerService) Log(level, msg string) {
	if s.logFn != nil {
		s.logFn(level, fmt.Sprintf("[Scheduler] %s", msg))
	}
}

func (s *BackgroundSchedulerService) Notify(title, message string) {
	if s.notifyFn != nil {
		s.notifyFn(title, message)
	}
	// Native OS desktop notification
	go SendDesktopNotification(title, message)
}

// Start begins the background monitoring loop
func (s *BackgroundSchedulerService) Start(parentCtx context.Context) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parentCtx)
	s.cancel = cancel
	s.isRunning = true
	s.mu.Unlock()

	s.Log("info", "Starting Background Scheduler Service (checking slots every 30s)")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.mu.Lock()
				s.isRunning = false
				s.mu.Unlock()
				s.Log("info", "Background Scheduler Service stopped")
				return
			case now := <-ticker.C:
				s.checkAndTrigger(now)
			}
		}
	}()
}

// Stop stops the scheduler loop
func (s *BackgroundSchedulerService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isRunning && s.cancel != nil {
		s.cancel()
		s.isRunning = false
	}
}

func (s *BackgroundSchedulerService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

// checkAndTrigger evaluates if current time matches any configured golden hours
func (s *BackgroundSchedulerService) checkAndTrigger(now time.Time) {
	if s.settingsRepo == nil {
		return
	}

	st, err := s.settingsRepo.Load()
	if err != nil || !st.AutoUploadEnabled {
		return
	}

	if s.isUploadingFn != nil && s.isUploadingFn() {
		return
	}

	targetHours := st.PublishNowGoldenHours
	if len(targetHours) == 0 {
		targetHours = st.GoldenHours
	}
	hours := domain.ValidateAndSortHours(targetHours)
	todayStr := now.Format("2006-01-02")
	curTimeStr := now.Format("15:04")

	matched := false
	matchedHour := ""

	for _, h := range hours {
		slotKey := fmt.Sprintf("%s_%s", todayStr, h)
		s.mu.Lock()
		alreadyDone := s.lastTriggeredSlot == slotKey
		s.mu.Unlock()
		if alreadyDone {
			continue
		}

		// Check if within 2-minute trigger window
		slotTime, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", todayStr, h), now.Location())
		if err == nil {
			diff := now.Sub(slotTime)
			if diff >= 0 && diff < 2*time.Minute {
				matched = true
				matchedHour = h
				s.mu.Lock()
				s.lastTriggeredSlot = slotKey
				s.mu.Unlock()
				break
			}
		}
	}

	if !matched {
		return
	}

	s.Log("info", fmt.Sprintf("⏰ REACHED SCHEDULED SLOT: %s (%s)! Initiating Publish Now execution...", matchedHour, curTimeStr))

	// Scan folder for next video
	if s.scanUC == nil || s.triggerUpload == nil {
		return
	}

	items, err := s.scanUC.Execute(st.VideoFolder, st.DefaultTag)
	if err != nil || len(items) == 0 {
		s.Log("warn", fmt.Sprintf("No pending videos found to publish in folder: %s", st.VideoFolder))
		s.Notify("Scheduled Auto Upload", fmt.Sprintf("Reached scheduled slot %s but no videos remain in source folder!", matchedHour))
		return
	}

	// Pick first pending video and set PublishNow mode
	targetVideo := items[0]
	targetVideo.PublishMode = domain.PublishModePublishNow
	targetVideo.ScheduledDate = todayStr
	targetVideo.ScheduledTime = matchedHour
	targetVideo.GoldenHourSlot = domain.GetHourLabel(matchedHour)

	s.Log("info", fmt.Sprintf("🎬 Auto-selected video for Publish Now: %s (Size: %s)", targetVideo.CustomTitle, targetVideo.FileSizeHuman))
	s.Notify("Starting Automated Upload", fmt.Sprintf("Slot %s: Uploading '%s' to %d channels...", matchedHour, targetVideo.CustomTitle, len(st.EnabledChannels)))

	if err := s.triggerUpload([]domain.VideoItem{targetVideo}, st.EnabledChannels); err != nil {
		s.Log("error", fmt.Sprintf("Error triggering automated upload: %v", err))
	}
}

// GetNextSlotStatus returns information about the next upcoming slot
func (s *BackgroundSchedulerService) GetNextSlotStatus(st domain.Settings) (nextDate, nextTime string, remainingSec int, slotLabel string) {
	targetHours := st.PublishNowGoldenHours
	if len(targetHours) == 0 {
		targetHours = st.GoldenHours
	}
	hours := domain.ValidateAndSortHours(targetHours)
	d, h, rem := domain.GetNextScheduledSlot(hours, time.Now())
	return d, h, int(rem.Seconds()), domain.GetHourLabel(h)
}

// TriggerManual immediately triggers an auto-upload for the next video
func (s *BackgroundSchedulerService) TriggerManual(st domain.Settings) error {
	if s.isUploadingFn != nil && s.isUploadingFn() {
		return fmt.Errorf("system is already uploading another video")
	}

	if s.scanUC == nil || s.triggerUpload == nil {
		return fmt.Errorf("scan or upload service not ready")
	}

	items, err := s.scanUC.Execute(st.VideoFolder, st.DefaultTag)
	if err != nil || len(items) == 0 {
		return fmt.Errorf("no videos found in folder %s", st.VideoFolder)
	}

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04")

	targetVideo := items[0]
	targetVideo.PublishMode = domain.PublishModePublishNow
	targetVideo.ScheduledDate = todayStr
	targetVideo.ScheduledTime = timeStr
	targetVideo.GoldenHourSlot = "Manual (Immediate)"

	s.Log("info", fmt.Sprintf("⚡ Manual instant upload triggered: %s", targetVideo.CustomTitle))
	s.Notify("Instant Upload Triggered", fmt.Sprintf("Uploading video: %s", targetVideo.CustomTitle))

	return s.triggerUpload([]domain.VideoItem{targetVideo}, st.EnabledChannels)
}

// SendDesktopNotification triggers OS desktop notification
func SendDesktopNotification(title, message string) {
	switch runtime.GOOS {
	case "linux":
		_ = exec.Command("notify-send", "-a", "UpTik", "-i", "video-x-generic", title, message).Run()
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, message, title)
		_ = exec.Command("osascript", "-e", script).Run()
	}
}
