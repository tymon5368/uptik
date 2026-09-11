package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ValidateAndSortHours sanitizes, validates "HH:mm" format, deduplicates, and sorts hours chronologically
func ValidateAndSortHours(hours []string) []string {
	seen := make(map[string]bool)
	var valid []string

	for _, h := range hours {
		trimmed := strings.TrimSpace(h)
		if trimmed == "" {
			continue
		}
		parts := strings.Split(trimmed, ":")
		if len(parts) != 2 {
			continue
		}
		hourVal, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		minVal, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || hourVal < 0 || hourVal > 23 || minVal < 0 || minVal > 59 {
			continue
		}
		formatted := fmt.Sprintf("%02d:%02d", hourVal, minVal)
		if !seen[formatted] {
			seen[formatted] = true
			valid = append(valid, formatted)
		}
	}

	sort.Slice(valid, func(i, j int) bool {
		return valid[i] < valid[j]
	})

	if len(valid) == 0 {
		return []string{"11:30", "18:30", "21:30"}
	}
	return valid
}

// GetHourLabel returns a friendly display label for any time slot (e.g. 08:00 (Sáng), 11:30 (Trưa))
func GetHourLabel(h string) string {
	parts := strings.Split(strings.TrimSpace(h), ":")
	if len(parts) != 2 {
		return h
	}
	hourVal, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return h
	}

	period := "Sáng"
	switch {
	case hourVal < 6:
		period = "Đêm"
	case hourVal < 11:
		period = "Sáng"
	case hourVal < 14:
		period = "Trưa"
	case hourVal < 18:
		period = "Chiều"
	case hourVal < 22:
		period = "Tối"
	default:
		period = "Đêm"
	}

	return fmt.Sprintf("%s (%s)", h, period)
}

// GetNextScheduledSlot finds the upcoming slot from the current moment
func GetNextScheduledSlot(hours []string, now time.Time) (nextDate, nextTime string, remaining time.Duration) {
	sorted := ValidateAndSortHours(hours)
	if len(sorted) == 0 {
		sorted = []string{"11:30", "18:30", "21:30"}
	}

	todayStr := now.Format("2006-01-02")
	for _, h := range sorted {
		slotTime, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", todayStr, h), now.Location())
		if err == nil && slotTime.After(now) {
			return todayStr, h, slotTime.Sub(now)
		}
	}

	// If all slots today have passed, pick the first slot of tomorrow
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowStr := tomorrow.Format("2006-01-02")
	firstHour := sorted[0]
	slotTime, _ := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", tomorrowStr, firstHour), now.Location())
	return tomorrowStr, firstHour, slotTime.Sub(now)
}

// Slot represents an available schedule slot
type Slot struct {
	DateStr string
	TimeStr string
	Label   string
}

// AssignScheduleSlots assigns videos to golden hour slots avoiding collisions with takenSlots
func AssignScheduleSlots(items []VideoItem, hours []string, maxDays int, startDate time.Time, takenSlots map[string]bool) []VideoItem {
	if takenSlots == nil {
		takenSlots = make(map[string]bool)
	}

	if len(hours) == 0 {
		hours = []string{"11:30", "18:30", "21:30"}
	}
	if maxDays <= 0 {
		maxDays = 30
	}

	var availableSlots []Slot
	// Generate slots up to maxDays + 2
	for d := 0; d < maxDays+2; d++ {
		cur := startDate.AddDate(0, 0, d)
		dateStr := cur.Format("2006-01-02")

		for _, h := range hours {
			slotKey := fmt.Sprintf("%s_%s", dateStr, h)
			if takenSlots[slotKey] {
				continue
			}

			// If slot is in the past, skip
			parts := strings.Split(h, ":")
			if len(parts) == 2 {
				slotTime, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, h), time.Local)
				if err == nil && slotTime.Before(time.Now().Add(15*time.Minute)) {
					continue
				}
			}

			availableSlots = append(availableSlots, Slot{
				DateStr: dateStr,
				TimeStr: h,
				Label:   GetHourLabel(h),
			})
		}
	}

	result := make([]VideoItem, len(items))
	for i, it := range items {
		result[i] = it
		if i < len(availableSlots) {
			result[i].ScheduledDate = availableSlots[i].DateStr
			result[i].ScheduledTime = availableSlots[i].TimeStr
			result[i].GoldenHourSlot = availableSlots[i].Label
			result[i].Status = "ready"
			if result[i].Channels != nil {
				newCh := make(map[string]ChannelStatus)
				for k, v := range result[i].Channels {
					v.Status = "ready"
					newCh[k] = v
				}
				result[i].Channels = newCh
			}
		}
	}

	return result
}
