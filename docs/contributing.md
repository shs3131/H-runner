# Contributing to Hrunner

Thank you for your interest in contributing to Hrunner!

---

## Code of Conduct
Please read and follow our [Code of Conduct](../CODE_OF_CONDUCT.md) in all project spaces.

---

## Development Workflow

1. **Fork and Clone**:
   ```powershell
   git clone https://github.com/<your-username>/H-runner.git
   cd H-runner
   ```

2. **Create a Branch**:
   ```powershell
   git checkout -b feature/your-feature-name
   ```

3. **Coding Standards**:
   * Pure Go for Windows without CGO dependencies.
   * Format code with `gofmt -s -w .`.
   * Ensure tests exist for all new functionality.

4. **Verify Tests**:
   ```powershell
   go test -v ./...
   ```

5. **Submit a Pull Request**:
   * Clearly describe the purpose of the change.
   * Reference any relevant open issues.
