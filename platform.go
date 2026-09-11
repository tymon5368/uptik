package main

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
)

// PlatformUploader represents a short-form video platform provider
type PlatformUploader interface {
	ID() string
	DisplayName() string
	LoginURL() string
	CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error)
	UploadVideo(ctx context.Context, b *rod.Browser, item *VideoItem, logFn func(level, msg string)) error
}

var platformRegistry = make(map[string]PlatformUploader)

// RegisterPlatform registers a platform uploader
func RegisterPlatform(p PlatformUploader) {
	platformRegistry[p.ID()] = p
}

// GetPlatform retrieves an uploader by its platform ID
func GetPlatform(id string) (PlatformUploader, error) {
	p, ok := platformRegistry[id]
	if !ok {
		return nil, fmt.Errorf("nền tảng không được hỗ trợ: %s", id)
	}
	return p, nil
}

// GetAllPlatformIDs returns a list of all registered platform IDs
func GetAllPlatformIDs() []string {
	return []string{"tiktok", "youtube", "facebook"}
}
