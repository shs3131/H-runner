package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CurrentFormatVersion defines the supported manifest schema format version.
const CurrentFormatVersion = 1

var (
	// appIDRegex ensures standard reverse-DNS or alphanumeric identifier: com.example.app
	appIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)*$`)
	// semverRegex checks standard X.Y.Z or X.Y format
	semverRegex = regexp.MustCompile(`^\d+\.\d+(\.\d+)?(-[a-zA-Z0-9._-]+)?$`)
)

// PythonConfig contains Python runtime requirements.
type PythonConfig struct {
	Version         string `json:"version"`                    // e.g. "3.13.7"
	AllowCompatible bool   `json:"allow_compatible,omitempty"` // If true, allows compatible minor version
}

// Manifest represents the application manifest embedded in generated executables.
type Manifest struct {
	FormatVersion     int               `json:"format_version"`
	ApplicationID     string            `json:"application_id"`
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	Publisher         string            `json:"publisher,omitempty"`
	Description       string            `json:"description,omitempty"`
	Icon              string            `json:"icon,omitempty"`
	Homepage          string            `json:"homepage,omitempty"`
	License           string            `json:"license,omitempty"`
	Author            string            `json:"author,omitempty"`
	MinHrunnerVersion string            `json:"min_hrunner_version,omitempty"`
	Python            PythonConfig      `json:"python"`
	Dependencies      map[string]string `json:"dependencies"` // e.g. "requests": "2.32.5"
	Entrypoint        string            `json:"entrypoint"`   // e.g. "main.py"
}

// Parse parses and validates a Manifest from JSON bytes.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid json manifest: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

// ParseFile parses and validates a Manifest from a file path.
func ParseFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	return Parse(data)
}

// Validate checks all required fields, format version, and semantic integrity.
func (m *Manifest) Validate() error {
	if m.FormatVersion <= 0 {
		return errors.New("manifest format_version must be specified and positive")
	}
	if m.FormatVersion > CurrentFormatVersion {
		return fmt.Errorf("unsupported manifest format_version %d (current supported: %d)", m.FormatVersion, CurrentFormatVersion)
	}

	m.ApplicationID = strings.TrimSpace(m.ApplicationID)
	if m.ApplicationID == "" {
		return errors.New("application_id cannot be empty")
	}
	if !appIDRegex.MatchString(m.ApplicationID) {
		return fmt.Errorf("invalid application_id '%s': must be alphanumeric with optional dots/hyphens/underscores", m.ApplicationID)
	}

	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("name cannot be empty")
	}

	m.Version = strings.TrimSpace(m.Version)
	if m.Version == "" {
		return errors.New("version cannot be empty")
	}
	if !semverRegex.MatchString(m.Version) {
		return fmt.Errorf("invalid version '%s': must be in semantic format (e.g. 1.0.0)", m.Version)
	}

	m.Python.Version = strings.TrimSpace(m.Python.Version)
	if m.Python.Version == "" {
		return errors.New("python.version must be specified")
	}
	if !semverRegex.MatchString(m.Python.Version) {
		return fmt.Errorf("invalid python.version '%s': must be semver format (e.g. 3.13.7)", m.Python.Version)
	}

	m.Entrypoint = strings.TrimSpace(m.Entrypoint)
	if m.Entrypoint == "" {
		return errors.New("entrypoint cannot be empty")
	}
	ext := strings.ToLower(filepath.Ext(m.Entrypoint))
	if ext != ".py" && ext != ".pyw" {
		return fmt.Errorf("entrypoint must be a Python script (.py or .pyw), got '%s'", m.Entrypoint)
	}

	// Validate dependency map
	for pkg, ver := range m.Dependencies {
		pkgTrimmed := strings.TrimSpace(pkg)
		verTrimmed := strings.TrimSpace(ver)
		if pkgTrimmed == "" {
			return errors.New("dependency package name cannot be empty")
		}
		if verTrimmed == "" {
			return fmt.Errorf("dependency '%s' must specify a pinned version", pkg)
		}
	}

	return nil
}

// ToJSON serializes the manifest to formatted JSON.
func (m *Manifest) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
