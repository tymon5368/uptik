<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Tabs } from '@ark-ui/svelte/tabs';
  import { Dialog } from '@ark-ui/svelte/dialog';
  import { Portal } from '@ark-ui/svelte/portal';
  import {
    Play,
    Square,
    Folder,
    RefreshCw,
    Calendar,
    Clock,
    CheckCircle2,
    AlertCircle,
    Terminal,
    Settings as SettingsIcon,
    ExternalLink,
    Edit3,
    X,
    Sun,
    Sunset,
    Moon,
    Search,
    ShieldCheck,
    Video,
    Layers,
    Sparkles,
    Trash2,
    Check,
    Music2,
    PlaySquare,
    Share2,
    EyeOff,
    Power,
    Shield,
    Zap,
    Rocket,
    Save,
    Plus,
    Download,
    RotateCcw,
    CheckCircle,
    Languages
  } from 'lucide-svelte';

  import {
    GetSettings,
    SaveSettings,
    ScanFolder,
    GenerateSlots,
    StartUpload,
    StartOmnichannelUpload,
    StopUpload,
    GetHistory,
    SelectFolder,
    OpenInFileManager,
    OpenChromeForLogin,
    OpenPlatformLogin,
    CheckPlatformLogin,
    ResumeQueue,
    CancelQueue,
    GetPendingJobs,
    QuitApp,
    GetSchedulerStatus,
    ToggleAutoUpload,
    SetPublishMode,
    TriggerAutoUploadNow,
    UpdateGoldenHours,
    GetAppVersion,
    CheckForUpdates,
    ApplyUpdate,
    RestartApp
  } from '../wailsjs/go/main/App';
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';
  import type { VideoItem, Settings, HistoryRecord, LogEntry, UploadProgress } from './lib/types';
  import type { updater } from '../wailsjs/go/models';
  import * as m from '$lib/paraglide/messages.js';
  import { i18n, SUPPORTED_LOCALES_LIST, type SupportedLocale } from './lib/i18n.svelte';
  import LanguageSwitcher from './lib/LanguageSwitcher.svelte';
  import logoMark from './assets/images/logo-mark.png';

  const platforms = [
    { id: 'tiktok', name: 'TikTok Studio', icon: Music2, badge: 'bg-rose-950/80 text-rose-300 border-rose-800/80', activeColor: 'bg-[#E50914] text-white border-[#E50914]' },
    { id: 'youtube', name: 'YouTube Shorts', icon: PlaySquare, badge: 'bg-red-950/80 text-red-300 border-red-800/80', activeColor: 'bg-red-600 text-white border-red-500' },
    { id: 'facebook', name: 'Facebook Reels', icon: Share2, badge: 'bg-blue-950/80 text-blue-300 border-blue-800/80', activeColor: 'bg-blue-600 text-white border-blue-500' }
  ];

  // Svelte 5 Runes
  let activeTab = $state<string>('queue');
  let settings = $state<Settings>({
    videoFolder: '/home/arch/Downloads/Movie Nights - Uploads from Movie Nights',
    chromeUserDataDir: '/home/arch/.config/google-chrome-mcp',
    chromePath: '/opt/google/chrome/chrome',
    defaultTag: '#phimbop',
    goldenHours: ['11:30', '18:30', '21:30'],
    scheduleGoldenHours: ['11:30', '18:30', '21:30'],
    publishNowGoldenHours: ['07:30', '11:30', '14:30', '18:30', '21:30'],
    maxDays: 30,
    headless: false,
    cdpPort: 9222,
    enabledChannels: ['tiktok', 'youtube'],
    autoStart: false,
    closeToTray: true,
    startHidden: true,
    publishMode: 'schedule',
    autoUploadEnabled: false,
    missedSlotPolicy: 'skip',
    locale: 'en'
  });

  let schedulerStatus = $state<{
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
  }>({
    isRunning: false,
    autoUploadEnabled: false,
    publishMode: 'schedule',
    nextDate: '',
    nextTime: '',
    remainingSec: 0,
    slotLabel: '',
    goldenHours: ['11:30', '18:30', '21:30'],
    scheduleGoldenHours: ['11:30', '18:30', '21:30'],
    publishNowGoldenHours: ['07:30', '11:30', '14:30', '18:30', '21:30']
  });

  let activeHours = $derived(
    settings.publishMode === 'publish_now'
      ? (settings.publishNowGoldenHours?.length ? settings.publishNowGoldenHours : settings.goldenHours)
      : (settings.scheduleGoldenHours?.length ? settings.scheduleGoldenHours : settings.goldenHours)
  );

  let newSlotTime = $state<string>('09:00');
  let countdownDisplay = $state<string>('');

  let videos = $state<VideoItem[]>([]);
  let history = $state<HistoryRecord[]>([]);
  let logs = $state<LogEntry[]>([]);
  let isUploading = $state<boolean>(false);
  let progress = $state<UploadProgress | null>(null);
  let searchQuery = $state<string>('');
  let statusFilter = $state<string>('all');
  let autoScrollLogs = $state<boolean>(true);

  // Crash Recovery State
  let recoveredCount = $state<number>(0);
  let showRecoveryBanner = $state<boolean>(false);

  // Dialog State
  let editingVideo = $state<VideoItem | null>(null);
  let isDialogOpen = $state<boolean>(false);
  let editTitle = $state<string>('');
  let editDate = $state<string>('');
  let editTime = $state<string>('');

  // Auto-Update State (Level 2 Self-Update)
  let appVersion = $state<string>('1.0.0');
  let updateInfo = $state<updater.UpdateInfo | null>(null);
  let isCheckingUpdate = $state<boolean>(false);
  let isApplyingUpdate = $state<boolean>(false);
  let updateProgress = $state<number>(0);
  let isUpdateComplete = $state<boolean>(false);
  let updateCheckMessage = $state<string>('');

  // Derived state
  let filteredVideos = $derived(
    videos.filter(v => {
      const matchSearch =
        v.customTitle.toLowerCase().includes(searchQuery.toLowerCase()) ||
        v.filename.toLowerCase().includes(searchQuery.toLowerCase());
      const matchStatus = statusFilter === 'all' || v.status === statusFilter;
      return matchSearch && matchStatus;
    })
  );

  let pendingCount = $derived(videos.filter(v => v.status === 'pending').length);
  let readyCount = $derived(videos.filter(v => v.status === 'ready').length);
  let scheduledCount = $derived(videos.filter(v => v.status === 'scheduled').length);
  let errorCount = $derived(videos.filter(v => v.status === 'error').length);

  // 30 Days Matrix Derived
  let calendarMatrix = $derived.by(() => {
    const daysMap = new Map<string, { [time: string]: { video?: VideoItem; history?: HistoryRecord } }>();
    const today = new Date();

    for (let i = 0; i < settings.maxDays; i++) {
      const d = new Date(today.getTime() + i * 24 * 60 * 60 * 1000);
      const dateStr = d.toISOString().split('T')[0];
      daysMap.set(dateStr, {});
    }

    // Fill history first
    for (const h of history) {
      if (daysMap.has(h.scheduledDate)) {
        const day = daysMap.get(h.scheduledDate)!;
        day[h.scheduledTime] = { history: h };
      }
    }

    // Fill queued videos
    for (const v of videos) {
      if (v.scheduledDate && daysMap.has(v.scheduledDate)) {
        const day = daysMap.get(v.scheduledDate)!;
        day[v.scheduledTime] = { video: v };
      }
    }

    return Array.from(daysMap.entries()).map(([dateStr, slots]) => ({
      dateStr,
      slots
    }));
  });

  async function loadInitialData() {
    try {
      const s = await GetSettings();
      if (s && s.videoFolder) {
        settings = {
          ...s,
          goldenHours: s.goldenHours ?? ['11:30', '18:30', '21:30'],
          scheduleGoldenHours: s.scheduleGoldenHours && s.scheduleGoldenHours.length > 0 ? s.scheduleGoldenHours : (s.publishMode === 'schedule' && s.goldenHours?.length ? s.goldenHours : ['11:30', '18:30', '21:30']),
          publishNowGoldenHours: s.publishNowGoldenHours && s.publishNowGoldenHours.length > 0 ? s.publishNowGoldenHours : (s.publishMode === 'publish_now' && s.goldenHours?.length ? s.goldenHours : ['07:30', '11:30', '14:30', '18:30', '21:30']),
          enabledChannels: s.enabledChannels && s.enabledChannels.length > 0 ? s.enabledChannels : ['tiktok', 'youtube'],
          autoStart: s.autoStart ?? false,
          closeToTray: s.closeToTray ?? true,
          startHidden: s.startHidden ?? true,
          publishMode: (s.publishMode === 'publish_now' ? 'publish_now' : 'schedule'),
          autoUploadEnabled: s.autoUploadEnabled ?? false,
          missedSlotPolicy: s.missedSlotPolicy || 'skip',
          locale: s.locale || 'en'
        };
        if (s.locale) {
          i18n.set(s.locale as SupportedLocale);
        }
      }
      await refreshVideos();
      await refreshHistory();
      await refreshSchedulerStatus();
    } catch (err) {
      addLog('error', `Failed to load initial settings: ${err}`);
    }
  }

  async function refreshVideos() {
    try {
      addLog('info', `Scanning video folder: ${settings.videoFolder}`);
      const scanned = await ScanFolder(settings.videoFolder);
      videos = (scanned as unknown as VideoItem[]) || [];
      addLog('success', `Found ${videos.length} unuploaded videos.`);
    } catch (err) {
      addLog('error', `Error scanning video folder: ${err}`);
    }
  }

  async function refreshHistory() {
    try {
      const h = await GetHistory();
      history = (h as unknown as HistoryRecord[]) || [];
    } catch (err) {
      addLog('error', `Error loading upload history: ${err}`);
    }
  }

  async function handleAutoSchedule() {
    if (videos.length === 0) {
      addLog('warn', 'No videos in queue to generate schedule.');
      return;
    }
    try {
      const slotCount = settings.scheduleGoldenHours?.length || 3;
      addLog('info', `Auto-scheduling ${videos.length} videos across ${slotCount} daily golden slots...`);
      const scheduled = await GenerateSlots(videos as any, '');
      videos = (scheduled as unknown as VideoItem[]) || [];
      addLog('success', `Schedule allocated successfully! Ready videos: ${readyCount}`);
    } catch (err) {
      addLog('error', `Error generating schedule slots: ${err}`);
    }
  }

  async function handleSelectFolder() {
    try {
      const folder = await SelectFolder();
      if (folder) {
        settings.videoFolder = folder;
        await SaveSettings(settings);
        await refreshVideos();
      }
    } catch (err) {
      addLog('error', `Error selecting directory: ${err}`);
    }
  }

  async function handleStartUpload() {
    const readyVideos = videos.filter(v => v.status === 'ready' || v.status === 'pending');
    if (readyVideos.length === 0) {
      addLog('warn', 'No ready videos found in queue. Click "Assign Golden Slots" first.');
      return;
    }

    // Auto schedule any remaining pending
    let queue = videos;
    if (readyVideos.length < videos.length || videos.some(v => !v.scheduledDate)) {
      const scheduled = await GenerateSlots(videos as any, '');
      queue = (scheduled as unknown as VideoItem[]) || [];
      videos = queue;
    }

    const channels = settings.enabledChannels && settings.enabledChannels.length > 0
      ? settings.enabledChannels
      : ['tiktok', 'youtube'];

    const channelNames = channels.map(c => platforms.find(p => p.id === c)?.name || c).join(' + ');
    addLog('info', `Starting automated omnichannel upload pipeline (${channelNames}) for ${queue.length} videos...`);
    isUploading = true;
    try {
      await StartOmnichannelUpload(queue as any, channels);
    } catch (err) {
      addLog('error', `Failed to initiate upload pipeline: ${err}`);
      isUploading = false;
    }
  }

  async function handleStopUpload() {
    try {
      await StopUpload();
      addLog('warn', 'Upload process cancellation requested.');
      isUploading = false;
    } catch (err) {
      addLog('error', `Error stopping upload: ${err}`);
    }
  }

  async function checkPendingRecovery() {
    try {
      const pending = await GetPendingJobs();
      if (pending && pending.length > 0) {
        recoveredCount = pending.length;
        showRecoveryBanner = true;
      }
    } catch (err) {
      console.error(err);
    }
  }

  async function handleResumeQueue() {
    addLog('info', 'Resuming pending upload queue from SQLite persistent store...');
    showRecoveryBanner = false;
    isUploading = true;
    try {
      await ResumeQueue();
    } catch (err) {
      addLog('error', `Error resuming queue: ${err}`);
      isUploading = false;
    }
  }

  async function handleCancelQueue() {
    try {
      await CancelQueue();
      showRecoveryBanner = false;
      recoveredCount = 0;
      addLog('warn', 'Cancelled all pending queue jobs.');
    } catch (err) {
      addLog('error', `Error clearing queue: ${err}`);
    }
  }

  function toggleChannel(id: string) {
    if (settings.enabledChannels.includes(id)) {
      if (settings.enabledChannels.length === 1) {
        addLog('warn', 'At least one distribution channel must remain enabled.');
        return;
      }
      settings.enabledChannels = settings.enabledChannels.filter(c => c !== id);
    } else {
      settings.enabledChannels = [...settings.enabledChannels, id];
    }
    handleSaveSettings();
  }

  async function handleOpenPlatform(id: string) {
    const p = platforms.find(pl => pl.id === id);
    addLog('info', `Opening browser session for ${p?.name || id}...`);
    try {
      await OpenPlatformLogin(id);
    } catch (err) {
      addLog('error', `Error opening browser for ${id}: ${err}`);
    }
  }

  async function handleOpenChrome() {
    handleOpenPlatform('tiktok');
  }

  async function handleSaveSettings() {
    try {
      await SaveSettings(settings);
      addLog('success', 'Settings saved successfully!');
    } catch (err) {
      addLog('error', `Error saving settings: ${err}`);
    }
  }

  async function handleLocaleChange(loc: SupportedLocale) {
    settings.locale = loc;
    i18n.set(loc);
    await handleSaveSettings();
  }

  async function handleQuitApp() {
    if (confirm(m.log_quit_confirm())) {
      try {
        await QuitApp();
      } catch (err) {
        addLog('error', `Failed to quit application: ${err}`);
      }
    }
  }

  function addLog(level: 'info' | 'success' | 'warn' | 'error' | 'cdp', message: string) {
    const entry: LogEntry = {
      level,
      message,
      timestamp: new Date().toLocaleTimeString('en-GB', { hour12: false })
    };
    logs = [...logs, entry];
    if (logs.length > 500) {
      logs = logs.slice(-400);
    }
  }

  function openEditDialog(video: VideoItem) {
    editingVideo = video;
    editTitle = video.customTitle;
    editDate = video.scheduledDate || new Date().toISOString().split('T')[0];
    editTime = video.scheduledTime || '11:30';
    isDialogOpen = true;
  }

  function saveEditedVideo() {
    if (!editingVideo) return;
    const idx = videos.findIndex(v => v.id === editingVideo!.id);
    if (idx !== -1) {
      videos[idx].customTitle = editTitle;
      videos[idx].scheduledDate = editDate;
      videos[idx].scheduledTime = editTime;
      videos[idx].status = 'ready';
      videos[idx].goldenHourSlot = getSlotLabel(editTime);
      videos = [...videos];
      addLog('info', `Manually updated schedule for: "${editTitle}" -> ${editDate} ${editTime}`);
    }
    isDialogOpen = false;
    editingVideo = null;
  }

  function getSlotLabel(timeStr: string) {
    if (!timeStr) return '';
    const parts = timeStr.trim().split(':');
    if (parts.length !== 2) return timeStr;
    const hour = parseInt(parts[0], 10);
    if (isNaN(hour)) return timeStr;
    let period = m.slot_morning();
    if (hour < 6) period = m.slot_night();
    else if (hour < 11) period = m.slot_morning();
    else if (hour < 14) period = m.slot_noon();
    else if (hour < 18) period = m.slot_afternoon();
    else if (hour < 22) period = m.slot_evening();
    else period = m.slot_night();
    return `${timeStr} (${period})`;
  }

  async function refreshSchedulerStatus() {
    try {
      const status = await GetSchedulerStatus();
      if (status) {
        schedulerStatus = {
          isRunning: status.isRunning ?? false,
          autoUploadEnabled: status.autoUploadEnabled ?? false,
          publishMode: status.publishMode ?? 'schedule',
          nextDate: status.nextDate ?? '',
          nextTime: status.nextTime ?? '',
          remainingSec: status.remainingSec ?? 0,
          slotLabel: status.slotLabel ?? '',
          goldenHours: status.goldenHours ?? settings.goldenHours,
          scheduleGoldenHours: status.scheduleGoldenHours ?? settings.scheduleGoldenHours,
          publishNowGoldenHours: status.publishNowGoldenHours ?? settings.publishNowGoldenHours
        };
        formatCountdown(schedulerStatus.remainingSec);
      }
    } catch (err) {
      console.error('Lỗi lấy trạng thái scheduler:', err);
    }
  }

  function formatCountdown(totalSeconds: number) {
    if (totalSeconds <= 0) {
      countdownDisplay = m.scheduler_time_reached();
      return;
    }
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    if (hours > 0) {
      countdownDisplay = `${hours}h ${minutes}m ${seconds}s`;
    } else if (minutes > 0) {
      countdownDisplay = `${minutes}m ${seconds}s`;
    } else {
      countdownDisplay = `${seconds}s`;
    }
  }

  async function handleToggleAutoUpload(enabled: boolean) {
    try {
      settings.autoUploadEnabled = enabled;
      await ToggleAutoUpload(enabled);
      await refreshSchedulerStatus();
      addLog('info', enabled ? 'Automated background scheduler ENABLED' : 'Automated background scheduler DISABLED');
    } catch (err) {
      addLog('error', `Error toggling automated scheduler: ${err}`);
    }
  }

  async function handleSetPublishMode(mode: 'schedule' | 'publish_now') {
    try {
      await SetPublishMode(mode);
      settings.publishMode = mode;
      if (mode === 'publish_now') {
        settings.goldenHours = settings.publishNowGoldenHours && settings.publishNowGoldenHours.length > 0
          ? [...settings.publishNowGoldenHours]
          : ['11:30', '18:30', '21:30'];
      } else {
        settings.goldenHours = settings.scheduleGoldenHours && settings.scheduleGoldenHours.length > 0
          ? [...settings.scheduleGoldenHours]
          : ['11:30', '18:30', '21:30'];
      }
      addLog('info', `Switched publish mode to: ${mode === 'publish_now' ? 'Publish Now (Automated Background)' : 'Schedule (Platform Native)'}`);
    } catch (err) {
      addLog('error', `Error changing publish mode: ${err}`);
    }
  }

  async function handleTriggerNow() {
    try {
      addLog('info', 'Triggering manual instant upload for next video...');
      await TriggerAutoUploadNow();
    } catch (err) {
      addLog('error', `Error triggering instant upload: ${err}`);
    }
  }

  async function handleAddGoldenHour() {
    if (!newSlotTime || !newSlotTime.includes(':')) {
      addLog('warn', 'Please specify a valid time slot (HH:mm)');
      return;
    }
    const cleanTime = newSlotTime.trim();
    if (activeHours.includes(cleanTime)) {
      addLog('warn', `Time slot ${cleanTime} already exists in golden hours.`);
      return;
    }
    const updated = [...activeHours, cleanTime];
    await handleUpdateGoldenHours(updated);
    newSlotTime = '';
  }

  async function handleRemoveGoldenHour(time: string) {
    if (activeHours.length <= 1) {
      addLog('warn', 'Must retain at least 1 daily golden slot.');
      return;
    }
    const updated = activeHours.filter(h => h !== time);
    await handleUpdateGoldenHours(updated);
  }

  async function handleUpdateGoldenHours(hours: string[]) {
    try {
      if (settings.publishMode === 'publish_now') {
        settings.publishNowGoldenHours = [...hours];
      } else {
        settings.scheduleGoldenHours = [...hours];
      }
      settings.goldenHours = [...hours];
      await UpdateGoldenHours(hours);
      const updatedSettings = await GetSettings();
      if (updatedSettings) {
        if (updatedSettings.goldenHours) settings.goldenHours = updatedSettings.goldenHours;
        if (updatedSettings.scheduleGoldenHours) settings.scheduleGoldenHours = updatedSettings.scheduleGoldenHours;
        if (updatedSettings.publishNowGoldenHours) settings.publishNowGoldenHours = updatedSettings.publishNowGoldenHours;
      }
      await refreshSchedulerStatus();
      const modeLabel = settings.publishMode === 'publish_now' ? 'Publish Now' : 'Schedule';
      addLog('success', `Updated golden slots (${modeLabel}): ${hours.join(', ')}`);
    } catch (err) {
      addLog('error', `Error updating golden slots: ${err}`);
    }
  }

  async function applyPresetHours(preset: string[]) {
    await handleUpdateGoldenHours(preset);
  }

  // Auto-Update Handlers (Level 2 Self-Update)
  async function handleCheckUpdate(silent = false) {
    if (isCheckingUpdate || isApplyingUpdate) return;
    isCheckingUpdate = true;
    updateCheckMessage = '';
    try {
      const info = await CheckForUpdates();
      updateInfo = info;
      if (info && !info.available && !silent) {
        updateCheckMessage = m.settings_update_none({ version: info.currentVersion });
      }
    } catch (err: any) {
      if (!silent) {
        addLog('error', `Update check failed: ${err?.message || err}`);
      }
    } finally {
      isCheckingUpdate = false;
    }
  }

  async function handleApplyUpdate() {
    if (!updateInfo || !updateInfo.available || isApplyingUpdate) return;
    isApplyingUpdate = true;
    updateProgress = 0;
    try {
      addLog('info', `Downloading and installing UpTik v${updateInfo.latestVersion}...`);
      const ok = await ApplyUpdate(updateInfo);
      if (ok) {
        isUpdateComplete = true;
        addLog('success', `UpTik successfully updated to v${updateInfo.latestVersion}!`);
      }
    } catch (err: any) {
      addLog('error', `Auto-update failed: ${err?.message || err}`);
    } finally {
      isApplyingUpdate = false;
    }
  }

  async function handleRestartApp() {
    try {
      addLog('info', 'Restarting UpTik application...');
      await RestartApp();
    } catch (err: any) {
      addLog('error', `Failed to restart application: ${err?.message || err}`);
    }
  }

  let logContainer = $state<HTMLElement | null>(null);
  $effect(() => {
    if (logs.length && autoScrollLogs && logContainer) {
      logContainer.scrollTop = logContainer.scrollHeight;
    }
  });

  let schedulerTimerInterval: any = null;

  onMount(() => {
    loadInitialData();
    checkPendingRecovery();

    GetAppVersion().then(v => {
      if (v) appVersion = v;
    });

    EventsOn('update:progress', (percent: number) => {
      updateProgress = percent;
    });

    // Check updates quietly in background after 3 seconds
    setTimeout(() => {
      handleCheckUpdate(true);
    }, 3000);

    // Timer interval to refresh countdown and scheduler status
    schedulerTimerInterval = setInterval(() => {
      if (schedulerStatus.remainingSec > 0) {
        schedulerStatus.remainingSec -= 1;
        formatCountdown(schedulerStatus.remainingSec);
      }
      // Periodic sync every 15s
      if (Math.random() < 0.07) {
        refreshSchedulerStatus();
      }
    }, 1000);

    // Wails Events
    EventsOn('log_entry', (entry: LogEntry) => {
      logs = [...logs, entry];
    });

    EventsOn('upload_progress', (p: UploadProgress) => {
      progress = p;
      isUploading = true;
      if (p.currentVideo) {
        const idx = videos.findIndex(v => v.id === p.currentVideo!.id);
        if (idx !== -1) {
          videos[idx].status = 'uploading';
          videos = [...videos];
        }
      }
    });

    EventsOn('video_success', (item: VideoItem) => {
      const idx = videos.findIndex(v => v.id === item.id);
      if (idx !== -1) {
        videos[idx].status = 'scheduled';
        videos[idx].uploadedAt = item.uploadedAt;
        videos = [...videos];
      }
      refreshHistory();
    });

    EventsOn('video_error', (item: VideoItem) => {
      const idx = videos.findIndex(v => v.id === item.id);
      if (idx !== -1) {
        videos[idx].status = 'error';
        videos[idx].errorMsg = item.errorMsg;
        videos = [...videos];
      }
    });

    EventsOn('upload_finished', (data: any) => {
      isUploading = false;
      addLog('success', `Omnichannel upload batch completed! Succeeded: ${data.success}/${data.total}, Failed: ${data.failed}`);
      refreshVideos();
      refreshHistory();
      refreshSchedulerStatus();
    });

    EventsOn('upload_cancelled', () => {
      isUploading = false;
      addLog('warn', 'Upload process safely stopped.');
    });

    EventsOn('queue_recovered', (data: { recoveredCount: number }) => {
      recoveredCount = data.recoveredCount;
      showRecoveryBanner = true;
      addLog('warn', `Detected ${data.recoveredCount} incomplete video uploads from previous session.`);
    });

    EventsOn('scheduler_toggled', (data: { autoUploadEnabled: boolean }) => {
      schedulerStatus.autoUploadEnabled = data.autoUploadEnabled;
      settings.autoUploadEnabled = data.autoUploadEnabled;
      refreshSchedulerStatus();
    });

    EventsOn('scheduler_notification', (data: { title: string; message: string }) => {
      addLog('info', `[Notification] ${data.title}: ${data.message}`);
    });
  });

  onDestroy(() => {
    if (schedulerTimerInterval) {
      clearInterval(schedulerTimerInterval);
    }
    EventsOff('log_entry');
    EventsOff('upload_progress');
    EventsOff('video_success');
    EventsOff('video_error');
    EventsOff('upload_finished');
    EventsOff('upload_cancelled');
    EventsOff('queue_recovered');
    EventsOff('scheduler_toggled');
    EventsOff('scheduler_notification');
    EventsOff('update:progress');
  });
</script>

<div class="min-h-screen bg-[#141414] text-white flex flex-col font-sans select-none" dir={i18n.isRtl ? 'rtl' : 'ltr'}>
  {#key i18n.current}
  <!-- TOP HEADER (Netflix Theme) -->
  <header class="bg-[#181818]/90 backdrop-blur border-b border-[#282828] sticky top-0 z-40 px-6 py-3.5 flex items-center justify-between">
    <!-- Brand Logo -->
    <div class="flex items-center gap-4">
      <div class="flex items-center gap-3">
        <img
          src={logoMark}
          alt="UpTik Logo"
          class="w-8 h-8 sm:w-9 sm:h-9 object-contain drop-shadow-[0_0_12px_rgba(229,9,20,0.5)] hover:scale-105 transition-transform duration-200"
        />
        <div class="flex items-center gap-2">
          <span class="text-2xl font-black tracking-wider text-[#E50914] drop-shadow-[0_2px_8px_rgba(229,9,20,0.4)]">
            {m.app_name()}
          </span>
          <span class="bg-[#E50914]/20 border border-[#E50914]/40 text-[#E50914] text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded">
            {m.app_tagline()}
          </span>
        </div>
      </div>

      <!-- Quick Path Pill -->
      <div class="hidden lg:flex items-center gap-1.5 text-xs text-neutral-400 bg-black/40 border border-neutral-800 rounded-full px-3 py-1 max-w-md truncate">
        <Folder class="w-3.5 h-3.5 text-[#E50914] shrink-0" />
        <span class="truncate">{settings.videoFolder}</span>
      </div>
    </div>

    <!-- Center Upload Status Bar (When active) -->
    {#if isUploading && progress}
      <div class="flex items-center gap-3 bg-neutral-900 border border-[#E50914]/40 rounded-full px-4 py-1.5 animate-pulse shadow-[0_0_15px_rgba(229,9,20,0.25)]">
        <div class="w-2.5 h-2.5 rounded-full bg-[#E50914] animate-ping"></div>
        <span class="text-xs font-semibold text-neutral-200">
          {m.uploading_progress({ current: progress.currentIndex, total: progress.totalVideos })}
        </span>
        <span class="text-xs text-neutral-400 truncate max-w-[200px]">
          {progress.currentVideo?.customTitle || ''}
        </span>
      </div>
    {/if}

    <!-- Channel Toggles in Header -->
    <div class="hidden xl:flex items-center gap-1.5 bg-neutral-900/90 border border-neutral-800 rounded-lg p-1">
      {#each platforms as p}
        {@const isEnabled = settings.enabledChannels.includes(p.id)}
        <button
          onclick={() => toggleChannel(p.id)}
          class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-semibold rounded transition border {isEnabled ? p.activeColor : 'text-neutral-500 border-transparent hover:text-neutral-300'}"
          title={m.platform_toggle_tooltip({ name: p.name })}
        >
          <p.icon class="w-3.5 h-3.5" />
          <span class="text-[11px]">{p.name}</span>
        </button>
      {/each}
    </div>

    <!-- Top Action Buttons -->
    <div class="flex items-center gap-2">
      <!-- Language Selector -->
      <LanguageSwitcher onLocaleChange={handleLocaleChange} />

      {#if updateInfo?.available}
        <button
          onclick={() => activeTab = 'settings'}
          class="flex items-center gap-1.5 px-3 py-1 bg-cyan-500/15 hover:bg-cyan-500/25 border border-cyan-500/40 text-cyan-300 text-xs font-semibold rounded-full transition shadow-[0_0_12px_rgba(6,182,212,0.2)] animate-pulse"
          title={m.header_new_version_tooltip({ version: updateInfo.latestVersion })}
        >
          <Sparkles class="w-3.5 h-3.5 text-cyan-400" />
          <span>{m.header_new_version_badge({ version: updateInfo.latestVersion })}</span>
        </button>
      {/if}

      <!-- Quick Platform Open Dropdown / Buttons -->
      <div class="hidden sm:flex items-center gap-1 bg-neutral-900 border border-neutral-800 rounded-lg p-0.5">
        <button
          onclick={() => handleOpenPlatform('tiktok')}
          title={m.header_open_chrome_platform({ platform: 'TikTok Studio' })}
          class="flex items-center gap-1.5 px-2.5 py-1 text-xs text-neutral-300 hover:text-white hover:bg-neutral-800 rounded transition"
        >
          <Music2 class="w-3.5 h-3.5 text-rose-400" />
          <span>{m.platform_tiktok()}</span>
        </button>
        <button
          onclick={() => handleOpenPlatform('youtube')}
          title={m.header_open_chrome_platform({ platform: 'YouTube Studio' })}
          class="flex items-center gap-1.5 px-2.5 py-1 text-xs text-neutral-300 hover:text-white hover:bg-neutral-800 rounded transition"
        >
          <PlaySquare class="w-3.5 h-3.5 text-red-500" />
          <span>{m.platform_youtube()}</span>
        </button>
        <button
          onclick={() => handleOpenPlatform('facebook')}
          title={m.header_open_chrome_platform({ platform: 'Meta Business Suite' })}
          class="flex items-center gap-1.5 px-2.5 py-1 text-xs text-neutral-300 hover:text-white hover:bg-neutral-800 rounded transition"
        >
          <Share2 class="w-3.5 h-3.5 text-blue-400" />
          <span>{m.platform_facebook()}</span>
        </button>
      </div>

      <button
        onclick={refreshVideos}
        title={m.btn_scan_title()}
        class="p-2 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded border border-neutral-800 transition"
      >
        <RefreshCw class="w-4 h-4" />
      </button>

      <button
        onclick={handleAutoSchedule}
        title={m.btn_auto_schedule_title({ count: settings.scheduleGoldenHours?.length || 3 })}
        class="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold text-white bg-neutral-800 hover:bg-neutral-700 border border-neutral-600 rounded transition"
      >
        <Sparkles class="w-3.5 h-3.5 text-amber-400" />
        <span>{m.btn_auto_schedule_slots({ count: settings.scheduleGoldenHours?.length || 3 })}</span>
      </button>

      {#if !isUploading}
        <button
          onclick={handleStartUpload}
          class="flex items-center gap-2 px-4 py-1.5 text-xs font-bold text-white bg-[#E50914] hover:bg-[#F40612] active:scale-95 rounded shadow-[0_2px_12px_rgba(229,9,20,0.4)] transition"
          title={m.header_start_schedule_tooltip({ channels: settings.enabledChannels.join(', ') })}
        >
          <Play class="w-3.5 h-3.5 fill-current" />
          <span>{m.btn_schedule_multichannel({ count: settings.enabledChannels.length })}</span>
        </button>
      {:else}
        <button
          onclick={handleStopUpload}
          class="flex items-center gap-2 px-4 py-1.5 text-xs font-bold text-white bg-amber-600 hover:bg-amber-500 active:scale-95 rounded transition"
        >
          <Square class="w-3.5 h-3.5 fill-current" />
          <span>{m.btn_stop_upload()}</span>
        </button>
      {/if}
    </div>
  </header>

  <!-- CRASH RECOVERY BANNER -->
  {#if showRecoveryBanner && recoveredCount > 0}
    <div class="bg-amber-950/90 border-b border-amber-700/80 px-6 py-3 flex items-center justify-between text-amber-200">
      <div class="flex items-center gap-3">
        <AlertCircle class="w-5 h-5 text-amber-400 shrink-0" />
        <span class="text-sm font-medium">
          {m.recovery_banner_detected({ count: recoveredCount })}
        </span>
      </div>
      <div class="flex items-center gap-3">
        <button
          onclick={handleCancelQueue}
          class="px-3 py-1.5 text-xs font-semibold bg-neutral-900 hover:bg-neutral-800 text-neutral-300 rounded border border-neutral-700 transition"
        >
          {m.recovery_cancel_queue()}
        </button>
        <button
          onclick={handleResumeQueue}
          class="px-4 py-1.5 text-xs font-bold bg-amber-500 hover:bg-amber-400 text-black rounded transition shadow flex items-center gap-1.5"
        >
          <Play class="w-3.5 h-3.5 fill-current" />
          <span>{m.recovery_resume_queue({ count: recoveredCount })}</span>
        </button>
      </div>
    </div>
  {/if}

  <!-- MAIN APP WITH ARK UI TABS -->
  <main class="flex-1 flex flex-col p-6 max-w-[1600px] w-full mx-auto">
    <Tabs.Root value={activeTab} onValueChange={(e) => activeTab = e.value} class="flex-1 flex flex-col">
      <!-- Tabs Navigation Bar -->
      <div class="flex items-center justify-between border-b border-neutral-800 pb-3 mb-6">
        <Tabs.List class="flex items-center gap-2 bg-neutral-900/80 p-1 rounded-lg border border-neutral-800">
          <Tabs.Trigger
            value="queue"
            class="flex items-center gap-2 px-4 py-2 text-xs font-semibold rounded-md transition duration-150 data-[selected]:bg-[#E50914] data-[selected]:text-white text-neutral-400 hover:text-white"
          >
            <Video class="w-4 h-4" />
            <span>{m.tab_queue()}</span>
            <span class="text-[10px] bg-black/40 px-1.5 py-0.5 rounded-full font-mono">
              {videos.length}
            </span>
          </Tabs.Trigger>

          <Tabs.Trigger
            value="matrix"
            class="flex items-center gap-2 px-4 py-2 text-xs font-semibold rounded-md transition duration-150 data-[selected]:bg-[#E50914] data-[selected]:text-white text-neutral-400 hover:text-white"
          >
            <Calendar class="w-4 h-4" />
            <span>{m.tab_matrix()}</span>
          </Tabs.Trigger>

          <Tabs.Trigger
            value="history"
            class="flex items-center gap-2 px-4 py-2 text-xs font-semibold rounded-md transition duration-150 data-[selected]:bg-[#E50914] data-[selected]:text-white text-neutral-400 hover:text-white"
          >
            <Layers class="w-4 h-4" />
            <span>{m.tab_history()}</span>
            <span class="text-[10px] bg-black/40 px-1.5 py-0.5 rounded-full font-mono">
              {history.length}
            </span>
          </Tabs.Trigger>

          <Tabs.Trigger
            value="logs"
            class="flex items-center gap-2 px-4 py-2 text-xs font-semibold rounded-md transition duration-150 data-[selected]:bg-[#E50914] data-[selected]:text-white text-neutral-400 hover:text-white"
          >
            <Terminal class="w-4 h-4" />
            <span>{m.tab_logs()}</span>
            {#if logs.some(l => l.level === 'error')}
              <span class="w-2 h-2 rounded-full bg-red-500"></span>
            {/if}
          </Tabs.Trigger>

          <Tabs.Trigger
            value="settings"
            class="flex items-center gap-2 px-4 py-2 text-xs font-semibold rounded-md transition duration-150 data-[selected]:bg-[#E50914] data-[selected]:text-white text-neutral-400 hover:text-white relative"
          >
            <SettingsIcon class="w-4 h-4" />
            <span>{m.tab_settings()}</span>
            {#if updateInfo?.available}
              <span class="w-2 h-2 rounded-full bg-cyan-400 animate-ping"></span>
            {/if}
          </Tabs.Trigger>
        </Tabs.List>

        <!-- Stats Chips -->
        <div class="hidden md:flex items-center gap-3 text-xs">
          <div class="flex items-center gap-1.5 px-2.5 py-1 rounded bg-neutral-900 border border-neutral-800">
            <span class="text-neutral-400">{m.stat_ready()}:</span>
            <span class="font-bold text-emerald-400">{readyCount}</span>
          </div>
          <div class="flex items-center gap-1.5 px-2.5 py-1 rounded bg-neutral-900 border border-neutral-800">
            <span class="text-neutral-400">{m.filter_pending()}:</span>
            <span class="font-bold text-amber-400">{pendingCount}</span>
          </div>
          <div class="flex items-center gap-1.5 px-2.5 py-1 rounded bg-neutral-900 border border-neutral-800">
            <span class="text-neutral-400">{m.stat_scheduled()}:</span>
            <span class="font-bold text-blue-400">{scheduledCount}</span>
          </div>
          {#if errorCount > 0}
            <div class="flex items-center gap-1.5 px-2.5 py-1 rounded bg-red-950/60 border border-red-800 text-red-300">
              <span>{m.stat_errors()}:</span>
              <span class="font-bold">{errorCount}</span>
            </div>
          {/if}
        </div>
      </div>

      <!-- TAB 1: VIDEO QUEUE -->
      <Tabs.Content value="queue" class="flex-1 flex flex-col">
        <!-- AUTOMATED PUBLISH NOW SCHEDULER WIDGET -->
        <div class="mb-6 p-4 rounded-xl border bg-neutral-900/90 flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4 transition-all {schedulerStatus.autoUploadEnabled ? 'border-emerald-500/40 shadow-[0_0_20px_rgba(16,185,129,0.12)]' : 'border-neutral-800'}">
          <div class="flex items-center gap-3.5">
            <div class="relative flex items-center justify-center w-11 h-11 rounded-xl {schedulerStatus.autoUploadEnabled ? 'bg-emerald-950/80 border border-emerald-600/70 text-emerald-400' : 'bg-neutral-800/80 border border-neutral-700 text-neutral-400'}">
              <Clock class="w-5 h-5" />
              {#if schedulerStatus.autoUploadEnabled}
                <span class="absolute -top-1 -right-1 w-3.5 h-3.5 bg-emerald-500 rounded-full border-2 border-neutral-900 animate-pulse"></span>
              {/if}
            </div>
            <div>
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-xs font-bold text-white tracking-wide uppercase">{m.settings_auto_upload()}</span>
                <span class="px-2 py-0.5 text-[10px] font-bold rounded-full border {schedulerStatus.autoUploadEnabled ? 'bg-emerald-950 text-emerald-300 border-emerald-700' : 'bg-neutral-800 text-neutral-400 border-neutral-700'}">
                  {schedulerStatus.autoUploadEnabled ? m.scheduler_status_active() : m.scheduler_status_inactive()}
                </span>
                <span class="px-2 py-0.5 text-[10px] font-mono rounded bg-neutral-800 text-neutral-300 border border-neutral-700">
                  {settings.publishMode === 'publish_now' ? m.publish_mode_now() : m.publish_mode_schedule()}
                </span>
              </div>
              <p class="text-xs text-neutral-400 mt-1">
                {#if schedulerStatus.autoUploadEnabled}
                  {m.scheduler_next_label()} <strong class="text-white font-mono text-sm">{schedulerStatus.nextTime}</strong> ({schedulerStatus.slotLabel})
                  {#if countdownDisplay}
                    <span class="ml-1 text-emerald-400 font-mono font-bold">({countdownDisplay})</span>
                  {/if}
                {:else}
                  {m.settings_auto_upload_desc()}: <span class="font-mono text-neutral-300 font-semibold">{(settings.publishNowGoldenHours && settings.publishNowGoldenHours.length > 0 ? settings.publishNowGoldenHours : settings.goldenHours).join(', ')}</span>
                {/if}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2.5 w-full lg:w-auto justify-end flex-wrap">
            <!-- Toggle Auto Upload Switch/Button -->
            <button
              type="button"
              onclick={() => handleToggleAutoUpload(!schedulerStatus.autoUploadEnabled)}
              class="flex items-center gap-2 px-3.5 py-1.5 text-xs font-bold rounded-lg border transition active:scale-95 {schedulerStatus.autoUploadEnabled ? 'bg-emerald-600 hover:bg-emerald-500 text-white border-emerald-500 shadow-[0_2px_10px_rgba(16,185,129,0.3)]' : 'bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border-neutral-700'}"
            >
              <Power class="w-3.5 h-3.5" />
              <span>{schedulerStatus.autoUploadEnabled ? m.scheduler_btn_stop() : m.scheduler_btn_start()}</span>
            </button>
          </div>
        </div>

        <!-- Search & Filters -->
        <div class="flex items-center justify-between gap-4 mb-5">
          <div class="relative flex-1 max-w-md">
            <Search class="w-4 h-4 text-neutral-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              bind:value={searchQuery}
              placeholder={m.queue_search_placeholder()}
              class="w-full bg-[#1e1e1e] border border-neutral-700 rounded-lg pl-9 pr-4 py-2 text-xs text-white placeholder-neutral-500 focus:outline-none focus:border-[#E50914] transition"
            />
          </div>

          <div class="flex items-center gap-2 text-xs">
            <span class="text-neutral-400">{m.col_status()}:</span>
            <select
              bind:value={statusFilter}
              class="bg-[#1e1e1e] border border-neutral-700 rounded-lg px-3 py-1.5 text-xs text-neutral-200 focus:outline-none focus:border-[#E50914]"
            >
              <option value="all">{m.filter_all()} ({videos.length})</option>
              <option value="ready">{m.filter_ready()} ({readyCount})</option>
              <option value="pending">{m.filter_pending()} ({pendingCount})</option>
              <option value="scheduled">{m.filter_scheduled()} ({scheduledCount})</option>
              <option value="error">{m.filter_error()} ({errorCount})</option>
            </select>
          </div>
        </div>

        <!-- Video Cards Grid -->
        {#if filteredVideos.length === 0}
          <div class="flex-1 flex flex-col items-center justify-center border border-dashed border-neutral-800 rounded-xl p-12 text-center my-8">
            <Video class="w-12 h-12 text-neutral-600 mb-3" />
            <h3 class="text-base font-semibold text-neutral-300">{m.no_videos_found()}</h3>
            <p class="text-xs text-neutral-500 mt-1 max-w-sm">
              {m.no_videos_desc()}
            </p>
            <button
              onclick={handleSelectFolder}
              class="mt-4 px-4 py-2 bg-neutral-800 hover:bg-neutral-700 text-xs font-semibold rounded-lg border border-neutral-700 transition"
            >
              {m.settings_btn_select_folder()}
            </button>
          </div>
        {:else}
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {#each filteredVideos as v, idx (v.id)}
              <div
                class="bg-[#1a1a1a] hover:bg-[#202020] border transition duration-200 rounded-xl p-4 flex flex-col justify-between group relative overflow-hidden shadow-lg {v.status === 'uploading' ? 'border-[#E50914] shadow-[0_0_20px_rgba(229,9,20,0.3)] ring-1 ring-[#E50914]' : v.status === 'scheduled' ? 'border-emerald-800/60' : v.status === 'error' ? 'border-red-700/80' : 'border-neutral-800 hover:border-neutral-700'}"
              >
                <!-- Top Badge Row -->
                <div class="flex items-start justify-between gap-2 mb-3">
                  <div class="flex items-center gap-1.5">
                    <span class="text-[11px] font-mono text-neutral-500">#{idx + 1}</span>
                    <span class="text-[10px] bg-neutral-800 text-neutral-300 px-2 py-0.5 rounded font-mono">
                      {v.fileSizeHuman}
                    </span>
                  </div>

                  <!-- Status Badge -->
                  {#if v.status === 'scheduled'}
                    <span class="flex items-center gap-1 text-[11px] text-emerald-400 bg-emerald-950/80 border border-emerald-800/80 px-2 py-0.5 rounded font-medium">
                      <CheckCircle2 class="w-3 h-3" /> {m.status_scheduled()}
                    </span>
                  {:else if v.status === 'uploading'}
                    <span class="flex items-center gap-1 text-[11px] text-[#E50914] bg-[#E50914]/15 border border-[#E50914]/40 px-2 py-0.5 rounded font-bold animate-pulse">
                      {m.status_uploading()}...
                    </span>
                  {:else if v.status === 'error'}
                    <span class="flex items-center gap-1 text-[11px] text-red-400 bg-red-950 border border-red-800 px-2 py-0.5 rounded font-medium">
                      <AlertCircle class="w-3 h-3" /> {m.status_error()}
                    </span>
                  {:else if v.scheduledDate}
                    <span class="text-[11px] text-blue-400 bg-blue-950/60 border border-blue-800/60 px-2 py-0.5 rounded font-medium">
                      {m.status_ready()}
                    </span>
                  {:else}
                    <span class="text-[11px] text-neutral-400 bg-neutral-800/60 border border-neutral-700 px-2 py-0.5 rounded">
                      {m.status_pending()}
                    </span>
                  {/if}
                </div>

                <!-- Title & Filename -->
                <div class="mb-4">
                  <h4 class="text-sm font-semibold text-neutral-100 line-clamp-2 group-hover:text-white transition">
                    {v.customTitle}
                  </h4>
                  <p class="text-[11px] text-neutral-500 font-mono truncate mt-1" title={v.filename}>
                    {v.filename}
                  </p>
                </div>

                <!-- Schedule Slot Badge (Golden Hour) -->
                <div class="pt-3 border-t border-neutral-800/80 flex items-center justify-between">
                  {#if v.scheduledDate && v.scheduledTime}
                    <div class="flex items-center gap-1.5 text-xs font-medium">
                      {#if v.scheduledTime === '11:30'}
                        <Sun class="w-3.5 h-3.5 text-amber-400" />
                        <span class="text-amber-300">{v.scheduledDate} · 11:30</span>
                      {:else if v.scheduledTime === '18:30'}
                        <Sunset class="w-3.5 h-3.5 text-orange-400" />
                        <span class="text-orange-300">{v.scheduledDate} · 18:30</span>
                      {:else if v.scheduledTime === '21:30'}
                        <Moon class="w-3.5 h-3.5 text-indigo-400" />
                        <span class="text-indigo-300">{v.scheduledDate} · 21:30</span>
                      {:else}
                        <Clock class="w-3.5 h-3.5 text-neutral-400" />
                        <span class="text-neutral-300">{v.scheduledDate} · {v.scheduledTime}</span>
                      {/if}
                    </div>
                  {:else}
                    <span class="text-[11px] text-neutral-500 italic">{m.queue_unassigned_slot()}</span>
                  {/if}

                  <!-- Edit Action Button -->
                  <button
                    onclick={() => openEditDialog(v)}
                    title={m.queue_btn_edit_tooltip()}
                    class="p-1.5 text-neutral-400 hover:text-white hover:bg-neutral-800 rounded transition"
                  >
                    <Edit3 class="w-3.5 h-3.5" />
                  </button>
                </div>

                <!-- Omnichannel Target Platform Badges -->
                <div class="mt-2.5 pt-2 border-t border-neutral-800/60 flex items-center gap-1.5 flex-wrap">
                  {#each settings.enabledChannels as ch}
                    {@const pInfo = platforms.find(p => p.id === ch)}
                    {@const chState = v.channels?.[ch]?.status || (v.status === 'scheduled' ? 'scheduled' : v.status === 'uploading' ? 'uploading' : 'ready')}
                    <div
                      class="flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded border transition {chState === 'scheduled' ? 'bg-emerald-950/70 border-emerald-800/70 text-emerald-300' : chState === 'uploading' ? 'bg-[#E50914]/20 border-[#E50914]/50 text-[#E50914] animate-pulse' : chState === 'error' ? 'bg-red-950 border-red-800 text-red-300' : 'bg-neutral-900 border-neutral-800 text-neutral-400'}"
                      title="{pInfo?.name || ch}: {chState}"
                    >
                      {#if pInfo?.icon}
                        {@const Icon = pInfo.icon}
                        <Icon class="w-3 h-3" />
                      {/if}
                      <span class="font-medium text-[10px]">{pInfo?.name ? pInfo.name.split(' ')[0] : ch}</span>
                      {#if chState === 'scheduled'}
                        <Check class="w-3 h-3 text-emerald-400" />
                      {:else if chState === 'uploading'}
                        <RefreshCw class="w-3 h-3 animate-spin text-[#E50914]" />
                      {:else if chState === 'error'}
                        <X class="w-3 h-3 text-red-400" />
                      {/if}
                    </div>
                  {/each}
                </div>

                <!-- Error Message if failed -->
                {#if v.errorMsg}
                  <div class="mt-2 text-[10px] text-red-300 bg-red-950/80 p-2 rounded border border-red-900 break-words">
                    {v.errorMsg}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </Tabs.Content>

      <!-- TAB 2: 30 DAYS MATRIX -->
      <Tabs.Content value="matrix" class="flex-1 flex flex-col">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h2 class="text-base font-bold text-white">{m.matrix_title()}</h2>
            <p class="text-xs text-neutral-400 mt-0.5">
              {m.matrix_subtitle()}
            </p>
          </div>
          <div class="flex items-center gap-4 text-xs">
            <div class="flex items-center gap-1.5">
              <div class="w-3 h-3 rounded bg-amber-500/20 border border-amber-500"></div>
              <span class="text-neutral-300">11:30 {m.slot_noon()}</span>
            </div>
            <div class="flex items-center gap-1.5">
              <div class="w-3 h-3 rounded bg-orange-500/20 border border-orange-500"></div>
              <span class="text-neutral-300">18:30 {m.slot_afternoon()}</span>
            </div>
            <div class="flex items-center gap-1.5">
              <div class="w-3 h-3 rounded bg-indigo-500/20 border border-indigo-500"></div>
              <span class="text-neutral-300">21:30 {m.slot_night()}</span>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-3.5 overflow-y-auto max-h-[calc(100vh-230px)] pr-2">
          {#each calendarMatrix as day}
            <div class="bg-[#1a1a1a] border border-neutral-800 rounded-xl p-3.5 flex flex-col gap-2.5">
              <!-- Day Header -->
              <div class="flex items-center justify-between border-b border-neutral-800 pb-2">
                <span class="text-xs font-bold text-neutral-200">{day.dateStr}</span>
                <span class="text-[10px] text-neutral-500">
                  {m.matrix_slots_count({ count: Object.keys(day.slots).length })}
                </span>
              </div>

              <!-- 3 Golden Slots -->
              <div class="flex flex-col gap-2">
                <!-- Slot 11:30 -->
                <div class="p-2 rounded-lg text-xs {day.slots['11:30']?.history ? 'bg-emerald-950/40 border border-emerald-800/60' : day.slots['11:30']?.video ? 'bg-amber-950/30 border border-amber-800/50' : 'bg-neutral-900 border border-neutral-800 text-neutral-600'}">
                  <div class="flex items-center justify-between text-[11px] mb-1">
                    <span class="font-semibold text-amber-400 flex items-center gap-1">
                      <Sun class="w-3 h-3" /> 11:30
                    </span>
                    {#if day.slots['11:30']?.history}
                      <span class="text-[9px] bg-emerald-900/60 text-emerald-300 px-1.5 py-0.2 rounded">{m.matrix_status_posted()}</span>
                    {:else if day.slots['11:30']?.video}
                      <span class="text-[9px] bg-blue-900/60 text-blue-300 px-1.5 py-0.2 rounded">{m.matrix_status_ready()}</span>
                    {:else}
                      <span class="text-[9px] text-neutral-600">{m.matrix_status_empty()}</span>
                    {/if}
                  </div>
                  <p class="text-[11px] text-neutral-300 truncate">
                    {day.slots['11:30']?.history?.title || day.slots['11:30']?.video?.customTitle || '—'}
                  </p>
                </div>

                <!-- Slot 18:30 -->
                <div class="p-2 rounded-lg text-xs {day.slots['18:30']?.history ? 'bg-emerald-950/40 border border-emerald-800/60' : day.slots['18:30']?.video ? 'bg-orange-950/30 border border-orange-800/50' : 'bg-neutral-900 border border-neutral-800 text-neutral-600'}">
                  <div class="flex items-center justify-between text-[11px] mb-1">
                    <span class="font-semibold text-orange-400 flex items-center gap-1">
                      <Sunset class="w-3 h-3" /> 18:30
                    </span>
                    {#if day.slots['18:30']?.history}
                      <span class="text-[9px] bg-emerald-900/60 text-emerald-300 px-1.5 py-0.2 rounded">{m.matrix_status_posted()}</span>
                    {:else if day.slots['18:30']?.video}
                      <span class="text-[9px] bg-blue-900/60 text-blue-300 px-1.5 py-0.2 rounded">{m.matrix_status_ready()}</span>
                    {:else}
                      <span class="text-[9px] text-neutral-600">{m.matrix_status_empty()}</span>
                    {/if}
                  </div>
                  <p class="text-[11px] text-neutral-300 truncate">
                    {day.slots['18:30']?.history?.title || day.slots['18:30']?.video?.customTitle || '—'}
                  </p>
                </div>

                <!-- Slot 21:30 -->
                <div class="p-2 rounded-lg text-xs {day.slots['21:30']?.history ? 'bg-emerald-950/40 border border-emerald-800/60' : day.slots['21:30']?.video ? 'bg-indigo-950/30 border border-indigo-800/50' : 'bg-neutral-900 border border-neutral-800 text-neutral-600'}">
                  <div class="flex items-center justify-between text-[11px] mb-1">
                    <span class="font-semibold text-indigo-400 flex items-center gap-1">
                      <Moon class="w-3 h-3" /> 21:30
                    </span>
                    {#if day.slots['21:30']?.history}
                      <span class="text-[9px] bg-emerald-900/60 text-emerald-300 px-1.5 py-0.2 rounded">{m.matrix_status_posted()}</span>
                    {:else if day.slots['21:30']?.video}
                      <span class="text-[9px] bg-blue-900/60 text-blue-300 px-1.5 py-0.2 rounded">{m.matrix_status_ready()}</span>
                    {:else}
                      <span class="text-[9px] text-neutral-600">{m.matrix_status_empty()}</span>
                    {/if}
                  </div>
                  <p class="text-[11px] text-neutral-300 truncate">
                    {day.slots['21:30']?.history?.title || day.slots['21:30']?.video?.customTitle || '—'}
                  </p>
                </div>
              </div>
            </div>
          {/each}
        </div>
      </Tabs.Content>

      <!-- TAB 3: HISTORY -->
      <Tabs.Content value="history" class="flex-1 flex flex-col">
        <div class="mb-4 flex items-center justify-between">
          <div>
            <h2 class="text-base font-bold text-white">{m.history_title()}</h2>
            <p class="text-xs text-neutral-400 mt-0.5">
              {m.history_subtitle()}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <button
              onclick={() => OpenInFileManager(`${settings.videoFolder}/uploaded`)}
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-neutral-300 bg-neutral-800 hover:bg-neutral-700 rounded border border-neutral-700 transition"
            >
              <Folder class="w-3.5 h-3.5 text-amber-400" />
              <span>{m.history_btn_open_folder()}</span>
            </button>
            <a
              href="https://www.tiktok.com/tiktokstudio/content"
              target="_blank"
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-[#E50914] hover:bg-[#F40612] rounded transition"
            >
              <ExternalLink class="w-3.5 h-3.5" />
              <span>{m.history_btn_view_tiktok()}</span>
            </a>
          </div>
        </div>

        <div class="bg-[#181818] border border-neutral-800 rounded-xl overflow-hidden flex-1 overflow-y-auto max-h-[calc(100vh-230px)]">
          <table class="w-full text-left text-xs">
            <thead class="bg-neutral-900 border-b border-neutral-800 text-neutral-400 sticky top-0">
              <tr>
                <th class="py-3 px-4 font-semibold">{m.history_th_no()}</th>
                <th class="py-3 px-4 font-semibold">{m.history_th_title()}</th>
                <th class="py-3 px-4 font-semibold">{m.history_th_channels()}</th>
                <th class="py-3 px-4 font-semibold">{m.history_th_date()}</th>
                <th class="py-3 px-4 font-semibold">{m.history_th_time()}</th>
                <th class="py-3 px-4 font-semibold">{m.history_th_filename()}</th>
                <th class="py-3 px-4 font-semibold">{m.history_th_recorded_at()}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-neutral-800">
              {#if history.length === 0}
                <tr>
                  <td colspan="7" class="py-8 text-center text-neutral-500 italic">
                    {m.history_empty()}
                  </td>
                </tr>
              {/if}
              {#each history as h, i}
                <tr class="hover:bg-neutral-800/40 transition">
                  <td class="py-3 px-4 font-mono text-neutral-500">#{i + 1}</td>
                  <td class="py-3 px-4 font-medium text-white max-w-sm truncate">{h.title}</td>
                  <td class="py-3 px-4">
                    <div class="flex items-center gap-1.5 flex-wrap">
                      {#each (h.channels && h.channels.length > 0 ? h.channels : ['tiktok']) as ch}
                        {@const pInfo = platforms.find(p => p.id === ch)}
                        <span class="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded bg-neutral-900 border border-neutral-700 text-neutral-300 font-medium">
                          {#if pInfo?.icon}
                            {@const Icon = pInfo.icon}
                            <Icon class="w-3 h-3" />
                          {:else}
                            <Video class="w-3 h-3" />
                          {/if}
                          <span>{pInfo?.name ? pInfo.name.split(' ')[0] : ch}</span>
                        </span>
                      {/each}
                    </div>
                  </td>
                  <td class="py-3 px-4 font-mono text-neutral-300">{h.scheduledDate}</td>
                  <td class="py-3 px-4">
                    <span class="px-2 py-0.5 rounded text-[11px] font-semibold {h.scheduledTime === '11:30' ? 'bg-amber-950 text-amber-300' : h.scheduledTime === '18:30' ? 'bg-orange-950 text-orange-300' : 'bg-indigo-950 text-indigo-300'}">
                      {h.scheduledTime}
                    </span>
                  </td>
                  <td class="py-3 px-4 font-mono text-neutral-500 max-w-xs truncate">{h.filename}</td>
                  <td class="py-3 px-4 text-neutral-500 text-[11px]">{new Date(h.timestamp).toLocaleString(i18n.current)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </Tabs.Content>

      <!-- TAB 4: CDP LIVE LOGS -->
      <Tabs.Content value="logs" class="flex-1 flex flex-col">
        <div class="mb-3 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
            <h3 class="text-sm font-bold text-white">{m.logs_title()}</h3>
          </div>
          <div class="flex items-center gap-3 text-xs">
            <label class="flex items-center gap-1.5 text-neutral-400 cursor-pointer">
              <input type="checkbox" bind:checked={autoScrollLogs} class="rounded border-neutral-700 bg-neutral-900 text-[#E50914]" />
              <span>{m.logs_autoscroll()}</span>
            </label>
            <button
              onclick={() => logs = []}
              class="flex items-center gap-1 text-neutral-400 hover:text-white px-2 py-1 rounded bg-neutral-800 transition cursor-pointer"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>{m.logs_clear()}</span>
            </button>
          </div>
        </div>

        <div
          bind:this={logContainer}
          class="flex-1 bg-black border border-neutral-800 rounded-xl p-4 font-mono text-xs overflow-y-auto max-h-[calc(100vh-230px)] space-y-1.5"
        >
          {#if logs.length === 0}
            <div class="text-neutral-600 italic">{m.logs_empty()}</div>
          {/if}
          {#each logs as log}
            <div class="flex items-start gap-2.5 leading-relaxed">
              <span class="text-neutral-600 shrink-0">[{log.timestamp}]</span>
              {#if log.level === 'cdp'}
                <span class="text-cyan-400 font-semibold shrink-0">[CDP]</span>
              {:else if log.level === 'success'}
                <span class="text-emerald-400 font-semibold shrink-0">[SUCCESS]</span>
              {:else if log.level === 'warn'}
                <span class="text-amber-400 font-semibold shrink-0">[WARN]</span>
              {:else if log.level === 'error'}
                <span class="text-red-500 font-semibold shrink-0">[ERROR]</span>
              {:else}
                <span class="text-blue-400 font-semibold shrink-0">[INFO]</span>
              {/if}
              <span class="{log.level === 'error' ? 'text-red-300' : log.level === 'success' ? 'text-emerald-200' : log.level === 'warn' ? 'text-amber-200' : 'text-neutral-300'}">
                {log.message}
              </span>
            </div>
          {/each}
        </div>
      </Tabs.Content>

      <!-- TAB 5: SETTINGS -->
      <Tabs.Content value="settings" class="flex-1 w-full space-y-6">
        <!-- SETTINGS PAGE HEADER -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-neutral-800">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-[#E50914]/10 border border-[#E50914]/30 flex items-center justify-center text-[#E50914] shrink-0">
              <SettingsIcon class="w-5 h-5" />
            </div>
            <div>
              <h2 class="text-base font-bold text-white flex items-center gap-2">
                <span>{m.settings_title()}</span>
              </h2>
              <p class="text-xs text-neutral-400 mt-0.5">
                {m.settings_subtitle()}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2.5">
            <button
              type="button"
              onclick={handleQuitApp}
              class="px-3.5 py-2 bg-neutral-900 hover:bg-red-950/60 text-xs font-semibold text-neutral-400 hover:text-red-400 rounded-lg border border-neutral-800 hover:border-red-800/80 transition flex items-center gap-1.5 cursor-pointer"
              title={m.settings_btn_quit()}
            >
              <Power class="w-3.5 h-3.5" />
              <span>{m.settings_btn_quit()}</span>
            </button>
            <button
              type="button"
              onclick={handleSaveSettings}
              class="px-4 py-2 bg-[#E50914] hover:bg-[#F40612] text-xs font-bold text-white rounded-lg shadow-md transition flex items-center gap-2 active:scale-95 cursor-pointer"
            >
              <Save class="w-3.5 h-3.5" />
              <span>{m.settings_btn_save()}</span>
            </button>
          </div>
        </div>

        <!-- SECTION: LANGUAGE SELECTION (FULL WIDTH) -->
        <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-neutral-800/80 pb-3">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center text-[#E50914]">
                <Languages class="w-4 h-4" />
              </div>
              <div>
                <h3 class="text-xs font-bold text-white uppercase tracking-wider">{m.settings_language()}</h3>
                <p class="text-[11px] text-neutral-400 mt-0.5">{m.settings_language_desc()}</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-[11px] text-neutral-400 font-medium">{m.settings_language_current()}:</span>
              <span class="text-xs font-mono font-bold px-2.5 py-1 rounded-md bg-neutral-900 border border-neutral-700 text-neutral-200">
                {i18n.current.toUpperCase()} · {i18n.info.nativeLabel}
              </span>
            </div>
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
            {#each SUPPORTED_LOCALES_LIST as loc}
              {@const isSelected = i18n.current === loc.code}
              <button
                type="button"
                onclick={() => handleLocaleChange(loc.code)}
                class="flex flex-col items-start p-3 rounded-xl border text-left transition-all duration-150 relative cursor-pointer {isSelected
                  ? 'bg-neutral-800/90 border-[#E50914] text-white shadow-[0_0_15px_rgba(229,9,20,0.18)] ring-1 ring-[#E50914]/60'
                  : 'bg-neutral-900/50 border-neutral-800 text-neutral-300 hover:bg-neutral-800/60 hover:border-neutral-700'}"
              >
                <div class="flex items-center justify-between w-full mb-1.5">
                  <span class="text-[10px] font-mono font-bold tracking-wider px-1.5 py-0.5 rounded {isSelected ? 'bg-[#E50914] text-white' : 'bg-neutral-800 text-neutral-400'}">
                    {loc.code.toUpperCase()}
                  </span>
                  {#if isSelected}
                    <Check class="w-3.5 h-3.5 text-[#E50914]" />
                  {/if}
                </div>
                <span class="text-xs font-bold truncate w-full text-white">{loc.nativeLabel}</span>
                <span class="text-[10px] text-neutral-500 truncate w-full mt-0.5">{loc.country}</span>
              </button>
            {/each}
          </div>
        </div>

        <!-- 2-COLUMN BALANCED GRID -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start w-full">
          <!-- COLUMN 1: SYSTEM & HARDWARE/ENVIRONMENT -->
          <div class="space-y-6">
            <!-- CARD 1: SYSTEM PATHS & CHROME -->
            <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
              <div class="border-b border-neutral-800/80 pb-3 flex items-center justify-between">
                <div class="flex items-center gap-2.5">
                  <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center text-blue-400">
                    <Folder class="w-4 h-4" />
                  </div>
                  <div>
                    <h3 class="text-xs font-bold text-white uppercase tracking-wider">{m.settings_section_general()}</h3>
                    <p class="text-[11px] text-neutral-400 mt-0.5">{m.settings_video_folder_desc()}</p>
                  </div>
                </div>
              </div>

              <div class="space-y-4">
                <!-- Video Folder -->
                <div>
                  <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
                    {m.settings_video_folder()} (.mp4)
                  </span>
                  <div class="flex items-center gap-2">
                    <input
                      type="text"
                      bind:value={settings.videoFolder}
                      class="flex-1 bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914] font-mono"
                    />
                    <button
                      type="button"
                      onclick={handleSelectFolder}
                      class="px-3.5 py-2 bg-neutral-800 hover:bg-neutral-700 text-xs font-semibold text-neutral-200 rounded-lg border border-neutral-700 transition cursor-pointer shrink-0"
                    >
                      {m.settings_btn_select_folder()}
                    </button>
                  </div>
                </div>

                <!-- Chrome User Data Dir -->
                <div>
                  <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
                    {m.settings_chrome_profile()}
                  </span>
                  <input
                    type="text"
                    bind:value={settings.chromeUserDataDir}
                    class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914] font-mono"
                  />
                  <p class="text-[11px] text-neutral-500 mt-1">
                    {m.settings_chrome_profile_hint()}
                  </p>
                </div>

                <!-- Chrome Executable -->
                <div>
                  <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
                    {m.settings_chrome_path()}
                  </span>
                  <input
                    type="text"
                    bind:value={settings.chromePath}
                    class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914] font-mono"
                  />
                  <p class="text-[11px] text-neutral-500 mt-1">
                    {m.settings_chrome_path_desc()}
                  </p>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <!-- Default Tags -->
                  <div>
                    <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
                      {m.settings_default_tag()}
                    </span>
                    <input
                      type="text"
                      bind:value={settings.defaultTag}
                      placeholder="#phimbop #movie #shorts"
                      class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
                    />
                  </div>

                  <!-- Max Days Scheduling -->
                  <div>
                    <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
                      {m.settings_max_days()} {m.settings_max_days_unit()}
                    </span>
                    <input
                      type="number"
                      min="1"
                      max="90"
                      bind:value={settings.maxDays}
                      class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914] font-mono"
                    />
                  </div>
                </div>
              </div>
            </div>

            <!-- CARD 2: SYSTEM LIFECYCLE & TRAY -->
            <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
              <div class="flex items-center justify-between border-b border-neutral-800/80 pb-3">
                <div class="flex items-center gap-2.5">
                  <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center text-amber-400">
                    <Zap class="w-4 h-4" />
                  </div>
                  <div>
                    <h3 class="text-xs font-bold text-white uppercase tracking-wider">{m.settings_section_system()}</h3>
                    <p class="text-[11px] text-neutral-400 mt-0.5">{m.settings_section_system_desc()}</p>
                  </div>
                </div>
              </div>

              <div class="space-y-3">
                <!-- Toggle 1: AutoStart -->
                <div class="flex items-start justify-between gap-4 p-3.5 rounded-xl bg-neutral-900/50 border border-neutral-800">
                  <div class="space-y-0.5 flex-1">
                    <span class="text-xs font-semibold text-neutral-200 flex items-center gap-1.5">
                      <Rocket class="w-4 h-4 text-[#E50914]" />
                      <span>{m.settings_auto_start()}</span>
                    </span>
                    <p class="text-[11px] text-neutral-400">
                      {m.settings_auto_start_desc()}
                    </p>
                  </div>
                  <label class="relative inline-flex items-center cursor-pointer shrink-0 mt-0.5">
                    <input
                      type="checkbox"
                      checked={settings.autoStart}
                      onchange={(e) => {
                        settings.autoStart = e.currentTarget.checked;
                      }}
                      class="sr-only peer"
                    />
                    <div class="w-9 h-5 bg-neutral-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-neutral-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[#E50914]"></div>
                  </label>
                </div>

                {#if settings.autoStart}
                  <!-- Sub-toggle: Start Hidden -->
                  <div class="flex items-start justify-between gap-4 p-3.5 ml-4 rounded-xl bg-neutral-900/30 border border-neutral-800/80 transition-all">
                    <div class="space-y-0.5 flex-1">
                      <span class="text-xs font-semibold text-neutral-300 flex items-center gap-1.5">
                        <EyeOff class="w-3.5 h-3.5 text-neutral-400" />
                        <span>{m.settings_start_hidden()}</span>
                      </span>
                      <p class="text-[11px] text-neutral-500">
                        {m.settings_start_hidden_desc()}
                      </p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer shrink-0 mt-0.5">
                      <input
                        type="checkbox"
                        checked={settings.startHidden}
                        onchange={(e) => {
                          settings.startHidden = e.currentTarget.checked;
                        }}
                        class="sr-only peer"
                      />
                      <div class="w-9 h-5 bg-neutral-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-neutral-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[#E50914]"></div>
                    </label>
                  </div>
                {/if}

                <!-- Toggle 2: Close to Tray -->
                <div class="flex items-start justify-between gap-4 p-3.5 rounded-xl bg-neutral-900/50 border border-neutral-800">
                  <div class="space-y-0.5 flex-1">
                    <span class="text-xs font-semibold text-neutral-200 flex items-center gap-1.5">
                      <Shield class="w-4 h-4 text-emerald-400" />
                      <span>{m.settings_close_tray()}</span>
                    </span>
                    <p class="text-[11px] text-neutral-400">
                      {m.settings_close_tray_desc()}
                    </p>
                  </div>
                  <label class="relative inline-flex items-center cursor-pointer shrink-0 mt-0.5">
                    <input
                      type="checkbox"
                      checked={settings.closeToTray}
                      onchange={(e) => {
                        settings.closeToTray = e.currentTarget.checked;
                      }}
                      class="sr-only peer"
                    />
                    <div class="w-9 h-5 bg-neutral-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-neutral-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[#E50914]"></div>
                  </label>
                </div>
              </div>
            </div>

            <!-- CARD 3: SOFTWARE UPDATE -->
            <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
              <div class="flex items-center justify-between border-b border-neutral-800/80 pb-3">
                <div class="flex items-center gap-2.5">
                  <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center text-cyan-400">
                    <Sparkles class="w-4 h-4" />
                  </div>
                  <div>
                    <h3 class="text-xs font-bold text-white uppercase tracking-wider">{m.settings_check_update()}</h3>
                    <p class="text-[11px] text-neutral-400 mt-0.5">
                      {m.settings_current_version()} <span class="font-mono text-white font-semibold">v{appVersion}</span>
                    </p>
                  </div>
                </div>

                <button
                  type="button"
                  onclick={() => handleCheckUpdate(false)}
                  disabled={isCheckingUpdate || isApplyingUpdate}
                  class="px-3.5 py-1.5 bg-neutral-800 hover:bg-neutral-700 disabled:opacity-50 text-xs font-semibold text-neutral-200 rounded-lg border border-neutral-700 transition flex items-center gap-1.5 cursor-pointer"
                >
                  <RefreshCw class="w-3.5 h-3.5 {isCheckingUpdate ? 'animate-spin' : ''}" />
                  <span>{isCheckingUpdate ? m.settings_update_checking() : m.settings_check_update()}</span>
                </button>
              </div>

              {#if updateInfo}
                {#if updateInfo.available}
                  <div class="p-4 bg-cyan-950/30 border border-cyan-800/60 rounded-xl space-y-3">
                    <div class="flex items-start justify-between gap-3">
                      <div class="space-y-1 flex-1">
                        <div class="flex items-center gap-2">
                          <span class="px-2 py-0.5 bg-cyan-500 text-black text-[10px] font-extrabold rounded">{m.settings_update_new_badge()}</span>
                          <span class="text-xs font-bold text-white">UpTik v{updateInfo.latestVersion}</span>
                          {#if updateInfo.publishedAt}
                            <span class="text-[10px] text-neutral-400">({new Date(updateInfo.publishedAt).toLocaleDateString()})</span>
                          {/if}
                        </div>
                        {#if updateInfo.releaseNotes}
                          <p class="text-[11px] text-neutral-300 line-clamp-3 leading-relaxed">
                            {updateInfo.releaseNotes}
                          </p>
                        {/if}
                      </div>

                      {#if !isUpdateComplete}
                        <button
                          type="button"
                          onclick={handleApplyUpdate}
                          disabled={isApplyingUpdate}
                          class="px-3.5 py-1.5 bg-cyan-500 hover:bg-cyan-400 disabled:opacity-50 text-xs font-bold text-black rounded-lg transition shrink-0 flex items-center gap-1.5 shadow-md active:scale-95 cursor-pointer"
                        >
                          <Download class="w-3.5 h-3.5" />
                          <span>{isApplyingUpdate ? m.settings_update_applying() : m.settings_update_btn({ version: updateInfo.latestVersion })}</span>
                        </button>
                      {:else}
                        <button
                          type="button"
                          onclick={handleRestartApp}
                          class="px-3.5 py-1.5 bg-emerald-500 hover:bg-emerald-400 text-xs font-bold text-black rounded-lg transition shrink-0 flex items-center gap-1.5 shadow-md active:scale-95 animate-pulse cursor-pointer"
                        >
                          <RotateCcw class="w-3.5 h-3.5" />
                          <span>{m.settings_btn_restart()}</span>
                        </button>
                      {/if}
                    </div>

                    {#if isApplyingUpdate}
                      <div class="space-y-1.5 pt-1">
                        <div class="flex justify-between text-[11px] text-neutral-300">
                          <span>{m.settings_update_downloading()}</span>
                          <span class="font-mono text-cyan-400">{updateProgress}%</span>
                        </div>
                        <div class="w-full bg-neutral-800 rounded-full h-1.5 overflow-hidden">
                          <div class="bg-cyan-500 h-1.5 transition-all duration-300" style="width: {updateProgress}%"></div>
                        </div>
                      </div>
                    {/if}

                    {#if isUpdateComplete}
                      <div class="text-[11px] text-emerald-400 flex items-center gap-1.5 pt-1">
                        <CheckCircle class="w-3.5 h-3.5 shrink-0" />
                        <span>{m.settings_update_success()}</span>
                      </div>
                    {/if}
                  </div>
                {:else if updateCheckMessage}
                  <div class="text-[11px] text-neutral-400 flex items-center gap-1.5 py-1">
                    <CheckCircle class="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                    <span>{updateCheckMessage}</span>
                  </div>
                {/if}
              {:else if updateCheckMessage}
                <div class="text-[11px] text-neutral-400 flex items-center gap-1.5 py-1">
                  <CheckCircle class="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                  <span>{updateCheckMessage}</span>
                </div>
              {/if}
            </div>
          </div>

          <!-- COLUMN 2: PUBLISHING & OMNICHANNEL AUTOMATION -->
          <div class="space-y-6">
            <!-- CARD 4: PUBLISHING MODE -->
            <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
              <div class="border-b border-neutral-800/80 pb-3 flex items-center justify-between">
                <div class="flex items-center gap-2.5">
                  <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center text-[#E50914]">
                    <Rocket class="w-4 h-4" />
                  </div>
                  <div>
                    <h3 class="text-xs font-bold text-white uppercase tracking-wider">{m.settings_publish_mode()}</h3>
                    <p class="text-[11px] text-neutral-400 mt-0.5">{m.settings_publish_mode_desc()}</p>
                  </div>
                </div>
                <span class="text-[10px] font-mono text-neutral-300 bg-neutral-900 px-2.5 py-1 rounded-md border border-neutral-700">
                  {settings.publishMode === 'publish_now' ? m.publish_mode_now() : m.publish_mode_schedule()}
                </span>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                <!-- Mode 1: Platform Schedule -->
                <label class="flex items-start gap-3.5 p-4 rounded-xl border cursor-pointer transition-all {settings.publishMode === 'schedule' ? 'bg-neutral-800/90 border-[#E50914] text-white shadow-[0_0_15px_rgba(229,9,20,0.15)] ring-1 ring-[#E50914]/50' : 'bg-neutral-900/50 border-neutral-800 text-neutral-400 hover:text-neutral-200 hover:border-neutral-700'}">
                  <input
                    type="radio"
                    name="publishMode"
                    value="schedule"
                    checked={settings.publishMode === 'schedule'}
                    onchange={() => handleSetPublishMode('schedule')}
                    class="mt-1 text-[#E50914] focus:ring-0"
                  />
                  <div class="space-y-1">
                    <div class="flex items-center gap-2">
                      <Calendar class="w-4 h-4 text-amber-400" />
                      <span class="text-xs font-bold text-white">{m.publish_mode_schedule()}</span>
                    </div>
                    <p class="text-[11px] text-neutral-400 leading-relaxed">
                      {m.settings_publish_mode_schedule_desc()}
                    </p>
                  </div>
                </label>

                <!-- Mode 2: Auto Publish Now -->
                <label class="flex items-start gap-3.5 p-4 rounded-xl border cursor-pointer transition-all {settings.publishMode === 'publish_now' ? 'bg-neutral-800/90 border-emerald-500 text-white shadow-[0_0_15px_rgba(16,185,129,0.15)] ring-1 ring-emerald-500/50' : 'bg-neutral-900/50 border-neutral-800 text-neutral-400 hover:text-neutral-200 hover:border-neutral-700'}">
                  <input
                    type="radio"
                    name="publishMode"
                    value="publish_now"
                    checked={settings.publishMode === 'publish_now'}
                    onchange={() => handleSetPublishMode('publish_now')}
                    class="mt-1 text-emerald-500 focus:ring-0"
                  />
                  <div class="space-y-1">
                    <div class="flex items-center gap-2">
                      <Zap class="w-4 h-4 text-emerald-400" />
                      <span class="text-xs font-bold text-white">{m.publish_mode_now()}</span>
                    </div>
                    <p class="text-[11px] text-neutral-400 leading-relaxed">
                      {m.settings_publish_mode_now_desc()}
                    </p>
                  </div>
                </label>
              </div>
            </div>

            <!-- CARD 5: TIME SLOTS MANAGER -->
            <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between border-b border-neutral-800/80 pb-3 gap-2">
                <div class="flex items-center gap-2.5">
                  <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center {settings.publishMode === 'publish_now' ? 'text-emerald-400' : 'text-amber-400'}">
                    <Clock class="w-4 h-4" />
                  </div>
                  <div>
                    <div class="flex items-center gap-2 flex-wrap">
                      <h3 class="text-xs font-bold text-white uppercase tracking-wider">
                        {m.settings_golden_hours_title()} {m.settings_golden_hours_slots_count({ count: activeHours.length })}
                      </h3>
                      <span class="text-[10px] px-2 py-0.5 rounded font-semibold border {settings.publishMode === 'publish_now' ? 'bg-emerald-950/60 border-emerald-600/50 text-emerald-400' : 'bg-amber-950/60 border-amber-600/50 text-amber-400'}">
                        {settings.publishMode === 'publish_now' ? m.settings_mode_badge_publish_now() : m.settings_mode_badge_schedule()}
                      </span>
                    </div>
                    <p class="text-[11px] text-neutral-400 mt-0.5">
                      {settings.publishMode === 'publish_now'
                        ? m.settings_golden_hours_publish_now_info()
                        : m.settings_golden_hours_schedule_info()}
                    </p>
                  </div>
                </div>

                <!-- Quick Presets -->
                <div class="flex items-center gap-1.5 flex-wrap shrink-0">
                  <span class="text-[10px] text-neutral-500 font-semibold uppercase">{m.settings_golden_hours_preset()}:</span>
                  <button
                    type="button"
                    onclick={() => applyPresetHours(['11:30', '18:30', '21:30'])}
                    class="px-2.5 py-1 text-[10px] font-medium bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded-md border border-neutral-700 transition cursor-pointer"
                    title={m.preset_3_hours()}
                  >
                    {m.preset_3_hours()}
                  </button>
                  <button
                    type="button"
                    onclick={() => applyPresetHours(['08:30', '11:30', '17:30', '20:30'])}
                    class="px-2.5 py-1 text-[10px] font-medium bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded-md border border-neutral-700 transition cursor-pointer"
                    title={m.preset_4_hours()}
                  >
                    {m.preset_4_hours()}
                  </button>
                  <button
                    type="button"
                    onclick={() => applyPresetHours(['07:30', '11:30', '14:30', '18:30', '21:30'])}
                    class="px-2.5 py-1 text-[10px] font-medium bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded-md border border-neutral-700 transition cursor-pointer"
                    title={m.preset_5_hours()}
                  >
                    {m.preset_5_hours()}
                  </button>
                </div>
              </div>

              <!-- Current Slots Badges -->
              <div>
                <span class="block text-[11px] font-semibold text-neutral-400 mb-2.5">
                  {m.settings_applied_slots_label()}
                </span>
                <div class="flex flex-wrap gap-2">
                  {#each activeHours as h}
                    {@const label = getSlotLabel(h)}
                    <div class="flex items-center gap-2 px-3 py-1.5 bg-neutral-900 border border-neutral-700 hover:border-neutral-500 rounded-lg text-xs font-semibold text-white transition group">
                      <Clock class="w-3.5 h-3.5 {settings.publishMode === 'publish_now' ? 'text-emerald-400' : 'text-amber-400'}" />
                      <span class="font-mono">{label}</span>
                      <button
                        type="button"
                        onclick={() => handleRemoveGoldenHour(h)}
                        title={m.settings_remove_slot_tooltip({ time: h })}
                        class="p-0.5 text-neutral-400 hover:text-red-400 hover:bg-neutral-800 rounded transition ml-1 cursor-pointer"
                      >
                        <X class="w-3.5 h-3.5" />
                      </button>
                    </div>
                  {/each}
                </div>
              </div>

              <!-- Add New Slot Form -->
              <div class="pt-3 border-t border-neutral-800 flex items-center gap-3 flex-wrap">
                <span class="text-xs font-semibold text-neutral-300">{m.settings_golden_hours_add()}:</span>
                <div class="flex items-center gap-2">
                  <input
                    type="time"
                    bind:value={newSlotTime}
                    class="bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-[#E50914] font-mono"
                  />
                  <button
                    type="button"
                    onclick={handleAddGoldenHour}
                    class="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-bold text-white bg-[#E50914] hover:bg-[#F40612] rounded-lg transition active:scale-95 shadow-sm cursor-pointer"
                  >
                    <Plus class="w-3.5 h-3.5" />
                    <span>{m.settings_golden_hours_add()}</span>
                  </button>
                </div>
              </div>
            </div>

            <!-- CARD 6: OMNICHANNEL TARGETS -->
            <div class="bg-[#181818] border border-neutral-800 rounded-xl p-5 space-y-4 shadow-sm">
              <div class="flex items-center justify-between border-b border-neutral-800/80 pb-3">
                <div class="flex items-center gap-2.5">
                  <div class="w-8 h-8 rounded-lg bg-neutral-800/80 border border-neutral-700/60 flex items-center justify-center text-purple-400">
                    <Share2 class="w-4 h-4" />
                  </div>
                  <div>
                    <h3 class="text-xs font-bold text-white uppercase tracking-wider">{m.settings_section_channels()}</h3>
                    <p class="text-[11px] text-neutral-400 mt-0.5">{m.settings_section_channels_desc()}</p>
                  </div>
                </div>
                <span class="text-[10px] text-neutral-300 bg-neutral-900 border border-neutral-700 px-2.5 py-1 rounded-md font-mono">
                  {m.settings_channels_count({ enabled: settings.enabledChannels.length, total: 3 })}
                </span>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                {#each platforms as p}
                  {@const isChecked = settings.enabledChannels.includes(p.id)}
                  <label class="flex items-center gap-3 p-3.5 rounded-xl border cursor-pointer transition-all {isChecked ? 'bg-neutral-800/90 border-neutral-600 text-white shadow-sm' : 'bg-neutral-900/50 border-neutral-800 text-neutral-400 hover:text-neutral-200 hover:border-neutral-700'}">
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onchange={() => toggleChannel(p.id)}
                      class="rounded border-neutral-700 bg-neutral-900 text-[#E50914] focus:ring-0"
                    />
                    <p.icon class="w-5 h-5 text-neutral-300 shrink-0" />
                    <div class="flex flex-col truncate">
                      <span class="text-xs font-semibold text-white">{p.name}</span>
                      <span class="text-[10px] text-neutral-500 font-mono">{p.id === 'tiktok' ? 'TikTok Studio' : p.id === 'youtube' ? 'YouTube Studio' : 'Business Suite'}</span>
                    </div>
                  </label>
                {/each}
              </div>

              <div class="pt-2.5 border-t border-neutral-800 flex items-center gap-2 flex-wrap text-xs">
                <span class="text-neutral-400 text-[11px]">{m.settings_quick_login()}</span>
                {#each platforms as p}
                  <button
                    type="button"
                    onclick={() => handleOpenPlatform(p.id)}
                    class="flex items-center gap-1.5 px-3 py-1 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs rounded-lg border border-neutral-700 transition cursor-pointer"
                  >
                    <p.icon class="w-3.5 h-3.5" />
                    <span>{p.name}</span>
                    <ExternalLink class="w-3 h-3 text-neutral-500" />
                  </button>
                {/each}
              </div>
            </div>
          </div>
        </div>

        <!-- SECTION 8: BOTTOM ACTIONS -->
        <div class="bg-[#181818] border border-neutral-800 rounded-xl p-4 flex items-center justify-between shadow-sm">
          <button
            type="button"
            onclick={handleQuitApp}
            class="px-4 py-2 bg-neutral-900 hover:bg-red-950/60 text-xs font-semibold text-neutral-400 hover:text-red-400 rounded-lg border border-neutral-800 hover:border-red-800/80 transition flex items-center gap-2 cursor-pointer"
            title={m.settings_btn_quit()}
          >
            <Power class="w-3.5 h-3.5" />
            <span>{m.settings_btn_quit()}</span>
          </button>

          <button
            type="button"
            onclick={handleSaveSettings}
            class="px-6 py-2.5 bg-[#E50914] hover:bg-[#F40612] text-xs font-bold text-white rounded-lg shadow-lg transition flex items-center gap-2 active:scale-95 cursor-pointer"
          >
            <Save class="w-4 h-4" />
            <span>{m.settings_btn_save()}</span>
          </button>
        </div>
      </Tabs.Content>
    </Tabs.Root>
  </main>

  <!-- ARK UI DIALOG (MODAL ĐỔI TIÊU ĐỀ & KHUNG GIỜ) -->
  <Dialog.Root open={isDialogOpen} onOpenChange={(e) => isDialogOpen = e.open}>
    <Portal>
      <Dialog.Backdrop class="fixed inset-0 bg-black/75 backdrop-blur-sm z-50 transition-opacity" />
      <Dialog.Positioner class="fixed inset-0 flex items-center justify-center z-50 p-4">
        <Dialog.Content class="bg-[#1f1f1f] border border-neutral-700 rounded-2xl w-full max-w-lg p-6 shadow-2xl relative text-white space-y-4">
          <Dialog.CloseTrigger
            onclick={() => isDialogOpen = false}
            class="absolute top-4 right-4 text-neutral-400 hover:text-white p-1 rounded-lg hover:bg-neutral-800 transition"
          >
            <X class="w-5 h-5" />
          </Dialog.CloseTrigger>

          <Dialog.Title class="text-base font-bold text-white flex items-center gap-2">
            <Edit3 class="w-4 h-4 text-[#E50914]" />
            <span>{m.dialog_edit_title()}</span>
          </Dialog.Title>

          <Dialog.Description class="text-xs text-neutral-400">
            {m.dialog_edit_subtitle()}
          </Dialog.Description>

          <div class="space-y-3.5 pt-2">
            <div>
              <span class="block text-xs font-medium text-neutral-300 mb-1">{m.dialog_field_title()}</span>
              <textarea
                rows="3"
                bind:value={editTitle}
                class="w-full bg-neutral-900 border border-neutral-700 rounded-lg p-3 text-xs text-white focus:outline-none focus:border-[#E50914]"
              ></textarea>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <span class="block text-xs font-medium text-neutral-300 mb-1">{m.dialog_field_date()}</span>
                <input
                  type="date"
                  bind:value={editDate}
                  class="w-full bg-neutral-900 border border-neutral-700 rounded-lg p-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
                />
              </div>

              <div>
                <span class="block text-xs font-medium text-neutral-300 mb-1">{m.dialog_field_time()}</span>
                <select
                  bind:value={editTime}
                  class="w-full bg-neutral-900 border border-neutral-700 rounded-lg p-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
                >
                  {#each settings.goldenHours as h}
                    <option value={h}>{getSlotLabel(h)}</option>
                  {/each}
                </select>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-end gap-2.5 pt-4 border-t border-neutral-800">
            <button
              onclick={() => isDialogOpen = false}
              class="px-4 py-2 text-xs font-semibold text-neutral-400 hover:text-white rounded-lg hover:bg-neutral-800 transition"
            >
              {m.dialog_btn_cancel()}
            </button>
            <button
              onclick={saveEditedVideo}
              class="px-4 py-2 text-xs font-bold text-white bg-[#E50914] hover:bg-[#F40612] rounded-lg shadow transition"
            >
              {m.dialog_btn_save()}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Positioner>
    </Portal>
  </Dialog.Root>
  {/key}
</div>
