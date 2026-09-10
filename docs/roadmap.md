# Hrunner Roadmap

This document outlines completed milestones and planned future capabilities for Hrunner.

---

## 1. Completed (MVP Release)

* [x] **Lightweight Native Windows Launcher**: Compact Go executable (~4.7 MB) with PE overlay payload reader.
* [x] **Hrunner Main App**: Central engine managing runtimes, package pool, and application registry.
* [x] **Official NSIS Installer**: Packages `HrunnerSetup.exe` (~6.9 MB) with Windows Registry integration and PATH setup.
* [x] **Local Named Pipe IPC**: Secure byte-mode pipe `\\.\pipe\hrunner` with automatic start on demand and clean idle auto-shutdown.
* [x] **Shared Package Pool with Physical Deduplication**: Packages stored once in `%LOCALAPPDATA%\Hrunner\packages\<name>\<version>\` with reference counting across applications.
* [x] **Logical Environment Isolation**: Applications run with isolated `sys.path` and DLL directories without version bleed.
* [x] **Multi-Version Coexistence**: Different versions of the same package (e.g. `numpy 1.26.4` and `numpy 2.3.2`) coexist in the pool.
* [x] **Python Runtime Store**: Manages official embeddable Windows Python distributions in `%LOCALAPPDATA%\Hrunner\runtimes\`.
* [x] **Pragmatic Dependency Resolver**: Resolves direct and transitive requirements from wheel metadata and PyPI.
* [x] **100% Native Windows UI**: TaskDialog installer flow (runtime choice, dependency confirmation, progress bar, completion dialog) and native Windows Manager GUI.
* [x] **Hrunner Builder (`hbuild`)**: Analyzes Python projects, extracts requirements, and generates native standalone executables in under 10 milliseconds.
* [x] **Storage Accounting & Cleanup**: Calculates disk consumption and safely removes orphaned packages.
* [x] **Comprehensive Test Suite**: Automated unit tests and end-to-end multi-app integration tests.

---

## 2. Planned (Future Milestones)

The following features are **planned** and under active consideration:

* [ ] **Automatic Application Updates**: Transparently checking for and applying newer versions of registered application payloads.
* [ ] **Custom & Private Package Indexes**: Supporting custom PyPI-compatible registry URLs or internal index mirrors.
* [ ] **Cryptographic Application Signing**: Verifying digital signatures on application manifests and payloads before execution.
* [ ] **Extended Python ABI Tags**: Broader support for specialized compiler and CPython builds.
* [ ] **Richer Native Win32 Manager Interface**: Advanced filtering, search, and detailed package dependency tree visualization.
* [ ] **Automated GitHub Release CI Tooling**: Automated cross-compilation and signing of release binaries on tag pushes.
