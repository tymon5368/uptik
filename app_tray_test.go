package main

import (
	"os"
	"path/filepath"
	"testing"
	"uptik/internal/domain"
)

func TestAppTrayAndAutostart(t *testing.T) {
	tempDir := t.TempDir()
	app := NewAppWithDBPath(filepath.Join(tempDir, "test_tray.db"))
	app.autostartMgr.SetCustomDir(tempDir)
	app.autostartMgr.SetCustomExec("/usr/bin/echo")

	// Initial default checks
	if !app.IsCloseToTray() {
		t.Errorf("expected CloseToTray to default to true")
	}
	if app.IsQuitting() {
		t.Errorf("expected IsQuitting to default to false")
	}
	if !app.IsWindowVisible() {
		t.Errorf("expected IsWindowVisible to default to true")
	}

	// Toggle window visibility
	app.ToggleAppWindow()
	if app.IsWindowVisible() {
		t.Errorf("expected IsWindowVisible to be false after toggle from visible")
	}

	app.ToggleAppWindow()
	if !app.IsWindowVisible() {
		t.Errorf("expected IsWindowVisible to be true after toggle from hidden")
	}

	// Test settings update listener
	var listenerCalled bool
	var receivedSettings domain.Settings
	app.RegisterSettingsUpdateListener(func(s domain.Settings) {
		listenerCalled = true
		receivedSettings = s
	})

	origSettings := app.GetSettings()
	defer func() {
		_ = app.SaveSettings(origSettings)
		_ = os.Remove(SettingsFileName)
	}()

	st := app.GetSettings()
	st.CloseToTray = false
	st.AutoStart = false
	if err := app.SaveSettings(st); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	if !listenerCalled {
		t.Errorf("expected onSettingsUpdated callback to be invoked")
	}
	if receivedSettings.CloseToTray != false {
		t.Errorf("expected CloseToTray to be false in callback")
	}
	if app.IsCloseToTray() != false {
		t.Errorf("expected IsCloseToTray to return false")
	}
}

func TestSaveSettingsAutostartFailureRollback(t *testing.T) {
	tempDir := t.TempDir()
	app := NewAppWithDBPath(filepath.Join(tempDir, "test_rollback.db"))

	// Set custom dir to a path where MkdirAll fails (child of a non-dir file)
	blockerFile := filepath.Join(tempDir, "blocker_file")
	if err := os.WriteFile(blockerFile, []byte("file"), 0644); err != nil {
		t.Fatalf("failed to create blocker file: %v", err)
	}
	impossibleDir := filepath.Join(blockerFile, "autostart_sub")
	app.autostartMgr.SetCustomDir(impossibleDir)
	app.autostartMgr.SetCustomExec("/usr/bin/echo")

	initial := app.GetSettings()
	initial.AutoStart = false
	if err := app.SaveSettings(initial); err != nil {
		t.Fatalf("failed to save initial settings: %v", err)
	}

	targetSettings := initial
	targetSettings.AutoStart = true

	err := app.SaveSettings(targetSettings)
	if err == nil {
		t.Fatalf("expected SaveSettings to return error when autostartMgr.Set fails, got nil")
	}

	current := app.GetSettings()
	if current.AutoStart != false {
		t.Errorf("expected app.settings.AutoStart to roll back to false upon autostart error, got %v", current.AutoStart)
	}
}
