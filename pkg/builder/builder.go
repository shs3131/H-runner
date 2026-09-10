package builder

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hrunner/hrunner/pkg/manifest"
	"github.com/hrunner/hrunner/pkg/packages"
)

// BuildConfig defines the build options for generating an application executable.
type BuildConfig struct {
	ProjectPath     string
	OutputPath      string
	ApplicationID   string
	Name            string
	Version         string
	Publisher       string
	PythonVersion   string
	AllowCompatible bool
	Entrypoint      string
	Dependencies    map[string]string
	LauncherStub    string
}

// BuildResult contains statistics about the generated executable.
type BuildResult struct {
	OutputPath string
	SizeBytes  int64
	Duration   time.Duration
}

// ParseRequirementsTxt parses a standard requirements.txt file for pinned dependencies.
func ParseRequirementsTxt(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	deps := make(map[string]string)
	scanner := bufio.NewScanner(f)
	reqPattern := regexp.MustCompile(`^([a-zA-Z0-9_.-]+)\s*==\s*([a-zA-Z0-9_.-]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches := reqPattern.FindStringSubmatch(line)
		if matches != nil {
			pkg := packages.NormalizePackageName(matches[1])
			ver := strings.TrimSpace(matches[2])
			deps[pkg] = ver
		}
	}

	return deps, scanner.Err()
}

// Build packages the Python project and binds it with the launcher stub.
func Build(cfg *BuildConfig) (*BuildResult, error) {
	start := time.Now()

	// 1. Validate project path
	projStat, err := os.Stat(cfg.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("project path does not exist: %w", err)
	}

	var projDir string
	if projStat.IsDir() {
		projDir = cfg.ProjectPath
	} else {
		projDir = filepath.Dir(cfg.ProjectPath)
		if cfg.Entrypoint == "" {
			cfg.Entrypoint = filepath.Base(cfg.ProjectPath)
		}
	}

	// 2. Determine Entrypoint
	if cfg.Entrypoint == "" {
		candidates := []string{"main.py", "app.py", "run.py", "__main__.py"}
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(projDir, c)); err == nil {
				cfg.Entrypoint = c
				break
			}
		}
		if cfg.Entrypoint == "" {
			return nil, fmt.Errorf("could not automatically detect entrypoint (looked for main.py, app.py). Specify --entrypoint")
		}
	}

	// 3. Defaults
	if cfg.Name == "" {
		cfg.Name = filepath.Base(projDir)
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.ApplicationID == "" {
		cleanName := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(cfg.Name, ".")
		cleanName = strings.Trim(cleanName, ".")
		cfg.ApplicationID = fmt.Sprintf("com.hrunner.%s", strings.ToLower(cleanName))
	}
	if cfg.PythonVersion == "" {
		cfg.PythonVersion = "3.13.7"
	}
	if cfg.OutputPath == "" {
		cfg.OutputPath = filepath.Join(".", fmt.Sprintf("%s.exe", cfg.Name))
	}

	// 4. Dependencies
	if cfg.Dependencies == nil {
		cfg.Dependencies = make(map[string]string)
	}
	reqTxt := filepath.Join(projDir, "requirements.txt")
	if _, err := os.Stat(reqTxt); err == nil {
		parsed, err := ParseRequirementsTxt(reqTxt)
		if err == nil {
			for k, v := range parsed {
				if _, exists := cfg.Dependencies[k]; !exists {
					cfg.Dependencies[k] = v
				}
			}
		}
	}

	// 5. Generate Manifest
	mf := &manifest.Manifest{
		FormatVersion: manifest.CurrentFormatVersion,
		ApplicationID: cfg.ApplicationID,
		Name:          cfg.Name,
		Version:       cfg.Version,
		Publisher:     cfg.Publisher,
		Python: manifest.PythonConfig{
			Version:         cfg.PythonVersion,
			AllowCompatible: cfg.AllowCompatible,
		},
		Dependencies: cfg.Dependencies,
		Entrypoint:   cfg.Entrypoint,
	}

	if err := mf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest configuration: %w", err)
	}

	manifestBytes, err := mf.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize manifest: %w", err)
	}

	// 6. Find launcher stub
	stubPath := cfg.LauncherStub
	if stubPath == "" {
		candidates := []string{
			"hlauncher.exe",
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Hrunner", "bin", "hlauncher.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				stubPath = c
				break
			}
		}
		if stubPath == "" {
			return nil, fmt.Errorf("hlauncher.exe stub not found. Specify --launcher-stub or build hlauncher first")
		}
	}

	stubBytes, err := os.ReadFile(stubPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read launcher stub %s: %w", stubPath, err)
	}

	// 7. Create output executable
	outF, err := os.Create(cfg.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", cfg.OutputPath, err)
	}
	defer outF.Close()

	// Write the Go launcher stub PE binary
	if _, err := outF.Write(stubBytes); err != nil {
		return nil, fmt.Errorf("failed to write launcher binary: %w", err)
	}

	// Append application zip archive (PE overlay)
	zw := zip.NewWriter(outF)

	// Add manifest.json
	mw, err := zw.Create("manifest.json")
	if err != nil {
		return nil, err
	}
	if _, err := mw.Write(manifestBytes); err != nil {
		return nil, err
	}

	// Add all Python files in project directory
	err = filepath.Walk(projDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "__pycache__" || name == ".venv" || name == "venv" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".py" && ext != ".pyw" && ext != ".txt" && ext != ".json" {
			return nil
		}

		relPath, err := filepath.Rel(projDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		w, err := zw.Create(relPath)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to package Python files: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize application zip payload: %w", err)
	}

	fi, err := outF.Stat()
	if err != nil {
		return nil, err
	}

	return &BuildResult{
		OutputPath: cfg.OutputPath,
		SizeBytes:  fi.Size(),
		Duration:   time.Since(start),
	}, nil
}
