package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCleanVideoTitle(t *testing.T) {
	cases := []struct {
		input    string
		tag      string
		expected string
	}{
		{
			input:    "A pack of crazy hippies broke into his home 🫣 #movie #film (1920p_24fps_H264-128kbit_AAC-English).mp4",
			tag:      "#shorts",
			expected: "A pack of crazy hippies broke into his home #movie #film #shorts",
		},
		{
			input:    "Welcome! (1080p_50fps_H264-128kbit_AAC-English).mp4",
			tag:      "#shorts",
			expected: "Welcome! #shorts",
		},
		{
			input:    "Already has tag #shorts (1080p).mp4",
			tag:      "#shorts",
			expected: "Already has tag #shorts",
		},
	}

	for _, c := range cases {
		got := CleanVideoTitle(c.input, c.tag)
		if got != c.expected {
			t.Errorf("CleanVideoTitle(%q, %q) = %q; want %q", c.input, c.tag, got, c.expected)
		}
	}
}

func TestAssignScheduleSlots(t *testing.T) {
	items := []VideoItem{
		{Filename: "video1.mp4", CustomTitle: "Video 1"},
		{Filename: "video2.mp4", CustomTitle: "Video 2"},
		{Filename: "video3.mp4", CustomTitle: "Video 3"},
		{Filename: "video4.mp4", CustomTitle: "Video 4"},
	}

	startDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	hours := []string{"11:30", "18:30", "21:30"}

	assigned := AssignScheduleSlots(items, hours, 30, startDate)

	if len(assigned) != 4 {
		t.Fatalf("expected 4 assigned items, got %d", len(assigned))
	}

	if assigned[0].ScheduledDate != "2026-10-01" || assigned[0].ScheduledTime != "11:30" {
		t.Errorf("slot 0 wrong: %s %s", assigned[0].ScheduledDate, assigned[0].ScheduledTime)
	}
	if assigned[1].ScheduledDate != "2026-10-01" || assigned[1].ScheduledTime != "18:30" {
		t.Errorf("slot 1 wrong: %s %s", assigned[1].ScheduledDate, assigned[1].ScheduledTime)
	}
	if assigned[2].ScheduledDate != "2026-10-01" || assigned[2].ScheduledTime != "21:30" {
		t.Errorf("slot 2 wrong: %s %s", assigned[2].ScheduledDate, assigned[2].ScheduledTime)
	}
	if assigned[3].ScheduledDate != "2026-10-02" || assigned[3].ScheduledTime != "11:30" {
		t.Errorf("slot 3 wrong: %s %s", assigned[3].ScheduledDate, assigned[3].ScheduledTime)
	}
}

func TestOmnichannelSlotsAndSettings(t *testing.T) {
	settings := GetDefaultSettings()
	if len(settings.EnabledChannels) == 0 {
		t.Fatal("expected default enabled channels")
	}

	items := []VideoItem{
		{
			Filename: "movie.mp4",
			Channels: map[string]ChannelStatus{
				"tiktok":  {Status: "pending"},
				"youtube": {Status: "pending"},
			},
		},
	}

	startDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	assigned := AssignScheduleSlots(items, []string{"11:30"}, 30, startDate)

	if assigned[0].Status != "ready" {
		t.Errorf("expected ready, got %s", assigned[0].Status)
	}
	if assigned[0].Channels["tiktok"].Status != "ready" {
		t.Errorf("expected tiktok channel ready, got %s", assigned[0].Channels["tiktok"].Status)
	}
	if assigned[0].Channels["youtube"].Status != "ready" {
		t.Errorf("expected youtube channel ready, got %s", assigned[0].Channels["youtube"].Status)
	}
}

func TestSeparateGoldenHoursSettings(t *testing.T) {
	st := GetDefaultSettings()
	if len(st.ScheduleGoldenHours) != 3 {
		t.Errorf("expected 3 schedule golden hours, got %d", len(st.ScheduleGoldenHours))
	}
	if len(st.PublishNowGoldenHours) != 5 {
		t.Errorf("expected 5 publish_now golden hours, got %d", len(st.PublishNowGoldenHours))
	}

	tmpDir := t.TempDir()
	app := NewAppWithDBPath(filepath.Join(tmpDir, "test_uptik.db"))
	app.settings = st

	// Update schedule hours
	_ = app.UpdateModeGoldenHours("schedule", []string{"09:00", "15:00"})
	if len(app.settings.ScheduleGoldenHours) != 2 {
		t.Errorf("expected 2 schedule hours, got %d", len(app.settings.ScheduleGoldenHours))
	}
	// Publish now hours should NOT be affected
	if len(app.settings.PublishNowGoldenHours) != 5 {
		t.Errorf("expected 5 publish_now hours preserved, got %d", len(app.settings.PublishNowGoldenHours))
	}

	// Update publish_now hours
	_ = app.UpdateModeGoldenHours("publish_now", []string{"08:00", "12:00", "16:00", "20:00"})
	if len(app.settings.PublishNowGoldenHours) != 4 {
		t.Errorf("expected 4 publish_now hours, got %d", len(app.settings.PublishNowGoldenHours))
	}
	// Schedule hours should remain untouched
	if len(app.settings.ScheduleGoldenHours) != 2 {
		t.Errorf("expected 2 schedule hours preserved, got %d", len(app.settings.ScheduleGoldenHours))
	}

	// Switch mode to publish_now
	_ = app.SetPublishMode("publish_now")
	if len(app.settings.GoldenHours) != 4 {
		t.Errorf("expected active GoldenHours to match publish_now (4), got %d", len(app.settings.GoldenHours))
	}

	// Switch back to schedule
	_ = app.SetPublishMode("schedule")
	if len(app.settings.GoldenHours) != 2 {
		t.Errorf("expected active GoldenHours to match schedule (2), got %d", len(app.settings.GoldenHours))
	}
}

