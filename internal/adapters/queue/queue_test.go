package queue

import (
	"context"
	"path/filepath"
	"testing"
	"uptik/internal/adapters/storage/sqlite"
	"uptik/internal/domain"
	"uptik/internal/ports"
)

func TestPersistentJobQueue(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "queue_test.db")

	storage, err := sqlite.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to open storage: %v", err)
	}
	defer storage.Close()

	q := NewPersistentJobQueue(storage)
	ctx := context.Background()

	// 1. Enqueue Batch
	items := []domain.VideoItem{
		{ID: "v1", Filename: "vid1.mp4", CustomTitle: "Video 1"},
		{ID: "v2", Filename: "vid2.mp4", CustomTitle: "Video 2"},
	}

	enqueued, err := q.EnqueueBatch(ctx, items, []string{"tiktok", "youtube"})
	if err != nil {
		t.Fatalf("EnqueueBatch failed: %v", err)
	}
	if len(enqueued) != 2 {
		t.Fatalf("Expected 2 enqueued jobs, got %d", len(enqueued))
	}

	// 2. Check pending count
	count, err := q.GetPendingCount(ctx)
	if err != nil || count != 2 {
		t.Fatalf("Expected pending count 2, got %d, err: %v", count, err)
	}

	// 3. Lease Next Job
	leased1, err := q.LeaseNext(ctx)
	if err != nil || leased1 == nil {
		t.Fatalf("Expected leased job 1, got nil, err: %v", err)
	}
	if leased1.State != ports.JobStateLeased || leased1.Video.ID != "v1" {
		t.Errorf("Leased job 1 unexpected: %+v", leased1)
	}

	// 4. Mark State to Uploading
	if err := q.MarkState(ctx, leased1.ID, ports.JobStateUploading, ""); err != nil {
		t.Fatalf("MarkState uploading failed: %v", err)
	}

	// 5. Simulate Crash & Recovery
	// While leased1 is uploading, simulate app crash and reboot
	recovered, err := q.RecoverIncompleteJobs(ctx)
	if err != nil {
		t.Fatalf("RecoverIncompleteJobs failed: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("Expected 1 recovered job, got %d", recovered)
	}

	// Re-lease job 1 after recovery
	reLeased, err := q.LeaseNext(ctx)
	if err != nil || reLeased == nil {
		t.Fatalf("Expected re-leased job after recovery, got nil")
	}
	if reLeased.ID != leased1.ID || reLeased.RetryCount != 1 {
		t.Errorf("Recovered job should have retryCount = 1, got %+v", reLeased)
	}

	// Complete job 1
	if err := q.MarkState(ctx, reLeased.ID, ports.JobStateCompleted, ""); err != nil {
		t.Fatalf("MarkState completed failed: %v", err)
	}

	// 6. Lease Job 2
	leased2, err := q.LeaseNext(ctx)
	if err != nil || leased2 == nil || leased2.Video.ID != "v2" {
		t.Fatalf("Expected leased job 2, got %+v, err: %v", leased2, err)
	}

	// Cancel remaining
	if err := q.CancelAll(ctx); err != nil {
		t.Fatalf("CancelAll failed: %v", err)
	}
}
