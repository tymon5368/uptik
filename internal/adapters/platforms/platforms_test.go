package platforms

import (
	"context"
	"testing"
	"uptik/internal/domain"

	"github.com/go-rod/rod"
)

type mockUploader struct {
	id   string
	name string
}

func (m *mockUploader) ID() string          { return m.id }
func (m *mockUploader) DisplayName() string { return m.name }
func (m *mockUploader) LoginURL() string    { return "https://example.com" }
func (m *mockUploader) CheckLogin(ctx context.Context, b *rod.Browser, logFn func(level, msg string)) (bool, error) {
	return true, nil
}
func (m *mockUploader) UploadVideo(ctx context.Context, b *rod.Browser, item *domain.VideoItem, logFn func(level, msg string)) error {
	return nil
}

func TestCircuitBreaker(t *testing.T) {
	// Normal content should pass
	if err := CheckCircuitBreaker("<html><body><h1>Upload completed</h1></body></html>"); err != nil {
		t.Errorf("Expected clean HTML to pass circuit breaker, got: %v", err)
	}

	// Flagged keywords should trigger circuit breaker
	badInputs := []string{
		"Warning: Too many requests. Please slow down.",
		"Action Blocked by security filters",
		"Please complete the CAPTCHA to continue",
		"Tài khoản bị hạn chế tính năng đăng bài",
	}

	for _, input := range badInputs {
		if err := CheckCircuitBreaker(input); err == nil {
			t.Errorf("Expected circuit breaker to trip on %q, but it passed", input)
		}
	}
}

func TestMemoryRegistry(t *testing.T) {
	reg := NewRegistry()

	p1 := &mockUploader{id: "tiktok", name: "TikTok Studio"}
	p2 := &mockUploader{id: "youtube", name: "YouTube Shorts"}

	reg.Register(p1)
	reg.Register(p2)

	gotP1, err := reg.Get("tiktok")
	if err != nil || gotP1.DisplayName() != "TikTok Studio" {
		t.Errorf("Failed to retrieve tiktok: %v", err)
	}

	all := reg.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(all))
	}

	_, err = reg.Get("unknown")
	if err == nil {
		t.Errorf("Expected error for unknown platform")
	}
}
