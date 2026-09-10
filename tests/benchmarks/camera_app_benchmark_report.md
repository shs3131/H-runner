# Hrunner vs PyInstaller — Benchmark & Implementation Report: CameraApp

**Date:** 2026-09-10  
**Application:** Hrunner Vision Camera Application (`examples/camera-app`)  
**Dependencies:** `opencv-python==5.0.0.93`, `pillow==12.3.0`, `numpy==2.5.3` (heavy native C++ compiled extensions, BLAS, SIMD, and Windows DLLs)  
**Platform:** Windows 11 x64 (Build 26200)

---

## 1. Executive Summary & Size Comparison

| Metric | Hrunner (`hbuild`) | PyInstaller (`--onefile`) | Difference |
| :--- | :--- | :--- | :--- |
| **Output Executable Size** | **4.6 MB** (4,816,887 B) | **63.9 MB** (66,998,587 B) | **Hrunner is 92.8% smaller** |
| **Complete NSIS Setup Installer** | **2.01 MB** (2,110,479 B) | *None* (Requires manual packaging) | **Built-in via `hbuild -installer`** |
| **Build Duration** | **62 ms** (standalone) / **830 ms** (with NSIS) | **30,253 ms** (~30.3 s) | **Hrunner is 36x – 480x faster** |
| **Dependency Bundling** | **Zero bundled wheels/Python** | All C++ DLLs & Python interpreter bundled | Complete separation of concerns |
| **Package Sharing** | **1 copy in central pool** | Duplicated in every `.exe` | Eliminates redundant disk bloat |
| **5-App Footprint (OpenCV+PIL+NumPy)** | **191.9 MB** total | **335.0 MB** total | Scales with O(1) for shared libraries |

---

## 2. Real-World Measured Metrics

### Executable Files in `dist/`
```text
dist/
├── CameraApp_Hrunner.exe          4,816,887 bytes    (4.6 MB)
├── CameraApp_HrunnerSetup.exe     2,110,479 bytes    (2.01 MB)  [NSIS Setup Installer]
└── CameraApp_PyInstaller.exe     66,998,587 bytes   (63.9 MB)
```

### Central Package Pool (`%LOCALAPPDATA%\Hrunner\packages`)
Packages were resolved and downloaded directly from official PyPI into the central deduplicated store:
```text
%LOCALAPPDATA%\Hrunner\packages\
├── numpy\2.5.3\               41.9 MB   [SHARED by 1 app]
├── opencv-python\5.0.0.93\   112.6 MB   [SHARED by 1 app]
└── pillow\12.3.0\             14.4 MB   [SHARED by 1 app]
Total Shared Pool:            168.9 MB
```

If you package 10 computer vision applications with PyInstaller, the total disk usage is **670 MB**.  
With Hrunner, 10 applications use only `(10 * 4.6 MB) + 168.9 MB = 214.9 MB` (**68% disk space saved** across the machine).

---

## 3. Application Installation System (NSIS Integration)

### Does Hrunner provide an application installer screen like NSIS?
**YES! Hrunner provides two complementary installation experiences:**

### A. Dynamic First-Launch Native Installer (`ConfirmInstallation`)
When an end-user runs a standalone `CameraApp_Hrunner.exe` for the first time:
1. The app connects to Hrunner via Windows Named Pipe (`\\.\pipe\hrunner`).
2. If dependencies (`Python`, `opencv-python`, `pillow`, `numpy`) are not present, Hrunner opens the native Windows Aero/Fluent TaskDialog:
   * Lists missing packages and their download sizes.
   * Displays total download footprint (`53.9 MB`).
   * Provides **[Install & Run]** and **[Cancel]** command links.
3. Automatically downloads wheels with progress reporting and launches the app upon completion.

### B. Generated Windows NSIS Setup Installer (`hbuild -installer`)
Hrunner Builder now includes native NSIS installer generation:
```powershell
hbuild build --installer examples/camera-app
```
* Generates `<AppName>Setup.exe` (e.g. `CameraApp_HrunnerSetup.exe`, **2.01 MB**).
* Standard Windows setup wizard (Welcome -> Install Directory -> Progress -> Finish).
* Installs executable to `%LOCALAPPDATA%\Programs\<AppName>`.
* Creates Windows Start Menu shortcuts and Desktop shortcut.
* Registers uninstaller in **Windows Settings / Installed Apps** and **Control Panel**.

---

## 4. Official Package & Distribution Sources

### Are packages and Python downloaded from official servers?
**YES, 100% from official authoritative infrastructure:**

1. **Python Embeddable Runtimes:**
   * **Source URL:** `https://www.python.org/ftp/python/{version}/python-{version}-embed-amd64.zip`
   * **Provider:** Python Software Foundation (PSF) official release servers.
   * **Security:** Clean official Windows x64 embeddable distributions, extracted atomically with checksum validation.

2. **Python Wheels & Packages:**
   * **API URL:** `https://pypi.org/pypi/{package}/json`
   * **Download URL:** `https://files.pythonhosted.org/packages/...`
   * **Provider:** Official Python Package Index (PyPI) Fastly Global CDN.
   * **Security:** Every wheel file is validated against the SHA-256 digest provided in PyPI's signed JSON metadata before extraction.

3. **CLI Direct Management:**
   Packages can be pre-installed or inspected directly via the CLI:
   ```powershell
   hrunner install opencv-python 5.0.0.93 --python 3.13.7
   hrunner install-requirements requirements.txt --python 3.13.7
   hrunner packages
   ```
