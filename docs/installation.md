# Installation Guide

This guide covers installing Hrunner on end user computers and setting up the development environment for developers.

---

## 1. End User Installation

End users do not need to install Python, configure virtual environments, or manage pip.

### Steps:
1. Download `HrunnerSetup.exe` from the official [Releases](https://github.com/shs3131/H-runner/releases) page.
2. Run the installer:
   * Installs to `%LOCALAPPDATA%\Programs\Hrunner` without requiring Administrator privileges.
   * Registers `hrunner.exe` in Windows Registry (`HKCU\Software\Hrunner`).
   * Adds Hrunner to the user's `PATH`.
   * Creates a Start Menu shortcut: **Hrunner Manager**.
3. Any application built with Hrunner will now detect Hrunner automatically and launch instantly.

---

## 2. Developer Installation (Building from Source)

### Prerequisites
* **Windows 10 / 11 (x64)**
* **Go 1.22+** (tested on Go 1.26.4)
* **Git for Windows**
* *(Optional)* **NSIS 3.x** (to compile `HrunnerSetup.exe`)

### 1. Clone the Repository
```powershell
git clone https://github.com/shs3131/H-runner.git
cd H-runner
```

### 2. Build Core Binaries
Compile the main components using pure Go (zero CGO or GCC required):

```powershell
# Build Hrunner Main App
go build -ldflags="-s -w" -o hrunner.exe ./cmd/hrunner

# Build Hrunner Launcher Stub
go build -ldflags="-s -w" -o hlauncher.exe ./cmd/hlauncher

# Build Hrunner Builder CLI
go build -ldflags="-s -w" -o hbuild.exe ./cmd/hbuild

# Build Hrunner Manager shortcut executable
go build -ldflags="-s -w" -o hmanager.exe ./cmd/hmanager
```

### 3. Build the NSIS Installer
If you have [NSIS](https://nsis.sourceforge.io/) installed:

```powershell
powershell -ExecutionPolicy Bypass -File packaging\build_dist.ps1
```

The output installer will be located at:
```text
dist\HrunnerSetup.exe (~6.9 MB)
```

### 4. Run the Test Suite
Ensure all tests pass on your machine:

```powershell
go test -v ./...
```
