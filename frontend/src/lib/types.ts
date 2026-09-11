export interface ChannelStatus {
  status: string;
  errorMsg?: string;
  uploadedAt?: string;
}

export interface VideoItem {
  id: string;
  filename: string;
  fullPath: string;
  cleanTitle: string;
  customTitle: string;
  fileSize: number;
  fileSizeHuman: string;
  scheduledDate: string;
  scheduledTime: string;
  goldenHourSlot: string;
  status: string;
  publishMode?: 'schedule' | 'publish_now';
  channels?: Record<string, ChannelStatus>;
  errorMsg?: string;
  uploadedAt?: string;
}

export interface Settings {
  videoFolder: string;
  chromeUserDataDir: string;
  chromePath: string;
  defaultTag: string;
  goldenHours: string[];
  scheduleGoldenHours: string[];
  publishNowGoldenHours: string[];
  maxDays: number;
  headless: boolean;
  cdpPort: number;
  enabledChannels: string[];
  autoStart: boolean;
  closeToTray: boolean;
  startHidden: boolean;
  publishMode: 'schedule' | 'publish_now';
  autoUploadEnabled: boolean;
  missedSlotPolicy: string;
  locale: string;
}

export interface SchedulerStatus {
  isRunning: boolean;
  autoUploadEnabled: boolean;
  publishMode: 'schedule' | 'publish_now';
  nextDate: string;
  nextTime: string;
  remainingSec: number;
  slotLabel: string;
  goldenHours: string[];
  scheduleGoldenHours?: string[];
  publishNowGoldenHours?: string[];
}

export interface HistoryRecord {
  filename: string;
  title: string;
  scheduledDate: string;
  scheduledTime: string;
  channels?: string[];
  timestamp: string;
}

export interface LogEntry {
  level: 'info' | 'success' | 'warn' | 'error' | 'cdp';
  message: string;
  timestamp: string;
}

export interface UploadProgress {
  currentIndex: number;
  totalVideos: number;
  currentVideo?: VideoItem;
  currentChannel?: string;
  successCount: number;
  failCount: number;
  status: string;
}

export interface PlatformInfo {
  id: string;
  displayName: string;
  loginUrl: string;
}
