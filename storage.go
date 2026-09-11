package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"uptik/internal/domain"
)

const (
	SettingsFileName = "settings.json"
	HistoryFileName  = "tiktok_schedule_history.json"
	DefaultFolder    = "/home/arch/Downloads/Movie Nights - Uploads from Movie Nights"
	DefaultUserData  = "/home/arch/.config/google-chrome-mcp"
	DefaultChrome    = "/opt/google/chrome/chrome"
)

func GetDefaultSettings() Settings {
	return Settings{
		VideoFolder:       DefaultFolder,
		ChromeUserDataDir: DefaultUserData,
		ChromePath:        DefaultChrome,
		DefaultTag:        "#phimbop",
		GoldenHours:       []string{"11:30", "18:30", "21:30"},
		MaxDays:           30,
		Headless:          false,
		CdpPort:           9222,
		EnabledChannels:   []string{"tiktok", "youtube"},
		CloseToTray:       true,
		AutoStart:         false,
		StartHidden:       true,
		PublishMode:       domain.PublishModeSchedule,
		AutoUploadEnabled: false,
		MissedSlotPolicy:  "skip",
	}
}

func LoadSettings() Settings {
	s := GetDefaultSettings()
	data, err := os.ReadFile(SettingsFileName)
	if err == nil {
		_ = json.Unmarshal(data, &s)
	}
	if len(s.EnabledChannels) == 0 {
		s.EnabledChannels = []string{"tiktok", "youtube"}
	}
	if s.PublishMode == "" {
		s.PublishMode = domain.PublishModeSchedule
	}
	if s.MissedSlotPolicy == "" {
		s.MissedSlotPolicy = "skip"
	}
	s.GoldenHours = domain.ValidateAndSortHours(s.GoldenHours)
	return s
}

func SaveSettingsToFile(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SettingsFileName, data, 0644)
}

func LoadHistory() []HistoryRecord {
	var records []HistoryRecord
	data, err := os.ReadFile(HistoryFileName)
	if err == nil {
		_ = json.Unmarshal(data, &records)
	}
	return records
}

func SaveHistoryToFile(records []HistoryRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(HistoryFileName, data, 0644)
}

func CleanVideoTitle(filename string, tag string) string {
	return domain.CleanVideoTitle(filename, tag)
}

func FormatFileSize(bytes int64) string {
	return domain.FormatFileSize(bytes)
}

func HashString(s string) string {
	return domain.HashString(s)
}

func GetHourLabel(h string) string {
	return domain.GetHourLabel(h)
}

func ScanPendingVideos(folder string, tag string) ([]VideoItem, error) {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return nil, fmt.Errorf("thư mục không tồn tại: %s", folder)
	}

	uploadedDir := filepath.Join(folder, "uploaded")
	_ = os.MkdirAll(uploadedDir, 0755)

	uploadedFiles := make(map[string]bool)
	if entries, err := os.ReadDir(uploadedDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				uploadedFiles[e.Name()] = true
			}
		}
	}

	history := LoadHistory()
	historyFiles := make(map[string]bool)
	for _, h := range history {
		historyFiles[h.Filename] = true
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	var items []VideoItem
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".mp4") {
			continue
		}
		if uploadedFiles[name] || historyFiles[name] {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		clean := CleanVideoTitle(name, tag)
		channels := make(map[string]ChannelStatus)
		channels["tiktok"] = ChannelStatus{Status: "pending"}
		channels["youtube"] = ChannelStatus{Status: "pending"}
		channels["facebook"] = ChannelStatus{Status: "pending"}

		items = append(items, VideoItem{
			ID:            HashString(name),
			Filename:      name,
			FullPath:      filepath.Join(folder, name),
			CleanTitle:    clean,
			CustomTitle:   clean,
			FileSize:      info.Size(),
			FileSizeHuman: FormatFileSize(info.Size()),
			Status:        "pending",
			Channels:      channels,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Filename < items[j].Filename
	})

	return items, nil
}

func AssignScheduleSlots(items []VideoItem, hours []string, maxDays int, startDate time.Time) []VideoItem {
	history := LoadHistory()
	takenSlots := make(map[string]bool)
	for _, h := range history {
		if h.ScheduledDate != "" && h.ScheduledTime != "" {
			takenSlots[fmt.Sprintf("%s_%s", h.ScheduledDate, h.ScheduledTime)] = true
		}
	}

	return domain.AssignScheduleSlots(items, hours, maxDays, startDate, takenSlots)
}
