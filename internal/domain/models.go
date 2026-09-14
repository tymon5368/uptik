package domain

import "errors"

// ErrContentRestricted is returned when video content is flagged as restricted/unoriginal by the platform
var ErrContentRestricted = errors.New("content restricted: unoriginal or low-quality content detected")

// ChannelStatus tracks progress and outcome on a specific platform
type ChannelStatus struct {
	Status     string `json:"status"` // pending, ready, uploading, scheduled, error, skipped
	ErrorMsg   string `json:"errorMsg,omitempty"`
	UploadedAt string `json:"uploadedAt,omitempty"`
}

// PublishMode defines whether to schedule on platform or publish immediately
type PublishMode string

const (
	PublishModeSchedule   PublishMode = "schedule"
	PublishModePublishNow PublishMode = "publish_now"
)

// VideoItem represents a video file to be scheduled/uploaded
type VideoItem struct {
	ID             string                   `json:"id"`
	Filename       string                   `json:"filename"`
	FullPath       string                   `json:"fullPath"`
	CleanTitle     string                   `json:"cleanTitle"`
	CustomTitle    string                   `json:"customTitle"`
	FileSize       int64                    `json:"fileSize"`
	FileSizeHuman  string                   `json:"fileSizeHuman"`
	ScheduledDate  string                   `json:"scheduledDate"`
	ScheduledTime  string                   `json:"scheduledTime"`
	GoldenHourSlot string                   `json:"goldenHourSlot"`
	Status         string                   `json:"status"` // pending, ready, uploading, scheduled, partial, error, skipped
	PublishMode    PublishMode              `json:"publishMode,omitempty"`
	Channels       map[string]ChannelStatus `json:"channels,omitempty"`
	ErrorMsg       string                   `json:"errorMsg,omitempty"`
	UploadedAt     string                   `json:"uploadedAt,omitempty"`
}

// TikTok restricted content handling policy
const (
	TikTokRestrictedPolicySkip       = "skip"
	TikTokRestrictedPolicyPostAnyway = "post_anyway"
)

// Settings stores user configuration
type Settings struct {
	VideoFolder            string      `json:"videoFolder"`
	ChromeUserDataDir      string      `json:"chromeUserDataDir"`
	ChromePath             string      `json:"chromePath"`
	DefaultTag             string      `json:"defaultTag"`
	GoldenHours            []string    `json:"goldenHours"`
	ScheduleGoldenHours    []string    `json:"scheduleGoldenHours"`
	PublishNowGoldenHours  []string    `json:"publishNowGoldenHours"`
	MaxDays                int         `json:"maxDays"`
	Headless               bool        `json:"headless"`
	CdpPort                int         `json:"cdpPort"`
	EnabledChannels        []string    `json:"enabledChannels"`
	AutoStart              bool        `json:"autoStart"`
	CloseToTray            bool        `json:"closeToTray"`
	StartHidden            bool        `json:"startHidden"`
	PublishMode            PublishMode `json:"publishMode"`
	AutoUploadEnabled      bool        `json:"autoUploadEnabled"`
	MissedSlotPolicy       string      `json:"missedSlotPolicy"`       // "skip" or "run_immediate"
	TikTokRestrictedPolicy string      `json:"tiktokRestrictedPolicy"` // "skip" (quarantine to restricted/) or "post_anyway"
	Locale                 string      `json:"locale"`                 // "en", "zh", "de", "ja", "ko", "fr", "es", "it", "nl", "pl", "pt", "ar", "vi"
}

// HistoryRecord represents an archived successful schedule entry
type HistoryRecord struct {
	ID            int64    `json:"id,omitempty"`
	Filename      string   `json:"filename"`
	Title         string   `json:"title"`
	ScheduledDate string   `json:"scheduledDate"`
	ScheduledTime string   `json:"scheduledTime"`
	Channels      []string `json:"channels,omitempty"`
	Timestamp     string   `json:"timestamp"`
}

// LogEntry is emitted to the frontend console
type LogEntry struct {
	Level     string `json:"level"` // info, success, warn, error, cdp
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// UploadProgress reports live multi-channel upload stats
type UploadProgress struct {
	CurrentIndex   int        `json:"currentIndex"`
	TotalVideos    int        `json:"totalVideos"`
	CurrentVideo   *VideoItem `json:"currentVideo,omitempty"`
	CurrentChannel string     `json:"currentChannel,omitempty"`
	SuccessCount   int        `json:"successCount"`
	FailCount      int        `json:"failCount"`
	Status         string     `json:"status"`
}

// PlatformInfo metadata for supported channels
type PlatformInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	LoginURL    string `json:"loginUrl"`
}
