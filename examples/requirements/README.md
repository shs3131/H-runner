# Requirements Example

Demonstrates declaring and resolving pinned third-party dependencies from PyPI using standard `requirements.txt`.

## Project Structure

```text
examples/requirements/
├── main.py            # Imports and tests 'requests'
├── requirements.txt   # Declares requests == 2.32.5
└── README.md          # Guide
```

## Building the Application

```powershell
hbuild build ./examples/requirements --output ./dist/RequestsApp.exe --name "Requests App" --version "1.0.0" --python "3.13.7"
```

## How Dependency Resolution Works

1. `hbuild` automatically parses `requirements.txt` and pins `requests == 2.32.5` into the application manifest.
2. The generated `RequestsApp.exe` contains only the launcher, manifest, and `main.py`. The `requests` wheel is **not** bundled into the executable.
3. On first launch, Hrunner queries PyPI, selects the compatible Windows wheel (`requests-2.32.5-py3-none-any.whl`), verifies its SHA-256 digest, extracts it into `%LOCALAPPDATA%\Hrunner\packages\requests\2.32.5\`, and executes the app with this package injected into `sys.path`.
4. Subsequent launches reuse the cached package immediately without network queries.
