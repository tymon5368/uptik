package main

import (
	"os"
	"testing"
	"uptik/internal/domain"
)

func TestAppTrayAndAutostart(t *testing.T) {
	app := NewApp()

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
