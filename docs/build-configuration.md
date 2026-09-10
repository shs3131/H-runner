# Builder Configuration & CLI Parameters

This document provides the definitive reference for all configuration options and command-line flags supported by `hbuild`.

---

## Command Syntax

```powershell
hbuild build <project_path> [flags]
```

* `project_path`: Path to the directory containing your Python application (or direct path to the main `.py` script). Defaults to current directory `.` if omitted.

---

## Parameter Reference

| Flag | Type | Default | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `--output`, `-o` | `string` | `<Name>.exe` in current directory | Absolute or relative path for the generated executable. | `--output "dist/MyApp.exe"` |
| `--name` | `string` | Directory name | Human-readable display name of the application. | `--name "Data Viewer"` |
| `--version` | `string` | `1.0.0` | Semantic version string of the application (`X.Y.Z`). | `--version "1.2.0"` |
| `--app-id` | `string` | `com.hrunner.<name>` | Unique reverse-DNS identifier for registry tracking. | `--app-id "com.example.viewer"` |
| `--python` | `string` | `3.13.7` | Python runtime version required by this application. | `--python "3.12.8"` |
| `--allow-compatible` | `bool` | `false` | If set, permits existing installed compatible minor versions (e.g. 3.13.8 for 3.13.7). | `--allow-compatible` |
| `--entrypoint` | `string` | Auto-detected (`main.py`, `app.py`) | Main Python script to execute when launched. | `--entrypoint "cli.py"` |
| `--publisher` | `string` | `""` | Publisher or company name displayed in Hrunner Manager. | `--publisher "Acme Corp"` |
| `--launcher-stub` | `string` | `hlauncher.exe` in PATH or adjacent | Custom path to the prebuilt Go launcher stub executable. | `--launcher-stub "bin/hlauncher.exe"` |

---

## Detailed Parameter Descriptions

### `--output, -o`
* **When to use**: When you want to specify a custom file name or write the output into a specific distribution folder (e.g. `dist/`).
* **Effect**: Determines where the final executable is placed on disk.

### `--name`
* **When to use**: Always recommended to give your application a clear user-facing title.
* **Effect**: Used in installer confirmation dialogs, the completion window, and in the Hrunner Manager application list.

### `--version`
* **When to use**: For version tracking, updates, and releases.
* **Effect**: Embedded in `manifest.json` and registered in `%LOCALAPPDATA%\Hrunner\registry.json`.

### `--app-id`
* **When to use**: To uniquely identify your application across multiple versions or installations.
* **Effect**: Serves as the primary key in the Hrunner registry and determines isolated cache directories.

### `--python`
* **When to use**: To specify the exact Python version required by your application code or C extensions.
* **Effect**: Hrunner will check for this specific version in `%LOCALAPPDATA%\Hrunner\runtimes` and download it from `python.org` if missing.

### `--allow-compatible`
* **When to use**: When your code does not depend on a specific micro patch release and can run on any compatible minor release.
* **Effect**: If requested `3.13.7` and `3.13.8` is already installed on the user's PC, Hrunner will reuse `3.13.8` rather than downloading `3.13.7`.

### `--entrypoint`
* **When to use**: If your main script is not named `main.py` or `app.py`.
* **Effect**: The generated bootstrap loader will execute this script via `runpy.run_path()`.

### `--requirements.txt` Automatic Parsing
If a `requirements.txt` file exists in the project root, `hbuild` automatically parses lines matching `package == version` and embeds them into `manifest.json`. Unpinned dependencies should be pinned before building to ensure reproducible environments.
