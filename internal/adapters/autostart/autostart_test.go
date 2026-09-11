package autostart

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAutostartManager(t *testing.T) {
	testIcon := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic
	mgr := NewManager("uptik-unit-test", "UpTik Unit Test", testIcon)
	mgr.SetCustomExec("/usr/bin/echo")

	// Ensure clean initial state
	_ = mgr.Set(false, false)

	if mgr.IsEnabled() {
		t.Fatalf("expected autostart to be disabled initially")
	}

	// Test enable without hidden
	if err := mgr.Set(true, false); err != nil {
		t.Fatalf("failed to enable autostart: %v", err)
	}

	if !mgr.IsEnabled() {
		t.Fatalf("expected autostart to be enabled after Set(true, false)")
	}

	if runtime.GOOS == "linux" {
		home, _ := os.UserHomeDir()
		desktopPath := filepath.Join(home, ".config", "autostart", "uptik-unit-test.desktop")
		content, err := os.ReadFile(desktopPath)
		if err != nil {
			t.Fatalf("failed to read desktop file: %v", err)
		}
		if !strings.Contains(string(content), "/usr/bin/echo") {
			t.Errorf("desktop file missing exec path: %s", string(content))
		}
	}

	// Test enable with startHidden = true
	if err := mgr.Set(true, true); err != nil {
		t.Fatalf("failed to set autostart with startHidden: %v", err)
	}

	if runtime.GOOS == "linux" {
		home, _ := os.UserHomeDir()
		desktopPath := filepath.Join(home, ".config", "autostart", "uptik-unit-test.desktop")
		content, err := os.ReadFile(desktopPath)
		if err != nil {
			t.Fatalf("failed to read desktop file: %v", err)
		}
		if !strings.Contains(string(content), "--hidden") {
			t.Errorf("desktop file missing --hidden flag: %s", string(content))
		}
	}

	// Test disable
	if err := mgr.Set(false, false); err != nil {
		t.Fatalf("failed to disable autostart: %v", err)
	}

	if mgr.IsEnabled() {
		t.Fatalf("expected autostart to be disabled after Set(false, false)")
	}
}

func TestEnsureDesktopIcon(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific icon test")
	}

	dummyData := []byte("fake-png-data")
	iconPath := EnsureDesktopIcon("uptik-icon-test", dummyData)
	if iconPath == "" {
		t.Fatalf("expected valid icon path")
	}

	data, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatalf("failed to read written icon: %v", err)
	}
	if string(data) != "fake-png-data" {
		t.Fatalf("icon content mismatch: got %s", string(data))
	}

	_ = os.Remove(iconPath)
}
