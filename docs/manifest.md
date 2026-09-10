# Application Manifest Specification

Every application packaged with Hrunner includes a versioned `manifest.json` embedded in its executable payload.

---

## 1. Schema Example (Format Version 1)

```json
{
  "format_version": 1,
  "application_id": "com.example.myapp",
  "name": "My Application",
  "version": "1.0.0",
  "publisher": "Example Corp",
  "description": "Desktop image processor",
  "min_hrunner_version": "1.0.0",
  "python": {
    "version": "3.13.7",
    "allow_compatible": false
  },
  "dependencies": {
    "requests": "2.32.5",
    "pillow": "11.3.0"
  },
  "entrypoint": "main.py"
}
```

---

## 2. Field Definitions

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `format_version` | `integer` | **Yes** | Schema version (must be `1`). |
| `application_id` | `string` | **Yes** | Unique reverse-DNS identifier (e.g. `com.company.app`). |
| `name` | `string` | **Yes** | User-facing application display name. |
| `version` | `string` | **Yes** | Semantic version string (`X.Y.Z`). |
| `publisher` | `string` | No | Publisher / author name. |
| `description` | `string` | No | Short description of the application. |
| `min_hrunner_version` | `string` | No | Minimum Hrunner version required to run this app. |
| `python.version` | `string` | **Yes** | Required Python version (e.g. `3.13.7`). |
| `python.allow_compatible`| `boolean` | No | If `true`, permits compatible installed minor versions. |
| `dependencies` | `object` | **Yes** | Key-value map of normalized package names to pinned versions. |
| `entrypoint` | `string` | **Yes** | Script to execute (`.py` or `.pyw`). |
