# Hrunner MVP — Walkthrough & Verification Summary

Hrunner has been implemented as a Windows-first (x64) platform for distributing Python desktop applications as lightweight native executables with centralized runtime and shared dependency management.

---

## Architecture Overview

Hrunner consists of two separate open-source components:

### 1. Hrunner Main App (`cmd/hrunner`)
* **Distribution & Installation**: Packaged as `dist/HrunnerSetup.exe` via **NSIS** to `%LOCALAPPDATA%\Programs\Hrunner`. Writes Windows Registry keys (`HKCU\Software\Hrunner\InstallPath`, App Paths, and PATH).
* **On-Demand IPC Server**: Exposes local Windows Named Pipe `\\.\pipe\hrunner`. It is **NOT** a persistent background service or startup daemon. When no managed applications are active and no management window is open, it terminates automatically after an idle timeout.
* **Python Runtime Store**: Manages embeddable Windows Python distributions in `%LOCALAPPDATA%\Hrunner\runtimes\python-<version>\`.
* **Shared Package Pool**: Deduplicates packages physically in `%LOCALAPPDATA%\Hrunner\packages\<name>\<version>\`. Multiple applications pointing to the exact same package version share a single immutable copy on disk.
* **Application Registry**: Tracks installed applications in `%LOCALAPPDATA%\Hrunner\registry.json` through formal lifecycle states (`discovered`, `installing`, `installed`, `ready`, `broken`, `removing`).
* **100% Native Windows UI**: Built with pure Go (`golang.org/x/sys/windows`) and Windows `comctl32.dll` TaskDialogs and Common Controls. **No localhost HTTP server, no web sockets, and no browser tabs.**

### 2. Hrunner Builder (`hbuild`) & Generated Launcher (`hlauncher`)
* **Builder CLI (`cmd/hbuild`)**: Analyzes Python projects, extracts requirements (`requirements.txt`), generates versioned manifests (`format_version: 1`), packages Python source files into a ZIP payload, and binds them to the native Go launcher stub in **under 10 milliseconds**.
* **Generated Launcher (`cmd/hlauncher`)**:
  * Native Windows x64 executable (**~4.7 MB**).
  * Contains **zero** Python runtime, **zero** third-party dependencies, and **zero** large DLLs.
  * Discovers installed Hrunner via Registry (`HKCU\Software\Hrunner\InstallPath`), App Paths, and standard installation directories.
  * Starts `hrunner.exe --pipe-server` on demand if not running.
  * Communicates with Hrunner strictly over Named Pipe `\\.\pipe\hrunner`.
  * Spawns the isolated Python process with application-specific package paths injected into `sys.path`.

---

## Executable Sizes Measured

| Component | Executable | Size on Disk | Description |
| :--- | :--- | :--- | :--- |
| **Hrunner Main App** | `hrunner.exe` | **7.2 MB** | Named Pipe server, runtime & package managers, native Win32 UI |
| **Launcher Stub** | `hlauncher.exe` | **4.7 MB** | Native launcher bootstrapper with PE overlay zip reader |
| **Builder CLI** | `hbuild.exe` | **4.5 MB** | Project analyzer, manifest generator, and packager |
| **Generated App** | `SampleApp.exe` | **4.8 MB** | Standalone native executable (launcher + manifest + code) |
| **NSIS Installer** | `HrunnerSetup.exe` | **6.9 MB** | Complete NSIS setup containing all Hrunner binaries |

*(For comparison: standard PyInstaller executables for basic apps typically exceed 50–150 MB).*

---

## Automated Test Suite Results

All unit and integration tests pass cleanly:

```text
=== RUN   TestValidManifest
--- PASS: TestValidManifest (0.00s)
=== RUN   TestInvalidManifests
--- PASS: TestInvalidManifests (0.00s)
=== RUN   TestParseFile
--- PASS: TestParseFile (0.01s)
PASS: github.com/hrunner/hrunner/pkg/manifest

=== RUN   TestWheelParsingAndCompatibility
--- PASS: TestWheelParsingAndCompatibility (0.00s)
=== RUN   TestPackageStoreDeduplicationAndMultipleVersions
--- PASS: TestPackageStoreDeduplicationAndMultipleVersions (0.02s)
PASS: github.com/hrunner/hrunner/pkg/packages

=== RUN   TestNamedPipeRoundTrip
--- PASS: TestNamedPipeRoundTrip (0.02s)
=== RUN   TestAutoShutdownOnIdle
--- PASS: TestAutoShutdownOnIdle (0.31s)
PASS: github.com/hrunner/hrunner/pkg/protocol

=== RUN   TestRegistryCRUD
--- PASS: TestRegistryCRUD (0.04s)
=== RUN   TestRegistryStateTransitions
--- PASS: TestRegistryStateTransitions (0.01s)
PASS: github.com/hrunner/hrunner/pkg/registry

=== RUN   TestParseRequiresDist
--- PASS: TestParseRequiresDist (0.00s)
=== RUN   TestResolverLocalAndConflict
--- PASS: TestResolverLocalAndConflict (16.53s)
PASS: github.com/hrunner/hrunner/pkg/resolver

=== RUN   TestParseVersion
--- PASS: TestParseVersion (0.00s)
=== RUN   TestVersionCompatibility
--- PASS: TestVersionCompatibility (0.00s)
=== RUN   TestRuntimeStoreDiscovery
--- PASS: TestRuntimeStoreDiscovery (0.01s)
=== RUN   TestBuildBootstrapScript
--- PASS: TestBuildBootstrapScript (0.00s)
PASS: github.com/hrunner/hrunner/pkg/runtime

=== RUN   TestStorageReferenceCountingAndCleanup
--- PASS: TestStorageReferenceCountingAndCleanup (0.13s)
PASS: github.com/hrunner/hrunner/pkg/storage

=== RUN   TestFormatBytes
--- PASS: TestFormatBytes (0.00s)
=== RUN   TestNativeManagerHeadless
--- PASS: TestNativeManagerHeadless (0.01s)
PASS: github.com/hrunner/hrunner/pkg/ui

=== RUN   TestParseRequirementsTxt
--- PASS: TestParseRequirementsTxt (0.01s)
=== RUN   TestBuilderBuild
--- PASS: TestBuilderBuild (0.02s)
PASS: github.com/hrunner/hrunner/pkg/builder

=== RUN   TestCompleteMVPIntegrationFlow
--- PASS: TestCompleteMVPIntegrationFlow (0.32s)
PASS: github.com/hrunner/hrunner/test/integration
```

---

## Integration Test Highlights

`TestCompleteMVPIntegrationFlow` verified:
1. **Multi-App Build**: Built App A (requires `requests==2.32.5`, `numpy==1.26.4`) and App B (requires `requests==2.32.5`, `numpy==2.3.2`).
2. **Physical Package Deduplication**: `requests==2.32.5` has a reference count of 2 across both applications, but exists in only **one physical directory** in the package pool.
3. **Multi-Version Coexistence**: `numpy==1.26.4` and `numpy==2.3.2` coexist side-by-side in the pool without interfering.
4. **Logical Environment Isolation**: App A only receives `numpy==1.26.4` in its environment; App B only receives `numpy==2.3.2`.
5. **Safe App Removal**: When App A was uninstalled, `requests==2.32.5` was preserved because App B still depended on it; orphaned `numpy==1.26.4` was cleaned. When App B was subsequently uninstalled, the remaining packages were cleaned.
6. **Auto-Shutdown IPC Server**: Proved the Named Pipe server automatically exits on idle after client disconnect.

---

## Verification Commands

To re-run the entire test suite:
```powershell
go test -v ./...
```

To build all distribution artifacts and the NSIS installer:
```powershell
powershell -ExecutionPolicy Bypass -File packaging\build_dist.ps1
```

To run the Hrunner Manager UI:
```powershell
.\hrunner.exe --manager
```

To build a Python application using `hbuild`:
```powershell
.\hbuild.exe build .\test\sample_app -o .\dist\MyApplication.exe --launcher-stub .\hlauncher.exe
```
