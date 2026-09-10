# Hello World Example

This example demonstrates the simplest possible application packaged with Hrunner.

## Project Structure

```text
examples/hello-world/
├── main.py       # Application entry point
└── README.md     # This walkthrough
```

## Building the Executable

From the root of the repository or with `hbuild` in your PATH:

```powershell
hbuild build ./examples/hello-world --output ./dist/HelloWorld.exe --name "Hello World" --version "1.0.0" --python "3.13.7"
```

## What Happens During Build

1. `hbuild` reads `main.py` and creates a versioned `manifest.json`.
2. Packages `main.py` and the manifest into an internal zip payload.
3. Binds the payload to the native Go launcher stub (`hlauncher.exe`).
4. Produces `HelloWorld.exe` (~4.7 MB).

## Running the Executable

```powershell
.\dist\HelloWorld.exe
```

When launched on an end user's machine:
1. `HelloWorld.exe` connects to Hrunner via `\\.\pipe\hrunner` (starting `hrunner.exe --pipe-server` if not already running).
2. Hrunner verifies that Python 3.13.7 is available (downloading and installing it if missing).
3. The application executes cleanly in an isolated runtime environment.
