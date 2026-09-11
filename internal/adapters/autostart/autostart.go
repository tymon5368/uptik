package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/emersion/go-autostart"
)

// EnsureDesktopIcon ensures the application icon is written to a standard icon location on Linux
func EnsureDesktopIcon(appName string, iconBytes []byte) string {
	if len(iconBytes) == 0 {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}

	pixmapsDir := filepath.Join(home, ".local", "share", "pixmaps")
	_ = os.MkdirAll(pixmapsDir, 0755)

	iconPath := filepath.Join(pixmapsDir, appName+".png")
	_ = os.WriteFile(iconPath, iconBytes, 0644)
	return iconPath
}

// Manager handles autostart configuration for the application
type Manager struct {
	AppName     string
	DisplayName string
	IconPath    string
	customExec  string
}

// NewManager creates a new autostart Manager
func NewManager(appName, displayName string, iconBytes []byte) *Manager {
	iconPath := ""
	if runtime.GOOS == "linux" {
		iconPath = EnsureDesktopIcon(appName, iconBytes)
	}

	return &Manager{
		AppName:     appName,
		DisplayName: displayName,
		IconPath:    iconPath,
	}
}

// SetCustomExec allows overriding the binary path (useful for testing)
func (m *Manager) SetCustomExec(execPath string) {
	m.customExec = execPath
}

func (m *Manager) resolveExec() (string, error) {
	if m.customExec != "" {
		return m.customExec, nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("không thể tìm đường dẫn file thực thi: %w", err)
	}

	if resolved, err := filepath.EvalSymlinks(execPath); err == nil && resolved != "" {
		execPath = resolved
	}

	return execPath, nil
}

func (m *Manager) getApp(hidden bool) (*autostart.App, error) {
	execPath, err := m.resolveExec()
	if err != nil {
		return nil, err
	}

	execArgs := []string{execPath}
	if hidden {
		execArgs = append(execArgs, "--hidden")
	}

	return &autostart.App{
		Name:        m.AppName,
		DisplayName: m.DisplayName,
		Exec:        execArgs,
		Icon:        m.IconPath,
	}, nil
}

// IsEnabled returns true if autostart is currently enabled in the OS
func (m *Manager) IsEnabled() bool {
	app, err := m.getApp(false)
	if err != nil {
		return false
	}
	return app.IsEnabled()
}

// Set enables or disables autostart with optional hidden/minimized startup
func (m *Manager) Set(enabled bool, startHidden bool) error {
	app, err := m.getApp(startHidden)
	if err != nil {
		return err
	}

	if enabled {
		return app.Enable()
	}
	return app.Disable()
}
