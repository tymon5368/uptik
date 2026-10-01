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

type TikTokUploader struct {
	policyProvider func() string
}

func NewTikTokUploader() *TikTokUploader {
	return &TikTokUploader{}
}

func (p *TikTokUploader) SetPolicyProvider(provider func() string) {
	p.policyProvider = provider
}

func (p *TikTokUploader) GetRestrictedPolicy() string {
	if p.policyProvider != nil {
		pol := p.policyProvider()
		if pol == domain.TikTokRestrictedPolicyPostAnyway {
			return domain.TikTokRestrictedPolicyPostAnyway
		}
		if pol == domain.TikTokRestrictedPolicySkip {
			return domain.TikTokRestrictedPolicySkip
		}
	}
	return domain.TikTokRestrictedPolicySkip
}

func (p *TikTokUploader) dismissRestrictedModal(page *rod.Page, log func(level, msg string)) bool {
	res, err := page.Eval(`() => {
		const modals = Array.from(document.querySelectorAll('[role="dialog"], .TUXModal, .TUXDialog, .modal-content, [class*="modal"], [class*="dialog"]'));
		for (const modal of modals) {
			const txt = (modal.innerText || '').toLowerCase();
			if (txt.includes('content may be restricted') || 
			    txt.includes('nội dung có thể bị hạn chế') || 
			    txt.includes('violation reason') || 
			    txt.includes('unoriginal, low-quality') ||
			    txt.includes('unoriginal') ||
			    txt.includes('không nguyên bản')) {
				
				// 1. Try finding close button
				const closeBtn = modal.querySelector('button[aria-label*="close" i], [class*="close"], [data-e2e*="close"], svg[class*="close"], [class*="modal-close"]') ||
				                 Array.from(modal.querySelectorAll('button')).find(b => {
				                 	const t = (b.innerText || '').trim().toLowerCase();
				                 	return t === '✕' || t === 'x' || t === '' || b.querySelector('svg');
				                 });
				if (closeBtn) {
					closeBtn.focus();
					closeBtn.click();
					closeBtn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
					return true;
				}

				// 2. Dispatch Escape key event to modal and document
				modal.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', keyCode: 27, which: 27, bubbles: true }));
				document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', keyCode: 27, which: 27, bubbles: true }));
				return true;
			}
		}
		return false;
	}`)
	if err == nil && res != nil && res.Value.Bool() {
		log("info", "Dismissed TikTok Studio Content Restriction warning modal.")
		time.Sleep(500 * time.Millisecond)
		return true
	}
	return false
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
		evalScript := `(title, targetYear, targetMonth, targetDay, targetHour, targetMin) => {
			return new Promise(async (resolve) => {
				// 1. Caption (Draft.js)
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
				const scheduleLabels = Array.from(document.querySelectorAll('label, [class*="Radio"], span')).filter(
					el => el.textContent && (el.textContent.trim() === 'Schedule' || el.textContent.trim() === 'Lên lịch')
				);
				if (scheduleLabels.length > 0) {
					scheduleLabels[0].click();
				}

				await new Promise(r => setTimeout(r, 400));

				// 3. Locate scheduled picker container
				const picker = document.querySelector('.scheduled-picker');
				if (!picker) {
					resolve({ success: false, error: 'Cannot find .scheduled-picker container' });
					return;
				}

				const pickerChildren = Array.from(picker.children);
				const timeBlock = pickerChildren[0];
				const dateBlock = pickerChildren[pickerChildren.length - 1];

				if (!dateBlock || !timeBlock) {
					resolve({ success: false, error: 'Cannot find date or time container inside .scheduled-picker' });
					return;
				}

				const dateInput = dateBlock.querySelector('input');
				const timeInput = timeBlock.querySelector('input');

				// 4. Open Date Picker Calendar
				let calendar = document.querySelector('.calendar-wrapper');
				if (!calendar) {
					const clickTarget = dateBlock.querySelector('.TUXInputBox') || dateInput || dateBlock;
					clickTarget.click();
					await new Promise(r => setTimeout(r, 350));
					calendar = document.querySelector('.calendar-wrapper');
				}

				if (!calendar) {
					resolve({ success: false, error: 'Cannot open date picker calendar (.calendar-wrapper)' });
					return;
				}

				// 5. Navigate Month & Year (Bidirectional & localized)
				const parseMonthNumber = (text) => {
					if (!text) return null;
					const clean = text.trim();
					const monthNames = ["january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"];
					const shortNames = ["jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"];
					const lower = clean.toLowerCase();
					for (let i = 0; i < 12; i++) {
						if (lower === monthNames[i] || lower === shortNames[i]) {
							return i + 1;
						}
					}
					const parts = clean.split(/\D+/).filter(Boolean);
					for (const part of parts) {
						const num = Number(part);
						if (num >= 1 && num <= 12 && part.length <= 2) {
							return num;
						}
					}
					return null;
				};

				const parseYearNumber = (yearText, monthText) => {
					const combined = (yearText || '') + ' ' + (monthText || '');
					const m = combined.match(/\b(20\d\d)\b/);
					return m ? Number(m[1]) : null;
				};

				let reachedTargetMonth = false;
				for (let m = 0; m < 24; m++) {
					const monthTitle = document.querySelector('.month-title')?.innerText?.trim();
					const yearTitle = document.querySelector('.year-title')?.innerText?.trim();

					const curMonth = parseMonthNumber(monthTitle);
					const curYear = parseYearNumber(yearTitle, monthTitle) || targetYear;

					const monthMatches = (curMonth === targetMonth);
					const yearMatches = (!yearTitle || curYear === targetYear);

					if (monthMatches && yearMatches) {
						reachedTargetMonth = true;
						break;
					}

					const arrows = document.querySelectorAll('.month-header-wrapper .arrow');
					if (arrows.length === 0) break;

					const prevArrow = arrows[0];
					const nextArrow = arrows[arrows.length - 1];

					if (curMonth !== null) {
						const curTotal = curYear * 12 + curMonth;
						const targetTotal = targetYear * 12 + targetMonth;
						if (targetTotal < curTotal) {
							prevArrow.click();
						} else {
							nextArrow.click();
						}
					} else {
						nextArrow.click();
					}
					await new Promise(r => setTimeout(r, 300));
				}

				if (!reachedTargetMonth) {
					resolve({
						success: false,
						error: 'Failed to navigate calendar to target month ' + targetMonth + '/' + targetYear
					});
					return;
				}

				// 6. Day Selection with a single standard DOM click
				const validDays = Array.from(document.querySelectorAll('.calendar-wrapper span.day.valid'));
				let dayEl = validDays.find(el => el.innerText.trim() === String(targetDay));

				if (!dayEl) {
					const allDays = Array.from(document.querySelectorAll('.calendar-wrapper [class*="day"]'));
					dayEl = allDays.find(el => el.innerText.trim() === String(targetDay) && !el.className.includes('disabled') && !el.className.includes('prev-month') && !el.className.includes('next-month'));
				}

				if (!dayEl) {
					resolve({
						success: false,
						error: 'Cannot find day ' + targetDay + ' in target month calendar'
					});
					return;
				}

				dayEl.scrollIntoView({ block: 'nearest' });
				dayEl.click();
				await new Promise(r => setTimeout(r, 350));

				// Close calendar popup if still open
				const stillOpenCalendar = document.querySelector('.calendar-wrapper');
				if (stillOpenCalendar) {
					const clickTarget = dateBlock.querySelector('.TUXInputBox') || dateInput || dateBlock;
					clickTarget.click();
					await new Promise(r => setTimeout(r, 200));
				}

				// 7. Time Picker: Open dropdown
				const timeBox = timeBlock.querySelector('.TUXInputBox') || timeInput || timeBlock;
				timeBox.click();
				await new Promise(r => setTimeout(r, 300));

				// Select Hour (.tiktok-timepicker-left)
				const hourEl = Array.from(document.querySelectorAll('.tiktok-timepicker-left')).find(
					el => el.innerText.trim() === targetHour
				);
				if (hourEl) {
					hourEl.scrollIntoView({ block: 'nearest' });
					hourEl.click();
				}
				await new Promise(r => setTimeout(r, 200));

				// Select Minute (.tiktok-timepicker-right)
				const minEl = Array.from(document.querySelectorAll('.tiktok-timepicker-right')).find(
					el => el.innerText.trim() === targetMin
				);
				if (minEl) {
					minEl.scrollIntoView({ block: 'nearest' });
					minEl.click();
				}
				await new Promise(r => setTimeout(r, 200));

				// Close time dropdown if visible
				const timeContainer = document.querySelector('.tiktok-timepicker-time-picker-container');
				if (timeContainer && !timeContainer.classList.contains('tiktok-timepicker-invisible')) {
					timeBox.click();
					await new Promise(r => setTimeout(r, 200));
				}

				resolve({
					success: true,
					dateSet: dateInput?.value,
					timeSet: timeInput?.value
				});
			});
		}`

		res, err := page.Eval(evalScript, item.CustomTitle, targetYear, targetMonth, targetDay, targetHour, targetMin)
		if err != nil {
			log("warn", fmt.Sprintf("Schedule configuration eval warning: %v", err))
		} else if res != nil {
			if !res.Value.Get("success").Bool() {
				errMsg := res.Value.Get("error").String()
				log("warn", fmt.Sprintf("Schedule configuration warning: %s", errMsg))
			} else {
				dateSet := res.Value.Get("dateSet").String()
				timeSet := res.Value.Get("timeSet").String()
				log("info", fmt.Sprintf("📅 Schedule set successfully: %s %s", dateSet, timeSet))
			}
		}
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
		if err == nil && strings.Contains(pageInfo.URL, "/content") {
			submitted = true
			break
		}

		// Auto dismiss or abort if Content Restriction modal pops up
		resRestrict, _ := page.Eval(`() => {
			const modals = Array.from(document.querySelectorAll('[role="dialog"], .TUXModal, .TUXDialog, .modal-content, [class*="modal"], [class*="dialog"]'));
			for (const modal of modals) {
				const txt = (modal.innerText || '').toLowerCase();
				if (txt.includes('content may be restricted') || 
				    txt.includes('nội dung có thể bị hạn chế') || 
				    txt.includes('violation reason') || 
				    txt.includes('unoriginal, low-quality') || 
				    txt.includes('unoriginal')) {
					return true;
				}
			}
			return false;
		}`)
		if resRestrict != nil && resRestrict.Value.Bool() {
			policy := p.GetRestrictedPolicy()
			if policy == domain.TikTokRestrictedPolicySkip {
				log("warn", "⚠️ TikTok Studio prompted Content Restriction modal during confirmation. Policy is 'skip' -> Aborting.")
				return domain.ErrContentRestricted
			}
			log("info", "Content Restriction modal detected during confirmation. Policy is 'post_anyway', dismissing...")
			_ = p.dismissRestrictedModal(page, log)
			time.Sleep(1 * time.Second)

			// Attempt re-click on Post button if it was unblocked
			_, _ = page.Eval(`() => {
				const btn = document.querySelector('button[data-e2e="post_video_button"], button[aria-label*="Post" i], button[aria-label*="Schedule" i], button[aria-label*="Đăng" i]');
				if (btn && !btn.disabled && btn.getAttribute('aria-disabled') !== 'true') {
					btn.click();
				}
			}`)
			continue
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
				    dText.includes('schedule anyway') ||
				    dText.includes('checking') ||
				    dText.includes('kiểm tra') ||
				    dText.includes('in progress') ||
				    dText.includes('quá trình')) {
					
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
				       t === 'vẫn đăng' || 
				       t === 'tiếp tục';
			});
			if (fallbackBtn) {
				fallbackBtn.focus();
				fallbackBtn.click();
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
				if (text.includes('your video has been published') || 
				    text.includes('video của bạn đã được đăng') || 
				    text.includes('your video is scheduled') || 
				    text.includes('your video was scheduled') || 
				    text.includes('video của bạn đã được lên lịch')) {
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

	// Final Smart Confirmation Check before reporting error
	if !submitted {
		pageInfo, err := page.Info()
		if err == nil && strings.Contains(pageInfo.URL, "/content") {
			log("info", "Smart Confirmation: detected TikTok Studio /content redirect after timeout.")
			submitted = true
		} else {
			finalCheck, _ := page.Eval(`() => {
				const txt = (document.body.innerText || '').toLowerCase();
				return txt.includes('your video has been published') || 
				       txt.includes('video của bạn đã được đăng') || 
				       txt.includes('your video is scheduled') || 
				       txt.includes('video của bạn đã được lên lịch');
			}`)
			if finalCheck != nil && finalCheck.Value.Bool() {
				log("info", "Smart Confirmation: detected post completion text on page after timeout.")
				submitted = true
			}
		}
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
	log("info", "⏳ Checking copyright and content status on TikTok Studio...")
	lastLogTime := time.Time{}
	wasChecking := false
	consecutiveEvalErrors := 0
	successfulEvals := 0

	copyrightEvalScript := `() => {
		const allEls = Array.from(document.querySelectorAll('div, section, p, span, label, [data-e2e*="copyright"], [data-e2e*="check"]'));
		let isChecking = false;
		let isComplete = false;
		let isTenMinuteCheck = false;
		let statusText = '';
		let foundSection = false;
		let isContentRestricted = false;
		let restrictedReason = '';
		let isRestrictedModalOpen = false;

		// 1. Check for modal popup (ONLY VISIBLE MODALS)
		const modals = Array.from(document.querySelectorAll('[role="dialog"], .TUXModal, .TUXDialog, .modal-content, [class*="modal"]'));
		for (const modal of modals) {
			if (modal.offsetParent === null || window.getComputedStyle(modal).display === 'none') continue;
			const txt = (modal.innerText || '').toLowerCase();
			if (txt.includes('content may be restricted') || 
			    txt.includes('nội dung có thể bị hạn chế') || 
			    txt.includes('violation reason') || 
			    txt.includes('unoriginal, low-quality') || 
			    txt.includes('unoriginal')) {
				isRestrictedModalOpen = true;
				isContentRestricted = true;
				restrictedReason = (modal.innerText || '').slice(0, 150);
				break;
			}
		}

		// 2. Scan checks sections on page (ONLY VISIBLE ELEMENTS)
		for (const el of allEls) {
			if (el.offsetParent === null) continue;
			if (el.closest('[data-show="false"]')) continue;
			const txt = (el.innerText || '').toLowerCase();
			if (!txt.includes('copyright') && !txt.includes('bản quyền') && !txt.includes('content check') && !txt.includes('kiểm tra nội dung')) continue;
			if ((el.innerText || '').length > 600) continue;

			foundSection = true;

			// Check for restricted indicators
			if ((txt.includes('content check') || txt.includes('kiểm tra nội dung')) && 
			    (txt.includes('content may be restricted') || txt.includes('nội dung có thể bị hạn chế') || txt.includes('restricted') || txt.includes('vi phạm'))) {
				isContentRestricted = true;
				if (!restrictedReason) restrictedReason = (el.innerText || '').trim();
			}

			// Check if TikTok mentions 10-minute non-blocking check
			if (txt.includes('10 minutes') || txt.includes('10 phút') || txt.includes('about 10')) {
				isTenMinuteCheck = true;
			}

			const hasSpinner = el.querySelector(
				'[class*="loading"], [class*="spinner"], [class*="circle-loading"], svg[class*="spin"], [class*="TUXLoading"]'
			) !== null;

			const checkingKeywords = [
				'checking...',
				'running copyright check',
				'checking for copyright',
				'checking for issues',
				'đang kiểm tra...',
				'đang kiểm tra bản quyền',
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
				if (el.offsetParent === null) return false;
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
			isTenMinuteCheck: isTenMinuteCheck,
			statusText: statusText,
			isContentRestricted: isContentRestricted,
			isRestrictedModalOpen: isRestrictedModalOpen,
			restrictedReason: restrictedReason
		};
	}`

	// Wait 2 seconds to allow TikTok Studio to initiate copyright and content check after upload
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}
	pollingStartTime := time.Now()

	for time.Since(pollingStartTime) < 12*time.Second {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		res, err := page.Eval(copyrightEvalScript)
		if err != nil {
			consecutiveEvalErrors++
			if consecutiveEvalErrors >= 5 {
				log("warn", fmt.Sprintf("Evaluation warning during copyright check: %v. Proceeding to schedule...", err))
				return nil
			}
			time.Sleep(1 * time.Second)
			continue
		}
		consecutiveEvalErrors = 0
		successfulEvals++

		if res != nil {
			isChecking := res.Value.Get("isChecking").Bool()
			isComplete := res.Value.Get("isComplete").Bool()
			isTenMinuteCheck := res.Value.Get("isTenMinuteCheck").Bool()
			statusText := res.Value.Get("statusText").String()
			isContentRestricted := res.Value.Get("isContentRestricted").Bool()
			isRestrictedModalOpen := res.Value.Get("isRestrictedModalOpen").Bool()
			restrictedReason := res.Value.Get("restrictedReason").String()

			// Handle Content Restriction
			if isContentRestricted || isRestrictedModalOpen {
				policy := p.GetRestrictedPolicy()
				if policy == domain.TikTokRestrictedPolicySkip {
					log("warn", fmt.Sprintf("⚠️ TikTok Studio Content Check flagged: Video may be restricted (Unoriginal/Low quality: %s). Policy is 'skip' -> Aborting to protect channel.", restrictedReason))
					return domain.ErrContentRestricted
				}

				// Policy is post_anyway
				log("warn", "⚠️ TikTok Studio Content Check flagged: Video may be restricted, but policy is 'post_anyway'. Dismissing warning modal to continue...")
				if isRestrictedModalOpen {
					dismissed := false
					for retry := 0; retry < 3; retry++ {
						if p.dismissRestrictedModal(page, log) {
							stillOpenRes, _ := page.Eval(`() => {
								const modals = Array.from(document.querySelectorAll('[role="dialog"], .TUXModal, .TUXDialog, .modal-content, [class*="modal"], [class*="dialog"]'));
								return modals.some(m => {
									const t = (m.innerText || '').toLowerCase();
									return t.includes('content may be restricted') || t.includes('nội dung có thể bị hạn chế');
								});
							}`)
							if stillOpenRes == nil || !stillOpenRes.Value.Bool() {
								dismissed = true
								break
							}
						}
						time.Sleep(500 * time.Millisecond)
					}
					if !dismissed {
						log("warn", "⚠️ Could not dismiss Content Restriction modal after retries. Post button may remain blocked.")
					}
					time.Sleep(500 * time.Millisecond)
				}
				return nil
			}

			if isComplete {
				log("success", "✅ Copyright and content checks completed! (No issues detected)")
				return nil
			}

			if isTenMinuteCheck {
				log("info", "ℹ️ TikTok Studio content quality check takes ~10 minutes (non-blocking). Proceeding with schedule...")
				return nil
			}

			if isChecking {
				wasChecking = true
				if time.Since(lastLogTime) >= 3*time.Second {
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

			// If it was checking and is no longer checking, it completed
			if wasChecking && !isChecking {
				log("success", "✅ Copyright check finished on TikTok Studio.")
				return nil
			}
		}

		if time.Since(pollingStartTime) >= 5*time.Second && !wasChecking && successfulEvals > 0 {
			log("info", "ℹ️ No copyright check in progress (ready to post).")
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	if wasChecking {
		log("info", "ℹ️ Copyright check is taking longer than expected. Proceeding to schedule (TikTok will verify in background)...")
		return nil
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
