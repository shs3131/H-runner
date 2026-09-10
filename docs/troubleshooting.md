# Troubleshooting Guide

Common issues and troubleshooting procedures for Hrunner.

---

## 1. "Hrunner is required" Dialog Appears

### Symptom:
When launching an application, a dialog states:
```text
Hrunner is required
This application requires Hrunner to run.
[ Install Hrunner ]    [ Cancel ]
```

### Cause:
The application launcher could not find `hrunner.exe` on your system.

### Solution:
1. Install Hrunner using `HrunnerSetup.exe`.
2. Ensure `HKCU\Software\Hrunner\InstallPath` exists in your registry.
3. If running portable, ensure `hrunner.exe` is either adjacent to the application or in your system `PATH`.

---

## 2. Named Pipe Unavailable / Timeout

### Symptom:
Launcher displays: `Timed out waiting for Hrunner IPC pipe`.

### Solution:
1. Test if Hrunner starts manually from PowerShell:
   ```powershell
   hrunner --pipe-server
   ```
2. Verify that your antivirus or security software does not block local Windows Named Pipes.
3. Check `%LOCALAPPDATA%\Hrunner` permissions.

---

## 3. Dependency Conflicts

### Symptom:
Hrunner installer reports:
```text
dependency conflict: package A requires X==1.0, but X is already pinned to 2.0
```

### Cause:
Two packages in the dependency tree declare conflicting strict version pins.

### Solution:
1. Pin compatible dependency versions in your application's `requirements.txt`.
2. Inspect package requirements using `pip show <package>`.

---

## 4. Native Extension (`.pyd`) Loading Failure

### Symptom:
Application crashes with `ImportError: DLL load failed while importing ...`.

### Solution:
1. On Windows with Python 3.8+, Python requires DLL search directories to be registered via `os.add_dll_directory()`. Hrunner automatically calls `os.add_dll_directory()` for each package folder in the pool.
2. Ensure you have the Microsoft Visual C++ Redistributable installed on Windows if your native library requires the MSVC runtime.

---

## 5. Cleaning Stale Cache or Unused Packages

Use the CLI to clean unused packages and verify storage:

```powershell
# Inspect storage breakdown
hrunner storage

# List packages and reference counts
hrunner packages

# Clean orphaned packages
hrunner clean
```
