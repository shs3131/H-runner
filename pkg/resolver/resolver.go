package resolver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/runtime"
)

// ResolvedPackage contains resolution details for a single dependency.
type ResolvedPackage struct {
	Name          string
	Version       string
	AlreadyInPool bool
	DownloadBytes int64
}

// ResolutionPlan represents the outcome of resolving all direct and transitive dependencies.
type ResolutionPlan struct {
	AllPackages        map[string]string // normalized name -> version
	MissingPackages    []*ResolvedPackage
	TotalDownloadBytes int64
}

// Resolver resolves dependencies using standard PyPI metadata and package store.
type Resolver struct {
	store *packages.PackageStore
	pypi  *packages.PyPIClient
}

// NewResolver creates a new dependency resolver.
func NewResolver(store *packages.PackageStore, pypi *packages.PyPIClient) *Resolver {
	if pypi == nil {
		pypi = packages.NewPyPIClient("")
	}
	return &Resolver{
		store: store,
		pypi:  pypi,
	}
}

// distReqRegex extracts package name and optional version requirement from Requires-Dist.
// Examples:
// "urllib3 (<3,>=1.21.1)"
// "certifi >=2017.4.17"
// "idna <4,>=2.5"
// "win-inet-pton; (sys_platform == 'win32')"
var distReqRegex = regexp.MustCompile(`^([a-zA-Z0-9_.-]+)\s*(?:\(([^)]+)\)|([<>=!~].*))?$`)

// ParseRequiresDist parses a Requires-Dist entry, ignoring environment markers for non-windows platforms or optional extras.
func ParseRequiresDist(reqStr string) (string, string, bool) {
	reqStr = strings.TrimSpace(reqStr)
	// Check for environment markers separated by ';'
	parts := strings.SplitN(reqStr, ";", 2)
	depPart := strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		marker := strings.ToLower(parts[1])
		// Ignore optional extras (e.g. extra == 'socks') for MVP base install
		if strings.Contains(marker, "extra ==") || strings.Contains(marker, "extra==") {
			return "", "", false
		}
		// Check platform marker: if explicitly requires non-win32/non-windows, skip
		if (strings.Contains(marker, "sys_platform") || strings.Contains(marker, "os_name")) &&
			!strings.Contains(marker, "win32") && !strings.Contains(marker, "windows") {
			return "", "", false
		}
	}

	sub := distReqRegex.FindStringSubmatch(depPart)
	if sub == nil {
		return "", "", false
	}

	pkgName := packages.NormalizePackageName(sub[1])
	verConstraint := ""
	if sub[2] != "" {
		verConstraint = strings.TrimSpace(sub[2])
	} else if sub[3] != "" {
		verConstraint = strings.TrimSpace(sub[3])
	}

	return pkgName, verConstraint, true
}

// Resolve processes direct application dependencies and their transitive requirements.
func (r *Resolver) Resolve(directDeps map[string]string, pyVer *runtime.Version) (*ResolutionPlan, error) {
	plan := &ResolutionPlan{
		AllPackages: make(map[string]string),
	}

	// First pass: add direct pinned dependencies
	for pkg, ver := range directDeps {
		norm := packages.NormalizePackageName(pkg)
		ver = strings.TrimSpace(ver)
		plan.AllPackages[norm] = ver
	}

	// Queue for BFS traversal
	type queueItem struct {
		name    string
		version string
	}
	queue := make([]queueItem, 0, len(directDeps))
	for pkg, ver := range directDeps {
		queue = append(queue, queueItem{name: packages.NormalizePackageName(pkg), version: ver})
	}

	visited := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr.name] {
			continue
		}
		visited[curr.name] = true

		// Fetch metadata for curr
		meta, err := r.pypi.FetchMetadata(curr.name, curr.version)
		if err != nil {
			// If offline or package not on PyPI, but already in local pool, we can continue
			if r.store.IsPackageInstalled(curr.name, curr.version) {
				continue
			}
			return nil, fmt.Errorf("failed to fetch metadata for %s %s: %w", curr.name, curr.version, err)
		}

		// Inspect transitive dependencies
		for _, reqLine := range meta.RequiresDist {
			depName, constraint, ok := ParseRequiresDist(reqLine)
			if !ok || depName == "" {
				continue
			}

			// Check if already in plan
			existingVer, hasExisting := plan.AllPackages[depName]
			if hasExisting {
				// Check for conflict: if constraint is strict "==X" and differs from existing
				if strings.HasPrefix(constraint, "==") {
					strictVer := strings.TrimPrefix(constraint, "==")
					strictVer = strings.TrimSpace(strictVer)
					if strictVer != existingVer {
						return nil, fmt.Errorf("dependency conflict: %s requires %s==%s, but %s is already pinned to %s",
							curr.name, depName, strictVer, depName, existingVer)
					}
				}
				continue
			}

			// Transitive dependency not yet in plan: fetch its latest metadata to select a version
			depMeta, err := r.pypi.FetchMetadata(depName, "")
			if err != nil {
				return nil, fmt.Errorf("failed to resolve transitive dependency %s (required by %s): %w", depName, curr.name, err)
			}

			plan.AllPackages[depName] = depMeta.Version
			queue = append(queue, queueItem{name: depName, version: depMeta.Version})
		}
	}

	// Calculate which packages are already in pool and which need download
	for pkg, ver := range plan.AllPackages {
		inPool := r.store.IsPackageInstalled(pkg, ver)
		var downloadBytes int64

		if !inPool {
			meta, err := r.pypi.FetchMetadata(pkg, ver)
			if err == nil {
				wheel, err := r.pypi.SelectBestWheel(meta, pyVer)
				if err == nil {
					downloadBytes = wheel.SizeBytes
				}
			}
			plan.TotalDownloadBytes += downloadBytes
			plan.MissingPackages = append(plan.MissingPackages, &ResolvedPackage{
				Name:          pkg,
				Version:       ver,
				AlreadyInPool: false,
				DownloadBytes: downloadBytes,
			})
		}
	}

	return plan, nil
}
