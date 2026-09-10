package builder

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrunner/hrunner/pkg/manifest"
)

func TestParseRequirementsTxt(t *testing.T) {
	tmpDir := t.TempDir()
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	content := `# Comments should be ignored
requests == 2.32.5
pillow==11.3.0
numpy == 2.3.2
`
	if err := os.WriteFile(reqPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := ParseRequirementsTxt(reqPath)
	if err != nil {
		t.Fatalf("failed to parse requirements.txt: %v", err)
	}

	if len(deps) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(deps))
	}
	if deps["requests"] != "2.32.5" {
		t.Errorf("expected requests 2.32.5, got %s", deps["requests"])
	}
	if deps["pillow"] != "11.3.0" {
		t.Errorf("expected pillow 11.3.0, got %s", deps["pillow"])
	}
}

func TestBuilderBuild(t *testing.T) {
	tmpDir := t.TempDir()

	// Create sample python project
	projDir := filepath.Join(tmpDir, "sample_project")
	_ = os.MkdirAll(projDir, 0755)
	mainPy := filepath.Join(projDir, "main.py")
	_ = os.WriteFile(mainPy, []byte("print('HELLO FROM TEST APP')"), 0644)
	reqTxt := filepath.Join(projDir, "requirements.txt")
	_ = os.WriteFile(reqTxt, []byte("requests == 2.32.5\n"), 0644)

	// Create dummy launcher stub
	stubPath := filepath.Join(tmpDir, "dummy_launcher.exe")
	_ = os.WriteFile(stubPath, []byte("DUMMY_PE_HEADER_AND_CODE"), 0644)

	outExe := filepath.Join(tmpDir, "SampleApp.exe")

	cfg := &BuildConfig{
		ProjectPath:   projDir,
		OutputPath:    outExe,
		Name:          "SampleApp",
		Version:       "1.0.0",
		ApplicationID: "com.test.sampleapp",
		PythonVersion: "3.13.7",
		LauncherStub:  stubPath,
	}

	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if res.SizeBytes <= 0 {
		t.Errorf("expected positive executable size")
	}

	// Verify the output executable can be read as a zip archive (PE overlay)
	zr, err := zip.OpenReader(outExe)
	if err != nil {
		t.Fatalf("failed to open generated exe as zip: %v", err)
	}
	defer zr.Close()

	foundManifest := false
	foundMain := false
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			foundManifest = true
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b := make([]byte, f.UncompressedSize64)
			_, _ = io.ReadFull(rc, b)
			_ = rc.Close()
			m, err := manifest.Parse(b)
			if err != nil {
				t.Fatalf("failed to parse manifest from zip: %v", err)
			}
			if m.ApplicationID != "com.test.sampleapp" {
				t.Errorf("expected app id com.test.sampleapp, got %s", m.ApplicationID)
			}
		}
		if f.Name == "main.py" {
			foundMain = true
		}
	}

	if !foundManifest {
		t.Errorf("manifest.json not found in generated exe payload")
	}
	if !foundMain {
		t.Errorf("main.py not found in generated exe payload")
	}
}
