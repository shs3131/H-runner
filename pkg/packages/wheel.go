package packages

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hrunner/hrunner/pkg/runtime"
)

// WheelInfo parses wheel filename according to PEP 427.
// Format: {distribution}-{version}(-{build tag})?-{python tag}-{abi tag}-{platform tag}.whl
type WheelInfo struct {
	Filename     string
	Distribution string
	Version      string
	BuildTag     string
	PythonTag    string
	ABITag       string
	PlatformTag  string
	URL          string
	SHA256       string
	SizeBytes    int64
}

var wheelPattern = regexp.MustCompile(`^([^-]+)-([^-]+)(?:-([^-]+))?-([^-]+)-([^-]+)-([^-]+)\.whl$`)

// ParseWheelFilename parses a wheel filename string.
func ParseWheelFilename(name string) (*WheelInfo, error) {
	matches := wheelPattern.FindStringSubmatch(name)
	if matches == nil {
		return nil, fmt.Errorf("invalid wheel filename: %s", name)
	}

	w := &WheelInfo{
		Filename:     name,
		Distribution: matches[1],
		Version:      matches[2],
		BuildTag:     matches[3],
		PythonTag:    matches[4],
		ABITag:       matches[5],
		PlatformTag:  matches[6],
	}
	return w, nil
}

// IsCompatibleWindowsX64 checks if the wheel is compatible with the given Python version on Windows x64.
func (w *WheelInfo) IsCompatibleWindowsX64(pyVer *runtime.Version) bool {
	// Platform check: must be win_amd64 or any
	plat := strings.ToLower(w.PlatformTag)
	if !strings.Contains(plat, "win_amd64") && plat != "any" {
		return false
	}

	// Pure Python wheels: py3-none-any or py2.py3-none-any
	pyTag := strings.ToLower(w.PythonTag)
	abiTag := strings.ToLower(w.ABITag)

	if strings.Contains(pyTag, "py3") || strings.Contains(pyTag, "py2.py3") {
		if abiTag == "none" || plat == "any" {
			return true
		}
	}

	// Python CPython version specific: e.g. cp312, cp313
	expectedCpTag := fmt.Sprintf("cp%d%d", pyVer.Major, pyVer.Minor)

	// Check stable ABI abi3 (works on all python versions >= minimum)
	if strings.Contains(abiTag, "abi3") {
		if strings.Contains(pyTag, "cp3") {
			return true
		}
	}

	// Exact CPython match
	if strings.Contains(pyTag, expectedCpTag) {
		if abiTag == "none" || strings.Contains(abiTag, expectedCpTag) || strings.Contains(abiTag, "abi3") {
			return true
		}
	}

	return false
}

// WheelScore ranks wheels to pick the best match (higher is better).
// Priority:
// 1. Exact CPython version match for win_amd64 (e.g. cp313-cp313-win_amd64) -> score 100
// 2. Stable ABI for win_amd64 (cp3-abi3-win_amd64) -> score 80
// 3. Universal pure python (py3-none-any) -> score 50
// 4. py2.py3-none-any -> score 40
func (w *WheelInfo) WheelScore(pyVer *runtime.Version) int {
	expectedCpTag := fmt.Sprintf("cp%d%d", pyVer.Major, pyVer.Minor)
	plat := strings.ToLower(w.PlatformTag)
	pyTag := strings.ToLower(w.PythonTag)
	abiTag := strings.ToLower(w.ABITag)

	if strings.Contains(plat, "win_amd64") {
		if strings.Contains(pyTag, expectedCpTag) && strings.Contains(abiTag, expectedCpTag) {
			return 100
		}
		if strings.Contains(abiTag, "abi3") {
			return 80
		}
	}

	if plat == "any" || strings.Contains(plat, "win_amd64") {
		if pyTag == "py3" && abiTag == "none" {
			return 50
		}
		if strings.Contains(pyTag, "py2.py3") && abiTag == "none" {
			return 40
		}
	}

	return 10
}
