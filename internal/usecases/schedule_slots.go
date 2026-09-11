package usecases

import (
	"fmt"
	"time"
	"uptik/internal/domain"
	"uptik/internal/ports"
)

type ScheduleSlotsUseCase struct {
	historyRepo ports.HistoryRepository
}

func NewScheduleSlotsUseCase(historyRepo ports.HistoryRepository) *ScheduleSlotsUseCase {
	return &ScheduleSlotsUseCase{
		historyRepo: historyRepo,
	}
}

// Execute fetches history, generates non-colliding slots, and assigns them to items
func (uc *ScheduleSlotsUseCase) Execute(items []domain.VideoItem, hours []string, maxDays int, startDate time.Time) []domain.VideoItem {
	takenSlots := make(map[string]bool)

	if uc.historyRepo != nil {
		if history, err := uc.historyRepo.GetAll(); err == nil {
			for _, h := range history {
				if h.ScheduledDate != "" && h.ScheduledTime != "" {
					takenSlots[fmt.Sprintf("%s_%s", h.ScheduledDate, h.ScheduledTime)] = true
				}
			}
		}
	}

	return domain.AssignScheduleSlots(items, hours, maxDays, startDate, takenSlots)
}
