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
    CheckCircle
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
  import { i18n, type SupportedLocale } from './lib/i18n.svelte';
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
      addLog('error', `Lỗi tải dữ liệu ban đầu: ${err}`);
    }
  }

  async function refreshVideos() {
    try {
      addLog('info', `Quét video trong: ${settings.videoFolder}`);
      const scanned = await ScanFolder(settings.videoFolder);
      videos = (scanned as unknown as VideoItem[]) || [];
      addLog('success', `Tìm thấy ${videos.length} video chưa upload.`);
    } catch (err) {
      addLog('error', `Lỗi quét video: ${err}`);
    }
  }

  async function refreshHistory() {
    try {
      const h = await GetHistory();
      history = (h as unknown as HistoryRecord[]) || [];
    } catch (err) {
      addLog('error', `Lỗi tải lịch sử: ${err}`);
    }
  }

  async function handleAutoSchedule() {
    if (videos.length === 0) {
      addLog('warn', 'Không có video nào trong danh sách để tạo lịch.');
      return;
    }
    try {
      const slotCount = settings.scheduleGoldenHours?.length || 3;
      addLog('info', `Tạo lịch tự động cho ${videos.length} video theo ${slotCount} khung giờ vàng...`);
      const scheduled = await GenerateSlots(videos as any, '');
      videos = (scheduled as unknown as VideoItem[]) || [];
      addLog('success', `Đã phân bổ lịch thành công! Số video sẵn sàng: ${readyCount}`);
    } catch (err) {
      addLog('error', `Lỗi tạo lịch: ${err}`);
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
      addLog('error', `Lỗi chọn thư mục: ${err}`);
    }
  }

  async function handleStartUpload() {
    const readyVideos = videos.filter(v => v.status === 'ready' || v.status === 'pending');
    if (readyVideos.length === 0) {
      addLog('warn', 'Không có video nào sẵn sàng lịch để upload. Hãy bấm "Tự Động Sinh Khung Giờ" trước.');
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
    addLog('info', `Bắt đầu chuỗi tự động upload đa kênh (${channelNames}) cho ${queue.length} video...`);
    isUploading = true;
    try {
      await StartOmnichannelUpload(queue as any, channels);
    } catch (err) {
      addLog('error', `Lỗi khởi động upload: ${err}`);
      isUploading = false;
    }
  }

  async function handleStopUpload() {
    try {
      await StopUpload();
      addLog('warn', 'Đã yêu cầu dừng tiến trình upload.');
      isUploading = false;
    } catch (err) {
      addLog('error', `Lỗi dừng upload: ${err}`);
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
    addLog('info', 'Tiếp tục thực thi hàng chờ upload đã lưu trong SQLite...');
    showRecoveryBanner = false;
    isUploading = true;
    try {
      await ResumeQueue();
    } catch (err) {
      addLog('error', `Lỗi khi resume: ${err}`);
      isUploading = false;
    }
  }

  async function handleCancelQueue() {
    try {
      await CancelQueue();
      showRecoveryBanner = false;
      recoveredCount = 0;
      addLog('warn', 'Đã hủy bỏ toàn bộ hàng chờ cũ.');
    } catch (err) {
      addLog('error', `Lỗi khi hủy hàng chờ: ${err}`);
    }
  }

  function toggleChannel(id: string) {
    if (settings.enabledChannels.includes(id)) {
      if (settings.enabledChannels.length === 1) {
        addLog('warn', 'Cần duy trì ít nhất 1 kênh phân phối.');
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
    addLog('info', `Mở trình duyệt quản trị cho ${p?.name || id}...`);
    try {
      await OpenPlatformLogin(id);
    } catch (err) {
      addLog('error', `Lỗi mở trình duyệt cho ${id}: ${err}`);
    }
  }

  async function handleOpenChrome() {
    handleOpenPlatform('tiktok');
  }

  async function handleSaveSettings() {
    try {
      await SaveSettings(settings);
      addLog('success', 'Đã lưu cài đặt thành công!');
    } catch (err) {
      addLog('error', `Lỗi lưu cài đặt: ${err}`);
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
        addLog('error', `Lỗi thoát ứng dụng: ${err}`);
      }
    }
  }

  function addLog(level: 'info' | 'success' | 'warn' | 'error' | 'cdp', message: string) {
    const entry: LogEntry = {
      level,
      message,
      timestamp: new Date().toLocaleTimeString('vi-VN', { hour12: false })
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
      addLog('info', `Đã cập nhật thủ công lịch cho: "${editTitle}" -> ${editDate} ${editTime}`);
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
      countdownDisplay = 'Đến khung giờ!';
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
      addLog('info', enabled ? 'Đã BẬT tính năng tự động upload theo giờ (Publish Now ngầm)' : 'Đã TẮT tính năng tự động upload theo giờ');
    } catch (err) {
      addLog('error', `Lỗi thay đổi tự động upload: ${err}`);
    }
  }

  async function handleSetPublishMode(mode: 'schedule' | 'publish_now') {
    try {
      settings.publishMode = mode;
      if (mode === 'publish_now' && settings.publishNowGoldenHours && settings.publishNowGoldenHours.length > 0) {
        settings.goldenHours = [...settings.publishNowGoldenHours];
      } else if (mode === 'schedule' && settings.scheduleGoldenHours && settings.scheduleGoldenHours.length > 0) {
        settings.goldenHours = [...settings.scheduleGoldenHours];
      }
      await SetPublishMode(mode);
      const updatedSettings = await GetSettings();
      if (updatedSettings) {
        if (updatedSettings.goldenHours) settings.goldenHours = updatedSettings.goldenHours;
        if (updatedSettings.scheduleGoldenHours) settings.scheduleGoldenHours = updatedSettings.scheduleGoldenHours;
        if (updatedSettings.publishNowGoldenHours) settings.publishNowGoldenHours = updatedSettings.publishNowGoldenHours;
      }
      await refreshSchedulerStatus();
      addLog('info', `Đã chuyển sang chế độ: ${mode === 'publish_now' ? 'Tự Động Đăng Ngay (Publish Now)' : 'Lên Lịch Trên Nền Tảng (Schedule)'}`);
    } catch (err) {
      addLog('error', `Lỗi đổi chế độ xuất bản: ${err}`);
    }
  }

  async function handleTriggerNow() {
    addLog('info', 'Kích hoạt đăng ngay 1 video theo yêu cầu thủ công...');
    try {
      await TriggerAutoUploadNow();
    } catch (err) {
      addLog('error', `Lỗi kích hoạt đăng ngay: ${err}`);
    }
  }

  async function handleAddGoldenHour() {
    if (!newSlotTime || !newSlotTime.includes(':')) {
      addLog('warn', 'Vui lòng chọn khung giờ hợp lệ (HH:mm)');
      return;
    }
    const cleanTime = newSlotTime.trim();
    if (activeHours.includes(cleanTime)) {
      addLog('warn', `Khung giờ ${cleanTime} đã tồn tại trong danh sách.`);
      return;
    }
    const updated = [...activeHours, cleanTime];
    await handleUpdateGoldenHours(updated);
  }

  async function handleRemoveGoldenHour(hourToRemove: string) {
    if (activeHours.length <= 1) {
      addLog('warn', 'Cần giữ ít nhất 1 khung giờ trong ngày.');
      return;
    }
    const updated = activeHours.filter(h => h !== hourToRemove);
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
      const modeLabel = settings.publishMode === 'publish_now' ? 'Đăng Ngay' : 'Lên Lịch';
      addLog('success', `Đã cập nhật khung giờ (${modeLabel}): ${hours.join(', ')}`);
    } catch (err) {
      addLog('error', `Lỗi cập nhật khung giờ: ${err}`);
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
        updateCheckMessage = `Bạn đang sử dụng phiên bản mới nhất (v${info.currentVersion}).`;
      }
    } catch (err: any) {
      if (!silent) {
        addLog('error', `Lỗi kiểm tra cập nhật: ${err?.message || err}`);
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
      addLog('info', `Bắt đầu tải và tự động cập nhật UpTik lên v${updateInfo.latestVersion}...`);
      const ok = await ApplyUpdate(updateInfo);
      if (ok) {
        isUpdateComplete = true;
        addLog('success', `Đã cập nhật UpTik lên phiên bản v${updateInfo.latestVersion} thành công!`);
      }
    } catch (err: any) {
      addLog('error', `Lỗi tự động cập nhật: ${err?.message || err}`);
    } finally {
      isApplyingUpdate = false;
    }
  }

  async function handleRestartApp() {
    try {
      addLog('info', 'Đang khởi động lại UpTik...');
      await RestartApp();
    } catch (err: any) {
      addLog('error', `Lỗi khởi động lại ứng dụng: ${err?.message || err}`);
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
      addLog('success', `Hoàn thành toàn bộ upload! Thành công: ${data.success}/${data.total}, Thất bại: ${data.failed}`);
      refreshVideos();
      refreshHistory();
      refreshSchedulerStatus();
    });

    EventsOn('upload_cancelled', () => {
      isUploading = false;
      addLog('warn', 'Tiến trình upload đã được dừng an toàn.');
    });

    EventsOn('queue_recovered', (data: { recoveredCount: number }) => {
      recoveredCount = data.recoveredCount;
      showRecoveryBanner = true;
      addLog('warn', `Phát hiện ${data.recoveredCount} video từ phiên làm việc trước chưa hoàn thành.`);
    });

    EventsOn('scheduler_toggled', (data: { autoUploadEnabled: boolean }) => {
      schedulerStatus.autoUploadEnabled = data.autoUploadEnabled;
      settings.autoUploadEnabled = data.autoUploadEnabled;
      refreshSchedulerStatus();
    });

    EventsOn('scheduler_notification', (data: { title: string; message: string }) => {
      addLog('info', `[Thông báo] ${data.title}: ${data.message}`);
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
          title="Có phiên bản mới v{updateInfo.latestVersion}. Bấm để cập nhật"
        >
          <Sparkles class="w-3.5 h-3.5 text-cyan-400" />
          <span>v{updateInfo.latestVersion} Mới!</span>
        </button>
      {/if}

      <!-- Quick Platform Open Dropdown / Buttons -->
      <div class="hidden sm:flex items-center gap-1 bg-neutral-900 border border-neutral-800 rounded-lg p-0.5">
        <button
          onclick={() => handleOpenPlatform('tiktok')}
          title="Mở Chrome TikTok Studio"
          class="flex items-center gap-1.5 px-2.5 py-1 text-xs text-neutral-300 hover:text-white hover:bg-neutral-800 rounded transition"
        >
          <Music2 class="w-3.5 h-3.5 text-rose-400" />
          <span>{m.platform_tiktok()}</span>
        </button>
        <button
          onclick={() => handleOpenPlatform('youtube')}
          title="Mở Chrome YouTube Studio"
          class="flex items-center gap-1.5 px-2.5 py-1 text-xs text-neutral-300 hover:text-white hover:bg-neutral-800 rounded transition"
        >
          <PlaySquare class="w-3.5 h-3.5 text-red-500" />
          <span>{m.platform_youtube()}</span>
        </button>
        <button
          onclick={() => handleOpenPlatform('facebook')}
          title="Mở Chrome Meta Business Suite"
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
          title="Bắt đầu lên lịch đa kênh ({settings.enabledChannels.join(', ')})"
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
                  {schedulerStatus.autoUploadEnabled ? 'ON (BACKGROUND)' : 'OFF'}
                </span>
                <span class="px-2 py-0.5 text-[10px] font-mono rounded bg-neutral-800 text-neutral-300 border border-neutral-700">
                  {settings.publishMode === 'publish_now' ? m.publish_mode_now() : m.publish_mode_schedule()}
                </span>
              </div>
              <p class="text-xs text-neutral-400 mt-1">
                {#if schedulerStatus.autoUploadEnabled}
                  Next: <strong class="text-white font-mono text-sm">{schedulerStatus.nextTime}</strong> ({schedulerStatus.slotLabel})
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
              <span>{schedulerStatus.autoUploadEnabled ? 'STOP AUTO' : 'START AUTO'}</span>
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
                      <CheckCircle2 class="w-3 h-3" /> Đã Lên Lịch
                    </span>
                  {:else if v.status === 'uploading'}
                    <span class="flex items-center gap-1 text-[11px] text-[#E50914] bg-[#E50914]/15 border border-[#E50914]/40 px-2 py-0.5 rounded font-bold animate-pulse">
                      Đang Upload...
                    </span>
                  {:else if v.status === 'error'}
                    <span class="flex items-center gap-1 text-[11px] text-red-400 bg-red-950 border border-red-800 px-2 py-0.5 rounded font-medium">
                      <AlertCircle class="w-3 h-3" /> Lỗi
                    </span>
                  {:else if v.scheduledDate}
                    <span class="text-[11px] text-blue-400 bg-blue-950/60 border border-blue-800/60 px-2 py-0.5 rounded font-medium">
                      Sẵn sàng
                    </span>
                  {:else}
                    <span class="text-[11px] text-neutral-400 bg-neutral-800/60 border border-neutral-700 px-2 py-0.5 rounded">
                      Chờ xếp lịch
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
                    <span class="text-[11px] text-neutral-500 italic">Chưa phân bổ khung giờ</span>
                  {/if}

                  <!-- Edit Action Button -->
                  <button
                    onclick={() => openEditDialog(v)}
                    title="Chỉnh sửa tiêu đề hoặc khung giờ"
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
            <h2 class="text-base font-bold text-white">Ma Trận 30 Ngày (3 Khung Giờ Vàng Mỗi Ngày)</h2>
            <p class="text-xs text-neutral-400 mt-0.5">
              Theo quy định của TikTok Studio: Cho phép lên lịch tối đa 30 ngày (tối đa 90 video cho 3 khung giờ: 11:30, 18:30, 21:30).
            </p>
          </div>
          <div class="flex items-center gap-4 text-xs">
            <div class="flex items-center gap-1.5">
              <div class="w-3 h-3 rounded bg-amber-500/20 border border-amber-500"></div>
              <span class="text-neutral-300">11:30 Trưa</span>
            </div>
            <div class="flex items-center gap-1.5">
              <div class="w-3 h-3 rounded bg-orange-500/20 border border-orange-500"></div>
              <span class="text-neutral-300">18:30 Chiều</span>
            </div>
            <div class="flex items-center gap-1.5">
              <div class="w-3 h-3 rounded bg-indigo-500/20 border border-indigo-500"></div>
              <span class="text-neutral-300">21:30 Tối</span>
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
                  {Object.keys(day.slots).length}/3 slots
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
                      <span class="text-[9px] bg-emerald-900/60 text-emerald-300 px-1.5 py-0.2 rounded">Đã Đăng</span>
                    {:else if day.slots['11:30']?.video}
                      <span class="text-[9px] bg-blue-900/60 text-blue-300 px-1.5 py-0.2 rounded">Sẵn Sàng</span>
                    {:else}
                      <span class="text-[9px] text-neutral-600">Trống</span>
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
                      <span class="text-[9px] bg-emerald-900/60 text-emerald-300 px-1.5 py-0.2 rounded">Đã Đăng</span>
                    {:else if day.slots['18:30']?.video}
                      <span class="text-[9px] bg-blue-900/60 text-blue-300 px-1.5 py-0.2 rounded">Sẵn Sàng</span>
                    {:else}
                      <span class="text-[9px] text-neutral-600">Trống</span>
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
                      <span class="text-[9px] bg-emerald-900/60 text-emerald-300 px-1.5 py-0.2 rounded">Đã Đăng</span>
                    {:else if day.slots['21:30']?.video}
                      <span class="text-[9px] bg-blue-900/60 text-blue-300 px-1.5 py-0.2 rounded">Sẵn Sàng</span>
                    {:else}
                      <span class="text-[9px] text-neutral-600">Trống</span>
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
            <h2 class="text-base font-bold text-white">Lịch Sử Upload & Kho Lưu Trữ (Anti-Duplication)</h2>
            <p class="text-xs text-neutral-400 mt-0.5">
              Toàn bộ video sau khi lên lịch thành công đã được di chuyển an toàn vào thư mục con <code class="text-neutral-300 bg-neutral-800 px-1.5 py-0.5 rounded">uploaded/</code>.
            </p>
          </div>
          <div class="flex items-center gap-2">
            <button
              onclick={() => OpenInFileManager(`${settings.videoFolder}/uploaded`)}
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-neutral-300 bg-neutral-800 hover:bg-neutral-700 rounded border border-neutral-700 transition"
            >
              <Folder class="w-3.5 h-3.5 text-amber-400" />
              <span>Mở Thư Mục Uploaded</span>
            </button>
            <a
              href="https://www.tiktok.com/tiktokstudio/content"
              target="_blank"
              class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-[#E50914] hover:bg-[#F40612] rounded transition"
            >
              <ExternalLink class="w-3.5 h-3.5" />
              <span>Xem Trên TikTok Studio</span>
            </a>
          </div>
        </div>

        <div class="bg-[#181818] border border-neutral-800 rounded-xl overflow-hidden flex-1 overflow-y-auto max-h-[calc(100vh-230px)]">
          <table class="w-full text-left text-xs">
            <thead class="bg-neutral-900 border-b border-neutral-800 text-neutral-400 sticky top-0">
              <tr>
                <th class="py-3 px-4 font-semibold">STT</th>
                <th class="py-3 px-4 font-semibold">Tiêu Đề Lên Lịch</th>
                <th class="py-3 px-4 font-semibold">Kênh Phân Phối</th>
                <th class="py-3 px-4 font-semibold">Ngày Đăng</th>
                <th class="py-3 px-4 font-semibold">Khung Giờ</th>
                <th class="py-3 px-4 font-semibold">File Gốc</th>
                <th class="py-3 px-4 font-semibold">Thời Gian Ghi Nhận</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-neutral-800">
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
                  <td class="py-3 px-4 text-neutral-500 text-[11px]">{new Date(h.timestamp).toLocaleString('vi-VN')}</td>
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
            <h3 class="text-sm font-bold text-white">Nhật Ký Tự Động Hóa Chrome CDP (Console)</h3>
          </div>
          <div class="flex items-center gap-3 text-xs">
            <label class="flex items-center gap-1.5 text-neutral-400 cursor-pointer">
              <input type="checkbox" bind:checked={autoScrollLogs} class="rounded border-neutral-700 bg-neutral-900 text-[#E50914]" />
              <span>Tự động cuộn</span>
            </label>
            <button
              onclick={() => logs = []}
              class="flex items-center gap-1 text-neutral-400 hover:text-white px-2 py-1 rounded bg-neutral-800 transition"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>Xóa log</span>
            </button>
          </div>
        </div>

        <div
          bind:this={logContainer}
          class="flex-1 bg-black border border-neutral-800 rounded-xl p-4 font-mono text-xs overflow-y-auto max-h-[calc(100vh-230px)] space-y-1.5"
        >
          {#if logs.length === 0}
            <div class="text-neutral-600 italic">Chưa có nhật ký nào được ghi lại...</div>
          {/if}
          {#each logs as log}
            <div class="flex items-start gap-2.5 leading-relaxed">
              <span class="text-neutral-600 shrink-0">[{log.timestamp}]</span>
              {#if log.level === 'cdp'}
                <span class="text-cyan-400 font-semibold shrink-0">[CDP]</span>
              {:else if log.level === 'success'}
                <span class="text-emerald-400 font-semibold shrink-0">[THÀNH CÔNG]</span>
              {:else if log.level === 'warn'}
                <span class="text-amber-400 font-semibold shrink-0">[CẢNH BÁO]</span>
              {:else if log.level === 'error'}
                <span class="text-red-500 font-semibold shrink-0">[LỖI]</span>
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
      <Tabs.Content value="settings" class="flex-1 max-w-2xl">
        <div class="bg-[#181818] border border-neutral-800 rounded-xl p-6 space-y-5">
          <h2 class="text-base font-bold text-white border-b border-neutral-800 pb-3">
            Cấu Hình Hệ Thống & Trình Duyệt Chrome
          </h2>

          <!-- Folder path -->
          <div>
            <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
              Thư Mục Chứa Video (.mp4)
            </span>
            <div class="flex items-center gap-2">
              <input
                type="text"
                bind:value={settings.videoFolder}
                class="flex-1 bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
              />
              <button
                onclick={handleSelectFolder}
                class="px-3 py-2 bg-neutral-800 hover:bg-neutral-700 text-xs font-semibold text-neutral-200 rounded-lg border border-neutral-700 transition"
              >
                Chọn...
              </button>
            </div>
          </div>

          <!-- Chrome User Data Dir -->
          <div>
            <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
              Thư Mục Chrome Profile (Đã đăng nhập TikTok Studio)
            </span>
            <input
              type="text"
              bind:value={settings.chromeUserDataDir}
              class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
            />
            <p class="text-[11px] text-neutral-500 mt-1">
              Khuyến nghị dùng profile riêng: <code>/home/arch/.config/google-chrome-mcp</code> để không ảnh hưởng đến phiên duyệt web cá nhân.
            </p>
          </div>

          <!-- Chrome Executable -->
          <div>
            <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
              Đường Dẫn File Thực Thi Chrome
            </span>
            <input
              type="text"
              bind:value={settings.chromePath}
              class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
            />
          </div>

          <!-- Default Tags -->
          <div>
            <span class="block text-xs font-semibold text-neutral-300 mb-1.5">
              Hashtags Mặc Định Tự Động Thêm Vào Caption
            </span>
            <input
              type="text"
              bind:value={settings.defaultTag}
              placeholder="#phimbop #movie #shorts"
              class="w-full bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-[#E50914]"
            />
          </div>

          <!-- PUBLISHING MODE SELECTION -->
          <div class="bg-neutral-900/80 border border-neutral-700/60 p-4 rounded-xl space-y-3">
            <div class="flex items-center justify-between">
              <span class="block text-xs font-bold text-white flex items-center gap-2">
                <Rocket class="w-4 h-4 text-[#E50914]" />
                <span>Chế Độ Xuất Bản Video (Publishing Mode)</span>
              </span>
              <span class="text-[10px] font-mono text-neutral-300 bg-neutral-800 px-2 py-0.5 rounded border border-neutral-700">
                {settings.publishMode === 'publish_now' ? 'Tự Động Đăng Ngay (Publish Now)' : 'Lên Lịch Trên Nền Tảng (Schedule)'}
              </span>
            </div>
            <p class="text-[11px] text-neutral-400">
              Lựa chọn phương thức hệ thống thực hiện khi xuất bản video lên các nền tảng:
            </p>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <!-- Mode 1: Platform Schedule -->
              <label class="flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition {settings.publishMode === 'schedule' ? 'bg-neutral-800/90 border-[#E50914] text-white shadow-[0_0_15px_rgba(229,9,20,0.15)]' : 'bg-neutral-900/40 border-neutral-800 text-neutral-400 hover:text-neutral-200'}">
                <input
                  type="radio"
                  name="publishMode"
                  value="schedule"
                  checked={settings.publishMode === 'schedule'}
                  onchange={() => handleSetPublishMode('schedule')}
                  class="mt-1 text-[#E50914] focus:ring-0"
                />
                <div>
                  <div class="flex items-center gap-1.5">
                    <Calendar class="w-3.5 h-3.5 text-amber-400" />
                    <span class="text-xs font-bold">Lên Lịch Trên Nền Tảng</span>
                  </div>
                  <p class="text-[11px] text-neutral-400 mt-1 leading-relaxed">
                    Mở trình duyệt và cài đặt ngày giờ hẹn phát (Schedule) trực tiếp trên TikTok Studio, YouTube Shorts, Meta Business Suite.
                  </p>
                </div>
              </label>

              <!-- Mode 2: Auto Publish Now -->
              <label class="flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition {settings.publishMode === 'publish_now' ? 'bg-neutral-800/90 border-emerald-500 text-white shadow-[0_0_15px_rgba(16,185,129,0.15)]' : 'bg-neutral-900/40 border-neutral-800 text-neutral-400 hover:text-neutral-200'}">
                <input
                  type="radio"
                  name="publishMode"
                  value="publish_now"
                  checked={settings.publishMode === 'publish_now'}
                  onchange={() => handleSetPublishMode('publish_now')}
                  class="mt-1 text-emerald-500 focus:ring-0"
                />
                <div>
                  <div class="flex items-center gap-1.5">
                    <Zap class="w-3.5 h-3.5 text-emerald-400" />
                    <span class="text-xs font-bold">Tự Động Đăng Ngay (Publish Now)</span>
                  </div>
                  <p class="text-[11px] text-neutral-400 mt-1 leading-relaxed">
                    UpTik chạy ngầm, đến đúng các khung giờ đã định sẽ tự động lấy video và bấm Đăng ngay (Publish Now) mà không cần thao tác tay.
                  </p>
                </div>
              </label>
            </div>
          </div>

          <!-- TIME SLOTS MANAGER -->
          <div class="bg-neutral-900/80 border border-neutral-700/60 p-4 rounded-xl space-y-4">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between border-b border-neutral-800 pb-3 gap-2">
              <div>
                <div class="flex items-center gap-2">
                  <Clock class="w-4 h-4 {settings.publishMode === 'publish_now' ? 'text-emerald-400' : 'text-amber-400'}" />
                  <span class="text-xs font-bold text-white">
                    Quản Lý Khung Giờ {settings.publishMode === 'publish_now' ? 'Đăng Ngay' : 'Lên Lịch'} ({activeHours.length} Khung Giờ)
                  </span>
                  <span class="text-[10px] px-2 py-0.5 rounded font-semibold border {settings.publishMode === 'publish_now' ? 'bg-emerald-950/60 border-emerald-600/50 text-emerald-400' : 'bg-amber-950/60 border-amber-600/50 text-amber-400'}">
                    {settings.publishMode === 'publish_now' ? 'Chế độ Đăng Ngay' : 'Chế độ Lên Lịch'}
                  </span>
                </div>
                <p class="text-[11px] text-neutral-400 mt-1">
                  {settings.publishMode === 'publish_now'
                    ? 'Các khung giờ chạy ngầm để UpTik tự động lấy video và bấm Đăng ngay. Cấu hình được lưu riêng biệt cho chế độ Đăng Ngay.'
                    : 'Các khung giờ dùng để tự động phân bổ ngày/giờ hẹn phát trên TikTok Studio, Shorts, Reels. Cấu hình được lưu riêng biệt cho chế độ Lên Lịch.'}
                </p>
              </div>

              <!-- Quick Presets -->
              <div class="flex items-center gap-1.5 flex-wrap">
                <span class="text-[10px] text-neutral-500 font-semibold uppercase">Presets:</span>
                <button
                  type="button"
                  onclick={() => applyPresetHours(['11:30', '18:30', '21:30'])}
                  class="px-2 py-0.5 text-[10px] font-medium bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded border border-neutral-700 transition"
                  title="3 khung giờ vàng chuẩn: 11:30, 18:30, 21:30"
                >
                  3 Giờ Vàng
                </button>
                <button
                  type="button"
                  onclick={() => applyPresetHours(['08:30', '11:30', '17:30', '20:30'])}
                  class="px-2 py-0.5 text-[10px] font-medium bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded border border-neutral-700 transition"
                  title="4 khung giờ tiêu chuẩn"
                >
                  4 Giờ Chuẩn
                </button>
                <button
                  type="button"
                  onclick={() => applyPresetHours(['07:30', '11:30', '14:30', '18:30', '21:30'])}
                  class="px-2 py-0.5 text-[10px] font-medium bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded border border-neutral-700 transition"
                  title="5 khung giờ dày đặc"
                >
                  5 Giờ Dày
                </button>
              </div>
            </div>

            <!-- Current Slots Badges -->
            <div>
              <span class="block text-[11px] font-semibold text-neutral-400 mb-2">
                Các khung giờ đang áp dụng cho {settings.publishMode === 'publish_now' ? 'Đăng Ngay' : 'Lên Lịch'} (Bấm dấu X để xóa):
              </span>
              <div class="flex flex-wrap gap-2">
                {#each activeHours as h}
                  {@const label = getSlotLabel(h)}
                  <div class="flex items-center gap-2 px-3 py-1.5 bg-neutral-800/90 border border-neutral-700 hover:border-neutral-500 rounded-lg text-xs font-semibold text-white transition group">
                    <Clock class="w-3.5 h-3.5 {settings.publishMode === 'publish_now' ? 'text-emerald-400' : 'text-amber-400'}" />
                    <span class="font-mono">{label}</span>
                    <button
                      type="button"
                      onclick={() => handleRemoveGoldenHour(h)}
                      title="Xóa khung giờ {h}"
                      class="p-0.5 text-neutral-400 hover:text-red-400 hover:bg-neutral-700 rounded transition ml-1"
                    >
                      <X class="w-3.5 h-3.5" />
                    </button>
                  </div>
                {/each}
              </div>
            </div>

            <!-- Add New Slot Form -->
            <div class="pt-3 border-t border-neutral-800 flex items-center gap-3 flex-wrap">
              <span class="text-xs font-semibold text-neutral-300">Thêm khung giờ mới:</span>
              <div class="flex items-center gap-2">
                <input
                  type="time"
                  bind:value={newSlotTime}
                  class="bg-neutral-900 border border-neutral-700 rounded-lg px-3 py-1.5 text-xs text-white focus:outline-none focus:border-[#E50914] font-mono"
                />
                <button
                  type="button"
                  onclick={handleAddGoldenHour}
                  class="flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold text-white bg-[#E50914] hover:bg-[#F40612] rounded-lg transition active:scale-95 shadow-sm"
                >
                  <Plus class="w-3.5 h-3.5" />
                  <span>Thêm Khung Giờ</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Omnichannel Targets -->
          <div class="bg-neutral-900/80 border border-neutral-700/60 p-4 rounded-xl space-y-3">
            <div class="flex items-center justify-between">
              <span class="block text-xs font-bold text-white">
                Nền Tảng Phân Phối Đa Kênh (Omnichannel Targets)
              </span>
              <span class="text-[10px] text-neutral-400 bg-neutral-800 px-2 py-0.5 rounded">
                Đang bật {settings.enabledChannels.length}/3 kênh
              </span>
            </div>
            <p class="text-[11px] text-neutral-400">
              Chọn các mạng xã hội video ngắn mà hệ thống sẽ tự động lên lịch đăng video song song:
            </p>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              {#each platforms as p}
                {@const isChecked = settings.enabledChannels.includes(p.id)}
                <label class="flex items-center gap-2.5 p-3 rounded-lg border cursor-pointer transition {isChecked ? 'bg-neutral-800/90 border-neutral-600 text-white' : 'bg-neutral-900/40 border-neutral-800 text-neutral-400 hover:text-neutral-200'}">
                  <input
                    type="checkbox"
                    checked={isChecked}
                    onchange={() => toggleChannel(p.id)}
                    class="rounded border-neutral-700 bg-neutral-900 text-[#E50914] focus:ring-0"
                  />
                  <p.icon class="w-5 h-5 text-neutral-300" />
                  <div class="flex flex-col">
                    <span class="text-xs font-semibold">{p.name}</span>
                    <span class="text-[10px] text-neutral-500 font-mono">{p.id === 'tiktok' ? 'TikTok Studio' : p.id === 'youtube' ? 'YouTube Studio' : 'Business Suite'}</span>
                  </div>
                </label>
              {/each}
            </div>
            <div class="pt-2 border-t border-neutral-800 flex items-center gap-2 flex-wrap text-xs">
              <span class="text-neutral-400 text-[11px]">Đăng nhập nhanh:</span>
              {#each platforms as p}
                <button
                  type="button"
                  onclick={() => handleOpenPlatform(p.id)}
                  class="flex items-center gap-1.5 px-2.5 py-1 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs rounded border border-neutral-700 transition"
                >
                  <p.icon class="w-3 h-3" />
                  <span>{p.name}</span>
                  <ExternalLink class="w-3 h-3 text-neutral-500" />
                </button>
              {/each}
            </div>
          </div>

          <!-- Startup & Background Settings -->
          <div class="bg-neutral-900/80 border border-neutral-700/60 p-4 rounded-xl space-y-4">
            <div class="flex items-center justify-between border-b border-neutral-800 pb-2.5">
              <div>
                <span class="text-xs font-bold text-white flex items-center gap-1.5">
                  <Zap class="w-4 h-4 text-amber-400" />
                  <span>Khởi Động Cùng Hệ Thống & Chạy Ngầm</span>
                </span>
                <p class="text-[11px] text-neutral-400 mt-0.5">
                  Tùy biến hành vi tự khởi động khi mở máy và thu nhỏ vào khay hệ thống (System Tray)
                </p>
              </div>
            </div>

            <!-- Toggle 1: AutoStart -->
            <div class="flex items-start justify-between gap-4 p-3 rounded-lg bg-neutral-900/50 border border-neutral-800">
              <div class="space-y-0.5 flex-1">
                <span class="text-xs font-semibold text-neutral-200 flex items-center gap-1.5">
                  <Rocket class="w-4 h-4 text-[#E50914]" />
                  <span>Tự động khởi động cùng hệ thống (Auto-start on boot)</span>
                </span>
                <p class="text-[11px] text-neutral-400">
                  Tự động kích hoạt UpTik khi bạn đăng nhập vào máy tính, sẵn sàng thực thi lịch đăng tải tự động.
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
              <div class="flex items-start justify-between gap-4 p-3 ml-4 rounded-lg bg-neutral-900/30 border border-neutral-800/80 transition-all">
                <div class="space-y-0.5 flex-1">
                  <span class="text-xs font-semibold text-neutral-300 flex items-center gap-1.5">
                    <EyeOff class="w-3.5 h-3.5 text-neutral-400" />
                    <span>Khởi động ẩn dưới khay hệ thống (Start minimized to tray)</span>
                  </span>
                  <p class="text-[11px] text-neutral-500">
                    Khi bật máy, UpTik sẽ tự động chạy ngầm dưới khay thông báo mà không bung mở cửa sổ giao diện.
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
            <div class="flex items-start justify-between gap-4 p-3 rounded-lg bg-neutral-900/50 border border-neutral-800">
              <div class="space-y-0.5 flex-1">
                <span class="text-xs font-semibold text-neutral-200 flex items-center gap-1.5">
                  <Shield class="w-4 h-4 text-emerald-400" />
                  <span>Chạy ngầm khi đóng ứng dụng (Minimize to Tray on close)</span>
                </span>
                <p class="text-[11px] text-neutral-400">
                  Khi nhấn nút đóng [X], ứng dụng sẽ tiếp tục chạy ngầm trong khay hệ thống để duy trì các tác vụ upload và lịch hẹn thay vì tắt hẳn.
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
          <!-- SOFTWARE UPDATE SECTION (AUTO-UPDATE LEVEL 2) -->
          <div class="bg-neutral-900/80 border border-neutral-700/60 p-4 rounded-xl space-y-3">
            <div class="flex items-center justify-between border-b border-neutral-800 pb-2.5">
              <div>
                <span class="text-xs font-bold text-white flex items-center gap-1.5">
                  <Sparkles class="w-4 h-4 text-cyan-400" />
                  <span>Cập Nhật Phần Mềm Tự Động (Auto-Updater)</span>
                </span>
                <p class="text-[11px] text-neutral-400 mt-0.5">
                  Phiên bản hiện tại: <span class="font-mono text-white font-semibold">v{appVersion}</span>
                </p>
              </div>

              <button
                type="button"
                onclick={() => handleCheckUpdate(false)}
                disabled={isCheckingUpdate || isApplyingUpdate}
                class="px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 disabled:opacity-50 text-xs font-semibold text-neutral-200 rounded-lg border border-neutral-700 transition flex items-center gap-1.5"
              >
                <RefreshCw class="w-3.5 h-3.5 {isCheckingUpdate ? 'animate-spin' : ''}" />
                <span>{isCheckingUpdate ? 'Đang kiểm tra...' : 'Kiểm tra cập nhật'}</span>
              </button>
            </div>

            {#if updateInfo}
              {#if updateInfo.available}
                <div class="p-3.5 bg-cyan-950/30 border border-cyan-800/60 rounded-xl space-y-3">
                  <div class="flex items-start justify-between gap-3">
                    <div class="space-y-1 flex-1">
                      <div class="flex items-center gap-2">
                        <span class="px-2 py-0.5 bg-cyan-500 text-black text-[10px] font-extrabold rounded">CÓ BẢN MỚI</span>
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
                        class="px-3 py-1.5 bg-cyan-500 hover:bg-cyan-400 disabled:opacity-50 text-xs font-bold text-black rounded-lg transition shrink-0 flex items-center gap-1.5 shadow-md active:scale-95"
                      >
                        <Download class="w-3.5 h-3.5" />
                        <span>{isApplyingUpdate ? 'Đang cập nhật...' : 'Cập nhật ngay'}</span>
                      </button>
                    {:else}
                      <button
                        type="button"
                        onclick={handleRestartApp}
                        class="px-3 py-1.5 bg-emerald-500 hover:bg-emerald-400 text-xs font-bold text-black rounded-lg transition shrink-0 flex items-center gap-1.5 shadow-md active:scale-95 animate-pulse"
                      >
                        <RotateCcw class="w-3.5 h-3.5" />
                        <span>Khởi động lại ngay</span>
                      </button>
                    {/if}
                  </div>

                  {#if isApplyingUpdate}
                    <div class="space-y-1.5 pt-1">
                      <div class="flex justify-between text-[11px] text-neutral-300">
                        <span>Đang tải bản cập nhật và kiểm tra mã SHA256...</span>
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
                      <span>Đã tải và cài đặt bản cập nhật thành công! Hãy nhấn "Khởi động lại ngay" để trải nghiệm.</span>
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

          <div class="pt-4 border-t border-neutral-800 flex items-center justify-between">
            <button
              type="button"
              onclick={handleQuitApp}
              class="px-4 py-2 bg-neutral-800/80 hover:bg-red-950/60 text-xs font-semibold text-neutral-300 hover:text-red-400 rounded-lg border border-neutral-700/80 hover:border-red-800/80 transition flex items-center gap-2"
              title="Đóng hoàn toàn tiến trình ứng dụng"
            >
              <Power class="w-3.5 h-3.5" />
              <span>{m.settings_btn_quit()}</span>
            </button>

            <button
              onclick={handleSaveSettings}
              class="px-5 py-2 bg-[#E50914] hover:bg-[#F40612] text-xs font-bold text-white rounded-lg shadow-md transition flex items-center gap-2"
            >
              <Save class="w-3.5 h-3.5" />
              <span>{m.settings_btn_save()}</span>
            </button>
          </div>
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
