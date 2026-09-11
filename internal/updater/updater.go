package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultRepo      = "tymon5368/uptik"
	DefaultUserAgent = "UpTik-Desktop-Updater"
	DefaultTimeout   = 30 * time.Second
)

// UpdateInfo holds information about a potential software update.
type UpdateInfo struct {
	Available        bool   `json:"available"`
	CurrentVersion   string `json:"currentVersion"`
	LatestVersion    string `json:"latestVersion"`
	ReleaseNotes     string `json:"releaseNotes"`
	ReleaseURL       string `json:"releaseUrl"`
	PublishedAt      string `json:"publishedAt"`
	AssetURL         string `json:"assetUrl"`
	AssetName        string `json:"assetName"`
	AssetSize        int64  `json:"assetSize"`
	ExpectedChecksum string `json:"expectedChecksum"`
	IsSupported      bool   `json:"isSupported"`
}

// GitHubRelease represents the structure of GitHub release API response.
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	HTMLURL     string        `json:"html_url"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a release file asset.
type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Updater handles checking, downloading and applying self-updates.
type Updater struct {
	Repo       string
	HttpClient *http.Client
	mu         sync.Mutex
	isUpdating bool
}

// NewUpdater creates an Updater instance.
func NewUpdater(repo string) *Updater {
	if repo == "" {
		repo = DefaultRepo
	}
	return &Updater{
		Repo: repo,
		HttpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// CheckForUpdate queries GitHub for the latest release and determines if an update is available.
func (u *Updater) CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateInfo, error) {
	cleanCurrent := strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.Repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact release server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected release API response status: %s", resp.Status)
	}

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("failed to parse release metadata: %w", err)
	}

	latestVer := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	hasUpdate := CompareSemver(latestVer, cleanCurrent) > 0

	info := &UpdateInfo{
		Available:      hasUpdate,
		CurrentVersion: cleanCurrent,
		LatestVersion:  latestVer,
		ReleaseNotes:   rel.Body,
		ReleaseURL:     rel.HTMLURL,
		PublishedAt:    rel.PublishedAt,
		IsSupported:    true,
	}

	// Match asset for current platform
	matchedAsset, checksumAsset := u.matchAsset(rel.Assets, latestVer)
	if matchedAsset != nil {
		info.AssetName = matchedAsset.Name
		info.AssetURL = matchedAsset.BrowserDownloadURL
		info.AssetSize = matchedAsset.Size
	} else {
		info.IsSupported = false
	}

	// If checksum asset is available, fetch and extract checksum
	if checksumAsset != nil && matchedAsset != nil {
		if expectedHash, err := u.fetchChecksum(ctx, checksumAsset.BrowserDownloadURL, matchedAsset.Name); err == nil {
			info.ExpectedChecksum = expectedHash
		}
	}

	return info, nil
}

// matchAsset selects the appropriate binary archive for the current OS and architecture.
func (u *Updater) matchAsset(assets []GitHubAsset, version string) (*GitHubAsset, *GitHubAsset) {
	var targetAsset *GitHubAsset
	var checksumAsset *GitHubAsset

	for i := range assets {
		a := &assets[i]
		if a.Name == "SHA256SUMS.txt" {
			checksumAsset = a
			continue
		}

		switch runtime.GOOS {
		case "linux":
			if strings.HasSuffix(a.Name, "-linux-amd64.tar.gz") {
				targetAsset = a
			}
		case "darwin":
			if strings.HasSuffix(a.Name, "-macos-universal.zip") {
				targetAsset = a
			}
		case "windows":
			// Prefer portable zip for in-place binary replacement
			if strings.HasSuffix(a.Name, "-windows-amd64-portable.zip") {
				targetAsset = a
			}
		}
	}

	return targetAsset, checksumAsset
}

// fetchChecksum downloads SHA256SUMS.txt and finds the hash for the specified asset.
func (u *Updater) fetchChecksum(ctx context.Context, checksumURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	resp, err := u.HttpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hash := parts[0]
			name := filepath.Base(parts[1])
			if name == assetName {
				return hash, nil
			}
		}
	}

	return "", fmt.Errorf("checksum for %s not found in SHA256SUMS.txt", assetName)
}

// ProgressFunc reports download percentage (0 - 100).
type ProgressFunc func(percent int)

// ApplyUpdate downloads the release asset, verifies SHA256, extracts the executable,
// and replaces the running binary in-place.
func (u *Updater) ApplyUpdate(ctx context.Context, info *UpdateInfo, progress ProgressFunc) error {
	u.mu.Lock()
	if u.isUpdating {
		u.mu.Unlock()
		return fmt.Errorf("another update is already in progress")
	}
	u.isUpdating = true
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.isUpdating = false
		u.mu.Unlock()
	}()

	if info == nil || info.AssetURL == "" {
		return fmt.Errorf("invalid update information or missing asset URL")
	}

	// 1. Download asset to temporary file
	tmpFile, err := os.CreateTemp("", "uptik-update-*"+filepath.Ext(info.AssetName))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	resp, err := u.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %s", resp.Status)
	}

	totalSize := resp.ContentLength
	if totalSize <= 0 && info.AssetSize > 0 {
		totalSize = info.AssetSize
	}

	hasher := sha256.New()
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := tmpFile.Write(buf[:n]); wErr != nil {
				return fmt.Errorf("failed to write update file: %w", wErr)
			}
			hasher.Write(buf[:n])
			downloaded += int64(n)

			if progress != nil && totalSize > 0 {
				percent := int((float64(downloaded) / float64(totalSize)) * 100)
				if percent > 100 {
					percent = 100
				}
				progress(percent)
			}
		}

		if rErr == io.EOF {
			break
		}
		if rErr != nil {
			return fmt.Errorf("error reading download stream: %w", rErr)
		}
	}

	// 2. Verify SHA256 checksum if provided
	if info.ExpectedChecksum != "" {
		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(actualHash, info.ExpectedChecksum) {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", info.ExpectedChecksum, actualHash)
		}
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// 3. Extract new binary from downloaded archive
	newBinaryPath, err := extractBinary(tmpFile.Name(), info.AssetName)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	defer os.Remove(newBinaryPath)

	// 4. In-place binary replacement
	currentExec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	realExec, err := filepath.EvalSymlinks(currentExec)
	if err != nil {
		realExec = currentExec
	}

	if err := replaceExecutable(realExec, newBinaryPath); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	return nil
}

// extractBinary extracts the target executable from either a tar.gz or zip archive.
func extractBinary(archivePath, assetName string) (string, error) {
	tempExec, err := os.CreateTemp("", "uptik-bin-*")
	if err != nil {
		return "", err
	}
	defer tempExec.Close()

	if strings.HasSuffix(assetName, ".tar.gz") {
		f, err := os.Open(archivePath)
		if err != nil {
			return "", err
		}
		defer f.Close()

		gzr, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		found := false

		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}

			// Look for "uptik" executable (ignore directory prefix)
			base := filepath.Base(hdr.Name)
			if (base == "uptik" || base == "uptik.exe") && hdr.Typeflag == tar.TypeReg {
				if _, err := io.Copy(tempExec, tr); err != nil {
					return "", err
				}
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("executable 'uptik' not found in archive")
		}

	} else if strings.HasSuffix(assetName, ".zip") {
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return "", err
		}
		defer zr.Close()

		found := false
		for _, file := range zr.File {
			base := filepath.Base(file.Name)
			// Target "uptik.exe" or "uptik" inside MacOS bundle
			if (base == "uptik.exe" || base == "uptik") && !file.FileInfo().IsDir() {
				rc, err := file.Open()
				if err != nil {
					return "", err
				}
				_, err = io.Copy(tempExec, rc)
				rc.Close()
				if err != nil {
					return "", err
				}
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("executable not found in zip archive")
		}
	} else {
		return "", fmt.Errorf("unsupported archive format: %s", assetName)
	}

	// Ensure executable permission
	if err := os.Chmod(tempExec.Name(), 0755); err != nil {
		return "", err
	}

	return tempExec.Name(), nil
}

// replaceExecutable safely replaces the target executable file with the new binary.
func replaceExecutable(targetPath, newPath string) error {
	if runtime.GOOS == "windows" {
		// On Windows, a running executable cannot be directly overwritten,
		// but it CAN be renamed to an .old file while running!
		oldPath := targetPath + ".old"
		_ = os.Remove(oldPath) // remove any previous old file

		if err := os.Rename(targetPath, oldPath); err != nil {
			return fmt.Errorf("windows rename current exe failed: %w", err)
		}

		// Copy new binary into place
		if err := copyFile(newPath, targetPath, 0755); err != nil {
			// Rollback if copy fails
			_ = os.Rename(oldPath, targetPath)
			return fmt.Errorf("windows copy new exe failed: %w", err)
		}

		return nil
	}

	// POSIX systems (Linux & macOS):
	// Replacing an active inode or renaming over it is atomic and safe.
	tempTarget := targetPath + ".new"
	if err := copyFile(newPath, tempTarget, 0755); err != nil {
		return fmt.Errorf("copy to .new failed: %w", err)
	}

	if err := os.Rename(tempTarget, targetPath); err != nil {
		_ = os.Remove(tempTarget)
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// CleanOldBinaries removes any lingering .old files from previous updates (mainly on Windows).
func CleanOldBinaries() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)
}

// RestartApplication starts a new process of the current application and exits the current one.
func RestartApplication() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	realExec, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realExec = execPath
	}

	cmd := exec.Command(realExec, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start updated application: %w", err)
	}

	// Allow new process a moment to initialize before exiting
	go func() {
		time.Sleep(300 * time.Millisecond)
		os.Exit(0)
	}()

	return nil
}

// CompareSemver compares two semantic versions (v1 and v2).
// Returns:
//
//	 1 if v1 > v2
//	-1 if v1 < v2
//	 0 if v1 == v2
func CompareSemver(v1, v2 string) int {
	parts1 := parseSemver(v1)
	parts2 := parseSemver(v2)

	for i := 0; i < 3; i++ {
		if parts1[i] > parts2[i] {
			return 1
		}
		if parts1[i] < parts2[i] {
			return -1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	var res [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// Strip any suffix like -beta, -rc1
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}

	segments := strings.Split(v, ".")
	for i := 0; i < len(segments) && i < 3; i++ {
		if val, err := strconv.Atoi(segments[i]); err == nil {
			res[i] = val
		}
	}
	return res
}
