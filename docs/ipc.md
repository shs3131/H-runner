# Windows Named Pipe IPC Protocol

All communication between generated application launchers and the Hrunner Main App occurs over a local Windows Named Pipe.

---

## 1. IPC Architecture

* **Endpoint**: `\\.\pipe\hrunner`
* **Transport**: Windows Named Pipe byte mode with local current-user security DACL.
* **Framing**: Newline-delimited JSON (`\n`).
* **Zero Networking**: No TCP ports, no HTTP servers, no localhost sockets.

---

## 2. Protocol Specification

All messages include a protocol envelope with `protocol_version: 1` and a `type` string.

### 2.1 `launch_request`
Sent by the application launcher to request execution.

```json
{
  "protocol_version": 1,
  "type": "launch_request",
  "application_id": "com.example.myapp",
  "name": "My Application",
  "version": "1.0.0",
  "python": "3.13.7",
  "allow_compatible_python": false,
  "dependencies": {
    "requests": "2.32.5",
    "pillow": "11.3.0"
  },
  "entrypoint": "main.py",
  "executable_path": "C:\\Program Files\\MyApp\\MyApp.exe"
}
```

### 2.2 `launch_response`
Returned by Hrunner after inspecting the local runtime and package store.

```json
{
  "protocol_version": 1,
  "type": "launch_response",
  "status": "ready",
  "python_available": true
}
```

If installation is needed:
```json
{
  "protocol_version": 1,
  "type": "launch_response",
  "status": "install_required",
  "missing_python": true,
  "missing_packages": [
    {
      "name": "pillow",
      "version": "11.3.0",
      "download_bytes": 9438212
    }
  ],
  "total_download_bytes": 24438212
}
```

### 2.3 `install_progress`
Streamed during download and installation.

```json
{
  "protocol_version": 1,
  "type": "install_progress",
  "step": "downloading_packages",
  "item_name": "pillow",
  "item_index": 1,
  "total_items": 2,
  "bytes_completed": 4719106,
  "total_bytes": 9438212,
  "percent": 50.0,
  "message": "Downloading pillow: 50.0%"
}
```

### 2.4 `launch_ready`
Sent when all dependencies and Python runtimes are prepared for execution.

```json
{
  "protocol_version": 1,
  "type": "launch_ready",
  "python_exe_path": "C:\\Users\\User\\AppData\\Local\\Hrunner\\runtimes\\python-3.13.7\\python.exe",
  "package_paths": [
    "C:\\Users\\User\\AppData\\Local\\Hrunner\\packages\\requests\\2.32.5",
    "C:\\Users\\User\\AppData\\Local\\Hrunner\\packages\\pillow\\11.3.0"
  ],
  "entrypoint": "main.py"
}
```

### 2.5 `ping` and `pong`
Used to probe connection liveliness.

---

## 3. Server Lifecycle & Auto-Shutdown

1. The server tracks `activeClients` and `activeOps`.
2. When an application connects, `activeClients` increments.
3. When the application terminates and closes the pipe, `activeClients` decrements.
4. If `activeClients == 0` and no operations/manager windows are open, an idle countdown timer starts (default: 5 seconds).
5. If no new client connects within the timeout, Hrunner exits automatically.
