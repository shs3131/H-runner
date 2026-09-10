# Hrunner Builder (`hbuild`)

`hbuild` is the developer CLI tool used to package standard Python projects into lightweight native Windows executables.

---

## 1. Quick Example

Given a Python project:
```text
my_project/
├── main.py
└── requirements.txt
```

Package it with a single command:
```powershell
hbuild build ./my_project --output ./dist/MyApplication.exe --name "My Application" --version "1.0.0" --python "3.13.7"
```

Output:
```text
Building Hrunner package for: ./my_project
✓ Successfully built MyApplication.exe in 8.1ms
✓ Final executable size: 4.6 MB (4800250 bytes)
  (Contains application code and manifest; zero bundled Python/wheels)
```

---

## 2. Builder Workflow

When you run `hbuild build`, it executes the following steps:

1. **Locates the Project**: Identifies entry point (`main.py`, `app.py`, or explicit `--entrypoint`).
2. **Parses Requirements**: Scans `requirements.txt` for pinned dependencies (e.g. `requests == 2.32.5`).
3. **Generates Manifest**: Creates a versioned `manifest.json` (`format_version: 1`) defining application ID, name, version, required Python version, dependencies, and entrypoint.
4. **Packages Source Files**: Packs `.py`, `.pyw`, `.json`, and `.txt` files into an in-memory ZIP archive, ignoring `.git`, `__pycache__`, and virtual environments.
5. **Appends Payload**: Takes the Go launcher stub (`hlauncher.exe`) and appends the ZIP archive as a PE overlay.
6. **Emits Standalone Executable**: Generates a self-contained `.exe` without needing a C compiler or re-compiling Go every time.
