package runtime

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic Python version (Major.Minor.Patch).
type Version struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// ParseVersion parses a version string like "3.12.8" or "3.13".
func ParseVersion(v string) (*Version, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, fmt.Errorf("empty version string")
	}

	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid version format '%s', expected X.Y or X.Y.Z", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version in '%s': %w", v, err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version in '%s': %w", v, err)
	}

	patch := 0
	if len(parts) == 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid patch version in '%s': %w", v, err)
		}
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
		Raw:   v,
	}, nil
}

// String returns formatted X.Y.Z version.
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsCompatible checks if installed version satisfies requested version.
// If allowCompatible is false, exact Major.Minor.Patch match is required.
// If allowCompatible is true, same Major and Minor is accepted.
func (v *Version) IsCompatible(requested *Version, allowCompatible bool) bool {
	if v.Major != requested.Major {
		return false
	}
	if v.Minor != requested.Minor {
		return false
	}
	if !allowCompatible {
		return v.Patch == requested.Patch
	}
	// Compatible mode: installed patch >= requested patch or any same minor
	return true
}

// Compare compares two versions. Returns:
// -1 if v < other
//  0 if v == other
//  1 if v > other
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}
