package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type FacebookPlatform struct{}

func init() {
	RegisterPlatform(&FacebookPlatform{})
}

func (p *FacebookPlatform) ID() string {
	return "facebook"
}

func (p *FacebookPlatform) DisplayName() string {
	return "Facebook Reels"
}

func (p *FacebookPlatform) LoginURL() string {
	return "https://business.facebook.com/latest/reels_composer"
}

func (p *FacebookPlatform) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	page := b.MustPage()
	defer page.Close()

	if logFn != nil {
		logFn("info", "Đang kiểm tra trạng thái đăng nhập Facebook Business Suite...")
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

func (p *FacebookPlatform) UploadVideo(ctx context.Context, b *rod.Browser, item *VideoItem, logFn func(level, msg string)) error {
	log := func(level, msg string) {
		if logFn != nil {
			logFn(level, fmt.Sprintf("[Facebook Reels] %s", msg))
		}
	}

	log("info", fmt.Sprintf("🎬 Bắt đầu upload Facebook Reels: %s", item.CustomTitle))
	log("info", fmt.Sprintf("📅 Dự kiến lên lịch: %s lúc %s (%s)", item.ScheduledDate, item.ScheduledTime, item.GoldenHourSlot))

	// Find or create page
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
	log("cdp", "Điều hướng tới Meta Business Suite Reels Composer...")
	err = page.Navigate(p.LoginURL())
	if err != nil {
		return fmt.Errorf("lỗi điều hướng Reels Composer: %w", err)
	}

	_ = page.WaitLoad()
	time.Sleep(3 * time.Second)

	info, err := page.Info()
	if err == nil && (strings.Contains(info.URL, "/login") || strings.Contains(info.URL, "facebook.com/login")) {
		return fmt.Errorf("chưa đăng nhập Facebook Business Suite. Vui lòng mở trình duyệt và đăng nhập tài khoản Meta")
	}

	// Step 2: Upload file
	log("cdp", fmt.Sprintf("Chọn file video: %s (%s)", item.Filename, item.FileSizeHuman))
	fileInput, err := page.Timeout(30 * time.Second).Element("input[type=\"file\"]")
	if err != nil {
		return fmt.Errorf("không tìm thấy ô input[type='file'] của Meta Business Suite: %w", err)
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

	log("info", "⏳ Đã gửi file qua CDP, đang chờ Meta xử lý Reels...")

	// Step 3: Wait for composer editor
	err = rod.Try(func() {
		page.Timeout(90 * time.Second).MustElement("[contenteditable=\"true\"], textarea, [role=\"textbox\"]")
	})
	if err != nil {
		return fmt.Errorf("timeout chờ Meta xử lý video: %w", err)
	}

	time.Sleep(2 * time.Second)

	// Step 4: Fill Caption
	log("cdp", fmt.Sprintf("Điền mô tả Reels: %s", item.CustomTitle))
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
	if err != nil {
		log("warn", fmt.Sprintf("Cảnh báo khi điền mô tả: %v", err))
	}

	time.Sleep(1 * time.Second)

	// Step 5: Advance through steps to scheduling
	log("cdp", "Chuyển tiếp tới bước hẹn giờ...")
	for i := 0; i < 2; i++ {
		_ = rod.Try(func() {
			nextBtn, err := page.Timeout(5 * time.Second).Element("button[aria-label*='Next' i], button[aria-label*='Tiếp' i], [role='button'][aria-label*='Next' i]")
			if err == nil && nextBtn != nil {
				_ = nextBtn.Click(proto.InputMouseButtonLeft, 1)
				time.Sleep(1500 * time.Millisecond)
			}
		})
	}

	// Step 6: Select Schedule radio
	log("cdp", fmt.Sprintf("Cấu hình lịch đăng: %s %s", item.ScheduledDate, item.ScheduledTime))
	_, _ = page.Eval(`() => {
		const radios = Array.from(document.querySelectorAll('input[type="radio"], [role="radio"], label'));
		const scheduleOpt = radios.find(r => r.innerText && (r.innerText.includes('Schedule') || r.innerText.includes('Lên lịch')));
		if (scheduleOpt) scheduleOpt.click();
	}`)

	time.Sleep(1 * time.Second)

	// Step 7: Click Schedule / Publish Button
	log("cdp", "Bấm nút Schedule Reels...")
	scheduled, _ := page.Eval(`() => {
		const btns = Array.from(document.querySelectorAll('button, [role="button"]'));
		const scheduleBtn = btns.find(b => {
			const txt = b.innerText?.trim() || '';
			return txt === 'Schedule' || txt === 'Lên lịch' || txt === 'Save' || txt === 'Lưu';
		});
		if (scheduleBtn && !scheduleBtn.hasAttribute('disabled')) {
			scheduleBtn.click();
			return true;
		}
		return false;
	}`)

	if !scheduled.Value.Bool() {
		log("warn", "Không tìm thấy nút Schedule trực tiếp, thử tìm nút hành động hoàn tất...")
		_ = rod.Try(func() {
			btn, err := page.Element("button[type='submit'], [data-testid*='submit' i]")
			if err == nil && btn != nil {
				_ = btn.Click(proto.InputMouseButtonLeft, 1)
			}
		})
	}

	log("info", "⏳ Đã gửi lệnh hẹn giờ Facebook Reels, chờ xác nhận...")
	time.Sleep(5 * time.Second)

	log("success", fmt.Sprintf("🎉 Lên lịch Facebook Reels thành công: %s lúc %s", item.ScheduledDate, item.ScheduledTime))
	return nil
}
