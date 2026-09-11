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
	mShowItem       *systray.MenuItem
	mHideItem       *systray.MenuItem
	mAutoStartItem  *systray.MenuItem
	mAutoUploadItem *systray.MenuItem
	mFolderItem     *systray.MenuItem
	mQuitItem       *systray.MenuItem
	trayMu          sync.Mutex
)

type trayTexts struct {
	Show          string
	Hide          string
	AutoUpload    string
	AutoStart     string
	Folder        string
	Quit          string
	Tooltip       string
	TooltipActive string
}

func getTrayTexts(locale string) trayTexts {
	switch locale {
	case "de":
		return trayTexts{
			Show:          "UpTik öffnen",
			Hide:          "Im Tray minimieren",
			AutoUpload:    "Automatischer Upload (Publish Now)",
			AutoStart:     "Mit dem System starten",
			Folder:        "Video-Ordner öffnen",
			Quit:          "Vollständig beenden",
			Tooltip:       "UpTik - Video Auto Scheduler (Läuft im Hintergrund)",
			TooltipActive: "UpTik - Automatischer Upload aktiv",
		}
	case "ja":
		return trayTexts{
			Show:          "UpTik を開く",
			Hide:          "トレイに最小化",
			AutoUpload:    "自動投稿 (今すぐ公開)",
			AutoStart:     "システム起動時に実行",
			Folder:        "動画フォルダを開く",
			Quit:          "完全に終了",
			Tooltip:       "UpTik - 動画自動スケジューラー (バックグラウンド実行中)",
			TooltipActive: "UpTik - 自動投稿有効",
		}
	case "ar":
		return trayTexts{
			Show:          "فتح UpTik",
			Hide:          "إخفاء في شريط المهام",
			AutoUpload:    "نشر تلقائي (النشر الآن)",
			AutoStart:     "تشغيل مع بدء النظام",
			Folder:        "فتح مجلد الفيديو",
			Quit:          "إنهاء كلياً",
			Tooltip:       "UpTik - مجدول الفيديو التلقائي (قيد التشغيل في الخلفية)",
			TooltipActive: "UpTik - النشر التلقائي مفعّل",
		}
	case "fr":
		return trayTexts{
			Show:          "Ouvrir UpTik",
			Hide:          "Réduire dans la barre",
			AutoUpload:    "Publication automatique (Publish Now)",
			AutoStart:     "Démarrer avec le système",
			Folder:        "Ouvrir le dossier vidéo",
			Quit:          "Quitter complètement",
			Tooltip:       "UpTik - Planificateur vidéo (En arrière-plan)",
			TooltipActive: "UpTik - Publication automatique activée",
		}
	case "vi":
		return trayTexts{
			Show:          "Mở UpTik",
			Hide:          "Ẩn vào khay",
			AutoUpload:    "Tự động đăng (Publish Now)",
			AutoStart:     "Khởi động cùng hệ thống",
			Folder:        "Mở thư mục video",
			Quit:          "Thoát hoàn toàn",
			Tooltip:       "UpTik - TikTok Auto Scheduler (Đang chạy ngầm)",
			TooltipActive: "UpTik - Tự động đăng video theo khung giờ (Đang bật)",
		}
	default: // "en"
		return trayTexts{
			Show:          "Open UpTik",
			Hide:          "Minimize to Tray",
			AutoUpload:    "Auto Publish (Publish Now)",
			AutoStart:     "Start with System",
			Folder:        "Open Video Folder",
			Quit:          "Quit Completely",
			Tooltip:       "UpTik - Video Auto Scheduler (Running in background)",
			TooltipActive: "UpTik - Auto-upload enabled",
		}
	}
}

func applyTrayLocale(locale string, isAutoUpload bool) {
	txt := getTrayTexts(locale)
	if mShowItem != nil {
		mShowItem.SetTitle(txt.Show)
	}
	if mHideItem != nil {
		mHideItem.SetTitle(txt.Hide)
	}
	if mAutoUploadItem != nil {
		mAutoUploadItem.SetTitle(txt.AutoUpload)
	}
	if mAutoStartItem != nil {
		mAutoStartItem.SetTitle(txt.AutoStart)
	}
	if mFolderItem != nil {
		mFolderItem.SetTitle(txt.Folder)
	}
	if mQuitItem != nil {
		mQuitItem.SetTitle(txt.Quit)
	}
	if isAutoUpload {
		systray.SetTooltip(txt.TooltipActive)
	} else {
		systray.SetTooltip(txt.Tooltip)
	}
}

func initTray(app *App, iconBytes []byte) {
	trayStarted.Do(func() {
		systray.Run(func() {
			if len(iconBytes) > 0 {
				systray.SetIcon(iconBytes)
			}
			systray.SetTitle("UpTik")

			st := app.GetSettings()
			txt := getTrayTexts(st.Locale)
			systray.SetTooltip(txt.Tooltip)

			// Left click or double click on tray icon toggles window
			systray.SetOnClick(func(menu systray.IMenu) {
				app.ToggleAppWindow()
			})
			systray.SetOnDClick(func(menu systray.IMenu) {
				app.ShowApp()
			})

			mShow := systray.AddMenuItem(txt.Show, "Hiện cửa sổ ứng dụng")
			mShow.Click(func() {
				app.ShowApp()
			})

			mHide := systray.AddMenuItem(txt.Hide, "Ẩn ứng dụng xuống khay hệ thống")
			mHide.Click(func() {
				app.HideApp()
			})

			systray.AddSeparator()

			mAutoUpload := systray.AddMenuItemCheckbox(txt.AutoUpload, "Tự động đăng video khi đến khung giờ", st.AutoUploadEnabled)
			mAuto := systray.AddMenuItemCheckbox(txt.AutoStart, "Tự động chạy UpTik khi mở máy", st.AutoStart)
			mFolder := systray.AddMenuItem(txt.Folder, "Mở thư mục video đã cấu hình")

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

			mFolder.Click(func() {
				s := app.GetSettings()
				folder := s.VideoFolder
				if folder != "" {
					openFolderInFileManager(folder)
				}
			})

			systray.AddSeparator()

			mQuit := systray.AddMenuItem(txt.Quit, "Đóng và thoát ứng dụng UpTik")
			mQuit.Click(func() {
				app.QuitApp()
			})

			trayMu.Lock()
			mShowItem = mShow
			mHideItem = mHide
			mAutoUploadItem = mAutoUpload
			mAutoStartItem = mAuto
			mFolderItem = mFolder
			mQuitItem = mQuit
			applyTrayLocale(st.Locale, st.AutoUploadEnabled)
			trayMu.Unlock()

			// Register listener for external settings update
			app.RegisterSettingsUpdateListener(func(s Settings) {
				trayMu.Lock()
				defer trayMu.Unlock()
				applyTrayLocale(s.Locale, s.AutoUploadEnabled)
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
					} else {
						mAutoUploadItem.Uncheck()
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
