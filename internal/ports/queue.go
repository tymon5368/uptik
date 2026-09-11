package ports

import (
	"context"
	"time"
	"uptik/internal/domain"
)

type JobState string

const (
	JobStateQueued     JobState = "queued"
	JobStateLeased     JobState = "leased"
	JobStateUploading  JobState = "uploading"
	JobStateCompleted  JobState = "completed"
	JobStatePartial    JobState = "partial"
	JobStateFailed     JobState = "failed"
	JobStateCancelled  JobState = "cancelled"
)

// JobItem represents an atomic persistent upload task
type JobItem struct {
	ID             string            `json:"id"`
	Video          domain.VideoItem  `json:"video"`
	TargetChannels []string          `json:"targetChannels"`
	State          JobState          `json:"state"`
	RetryCount     int               `json:"retryCount"`
	MaxRetries     int               `json:"maxRetries"`
	ErrorMsg       string            `json:"errorMsg,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	CompletedAt    *time.Time        `json:"completedAt,omitempty"`
}

// JobQueue manages reliable task distribution and crash recovery
type JobQueue interface {
	Enqueue(ctx context.Context, item domain.VideoItem, channels []string) (*JobItem, error)
	EnqueueBatch(ctx context.Context, items []domain.VideoItem, channels []string) ([]JobItem, error)
	LeaseNext(ctx context.Context) (*JobItem, error)
	MarkState(ctx context.Context, jobID string, state JobState, errMsg string) error
	RecoverIncompleteJobs(ctx context.Context) (int, error)
	GetPendingCount(ctx context.Context) (int, error)
	ListAll(ctx context.Context) ([]JobItem, error)
	CancelAll(ctx context.Context) error
}
