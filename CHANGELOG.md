# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-03-12

### Added
- **ASCII Latency History Chart**: Added an inline sparkline plot using `asciigraph` to the terminal UI that visually tracks latency history.
- **System Tray Integration**: Added a cross-platform system tray icon using `getlantern/systray`.
  - Displays live Latency and Jitter metrics clearly using Unicode mathematical bold characters directly in the menu bar.
  - Interactive dropdown showing the most recent network event (log) and the top bandwidth-consuming application.
- **Background Daemon Mode**: Added a `-bg` (or `-b`) flag.
  - Completely detaches the application from the host terminal, running silently as a background process while continuing to update the system tray.

## [1.1.0] - 2026-03-12

### Added
- **Interactive TUI**: Upgraded to a full Terminal User Interface (TUI) using `Bubble Tea` and `Lip Gloss`.
- **Log Box**: Added a centered box for event logging (alerts, resets, high latency).
- **Status Bar**: Added a bottom status bar for real-time metrics (Lat, Jitter, Loss).
- **Auto-Scan Target**: Automatically scans and selects the best DNS target on startup if none is specified.
- **Internet Check**: Added pre-startup internet connectivity verification.
- **Traffic Visibility**: Added a "Top App" / "Net Load" monitor to identify bandwidth-heavy processes.
- **Improved Logging**: Background library errors are now silenced to prevent TUI corruption.
- **Profile Normalization**: Case-insensitive profile names (e.g., `-p gaming` works).
- **Unit Tests**: Added tests for bandwidth calculation, target scanning, and profile normalization.

## [1.0.0] - 2026-03-12

### Added
- **Core Monitoring Engine**: Continuous network health tracking using ICMP pings.
- **Metrics Calculation**: Real-time measurement of Latency (RTT), Jitter (StdDev), and Packet Loss.
- **Activity Profiles**:
  - `Gaming`: Low-latency optimization (Threshold: 100ms latency, 30ms jitter).
  - `Meeting`: Stability for video calls (Threshold: 250ms latency, 5.0% loss).
  - `Browsing`: General purpose monitoring.
- **Automated Recovery Actions**:
  - `Flush DNS Cache`: Platform-specific DNS clearing (Windows, macOS, Linux).
  - `Soft Reset`: Network interface reset/renew (Release/Renew on Windows, Wi-Fi toggle on macOS, NetworkManager restart on Linux).
- **CLI Interface**:
  - Command-line flags for profiles (`--profile`, `-p`) and target IP (`--target`, `-t`).
  - Real-time logging of network statistics and recovery actions.
- **Cross-Platform Support**: Full compatibility with Windows, macOS, and Linux.
- **Documentation**: Comprehensive `README.md` and GoDoc comments for all exported symbols.
- **Testing**: Initial suite of unit tests in `main_test.go`.
- **Cooldown Logic**: 2-minute "rest period" after hard resets to prevent infinite restart loops.
