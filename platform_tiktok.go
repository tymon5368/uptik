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
	evalScript := `(title, targetDay, targetMonth, targetHour, targetMin) => {
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

	_, err = page.Eval(evalScript, item.CustomTitle, targetDay, targetMonth, targetHour, targetMin)
	if err != nil {
		log("warn", fmt.Sprintf("Form fill warning: %v", err))
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

		// Handle copyright modal if prompted
		res, _ := page.Eval(`() => {
			const btns = Array.from(document.querySelectorAll('button'));
			const postAnyway = btns.find(b => b.innerText && (b.innerText.includes('Post anyway') || b.innerText.includes('Schedule anyway') || b.innerText.includes('Vẫn lên lịch')));
			if (postAnyway) {
				postAnyway.click();
				return true;
			}
			return false;
		}`)
		if res.Value.Bool() {
			log("info", "Auto-confirmed 'Post anyway' (bypassed copyright check wait).")
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
		if modalRes.Value.Bool() {
			log("warn", "Detected blocking modal, dismissed and retrying Schedule click...")
			time.Sleep(500 * time.Millisecond)
			_ = scheduleBtn.Click(proto.InputMouseButtonLeft, 1)
		}

		// Check success modal
		successModal, _ := page.Eval(`() => {
			const modal = document.querySelector('.TUXModal, [role="dialog"], .modal-content');
			if (!modal) return false;
			const text = modal.innerText || '';
			return text.includes('Manage your posts') || text.includes('scheduled') || text.includes('uploaded');
		}`)
		if successModal.Value.Bool() {
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
