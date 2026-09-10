package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/registry"
)

// StorageSummary contains disk usage statistics.
type StorageSummary struct {
	RuntimesBytes     int64 `json:"runtimes_bytes"`
	PackagesBytes     int64 `json:"packages_bytes"`
	ApplicationsBytes int64 `json:"applications_bytes"`
	CacheBytes        int64 `json:"cache_bytes"`
	TotalBytes        int64 `json:"total_bytes"`
}

// PackageUsage tracks how many and which applications use a specific package version.
type PackageUsage struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	UsedByApps []string `json:"used_by_apps"` // application IDs
	RefCount   int      `json:"ref_count"`
	SizeBytes  int64    `json:"size_bytes"`
}

// RemovalImpact calculates what happens when an application is uninstalled.
type RemovalImpact struct {
	ApplicationID      string          `json:"application_id"`
	ApplicationName    string          `json:"application_name"`
	PackagesNowUnused  []*PackageUsage `json:"packages_now_unused"`
	ReclaimedBytesPool int64           `json:"reclaimed_bytes_pool"`
}

// StorageManager provides storage metrics, reference counting, and cleanup.
type StorageManager struct {
	mu           sync.RWMutex
	baseDir      string
	registry     *registry.RegistryStore
	packageStore *packages.PackageStore
}

// NewStorageManager creates a new storage manager.
func NewStorageManager(baseDir string, reg *registry.RegistryStore, pkgStore *packages.PackageStore) (*StorageManager, error) {
	if baseDir == "" {
		baseDir = registry.DefaultHrunnerDir()
	}

	return &StorageManager{
		baseDir:      baseDir,
		registry:     reg,
		packageStore: pkgStore,
	}, nil
}

// GetSummary calculates the current disk space consumed by Hrunner components.
func (s *StorageManager) GetSummary() (*StorageSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runtimesDir := filepath.Join(s.baseDir, "runtimes")
	packagesDir := filepath.Join(s.baseDir, "packages")
	appsDir := filepath.Join(s.baseDir, "applications")
	cacheDir := filepath.Join(s.baseDir, "cache")

	runtimesBytes := calculateDirSize(runtimesDir)
	packagesBytes := calculateDirSize(packagesDir)
	appsBytes := calculateDirSize(appsDir)
	cacheBytes := calculateDirSize(cacheDir)

	return &StorageSummary{
		RuntimesBytes:     runtimesBytes,
		PackagesBytes:     packagesBytes,
		ApplicationsBytes: appsBytes,
		CacheBytes:        cacheBytes,
		TotalBytes:        runtimesBytes + packagesBytes + appsBytes + cacheBytes,
	}, nil
}

// GetPackageUsageMap returns reference counts for all packages in the pool.
func (s *StorageManager) GetPackageUsageMap() ([]*PackageUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getPackageUsageMapLocked()
}

func (s *StorageManager) getPackageUsageMapLocked() ([]*PackageUsage, error) {
	poolPackages, err := s.packageStore.ListAllPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	apps := s.registry.List()

	usageMap := make(map[string]*PackageUsage)
	for _, p := range poolPackages {
		key := fmt.Sprintf("%s@%s", packages.NormalizePackageName(p.Name), p.Version)
		usageMap[key] = &PackageUsage{
			Name:       p.Name,
			Version:    p.Version,
			UsedByApps: []string{},
			RefCount:   0,
			SizeBytes:  p.SizeBytes,
		}
	}

	for _, app := range apps {
		for pkgName, pkgVer := range app.Dependencies {
			normName := packages.NormalizePackageName(pkgName)
			key := fmt.Sprintf("%s@%s", normName, pkgVer)
			if usage, ok := usageMap[key]; ok {
				usage.UsedByApps = append(usage.UsedByApps, app.ApplicationID)
				usage.RefCount++
			}
		}
	}

	result := make([]*PackageUsage, 0, len(usageMap))
	for _, u := range usageMap {
		result = append(result, u)
	}

	return result, nil
}

// GetUnusedPackages returns packages in the pool that have a reference count of 0.
func (s *StorageManager) GetUnusedPackages() ([]*PackageUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usages, err := s.getPackageUsageMapLocked()
	if err != nil {
		return nil, err
	}

	var unused []*PackageUsage
	for _, u := range usages {
		if u.RefCount == 0 {
			unused = append(unused, u)
		}
	}
	return unused, nil
}

// CleanUnusedPackages deletes all packages with reference count of 0.
func (s *StorageManager) CleanUnusedPackages() (int, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usages, err := s.getPackageUsageMapLocked()
	if err != nil {
		return 0, 0, err
	}

	var deletedCount int
	var freedBytes int64

	for _, u := range usages {
		if u.RefCount == 0 {
			if err := s.packageStore.RemovePackageVersion(u.Name, u.Version); err == nil {
				deletedCount++
				freedBytes += u.SizeBytes
			}
		}
	}

	return deletedCount, freedBytes, nil
}

// CalculateAppRemovalImpact determines which packages would become orphaned if the specified app is uninstalled.
func (s *StorageManager) CalculateAppRemovalImpact(appID string) (*RemovalImpact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.calculateAppRemovalImpactLocked(appID)
}

func (s *StorageManager) calculateAppRemovalImpactLocked(appID string) (*RemovalImpact, error) {
	app, exists := s.registry.Get(appID)
	if !exists {
		return nil, fmt.Errorf("application '%s' not found", appID)
	}

	usages, err := s.getPackageUsageMapLocked()
	if err != nil {
		return nil, err
	}

	impact := &RemovalImpact{
		ApplicationID:   app.ApplicationID,
		ApplicationName: app.Name,
	}

	for _, u := range usages {
		// If only this app uses this package
		if u.RefCount == 1 && len(u.UsedByApps) == 1 && u.UsedByApps[0] == appID {
			impact.PackagesNowUnused = append(impact.PackagesNowUnused, u)
			impact.ReclaimedBytesPool += u.SizeBytes
		}
	}

	return impact, nil
}

// RemoveApplicationAndCleanup removes an application from registry and optionally removes newly unused packages.
func (s *StorageManager) RemoveApplicationAndCleanup(appID string, cleanUnusedPackages bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	impact, err := s.calculateAppRemovalImpactLocked(appID)
	if err != nil {
		return err
	}

	if err := s.registry.Remove(appID); err != nil {
		return fmt.Errorf("failed to remove application record: %w", err)
	}

	if cleanUnusedPackages {
		for _, u := range impact.PackagesNowUnused {
			_ = s.packageStore.RemovePackageVersion(u.Name, u.Version)
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
