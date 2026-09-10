package packages

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrunner/hrunner/pkg/runtime"
)

func TestWheelParsingAndCompatibility(t *testing.T) {
	v313, _ := runtime.ParseVersion("3.13.0")
	v312, _ := runtime.ParseVersion("3.12.0")

	tests := []struct {
		filename    string
		pyVer       *runtime.Version
		expectValid bool
		expectScore int
	}{
		// Pure python wheel: compatible with any python 3 on Windows x64
		{"requests-2.32.5-py3-none-any.whl", v313, true, 50},
		// Python 3.13 native wheel for Windows x64
		{"pillow-11.3.0-cp313-cp313-win_amd64.whl", v313, true, 100},
		// Python 3.13 native wheel should NOT be selected for Python 3.12
		{"pillow-11.3.0-cp313-cp313-win_amd64.whl", v312, false, 10},
		// Linux wheel should NOT be compatible with Windows
		{"pillow-11.3.0-cp313-cp313-manylinux_2_17_x86_64.whl", v313, false, 10},
		// macOS wheel should NOT be compatible with Windows
		{"numpy-2.3.2-cp313-cp313-macosx_14_0_arm64.whl", v313, false, 10},
		// Stable ABI wheel for Windows x64
		{"cryptography-43.0.1-cp39-abi3-win_amd64.whl", v313, true, 80},
	}

	for _, tc := range tests {
		w, err := ParseWheelFilename(tc.filename)
		if err != nil {
			t.Fatalf("failed to parse wheel %s: %v", tc.filename, err)
		}

		compat := w.IsCompatibleWindowsX64(tc.pyVer)
		if compat != tc.expectValid {
			t.Errorf("wheel %s with python %s: expected compat=%v, got %v", tc.filename, tc.pyVer.String(), tc.expectValid, compat)
		}
	}
}

func TestPackageStoreDeduplicationAndMultipleVersions(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewPackageStore(tmpDir, nil)
	if err != nil {
		t.Fatalf("failed to create package store: %v", err)
	}

	// Helper to create a dummy wheel zip
	createMockWheel := func(name string, files map[string]string) string {
		whlPath := filepath.Join(tmpDir, name)
		f, err := os.Create(whlPath)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		zw := zip.NewWriter(f)
		for fname, content := range files {
			w, err := zw.Create(fname)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(content))
		}
		_ = zw.Close()
		return whlPath
	}

	whl1264 := createMockWheel("numpy-1.26.4-py3-none-any.whl", map[string]string{
		"numpy/__init__.py": "__version__ = '1.26.4'",
	})
	whl232 := createMockWheel("numpy-2.3.2-py3-none-any.whl", map[string]string{
		"numpy/__init__.py": "__version__ = '2.3.2'",
	})

	// Install numpy 1.26.4
	pkg1, err := store.InstallWheel(whl1264, "numpy", "1.26.4")
	if err != nil {
		t.Fatalf("failed to install numpy 1.26.4: %v", err)
	}
	if !store.IsPackageInstalled("numpy", "1.26.4") {
		t.Fatal("expected numpy 1.26.4 to be installed")
	}

	// Install numpy 2.3.2 (multiple versions coexistence)
	pkg2, err := store.InstallWheel(whl232, "numpy", "2.3.2")
	if err != nil {
		t.Fatalf("failed to install numpy 2.3.2: %v", err)
	}
	if !store.IsPackageInstalled("numpy", "2.3.2") {
		t.Fatal("expected numpy 2.3.2 to be installed")
	}

	// Verify they have distinct physical paths
	if pkg1.Path == pkg2.Path {
		t.Errorf("expected distinct paths for numpy 1.26.4 and 2.3.2, got same: %s", pkg1.Path)
	}

	// Test Deduplication: App B installing same numpy 1.26.4 reuses existing without re-extracting
	pkg1Duplicate, err := store.InstallWheel(whl1264, "numpy", "1.26.4")
	if err != nil {
		t.Fatalf("failed duplicate install check: %v", err)
	}
	if pkg1Duplicate.Path != pkg1.Path {
		t.Errorf("deduplication failed: expected path %s, got %s", pkg1.Path, pkg1Duplicate.Path)
	}

	// List all packages
	all, err := store.ListAllPackages()
	if err != nil {
		t.Fatalf("failed to list packages: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 package versions in store, got %d", len(all))
	}
}
