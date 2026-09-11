package ports

import (
	"context"
	"uptik/internal/domain"

	"github.com/go-rod/rod"
)

// PlatformUploader represents a short-form video platform provider
type PlatformUploader interface {
	ID() string
	DisplayName() string
	LoginURL() string
	CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error)
	UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error
}

// PlatformRegistry manages registered video platform uploaders
type PlatformRegistry interface {
	Register(p PlatformUploader)
	Get(id string) (PlatformUploader, error)
	GetAll() []PlatformUploader
	GetSupportedIDs() []string
}
