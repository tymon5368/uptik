package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"uptik/internal/adapters/autostart"
	"uptik/internal/adapters/platforms"
	"uptik/internal/adapters/platforms/facebook"
	"uptik/internal/adapters/platforms/tiktok"
	"uptik/internal/adapters/platforms/youtube"
	"uptik/internal/adapters/queue"
	"uptik/internal/adapters/storage/sqlite"
	"uptik/internal/domain"
	"uptik/internal/ports"
	"uptik/internal/updater"
	"uptik/internal/usecases"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx               context.Context
	storage           *sqlite.Storage
	registry          *platforms.MemoryRegistry
	jobQueue          *queue.PersistentJobQueue
	scanUC            *usecases.ScanVideosUseCase
	schedUC           *usecases.ScheduleSlotsUseCase
	pipelineUC        *usecases.UploadPipelineUseCase
	schedulerSvc      *usecases.BackgroundSchedulerService
	appUpdater        *updater.Updater
	browser           *rod.Browser
	settings          domain.Settings
	mu                sync.Mutex
	isUploading       bool
	cancelUpload      context.CancelFunc
	autostartMgr      *autostart.Manager
	isQuitting        bool
	isWindowVisible   bool
	onSettingsUpdated func(s domain.Settings)
}

func NewApp() *App {
	return NewAppWithDBPath(filepath.Join(".", "uptik.db"))
}

func NewAppWithDBPath(dbPath string) *App {
	storage, err := sqlite.NewStorage(dbPath)
	if err != nil {
		fmt.Printf("Warning: failed to open SQLite: %v\n", err)
	}

	reg := platforms.NewRegistry()
	reg.Register(tiktok.NewTikTokUploader())
	reg.Register(youtube.NewYouTubeUploader())
	reg.Register(facebook.NewFacebookUploader())

	var jobQ *queue.PersistentJobQueue
	var scanUC *usecases.ScanVideosUseCase
	var schedUC *usecases.ScheduleSlotsUseCase
	if storage != nil {
		jobQ = queue.NewPersistentJobQueue(storage)
		scanUC = usecases.NewScanVideosUseCase(storage)
		schedUC = usecases.NewScheduleSlotsUseCase(storage)
	}

	st := GetDefaultSettings()
	if storage != nil {
		if loaded, err := storage.Load(); err == nil {
			st = loaded
		}
	}

	autostartMgr := autostart.NewManager("uptik", "UpTik - TikTok Auto Scheduler", appIcon)
	if autostartMgr.IsEnabled() != st.AutoStart {
		st.AutoStart = autostartMgr.IsEnabled()
	}

	return &App{
		storage:         storage,
		registry:        reg,
		jobQueue:        jobQ,
		scanUC:          scanUC,
		schedUC:         schedUC,
		appUpdater:      updater.NewUpdater(updater.DefaultRepo),
		settings:        st,
		autostartMgr:    autostartMgr,
		isWindowVisible: true,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Clean any old leftover binaries from past updates (Windows)
	updater.CleanOldBinaries()

	// Wiring Pipeline UseCase with event emitter
	a.pipelineUC = usecases.NewUploadPipelineUseCase(
		a.registry,
		a.storage,
		a.jobQueue,
		func(level, msg string) {
			runtime.EventsEmit(a.ctx, "log_entry", domain.LogEntry{
				Level:     level,
				Message:   msg,
				Timestamp: time.Now().Format("15:04:05"),
			})
		},
	)

	// Wiring BackgroundSchedulerService for automated upload at scheduled time slots
	a.schedulerSvc = usecases.NewBackgroundSchedulerService(
		a.storage,
		a.scanUC,
		a.jobQueue,
		func(level, msg string) {
			runtime.EventsEmit(a.ctx, "log_entry", domain.LogEntry{
				Level:     level,
				Message:   msg,
				Timestamp: time.Now().Format("15:04:05"),
			})
		},
		func(title, msg string) {
			runtime.EventsEmit(a.ctx, "scheduler_notification", map[string]string{
				"title":   title,
				"message": msg,
			})
		},
		func(items []domain.VideoItem, channels []string) error {
			a.StartOmnichannelUpload(items, channels)
			return nil
		},
		func() bool {
			return a.IsUploading()
		},
	)
	a.schedulerSvc.Start(a.ctx)

	// Crash Recovery: Check for in-flight tasks from prior sessions
	if a.jobQueue != nil {
		recovered, err := a.jobQueue.RecoverIncompleteJobs(context.Background())
		if err == nil && recovered > 0 {
			runtime.EventsEmit(a.ctx, "log_entry", domain.LogEntry{
				Level:     "warn",
				Message:   fmt.Sprintf("Recovered %d interrupted upload tasks from previous session.", recovered),
				Timestamp: time.Now().Format("15:04:05"),
			})
			runtime.EventsEmit(a.ctx, "queue_recovered", map[string]interface{}{
				"recoveredCount": recovered,
			})
		}
	}
}

func (a *App) GetSettings() domain.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.storage != nil {
		if s, err := a.storage.Load(); err == nil {
			a.settings = s
		}
	}
	return a.settings
}

// GetAppVersion returns the current application version
func (a *App) GetAppVersion() string {
	return Version
}

func (a *App) SaveSettings(s domain.Settings) error {
	a.mu.Lock()
	prevAutoStart := a.settings.AutoStart
	prevStartHidden := a.settings.StartHidden
	a.settings = s
	listener := a.onSettingsUpdated
	a.mu.Unlock()

	if a.autostartMgr != nil && (s.AutoStart != prevAutoStart || s.StartHidden != prevStartHidden) {
		if err := a.autostartMgr.Set(s.AutoStart, s.StartHidden); err != nil {
			fmt.Printf("Warning: failed to update autostart: %v\n", err)
		}
	}

	if listener != nil {
		listener(s)
	}

	if a.storage != nil {
		_ = a.storage.Save(s)
	}
	return SaveSettingsToFile(s)
}

func (a *App) ScanFolder(folder string) ([]domain.VideoItem, error) {
	if folder == "" {
		folder = a.settings.VideoFolder
	}
	if a.scanUC != nil {
		return a.scanUC.Execute(folder, a.settings.DefaultTag)
	}
	return ScanPendingVideos(folder, a.settings.DefaultTag)
}

func (a *App) GenerateSlots(items []domain.VideoItem, startDateStr string) []domain.VideoItem {
	startDate := time.Now()
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}
	schedHours := a.settings.ScheduleGoldenHours
	if len(schedHours) == 0 {
		schedHours = a.settings.GoldenHours
	}
	if a.schedUC != nil {
		return a.schedUC.Execute(items, schedHours, a.settings.MaxDays, startDate)
	}
	return AssignScheduleSlots(items, schedHours, a.settings.MaxDays, startDate)
}

func (a *App) GetHistory() []domain.HistoryRecord {
	if a.storage != nil {
		if records, err := a.storage.GetAll(); err == nil {
			return records
		}
	}
	return LoadHistory()
}

func (a *App) SelectFolder() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Chọn thư mục chứa video",
		DefaultDirectory: a.settings.VideoFolder,
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) OpenInFileManager(targetPath string) error {
	if targetPath == "" {
		targetPath = a.settings.VideoFolder
	}
	return exec.Command("xdg-open", targetPath).Start()
}

func (a *App) connectOrLaunchBrowser() error {
	if a.browser != nil {
		return nil
	}

	cdpURL := fmt.Sprintf("http://127.0.0.1:%d", a.settings.CdpPort)
	b := rod.New().ControlURL(cdpURL)
	if err := b.Connect(); err == nil {
		a.browser = b
		return nil
	}

	_ = os.MkdirAll(a.settings.ChromeUserDataDir, 0755)
	l := launcher.New().
		Bin(a.settings.ChromePath).
		UserDataDir(a.settings.ChromeUserDataDir).
		Headless(a.settings.Headless).
		Set("no-sandbox").
		Set("disable-setuid-sandbox").
		Set("disable-blink-features", "AutomationControlled").
		Set("remote-debugging-port", fmt.Sprintf("%d", a.settings.CdpPort))

	launchURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("không thể khởi động Chrome: %w", err)
	}

	a.browser = rod.New().ControlURL(launchURL).MustConnect()
	return nil
}

func (a *App) OpenPlatformLogin(platformID string) error {
	if platformID == "" {
		platformID = "tiktok"
	}
	if err := a.connectOrLaunchBrowser(); err != nil {
		return err
	}

	platform, err := a.registry.Get(platformID)
	if err != nil {
		return err
	}

	page := a.browser.MustPage()
	runtime.EventsEmit(a.ctx, "log_entry", domain.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Opening platform dashboard: %s (%s)", platform.DisplayName(), platform.LoginURL()),
		Timestamp: time.Now().Format("15:04:05"),
	})
	return page.Navigate(platform.LoginURL())
}

func (a *App) OpenChromeForLogin() error {
	return a.OpenPlatformLogin("tiktok")
}

func (a *App) CheckPlatformLogin(platformID string) (bool, error) {
	if platformID == "" {
		platformID = "tiktok"
	}
	if err := a.connectOrLaunchBrowser(); err != nil {
		return false, err
	}

	platform, err := a.registry.Get(platformID)
	if err != nil {
		return false, err
	}

	return platform.CheckLogin(context.Background(), a.browser, func(level, msg string) {
		runtime.EventsEmit(a.ctx, "log_entry", domain.LogEntry{
			Level:     level,
			Message:   msg,
			Timestamp: time.Now().Format("15:04:05"),
		})
	})
}

func (a *App) GetSupportedPlatforms() []domain.PlatformInfo {
	return []domain.PlatformInfo{
		{ID: "tiktok", DisplayName: "TikTok Studio", LoginURL: "https://www.tiktok.com/tiktokstudio/upload"},
		{ID: "youtube", DisplayName: "YouTube Shorts", LoginURL: "https://studio.youtube.com"},
		{ID: "facebook", DisplayName: "Facebook Reels", LoginURL: "https://business.facebook.com/latest/reels_composer"},
	}
}

func (a *App) IsUploading() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isUploading
}

func (a *App) StartUpload(queueItems []domain.VideoItem) {
	a.StartOmnichannelUpload(queueItems, a.settings.EnabledChannels)
}

func (a *App) StartOmnichannelUpload(queueItems []domain.VideoItem, channels []string) {
	a.mu.Lock()
	if a.isUploading {
		a.mu.Unlock()
		return
	}
	a.isUploading = true
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelUpload = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			a.isUploading = false
			a.cancelUpload = nil
			a.mu.Unlock()
		}()

		if len(channels) == 0 {
			channels = a.settings.EnabledChannels
		}
		if len(channels) == 0 {
			channels = []string{"tiktok"}
		}

		// Enqueue into persistent SQLite queue
		if a.jobQueue != nil {
			_, _ = a.jobQueue.EnqueueBatch(ctx, queueItems, channels)
		}

		if err := a.connectOrLaunchBrowser(); err != nil {
			runtime.EventsEmit(a.ctx, "log_entry", domain.LogEntry{
				Level:     "error",
				Message:   fmt.Sprintf("Chrome CDP connection error: %v", err),
				Timestamp: time.Now().Format("15:04:05"),
			})
			return
		}

		// Auto dismiss dialogs
		go a.browser.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
			_ = proto.PageHandleJavaScriptDialog{Accept: true}.Call(a.browser)
		})()

		runtime.EventsEmit(a.ctx, "upload_started", map[string]interface{}{
			"total":    len(queueItems),
			"channels": channels,
		})

		successCount := 0
		failCount := 0

		for {
			select {
			case <-ctx.Done():
				runtime.EventsEmit(a.ctx, "upload_cancelled", nil)
				return
			default:
			}

			// Lease next job from persistent queue
			job, err := a.jobQueue.LeaseNext(ctx)
			if err != nil || job == nil {
				break // Queue is empty
			}

			progress := domain.UploadProgress{
				CurrentIndex: successCount + failCount + 1,
				TotalVideos:  len(queueItems),
				CurrentVideo: &job.Video,
				SuccessCount: successCount,
				FailCount:    failCount,
				Status:       "uploading",
			}
			runtime.EventsEmit(a.ctx, "upload_progress", progress)

			err = a.pipelineUC.ProcessJob(ctx, a.browser, job)
			if err != nil {
				failCount++
				job.Video.Status = "error"
				job.Video.ErrorMsg = err.Error()
				runtime.EventsEmit(a.ctx, "video_error", job.Video)
				time.Sleep(2 * time.Second)
			} else {
				successCount++
				job.Video.Status = "scheduled"
				runtime.EventsEmit(a.ctx, "video_success", job.Video)
				time.Sleep(2 * time.Second)
			}
		}

		runtime.EventsEmit(a.ctx, "upload_finished", map[string]interface{}{
			"success": successCount,
			"failed":  failCount,
			"total":   len(queueItems),
		})
	}()
}

func (a *App) StopUpload() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.isUploading && a.cancelUpload != nil {
		a.cancelUpload()
		a.isUploading = false
	}
}

// GetPendingJobs returns incomplete jobs from persistent storage
func (a *App) GetPendingJobs() ([]ports.JobItem, error) {
	if a.jobQueue == nil {
		return nil, nil
	}
	return a.jobQueue.ListAll(context.Background())
}

// ResumeQueue resumes any incomplete/failed jobs in the persistent queue
func (a *App) ResumeQueue() {
	if a.jobQueue == nil {
		return
	}
	pending, err := a.jobQueue.ListAll(context.Background())
	if err != nil || len(pending) == 0 {
		return
	}

	var items []domain.VideoItem
	for _, j := range pending {
		if j.State != ports.JobStateCompleted {
			items = append(items, j.Video)
		}
	}
	if len(items) > 0 {
		a.StartOmnichannelUpload(items, a.settings.EnabledChannels)
	}
}

// CancelQueue cancels all in-flight jobs in the persistent queue
func (a *App) CancelQueue() error {
	a.StopUpload()
	if a.jobQueue != nil {
		return a.jobQueue.CancelAll(context.Background())
	}
	return nil
}

// RegisterSettingsUpdateListener registers a callback when settings are modified
func (a *App) RegisterSettingsUpdateListener(fn func(s domain.Settings)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onSettingsUpdated = fn
}

// IsCloseToTray returns whether the app should minimize to tray on close
func (a *App) IsCloseToTray() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings.CloseToTray
}

// IsQuitting returns whether the app is in the process of quitting
func (a *App) IsQuitting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isQuitting
}

// SetWindowVisible tracks visibility state
func (a *App) SetWindowVisible(v bool) {
	a.mu.Lock()
	a.isWindowVisible = v
	a.mu.Unlock()
}

// IsWindowVisible returns current visibility state
func (a *App) IsWindowVisible() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isWindowVisible
}

// ShowApp brings the window to the foreground
func (a *App) ShowApp() {
	a.mu.Lock()
	a.isWindowVisible = true
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.Show(a.ctx)
	}
}

// HideApp minimizes/hides the window to background
func (a *App) HideApp() {
	a.mu.Lock()
	a.isWindowVisible = false
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
		_ = runtime.SendNotification(a.ctx, runtime.NotificationOptions{
			Title: "UpTik đang chạy ngầm",
			Body:  "Nhấp vào biểu tượng ở khay hệ thống để mở lại giao diện.",
		})
	}
}

// ToggleAppWindow toggles between showing and hiding the window
func (a *App) ToggleAppWindow() {
	a.mu.Lock()
	vis := a.isWindowVisible
	a.mu.Unlock()
	if vis {
		a.HideApp()
	} else {
		a.ShowApp()
	}
}

// QuitApp shuts down the application completely
func (a *App) QuitApp() {
	a.mu.Lock()
	a.isQuitting = true
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// GetAutoStart returns whether autostart is enabled in OS
func (a *App) GetAutoStart() bool {
	if a.autostartMgr != nil {
		return a.autostartMgr.IsEnabled()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings.AutoStart
}

// SetAutoStart toggles autostart
func (a *App) SetAutoStart(enabled bool) error {
	a.mu.Lock()
	s := a.settings
	s.AutoStart = enabled
	a.mu.Unlock()
	return a.SaveSettings(s)
}

// GetSchedulerStatus returns the live status of the background auto-upload scheduler
func (a *App) GetSchedulerStatus() map[string]interface{} {
	a.mu.Lock()
	st := a.settings
	sched := a.schedulerSvc
	a.mu.Unlock()

	isRunning := false
	nextDate := ""
	nextTime := ""
	remainingSec := 0
	slotLabel := ""

	if sched != nil {
		isRunning = sched.IsRunning()
		nextDate, nextTime, remainingSec, slotLabel = sched.GetNextSlotStatus(st)
	}

	return map[string]interface{}{
		"isRunning":             isRunning,
		"autoUploadEnabled":     st.AutoUploadEnabled,
		"publishMode":           st.PublishMode,
		"nextDate":              nextDate,
		"nextTime":              nextTime,
		"remainingSec":          remainingSec,
		"slotLabel":             slotLabel,
		"goldenHours":           st.GoldenHours,
		"scheduleGoldenHours":   st.ScheduleGoldenHours,
		"publishNowGoldenHours": st.PublishNowGoldenHours,
	}
}

// ToggleAutoUpload enables or disables the background auto-uploader
func (a *App) ToggleAutoUpload(enabled bool) error {
	a.mu.Lock()
	s := a.settings
	s.AutoUploadEnabled = enabled
	a.settings = s
	a.mu.Unlock()

	err := a.SaveSettings(s)
	if err == nil {
		runtime.EventsEmit(a.ctx, "scheduler_toggled", map[string]interface{}{
			"autoUploadEnabled": enabled,
		})
	}
	return err
}

// SetPublishMode changes between schedule and publish_now modes
func (a *App) SetPublishMode(mode string) error {
	if mode == "" {
		mode = string(domain.PublishModeSchedule)
	}
	pMode := domain.PublishMode(mode)
	if pMode != domain.PublishModeSchedule && pMode != domain.PublishModePublishNow {
		return fmt.Errorf("unsupported publish mode: %s", mode)
	}

	a.mu.Lock()
	s := a.settings
	s.PublishMode = pMode
	if s.PublishMode == domain.PublishModePublishNow {
		if len(s.PublishNowGoldenHours) > 0 {
			s.GoldenHours = s.PublishNowGoldenHours
		}
	} else {
		if len(s.ScheduleGoldenHours) > 0 {
			s.GoldenHours = s.ScheduleGoldenHours
		}
	}
	a.settings = s
	a.mu.Unlock()

	return a.SaveSettings(s)
}

// TriggerAutoUploadNow manually triggers an immediate publish_now upload for the next video
func (a *App) TriggerAutoUploadNow() error {
	a.mu.Lock()
	st := a.settings
	sched := a.schedulerSvc
	a.mu.Unlock()

	if sched == nil {
		return fmt.Errorf("scheduler service chưa sẵn sàng")
	}
	return sched.TriggerManual(st)
}

// UpdateGoldenHours updates and sorts configured schedule hours for the active publish mode
func (a *App) UpdateGoldenHours(hours []string) error {
	valid := domain.ValidateAndSortHours(hours)
	a.mu.Lock()
	s := a.settings
	s.GoldenHours = valid
	if s.PublishMode == domain.PublishModePublishNow {
		s.PublishNowGoldenHours = valid
	} else {
		s.ScheduleGoldenHours = valid
	}
	a.settings = s
	a.mu.Unlock()

	return a.SaveSettings(s)
}

// UpdateModeGoldenHours updates hours specifically for the designated publish mode
func (a *App) UpdateModeGoldenHours(mode string, hours []string) error {
	pMode := domain.PublishMode(mode)
	if pMode != domain.PublishModeSchedule && pMode != domain.PublishModePublishNow {
		return fmt.Errorf("unsupported publish mode: %s", mode)
	}

	valid := domain.ValidateAndSortHours(hours)
	a.mu.Lock()
	s := a.settings
	if pMode == domain.PublishModePublishNow {
		s.PublishNowGoldenHours = valid
		if s.PublishMode == domain.PublishModePublishNow {
			s.GoldenHours = valid
		}
	} else {
		s.ScheduleGoldenHours = valid
		if s.PublishMode == domain.PublishModeSchedule || s.PublishMode == "" {
			s.GoldenHours = valid
		}
	}
	a.settings = s
	a.mu.Unlock()

	return a.SaveSettings(s)
}

// CheckForUpdates queries GitHub releases for an available software update.
func (a *App) CheckForUpdates() (*updater.UpdateInfo, error) {
	if a.appUpdater == nil {
		a.appUpdater = updater.NewUpdater(updater.DefaultRepo)
	}
	return a.appUpdater.CheckForUpdate(context.Background(), a.GetAppVersion())
}

// ApplyUpdate downloads, verifies, and installs the update in-place.
func (a *App) ApplyUpdate(info updater.UpdateInfo) (bool, error) {
	if a.appUpdater == nil {
		a.appUpdater = updater.NewUpdater(updater.DefaultRepo)
	}

	progressFn := func(percent int) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "update:progress", percent)
		}
	}

	if err := a.appUpdater.ApplyUpdate(context.Background(), &info, progressFn); err != nil {
		return false, err
	}
	return true, nil
}

// RestartApp relaunches the updated application and terminates the old process.
func (a *App) RestartApp() error {
	return updater.RestartApplication()
}

