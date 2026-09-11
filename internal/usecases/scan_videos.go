package usecases

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"uptik/internal/domain"
	"uptik/internal/ports"
)

type ScanVideosUseCase struct {
	historyRepo ports.HistoryRepository
}

func NewScanVideosUseCase(historyRepo ports.HistoryRepository) *ScanVideosUseCase {
	return &ScanVideosUseCase{
		historyRepo: historyRepo,
	}
}

// Execute scans pending videos in the target folder, deduplicating with history
func (uc *ScanVideosUseCase) Execute(folder string, tag string) ([]domain.VideoItem, error) {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return nil, fmt.Errorf("thư mục không tồn tại: %s", folder)
	}

	uploadedDir := filepath.Join(folder, "uploaded")
	_ = os.MkdirAll(uploadedDir, 0755)

	uploadedFiles := make(map[string]bool)
	if entries, err := os.ReadDir(uploadedDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				uploadedFiles[e.Name()] = true
			}
		}
	}

	historyFiles := make(map[string]bool)
	if uc.historyRepo != nil {
		if history, err := uc.historyRepo.GetAll(); err == nil {
			for _, h := range history {
				historyFiles[h.Filename] = true
			}
		}
	}

	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	var items []domain.VideoItem
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".mp4") {
			continue
		}
		if uploadedFiles[name] || historyFiles[name] {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		clean := domain.CleanVideoTitle(name, tag)
		channels := make(map[string]domain.ChannelStatus)
		channels["tiktok"] = domain.ChannelStatus{Status: "pending"}
		channels["youtube"] = domain.ChannelStatus{Status: "pending"}
		channels["facebook"] = domain.ChannelStatus{Status: "pending"}

		items = append(items, domain.VideoItem{
			ID:            domain.HashString(name),
			Filename:      name,
			FullPath:      filepath.Join(folder, name),
			CleanTitle:    clean,
			CustomTitle:   clean,
			FileSize:      info.Size(),
			FileSizeHuman: domain.FormatFileSize(info.Size()),
			Status:        "pending",
			Channels:      channels,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Filename < items[j].Filename
	})

	return items, nil
}
