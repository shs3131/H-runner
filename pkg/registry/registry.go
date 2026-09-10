package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ApplicationState represents the lifecycle state of an installed application.
type ApplicationState string

const (
	StateDiscovered ApplicationState = "discovered"
	StateInstalling ApplicationState = "installing"
	StateInstalled  ApplicationState = "installed"
	StateReady      ApplicationState = "ready"
	StateBroken     ApplicationState = "broken"
	StateRemoving   ApplicationState = "removing"
)

// ApplicationRecord stores registered application metadata.
type ApplicationRecord struct {
	ApplicationID  string            `json:"application_id"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Publisher      string            `json:"publisher,omitempty"`
	ExecutablePath string            `json:"executable_path"`
	PythonVersion  string            `json:"python_version"`
	Dependencies   map[string]string `json:"dependencies"` // package -> version
	Entrypoint     string            `json:"entrypoint"`
	State          ApplicationState  `json:"state"`
	InstalledAt    time.Time         `json:"installed_at"`
	LastLaunchedAt time.Time         `json:"last_launched_at"`
}

// RegistryStore manages application registration in registry.json.
type RegistryStore struct {
	mu       sync.RWMutex
	filePath string
	data     *registryData
}

type registryData struct {
	FormatVersion int                           `json:"format_version"`
	Applications  map[string]*ApplicationRecord `json:"applications"`
}

// DefaultHrunnerDir returns %LOCALAPPDATA%\Hrunner on Windows or user config dir.
func DefaultHrunnerDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		return filepath.Join(localAppData, "Hrunner")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, "AppData", "Local", "Hrunner")
	}
	return filepath.Join(".", ".hrunner")
}

// NewRegistry creates or loads the registry store from baseDir.
func NewRegistry(baseDir string) (*RegistryStore, error) {
	if baseDir == "" {
		baseDir = DefaultHrunnerDir()
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %w", err)
	}

	regFile := filepath.Join(baseDir, "registry.json")
	r := &RegistryStore{
		filePath: regFile,
		data: &registryData{
			FormatVersion: 1,
			Applications:  make(map[string]*ApplicationRecord),
		},
	}

	if err := r.load(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RegistryStore) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return r.saveLocked()
		}
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	var reg registryData
	if err := json.Unmarshal(data, &reg); err != nil {
		return fmt.Errorf("corrupted registry file: %w", err)
	}

	if reg.Applications == nil {
		reg.Applications = make(map[string]*ApplicationRecord)
	}

	r.data = &reg
	return nil
}

func (r *RegistryStore) saveLocked() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry data: %w", err)
	}

	// Atomic write via temp file
	tmpFile := r.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp registry file: %w", err)
	}

	if err := os.Rename(tmpFile, r.filePath); err != nil {
		// On Windows, rename might fail if target exists without removing
		_ = os.Remove(r.filePath)
		if err2 := os.Rename(tmpFile, r.filePath); err2 != nil {
			return fmt.Errorf("failed to commit registry file: %w", err2)
		}
	}

	return nil
}

// Register adds or updates an application record in the registry.
func (r *RegistryStore) Register(app *ApplicationRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if app.ApplicationID == "" {
		return errors.New("application_id cannot be empty")
	}

	if app.InstalledAt.IsZero() {
		app.InstalledAt = time.Now()
	}

	r.data.Applications[app.ApplicationID] = app
	return r.saveLocked()
}

// UpdateState updates the application's lifecycle state.
func (r *RegistryStore) UpdateState(appID string, state ApplicationState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, exists := r.data.Applications[appID]
	if !exists {
		return fmt.Errorf("application '%s' not found in registry", appID)
	}

	app.State = state
	return r.saveLocked()
}

// UpdateLastLaunched updates the last launched timestamp.
func (r *RegistryStore) UpdateLastLaunched(appID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, exists := r.data.Applications[appID]
	if !exists {
		return fmt.Errorf("application '%s' not found in registry", appID)
	}

	app.LastLaunchedAt = time.Now()
	return r.saveLocked()
}

// Get retrieves an application by ID.
func (r *RegistryStore) Get(appID string) (*ApplicationRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	app, exists := r.data.Applications[appID]
	if !exists {
		return nil, false
	}
	// Return a copy
	appCopy := *app
	return &appCopy, true
}

// List returns all registered applications.
func (r *RegistryStore) List() []*ApplicationRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*ApplicationRecord, 0, len(r.data.Applications))
	for _, app := range r.data.Applications {
		appCopy := *app
		list = append(list, &appCopy)
	}
	return list
}

// Remove deletes an application from the registry.
func (r *RegistryStore) Remove(appID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data.Applications[appID]; !exists {
		return fmt.Errorf("application '%s' not found in registry", appID)
	}

	delete(r.data.Applications, appID)
	return r.saveLocked()
}
