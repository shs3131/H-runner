# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.2] - 2026-09-10

### Added
* **Windows 11 Fluent Design / WinUI 3 Manager UI (`hmanager.exe`)**:
  * Redesigned Hrunner Manager from scratch as a modern Windows 11 desktop application styled after Windows 11 Settings and Lossless Scaling.
  * 100% native Go Windows desktop app: zero WebView, zero Electron, zero browser, zero localhost HTTP servers, zero Node.js.
  * Native Windows 11 DWM Mica backdrop integration (`DWMWA_SYSTEMBACKDROP_TYPE = DWMSBT_MAINWINDOW`) and rounded window geometry (`DWMWA_WINDOW_CORNER_PREFERENCE = DWMWCP_ROUND`).
  * Custom GDI+ double-buffered anti-aliased rendering engine (`pkg/ui/fluent_render.go`) for 60 FPS flicker-free resizing, smooth scrolling, and hover transitions.
  * Segoe UI Variable typography with subpixel ClearType rendering and dynamic DPI scaling.
  * Custom resolution-independent Fluent vector icons (Home, Applications, Chip, Cube, Storage Layers, Settings Gear, Play, Trash, Refresh, Folder).
  * Modern sidebar navigation with active indicator bars, hover highlights, and real-time service status.
  * 6 dedicated views:
    * **Home Dashboard**: Metric stat cards, Quick Actions bar, and registered application list.
    * **Applications**: Application cards with Python version badges, entrypoint details, direct Launch action, and unregister flow with dependency impact analysis.
    * **Python Runtimes**: Central embeddable distribution viewer with disk paths and installation status.
    * **Shared Package Pool**: Deduplicated package viewer with application reference counts and disk sizes.
    * **Storage & Cache**: Interactive segmented disk usage visualizer (Runtimes, Packages, Applications, Cache) with one-click cleanup for orphaned dependencies.
    * **Settings & Diagnostics**: System paths, Named Pipe IPC health, DWM backdrop status, and runtime diagnostics.
  * Per-monitor High-DPI v2 awareness.
  * Included `hmanager.exe` in NSIS installer (`dist/HrunnerSetup.exe`), Start Menu shortcuts, and App Paths registry.

## [1.0.1] - 2026-09-10

### Fixed
* **Native Manager (`hmanager.exe`)**:
  * Fixed crash/immediate exit on startup caused by `TaskDialogIndirect` procedure lookup panic in legacy `comctl32.dll`.
  * Implemented dynamic activation context (`ACTCTX`) for `Microsoft.Windows.Common-Controls` version 6.0.0.0.
  * Added safe procedure address checking (`proc.Find()`) to prevent any unhandled Win32 DLL panics.
  * Implemented genuine Win32 window creation and message loop (`CreateWindowExW`, `GetMessageW`, `DispatchMessageW`) keeping the Manager window persistently open until closed by user.
  * Added interactive tabs and real-time views for Applications, Shared Package Pool, Python Runtimes, and Storage Breakdown.
* **Application Launcher (`hlauncher.exe` / `SampleApp.exe`)**:
  * Fixed black console hang caused by uninitialized `BaseMessage` in `LaunchRequest` resulting in protocol version validation failure on Hrunner Named Pipe server.
  * Added error checking on `MsgLaunchResponse` so error states immediately report `ErrorMessage` to stderr and exit instead of blocking indefinitely on pipe reads.
  * Added CLI argument forwarding (`os.Args[1:]`) and initialized `sys.argv = [entrypoint] + sys.argv[1:]` in the runtime bootstrap script.
* **Regression Testing**:
  * Added `TestRegressionManagerComctl32V6` in `pkg/ui` verifying Comctl32 v6 activation and no-panic TaskDialog execution.
  * Added `TestRegressionLauncherRequestNoHang` in `test/integration` verifying launcher request framing and no-hang IPC flow.

## [1.0.0] - 2026-09-10


### Added
* **Hrunner Main App (`hrunner`)**:
  * Windows Named Pipe IPC server (`\\.\pipe\hrunner`) with auto-shutdown on idle.
  * Central Python runtime store supporting embeddable Python distributions in `%LOCALAPPDATA%\Hrunner\runtimes`.
  * Shared package pool in `%LOCALAPPDATA%\Hrunner\packages` with physical deduplication and reference counting.
  * Multi-version package coexistence (e.g. `numpy 1.26.4` and `numpy 2.3.2`).
  * Logical environment isolation using custom runtime bootstrap scripts and `os.add_dll_directory()`.
  * Application registry (`registry.json`) tracking lifecycle states.
  * Storage accounting and orphaned package cleanup.
  * 100% native Windows TaskDialog installer flow with download size calculation and progress reporting.
  * Native Windows Manager GUI for inspecting and launching applications, runtimes, package pool, and storage.
* **Hrunner Builder (`hbuild`)**:
  * Automated project analyzer and `requirements.txt` parser.
  * Manifest generation with schema `format_version: 1`.
  * Fast PE overlay ZIP binding producing standalone executables in under 10 milliseconds.
* **Hrunner Launcher (`hlauncher`)**:
  * Lightweight native Go executable (~4.7 MB) with zero bundled Python or wheels.
  * Windows Registry and standard path discovery for Hrunner.
  * Named Pipe client with isolated execution proxying.
* **Packaging**:
  * Official NSIS installer script producing `dist/HrunnerSetup.exe` (~6.9 MB).
  * Automated build script `packaging/build_dist.ps1`.
* **Test Suite & Examples**:
  * Complete unit tests across all packages.
  * End-to-end integration test verifying multi-app build, shared package deduplication, logical isolation, safe removal, and on-demand IPC lifecycle.
  * Working examples: `hello-world`, `requirements`, and `multi-app`.
