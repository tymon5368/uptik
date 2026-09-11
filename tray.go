package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/energye/systray"
)

var (
	trayStarted     sync.Once
	mAutoStartItem  *systray.MenuItem
	mAutoUploadItem *systray.MenuItem
	trayMu          sync.Mutex
)

func initTray(app *App, iconBytes []byte) {
	trayStarted.Do(func() {
		systray.Run(func() {
			if len(iconBytes) > 0 {
				systray.SetIcon(iconBytes)
			}
			systray.SetTitle("UpTik")
			systray.SetTooltip("UpTik - TikTok Auto Scheduler (Đang chạy ngầm)")

			// Left click or double click on tray icon toggles window
			systray.SetOnClick(func(menu systray.IMenu) {
				app.ToggleAppWindow()
			})
			systray.SetOnDClick(func(menu systray.IMenu) {
				app.ShowApp()
			})

			mShow := systray.AddMenuItem("Mở UpTik", "Hiện cửa sổ ứng dụng")
			mShow.Click(func() {
				app.ShowApp()
			})

			mHide := systray.AddMenuItem("Ẩn vào khay", "Ẩn ứng dụng xuống khay hệ thống")
			mHide.Click(func() {
				app.HideApp()
			})

			systray.AddSeparator()

			st := app.GetSettings()
			mAutoUpload := systray.AddMenuItemCheckbox("Tự động đăng (Publish Now)", "Tự động đăng video khi đến khung giờ", st.AutoUploadEnabled)
			trayMu.Lock()
			mAutoUploadItem = mAutoUpload
			trayMu.Unlock()

			mAutoUpload.Click(func() {
				current := app.GetSettings()
				newVal := !current.AutoUploadEnabled
				_ = app.ToggleAutoUpload(newVal)
				if newVal {
					mAutoUpload.Check()
				} else {
					mAutoUpload.Uncheck()
				}
			})

			mAuto := systray.AddMenuItemCheckbox("Khởi động cùng hệ thống", "Tự động chạy UpTik khi mở máy", st.AutoStart)
			trayMu.Lock()
			mAutoStartItem = mAuto
			trayMu.Unlock()

			mAuto.Click(func() {
				current := app.GetSettings()
				newVal := !current.AutoStart
				current.AutoStart = newVal
				_ = app.SaveSettings(current)
				if newVal {
					mAuto.Check()
				} else {
					mAuto.Uncheck()
				}
			})

			mFolder := systray.AddMenuItem("Mở thư mục video", "Mở thư mục video đã cấu hình")
			mFolder.Click(func() {
				s := app.GetSettings()
				folder := s.VideoFolder
				if folder != "" {
					openFolderInFileManager(folder)
				}
			})

			systray.AddSeparator()

			mQuit := systray.AddMenuItem("Thoát hoàn toàn", "Đóng và thoát ứng dụng UpTik")
			mQuit.Click(func() {
				app.QuitApp()
			})

			// Register listener for external settings update
			app.RegisterSettingsUpdateListener(func(s Settings) {
				trayMu.Lock()
				defer trayMu.Unlock()
				if mAutoStartItem != nil {
					if s.AutoStart {
						mAutoStartItem.Check()
					} else {
						mAutoStartItem.Uncheck()
					}
				}
				if mAutoUploadItem != nil {
					if s.AutoUploadEnabled {
						mAutoUploadItem.Check()
						systray.SetTooltip("UpTik - Tự động đăng video theo khung giờ (Đang bật)")
					} else {
						mAutoUploadItem.Uncheck()
						systray.SetTooltip("UpTik - TikTok Auto Scheduler (Đang chạy ngầm)")
					}
				}
			})
		}, func() {
			// On tray exit
		})
	})
}

func openFolderInFileManager(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", filepath.FromSlash(path))
	case "darwin":
		cmd = exec.Command("open", path)
	default: // Linux / Unix
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}
