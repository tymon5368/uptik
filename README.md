<div align="center">

<img src="brand/uptik-logo-mark-transparent-1024.png" alt="UpTik Logo" width="120" height="120" />

# UpTik
### Omnichannel Video Auto-Scheduler & Desktop Studio

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-black?style=flat-square&logo=linux)](https://github.com/tymon5368/uptik/releases)
[![Wails v2](https://img.shields.io/badge/Wails-v2.15-DF1B52?style=flat-square&logo=go)](https://wails.io)
[![Svelte 5](https://img.shields.io/badge/Svelte-5%20Runes-FF3E00?style=flat-square&logo=svelte)](https://svelte.dev)
[![Bun](https://img.shields.io/badge/Bun-1.2+-fbf0df?style=flat-square&logo=bun)](https://bun.sh)
[![Release Workflow](https://img.shields.io/github/actions/workflow/status/tymon5368/uptik/release.yml?style=flat-square&label=Release%20CI)](https://github.com/tymon5368/uptik/actions)

<p>
  <strong>UpTik</strong> is a high-performance native desktop application designed to automate scheduling and omnichannel publishing of short-form videos across multiple creator platforms: <strong>TikTok Studio</strong>, <strong>YouTube Shorts</strong>, and <strong>Meta Business Suite (Facebook Reels)</strong> using standardized peak-engagement golden hours.
</p>

</div>

---

## 🌟 Key Features

### 1. Omnichannel Golden Hour Scheduler
- **Automated Time-Slot Allocation**: Intelligently schedules videos across the top 3 peak daily engagement windows: **11:30 AM (Midday)**, **6:30 PM (Evening)**, and **9:30 PM (Night)** across an upcoming 30-day horizon.
- **Collision Guard**: Continuously checks publication history from SQLite and local storage directories to guarantee that no two videos collide in the same time slot on any given channel.

### 2. Zero-API Browser Automation (Chrome CDP Driver)
- Operates directly on platform web management interfaces through the **Chrome DevTools Protocol (CDP)** powered by Go Rod.
- Reuses existing logged-in sessions and browser profiles—eliminating the need for restricted, costly official enterprise APIs or complex app review processes.
- Full compatibility with modern rich-text editors (Draft.js, Lexical) used by TikTok and Meta via native simulated keyboard and input events.

### 3. Crash-Resilient SQLite Outbox Queue
- Durable job storage managed by **ModernC SQLite** (pure Go, zero-CGO, WAL mode with a 5000ms busy timeout).
- **Self-Healing State Machine**: In case of unexpected system crashes, shutdowns, or process termination, all incomplete in-flight jobs are safely recovered to `pending` upon next startup.

### 4. Safety Circuit Breaker & Anti-Ban Protection
- Adapters continuously analyze DOM responses from social media platforms.
- Immediately trips the **Circuit Breaker** upon detecting sensitive roadblocks (captchas, identity checkpoints, phone verification, rate limits / HTTP 429), pausing the queue instantly to safeguard creator accounts.

### 5. Native Desktop Lifecycle & System Tray
- **System Tray Integration**: Full support for D-Bus StatusNotifierItem (SNI) on Linux and native trays on Windows and macOS.
- **Close-to-Tray (24/7 Background Run)**: Closing the main application window minimizes it seamlessly to the system tray, keeping scheduled video uploads uninterrupted.
- **OS Autostart**: Built-in toggle to launch automatically on system boot in background mode (`--hidden`).

### 6. Netflix Studio Dark Theme UI (Svelte 5 Runes)
- Engineered on **Svelte 5** leveraging modern runes (`$state`, `$derived`, `$effect`) for reactive rendering performance.
- Sleek, Netflix Studio-inspired dark aesthetic featuring accessible tabbed navigation powered by **Ark UI Svelte**.
- **100% Lucide Icons** (`lucide-svelte`) for a consistent, professional design system free of arbitrary emoji glyphs.

---

## 🏗️ System Architecture (Hexagonal Ports & Adapters)

UpTik strictly adheres to **Hexagonal Architecture** (Ports and Adapters) to isolate domain rules from platform-specific delivery mechanisms:

```
uptik/
├── internal/
│   ├── domain/               # Pure domain logic, models, golden hour slot calculation
│   ├── ports/                # Contract boundaries (JobQueuePort, StoragePort, PlatformUploader)
│   ├── usecases/             # Application workflows (upload pipeline, video scan, scheduler)
│   └── adapters/             # Concrete infrastructural adapters
│       ├── storage/sqlite/   # SQLite database (WAL mode, automated schema migrations)
│       ├── queue/            # Resilient outbox queue (atomic lease & crash recovery)
│       ├── autostart/        # Cross-platform startup launcher integration
│       └── platforms/        # Platform-specific browser upload controllers
│           ├── tiktok/       # TikTok Studio CDP Driver
│           ├── youtube/      # YouTube Shorts CDP Driver
│           └── facebook/     # Meta Business Suite CDP Driver
├── frontend/                 # Svelte 5 + Vite + Tailwind CSS (Netflix Studio Dark Theme)
├── brand/                    # Brand identity assets, squircle logo, vector favicons
├── build/                    # Binary packaging manifests and multi-platform app icons
└── .github/workflows/        # Automated multi-platform CI/CD release workflow
```

---

## 💻 Prerequisites

- **Go**: 1.23 or higher
- **Bun**: 1.2 or higher (Mandatory package manager for frontend dependencies)
- **Wails CLI**: v2.15.0 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0`)
- **Google Chrome**: Installed on the host system for CDP browser automation

### System Dependencies on Linux (Debian / Ubuntu)
```bash
sudo apt update
sudo apt install -y libgtk-3-dev libwebkit2gtk-4.1-dev libayatana-appindicator3-dev
```

### System Dependencies on Linux (Arch Linux)
```bash
sudo pacman -S gtk3 webkit2gtk-4.1 libappindicator-gtk3
```

---

## 🛠️ Development Guide

### 1. Install Frontend Dependencies
```bash
cd frontend
bun install
cd ..
```

### 2. Launch in Live Development Mode
```bash
wails dev
```
Wails starts the Vite dev server with Hot Module Replacement (HMR) and automatically binds Go backend methods to the desktop webview.

### 3. Run Quality Gates & Tests
```bash
# Verify TypeScript and Svelte 5 compiler diagnostics
cd frontend && bun run check && cd ..

# Execute all backend unit test suites
go test -tags "webkit2_41" ./...
```

---

## 📦 Multi-Platform Binary Packaging

### 🐧 1. Linux (Ubuntu, Debian, Fedora, Arch Linux)
```bash
wails build -platform linux/amd64 -tags "webkit2_41" -clean -trimpath
```
*Generated executable: `build/bin/uptik`*

### 🪟 2. Windows (x86_64)
Builds both an NSIS installer and a standalone portable executable:
```bash
wails build -platform windows/amd64 -nsis -clean -trimpath
```
*Generated files: `build/bin/uptik-amd64-installer.exe` and `build/bin/uptik.exe`*

### 🍎 3. macOS (Apple Silicon & Intel Universal)
```bash
wails build -platform darwin/universal -clean -trimpath
```
*Generated application bundle: `build/bin/uptik.app`*

---

## 🚀 Release & CI/CD Workflow

The repository includes a battle-tested **GitHub Actions Multi-Platform Release Workflow** configured at [`.github/workflows/release.yml`](.github/workflows/release.yml).

### Method 1: Automated Release Script (Recommended)
Run the automated release helper:
```bash
./scripts/release.sh 1.0.0
```
The script will sequentially:
1. Synchronize version declarations in `version.go`, `wails.json`, and `frontend/package.json`.
2. Run frontend type checking and production build (`bun run check`, `bun run build`).
3. Run all backend tests (`go test ./...`).
4. Commit the release bump: `chore(release): bump version to v1.0.0`.
5. Create a signed/annotated git tag: `v1.0.0`.

Push the tag to GitHub to trigger automated binary builds:
```bash
git push origin main --tags
```

### Method 2: Manual Trigger via GitHub Actions
Navigate to the **Actions** tab on GitHub > select the **Release UpTik** workflow > click **Run workflow** and enter the desired semver tag.

The workflow automatically:
- Builds concurrently across 3 runners: Linux (`ubuntu-22.04`), Windows (`windows-latest`), and macOS (`macos-latest`).
- Bundles installers and compressed archives for each target operating system.
- Generates a cryptographic verification file `SHA256SUMS.txt`.
- Publishes a formal **GitHub Release** populated with all distribution assets and automated release notes.

---

## 📄 License

This project is licensed under the **[MIT License](LICENSE)**. Free and open-source for personal and commercial use.
