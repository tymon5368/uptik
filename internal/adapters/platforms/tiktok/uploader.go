package tiktok

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

type TikTokUploader struct{}

func NewTikTokUploader() *TikTokUploader {
	return &TikTokUploader{}
}

func (p *TikTokUploader) ID() string {
	return "tiktok"
}

func (p *TikTokUploader) DisplayName() string {
	return "TikTok Studio"
}

func (p *TikTokUploader) LoginURL() string {
	return "https://www.tiktok.com/tiktokstudio/upload"
}

func (p *TikTokUploader) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	page := b.MustPage()
	defer page.Close()

	if logFn != nil {
		logFn("info", "Checking TikTok Studio login status...")
	}
	_ = page.Navigate(p.LoginURL())
	_ = page.WaitLoad()
	time.Sleep(2 * time.Second)

	info, err := page.Info()
	if err != nil {
		return false, err
	}

	if strings.Contains(info.URL, "/login") {
		return false, nil
	}

	// Selector matrix for file input
	for _, sel := range []string{"input[type=\"file\"]", "[data-e2e=\"upload-file-input\"]"} {
		hasUpload, _ := page.Element(sel)
		if hasUpload != nil {
			return true, nil
		}
	}

	return !strings.Contains(info.URL, "login"), nil
}

func (p *TikTokUploader) UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error {
	log := func(level, msg string) {
		if logFn != nil {
			logFn(level, fmt.Sprintf("[TikTok] %s", msg))
		}
	}

	log("info", fmt.Sprintf("🎬 Starting upload: %s", item.CustomTitle))
	log("info", fmt.Sprintf("📅 Scheduled for: %s at %s (%s)", item.ScheduledDate, item.ScheduledTime, item.GoldenHourSlot))

	// Find existing page or create new
	pages, err := b.Pages()
	var page *rod.Page
	if err == nil {
		for _, pg := range pages {
			info, err := pg.Info()
			if err == nil && strings.Contains(info.URL, "tiktok.com") {
				page = pg
				break
			}
		}
	}
	if page == nil {
		page = b.MustPage()
	}

	page.MustSetViewport(1440, 900, 1, false)

	// Step 1: Navigate to upload page
	log("cdp", "Navigating to TikTok Studio Upload...")
	if err := page.Navigate(p.LoginURL()); err != nil {
		return fmt.Errorf("navigation error: %w", err)
	}

	_ = page.WaitLoad()
	time.Sleep(2 * time.Second)

	// Check Circuit Breaker
	pageHTML, _ := page.HTML()
	if cbErr := platforms.CheckCircuitBreaker(pageHTML); cbErr != nil {
		return cbErr
	}

	info, err := page.Info()
	if err == nil && strings.Contains(info.URL, "/login") {
		return fmt.Errorf("not logged into TikTok Studio. Please log in via browser")
	}

	// Remove iframe blockers
	_, _ = page.Eval(`() => {
		const blockers = document.querySelectorAll('iframe[src*="verify"], .captcha-verify-container');
		blockers.forEach(el => el.remove());
	}`)

	// Step 2: Upload file with Selector Fallback Matrix
	log("cdp", fmt.Sprintf("Selecting video file: %s (%s)", item.Filename, item.FileSizeHuman))
	fileInputSelectors := []string{
		"input[type=\"file\"]",
		"[data-e2e=\"upload-file-input\"]",
		"input[accept*=\"video\"]",
	}

	var fileInput *rod.Element
	for _, sel := range fileInputSelectors {
		el, err := page.Timeout(5 * time.Second).Element(sel)
		if err == nil && el != nil {
			fileInput = el
			break
		}
	}

	if fileInput == nil {
		return fmt.Errorf("file input element not found via Selector Fallback Matrix")
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

	log("info", "⏳ File sent via CDP, waiting for TikTok to process video...")

	// Step 3: Wait for editor
	err = rod.Try(func() {
		page.Timeout(90 * time.Second).MustElement(".file-content, .upload-content, [contenteditable=\"true\"], .caption-editor")
	})
	if err != nil {
		return fmt.Errorf("timeout waiting for TikTok to process video (after 90s): %w", err)
	}

	time.Sleep(3 * time.Second)

	isPublishNow := item.PublishMode == domain.PublishModePublishNow

	if isPublishNow {
		log("cdp", fmt.Sprintf("Publish Now mode: Filling caption and preparing to post (%s)", item.CustomTitle))
		publishNowScript := `(title) => {
			return new Promise((resolve) => {
				// 1. Caption
				const editor = document.querySelector('[contenteditable="true"]');
				if (editor) {
					editor.focus();
					const sel = window.getSelection();
					const range = document.createRange();
					range.selectNodeContents(editor);
					if (sel) {
						sel.removeAllRanges();
						sel.addRange(range);
					}
					document.execCommand('delete');
					document.execCommand('insertText', false, title);
				}

				// 2. Ensure post now (default on TikTok Studio)
				const postNowRadio = document.querySelector('input[type="radio"][value="post_now"], input[type="radio"][value="direct"]');
				if (postNowRadio && !postNowRadio.checked) postNowRadio.click();

				resolve(true);
			});
		}`
		_, _ = page.Eval(publishNowScript, item.CustomTitle)
		time.Sleep(1 * time.Second)

		// 1. Chờ video tải lên hoàn tất trên máy chủ TikTok Studio
		if err := p.waitForVideoUpload(ctx, page, log); err != nil {
			return err
		}

		// 2. Chờ TikTok kiểm tra bản quyền xong trước khi bấm nút submit đăng video
		if err := p.waitForCopyrightCheck(ctx, page, log); err != nil {
			return err
		}

		// 3. Chờ nút Đăng ngay (Post Now) sẵn sàng để click
		if err := p.waitForPostButtonReady(ctx, page, log); err != nil {
			return err
		}
	} else {
		// Step 4: Parse Scheduled Date & Time
		if item.ScheduledDate == "" || item.ScheduledTime == "" {
			return fmt.Errorf("video missing scheduled date or time")
		}

		dateParts := strings.Split(item.ScheduledDate, "-")
		if len(dateParts) != 3 {
			return fmt.Errorf("invalid scheduled date format: %s", item.ScheduledDate)
		}
		targetYear, _ := strconv.Atoi(dateParts[0])
		targetMonth, _ := strconv.Atoi(dateParts[1])
		targetDay, _ := strconv.Atoi(dateParts[2])

		timeParts := strings.Split(item.ScheduledTime, ":")
		if len(timeParts) != 2 {
			return fmt.Errorf("invalid scheduled time format: %s", item.ScheduledTime)
		}
		targetHour := fmt.Sprintf("%02s", strings.TrimSpace(timeParts[0]))
		targetMin := fmt.Sprintf("%02s", strings.TrimSpace(timeParts[1]))

		log("cdp", fmt.Sprintf("Filling caption and configuring schedule: %04d-%02d-%02d %s:%s", targetYear, targetMonth, targetDay, targetHour, targetMin))

		// Step 5: Fill Caption & React 18 pickers
		evalScript := `(title, targetDay, targetMonth, targetHour, targetMin) => {
			return new Promise(async (resolve) => {
				// 1. Caption
				const editor = document.querySelector('[contenteditable="true"]');
				if (editor) {
					editor.focus();
					const sel = window.getSelection();
					const range = document.createRange();
					range.selectNodeContents(editor);
					if (sel) {
						sel.removeAllRanges();
						sel.addRange(range);
					}
					document.execCommand('delete');
					document.execCommand('insertText', false, title);
				}

				// 2. Schedule radio
				const scheduleRadio = document.querySelector('input[type="radio"][value="schedule"]');
				if (scheduleRadio && !scheduleRadio.checked) scheduleRadio.click();
				const scheduleLabel = Array.from(document.querySelectorAll('label, [class*="Radio"], span')).find(
					el => el.textContent && el.textContent.trim() === 'Schedule'
				);
				if (scheduleLabel) scheduleLabel.click();

				await new Promise(r => setTimeout(r, 450));

				// 3. Date picker
				const dateInput = document.querySelector('input[value*="2026-"], input[value*="2025-"], input[value*="2027-"]');
				if (dateInput) {
					const container = dateInput.closest('.TUXTextInputCore') || dateInput;
					container.click();
					await new Promise(r => setTimeout(r, 400));

					const monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
					const targetMonthName = monthNames[targetMonth - 1];
					let currentMonth = document.querySelector('.month-title')?.innerText?.trim();
					if (currentMonth && currentMonth !== targetMonthName) {
						const nextArrow = document.querySelectorAll('.month-header-wrapper .arrow')[1];
						if (nextArrow) {
							nextArrow.click();
							await new Promise(r => setTimeout(r, 450));
						}
					}

					const dayCells = Array.from(document.querySelectorAll('.day, .calendar-day, [class*="day-"]'));
					const targetCell = dayCells.find(el => {
						const txt = el.innerText?.trim();
						return txt === String(targetDay) && !el.classList.contains('disabled') && !el.classList.contains('prev-month') && !el.classList.contains('next-month');
					});
					if (targetCell) {
						targetCell.click();
					}
					await new Promise(r => setTimeout(r, 450));
				}

				// 4. Time picker
				const timeInput = document.querySelector('input[value*=":"]');
				if (timeInput) {
					const timeContainer = timeInput.closest('.TUXTextInputCore') || timeInput;
					timeContainer.click();
					await new Promise(r => setTimeout(r, 450));

					const hourSpan = Array.from(document.querySelectorAll('.time-picker-item, [class*="time-item"], span, li')).find(
						el => el.innerText && el.innerText.trim() === targetHour
					);
					if (hourSpan) {
						hourSpan.scrollIntoView({ block: 'nearest' });
						hourSpan.click();
					}
					await new Promise(r => setTimeout(r, 300));

					const minSpan = Array.from(document.querySelectorAll('.time-picker-item, [class*="time-item"], span, li')).find(
						el => el.innerText && el.innerText.trim() === targetMin
					);
					if (minSpan) {
						minSpan.scrollIntoView({ block: 'nearest' });
						minSpan.click();
					}
					await new Promise(r => setTimeout(r, 300));

					document.body.click();
				}

				resolve(true);
			});
		}`

		_, _ = page.Eval(evalScript, item.CustomTitle, targetDay, targetMonth, targetHour, targetMin)
		time.Sleep(1 * time.Second)

		// 1. Đảm bảo video tải lên hoàn tất trước khi bấm Schedule
		if err := p.waitForVideoUpload(ctx, page, log); err != nil {
			return err
		}
		// 2. Chờ TikTok kiểm tra bản quyền xong trước khi bấm Schedule
		if err := p.waitForCopyrightCheck(ctx, page, log); err != nil {
			return err
		}
		// 3. Chờ nút Schedule sẵn sàng để click
		if err := p.waitForPostButtonReady(ctx, page, log); err != nil {
			return err
		}
	}

	// Step 6: Click Schedule / Post Button with Selector Fallback Matrix
	if isPublishNow {
		log("cdp", "Clicking Post Now button on TikTok Studio...")
	} else {
		log("cdp", "Clicking Schedule button on TikTok Studio...")
	}
	scheduleSelectors := []string{
		"button[data-e2e=\"post_video_button\"]",
		"button[aria-label*=\"Post\" i]",
		"button[aria-label*=\"Schedule\" i]",
		"button[aria-label*=\"Đăng\" i]",
	}

	var scheduleBtn *rod.Element
	for _, sel := range scheduleSelectors {
		el, err := page.Element(sel)
		if err == nil && el != nil {
			scheduleBtn = el
			break
		}
	}

	if scheduleBtn == nil {
		return fmt.Errorf("failed to locate Post/Schedule button on TikTok Studio")
	}

	_ = scheduleBtn.ScrollIntoView()
	time.Sleep(300 * time.Millisecond)
	if err := scheduleBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("failed to click Post/Schedule button: %w", err)
	}

	if isPublishNow {
		log("info", "⏳ Clicked Post Now, awaiting TikTok Studio confirmation...")
	} else {
		log("info", "⏳ Clicked Schedule, awaiting TikTok Studio confirmation...")
	}

	// Step 7: Wait for confirmation (redirect or modal)
	submitted := false
	startTime := time.Now()

	for time.Since(startTime) < 90*time.Second {
		pageInfo, err := page.Info()
		if err == nil && (strings.Contains(pageInfo.URL, "/content") || strings.Contains(pageInfo.URL, "/manage")) {
			submitted = true
			break
		}

		// Auto confirm dialog if prompted (e.g. issues warning or final confirmation)
		res, _ := page.Eval(`() => {
			// 1. Scan for any dialog/modal container
			const dialogs = Array.from(document.querySelectorAll('[role="dialog"], .TUXModal, .TUXDialog, .modal-content, [class*="modal"], [class*="dialog"]'));
			for (const dialog of dialogs) {
				const dText = (dialog.innerText || '').toLowerCase();
				if (dText.includes('continue to post') || 
				    dText.includes('copyright') || 
				    dText.includes('tiếp tục đăng') || 
				    dText.includes('bản quyền') || 
				    dText.includes('incomplete') || 
				    dText.includes('chưa hoàn tất') || 
				    dText.includes('post anyway') || 
				    dText.includes('schedule anyway')) {
					
					const btns = Array.from(dialog.querySelectorAll('button'));
					const targetBtn = btns.find(b => {
						const t = (b.innerText || '').trim().toLowerCase();
						if (t === 'cancel' || t === 'hủy' || t === 'quay lại' || t === 'back') return false;
						return t === 'post now' || 
						       t === 'post anyway' || 
						       t === 'schedule anyway' || 
						       t === 'continue' || 
						       t === 'đăng ngay' || 
						       t === 'vẫn đăng' || 
						       t === 'tiếp tục đăng' || 
						       t === 'vẫn lên lịch';
					}) || btns.find(b => {
						const t = (b.innerText || '').trim().toLowerCase();
						if (t.includes('cancel') || t.includes('hủy')) return false;
						return t.includes('post now') || 
						       t.includes('post anyway') || 
						       t.includes('schedule anyway') || 
						       t.includes('đăng ngay') || 
						       t.includes('vẫn đăng') || 
						       t.includes('tiếp tục');
					});

					if (targetBtn) {
						targetBtn.focus();
						targetBtn.click();
						targetBtn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
						return true;
					}
				}
			}

			// 2. Global fallback: search buttons for modal-specific action text
			const btns = Array.from(document.querySelectorAll('button'));
			const fallbackBtn = btns.find(b => {
				const t = (b.innerText || '').trim().toLowerCase();
				return t === 'post now' || 
				       t === 'post anyway' || 
				       t === 'schedule anyway' || 
				       t === 'vẫn lên lịch' || 
				       t === 'đăng ngay' || 
				       t === 'vẫn đăng';
			});
			if (fallbackBtn) {
				fallbackBtn.focus();
				fallbackBtn.click();
				fallbackBtn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
				return true;
			}

			return false;
		}`)
		if res != nil && res.Value.Bool() {
			log("info", "Confirmed TikTok Studio dialog (Continue to post / Post anyway).")
			time.Sleep(1 * time.Second)
			continue
		}

		// Check modal content for success
		successModal, _ := page.Eval(`() => {
			const modals = Array.from(document.querySelectorAll('.TUXModal, [role="dialog"], .TUXDialog, .modal-content, [class*="modal"]'));
			for (const modal of modals) {
				const text = (modal.innerText || '').toLowerCase();
				// Ignore copyright / confirmation modal
				if (text.includes('continue to post') || text.includes('copyright') || text.includes('tiếp tục đăng') || text.includes('bản quyền')) {
					continue;
				}
				if (text.includes('manage your posts') || 
				    text.includes('your video has been published') || 
				    text.includes('has been uploaded') || 
				    text.includes('is uploaded') || 
				    text.includes('scheduled') || 
				    text.includes('uploaded') || 
				    text.includes('published') || 
				    text.includes('quản lý bài đăng') || 
				    text.includes('đã được đăng') || 
				    text.includes('đã lên lịch') || 
				    text.includes('tải lên video khác') || 
				    text.includes('upload another video')) {
					return true;
				}
			}
			return false;
		}`)
		if successModal != nil && successModal.Value.Bool() {
			submitted = true
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !submitted {
		return fmt.Errorf("TikTok Studio did not confirm success within 90 seconds")
	}

	if isPublishNow {
		log("success", fmt.Sprintf("🎉 TikTok published successfully (Publish Now): %s", item.CustomTitle))
	} else {
		log("success", fmt.Sprintf("🎉 TikTok scheduled successfully: %s at %s", item.ScheduledDate, item.ScheduledTime))
	}
	return nil
}

func (p *TikTokUploader) waitForVideoUpload(ctx context.Context, page *rod.Page, log func(level, msg string)) error {
	log("info", "⏳ Checking video upload progress on TikTok Studio...")
	startTime := time.Now()
	lastLogTime := time.Time{}
	consecutiveEvalErrors := 0
	wasUploading := false

	uploadEvalScript := `() => {
		// 1. Check for explicit error/rejection
		const errorSelectors = [
			'.upload-error',
			'[class*="error"]',
			'[class*="fail"]',
			'.file-error',
			'[role="alert"]'
		];
		for (const sel of errorSelectors) {
			const els = document.querySelectorAll(sel);
			for (const el of els) {
				const txt = (el.innerText || el.textContent || '').trim().toLowerCase();
				if (txt.includes('failed') || txt.includes('error') || txt.includes('not supported') || 
				    txt.includes('thất bại') || txt.includes('lỗi') || txt.includes('không được hỗ trợ')) {
					return { isUploading: false, isFailed: true, isCompleted: false, text: el.innerText.trim() };
				}
			}
		}

		// 2. Check for upload progress or percentage
		const progressSelectors = [
			'.byte-progress-bar',
			'[class*="progress"]',
			'[role="progressbar"]',
			'.upload-progress',
			'.file-info',
			'[class*="upload-status"]',
			'[class*="file-status"]',
			'.stage-item',
			'[class*="uploading"]',
			'[class*="file-item"]'
		];
		for (const sel of progressSelectors) {
			const els = document.querySelectorAll(sel);
			for (const el of els) {
				const txt = (el.innerText || el.textContent || '').trim();
				if (!txt) continue;
				const low = txt.toLowerCase();
				const percentMatch = txt.match(/(\d{1,3})\s*%/);
				if (percentMatch) {
					const pct = parseInt(percentMatch[1], 10);
					if (pct < 100) {
						return { isUploading: true, isFailed: false, isCompleted: false, text: txt };
					}
					if (pct >= 100) {
						return { isUploading: false, isFailed: false, isCompleted: true, text: txt };
					}
				}
				if ((low.includes('uploading') || low.includes('đang tải lên')) &&
					!low.includes('uploaded') && !low.includes('đã tải lên') && !low.includes('100%')) {
					return { isUploading: true, isFailed: false, isCompleted: false, text: txt };
				}
				if (low.includes('uploaded') || low.includes('đã tải lên') || low.includes('upload complete')) {
					return { isUploading: false, isFailed: false, isCompleted: true, text: txt };
				}
			}
		}

		// 3. Check for editor readiness as completion signal
		const editor = document.querySelector('[contenteditable="true"], .caption-editor');
		const filePreview = document.querySelector('.file-content, .upload-content, video, [class*="player"]');
		const hasReadyContent = !!(editor || filePreview);

		return { isUploading: false, isFailed: false, isCompleted: hasReadyContent, text: '' };
	}`

	for time.Since(startTime) < 180*time.Second {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		res, err := page.Eval(uploadEvalScript)
		if err != nil {
			consecutiveEvalErrors++
			if consecutiveEvalErrors >= 5 {
				return fmt.Errorf("repeated evaluation failures while waiting for video upload: %w", err)
			}
			time.Sleep(1 * time.Second)
			continue
		}
		consecutiveEvalErrors = 0

		if res != nil {
			isFailed := res.Value.Get("isFailed").Bool()
			isUploading := res.Value.Get("isUploading").Bool()
			isCompleted := res.Value.Get("isCompleted").Bool()
			uploadText := res.Value.Get("text").String()

			if isFailed {
				return fmt.Errorf("video upload rejected or failed on TikTok Studio: %s", uploadText)
			}

			if isUploading {
				wasUploading = true
				if time.Since(lastLogTime) >= 5*time.Second {
					msg := "⏳ Video upload in progress on TikTok Studio..."
					if uploadText != "" {
						msg = fmt.Sprintf("⏳ Video upload in progress on TikTok Studio (%s)...", uploadText)
					}
					log("info", msg)
					lastLogTime = time.Now()
				}
				time.Sleep(2 * time.Second)
				continue
			}

			if isCompleted {
				log("info", "✅ Video upload completed on TikTok Studio.")
				return nil
			}

			// If it was uploading and now neither uploading nor failed, give 2 seconds to stabilize
			if wasUploading {
				time.Sleep(2 * time.Second)
				log("info", "✅ Video upload completed on TikTok Studio.")
				return nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("video upload timed out after 180 seconds")
}

func (p *TikTokUploader) waitForCopyrightCheck(ctx context.Context, page *rod.Page, log func(level, msg string)) error {
	log("info", "⏳ Checking copyright status on TikTok Studio...")
	lastLogTime := time.Time{}
	wasChecking := false

	copyrightEvalScript := `() => {
		const allEls = Array.from(document.querySelectorAll('div, section, p, span, label, [data-e2e*="copyright"]'));
		let isChecking = false;
		let isComplete = false;
		let statusText = '';
		let foundSection = false;

		for (const el of allEls) {
			const txt = (el.innerText || '').toLowerCase();
			if (!txt.includes('copyright') && !txt.includes('bản quyền')) continue;
			if ((el.innerText || '').length > 600) continue;

			foundSection = true;

			const hasSpinner = el.querySelector(
				'[class*="loading"], [class*="spinner"], [class*="circle-loading"], svg[class*="spin"], [class*="TUXLoading"]'
			) !== null;

			const checkingKeywords = [
				'checking...',
				'checking',
				'running copyright check',
				'checking for copyright',
				'checking for issues',
				'đang kiểm tra...',
				'đang kiểm tra bản quyền',
				'đang kiểm tra',
				'quy trình kiểm tra bản quyền đang diễn ra'
			];

			const hasCheckingKeyword = checkingKeywords.some(k => txt.includes(k));

			if (hasCheckingKeyword || hasSpinner) {
				isChecking = true;
				statusText = (el.innerText || '').trim();
				break;
			}

			const completeKeywords = [
				'no issues found',
				'no copyright issues',
				'không phát hiện vấn đề',
				'không phát hiện thấy vấn đề',
				'không có vi phạm',
				'issues detected',
				'phát hiện vấn đề',
				'check complete',
				'đã hoàn tất kiểm tra',
				'kiểm tra hoàn tất',
				'đã kiểm tra xong'
			];

			if (completeKeywords.some(k => txt.includes(k))) {
				isComplete = true;
				statusText = (el.innerText || '').trim();
			}
		}

		if (!isChecking) {
			const standalone = Array.from(document.querySelectorAll('span, div, p')).find(el => {
				const t = (el.innerText || '').trim().toLowerCase();
				return t === 'checking...' || 
				       t === 'đang kiểm tra...' || 
				       t.includes('đang kiểm tra bản quyền') || 
				       t.includes('running copyright check');
			});
			if (standalone) {
				isChecking = true;
				statusText = standalone.innerText.trim();
			}
		}

		return {
			foundSection: foundSection,
			isChecking: isChecking,
			isComplete: isComplete,
			statusText: statusText
		};
	}`

	// Wait 3 seconds to allow TikTok Studio to initiate copyright check after upload
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}
	pollingStartTime := time.Now()

	for time.Since(pollingStartTime) < 150*time.Second {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		res, err := page.Eval(copyrightEvalScript)
		if err == nil && res != nil {
			isChecking := res.Value.Get("isChecking").Bool()
			isComplete := res.Value.Get("isComplete").Bool()
			statusText := res.Value.Get("statusText").String()

			if isChecking {
				wasChecking = true
				if time.Since(lastLogTime) >= 5*time.Second {
					msg := "⏳ Copyright check in progress, waiting for TikTok Studio to finish..."
					if statusText != "" && len(statusText) < 80 {
						msg = fmt.Sprintf("⏳ Copyright check in progress: %s...", statusText)
					}
					log("info", msg)
					lastLogTime = time.Now()
				}
				time.Sleep(2 * time.Second)
				continue
			}

			if isComplete {
				log("success", "✅ Copyright check completed! (No issues detected)")
				return nil
			}

			// If it was checking and is no longer checking, it completed
			if wasChecking && !isChecking {
				log("success", "✅ Copyright check finished on TikTok Studio.")
				return nil
			}
		}

		// If at least 8 seconds of polling elapsed without entering checking state and was never checking,
		// then copyright check is either not enabled or already done
		if time.Since(pollingStartTime) >= 8*time.Second && !wasChecking {
			log("info", "ℹ️ No copyright check in progress (ready to post).")
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	if wasChecking {
		log("warn", "⚠️ Copyright check wait timeout reached (150s). Aborting to prevent publishing unchecked video.")
		return fmt.Errorf("tiktok copyright check timed out after 150 seconds")
	}
	return nil
}

func (p *TikTokUploader) waitForPostButtonReady(ctx context.Context, page *rod.Page, log func(level, msg string)) error {
	startTime := time.Now()
	for time.Since(startTime) < 20*time.Second {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		res, err := page.Eval(`() => {
			const btn = document.querySelector('button[data-e2e="post_video_button"], button[aria-label*="Post" i], button[aria-label*="Schedule" i], button[aria-label*="Đăng" i]');
			if (!btn) return { exists: false, disabled: true };
			const disabled = btn.disabled || 
				btn.getAttribute('aria-disabled') === 'true' || 
				btn.classList.contains('disabled') || 
				btn.classList.contains('TUXButton--disabled') || 
				btn.classList.contains('tux-button--disabled');
			return { exists: true, disabled: disabled };
		}`)
		if err == nil && res != nil {
			exists := res.Value.Get("exists").Bool()
			disabled := res.Value.Get("disabled").Bool()
			if exists && !disabled {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("post/schedule button not found or remained disabled after 20 seconds")
}
