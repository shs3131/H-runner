# Frequently Asked Questions (FAQ)

---

### What is Hrunner?
Hrunner is a Windows runtime and dependency management platform for Python desktop applications. It separates application code from the Python interpreter and shared libraries, allowing applications to be distributed as tiny native executables (~4.7 MB) while sharing centrally managed runtimes and packages.

---

### Is Hrunner a replacement for PyInstaller?
Yes, for applications targeting Windows. Unlike PyInstaller which bundles a copy of the Python interpreter, standard library, and wheels into every single application, Hrunner manages runtimes and packages centrally.

---

### Does Hrunner bundle Python into each executable?
**No.** The generated executable contains only the native Go launcher, the application manifest, and the application's `.py` source files. It contains **no** Python interpreter and **no** wheels.

---

### Why are generated EXEs so small?
Because they do not bundle the 50–150 MB Python interpreter or third-party packages. The Go launcher stub compiled with `-ldflags="-s -w"` is approximately 4.7 MB.

---

### Does Hrunner require internet access?
Internet access is required only on first launch if the required Python runtime or packages have not yet been downloaded to the user's PC. Once installed into `%LOCALAPPDATA%\Hrunner`, applications run entirely offline.

---

### Where are packages stored?
In `%LOCALAPPDATA%\Hrunner\packages\<package_name>\<version>\`.

---

### Can two applications use different versions of NumPy?
**Yes.** Different versions (such as `numpy 1.26.4` and `numpy 2.3.2`) coexist side-by-side in separate directories in the package pool. Hrunner isolates `sys.path` so each application loads only its declared version.

---

### Are packages duplicated across applications?
**No.** If Application A and Application B both require `requests == 2.32.5`, only **one** physical copy exists in the package pool with a reference count of 2.

---

### Does Hrunner run permanently in the background?
**No.** Hrunner is an on-demand process. When an application starts, it wakes Hrunner if not already running. When no applications or management windows are active, Hrunner automatically shuts down after a 5-second idle timeout.

---

### Does Hrunner use localhost or open TCP ports?
**No.** Hrunner uses Windows Named Pipes (`\\.\pipe\hrunner`) exclusively for local IPC. It does not open local HTTP servers or TCP ports.

---

### Does Hrunner install a Windows service or startup item?
**No.** Hrunner does not install any Windows service, startup registry item, or system tray daemon.

---

### What happens if Hrunner is not installed on the user's PC?
The application launcher displays a native Windows dialog informing the user that Hrunner is required, with an **[ Install Hrunner ]** button that directs them to the official installer.

---

### What happens if a download or installation is interrupted?
Hrunner uses atomic operations: files are downloaded to temporary staging paths and verified with SHA-256 before being committed via atomic rename. Partially downloaded or corrupted files are detected and discarded.

---

### Can native Python packages (C-extensions) work?
**Yes.** Hrunner selects official Windows x64 binary wheels containing compiled `.pyd` and `.dll` files (e.g. `numpy`, `pillow`, `cryptography`, `opencv-python`) and registers DLL directories via `os.add_dll_directory()`.

---

### What Python versions are supported?
Official Python 3.10+ Windows embeddable releases from `python.org` (e.g. 3.11, 3.12, 3.13).

---

### Is Hrunner Windows-only?
Yes. The MVP targets Windows x64.

---

### Is Hrunner open source, and what license does it use?
Hrunner is free and open-source software licensed under the **GNU General Public License v3.0 (GPL-3.0)**.

---

### How does Hrunner compare to PyInstaller?
* **PyInstaller**: Duplicates Python runtime and dependencies into every `.exe` (50–200 MB per app). Slow build times (30–120s). Updating dependencies requires rebuilding and redistributing the entire monolithic binary.
* **Hrunner**: Executable is ~4.7 MB. Build times are under 10 milliseconds. Dependencies are shared centrally and deduplicated across applications.

---

### How does Hrunner compare to PyApp?
* **PyApp**: Embeds a dedicated Python environment per application, but does not share packages across different applications.
* **Hrunner**: Provides a centralized package pool with physical deduplication, reference counting, and an integrated native Windows management UI.

---

### How does Hrunner compare to virtual environments?
Virtual environments (`venv`) require developer familiarity with command-line tools, Python paths, and pip. Hrunner provides an automated, installer-like consumer desktop experience with zero Python knowledge required from end users.
