package registry

import (
	"path/filepath"
	"testing"
)

func TestRegistryCRUD(t *testing.T) {
	tmpDir := t.TempDir()

	reg, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	app := &ApplicationRecord{
		ApplicationID:  "com.test.myapp",
		Name:           "My App",
		Version:        "1.0.0",
		Publisher:      "Test Org",
		ExecutablePath: filepath.Join(tmpDir, "myapp.exe"),
		PythonVersion:  "3.13.7",
		Dependencies: map[string]string{
			"requests": "2.32.5",
			"pillow":   "11.3.0",
		},
		Entrypoint: "main.py",
		State:      StateDiscovered,
	}

	// Register
	if err := reg.Register(app); err != nil {
		t.Fatalf("failed to register app: %v", err)
	}

	// Get
	retrieved, exists := reg.Get("com.test.myapp")
	if !exists {
		t.Fatal("expected app to exist in registry")
	}
	if retrieved.Name != "My App" {
		t.Errorf("expected name 'My App', got '%s'", retrieved.Name)
	}
	if retrieved.State != StateDiscovered {
		t.Errorf("expected state StateDiscovered, got '%s'", retrieved.State)
	}

	// Update state
	if err := reg.UpdateState("com.test.myapp", StateReady); err != nil {
		t.Fatalf("failed to update state: %v", err)
	}
	retrieved, _ = reg.Get("com.test.myapp")
	if retrieved.State != StateReady {
		t.Errorf("expected state StateReady, got '%s'", retrieved.State)
	}

	// Update last launched
	if err := reg.UpdateLastLaunched("com.test.myapp"); err != nil {
		t.Fatalf("failed to update last launched: %v", err)
	}
	retrieved, _ = reg.Get("com.test.myapp")
	if retrieved.LastLaunchedAt.IsZero() {
		t.Errorf("expected last launched to be set")
	}

	// List
	list := reg.List()
	if len(list) != 1 {
		t.Errorf("expected 1 app in list, got %d", len(list))
	}

	// Test persistence by opening another registry instance pointing to same tmpDir
	reg2, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("failed to reload registry: %v", err)
	}
	appFromDisk, exists := reg2.Get("com.test.myapp")
	if !exists {
		t.Fatal("expected app to persist on disk")
	}
	if appFromDisk.State != StateReady {
		t.Errorf("expected persisted state StateReady, got '%s'", appFromDisk.State)
	}

	// Remove
	if err := reg2.Remove("com.test.myapp"); err != nil {
		t.Fatalf("failed to remove app: %v", err)
	}
	_, exists = reg2.Get("com.test.myapp")
	if exists {
		t.Fatal("expected app to be deleted from registry")
	}
}

func TestRegistryStateTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	app := &ApplicationRecord{
		ApplicationID: "com.state.test",
		Name:          "State Test",
		Version:       "1.0.0",
		PythonVersion: "3.12.0",
		State:         StateDiscovered,
		Dependencies:  map[string]string{},
	}
	_ = reg.Register(app)

	states := []ApplicationState{
		StateInstalling,
		StateInstalled,
		StateReady,
		StateBroken,
		StateRemoving,
	}

	for _, s := range states {
		if err := reg.UpdateState("com.state.test", s); err != nil {
			t.Fatalf("failed to transition to state %s: %v", s, err)
		}
		item, _ := reg.Get("com.state.test")
		if item.State != s {
			t.Errorf("expected state %s, got %s", s, item.State)
		}
	}
}
