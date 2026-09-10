# Hrunner

A lightweight Windows runtime and dependency manager for Python applications.

[![CI](https://github.com/shs3131/H-runner/actions/workflows/ci.yml/badge.svg)](https://github.com/shs3131/H-runner/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue.svg)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Windows%20x64-0078d7.svg)](https://microsoft.com/windows)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Issues](https://img.shields.io/github/issues/shs3131/H-runner.svg)](https://github.com/shs3131/H-runner/issues)
[![Release](https://img.shields.io/github/v/release/shs3131/H-runner?include_prereleases)](https://github.com/shs3131/H-runner/releases)

Hrunner separates the desktop application from its Python runtime and dependencies. A generated application contains a tiny native launcher, manifest, and application code, while Hrunner manages Python runtimes and shared dependencies on demand.

---

## Table of Contents

- [Overview](#overview)
- [Why Hrunner?](#why-hrunner)
- [Comparison](#comparison)
- [Architecture](#architecture)
- [How It Works](#how-it-works)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Build Your First Application](#build-your-first-application)
- [Builder Configuration](#builder-configuration)
- [Build Parameters](#build-parameters)
- [Runtime Management](#runtime-management)
- [Package Management](#package-management)
- [Application Isolation](#application-isolation)
- [Manifest Format](#manifest-format)
- [CLI Reference](#cli-reference)
- [Hrunner Manager](#hrunner-manager)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)
- [Architecture Details](#architecture-details)
- [Development](#development)
- [Building Hrunner](#building-hrunner)
- [Testing](#testing)
- [Contributing](#contributing)
- [Security](#security)
- [Roadmap](#roadmap)
- [License](#license)

---

## Overview

Traditional tools such as PyInstaller create large executables by bundling the full Python interpreter, standard library, and all installed packages into every standalone binary (typically 50–200 MB per application).

Hrunner takes a different approach:
* Applications are distributed as **tiny native executables (~4.7 MB)** containing only the application's source code and a manifest.
* The executable contains **no Python runtime, no standard library, and no third-party packages**.
* The **Hrunner Main App** is installed on the user's PC via an NSIS installer and centrally manages Python runtimes and packages.
* Applications communicate with Hrunner through **local Windows Named Pipes (`\\.\pipe\hrunner`)**.
* Packages are **physically deduplicated** in a central pool, yet **logically isolated** during execution.
* Hrunner is **never a persistent background daemon or service**—it starts on demand when an application launches and shuts down automatically when idle.

---

## Why Hrunner?

1. **Ultra-Small Binaries**: Output executables are ~4.7 MB instead of 100+ MB.
2. **Lightning-Fast Builds**: `hbuild` builds executables in under 10 milliseconds.
3. **Shared Package Pool**: If 5 applications use `requests 2.32.5`, it is downloaded and stored on disk only once.
4. **Multi-Version Coexistence**: App A can use `numpy 1.26.4` while App B uses `numpy 2.3.2` on the same machine without interference.
5. **Polished Consumer Experience**: First launch presents a clean native Windows dialog showing missing dependencies and download size, followed by automated installation.
6. **Zero Web / Localhost Ports**: 100% native Windows architecture with zero localhost HTTP servers, zero web sockets, and zero browser tabs.

---

## Comparison

| Feature | Hrunner | PyInstaller | PyApp |
| :--- | :--- | :--- | :--- |
| **Output Binary Size** | **~4.7 MB** | 50 MB – 200 MB | 25 MB – 60 MB |
| **Build Time** | **< 10 ms** | 30 s – 120 s | 5 s – 30 s |
| **Dependency Sharing** | **Central pool (deduplicated)** | None (duplicated per app) | None (isolated per app) |
| **Multi-App Storage** | **Shared & reference-counted** | Multiplied per application | Multiplied per application |
| **Daemon / Service** | **None (on-demand auto-exit)** | None | None |
| **Management UI** | **Native Win32 Manager** | None | None |
| **IPC Mechanism** | **Windows Named Pipes** | N/A | N/A |

---

## Architecture

Hrunner consists of two distinct components:

```text
┌────────────────────────────────────────────────────────┐
│               Hrunner Main App (End User)              │
│  Installed via NSIS to %LOCALAPPDATA%\Programs\Hrunner │
│                                                        │
│  ├── Windows Named Pipe Server (\\.\pipe\hrunner)      │
│  ├── Python Runtime Store (%LOCALAPPDATA%\Hrunner)     │
│  ├── Shared Package Pool (Physical Deduplication)      │
│  ├── Application Registry (registry.json)              │
│  ├── Native Windows TaskDialog Installer Flow          │
│  └── Native Win32 Manager GUI (hmanager)               │
└──────────────────────────▲─────────────────────────────┘
                           │ Windows Named Pipe
                           │ (\\.\pipe\hrunner)
┌──────────────────────────┴─────────────────────────────┐
│          Generated Application (hlauncher Stub)        │
│                                                        │
│  ├── Native Go Launcher Executable (~4.7 MB)          │
│  ├── Application Manifest (format_version: 1)          │
│  └── Embedded / Appended Application Python Code       │
└────────────────────────────────────────────────────────┘
```

1. **Hrunner Main App**: Installed on the user's PC via `HrunnerSetup.exe`. Manages the runtime store, shared package pool, registry, native Windows installer UI, and Named Pipe IPC server.
2. **Hrunner Builder (`hbuild`)**: Developer CLI that analyzes Python projects, parses `requirements.txt`, generates manifests, and packages code into lightweight native executables.

---

## How It Works

### Developer Machine
```text
Python Project
     │
     ▼
hbuild build ./myproject
     │
     ▼ (8 ms)
MyApplication.exe (~4.7 MB)
```

### End User Machine
```text
MyApplication.exe
     │
     ▼
Check \\.\pipe\hrunner
     │
     ├── Inactive ──> Start installed hrunner.exe --pipe-server
     └── Active ────> Connect to pipe
     │
     ▼
Send LaunchRequest (Manifest)
     │
     ▼
Hrunner checks Python runtime & packages
     │
     ├── Missing ──> Show native dialog -> Download & verify -> Install
     └── Ready   ──> Instant launch
     │
     ▼
Launch isolated Python process with tailored sys.path
     │
     ▼
Application exits ──> Pipe closes ──> Hrunner exits after 5s idle
```

---

## Installation

### For End Users
1. Download `HrunnerSetup.exe` from [GitHub Releases](https://github.com/shs3131/H-runner/releases).
2. Run `HrunnerSetup.exe`. It installs to `%LOCALAPPDATA%\Programs\Hrunner` and registers with the Windows shell.
3. Launch any application built with Hrunner.

### For Developers (Building from Source)
```powershell
# Clone the repository
git clone https://github.com/shs3131/H-runner.git
cd H-runner

# Compile the developer tools and core binaries
go build -ldflags="-s -w" -o hrunner.exe ./cmd/hrunner
go build -ldflags="-s -w" -o hlauncher.exe ./cmd/hlauncher
go build -ldflags="-s -w" -o hbuild.exe ./cmd/hbuild
go build -ldflags="-s -w" -o hmanager.exe ./cmd/hmanager
```

---

## Quick Start

### 1. Create a Python Project
Create a directory with `main.py` and `requirements.txt`:

`main.py`:
```python
import sys
import requests

print(f"Running Python {sys.version.split()[0]}")
print(f"Requests {requests.__version__} loaded successfully!")
```

`requirements.txt`:
```text
requests == 2.32.5
```

### 2. Build the Application
```powershell
hbuild build . --output MyApp.exe --name "My App" --version "1.0.0" --python "3.13.7"
```

### 3. Run the Executable
```powershell
.\MyApp.exe
```

On first launch, Hrunner prompts the user to download Python 3.13.7 and `requests 2.32.5`, installs them into the shared pool, and starts the application. On subsequent launches, the application starts immediately.

---

## Build Parameters

The `hbuild build` command supports the following parameters:

| Parameter | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `--output`, `-o` | `string` | `<Name>.exe` | Output executable path. |
| `--name` | `string` | Directory name | Application display name. |
| `--version` | `string` | `1.0.0` | Semantic version string (`X.Y.Z`). |
| `--app-id` | `string` | `com.hrunner.<name>` | Unique reverse-DNS application ID. |
| `--python` | `string` | `3.13.7` | Required Python runtime version. |
| `--allow-compatible` | `bool` | `false` | Allow existing installed compatible minor runtimes. |
| `--entrypoint` | `string` | Auto-detected | Main Python script (`main.py`, `app.py`). |
| `--publisher` | `string` | `""` | Publisher name. |
| `--launcher-stub` | `string` | `hlauncher.exe` | Path to custom Go launcher stub binary. |

For detailed usage, see [docs/build-configuration.md](docs/build-configuration.md).

---

## Runtime Management

Hrunner maintains a central runtime store under `%LOCALAPPDATA%\Hrunner\runtimes\`:
* Downloads official Windows x64 embeddable distributions from `python.org`.
* Verifies download integrity before extraction.
* Isolates `sys.path` so the runtime cannot leak into or read from `%APPDATA%\Python\site-packages`.
* Multiple Python versions (e.g. `python-3.12.8` and `python-3.13.7`) coexist seamlessly.

For details, see [docs/runtime-management.md](docs/runtime-management.md).

---

## Package Management

All third-party packages are installed into `%LOCALAPPDATA%\Hrunner\packages\<name>\<version>\`:

* **Physical Deduplication**: If multiple applications require `requests == 2.32.5`, it is physically stored only once.
* **Native C-Extensions Supported**: Automatically selects and extracts Windows x64 binary wheels containing compiled `.pyd` and `.dll` extensions (`numpy`, `pillow`, `cryptography`, `opencv-python`).
* **Checksum Verification**: Every wheel is checked against SHA-256 digests from PyPI.

For details, see [docs/package-management.md](docs/package-management.md).

---

## Application Isolation

Applications share packages physically on disk, but execute in total logical isolation:

* App A (`numpy == 1.26.4`) and App B (`numpy == 2.3.2`) run without version bleed.
* The launcher injects only the packages declared in the application's manifest into `sys.path`.
* Native DLL search paths are configured via `os.add_dll_directory()`.

For details, see [docs/isolation.md](docs/isolation.md).

---

## Manifest Format

Every executable embeds a versioned `manifest.json` (`format_version: 1`):

```json
{
  "format_version": 1,
  "application_id": "com.example.myapp",
  "name": "My Application",
  "version": "1.0.0",
  "publisher": "Example Corp",
  "python": {
    "version": "3.13.7",
    "allow_compatible": false
  },
  "dependencies": {
    "requests": "2.32.5",
    "pillow": "11.3.0"
  },
  "entrypoint": "main.py"
}
```

For details, see [docs/manifest.md](docs/manifest.md).

---

## CLI Reference

### `hrunner`
The main Hrunner service, management UI, and diagnostic CLI.

```powershell
hrunner apps                  # List all registered applications
hrunner runtimes              # List installed Python runtime distributions
hrunner packages              # List shared package pool with reference counts
hrunner clean                 # Safely delete orphaned packages with refcount == 0
hrunner storage               # Display disk space breakdown
hrunner remove <app_id>       # Unregister an application (--clean-unused to delete orphaned packages)
hrunner --manager             # Open the native Windows Manager GUI
hrunner --pipe-server         # Start on-demand Named Pipe IPC server
```

### `hmanager`
Direct shortcut executable to launch the native Hrunner Manager GUI.

```powershell
hmanager
```

### `hbuild`
Developer CLI for building application executables.

```powershell
hbuild build <project_path> [flags]
```

---

## Hrunner Manager

Hrunner includes a 100% native Windows GUI (built with Win32 Common Controls and TaskDialogs) accessible by running `hmanager` or `hrunner --manager`:

* **Applications**: View all registered applications, launch them, or uninstall them.
* **Package Pool**: Inspect package versions, see which applications use each package, and identify unused packages.
* **Remove Unused Packages**: Free disk space by cleaning orphaned packages with zero applications depending on them.
* **Storage**: View live disk consumption for runtimes, packages, applications, and cache.

---

## Troubleshooting

See [docs/troubleshooting.md](docs/troubleshooting.md) for solutions to common issues:
* Application reports "Hrunner is required"
* Named Pipe timeouts or connection issues
* Dependency conflict resolution
* Resolving `.pyd` native extension DLL errors

---

## FAQ

See [docs/faq.md](docs/faq.md) for answers to frequently asked questions about security, performance, licensing, and comparisons with PyInstaller.

---

## Development & Testing

Run all automated unit and integration tests:

```powershell
go test -v ./...
```

Run specific test packages:
```powershell
go test -v ./pkg/manifest       # Manifest parsing & validation
go test -v ./pkg/packages       # Wheel matching & store deduplication
go test -v ./pkg/protocol       # Named Pipe IPC & auto-shutdown
go test -v ./pkg/registry       # Registry state machine
go test -v ./pkg/resolver       # Dependency resolution & conflicts
go test -v ./pkg/runtime        # Python runtime manager
go test -v ./pkg/storage        # Storage metrics & cleanup
go test -v ./pkg/ui             # Native Windows UI
go test -v ./pkg/builder        # Project packaging
go test -v ./test/integration   # End-to-end multi-app integration flow
```

To compile release artifacts and the NSIS installer:
```powershell
powershell -ExecutionPolicy Bypass -File packaging\build_dist.ps1
```

---

## Contributing

We welcome contributions! Please review [CONTRIBUTING.md](CONTRIBUTING.md) and our [Code of Conduct](CODE_OF_CONDUCT.md) before submitting pull requests.

---

## Security

Please report security issues responsibly. See [SECURITY.md](SECURITY.md) for details on our reporting process and supported versions.

---

## Roadmap

See [docs/roadmap.md](docs/roadmap.md) for our feature roadmap and planned capabilities.

---

## License

Hrunner is licensed under the [GNU General Public License v3.0 (GPL-3.0)](LICENSE).
