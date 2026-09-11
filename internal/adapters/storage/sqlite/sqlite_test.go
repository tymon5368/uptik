package sqlite

import (
	"path/filepath"
	"testing"
	"time"
	"uptik/internal/domain"
	"uptik/internal/ports"
)

func TestSQLiteStorage(t *testing.T) {
	// Use unique temp db for test
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_uptik.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// 1. Settings Test
	s, err := storage.Load()
	if err != nil {
		t.Fatalf("Load settings failed: %v", err)
	}
	if len(s.EnabledChannels) == 0 {
		t.Errorf("Expected default enabled channels, got empty")
	}

	s.DefaultTag = "#customtag"
	s.EnabledChannels = []string{"tiktok", "youtube", "facebook"}
	if err := storage.Save(s); err != nil {
		t.Fatalf("Save settings failed: %v", err)
	}

	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Reload settings failed: %v", err)
	}
	if loaded.DefaultTag != "#customtag" || len(loaded.EnabledChannels) != 3 {
		t.Errorf("Loaded settings mismatch: %+v", loaded)
	}

	// 2. History Test
	hRec := domain.HistoryRecord{
		Filename:      "movie.mp4",
		Title:         "Movie Title",
		ScheduledDate: "2026-10-01",
		ScheduledTime: "11:30",
		Channels:      []string{"tiktok", "youtube"},
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
	if err := storage.Append(hRec); err != nil {
		t.Fatalf("Append history failed: %v", err)
	}

	history, err := storage.GetAll()
	if err != nil {
		t.Fatalf("GetAll history failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("Expected 1 history record, got %d", len(history))
	}
	if history[0].Title != "Movie Title" || len(history[0].Channels) != 2 {
		t.Errorf("History record mismatch: %+v", history[0])
	}

	// 3. Jobs Queue Storage Test
	job := ports.JobItem{
		ID: "job-1",
		Video: domain.VideoItem{
			ID:          "vid-1",
			Filename:    "clip.mp4",
			CustomTitle: "Clip 1",
		},
		TargetChannels: []string{"tiktok", "youtube"},
		State:          ports.JobStateQueued,
		RetryCount:     0,
		MaxRetries:     3,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := storage.SaveJob(job); err != nil {
		t.Fatalf("SaveJob failed: %v", err)
	}

	pending, err := storage.GetPendingJobs()
	if err != nil {
		t.Fatalf("GetPendingJobs failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("Expected 1 pending job, got %d", len(pending))
	}

	// Update job state
	job.State = ports.JobStateCompleted
	now := time.Now()
	job.CompletedAt = &now
	if err := storage.UpdateJob(job); err != nil {
		t.Fatalf("UpdateJob failed: %v", err)
	}

	pendingAfter, err := storage.GetPendingJobs()
	if err != nil {
		t.Fatalf("GetPendingJobs after update failed: %v", err)
	}
	if len(pendingAfter) != 0 {
		t.Errorf("Expected 0 pending jobs after complete, got %d", len(pendingAfter))
	}

	if err := storage.ClearCompleted(); err != nil {
		t.Fatalf("ClearCompleted failed: %v", err)
	}
}
