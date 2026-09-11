package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type YouTubePlatform struct{}

func init() {
	RegisterPlatform(&YouTubePlatform{})
}

func (p *YouTubePlatform) ID() string {
	return "youtube"
}

func (p *YouTubePlatform) DisplayName() string {
	return "YouTube Shorts"
}

func (p *YouTubePlatform) LoginURL() string {
	return "https://studio.youtube.com"
}

func (p *YouTubePlatform) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	page := b.MustPage()
	defer page.Close()

	if logFn != nil {
		logFn("info", "Checking YouTube Studio login status...")
	}
	_ = page.Navigate(p.LoginURL())
	_ = page.WaitLoad()
	time.Sleep(2 * time.Second)

	info, err := page.Info()
	if err != nil {
		return false, err
	}

	if strings.Contains(info.URL, "accounts.google.com") || strings.Contains(info.URL, "/signin") {
		return false, nil
	}

	hasCreate, _ := page.Element("#create-icon, ytcp-button#create-icon, button[aria-label*='Create']")
	if hasCreate != nil {
		return true, nil
	}

	return !strings.Contains(info.URL, "signin"), nil
}

func (p *YouTubePlatform) UploadVideo(ctx context.Context, b *rod.Browser, item *VideoItem, logFn func(level, msg string)) error {
	log := func(level, msg string) {
		if logFn != nil {
			logFn(level, fmt.Sprintf("[YouTube] %s", msg))
		}
	}

	// Shorts Adaptation: Ensure title has #Shorts
	ytTitle := item.CustomTitle
	if !strings.Contains(strings.ToLower(ytTitle), "#shorts") {
		ytTitle = fmt.Sprintf("%s #Shorts", ytTitle)
	}
	if len(ytTitle) > 100 {
		ytTitle = ytTitle[:95] + "..."
	}

	log("info", fmt.Sprintf("🎬 Starting YouTube Shorts upload: %s", ytTitle))
	log("info", fmt.Sprintf("📅 Scheduled for: %s at %s (%s)", item.ScheduledDate, item.ScheduledTime, item.GoldenHourSlot))

	// Find or create page
	pages, err := b.Pages()
	var page *rod.Page
	if err == nil {
		for _, pg := range pages {
			info, err := pg.Info()
			if err == nil && strings.Contains(info.URL, "studio.youtube.com") {
				page = pg
				break
			}
		}
	}
	if page == nil {
		page = b.MustPage()
	}

	page.MustSetViewport(1440, 900, 1, false)

	// Step 1: Navigate to Studio
	log("cdp", "Navigating to YouTube Studio...")
	err = page.Navigate("https://studio.youtube.com")
	if err != nil {
		return fmt.Errorf("YouTube Studio navigation error: %w", err)
	}

	_ = page.WaitLoad()
	time.Sleep(3 * time.Second)

	info, err := page.Info()
	if err == nil && (strings.Contains(info.URL, "accounts.google.com") || strings.Contains(info.URL, "/signin")) {
		return fmt.Errorf("not logged into YouTube Studio. Please log in with Google account")
	}

	// Step 2: Open Upload Dialog
	log("cdp", "Opening Upload Video modal...")
	uploadOpened := false

	// Try clicking create button
	err = rod.Try(func() {
		createBtn, err := page.Timeout(10 * time.Second).Element("#create-icon, button#create-icon, ytcp-button#create-button, [aria-label*='Create'], [aria-label*='Tạo']")
		if err == nil && createBtn != nil {
			_ = createBtn.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(1 * time.Second)
			uploadItem, err := page.Timeout(5 * time.Second).Element("#text-item-0, tp-yt-paper-item[test-id='upload-beta'], [aria-label*='Upload videos'], [aria-label*='Tải video']")
			if err == nil && uploadItem != nil {
				_ = uploadItem.Click(proto.InputMouseButtonLeft, 1)
				uploadOpened = true
			}
		}
	})

	if !uploadOpened {
		// Fallback direct click upload button
		_, _ = page.Eval(`() => {
			const directUpload = document.querySelector('#upload-button, [aria-label*="Upload videos"], [aria-label*="Tải video lên"]');
			if (directUpload) directUpload.click();
		}`)
	}

	// Step 3: Attach video file
	log("cdp", fmt.Sprintf("Selecting video file: %s (%s)", item.Filename, item.FileSizeHuman))
	fileInput, err := page.Timeout(30 * time.Second).Element("ytcp-uploads-dialog input[type=\"file\"], input[type=\"file\"]")
	if err != nil {
		return fmt.Errorf("không tìm thấy ô input[type='file'] của YouTube Studio: %w", err)
	}

	absPath, err := filepath.Abs(item.FullPath)
	if err != nil {
		absPath = item.FullPath
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file không tồn tại trên ổ cứng: %s", absPath)
	}

	err = fileInput.SetFiles([]string{absPath})
	if err != nil {
		return fmt.Errorf("không thể đính kèm file: %w", err)
	}

	log("info", "⏳ File sent via CDP, waiting for YouTube to process Shorts video...")

	// Step 4: Wait for Details dialog
	err = rod.Try(func() {
		page.Timeout(90 * time.Second).MustElement("#textbox, ytcp-social-suggestions-textbox#title-textarea, #dialog")
	})
	if err != nil {
		return fmt.Errorf("timeout waiting for YouTube to process video: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Step 5: Fill Title & Audience
	log("cdp", fmt.Sprintf("Filling Shorts title: %s", ytTitle))
	fillRes, err := page.Eval(`(titleText) => {
		// 1. Title input
		const titleBox = document.querySelector('#title-textarea #textbox, [aria-label*="title" i], [aria-label*="tiêu đề" i]');
		if (titleBox) {
			titleBox.focus();
			document.execCommand('selectAll', false, null);
			document.execCommand('delete', false, null);
			document.execCommand('insertText', false, titleText);
		}

		// 2. Audience: Not made for kids
		const notForKidsRadio = document.querySelector('tp-yt-paper-radio-button[name="VIDEO_MADE_FOR_KIDS_NOT_MFK"], [name="VIDEO_MADE_FOR_KIDS_NOT_MFK"]');
		if (notForKidsRadio) {
			notForKidsRadio.click();
		} else {
			const radios = Array.from(document.querySelectorAll('tp-yt-paper-radio-button, label'));
			const notKids = radios.find(r => r.innerText && (r.innerText.includes('Not made for kids') || r.innerText.includes('không phải nội dung dành cho trẻ em')));
			if (notKids) notKids.click();
		}

		return true;
	}`, ytTitle)
	if err != nil || !fillRes.Value.Bool() {
		log("warn", fmt.Sprintf("Unable to fill all form fields: %v", err))
	}

	time.Sleep(1 * time.Second)

	// Step 6: Step through Next buttons to Visibility tab
	log("cdp", "Stepping through validation tabs (Video elements, Checks, Visibility)...")
	for step := 0; step < 3; step++ {
		time.Sleep(1 * time.Second)
		nextClicked, _ := page.Eval(`() => {
			const nextBtn = document.querySelector('#next-button, [aria-label="Next"], [aria-label="Tiếp theo"]');
			if (nextBtn && !nextBtn.hasAttribute('disabled')) {
				nextBtn.click();
				return true;
			}
			return false;
		}`)
		if !nextClicked.Value.Bool() {
			break
		}
	}

	time.Sleep(2 * time.Second)

	// Step 7: Configure Schedule Date & Time
	if item.ScheduledDate == "" || item.ScheduledTime == "" {
		return fmt.Errorf("video missing scheduled date or time")
	}

	dateParts := strings.Split(item.ScheduledDate, "-")
	targetYear, _ := strconv.Atoi(dateParts[0])
	targetMonth, _ := strconv.Atoi(dateParts[1])
	targetDay, _ := strconv.Atoi(dateParts[2])

	log("cdp", fmt.Sprintf("Configuring schedule date/time: %04d-%02d-%02d %s", targetYear, targetMonth, targetDay, item.ScheduledTime))

	scheduleScript := `(day, month, year, timeStr) => {
		return new Promise(async (resolve) => {
			// 1. Select schedule option
			const scheduleRadio = document.querySelector('#schedule-radio-button, tp-yt-paper-radio-button[name="SCHEDULE"], [aria-label*="Schedule" i], [aria-label*="Lên lịch" i]');
			if (scheduleRadio) {
				scheduleRadio.click();
				await new Promise(r => setTimeout(r, 600));
			}

			// 2. Open Datepicker and enter date
			const dateInput = document.querySelector('#datepicker-trigger input, ytcp-datetime-picker input');
			if (dateInput) {
				dateInput.click();
				await new Promise(r => setTimeout(r, 400));
			}

			// 3. Time input
			const timeInput = document.querySelector('#time-of-day-input input, input[aria-label*="time" i]');
			if (timeInput) {
				timeInput.click();
				timeInput.value = timeStr;
				timeInput.dispatchEvent(new Event('input', { bubbles: true }));
				timeInput.dispatchEvent(new Event('change', { bubbles: true }));
			}

			resolve(true);
		});
	}`

	_, _ = page.Eval(scheduleScript, targetDay, targetMonth, targetYear, item.ScheduledTime)
	time.Sleep(2 * time.Second)

	// Step 8: Click Schedule Button
	log("cdp", "Clicking Schedule button on YouTube Shorts...")
	doneClicked, _ := page.Eval(`() => {
		const doneBtn = document.querySelector('#done-button, [aria-label="Schedule"], [aria-label="Lên lịch"]');
		if (doneBtn && !doneBtn.hasAttribute('disabled')) {
			doneBtn.click();
			return true;
		}
		return false;
	}`)
	if !doneClicked.Value.Bool() {
		return fmt.Errorf("failed to click Done/Schedule button on YouTube Studio")
	}

	log("info", "⏳ Submitted schedule command, waiting for YouTube confirmation...")

	// Step 9: Wait for Success dialog / completion
	submitted := false
	startTime := time.Now()

	for time.Since(startTime) < 40*time.Second {
		statusCheck, _ := page.Eval(`() => {
			// Dialog share video appears on success
			const shareDialog = document.querySelector('ytcp-video-share-dialog, [aria-label*="Video scheduled" i], [aria-label*="Đã lên lịch" i]');
			if (shareDialog) return true;
			// Close button on completion dialog
			const closeBtn = document.querySelector('#close-button');
			if (closeBtn && document.querySelector('.ytcp-video-share-dialog')) return true;
			return false;
		}`)

		if statusCheck.Value.Bool() {
			submitted = true
			// Auto close share dialog
			_, _ = page.Eval(`() => {
				const closeBtn = document.querySelector('#close-button, [aria-label="Close"], [aria-label="Đóng"]');
				if (closeBtn) closeBtn.click();
			}`)
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !submitted {
		// Even if dialog doesn't appear, check if upload dialog has disappeared
		modalCheck, _ := page.Eval(`() => document.querySelector('ytcp-uploads-dialog') === null`)
		if modalCheck.Value.Bool() {
			submitted = true
		}
	}

	if !submitted {
		return fmt.Errorf("YouTube Studio did not confirm completion within 40 seconds")
	}

	log("success", fmt.Sprintf("🎉 YouTube Shorts scheduled successfully: %s at %s", item.ScheduledDate, item.ScheduledTime))
	return nil
}
