package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"uptik/internal/domain"
	"uptik/internal/ports"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

// NewStorage opens SQLite with WAL mode and runs schema migrations
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("không thể mở cơ sở dữ liệu SQLite: %w", err)
	}

	// Performance & Concurrency Pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("lỗi thiết lập pragma %s: %w", p, err)
		}
	}

	s := &Storage{db: db}
	if err := s.migrateSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("lỗi khởi tạo bảng SQLite: %w", err)
	}

	// Auto-migrate legacy JSON files if database is fresh
	_ = s.migrateLegacyJSON("settings.json", "tiktok_schedule_history.json")

	return s, nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) migrateSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		title TEXT NOT NULL,
		scheduled_date TEXT NOT NULL,
		scheduled_time TEXT NOT NULL,
		channels TEXT NOT NULL,
		timestamp TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_history_slot ON history(scheduled_date, scheduled_time);

	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		video_json TEXT NOT NULL,
		target_channels TEXT NOT NULL,
		state TEXT NOT NULL,
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		error_msg TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		completed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs(state);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Storage) migrateLegacyJSON(settingsJSONPath, historyJSONPath string) error {
	// 1. Check if settings empty
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count)
	if count == 0 {
		if data, err := os.ReadFile(settingsJSONPath); err == nil {
			var legacySettings domain.Settings
			if json.Unmarshal(data, &legacySettings) == nil {
				_ = s.Save(legacySettings)
			}
		}
	}

	// 2. Check if history empty
	_ = s.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&count)
	if count == 0 {
		if data, err := os.ReadFile(historyJSONPath); err == nil {
			var legacyHistory []domain.HistoryRecord
			if json.Unmarshal(data, &legacyHistory) == nil {
				for _, h := range legacyHistory {
					if len(h.Channels) == 0 {
						h.Channels = []string{"tiktok"}
					}
					_ = s.Append(h)
				}
			}
		}
	}

	return nil
}

// ----------------------------------------------------
// SettingsRepository Implementation
// ----------------------------------------------------

func (s *Storage) Load() (domain.Settings, error) {
	defaultSettings := domain.Settings{
		VideoFolder:       "/home/arch/Downloads/Movie Nights - Uploads from Movie Nights",
		ChromeUserDataDir: "/home/arch/.config/google-chrome-mcp",
		ChromePath:        "/opt/google/chrome/chrome",
		DefaultTag:        "",
		GoldenHours:           []string{"11:30", "18:30", "21:30"},
		ScheduleGoldenHours:   []string{"11:30", "18:30", "21:30"},
		PublishNowGoldenHours: []string{"07:30", "11:30", "14:30", "18:30", "21:30"},
		MaxDays:               30,
		Headless:              false,
		CdpPort:               9222,
		EnabledChannels:       []string{"tiktok", "youtube"},
		CloseToTray:           true,
		AutoStart:             false,
		StartHidden:           true,
		PublishMode:           domain.PublishModeSchedule,
		AutoUploadEnabled:     false,
		MissedSlotPolicy:      "skip",
		TikTokRestrictedPolicy: domain.TikTokRestrictedPolicySkip,
		Locale:                "en",
	}

	var val string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = 'app_settings'").Scan(&val)
	if err == sql.ErrNoRows {
		_ = s.Save(defaultSettings)
		return defaultSettings, nil
	}
	if err != nil {
		return defaultSettings, err
	}

	var rawFields map[string]json.RawMessage
	_ = json.Unmarshal([]byte(val), &rawFields)
	_, hasSchedule := rawFields["scheduleGoldenHours"]
	_, hasPublishNow := rawFields["publishNowGoldenHours"]

	loaded := defaultSettings
	if err := json.Unmarshal([]byte(val), &loaded); err != nil {
		return defaultSettings, nil
	}
	if len(loaded.EnabledChannels) == 0 {
		loaded.EnabledChannels = []string{"tiktok", "youtube"}
	}
	if loaded.PublishMode == "" {
		loaded.PublishMode = domain.PublishModeSchedule
	}
	if loaded.MissedSlotPolicy == "" {
		loaded.MissedSlotPolicy = "skip"
	}
	if loaded.TikTokRestrictedPolicy != domain.TikTokRestrictedPolicySkip && loaded.TikTokRestrictedPolicy != domain.TikTokRestrictedPolicyPostAnyway {
		loaded.TikTokRestrictedPolicy = domain.TikTokRestrictedPolicySkip
	}
	if loaded.Locale == "" {
		loaded.Locale = "en"
	}

	if !hasSchedule {
		if len(loaded.GoldenHours) > 0 && loaded.PublishMode == domain.PublishModeSchedule {
			loaded.ScheduleGoldenHours = loaded.GoldenHours
		} else {
			loaded.ScheduleGoldenHours = []string{"11:30", "18:30", "21:30"}
		}
	}
	if !hasPublishNow {
		if len(loaded.GoldenHours) > 0 && loaded.PublishMode == domain.PublishModePublishNow {
			loaded.PublishNowGoldenHours = loaded.GoldenHours
		} else {
			loaded.PublishNowGoldenHours = []string{"07:30", "11:30", "14:30", "18:30", "21:30"}
		}
	}
	loaded.ScheduleGoldenHours = domain.ValidateAndSortHours(loaded.ScheduleGoldenHours)
	loaded.PublishNowGoldenHours = domain.ValidateAndSortHours(loaded.PublishNowGoldenHours)

	if loaded.PublishMode == domain.PublishModePublishNow {
		loaded.GoldenHours = loaded.PublishNowGoldenHours
	} else {
		loaded.GoldenHours = loaded.ScheduleGoldenHours
	}
	return loaded, nil
}

func (s *Storage) Save(st domain.Settings) error {
	if st.TikTokRestrictedPolicy != domain.TikTokRestrictedPolicySkip && st.TikTokRestrictedPolicy != domain.TikTokRestrictedPolicyPostAnyway {
		st.TikTokRestrictedPolicy = domain.TikTokRestrictedPolicySkip
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO settings(key, value) VALUES('app_settings', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, string(data))
	return err
}

// ----------------------------------------------------
// HistoryRepository Implementation
// ----------------------------------------------------

func (s *Storage) GetAll() ([]domain.HistoryRecord, error) {
	rows, err := s.db.Query("SELECT id, filename, title, scheduled_date, scheduled_time, channels, timestamp FROM history ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.HistoryRecord
	for rows.Next() {
		var r domain.HistoryRecord
		var channelsStr string
		err := rows.Scan(&r.ID, &r.Filename, &r.Title, &r.ScheduledDate, &r.ScheduledTime, &channelsStr, &r.Timestamp)
		if err != nil {
			return nil, err
		}
		if channelsStr != "" {
			r.Channels = strings.Split(channelsStr, ",")
		} else {
			r.Channels = []string{"tiktok"}
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Storage) Append(rec domain.HistoryRecord) error {
	channelsStr := strings.Join(rec.Channels, ",")
	if channelsStr == "" {
		channelsStr = "tiktok"
	}
	_, err := s.db.Exec(`
		INSERT INTO history(filename, title, scheduled_date, scheduled_time, channels, timestamp)
		VALUES(?, ?, ?, ?, ?, ?)
	`, rec.Filename, rec.Title, rec.ScheduledDate, rec.ScheduledTime, channelsStr, rec.Timestamp)
	return err
}

// ----------------------------------------------------
// JobRepository Implementation
// ----------------------------------------------------

func (s *Storage) SaveJob(job ports.JobItem) error {
	videoBytes, err := json.Marshal(job.Video)
	if err != nil {
		return err
	}
	channelsStr := strings.Join(job.TargetChannels, ",")

	_, err = s.db.Exec(`
		INSERT INTO jobs(id, video_json, target_channels, state, retry_count, max_retries, error_msg, created_at, updated_at, completed_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			video_json = excluded.video_json,
			target_channels = excluded.target_channels,
			state = excluded.state,
			retry_count = excluded.retry_count,
			error_msg = excluded.error_msg,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at
	`, job.ID, string(videoBytes), channelsStr, string(job.State), job.RetryCount, job.MaxRetries, job.ErrorMsg, job.CreatedAt, job.UpdatedAt, job.CompletedAt)
	return err
}

func (s *Storage) GetJob(id string) (*ports.JobItem, error) {
	row := s.db.QueryRow(`
		SELECT id, video_json, target_channels, state, retry_count, max_retries, error_msg, created_at, updated_at, completed_at
		FROM jobs WHERE id = ?
	`, id)

	return scanJobRow(row)
}

func (s *Storage) GetPendingJobs() ([]ports.JobItem, error) {
	rows, err := s.db.Query(`
		SELECT id, video_json, target_channels, state, retry_count, max_retries, error_msg, created_at, updated_at, completed_at
		FROM jobs
		WHERE state IN ('queued', 'leased', 'uploading')
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []ports.JobItem
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

func (s *Storage) UpdateJob(job ports.JobItem) error {
	return s.SaveJob(job)
}

func (s *Storage) DeleteJob(id string) error {
	_, err := s.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}

func (s *Storage) ClearCompleted() error {
	_, err := s.db.Exec("DELETE FROM jobs WHERE state = 'completed'")
	return err
}

func scanJobRow(row *sql.Row) (*ports.JobItem, error) {
	var j ports.JobItem
	var videoStr, channelsStr, stateStr string
	var completedAt sql.NullTime

	err := row.Scan(&j.ID, &videoStr, &channelsStr, &stateStr, &j.RetryCount, &j.MaxRetries, &j.ErrorMsg, &j.CreatedAt, &j.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(videoStr), &j.Video)
	if channelsStr != "" {
		j.TargetChannels = strings.Split(channelsStr, ",")
	}
	j.State = ports.JobState(stateStr)
	if completedAt.Valid {
		j.CompletedAt = &completedAt.Time
	}
	return &j, nil
}

func scanJob(rows *sql.Rows) (*ports.JobItem, error) {
	var j ports.JobItem
	var videoStr, channelsStr, stateStr string
	var completedAt sql.NullTime

	err := rows.Scan(&j.ID, &videoStr, &channelsStr, &stateStr, &j.RetryCount, &j.MaxRetries, &j.ErrorMsg, &j.CreatedAt, &j.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(videoStr), &j.Video)
	if channelsStr != "" {
		j.TargetChannels = strings.Split(channelsStr, ",")
	}
	j.State = ports.JobState(stateStr)
	if completedAt.Valid {
		j.CompletedAt = &completedAt.Time
	}
	return &j, nil
}
