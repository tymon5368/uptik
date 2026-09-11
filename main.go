package main

import (
	"context"
	"embed"
	"os"
	"strings"
	"syscall"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func init() {
	// Fix WebKitGTK 4.1 blank / black screen on Linux (Intel TigerLake Iris Xe / Nvidia / Mesa compositor bug)
	needsRestart := false
	if os.Getenv("WEBKIT_DISABLE_COMPOSITING_MODE") != "1" {
		_ = os.Setenv("WEBKIT_DISABLE_COMPOSITING_MODE", "1")
		needsRestart = true
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") != "1" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
		needsRestart = true
	}
	if needsRestart && os.Getenv("UPTIK_REEXEC") != "1" {
		_ = os.Setenv("UPTIK_REEXEC", "1")
		if exe, err := os.Executable(); err == nil {
			_ = syscall.Exec(exe, os.Args, os.Environ())
		}
	}
}

func main() {

	// Check if started with --hidden, --minimized, or -m (e.g. from system autostart)
	startHidden := false
	for _, arg := range os.Args[1:] {
		low := strings.ToLower(arg)
		if low == "--hidden" || low == "-hidden" || low == "--minimized" || low == "-m" {
			startHidden = true
			break
		}
	}

	// Create an instance of the app structure
	app := NewApp()
	if startHidden {
		app.SetWindowVisible(false)
	}

	// Start system tray in background goroutine
	go initTray(app, appIcon)

	// Create application with options
	err := wails.Run(&options.App{
		Title:       "UpTik - TikTok Auto Scheduler (Netflix Edition)",
		Width:       1280,
		Height:      850,
		MinWidth:    1024,
		MinHeight:   700,
		StartHidden: startHidden,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 20, B: 20, A: 255},
		OnStartup:        app.startup,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.IsQuitting() {
				return false // Allow clean exit
			}
			if app.IsCloseToTray() {
				app.HideApp()
				return true // Prevent close, run in background
			}
			return false // Normal exit
		},
		OnShutdown: func(ctx context.Context) {
			systray.Quit()
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "uptik-omnichannel-scheduler-instance-lock",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				app.ShowApp()
			},
		},
		EnableDefaultContextMenu: true,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon:             appIcon,
			ProgramName:      "uptik",
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
