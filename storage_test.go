package main

import (
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
			tag:      "#phimbop",
			expected: "A pack of crazy hippies broke into his home #movie #film #phimbop",
		},
		{
			input:    "Welcome! (1080p_50fps_H264-128kbit_AAC-English).mp4",
			tag:      "#phimbop",
			expected: "Welcome! #phimbop",
		},
		{
			input:    "Already has tag #phimbop (1080p).mp4",
			tag:      "#phimbop",
			expected: "Already has tag #phimbop",
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
