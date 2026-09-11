package platforms

import (
	"fmt"
	"sync"
	"uptik/internal/ports"
)

type MemoryRegistry struct {
	mu        sync.RWMutex
	platforms map[string]ports.PlatformUploader
}

// NewRegistry creates a new thread-safe platform registry
func NewRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		platforms: make(map[string]ports.PlatformUploader),
	}
}

func (r *MemoryRegistry) Register(p ports.PlatformUploader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.platforms[p.ID()] = p
}

func (r *MemoryRegistry) Get(id string) (ports.PlatformUploader, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.platforms[id]
	if !ok {
		return nil, fmt.Errorf("nền tảng không được hỗ trợ: %s", id)
	}
	return p, nil
}

func (r *MemoryRegistry) GetAll() []ports.PlatformUploader {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []ports.PlatformUploader
	for _, p := range r.platforms {
		list = append(list, p)
	}
	return list
}

func (r *MemoryRegistry) GetSupportedIDs() []string {
	return []string{"tiktok", "youtube", "facebook"}
}
