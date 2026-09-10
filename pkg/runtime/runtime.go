package runtime

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RuntimeInfo describes an installed Python runtime.
type RuntimeInfo struct {
	Version   *Version `json:"version"`
	Path      string   `json:"path"`
	PythonExe string   `json:"python_exe"`
	SizeBytes int64    `json:"size_bytes"`
}

// RuntimeStore manages Python runtimes under %LOCALAPPDATA%\Hrunner\runtimes.
type RuntimeStore struct {
	mu          sync.RWMutex
	runtimesDir string
	cacheDir    string
}

// NewRuntimeStore initializes a RuntimeStore.
func NewRuntimeStore(baseDir string) (*RuntimeStore, error) {
	if baseDir == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			baseDir = filepath.Join(localAppData, "Hrunner")
		} else {
			baseDir = filepath.Join(".", ".hrunner")
		}
	}

	runtimesDir := filepath.Join(baseDir, "runtimes")
	cacheDir := filepath.Join(baseDir, "cache")

	if err := os.MkdirAll(runtimesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create runtimes directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &RuntimeStore{
		runtimesDir: runtimesDir,
		cacheDir:    cacheDir,
	}, nil
}

// List returns all currently installed Python runtimes.
func (s *RuntimeStore) List() ([]*RuntimeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.runtimesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read runtimes directory: %w", err)
	}

	var runtimes []*RuntimeInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "python-") {
			continue
		}
		verStr := strings.TrimPrefix(name, "python-")
		ver, err := ParseVersion(verStr)
		if err != nil {
			continue
		}

		runtimePath := filepath.Join(s.runtimesDir, name)
		pythonExe := filepath.Join(runtimePath, "python.exe")
		if _, err := os.Stat(pythonExe); err != nil {
			continue
		}

		size := calculateDirSize(runtimePath)
		runtimes = append(runtimes, &RuntimeInfo{
			Version:   ver,
			Path:      runtimePath,
			PythonExe: pythonExe,
			SizeBytes: size,
		})
	}

	return runtimes, nil
}

// Find looks for an installed runtime satisfying the requested version.
func (s *RuntimeStore) Find(requestedStr string, allowCompatible bool) (*RuntimeInfo, bool, error) {
	reqVer, err := ParseVersion(requestedStr)
	if err != nil {
		return nil, false, fmt.Errorf("invalid requested python version: %w", err)
	}

	runtimes, err := s.List()
	if err != nil {
		return nil, false, err
	}

	// First pass: exact match
	for _, r := range runtimes {
		if r.Version.Compare(reqVer) == 0 {
			return r, true, nil
		}
	}

	// Second pass: compatible if allowed
	if allowCompatible {
		var best *RuntimeInfo
		for _, r := range runtimes {
			if r.Version.IsCompatible(reqVer, true) {
				if best == nil || r.Version.Compare(best.Version) > 0 {
					best = r
				}
			}
		}
		if best != nil {
			return best, true, nil
		}
	}

	return nil, false, nil
}

// DownloadAndInstall downloads the official Windows embeddable Python archive and installs it atomically.
func (s *RuntimeStore) DownloadAndInstall(versionStr string, progress func(downloaded, total int64, pct float64)) (*RuntimeInfo, error) {
	ver, err := ParseVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version '%s': %w", versionStr, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	targetDirName := fmt.Sprintf("python-%s", ver.String())
	targetDir := filepath.Join(s.runtimesDir, targetDirName)
	pythonExe := filepath.Join(targetDir, "python.exe")

	// If already installed and healthy
	if _, err := os.Stat(pythonExe); err == nil {
		return &RuntimeInfo{
			Version:   ver,
			Path:      targetDir,
			PythonExe: pythonExe,
			SizeBytes: calculateDirSize(targetDir),
		}, nil
	}

	// Official download URL for Windows x64 embeddable python
	zipFileName := fmt.Sprintf("python-%s-embed-amd64.zip", ver.String())
	zipURL := fmt.Sprintf("https://www.python.org/ftp/python/%s/%s", ver.String(), zipFileName)
	cacheZipPath := filepath.Join(s.cacheDir, zipFileName)

	// Download to cache if not already fully downloaded
	if err := s.downloadFile(zipURL, cacheZipPath, progress); err != nil {
		return nil, fmt.Errorf("failed to download Python runtime %s: %w", ver.String(), err)
	}

	// Atomic extraction into temp dir first
	tmpExtractDir := filepath.Join(s.runtimesDir, fmt.Sprintf("tmp_%s_%d", ver.String(), time.Now().UnixNano()))
	if err := os.MkdirAll(tmpExtractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp extraction directory: %w", err)
	}
	defer func() {
		// Clean up temp dir if still exists
		_ = os.RemoveAll(tmpExtractDir)
	}()

	if err := extractZip(cacheZipPath, tmpExtractDir); err != nil {
		_ = os.Remove(cacheZipPath) // discard corrupted cache zip
		return nil, fmt.Errorf("failed to extract Python archive: %w", err)
	}

	// Verify python.exe exists in extracted dir
	extractedExe := filepath.Join(tmpExtractDir, "python.exe")
	if _, err := os.Stat(extractedExe); err != nil {
		return nil, fmt.Errorf("corrupted Python archive: python.exe not found")
	}

	// Commit atomic rename to target directory
	_ = os.RemoveAll(targetDir)
	if err := os.Rename(tmpExtractDir, targetDir); err != nil {
		return nil, fmt.Errorf("failed to move extracted runtime to %s: %w", targetDir, err)
	}

	return &RuntimeInfo{
		Version:   ver,
		Path:      targetDir,
		PythonExe: pythonExe,
		SizeBytes: calculateDirSize(targetDir),
	}, nil
}

func (s *RuntimeStore) downloadFile(url string, destPath string, progress func(downloaded, total int64, pct float64)) error {
	// If file exists and is valid zip, skip download
	if fi, err := os.Stat(destPath); err == nil && fi.Size() > 1000000 {
		zr, err := zip.OpenReader(destPath)
		if err == nil {
			zr.Close()
			if progress != nil {
				progress(fi.Size(), fi.Size(), 100.0)
			}
			return nil
		}
		_ = os.Remove(destPath) // Remove corrupted cached file
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d for %s", resp.StatusCode, url)
	}

	totalBytes := resp.ContentLength
	tmpDest := destPath + ".tmp"
	outFile, err := os.Create(tmpDest)
	if err != nil {
		return err
	}
	defer func() {
		outFile.Close()
		_ = os.Remove(tmpDest)
	}()

	var downloaded int64
	buf := make([]byte, 64*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := outFile.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progress != nil && totalBytes > 0 {
				pct := float64(downloaded) / float64(totalBytes) * 100.0
				progress(downloaded, totalBytes, pct)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	outFile.Close()

	// Verify downloaded zip is valid before committing
	zr, err := zip.OpenReader(tmpDest)
	if err != nil {
		return fmt.Errorf("downloaded file is not a valid zip archive: %w", err)
	}
	zr.Close()

	_ = os.Remove(destPath)
	return os.Rename(tmpDest, destPath)
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
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

// BuildBootstrapScript generates Python code that configures sys.path and executes the app entrypoint.
func BuildBootstrapScript(appDir, entrypointPath string, packagePaths []string) string {
	var pkgLines strings.Builder
	for _, p := range packagePaths {
		escaped := strings.ReplaceAll(p, `\`, `\\`)
		pkgLines.WriteString(fmt.Sprintf(`        %q,`+"\n", escaped))
	}

	escapedAppDir := strings.ReplaceAll(appDir, `\`, `\\`)
	escapedEntrypoint := strings.ReplaceAll(entrypointPath, `\`, `\\`)

	return fmt.Sprintf(`import os
import sys
import runpy

# Ensure logical environment isolation by prepending application-specific package pool paths
package_paths = [
%s]

for path in package_paths:
    if path and os.path.isdir(path):
        if path not in sys.path:
            sys.path.insert(0, path)
        if hasattr(os, 'add_dll_directory'):
            try:
                os.add_dll_directory(path)
            except Exception:
                pass

app_dir = %q
if app_dir and os.path.isdir(app_dir) and app_dir not in sys.path:
    sys.path.insert(0, app_dir)

entrypoint = %q
if __name__ == '__main__':
    sys.argv = [entrypoint] + sys.argv[1:]
    runpy.run_path(entrypoint, run_name='__main__')
`, pkgLines.String(), escapedAppDir, escapedEntrypoint)
}

// RunApplication creates an isolated execution command for the application.
func (s *RuntimeStore) RunApplication(runtime *RuntimeInfo, appDir, entrypoint string, packagePaths []string, extraArgs []string) (*exec.Cmd, error) {
	bootstrapCode := BuildBootstrapScript(appDir, entrypoint, packagePaths)
	args := []string{"-c", bootstrapCode}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	cmd := exec.Command(runtime.PythonExe, args...)
	cmd.Dir = appDir
	return cmd, nil
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
