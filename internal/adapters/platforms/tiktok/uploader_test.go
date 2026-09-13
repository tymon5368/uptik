package tiktok

import (
	"testing"
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
