# Shared Package Pool & Dependency Management

One of the core innovations of Hrunner is the **Shared Package Pool** with **Physical Deduplication**.

---

## 1. Storage Organization

All packages downloaded from PyPI are installed into `%LOCALAPPDATA%\Hrunner\packages`:

```text
%LOCALAPPDATA%\Hrunner\packages\
├── requests\
│   └── 2.32.5\
│       ├── requests/
│       └── requests-2.32.5.dist-info/
├── numpy\
│   ├── 1.26.4\
│   └── 2.3.2\
└── pillow\
    └── 11.3.0\
```

Every package name is normalized according to PEP 503 (lowercase, underscores replaced by hyphens). Every version is stored in an independent directory.

---

## 2. Physical Deduplication

If three different applications installed on the computer all require `requests == 2.32.5`:

```text
App A (requests 2.32.5) ──┐
App B (requests 2.32.5) ──┼──> %LOCALAPPDATA%\Hrunner\packages\requests\2.32.5\
App C (requests 2.32.5) ──┘    (Single physical directory on disk, RefCount = 3)
```

* Only **one** copy is stored on disk.
* Disk space is saved across all applications.
* Package installations are immutable after extraction.

---

## 3. Multi-Version Coexistence

Different versions of the same package coexist simultaneously:

```text
Application A ──> numpy 1.26.4
Application B ──> numpy 2.3.2
```

Both directories exist independently in `%LOCALAPPDATA%\Hrunner\packages\numpy\`. Updating or removing Application A never breaks Application B.

---

## 4. Wheel Selection & Native Extensions

Hrunner supports pure Python packages and native C-extensions (`.pyd`, `.dll`):

1. Queries PyPI JSON API: `https://pypi.org/pypi/<package>/<version>/json`.
2. Inspects wheels for Windows x64 tags:
   * `win_amd64`
   * `any`
   * Python ABI tags (`cp313-cp313`, `cp312-cp312`, `cp3-abi3`)
3. Ranks available wheels and selects the best matching binary wheel.
4. Verifies the SHA-256 digest against the PyPI hash.
5. Atomically extracts the wheel archive into the package pool.
