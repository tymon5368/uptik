package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
	customDir   string
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

// SetCustomDir allows overriding the autostart directory (useful for testing)
func (m *Manager) SetCustomDir(dir string) {
	m.customDir = dir
}

// GetAutostartDir returns the directory where autostart entries are stored
func (m *Manager) GetAutostartDir() string {
	if m.customDir != "" {
		return m.customDir
	}

	// In unit tests, isolate by default to avoid contaminating user's ~/.config/autostart
	if testing.Testing() {
		return filepath.Join(os.TempDir(), "uptik-test-autostart")
	}

	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "autostart")
	}

	home, _ := os.UserHomeDir()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "autostart")
}

// GetDesktopFilePath returns the full path to the XDG desktop file on Linux
func (m *Manager) GetDesktopFilePath() string {
	return filepath.Join(m.GetAutostartDir(), m.AppName+".desktop")
}

func isEphemeralBinary(p string) bool {
	low := strings.ToLower(p)
	if strings.Contains(low, "/go-build") ||
		strings.Contains(low, "\\go-build") ||
		strings.HasSuffix(low, ".test") ||
		strings.HasSuffix(low, ".test.exe") {
		return true
	}
	if !testing.Testing() {
		if strings.HasPrefix(low, strings.ToLower(os.TempDir())) ||
			strings.HasPrefix(low, "/tmp/") ||
			strings.HasPrefix(low, "/var/tmp/") {
			return true
		}
	}
	return false
}

func findInstalledBinary(appName string) string {
	// 1. Check user local bin (~/.local/bin/uptik)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		localBin := filepath.Join(home, ".local", "bin", appName)
		if fi, err := os.Stat(localBin); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
			return localBin
		}
	}

	// 2. Check PATH
	if lp, err := exec.LookPath(appName); err == nil && lp != "" {
		if !isEphemeralBinary(lp) {
			return lp
		}
	}

	// 3. Check system standard paths
	for _, dir := range []string{"/usr/local/bin", "/usr/bin"} {
		p := filepath.Join(dir, appName)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
			return p
		}
	}

	return ""
}

// ResolveExec resolves the target executable path
func (m *Manager) ResolveExec() (string, error) {
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

	// Check if the current running executable is ephemeral/temporary (e.g. go test or go run)
	if isEphemeralBinary(execPath) {
		if installed := findInstalledBinary(m.AppName); installed != "" {
			return installed, nil
		}

		if testing.Testing() {
			return "/bin/true", nil
		}

		return "", fmt.Errorf("không thể bật autostart từ tiến trình tạm thời (%s) khi chưa cài đặt %s vào hệ thống", execPath, m.AppName)
	}

	// If running from source build tree, prefer permanently installed binary if available
	if strings.Contains(execPath, "/build/bin/") || strings.Contains(execPath, "/target/") {
		if installed := findInstalledBinary(m.AppName); installed != "" {
			return installed, nil
		}
	}

	return execPath, nil
}

// parseDesktopExecArgs parses command-line arguments in an Exec= line according to the XDG spec,
// properly respecting single quotes, double quotes, and backslash escapes across whitespace.
func parseDesktopExecArgs(raw string) []string {
	var tokens []string
	var cur strings.Builder
	inDouble := false
	inSingle := false
	escaped := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if escaped {
			cur.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if (ch == ' ' || ch == '\t') && !inDouble && !inSingle {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}

		cur.WriteByte(ch)
	}

	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}

	return tokens
}

// quoteDesktopArg quotes a command-line argument for an Exec= line if it contains spaces or reserved chars.
func quoteDesktopArg(arg string) string {
	if !strings.ContainsAny(arg, " \t\"'\\$") {
		return arg
	}
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(arg); i++ {
		c := arg[i]
		if c == '"' || c == '\\' || c == '$' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

func extractExecTarget(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Exec=") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "Exec="))
		tokens := parseDesktopExecArgs(raw)

		for _, token := range tokens {
			// Skip env binary
			if token == "env" {
				continue
			}
			// Skip environment variable assignments VAR=VALUE
			if strings.Contains(token, "=") {
				continue
			}
			// Skip command flags
			if strings.HasPrefix(token, "-") {
				continue
			}
			return token
		}
	}
	return ""
}

// HasEntry returns true if the autostart entry file exists on disk
func (m *Manager) HasEntry() bool {
	if runtime.GOOS != "linux" {
		return m.IsEnabled()
	}
	_, err := os.Stat(m.GetDesktopFilePath())
	return err == nil
}

// IsEnabled returns true if autostart is currently enabled and pointing to a valid, permanent executable
func (m *Manager) IsEnabled() bool {
	if runtime.GOOS != "linux" {
		app, err := m.getApp(false)
		if err != nil {
			return false
		}
		return app.IsEnabled()
	}

	desktopPath := m.GetDesktopFilePath()
	contentBytes, err := os.ReadFile(desktopPath)
	if err != nil {
		return false
	}
	content := string(contentBytes)

	// Check if disabled by desktop environment
	if strings.Contains(content, "Hidden=true") || strings.Contains(content, "X-GNOME-Autostart-enabled=false") {
		return false
	}

	// Verify the executable target
	target := extractExecTarget(content)
	if target == "" {
		return false
	}

	// Reject ephemeral/test binaries even if they temporarily exist on disk
	if isEphemeralBinary(target) {
		return false
	}

	fi, err := os.Stat(target)
	if err != nil || fi.IsDir() {
		return false
	}

	// On non-Windows platforms, verify the target file has executable permissions (mode & 0111 != 0)
	if runtime.GOOS != "windows" && fi.Mode()&0111 == 0 {
		return false
	}

	return true
}

func (m *Manager) writeDesktopFile(execPath string, startHidden bool) error {
	dir := m.GetAutostartDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("không thể tạo thư mục autostart %s: %w", dir, err)
	}

	icon := m.IconPath
	if icon == "" {
		icon = m.AppName
	}

	quotedExec := quoteDesktopArg(execPath)
	execCmd := fmt.Sprintf("env WEBKIT_DISABLE_COMPOSITING_MODE=1 WEBKIT_DISABLE_DMABUF_RENDERER=1 %s", quotedExec)
	if startHidden {
		execCmd += " --hidden"
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
GenericName=TikTok Auto Scheduler
Comment=TikTok Auto Scheduler & Omnichannel Video Studio
Exec=%s
Icon=%s
Terminal=false
StartupNotify=false
Categories=Utility;AudioVideo;
StartupWMClass=%s
X-GNOME-Autostart-enabled=true
`, m.DisplayName, execCmd, icon, m.AppName)

	destPath := m.GetDesktopFilePath()
	tmpPath := destPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("không thể ghi file autostart tạm thời %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("không thể cập nhật file autostart %s: %w", destPath, err)
	}

	return nil
}

func (m *Manager) removeDesktopFile() error {
	destPath := m.GetDesktopFilePath()
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("không thể xóa file autostart %s: %w", destPath, err)
	}
	return nil
}

func (m *Manager) getApp(hidden bool) (*autostart.App, error) {
	execPath, err := m.ResolveExec()
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

// Set enables or disables autostart with optional hidden/minimized startup
func (m *Manager) Set(enabled bool, startHidden bool) error {
	if runtime.GOOS != "linux" {
		app, err := m.getApp(startHidden)
		if err != nil {
			return err
		}

		if enabled {
			return app.Enable()
		}
		return app.Disable()
	}

	if !enabled {
		return m.removeDesktopFile()
	}

	execPath, err := m.ResolveExec()
	if err != nil {
		return err
	}

	return m.writeDesktopFile(execPath, startHidden)
}
