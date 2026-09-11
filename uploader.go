package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Uploader struct {
	browser  *rod.Browser
	settings Settings
	logFn    func(level, msg string)
}

func NewUploader(settings Settings, logFn func(level, msg string)) *Uploader {
	return &Uploader{
		settings: settings,
		logFn:    logFn,
	}
}

func (u *Uploader) Log(level, msg string) {
	if u.logFn != nil {
		u.logFn(level, msg)
	}
}

// ConnectOrLaunch connects to an active Chrome CDP or starts a new instance
func (u *Uploader) ConnectOrLaunch() error {
	cdpURL := fmt.Sprintf("http://127.0.0.1:%d", u.settings.CdpPort)
	u.Log("cdp", fmt.Sprintf("Checking connection to Chrome CDP at %s...", cdpURL))

	b := rod.New().ControlURL(cdpURL)
	err := b.Connect()
	if err == nil {
		u.Log("success", "Successfully connected to running Chrome CDP instance.")
		u.browser = b
		return nil
	}

	u.Log("info", fmt.Sprintf("Launching new Chrome instance with profile: %s", u.settings.ChromeUserDataDir))
	_ = os.MkdirAll(u.settings.ChromeUserDataDir, 0755)

	l := launcher.New().
		Bin(u.settings.ChromePath).
		UserDataDir(u.settings.ChromeUserDataDir).
		Headless(u.settings.Headless).
		Set("no-sandbox").
		Set("disable-setuid-sandbox").
		Set("disable-blink-features", "AutomationControlled").
		Set("remote-debugging-port", fmt.Sprintf("%d", u.settings.CdpPort))

	launchURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}

	u.browser = rod.New().ControlURL(launchURL).MustConnect()
	u.Log("success", "Chrome launched and CDP connected successfully.")
	return nil
}

func (u *Uploader) Close() {
	if u.browser != nil {
		_ = u.browser.Close()
		u.browser = nil
	}
}

func (u *Uploader) CheckLoginStatus(platformID string) (bool, error) {
	if platformID == "" {
		platformID = "tiktok"
	}
	if u.browser == nil {
		if err := u.ConnectOrLaunch(); err != nil {
			return false, err
		}
	}

	platform, err := GetPlatform(platformID)
	if err != nil {
		return false, err
	}

	return platform.CheckLogin(context.Background(), u.browser, u.logFn)
}

func (u *Uploader) OpenPlatformLogin(platformID string) error {
	if platformID == "" {
		platformID = "tiktok"
	}
	if u.browser == nil {
		if err := u.ConnectOrLaunch(); err != nil {
			return err
		}
	}

	platform, err := GetPlatform(platformID)
	if err != nil {
		return err
	}

	page := u.browser.MustPage()
	u.Log("info", fmt.Sprintf("Opening platform dashboard: %s (%s)", platform.DisplayName(), platform.LoginURL()))
	return page.Navigate(platform.LoginURL())
}

// UploadSingleVideo orchestrates sequential upload across all selected channels
func (u *Uploader) UploadSingleVideo(ctx context.Context, item *VideoItem, targetChannels []string) error {
	if len(targetChannels) == 0 {
		targetChannels = u.settings.EnabledChannels
	}
	if len(targetChannels) == 0 {
		targetChannels = []string{"tiktok"}
	}

	if item.Channels == nil {
		item.Channels = make(map[string]ChannelStatus)
	}

	if u.browser == nil {
		if err := u.ConnectOrLaunch(); err != nil {
			return err
		}
	}

	// Auto-dismiss dialogs
	go u.browser.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
		_ = proto.PageHandleJavaScriptDialog{Accept: true}.Call(u.browser)
	})()

	u.Log("info", fmt.Sprintf("🚀 [OMNICHANNEL] Starting upload job: %s (%d channels: %s)", item.CustomTitle, len(targetChannels), strings.Join(targetChannels, ", ")))

	var successfulChannels []string
	var failedChannels []string

	for idx, ch := range targetChannels {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		platform, err := GetPlatform(ch)
		if err != nil {
			u.Log("error", fmt.Sprintf("Skipping invalid platform %s: %v", ch, err))
			continue
		}

		u.Log("info", fmt.Sprintf("▶️ [%d/%d] Uploading to %s...", idx+1, len(targetChannels), platform.DisplayName()))
		item.Channels[ch] = ChannelStatus{Status: "uploading"}

		uploadErr := platform.UploadVideo(ctx, u.browser, item, u.logFn)
		if uploadErr != nil {
			u.Log("error", fmt.Sprintf("❌ Upload failed on %s: %v", platform.DisplayName(), uploadErr))
			item.Channels[ch] = ChannelStatus{
				Status:   "error",
				ErrorMsg: uploadErr.Error(),
			}
			failedChannels = append(failedChannels, ch)
		} else {
			u.Log("success", fmt.Sprintf("✅ Upload succeeded on %s!", platform.DisplayName()))
			item.Channels[ch] = ChannelStatus{
				Status:     "scheduled",
				UploadedAt: time.Now().Format("15:04:05"),
			}
			successfulChannels = append(successfulChannels, ch)
		}

		// Courtesy delay between platforms if there's another channel
		if idx < len(targetChannels)-1 {
			u.Log("info", "⏳ Pausing 5s before next channel to release resources...")
			time.Sleep(5 * time.Second)
		}
	}

	// Record to history if at least one succeeded
	if len(successfulChannels) > 0 {
		history := LoadHistory()
		history = append(history, HistoryRecord{
			Filename:      item.Filename,
			Title:         item.CustomTitle,
			ScheduledDate: item.ScheduledDate,
			ScheduledTime: item.ScheduledTime,
			Channels:      successfulChannels,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		})
		_ = SaveHistoryToFile(history)
	}

	// If all target channels succeeded: safely move to uploaded/
	if len(failedChannels) == 0 && len(successfulChannels) > 0 {
		item.Status = "scheduled"
		item.UploadedAt = time.Now().Format("15:04:05")
		folder := filepath.Dir(item.FullPath)
		uploadedDir := filepath.Join(folder, "uploaded")
		_ = os.MkdirAll(uploadedDir, 0755)

		destPath := filepath.Join(uploadedDir, item.Filename)
		err := os.Rename(item.FullPath, destPath)
		if err != nil {
			u.Log("warn", fmt.Sprintf("Warning: could not move uploaded file: %v", err))
		} else {
			u.Log("success", fmt.Sprintf("📁 Video safely archived to: uploaded/%s", item.Filename))
		}
		return nil
	}

	if len(successfulChannels) > 0 && len(failedChannels) > 0 {
		item.Status = "partial"
		item.UploadedAt = time.Now().Format("15:04:05")
		return fmt.Errorf("thành công %s, thất bại trên: %s", strings.Join(successfulChannels, ", "), strings.Join(failedChannels, ", "))
	}

	item.Status = "error"
	return fmt.Errorf("thất bại trên toàn bộ các kênh đã chọn: %s", strings.Join(failedChannels, ", "))
}
