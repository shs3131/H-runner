package storage

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/registry"
)

func createDummyWheel(t *testing.T, dir, filename string) string {
	t.Helper()
	p := filepath.Join(dir, filename)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, _ := zw.Create("mod/__init__.py")
	_, _ = w.Write([]byte("# code"))
	_ = zw.Close()
	return p
}

func TestStorageReferenceCountingAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	pkgStore, err := packages.NewPackageStore(tmpDir, nil)
	if err != nil {
		t.Fatalf("failed to create package store: %v", err)
	}

	storageMgr, err := NewStorageManager(tmpDir, reg, pkgStore)
	if err != nil {
		t.Fatalf("failed to create storage manager: %v", err)
	}

	// Install 3 packages into pool:
	// 1. requests 2.32.5
	// 2. pillow 11.3.0
	// 3. unused-pkg 1.0.0
	whl1 := createDummyWheel(t, tmpDir, "requests-2.32.5-py3-none-any.whl")
	whl2 := createDummyWheel(t, tmpDir, "pillow-11.3.0-py3-none-any.whl")
	whl3 := createDummyWheel(t, tmpDir, "unused_pkg-1.0.0-py3-none-any.whl")

	_, _ = pkgStore.InstallWheel(whl1, "requests", "2.32.5")
	_, _ = pkgStore.InstallWheel(whl2, "pillow", "11.3.0")
	_, _ = pkgStore.InstallWheel(whl3, "unused-pkg", "1.0.0")

	// Register App A: uses requests and pillow
	_ = reg.Register(&registry.ApplicationRecord{
		ApplicationID: "com.app.a",
		Name:          "App A",
		Version:       "1.0.0",
		Dependencies: map[string]string{
			"requests": "2.32.5",
			"pillow":   "11.3.0",
		},
		State: registry.StateReady,
	})

	// Register App B: uses requests only
	_ = reg.Register(&registry.ApplicationRecord{
		ApplicationID: "com.app.b",
		Name:          "App B",
		Version:       "2.0.0",
		Dependencies: map[string]string{
			"requests": "2.32.5",
		},
		State: registry.StateReady,
	})

	// 1. Verify Reference Counts
	usages, err := storageMgr.GetPackageUsageMap()
	if err != nil {
		t.Fatalf("failed to get package usages: %v", err)
	}

	counts := make(map[string]int)
	for _, u := range usages {
		counts[u.Name] = u.RefCount
	}

	if counts["requests"] != 2 {
		t.Errorf("expected requests refcount 2, got %d", counts["requests"])
	}
	if counts["pillow"] != 1 {
		t.Errorf("expected pillow refcount 1, got %d", counts["pillow"])
	}
	if counts["unused-pkg"] != 0 {
		t.Errorf("expected unused-pkg refcount 0, got %d", counts["unused-pkg"])
	}

	// 2. Verify Unused Packages list
	unused, err := storageMgr.GetUnusedPackages()
	if err != nil {
		t.Fatalf("failed to get unused packages: %v", err)
	}
	if len(unused) != 1 || unused[0].Name != "unused-pkg" {
		t.Errorf("expected 1 unused package 'unused-pkg', got %v", unused)
	}

	// 3. Test App Removal Impact calculation
	impact, err := storageMgr.CalculateAppRemovalImpact("com.app.a")
	if err != nil {
		t.Fatalf("failed to calculate impact: %v", err)
	}
	// Removing App A should make pillow unused, but NOT requests (since App B uses requests)
	if len(impact.PackagesNowUnused) != 1 || impact.PackagesNowUnused[0].Name != "pillow" {
		t.Errorf("expected pillow to be orphaned by App A removal, got %v", impact.PackagesNowUnused)
	}

	// 4. Test Cleaning Unused Packages
	deleted, _, err := storageMgr.CleanUnusedPackages()
	if err != nil {
		t.Fatalf("failed to clean unused packages: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 package cleaned, got %d", deleted)
	}
	if pkgStore.IsPackageInstalled("unused-pkg", "1.0.0") {
		t.Errorf("expected unused-pkg to be deleted from pool")
	}
	if !pkgStore.IsPackageInstalled("requests", "2.32.5") {
		t.Errorf("expected requests to remain in pool")
	}

	// 5. Test Summary
	summary, err := storageMgr.GetSummary()
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}
	if summary.PackagesBytes <= 0 {
		t.Errorf("expected positive packages size, got %d", summary.PackagesBytes)
	}
}
