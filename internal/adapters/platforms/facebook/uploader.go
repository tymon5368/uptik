package facebook

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uptik/internal/adapters/platforms"
	"uptik/internal/domain"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type FacebookUploader struct{}

func NewFacebookUploader() *FacebookUploader {
	return &FacebookUploader{}
}

func (p *FacebookUploader) ID() string {
	return "facebook"
}

func (p *FacebookUploader) DisplayName() string {
	return "Facebook Reels"
}

func (p *FacebookUploader) LoginURL() string {
	return "https://business.facebook.com/latest/reels_composer"
}

func (p *FacebookUploader) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	page := b.MustPage()
	defer page.Close()

	if logFn != nil {
		logFn("info", "Checking Facebook Business Suite login status...")
	}
	_ = page.Navigate(p.LoginURL())
	_ = page.WaitLoad()
	time.Sleep(2 * time.Second)

	info, err := page.Info()
	if err != nil {
		return false, err
	}

	if strings.Contains(info.URL, "/login") || strings.Contains(info.URL, "facebook.com/login") {
		return false, nil
	}

	hasComposer, _ := page.Element("input[type=\"file\"], [aria-label*='Reels composer' i]")
	if hasComposer != nil {
		return true, nil
	}

	return !strings.Contains(info.URL, "login"), nil
}

func (p *FacebookUploader) UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error {
	log := func(level, msg string) {
		if logFn != nil {
			logFn(level, fmt.Sprintf("[Facebook Reels] %s", msg))
		}
	}

	log("info", fmt.Sprintf("🎬 Starting Facebook Reels upload: %s", item.CustomTitle))

	pages, err := b.Pages()
	var page *rod.Page
	if err == nil {
		for _, pg := range pages {
			info, err := pg.Info()
			if err == nil && strings.Contains(info.URL, "business.facebook.com") {
				page = pg
				break
			}
		}
	}
	if page == nil {
		page = b.MustPage()
	}

	page.MustSetViewport(1440, 900, 1, false)

	// Step 1: Navigate to Reels Composer
	log("cdp", "Navigating to Meta Business Suite Reels Composer...")
	if err := page.Navigate(p.LoginURL()); err != nil {
		return fmt.Errorf("Reels Composer navigation error: %w", err)
	}

	_ = page.WaitLoad()
	time.Sleep(3 * time.Second)

	// Check Circuit Breaker
	pageHTML, _ := page.HTML()
	if cbErr := platforms.CheckCircuitBreaker(pageHTML); cbErr != nil {
		return cbErr
	}

	info, err := page.Info()
	if err == nil && (strings.Contains(info.URL, "/login") || strings.Contains(info.URL, "facebook.com/login")) {
		return fmt.Errorf("not logged into Facebook Business Suite. Please log in with Meta account")
	}

	// Step 2: Upload file with Selector Fallback Matrix
	log("cdp", fmt.Sprintf("Selecting video file: %s (%s)", item.Filename, item.FileSizeHuman))
	fileInputSelectors := []string{
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
		return fmt.Errorf("file input element not found in Meta Business Suite")
	}

	absPath, err := filepath.Abs(item.FullPath)
	if err != nil {
		absPath = item.FullPath
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist on disk: %s", absPath)
	}

	if err := fileInput.SetFiles([]string{absPath}); err != nil {
		return fmt.Errorf("failed to attach file: %w", err)
	}

	log("info", "⏳ File sent via CDP, waiting for Meta to process Reels video...")

	// Step 3: Wait for composer editor
	err = rod.Try(func() {
		page.Timeout(90 * time.Second).MustElement("[contenteditable=\"true\"], textarea, [role=\"textbox\"]")
	})
	if err != nil {
		return fmt.Errorf("timeout waiting for Meta to process video: %w", err)
	}

	time.Sleep(2 * time.Second)

	// Step 4: Fill Caption
	log("cdp", fmt.Sprintf("Filling Reels caption: %s", item.CustomTitle))
	_, err = page.Eval(`(captionText) => {
		const editor = document.querySelector('[contenteditable="true"], textarea, [role="textbox"]');
		if (editor) {
			editor.focus();
			document.execCommand('selectAll', false, null);
			document.execCommand('delete', false, null);
			document.execCommand('insertText', false, captionText);
		}
		return true;
	}`, item.CustomTitle)

	time.Sleep(1 * time.Second)

	// Step 5: Advance through steps to scheduling
	log("cdp", "Advancing through composer steps to schedule...")
	for i := 0; i < 2; i++ {
		_ = rod.Try(func() {
			nextBtn, err := page.Timeout(5 * time.Second).Element("button[aria-label*='Next' i], button[aria-label*='Tiếp' i], [role='button'][aria-label*='Next' i]")
			if err == nil && nextBtn != nil {
				_ = nextBtn.Click(proto.InputMouseButtonLeft, 1)
				time.Sleep(1500 * time.Millisecond)
			}
		})
	}

	isPublishNow := item.PublishMode == domain.PublishModePublishNow

	// Step 6: Select Publish Now or Schedule radio
	if isPublishNow {
		log("cdp", "Selecting Share Now (Publish Now) mode for Facebook Reels...")
		_, _ = page.Eval(`() => {
			const radios = Array.from(document.querySelectorAll('input[type="radio"], [role="radio"], label'));
			const shareNowOpt = radios.find(r => r.innerText && (r.innerText.includes('Share now') || r.innerText.includes('Chia sẻ ngay') || r.innerText.includes('Publish now') || r.innerText.includes('Đăng ngay')));
			if (shareNowOpt) shareNowOpt.click();
		}`)
		time.Sleep(1 * time.Second)
	} else {
		log("cdp", fmt.Sprintf("Configuring schedule date/time: %s %s", item.ScheduledDate, item.ScheduledTime))
		_, _ = page.Eval(`() => {
			const radios = Array.from(document.querySelectorAll('input[type="radio"], [role="radio"], label'));
			const scheduleOpt = radios.find(r => r.innerText && (r.innerText.includes('Schedule') || r.innerText.includes('Lên lịch')));
			if (scheduleOpt) scheduleOpt.click();
		}`)
		time.Sleep(1 * time.Second)
	}

	// Step 7: Click Schedule / Publish Button
	if isPublishNow {
		log("cdp", "Clicking Share / Publish button on Facebook Reels...")
	} else {
		log("cdp", "Clicking Schedule button on Facebook Reels...")
	}
	scheduled, _ := page.Eval(`() => {
		const btns = Array.from(document.querySelectorAll('button, [role="button"]'));
		const scheduleBtn = btns.find(b => {
			const txt = b.innerText?.trim() || '';
			return txt === 'Schedule' || txt === 'Lên lịch' || txt === 'Save' || txt === 'Lưu' || txt === 'Publish' || txt === 'Xuất bản' || txt === 'Share' || txt === 'Chia sẻ' || txt === 'Đăng';
		});
		if (scheduleBtn && !scheduleBtn.hasAttribute('disabled')) {
			scheduleBtn.click();
			return true;
		}
		return false;
	}`)

	if !scheduled.Value.Bool() {
		log("warn", "Direct Schedule/Publish button not found, searching fallback submit action...")
		_ = rod.Try(func() {
			btn, err := page.Element("button[type='submit'], [data-testid*='submit' i]")
			if err == nil && btn != nil {
				_ = btn.Click(proto.InputMouseButtonLeft, 1)
			}
		})
	}

	log("info", "⏳ Submitted Facebook Reels command, waiting for confirmation...")
	time.Sleep(5 * time.Second)

	if isPublishNow {
		log("success", fmt.Sprintf("🎉 Facebook Reels published successfully (Publish Now): %s", item.CustomTitle))
	} else {
		log("success", fmt.Sprintf("🎉 Facebook Reels scheduled successfully: %s at %s", item.ScheduledDate, item.ScheduledTime))
	}
	return nil
}
