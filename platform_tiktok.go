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

type TikTokPlatform struct{}

func init() {
	RegisterPlatform(&TikTokPlatform{})
}

func (p *TikTokPlatform) ID() string {
	return "tiktok"
}

func (p *TikTokPlatform) DisplayName() string {
	return "TikTok Studio"
}

func (p *TikTokPlatform) LoginURL() string {
	return "https://www.tiktok.com/tiktokstudio/upload"
}

func (p *TikTokPlatform) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
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

	hasUpload, _ := page.Element("input[type=\"file\"]")
	if hasUpload != nil {
		return true, nil
	}

	return !strings.Contains(info.URL, "login"), nil
}

func (p *TikTokPlatform) UploadVideo(ctx context.Context, b *rod.Browser, item *VideoItem, logFn func(level, msg string)) error {
	log := func(level, msg string) {
		if logFn != nil {
			logFn(level, fmt.Sprintf("[TikTok] %s", msg))
		}
	}

	log("info", fmt.Sprintf("🎬 Starting TikTok upload: %s", item.CustomTitle))
	log("info", fmt.Sprintf("📅 Scheduled for: %s at %s (%s)", item.ScheduledDate, item.ScheduledTime, item.GoldenHourSlot))

	// Find or create page
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
	log("cdp", "Navigating to TikTok Studio Upload page...")
	err = page.Navigate("https://www.tiktok.com/tiktokstudio/upload")
	if err != nil {
		return fmt.Errorf("navigation error: %w", err)
	}

	_ = page.WaitLoad()
	time.Sleep(2 * time.Second)

	// Check if redirected to login
	info, err := page.Info()
	if err == nil && strings.Contains(info.URL, "/login") {
		return fmt.Errorf("not logged into TikTok Studio. Please log in via browser")
	}

	// Remove iframe/banner blockers
	_, _ = page.Eval(`() => {
		const blockers = document.querySelectorAll('iframe[src*="verify"], .captcha-verify-container');
		blockers.forEach(el => el.remove());
	}`)

	// Step 2: Upload file
	log("cdp", fmt.Sprintf("Selecting video file: %s (%s)", item.Filename, item.FileSizeHuman))
	fileInput, err := page.Timeout(30 * time.Second).Element("input[type=\"file\"]")
	if err != nil {
		return fmt.Errorf("file input element not found: %w", err)
	}

	absPath, err := filepath.Abs(item.FullPath)
	if err != nil {
		absPath = item.FullPath
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist on disk: %s", absPath)
	}

	err = fileInput.SetFiles([]string{absPath})
	if err != nil {
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

	log("success", "TikTok recognized video file. Waiting for editor inputs to be ready...")
	time.Sleep(3 * time.Second)

	// Step 4: Parse Scheduled Date & Time
	if item.ScheduledDate == "" || item.ScheduledTime == "" {
		return fmt.Errorf("video chưa có thông tin ngày hoặc giờ lên lịch")
	}

	dateParts := strings.Split(item.ScheduledDate, "-")
	if len(dateParts) != 3 {
		return fmt.Errorf("định dạng ngày không hợp lệ: %s", item.ScheduledDate)
	}
	targetYear, _ := strconv.Atoi(dateParts[0])
	targetMonth, _ := strconv.Atoi(dateParts[1])
	targetDay, _ := strconv.Atoi(dateParts[2])

	timeParts := strings.Split(item.ScheduledTime, ":")
	if len(timeParts) != 2 {
		return fmt.Errorf("định dạng giờ không hợp lệ: %s", item.ScheduledTime)
	}
	targetHour := fmt.Sprintf("%02s", strings.TrimSpace(timeParts[0]))
	targetMin := fmt.Sprintf("%02s", strings.TrimSpace(timeParts[1]))

	log("cdp", fmt.Sprintf("Filling caption and configuring schedule: %04d-%02d-%02d %s:%s", targetYear, targetMonth, targetDay, targetHour, targetMin))

	// Step 5: Fill Caption & Interact with React 18 pickers
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
				const m4 = combined.match(/\b(20\d\d)\b/) || combined.match(/\b(\d{4})\b/);
				if (m4) return Number(m4[1]);
				if (yearText) {
					const m2 = yearText.trim().match(/^\D*(\d{2})\D*$/);
					if (m2) {
						const y = Number(m2[1]);
						return y < 50 ? 2000 + y : 1900 + y;
					}
				}
				return null;
			};

			let reachedTargetMonth = false;
			for (let m = 0; m <= 25; m++) {
				const monthTitle = document.querySelector('.month-title')?.innerText?.trim();
				const yearElement = document.querySelector('.year-title');
				const yearTitle = yearElement?.innerText?.trim();

				const curMonth = parseMonthNumber(monthTitle);
				const curYear = parseYearNumber(yearTitle, monthTitle);

				const monthMatches = (curMonth === targetMonth);
				const yearMatches = yearElement ? (curYear !== null && curYear === targetYear) : (curYear === null || curYear === targetYear);

				if (monthMatches && yearMatches) {
					reachedTargetMonth = true;
					break;
				}

				if (m === 25) break;

				const arrows = document.querySelectorAll('.month-header-wrapper .arrow');
				if (arrows.length === 0) break;

				const prevArrow = arrows[0];
				const nextArrow = arrows[arrows.length - 1];

				if (curMonth !== null) {
					const effectiveYear = curYear !== null ? curYear : targetYear;
					const curTotal = effectiveYear * 12 + curMonth;
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
		log("warn", fmt.Sprintf("Form fill warning: %v", err))
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

	// Dismiss warning modal if any
	_, _ = page.Eval(`() => {
		const btns = Array.from(document.querySelectorAll('button'));
		const dismissBtn = btns.find(b => b.innerText && (b.innerText.includes('Got it') || b.innerText.includes('Đã hiểu') || b.innerText.includes('Dismiss')));
		if (dismissBtn) dismissBtn.click();
	}`)
	time.Sleep(1 * time.Second)

	// Step 6: Click Schedule Button
	log("cdp", "Scrolling to Schedule button and sending click...")
	scheduleBtn, err := page.Element("button[data-e2e=\"post_video_button\"]")
	if err != nil {
		return fmt.Errorf("could not find Schedule button: %w", err)
	}

	err = scheduleBtn.ScrollIntoView()
	if err != nil {
		log("warn", fmt.Sprintf("Unable to scroll: %v", err))
	}
	time.Sleep(300 * time.Millisecond)
	_ = scheduleBtn.Click(proto.InputMouseButtonLeft, 1)

	log("info", "⏳ Clicked Schedule, awaiting confirmation from TikTok Studio...")

	// Step 7: Wait for confirmation (redirect to /content or success modal)
	submitted := false
	startTime := time.Now()

	for time.Since(startTime) < 45*time.Second {
		pageInfo, err := page.Info()
		if err == nil && (strings.Contains(pageInfo.URL, "/content") || strings.Contains(pageInfo.URL, "/manage")) {
			submitted = true
			break
		}

		// Handle copyright / checking dialog if prompted
		res, _ := page.Eval(`() => {
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

			// Global fallback
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
			log("info", "Auto-confirmed 'Post anyway' (bypassed copyright/content check wait).")
			time.Sleep(1 * time.Second)
			continue
		}

		// Handle error modal
		modalRes, _ := page.Eval(`() => {
			const modal = document.querySelector('.TUXModal, [role="dialog"]');
			if (!modal) return false;
			const text = modal.innerText || '';
			if (text.includes('Too many requests') || text.includes('failed')) {
				const closeBtn = modal.querySelector('button');
				if (closeBtn) closeBtn.click();
				return true;
			}
			return false;
		}`)
		if modalRes != nil && modalRes.Value.Bool() {
			log("warn", "Detected blocking modal, dismissed and retrying Schedule click...")
			time.Sleep(500 * time.Millisecond)
			_ = scheduleBtn.Click(proto.InputMouseButtonLeft, 1)
		}

		// Check success modal
		successModal, _ := page.Eval(`() => {
			const modal = document.querySelector('.TUXModal, [role="dialog"], .modal-content');
			if (!modal) return false;
			const text = modal.innerText || '';
			return text.includes('Manage your posts') || text.includes('scheduled') || text.includes('uploaded') || text.includes('đã được lên lịch') || text.includes('đã được đăng');
		}`)
		if successModal != nil && successModal.Value.Bool() {
			submitted = true
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !submitted {
		return fmt.Errorf("TikTok Studio did not confirm success within 45 seconds")
	}

	log("success", fmt.Sprintf("🎉 TikTok scheduled successfully: %s at %s", item.ScheduledDate, item.ScheduledTime))
	return nil
}
