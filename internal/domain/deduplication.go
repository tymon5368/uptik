package domain

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Regex for resolution indicators like (1080p_50fps_H264-128kbit_AAC-English) or (1080p)
	resRegex = regexp.MustCompile(`\s*\(\d+p[^\)]*\)`)
	// Regex for multiple spaces
	spacesRegex = regexp.MustCompile(`\s+`)
	// Regex to remove emojis
	emojiRegex = regexp.MustCompile(`[\x{1F1E0}-\x{1F1FF}\x{1F300}-\x{1F5FF}\x{1F600}-\x{1F64F}\x{1F680}-\x{1F6FF}\x{1F700}-\x{1F77F}\x{1F780}-\x{1F7FF}\x{1F800}-\x{1F8FF}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{2600}-\x{27BF}\x{24C2}-\x{1F251}]`)
)

// CleanVideoTitle strips resolution tags, emojis, multiple whitespace and appends default tag
func CleanVideoTitle(filename string, tag string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = resRegex.ReplaceAllString(base, "")
	base = emojiRegex.ReplaceAllString(base, "")
	base = spacesRegex.ReplaceAllString(base, " ")
	base = strings.TrimSpace(base)

	if tag != "" && !strings.Contains(strings.ToLower(base), strings.ToLower(tag)) {
		base = fmt.Sprintf("%s %s", base, tag)
	}
	return base
}

// FormatFileSize formats raw bytes into human readable binary format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// HashString generates an 8-character hex hash for identifiers
func HashString(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:8])
}
