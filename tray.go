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
	ShowTip       string
	Hide          string
	HideTip       string
	AutoUpload    string
	AutoUploadTip string
	AutoStart     string
	AutoStartTip  string
	Folder        string
	FolderTip     string
	Quit          string
	QuitTip       string
	Tooltip       string
	TooltipActive string
}

func getTrayTexts(locale string) trayTexts {
	switch locale {
	case "de":
		return trayTexts{
			Show:          "UpTik öffnen",
			ShowTip:       "UpTik-Anwendungsfenster anzeigen",
			Hide:          "Im Tray minimieren",
			HideTip:       "In den Infobereich der Taskleiste minimieren",
			AutoUpload:    "Automatischer Upload (Publish Now)",
			AutoUploadTip: "Videos zu den Golden Hours automatisch hochladen",
			AutoStart:     "Mit dem System starten",
			AutoStartTip:  "UpTik beim Hochfahren des Systems starten",
			Folder:        "Video-Ordner öffnen",
			FolderTip:     "Konfigurierten Videoordner öffnen",
			Quit:          "Vollständig beenden",
			QuitTip:       "UpTik schließen und beenden",
			Tooltip:       "UpTik - Video Auto Scheduler (Läuft im Hintergrund)",
			TooltipActive: "UpTik - Automatischer Upload aktiv",
		}
	case "ja":
		return trayTexts{
			Show:          "UpTik を開く",
			ShowTip:       "アプリウィンドウを表示",
			Hide:          "トレイに最小化",
			HideTip:       "システムトレイに最小化",
			AutoUpload:    "自動投稿 (今すぐ公開)",
			AutoUploadTip: "設定された時間帯に動画を自動投稿",
			AutoStart:     "システム起動時に実行",
			AutoStartTip:  "PC起動時にUpTikを自動起動",
			Folder:        "動画フォルダを開く",
			FolderTip:     "設定された動画フォルダを開く",
			Quit:          "完全に終了",
			QuitTip:       "UpTikを完全に終了",
			Tooltip:       "UpTik - 動画自動スケジューラー (バックグラウンド実行中)",
			TooltipActive: "UpTik - 自動投稿有効",
		}
	case "ar":
		return trayTexts{
			Show:          "فتح UpTik",
			ShowTip:       "إظهار نافذة التطبيق",
			Hide:          "إخفاء في شريط المهام",
			HideTip:       "إخفاء في علبة النظام",
			AutoUpload:    "نشر تلقائي (النشر الآن)",
			AutoUploadTip: "نشر مقاطع الفيديو تلقائياً في الساعات المحددة",
			AutoStart:     "تشغيل مع بدء النظام",
			AutoStartTip:  "تشغيل UpTik تلقائياً عند بدء التشغيل",
			Folder:        "فتح مجلد الفيديو",
			FolderTip:     "فتح مجلد الفيديو المكوّن",
			Quit:          "إنهاء كلياً",
			QuitTip:       "إغلاق وإنهاء UpTik",
			Tooltip:       "UpTik - مجدول الفيديو التلقائي (قيد التشغيل في الخلفية)",
			TooltipActive: "UpTik - النشر التلقائي مفعّل",
		}
	case "fr":
		return trayTexts{
			Show:          "Ouvrir UpTik",
			ShowTip:       "Afficher la fenêtre de l'application",
			Hide:          "Réduire dans la barre",
			HideTip:       "Réduire dans la zone de notification",
			AutoUpload:    "Publication automatique (Publish Now)",
			AutoUploadTip: "Publier automatiquement les vidéos aux heures configurées",
			AutoStart:     "Démarrer avec le système",
			AutoStartTip:  "Lancer UpTik au démarrage du système",
			Folder:        "Ouvrir le dossier vidéo",
			FolderTip:     "Ouvrir le dossier vidéo configuré",
			Quit:          "Quitter complètement",
			QuitTip:       "Fermer et quitter UpTik",
			Tooltip:       "UpTik - Planificateur vidéo (En arrière-plan)",
			TooltipActive: "UpTik - Publication automatique activée",
		}
	case "vi":
		return trayTexts{
			Show:          "Mở UpTik",
			ShowTip:       "Hiện cửa sổ ứng dụng",
			Hide:          "Ẩn vào khay",
			HideTip:       "Ẩn ứng dụng xuống khay hệ thống",
			AutoUpload:    "Tự động đăng (Publish Now)",
			AutoUploadTip: "Tự động đăng video khi đến khung giờ",
			AutoStart:     "Khởi động cùng hệ thống",
			AutoStartTip:  "Tự động chạy UpTik khi mở máy",
			Folder:        "Mở thư mục video",
			FolderTip:     "Mở thư mục video đã cấu hình",
			Quit:          "Thoát hoàn toàn",
			QuitTip:       "Đóng và thoát ứng dụng UpTik",
			Tooltip:       "UpTik - TikTok Auto Scheduler (Đang chạy ngầm)",
			TooltipActive: "UpTik - Tự động đăng video theo khung giờ (Đang bật)",
		}
	default: // "en"
		return trayTexts{
			Show:          "Open UpTik",
			ShowTip:       "Show application window",
			Hide:          "Minimize to Tray",
			HideTip:       "Minimize application to system tray",
			AutoUpload:    "Auto Publish (Publish Now)",
			AutoUploadTip: "Automatically publish videos at scheduled golden hours",
			AutoStart:     "Start with System",
			AutoStartTip:  "Launch UpTik on system startup",
			Folder:        "Open Video Folder",
			FolderTip:     "Open configured video folder",
			Quit:          "Quit Completely",
			QuitTip:       "Close and exit UpTik",
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

			mShow := systray.AddMenuItem(txt.Show, txt.ShowTip)
			mShow.Click(func() {
				app.ShowApp()
			})

			mHide := systray.AddMenuItem(txt.Hide, txt.HideTip)
			mHide.Click(func() {
				app.HideApp()
			})

			systray.AddSeparator()

			mAutoUpload := systray.AddMenuItemCheckbox(txt.AutoUpload, txt.AutoUploadTip, st.AutoUploadEnabled)
			mAuto := systray.AddMenuItemCheckbox(txt.AutoStart, txt.AutoStartTip, st.AutoStart)
			mFolder := systray.AddMenuItem(txt.Folder, txt.FolderTip)

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

			mQuit := systray.AddMenuItem(txt.Quit, txt.QuitTip)
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
