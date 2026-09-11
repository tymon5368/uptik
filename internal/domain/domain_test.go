package domain

import (
	"testing"
	"time"
)

func TestDomainCleanVideoTitle(t *testing.T) {
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

func TestDomainFormatFileSize(t *testing.T) {
	if got := FormatFileSize(500); got != "500 B" {
		t.Errorf("FormatFileSize(500) = %q, want '500 B'", got)
	}
	if got := FormatFileSize(1024 * 1024 * 5); got != "5.0 MB" {
		t.Errorf("FormatFileSize(5MB) = %q, want '5.0 MB'", got)
	}
	if got := FormatFileSize(1024 * 1024 * 1024 * 2); got != "2.0 GB" {
		t.Errorf("FormatFileSize(2GB) = %q, want '2.0 GB'", got)
	}
}

func TestDomainHashString(t *testing.T) {
	h1 := HashString("test1.mp4")
	h2 := HashString("test1.mp4")
	h3 := HashString("test2.mp4")

	if h1 != h2 {
		t.Errorf("HashString should be deterministic: %q != %q", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("HashString collision between distinct inputs: %q == %q", h1, h3)
	}
	if len(h1) != 16 {
		t.Errorf("HashString length should be 16 hex chars, got %d", len(h1))
	}
}

func TestDomainAssignScheduleSlots(t *testing.T) {
	items := []VideoItem{
		{
			Filename:    "video1.mp4",
			CustomTitle: "Video 1",
			Channels: map[string]ChannelStatus{
				"tiktok":  {Status: "pending"},
				"youtube": {Status: "pending"},
			},
		},
		{
			Filename:    "video2.mp4",
			CustomTitle: "Video 2",
		},
	}

	startDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	taken := map[string]bool{
		"2026-10-01_11:30": true, // Taken by prior upload
	}

	assigned := AssignScheduleSlots(items, []string{"11:30", "18:30"}, 30, startDate, taken)

	if len(assigned) != 2 {
		t.Fatalf("expected 2 items, got %d", len(assigned))
	}

	// First available should skip taken 11:30 and pick 18:30
	if assigned[0].ScheduledDate != "2026-10-01" || assigned[0].ScheduledTime != "18:30" {
		t.Errorf("slot 0 wrong, expected 2026-10-01 18:30, got %s %s", assigned[0].ScheduledDate, assigned[0].ScheduledTime)
	}
	if assigned[0].Channels["tiktok"].Status != "ready" {
		t.Errorf("expected channel tiktok ready, got %s", assigned[0].Channels["tiktok"].Status)
	}

	// Second item should go to 2026-10-02 11:30
	if assigned[1].ScheduledDate != "2026-10-02" || assigned[1].ScheduledTime != "11:30" {
		t.Errorf("slot 1 wrong, expected 2026-10-02 11:30, got %s %s", assigned[1].ScheduledDate, assigned[1].ScheduledTime)
	}
}

func TestDomainValidateAndSortHours(t *testing.T) {
	input := []string{"21:30", "08:00", "18:30", "invalid", "11:30", "08:00", " 15:00 "}
	expected := []string{"08:00", "11:30", "15:00", "18:30", "21:30"}

	got := ValidateAndSortHours(input)
	if len(got) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("at index %d: expected %q, got %q", i, expected[i], got[i])
		}
	}

	// Empty input fallback
	emptyGot := ValidateAndSortHours([]string{})
	if len(emptyGot) != 3 || emptyGot[0] != "11:30" {
		t.Errorf("empty input fallback failed, got %v", emptyGot)
	}
}

func TestDomainGetHourLabel(t *testing.T) {
	cases := []struct {
		hour     string
		expected string
	}{
		{"05:30", "05:30 (Đêm)"},
		{"08:00", "08:00 (Sáng)"},
		{"11:30", "11:30 (Trưa)"},
		{"15:00", "15:00 (Chiều)"},
		{"18:30", "18:30 (Tối)"},
		{"23:00", "23:00 (Đêm)"},
		{"invalid", "invalid"},
	}

	for _, c := range cases {
		if got := GetHourLabel(c.hour); got != c.expected {
			t.Errorf("GetHourLabel(%q) = %q, want %q", c.hour, got, c.expected)
		}
	}
}

func TestDomainGetNextScheduledSlot(t *testing.T) {
	hours := []string{"08:00", "12:00", "18:00"}

	// Test 1: Morning, before 12:00 -> should pick 12:00 today
	now := time.Date(2026, 9, 11, 9, 30, 0, 0, time.Local)
	d, h, rem := GetNextScheduledSlot(hours, now)
	if d != "2026-09-11" || h != "12:00" {
		t.Errorf("expected 2026-09-11 12:00, got %s %s", d, h)
	}
	if rem != 2*time.Hour+30*time.Minute {
		t.Errorf("expected 2h30m remaining, got %v", rem)
	}

	// Test 2: Late night, after 18:00 -> should pick 08:00 tomorrow
	late := time.Date(2026, 9, 11, 22, 0, 0, 0, time.Local)
	d2, h2, rem2 := GetNextScheduledSlot(hours, late)
	if d2 != "2026-09-12" || h2 != "08:00" {
		t.Errorf("expected 2026-09-12 08:00, got %s %s", d2, h2)
	}
	if rem2 != 10*time.Hour {
		t.Errorf("expected 10h remaining, got %v", rem2)
	}
}

