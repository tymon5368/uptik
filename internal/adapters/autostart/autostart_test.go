package autostart

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAutostartManager(t *testing.T) {
	testDir := t.TempDir()
	testIcon := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic
	mgr := NewManager("uptik-unit-test", "UpTik Unit Test", testIcon)
	mgr.SetCustomDir(testDir)
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
		desktopPath := filepath.Join(testDir, "uptik-unit-test.desktop")
		content, err := os.ReadFile(desktopPath)
		if err != nil {
			t.Fatalf("failed to read desktop file: %v", err)
		}
		cStr := string(content)
		if !strings.Contains(cStr, "/usr/bin/echo") {
			t.Errorf("desktop file missing exec path: %s", cStr)
		}
		if !strings.Contains(cStr, "WEBKIT_DISABLE_COMPOSITING_MODE=1") {
			t.Errorf("desktop file missing WebKitGTK compositing flag: %s", cStr)
		}
		if !strings.Contains(cStr, "WEBKIT_DISABLE_DMABUF_RENDERER=1") {
			t.Errorf("desktop file missing WebKitGTK dmabuf flag: %s", cStr)
		}
		if strings.Contains(cStr, "--hidden") {
			t.Errorf("desktop file unexpectedly contains --hidden flag: %s", cStr)
		}
	}

	// Test enable with startHidden = true
	if err := mgr.Set(true, true); err != nil {
		t.Fatalf("failed to set autostart with startHidden: %v", err)
	}

	if runtime.GOOS == "linux" {
		desktopPath := filepath.Join(testDir, "uptik-unit-test.desktop")
		content, err := os.ReadFile(desktopPath)
		if err != nil {
			t.Fatalf("failed to read desktop file: %v", err)
		}
		cStr := string(content)
		if !strings.Contains(cStr, "--hidden") {
			t.Errorf("desktop file missing --hidden flag: %s", cStr)
		}
	}

	// Test IsEnabled returns false if target executable does not exist
	mgr.SetCustomExec("/nonexistent/binary/path")
	_ = mgr.Set(true, false)
	if mgr.IsEnabled() {
		t.Errorf("expected IsEnabled to return false for non-existent executable target")
	}

	// Restore valid exec for clean disable
	mgr.SetCustomExec("/usr/bin/echo")
	// Test disable
	if err := mgr.Set(false, false); err != nil {
		t.Fatalf("failed to disable autostart: %v", err)
	}

	if mgr.IsEnabled() {
		t.Fatalf("expected autostart to be disabled after Set(false, false)")
	}
	if mgr.HasEntry() {
		t.Fatalf("expected autostart entry to be removed after Set(false, false)")
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

func TestEphemeralBinaryDetection(t *testing.T) {
	if !isEphemeralBinary("/tmp/go-build111483460/b001/uptik.test") {
		t.Errorf("expected test binary in /tmp to be detected as ephemeral")
	}
	if !isEphemeralBinary("/tmp/go-build123/exe/uptik") {
		t.Errorf("expected go-build binary to be detected as ephemeral")
	}
	if isEphemeralBinary("/home/arch/.local/bin/uptik") {
		t.Errorf("did not expect installed binary to be detected as ephemeral")
	}
	if isEphemeralBinary("/usr/bin/uptik") {
		t.Errorf("did not expect /usr/bin to be detected as ephemeral")
	}
}

func TestNonExecutableFileRejected(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific permissions test")
	}

	tempDir := t.TempDir()
	nonExecFile := filepath.Join(tempDir, "fake_app")
	if err := os.WriteFile(nonExecFile, []byte("#!/bin/sh\necho hi\n"), 0644); err != nil {
		t.Fatalf("failed to create non-exec file: %v", err)
	}

	mgr := NewManager("test-non-exec", "Test Non Exec", nil)
	mgr.SetCustomDir(tempDir)
	mgr.SetCustomExec(nonExecFile)

	if err := mgr.Set(true, false); err != nil {
		t.Fatalf("failed to set autostart: %v", err)
	}

	// Mode 0644 lacks execute bit; IsEnabled must reject it
	if mgr.IsEnabled() {
		t.Errorf("expected IsEnabled to return false for non-executable file (0644)")
	}

	// Add execute permission
	if err := os.Chmod(nonExecFile, 0755); err != nil {
		t.Fatalf("failed to chmod 0755: %v", err)
	}

	// Now it must be accepted
	if !mgr.IsEnabled() {
		t.Errorf("expected IsEnabled to return true after chmod 0755")
	}
}

func TestEphemeralBinaryStillExistingRejected(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test")
	}

	tempDir := t.TempDir()
	// Create an executable file with a .test suffix that physically exists right now
	testBinFile := filepath.Join(tempDir, "my_unit_test.test")
	if err := os.WriteFile(testBinFile, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to create test binary: %v", err)
	}

	mgr := NewManager("test-ephemeral", "Test Ephemeral", nil)
	mgr.SetCustomDir(tempDir)
	mgr.SetCustomExec(testBinFile)

	if err := mgr.Set(true, false); err != nil {
		t.Fatalf("failed to set autostart: %v", err)
	}

	// Even though the file physically exists on disk and is 0755, IsEnabled must reject it
	// because it is an ephemeral/test binary and would vanish or fail on reboot
	if mgr.IsEnabled() {
		t.Errorf("expected IsEnabled to return false for ephemeral .test binary even while it exists")
	}
}

func TestPathWithSpacesAndQuoting(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific quoting test")
	}

	tempDir := t.TempDir()
	spaceDir := filepath.Join(tempDir, "My Application Dir")
	if err := os.MkdirAll(spaceDir, 0755); err != nil {
		t.Fatalf("failed to create dir with spaces: %v", err)
	}

	spaceBin := filepath.Join(spaceDir, "uptik launcher")
	if err := os.WriteFile(spaceBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to create binary with space: %v", err)
	}

	mgr := NewManager("test-spaces", "Test Spaces", nil)
	mgr.SetCustomDir(tempDir)
	mgr.SetCustomExec(spaceBin)

	if err := mgr.Set(true, true); err != nil {
		t.Fatalf("failed to set autostart with spaces: %v", err)
	}

	// Verify desktop file contains correctly quoted path
	desktopPath := mgr.GetDesktopFilePath()
	contentBytes, err := os.ReadFile(desktopPath)
	if err != nil {
		t.Fatalf("failed to read desktop file: %v", err)
	}
	content := string(contentBytes)

	expectedQuoted := `"` + spaceBin + `"`
	if !strings.Contains(content, expectedQuoted) {
		t.Errorf("expected desktop file to contain quoted path %s, got:\n%s", expectedQuoted, content)
	}

	// Verify extractExecTarget extracts the full path with spaces, not just the first word
	extracted := extractExecTarget(content)
	if extracted != spaceBin {
		t.Errorf("expected extractExecTarget to return %q, got %q", spaceBin, extracted)
	}

	// Verify IsEnabled succeeds with spaces in path
	if !mgr.IsEnabled() {
		t.Errorf("expected IsEnabled to be true for valid executable with spaces in path")
	}
}

func TestParseDesktopExecArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple unquoted",
			input:    "env FOO=BAR /usr/bin/uptik --hidden",
			expected: []string{"env", "FOO=BAR", "/usr/bin/uptik", "--hidden"},
		},
		{
			name:     "double quoted path with spaces",
			input:    `env WEBKIT_1=1 "/opt/Up Tik/uptik" --hidden`,
			expected: []string{"env", "WEBKIT_1=1", "/opt/Up Tik/uptik", "--hidden"},
		},
		{
			name:     "single quoted path with spaces",
			input:    `env WEBKIT_1=1 '/opt/Up Tik/uptik' --hidden`,
			expected: []string{"env", "WEBKIT_1=1", "/opt/Up Tik/uptik", "--hidden"},
		},
		{
			name:     "backslash escaped space",
			input:    `/opt/Up\ Tik/uptik --hidden`,
			expected: []string{"/opt/Up Tik/uptik", "--hidden"},
		},
		{
			name:     "multiple spaces between arguments",
			input:    `  /usr/bin/uptik    --hidden   -m  `,
			expected: []string{"/usr/bin/uptik", "--hidden", "-m"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDesktopExecArgs(tc.input)
			if len(got) != len(tc.expected) {
				t.Fatalf("token length mismatch: got %v, want %v", got, tc.expected)
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("token %d mismatch: got %q, want %q", i, got[i], tc.expected[i])
				}
			}
		})
	}
}

func TestExtractExecTarget(t *testing.T) {
	desktopContent := `[Desktop Entry]
Type=Application
Name=UpTik
Exec=env WEBKIT_DISABLE_COMPOSITING_MODE=1 WEBKIT_DISABLE_DMABUF_RENDERER=1 /home/arch/.local/bin/uptik --hidden
Icon=uptik
`
	target := extractExecTarget(desktopContent)
	if target != "/home/arch/.local/bin/uptik" {
		t.Errorf("expected target to be /home/arch/.local/bin/uptik, got %s", target)
	}

	quotedContent := `[Desktop Entry]
Exec="/usr/bin/echo" "--hidden"
`
	target2 := extractExecTarget(quotedContent)
	if target2 != "/usr/bin/echo" {
		t.Errorf("expected target to be /usr/bin/echo, got %s", target2)
	}

	spacesContent := `[Desktop Entry]
Exec=env WEBKIT_FLAG=1 "/opt/Up Tik/bin/uptik" --hidden
`
	target3 := extractExecTarget(spacesContent)
	if target3 != "/opt/Up Tik/bin/uptik" {
		t.Errorf("expected target to be /opt/Up Tik/bin/uptik, got %s", target3)
	}
}

func TestFindInstalledBinary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific installed binary test")
	}

	installed := findInstalledBinary("uptik")
	if installed == "" {
		t.Logf("uptik not installed in standard locations")
	} else {
		if !strings.HasSuffix(installed, "/uptik") {
			t.Errorf("expected path to end in /uptik, got %s", installed)
		}
	}
}

func TestQuoteDesktopArgReservedChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"/opt/app (x86)/bin", `"/opt/app (x86)/bin"`},
		{"/opt/app;cmd", `"/opt/app;cmd"`},
		{"/opt/app&bg", `"/opt/app&bg"`},
		{"/opt/app$var", `"/opt/app\$var"`},
		{"/opt/app`cmd`", "\"/opt/app\\`cmd\\`\""},
		{"/opt/app\"quotes\"", `"/opt/app\"quotes\""`},
		{"/opt/app%20dir", `"/opt/app%%20dir"`},
	}

	for _, tc := range tests {
		got := quoteDesktopArg(tc.input)
		if got != tc.expected {
			t.Errorf("quoteDesktopArg(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestExtractExecTargetWithEqualsInPath(t *testing.T) {
	content := `[Desktop Entry]
Exec=env WEBKIT_FLAG=1 "/opt/app=v2/uptik" --hidden
`
	target := extractExecTarget(content)
	if target != "/opt/app=v2/uptik" {
		t.Errorf("expected /opt/app=v2/uptik, got %q", target)
	}

	if !isEnvAssignment("WEBKIT_FLAG=1") {
		t.Errorf("expected WEBKIT_FLAG=1 to be identified as env assignment")
	}
	if isEnvAssignment("/opt/app=v2/uptik") {
		t.Errorf("did not expect path with slash to be identified as env assignment")
	}
}

func TestExtractExecTargetWithAbsoluteEnv(t *testing.T) {
	content := `[Desktop Entry]
Exec=/usr/bin/env WEBKIT_FLAG=1 /home/arch/.local/bin/uptik --hidden
`
	target := extractExecTarget(content)
	if target != "/home/arch/.local/bin/uptik" {
		t.Errorf("expected /home/arch/.local/bin/uptik, got %q", target)
	}
}

func TestIsExplicitlyDisabled(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test")
	}

	tempDir := t.TempDir()
	mgr := NewManager("test-disabled", "Test Disabled", nil)
	mgr.SetCustomDir(tempDir)

	if mgr.IsExplicitlyDisabled() {
		t.Errorf("expected IsExplicitlyDisabled false when file does not exist")
	}

	desktopFile := mgr.GetDesktopFilePath()
	if err := os.WriteFile(desktopFile, []byte("[Desktop Entry]\nHidden=true\n"), 0644); err != nil {
		t.Fatalf("failed to write desktop file: %v", err)
	}

	if !mgr.IsExplicitlyDisabled() {
		t.Errorf("expected IsExplicitlyDisabled true when Hidden=true")
	}

	if err := os.WriteFile(desktopFile, []byte("[Desktop Entry]\nX-GNOME-Autostart-enabled=false\n"), 0644); err != nil {
		t.Fatalf("failed to write desktop file: %v", err)
	}

	if !mgr.IsExplicitlyDisabled() {
		t.Errorf("expected IsExplicitlyDisabled true when X-GNOME-Autostart-enabled=false")
	}
}

