package ports

import (
	"uptik/internal/domain"
)

// SettingsRepository abstracts configuration storage
type SettingsRepository interface {
	Load() (domain.Settings, error)
	Save(s domain.Settings) error
}

// HistoryRepository abstracts upload history persistence
type HistoryRepository interface {
	GetAll() ([]domain.HistoryRecord, error)
	Append(rec domain.HistoryRecord) error
}

// JobRepository abstracts persistent queue storage
type JobRepository interface {
	SaveJob(job JobItem) error
	GetJob(id string) (*JobItem, error)
	GetPendingJobs() ([]JobItem, error)
	UpdateJob(job JobItem) error
	DeleteJob(id string) error
	ClearCompleted() error
}
