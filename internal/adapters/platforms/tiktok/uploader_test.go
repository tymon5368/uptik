package tiktok

import (
	"context"
	"testing"
	"time"
)

func TestTikTokUploader_Basic(t *testing.T) {
	uploader := NewTikTokUploader()
	if uploader.ID() != "tiktok" {
		t.Errorf("Expected ID 'tiktok', got '%s'", uploader.ID())
	}
	if uploader.DisplayName() != "TikTok Studio" {
		t.Errorf("Expected DisplayName 'TikTok Studio', got '%s'", uploader.DisplayName())
	}
	if uploader.LoginURL() != "https://www.tiktok.com/tiktokstudio/upload" {
		t.Errorf("Unexpected LoginURL: %s", uploader.LoginURL())
	}
}

func TestTikTokUploader_ContextCancellation(t *testing.T) {
	uploader := NewTikTokUploader()
	logFunc := func(level, msg string) {}

	t.Run("waitForVideoUpload respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := uploader.waitForVideoUpload(ctx, nil, logFunc)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	})

	t.Run("waitForCopyrightCheck respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		start := time.Now()
		err := uploader.waitForCopyrightCheck(ctx, nil, logFunc)
		elapsed := time.Since(start)

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
		if elapsed > 5*time.Second {
			t.Errorf("waitForCopyrightCheck did not return promptly upon canceled context, took %v", elapsed)
		}
	})

	t.Run("waitForPostButtonReady respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := uploader.waitForPostButtonReady(ctx, nil, logFunc)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	})
}
