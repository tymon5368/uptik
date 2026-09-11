package youtube

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"uptik/internal/adapters/platforms"
	"uptik/internal/domain"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type YouTubeUploader struct{}

func NewYouTubeUploader() *YouTubeUploader {
	return &YouTubeUploader{}
}

func (p *YouTubeUploader) ID() string {
	return "youtube"
}

func (p *YouTubeUploader) DisplayName() string {
	return "YouTube Shorts"
}

func (p *YouTubeUploader) LoginURL() string {
	return "https://studio.youtube.com"
}

func (p *YouTubeUploader) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
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

	hasCreate, _ := page.Element("#create-icon, ytcp-button#create-icon, button[aria-label*='Create' i], button[aria-label*='Tạo' i]")
	if hasCreate != nil {
		return true, nil
	}

	return !strings.Contains(info.URL, "signin"), nil
}

func (p *YouTubeUploader) UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error {
	log := func(level, msg string) {
		if logFn != nil {
			logFn(level, fmt.Sprintf("[YouTube] %s", msg))
		}
	}

	ytTitle := item.CustomTitle
	if !strings.Contains(strings.ToLower(ytTitle), "#shorts") {
		ytTitle = fmt.Sprintf("%s #Shorts", ytTitle)
	}
	if len(ytTitle) > 100 {
		ytTitle = ytTitle[:95] + "..."
	}

	log("info", fmt.Sprintf("🎬 Starting YouTube Shorts upload: %s", ytTitle))

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
	if err := page.Navigate(p.LoginURL()); err != nil {
		return fmt.Errorf("YouTube Studio navigation error: %w", err)
	}

	_ = page.WaitLoad()
	time.Sleep(3 * time.Second)

	// Check Circuit Breaker
	pageHTML, _ := page.HTML()
	if cbErr := platforms.CheckCircuitBreaker(pageHTML); cbErr != nil {
		return cbErr
	}

	// Step 2: Open Upload Dialog with Fallback Matrix
	log("cdp", "Opening Upload Video modal...")
	uploadOpened := false

	createSelectors := []string{
		"#create-icon",
		"button#create-icon",
		"ytcp-button#create-button",
		"[aria-label*='Create' i]",
		"[aria-label*='Tạo' i]",
	}

	for _, sel := range createSelectors {
		createBtn, err := page.Timeout(5 * time.Second).Element(sel)
		if err == nil && createBtn != nil {
			_ = createBtn.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(1 * time.Second)

			uploadItem, err := page.Timeout(5 * time.Second).Element("#text-item-0, tp-yt-paper-item[test-id='upload-beta'], [aria-label*='Upload videos' i], [aria-label*='Tải video' i]")
			if err == nil && uploadItem != nil {
				_ = uploadItem.Click(proto.InputMouseButtonLeft, 1)
				uploadOpened = true
				break
			}
		}
	}

	if !uploadOpened {
		_, _ = page.Eval(`() => {
			const directUpload = document.querySelector('#upload-button, [aria-label*="Upload videos" i], [aria-label*="Tải video lên" i]');
			if (directUpload) directUpload.click();
		}`)
	}

	// Step 3: Attach video file with Selector Fallback Matrix
	log("cdp", fmt.Sprintf("Selecting video file: %s (%s)", item.Filename, item.FileSizeHuman))
	fileInputSelectors := []string{
		"ytcp-uploads-dialog input[type=\"file\"]",
		"input[type=\"file\"][accept*=\"video\"]",
		"input[type=\"file\"]",
	}

	var fileInput *rod.Element
	for _, sel := range fileInputSelectors {
		el, err := page.Timeout(30 * time.Second).Element(sel)
		if err == nil && el != nil {
			fileInput = el
			break
		}
	}

	if fileInput == nil {
		return fmt.Errorf("không tìm thấy ô input[type='file'] của YouTube Studio")
	}

	absPath, err := filepath.Abs(item.FullPath)
	if err != nil {
		absPath = item.FullPath
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file không tồn tại trên ổ cứng: %s", absPath)
	}

	if err := fileInput.SetFiles([]string{absPath}); err != nil {
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
	_, _ = page.Eval(`(titleText) => {
		const titleBox = document.querySelector('#title-textarea #textbox, [aria-label*="title" i], [aria-label*="tiêu đề" i]');
		if (titleBox) {
			titleBox.focus();
			document.execCommand('selectAll', false, null);
			document.execCommand('delete', false, null);
			document.execCommand('insertText', false, titleText);
		}

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

	time.Sleep(1 * time.Second)

	// Step 6: Step through Next buttons to Visibility tab
	log("cdp", "Stepping through validation tabs (Checks, Visibility)...")
	for step := 0; step < 3; step++ {
		time.Sleep(1 * time.Second)
		nextClicked, _ := page.Eval(`() => {
			const nextBtn = document.querySelector('#next-button, [aria-label="Next" i], [aria-label="Tiếp theo" i]');
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

	isPublishNow := item.PublishMode == domain.PublishModePublishNow

	if isPublishNow {
		log("cdp", "Setting visibility: Public (Publish Now)...")
		publicScript := `() => {
			return new Promise(async (resolve) => {
				const publicRadio = document.querySelector('tp-yt-paper-radio-button[name="PUBLIC"], #first-container tp-yt-paper-radio-button[name="PUBLIC"], [aria-label*="Public" i], [aria-label*="Công khai" i]');
				if (publicRadio) {
					publicRadio.click();
					await new Promise(r => setTimeout(r, 600));
				}
				resolve(true);
			});
		}`
		_, _ = page.Eval(publicScript)
		time.Sleep(2 * time.Second)
	} else {
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
				const scheduleRadio = document.querySelector('#schedule-radio-button, tp-yt-paper-radio-button[name="SCHEDULE"], [aria-label*="Schedule" i], [aria-label*="Lên lịch" i]');
				if (scheduleRadio) {
					scheduleRadio.click();
					await new Promise(r => setTimeout(r, 600));
				}

				const dateInput = document.querySelector('#datepicker-trigger input, ytcp-datetime-picker input');
				if (dateInput) {
					dateInput.click();
					await new Promise(r => setTimeout(r, 400));
				}

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
	}

	// Step 8: Click Done / Publish / Schedule Button
	if isPublishNow {
		log("cdp", "Clicking Publish / Done on YouTube Shorts...")
	} else {
		log("cdp", "Clicking Schedule button on YouTube Shorts...")
	}
	doneClicked, _ := page.Eval(`() => {
		const doneBtn = document.querySelector('#done-button, [aria-label="Publish" i], [aria-label="Xuất bản" i], [aria-label="Schedule" i], [aria-label="Lên lịch" i], [aria-label="Save" i], [aria-label="Lưu" i]');
		if (doneBtn && !doneBtn.hasAttribute('disabled')) {
			doneBtn.click();
			return true;
		}
		return false;
	}`)
	if !doneClicked.Value.Bool() {
		return fmt.Errorf("failed to click Done/Publish button on YouTube Studio")
	}

	log("info", "⏳ Submitted publish/schedule command, waiting for YouTube confirmation...")

	// Step 9: Wait for Success dialog
	submitted := false
	startTime := time.Now()

	for time.Since(startTime) < 40*time.Second {
		statusCheck, _ := page.Eval(`() => {
			const shareDialog = document.querySelector('ytcp-video-share-dialog, [aria-label*="Video scheduled" i], [aria-label*="Đã lên lịch" i], [aria-label*="Video published" i], [aria-label*="Đã xuất bản" i]');
			if (shareDialog) return true;
			const closeBtn = document.querySelector('#close-button');
			if (closeBtn && document.querySelector('.ytcp-video-share-dialog')) return true;
			return false;
		}`)

		if statusCheck.Value.Bool() {
			submitted = true
			_, _ = page.Eval(`() => {
				const closeBtn = document.querySelector('#close-button, [aria-label="Close" i], [aria-label="Đóng" i]');
				if (closeBtn) closeBtn.click();
			}`)
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !submitted {
		modalCheck, _ := page.Eval(`() => document.querySelector('ytcp-uploads-dialog') === null`)
		if modalCheck.Value.Bool() {
			submitted = true
		}
	}

	if !submitted {
		return fmt.Errorf("YouTube Studio did not confirm completion within 40 seconds")
	}

	if isPublishNow {
		log("success", fmt.Sprintf("🎉 YouTube Shorts published successfully (Publish Now): %s", item.CustomTitle))
	} else {
		log("success", fmt.Sprintf("🎉 YouTube Shorts scheduled successfully: %s at %s", item.ScheduledDate, item.ScheduledTime))
	}
	return nil
}
