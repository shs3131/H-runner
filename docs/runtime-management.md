# Python Runtime Management

Hrunner centrally manages Python runtimes on Windows machines to prevent applications from repeatedly bundling 50–100 MB interpreters.

---

## 1. Runtime Store Layout

Runtimes are maintained in `%LOCALAPPDATA%\Hrunner\runtimes`:

```text
%LOCALAPPDATA%\Hrunner\runtimes\
├── python-3.12.8\
│   ├── python.exe
│   ├── python3.dll
│   ├── python312.dll
│   ├── python312.zip
│   └── python312._pth
└── python-3.13.7\
```

Different versions of Python coexist side-by-side without interference.

---

## 2. Python Embeddable Empirical Verification

Hrunner uses official Windows x64 embeddable distributions from `python.org` (`python-<version>-embed-amd64.zip`).

### Critical Behavioral Findings
Through empirical verification on Windows:
1. **Isolated Mode by Default**: The presence of `python3XX._pth` causes Python to enable isolated mode (`Py_IsolatedFlag = 1`), completely ignoring `PYTHONPATH` and the system `PATH`.
2. **Site Packages Leaking Prevention**: If `#import site` is uncommented inside `._pth`, Python scans the user's roaming AppData (`%APPDATA%\Python\Python3XX\site-packages`), which breaks application environment isolation.
3. **The Solution**: Hrunner keeps the runtime distribution clean and unpolluted. When executing an application, Hrunner passes a custom Python bootstrap script that programmatically inserts only the application's declared package pool paths into `sys.path` and registers native DLL search paths via `os.add_dll_directory()`.

---

## 3. Runtime Acquisition & Integrity

1. **Source**: Official HTTPS downloads from `https://www.python.org/ftp/python/<version>/python-<version>-embed-amd64.zip`.
2. **Cache**: Stored in `%LOCALAPPDATA%\Hrunner\cache\`.
3. **Atomic Extraction**: Archives are unpacked into a temporary folder (`tmp_<version>_<timestamp>`). Once extraction is verified, the directory is atomically renamed to `python-<version>`.
4. **Crash Safety**: Partially downloaded or corrupted zip files are detected, removed, and safely redownloaded.
