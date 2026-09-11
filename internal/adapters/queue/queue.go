package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
	"uptik/internal/domain"
	"uptik/internal/ports"
)

type PersistentJobQueue struct {
	repo ports.JobRepository
	mu   sync.Mutex
}

// NewPersistentJobQueue creates a persistent task queue backed by SQLite
func NewPersistentJobQueue(repo ports.JobRepository) *PersistentJobQueue {
	return &PersistentJobQueue{
		repo: repo,
	}
}

// Enqueue adds a single video upload task
func (q *PersistentJobQueue) Enqueue(ctx context.Context, item domain.VideoItem, channels []string) (*ports.JobItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	job := ports.JobItem{
		ID:             fmt.Sprintf("job_%s_%d", item.ID, now.UnixNano()),
		Video:          item,
		TargetChannels: channels,
		State:          ports.JobStateQueued,
		RetryCount:     0,
		MaxRetries:     3,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := q.repo.SaveJob(job); err != nil {
		return nil, err
	}
	return &job, nil
}

// EnqueueBatch adds a list of video upload tasks
func (q *PersistentJobQueue) EnqueueBatch(ctx context.Context, items []domain.VideoItem, channels []string) ([]ports.JobItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	var enqueued []ports.JobItem

	for _, item := range items {
		job := ports.JobItem{
			ID:             fmt.Sprintf("job_%s_%d", item.ID, now.UnixNano()),
			Video:          item,
			TargetChannels: channels,
			State:          ports.JobStateQueued,
			RetryCount:     0,
			MaxRetries:     3,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := q.repo.SaveJob(job); err != nil {
			return nil, err
		}
		enqueued = append(enqueued, job)
	}

	return enqueued, nil
}

// LeaseNext retrieves and leases the next queued job atomically
func (q *PersistentJobQueue) LeaseNext(ctx context.Context) (*ports.JobItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	pending, err := q.repo.GetPendingJobs()
	if err != nil {
		return nil, err
	}

	for _, job := range pending {
		if job.State == ports.JobStateQueued {
			job.State = ports.JobStateLeased
			job.UpdatedAt = time.Now()
			if err := q.repo.UpdateJob(job); err != nil {
				return nil, err
			}
			return &job, nil
		}
	}

	return nil, nil // No queued jobs available
}

// MarkState updates state and error message for a job
func (q *PersistentJobQueue) MarkState(ctx context.Context, jobID string, state ports.JobState, errMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, err := q.repo.GetJob(jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job không tồn tại: %s", jobID)
	}

	job.State = state
	job.ErrorMsg = errMsg
	job.UpdatedAt = time.Now()
	if state == ports.JobStateCompleted {
		now := time.Now()
		job.CompletedAt = &now
	}

	return q.repo.UpdateJob(*job)
}

// RecoverIncompleteJobs recovers any jobs left in 'leased' or 'uploading' state after an unexpected shutdown
func (q *PersistentJobQueue) RecoverIncompleteJobs(ctx context.Context) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	pending, err := q.repo.GetPendingJobs()
	if err != nil {
		return 0, err
	}

	recovered := 0
	for _, job := range pending {
		if job.State == ports.JobStateLeased || job.State == ports.JobStateUploading {
			job.State = ports.JobStateQueued
			job.RetryCount++
			job.UpdatedAt = time.Now()
			job.ErrorMsg = "Tự động phục hồi sau khi ứng dụng khởi động lại"
			if err := q.repo.UpdateJob(job); err != nil {
				return recovered, err
			}
			recovered++
		}
	}

	return recovered, nil
}

// GetPendingCount returns the count of queued, leased, or uploading jobs
func (q *PersistentJobQueue) GetPendingCount(ctx context.Context) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	pending, err := q.repo.GetPendingJobs()
	if err != nil {
		return 0, err
	}
	return len(pending), nil
}

// ListAll returns all pending jobs
func (q *PersistentJobQueue) ListAll(ctx context.Context) ([]ports.JobItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.repo.GetPendingJobs()
}

// CancelAll marks all pending jobs as cancelled
func (q *PersistentJobQueue) CancelAll(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	pending, err := q.repo.GetPendingJobs()
	if err != nil {
		return err
	}

	for _, job := range pending {
		job.State = ports.JobStateCancelled
		job.UpdatedAt = time.Now()
		_ = q.repo.UpdateJob(job)
	}
	return nil
}
