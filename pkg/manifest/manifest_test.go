package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidManifest(t *testing.T) {
	jsonStr := `{
		"format_version": 1,
		"application_id": "com.example.myapp",
		"name": "My Application",
		"version": "1.0.0",
		"publisher": "Example Corp",
		"python": {
			"version": "3.13.7",
			"allow_compatible": false
		},
		"dependencies": {
			"requests": "2.32.5",
			"pillow": "11.3.0"
		},
		"entrypoint": "main.py"
	}`

	m, err := Parse([]byte(jsonStr))
	if err != nil {
		t.Fatalf("unexpected error parsing valid manifest: %v", err)
	}

	if m.FormatVersion != 1 {
		t.Errorf("expected format_version 1, got %d", m.FormatVersion)
	}
	if m.ApplicationID != "com.example.myapp" {
		t.Errorf("expected app id com.example.myapp, got %s", m.ApplicationID)
	}
	if len(m.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(m.Dependencies))
	}
	if m.Dependencies["requests"] != "2.32.5" {
		t.Errorf("expected requests 2.32.5, got %s", m.Dependencies["requests"])
	}

	out, err := m.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize manifest to json: %v", err)
	}
	m2, err := Parse(out)
	if err != nil {
		t.Fatalf("failed to re-parse serialized manifest: %v", err)
	}
	if m2.Name != m.Name {
		t.Errorf("expected name %s, got %s", m.Name, m2.Name)
	}
}

func TestInvalidManifests(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
	}{
		{
			name: "unsupported format version",
			jsonStr: `{
				"format_version": 999,
				"application_id": "com.example.myapp",
				"name": "App",
				"version": "1.0.0",
				"python": {"version": "3.13.7"},
				"entrypoint": "main.py"
			}`,
		},
		{
			name: "missing format version",
			jsonStr: `{
				"application_id": "com.example.myapp",
				"name": "App",
				"version": "1.0.0",
				"python": {"version": "3.13.7"},
				"entrypoint": "main.py"
			}`,
		},
		{
			name: "invalid application id",
			jsonStr: `{
				"format_version": 1,
				"application_id": "com/example/invalid!",
				"name": "App",
				"version": "1.0.0",
				"python": {"version": "3.13.7"},
				"entrypoint": "main.py"
			}`,
		},
		{
			name: "missing name",
			jsonStr: `{
				"format_version": 1,
				"application_id": "com.example.myapp",
				"version": "1.0.0",
				"python": {"version": "3.13.7"},
				"entrypoint": "main.py"
			}`,
		},
		{
			name: "invalid entrypoint extension",
			jsonStr: `{
				"format_version": 1,
				"application_id": "com.example.myapp",
				"name": "App",
				"version": "1.0.0",
				"python": {"version": "3.13.7"},
				"entrypoint": "main.exe"
			}`,
		},
		{
			name: "empty dependency version",
			jsonStr: `{
				"format_version": 1,
				"application_id": "com.example.myapp",
				"name": "App",
				"version": "1.0.0",
				"python": {"version": "3.13.7"},
				"dependencies": {"requests": ""},
				"entrypoint": "main.py"
			}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.jsonStr))
			if err == nil {
				t.Fatalf("expected error for test case '%s', got nil", tc.name)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "manifest.json")
	validJSON := `{
		"format_version": 1,
		"application_id": "com.test.file",
		"name": "File Test",
		"version": "1.2.3",
		"python": {"version": "3.12.0"},
		"entrypoint": "app.py"
	}`
	if err := os.WriteFile(filePath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("failed to write test manifest file: %v", err)
	}

	m, err := ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if m.ApplicationID != "com.test.file" {
		t.Errorf("expected com.test.file, got %s", m.ApplicationID)
	}
}
