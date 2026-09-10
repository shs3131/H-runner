package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{"3.13.7", 3, 13, 7, false},
		{"3.12", 3, 12, 0, false},
		{"3.11.0", 3, 11, 0, false},
		{"invalid", 0, 0, 0, true},
		{"", 0, 0, 0, true},
		{"3.12.3.4", 0, 0, 0, true},
	}

	for _, tc := range tests {
		v, err := ParseVersion(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for '%s', got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for '%s': %v", tc.input, err)
		}
		if v.Major != tc.major || v.Minor != tc.minor || v.Patch != tc.patch {
			t.Errorf("version mismatch for '%s': got %v", tc.input, v)
		}
	}
}

func TestVersionCompatibility(t *testing.T) {
	v3137, _ := ParseVersion("3.13.7")
	v3138, _ := ParseVersion("3.13.8")
	v3120, _ := ParseVersion("3.12.0")

	// Strict mode
	if v3138.IsCompatible(v3137, false) {
		t.Errorf("expected 3.13.8 not strictly compatible with 3.13.7")
	}
	if !v3137.IsCompatible(v3137, false) {
		t.Errorf("expected 3.13.7 strictly compatible with 3.13.7")
	}

	// Compatible mode
	if !v3138.IsCompatible(v3137, true) {
		t.Errorf("expected 3.13.8 compatible with 3.13.7 in compatible mode")
	}
	if v3120.IsCompatible(v3137, true) {
		t.Errorf("expected 3.12.0 NOT compatible with 3.13.7 even in compatible mode")
	}
}

func TestRuntimeStoreDiscovery(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewRuntimeStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create runtime store: %v", err)
	}

	// Initially empty
	list, err := store.List()
	if err != nil {
		t.Fatalf("store.List() failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 runtimes, got %d", len(list))
	}

	// Create a mock python runtime
	rtDir := filepath.Join(tmpDir, "runtimes", "python-3.13.7")
	if err := os.MkdirAll(rtDir, 0755); err != nil {
		t.Fatalf("failed to create mock runtime dir: %v", err)
	}
	mockExe := filepath.Join(rtDir, "python.exe")
	if err := os.WriteFile(mockExe, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("failed to write mock python.exe: %v", err)
	}

	// Check discovery
	list, err = store.List()
	if err != nil {
		t.Fatalf("store.List() failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(list))
	}
	if list[0].Version.String() != "3.13.7" {
		t.Errorf("expected version 3.13.7, got %s", list[0].Version.String())
	}

	// Test Find exact
	found, exists, err := store.Find("3.13.7", false)
	if err != nil || !exists || found == nil {
		t.Fatalf("expected to find exact 3.13.7: found=%v, exists=%v, err=%v", found, exists, err)
	}

	// Test Find non-existent
	found2, exists2, err2 := store.Find("3.11.0", false)
	if err2 != nil || exists2 || found2 != nil {
		t.Fatalf("expected not to find 3.11.0: found=%v, exists=%v, err=%v", found2, exists2, err2)
	}
}

func TestBuildBootstrapScript(t *testing.T) {
	appDir := `C:\MyApp`
	entrypoint := `C:\MyApp\main.py`
	pkgPaths := []string{
		`C:\Hrunner\packages\requests\2.32.5`,
		`C:\Hrunner\packages\pillow\11.3.0`,
	}

	script := BuildBootstrapScript(appDir, entrypoint, pkgPaths)
	if !strings.Contains(script, "requests") {
		t.Errorf("expected script to contain requests package path")
	}
	if !strings.Contains(script, "pillow") {
		t.Errorf("expected script to contain pillow package path")
	}
	if !strings.Contains(script, "main.py") {
		t.Errorf("expected script to reference main.py")
	}
	if !strings.Contains(script, "add_dll_directory") {
		t.Errorf("expected script to handle add_dll_directory for native extensions")
	}
}
