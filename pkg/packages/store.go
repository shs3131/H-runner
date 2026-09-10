package packages

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hrunner/hrunner/pkg/runtime"
)

// InstalledPackage represents a package version residing in the package pool.
type InstalledPackage struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}

// PackageStore manages the shared, deduplicated package pool under %LOCALAPPDATA%\Hrunner\packages.
type PackageStore struct {
	mu          sync.RWMutex
	packagesDir string
	cacheDir    string
	pypi        *PyPIClient
}

// NewPackageStore initializes the package store.
func NewPackageStore(baseDir string, pypi *PyPIClient) (*PackageStore, error) {
	if baseDir == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			baseDir = filepath.Join(localAppData, "Hrunner")
		} else {
			baseDir = filepath.Join(".", ".hrunner")
		}
	}

	packagesDir := filepath.Join(baseDir, "packages")
	cacheDir := filepath.Join(baseDir, "cache")

	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create packages directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	if pypi == nil {
		pypi = NewPyPIClient("")
	}

	return &PackageStore{
		packagesDir: packagesDir,
		cacheDir:    cacheDir,
		pypi:        pypi,
	}, nil
}

// NormalizePackageName standardizes Python package names (PEP 503: lowercase, replace _ and . with -).
func NormalizePackageName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

// GetPackageDir returns the canonical path for a package version in the pool.
func (s *PackageStore) GetPackageDir(name, version string) string {
	norm := NormalizePackageName(name)
	return filepath.Join(s.packagesDir, norm, version)
}

// IsPackageInstalled returns true if the package version is already installed in the pool.
func (s *PackageStore) IsPackageInstalled(name, version string) bool {
	dir := s.GetPackageDir(name, version)
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return false
	}
	// Verify directory is non-empty
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}

// ListAllPackages returns all packages and versions in the pool.
func (s *PackageStore) ListAllPackages() ([]*InstalledPackage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*InstalledPackage
	pkgDirs, err := os.ReadDir(s.packagesDir)
	if err != nil {
		return nil, err
	}

	for _, pEntry := range pkgDirs {
		if !pEntry.IsDir() {
			continue
		}
		pkgName := pEntry.Name()
		verDirs, err := os.ReadDir(filepath.Join(s.packagesDir, pkgName))
		if err != nil {
			continue
		}

		for _, vEntry := range verDirs {
			if !vEntry.IsDir() {
				continue
			}
			verStr := vEntry.Name()
			fullPath := filepath.Join(s.packagesDir, pkgName, verStr)
			size := calculateDirSize(fullPath)
			result = append(result, &InstalledPackage{
				Name:      pkgName,
				Version:   verStr,
				Path:      fullPath,
				SizeBytes: size,
			})
		}
	}

	return result, nil
}

// InstallWheel extracts a wheel file into the package pool atomically.
func (s *PackageStore) InstallWheel(wheelPath, pkgName, version string) (*InstalledPackage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normName := NormalizePackageName(pkgName)
	targetDir := filepath.Join(s.packagesDir, normName, version)

	// If already installed, return immediately (deduplication)
	if s.IsPackageInstalled(normName, version) {
		return &InstalledPackage{
			Name:      normName,
			Version:   version,
			Path:      targetDir,
			SizeBytes: calculateDirSize(targetDir),
		}, nil
	}

	// Temporary extraction directory
	tmpDir := filepath.Join(s.packagesDir, fmt.Sprintf("tmp_%s_%s_%d", normName, version, time.Now().UnixNano()))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp package extraction dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Extract wheel (zip)
	if err := extractWheel(wheelPath, tmpDir); err != nil {
		return nil, fmt.Errorf("failed to extract wheel %s: %w", wheelPath, err)
	}

	// Ensure parent directory exists: packages/<normName>
	parentDir := filepath.Join(s.packagesDir, normName)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create package parent dir: %w", err)
	}

	// Commit atomic rename
	_ = os.RemoveAll(targetDir)
	if err := os.Rename(tmpDir, targetDir); err != nil {
		return nil, fmt.Errorf("failed to commit package installation to %s: %w", targetDir, err)
	}

	return &InstalledPackage{
		Name:      normName,
		Version:   version,
		Path:      targetDir,
		SizeBytes: calculateDirSize(targetDir),
	}, nil
}

// EnsurePackage downloads and installs a package if missing, or reuses existing (deduplication).
func (s *PackageStore) EnsurePackage(name, version string, pyVer *runtime.Version, progress func(downloaded, total int64, pct float64)) (*InstalledPackage, error) {
	normName := NormalizePackageName(name)
	targetDir := filepath.Join(s.packagesDir, normName, version)

	// Check if already in pool (Shared Package Deduplication)
	if s.IsPackageInstalled(normName, version) {
		return &InstalledPackage{
			Name:      normName,
			Version:   version,
			Path:      targetDir,
			SizeBytes: calculateDirSize(targetDir),
		}, nil
	}

	meta, err := s.pypi.FetchMetadata(name, version)
	if err != nil {
		return nil, err
	}

	wheel, err := s.pypi.SelectBestWheel(meta, pyVer)
	if err != nil {
		return nil, err
	}

	cacheWheelPath := filepath.Join(s.cacheDir, wheel.Filename)
	if err := s.pypi.DownloadWheel(wheel, cacheWheelPath, progress); err != nil {
		return nil, err
	}

	return s.InstallWheel(cacheWheelPath, name, version)
}

// RemovePackageVersion deletes a specific package version from the pool.
func (s *PackageStore) RemovePackageVersion(name, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normName := NormalizePackageName(name)
	targetDir := filepath.Join(s.packagesDir, normName, version)
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}

	// If package parent directory is empty, remove it too
	parentDir := filepath.Join(s.packagesDir, normName)
	entries, _ := os.ReadDir(parentDir)
	if len(entries) == 0 {
		_ = os.Remove(parentDir)
	}

	return nil
}

func extractWheel(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in wheel: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func calculateDirSize(dir string) int64 {
	var size int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
