# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

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
