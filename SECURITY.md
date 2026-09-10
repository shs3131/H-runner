# Security Policy

---

## 1. Supported Versions

| Version | Supported |
| :--- | :--- |
| 1.0.x | :white_check_mark: |

---

## 2. Security Architecture & Safeguards

Hrunner incorporates the following security controls:

* **HTTPS Enforcement**: All downloads of Python runtimes and PyPI packages are performed strictly over encrypted HTTPS connections.
* **Cryptographic Hash Verification**: Package downloads are verified against SHA-256 digests retrieved from PyPI metadata before extraction.
* **Atomic Installation**: Downloads are staged in temporary files, and extractions occur in temporary staging directories (`tmp_*`) before atomic commit. Corrupted or partially downloaded archives are never committed to the package store.
* **Immutable Package Pool**: Package files in `%LOCALAPPDATA%\Hrunner\packages` are intended to be immutable once installed.
* **Local Named Pipe IPC**: Hrunner uses Windows Named Pipes (`\\.\pipe\hrunner`) restricted by Windows security descriptors to the current user. It exposes **no** TCP sockets, **no** web servers, and **no** localhost networking ports.
* **Process Isolation**: Applications run as independent processes with isolated `sys.path` environments.

> [!NOTE]
> In the current MVP version, cryptographic code signing of third-party application payloads is not yet implemented. Always verify the source and publisher of executables before running them.

---

## 3. Reporting a Vulnerability

If you discover a security vulnerability in Hrunner, please report it privately:

1. **Email**: Send details to [tgtygaming60@gmail.com](mailto:tgtygaming60@gmail.com).
2. **GitHub Advisory**: Alternatively, submit a private advisory through the GitHub repository security tab.
3. **Response Time**: You will receive an initial response within 48 hours.

Please do not disclose security vulnerabilities publicly until a fix has been released.
