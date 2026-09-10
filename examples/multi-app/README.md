# Multi-App Example: Physical Deduplication & Logical Isolation

This example demonstrates the core architectural value of Hrunner:
1. **Physical Deduplication**: Multiple applications referencing the exact same package version share a single immutable copy on disk.
2. **Logical Isolation**: Applications requesting different versions of the same package coexist without interference or version bleed.

## Structure

```text
examples/multi-app/
├── app-a/
│   ├── main.py            # Uses requests 2.32.5 + numpy 1.26.4
│   └── requirements.txt
├── app-b/
│   ├── main.py            # Uses requests 2.32.5 + numpy 2.3.2
│   └── requirements.txt
└── README.md
```

## Building Both Applications

Build Application A:
```powershell
hbuild build ./examples/multi-app/app-a --output ./dist/AppA.exe --name "App A" --version "1.0.0" --python "3.13.7"
```

Build Application B:
```powershell
hbuild build ./examples/multi-app/app-b --output ./dist/AppB.exe --name "App B" --version "1.0.0" --python "3.13.7"
```

## Verification in Hrunner

Once installed through Hrunner:

1. **Package Pool Inspection (`hrunner packages`)**:
   ```text
   Shared Package Pool:
     • requests 2.32.5 [SHARED] (Used by 2 apps: App A, App B)
     • numpy 1.26.4 [SHARED] (Used by 1 app: App A)
     • numpy 2.3.2 [SHARED] (Used by 1 app: App B)
   ```
   * `requests 2.32.5` exists **only once** on disk in `%LOCALAPPDATA%\Hrunner\packages\requests\2.32.5\`.
   * `numpy 1.26.4` and `numpy 2.3.2` coexist in separate subdirectories under `%LOCALAPPDATA%\Hrunner\packages\numpy\`.

2. **Isolated Execution**:
   * Running `AppA.exe` imports `numpy 1.26.4`.
   * Running `AppB.exe` imports `numpy 2.3.2`.
   * Neither application can see or load the other's version because Hrunner isolates `sys.path` to only the packages declared in each application's manifest.

3. **Safe Cleanup**:
   * If you remove Application A, `numpy 1.26.4` becomes unused and eligible for cleanup.
   * `requests 2.32.5` **remains installed** because Application B still depends on it.
