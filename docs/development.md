# Development Guide

This guide explains how to build, test, and contribute to Hrunner.

---

## 1. Project Layout

```text
H-runner/
├── cmd/
│   ├── hrunner/         # Main Hrunner service & CLI entrypoint
│   ├── hmanager/        # Direct Manager UI entrypoint
│   ├── hbuild/          # Developer Builder CLI
│   └── hlauncher/       # Lightweight native Go launcher stub
├── pkg/
│   ├── manifest/        # Manifest schema v1 parser and validator
│   ├── packages/        # PyPI client, wheel matching, package pool store
│   ├── protocol/        # Windows Named Pipe server/client, auto-shutdown
│   ├── registry/        # registry.json management and application states
│   ├── resolver/        # Dependency resolution and conflict detection
│   ├── runtime/         # Python runtime manager and execution bootstrap
│   ├── storage/         # Disk usage summary, reference counting, cleanup
│   └── ui/              # Native Windows TaskDialog and Manager interfaces
├── test/
│   └── integration/     # End-to-end multi-app integration tests
├── examples/            # Working example projects
└── packaging/           # NSIS installer and build scripts
```

---

## 2. Building Components

```powershell
# Build all components with release flags
go build -ldflags="-s -w" -o hrunner.exe ./cmd/hrunner
go build -ldflags="-s -w" -o hlauncher.exe ./cmd/hlauncher
go build -ldflags="-s -w" -o hbuild.exe ./cmd/hbuild
go build -ldflags="-s -w" -o hmanager.exe ./cmd/hmanager
```

---

## 3. Running Automated Tests

Run the complete test suite:
```powershell
go test -v ./...
```

Run specific package tests:
```powershell
go test -v ./pkg/manifest
go test -v ./pkg/protocol
go test -v ./pkg/packages
go test -v ./pkg/resolver
go test -v ./pkg/runtime
go test -v ./pkg/storage
go test -v ./pkg/builder
go test -v ./test/integration
```

---

## 4. Building the NSIS Installer

To compile `HrunnerSetup.exe`, run the distribution script:

```powershell
powershell -ExecutionPolicy Bypass -File packaging\build_dist.ps1
```
