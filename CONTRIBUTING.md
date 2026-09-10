# Contributing to Hrunner

We welcome contributions from the community! This document provides instructions for developing, testing, and submitting pull requests.

---

## 1. Prerequisites

* Windows 10 or Windows 11 (x64)
* Go 1.22 or newer
* Git for Windows
* *(Optional)* NSIS 3.x (to build `HrunnerSetup.exe`)

---

## 2. Getting Started

1. Fork the repository on GitHub: `https://github.com/shs3131/H-runner`.
2. Clone your fork locally:
   ```powershell
   git clone https://github.com/<your-username>/H-runner.git
   cd H-runner
   ```
3. Create a descriptive feature branch:
   ```powershell
   git checkout -b feature/my-feature-name
   ```

---

## 3. Development & Testing

Compile all components:
```powershell
go build -ldflags="-s -w" -o hrunner.exe ./cmd/hrunner
go build -ldflags="-s -w" -o hlauncher.exe ./cmd/hlauncher
go build -ldflags="-s -w" -o hbuild.exe ./cmd/hbuild
go build -ldflags="-s -w" -o hmanager.exe ./cmd/hmanager
```

Run tests before committing:
```powershell
# Run all unit and integration tests
go test -v ./...
```

Format your Go code:
```powershell
gofmt -s -w .
```

---

## 4. Submitting Pull Requests

1. Commit your changes with clear, semantic commit messages (e.g. `fix: handle edge case in wheel parser`).
2. Push to your fork:
   ```powershell
   git push origin feature/my-feature-name
   ```
3. Open a Pull Request on the main repository targeting the `main` branch.
4. Ensure all CI checks pass.
