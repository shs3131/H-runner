package packages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hrunner/hrunner/pkg/runtime"
)

// PyPIClient fetches package metadata and wheels from PyPI.
type PyPIClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewPyPIClient creates a new PyPI API client.
func NewPyPIClient(baseURL string) *PyPIClient {
	if baseURL == "" {
		baseURL = "https://pypi.org/pypi"
	}
	return &PyPIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}
}

type pypiReleaseFile struct {
	Filename    string            `json:"filename"`
	Packagetype string            `json:"packagetype"`
	URL         string            `json:"url"`
	Digests     map[string]string `json:"digests"`
	Size        int64             `json:"size"`
	PythonVer   string            `json:"python_version"`
}

type pypiResponse struct {
	Info struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Summary     string `json:"summary"`
		RequiresDist []string `json:"requires_dist"`
	} `json:"info"`
	URLs []pypiReleaseFile `json:"urls"`
}

// PackageMetadata holds parsed release information.
type PackageMetadata struct {
	Name         string
	Version      string
	RequiresDist []string
	Wheels       []*WheelInfo
}

// FetchMetadata retrieves package information from PyPI for a specific version (or latest).
func (c *PyPIClient) FetchMetadata(pkgName, version string) (*PackageMetadata, error) {
	pkgName = strings.TrimSpace(pkgName)
	version = strings.TrimSpace(version)

	var url string
	if version != "" {
		url = fmt.Sprintf("%s/%s/%s/json", c.baseURL, pkgName, version)
	} else {
		url = fmt.Sprintf("%s/%s/json", c.baseURL, pkgName)
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query PyPI for %s: %w", pkgName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package '%s' (version '%s') not found on PyPI", pkgName, version)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PyPI returned HTTP %d for %s", resp.StatusCode, pkgName)
	}

	var data pypiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode PyPI response: %w", err)
	}

	meta := &PackageMetadata{
		Name:         data.Info.Name,
		Version:      data.Info.Version,
		RequiresDist: data.Info.RequiresDist,
	}

	for _, u := range data.URLs {
		if u.Packagetype != "bdist_wheel" {
			continue
		}
		w, err := ParseWheelFilename(u.Filename)
		if err != nil {
			continue
		}
		w.URL = u.URL
		w.SizeBytes = u.Size
		if sha, ok := u.Digests["sha256"]; ok {
			w.SHA256 = sha
		}
		meta.Wheels = append(meta.Wheels, w)
	}

	return meta, nil
}

// SelectBestWheel finds the most compatible Windows x64 wheel for the given Python version.
func (c *PyPIClient) SelectBestWheel(meta *PackageMetadata, pyVer *runtime.Version) (*WheelInfo, error) {
	var compatible []*WheelInfo
	for _, w := range meta.Wheels {
		if w.IsCompatibleWindowsX64(pyVer) {
			compatible = append(compatible, w)
		}
	}

	if len(compatible) == 0 {
		return nil, fmt.Errorf("no compatible Windows x64 wheel found for %s %s (Python %s)", meta.Name, meta.Version, pyVer.String())
	}

	// Sort by highest score
	sort.Slice(compatible, func(i, j int) bool {
		return compatible[i].WheelScore(pyVer) > compatible[j].WheelScore(pyVer)
	})

	return compatible[0], nil
}

// DownloadWheel downloads a wheel and validates its SHA-256 digest.
func (c *PyPIClient) DownloadWheel(w *WheelInfo, destPath string, progress func(downloaded, total int64, pct float64)) error {
	// If already cached and valid
	if fi, err := os.Stat(destPath); err == nil && fi.Size() == w.SizeBytes {
		// Verify checksum of cached file
		if checkFileSHA256(destPath, w.SHA256) {
			if progress != nil {
				progress(fi.Size(), fi.Size(), 100.0)
			}
			return nil
		}
		_ = os.Remove(destPath)
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", w.URL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download wheel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d for wheel download", resp.StatusCode)
	}

	tmpDest := destPath + ".tmp"
	outFile, err := os.Create(tmpDest)
	if err != nil {
		return err
	}
	defer func() {
		outFile.Close()
		_ = os.Remove(tmpDest)
	}()

	hasher := sha256.New()
	writer := io.MultiWriter(outFile, hasher)

	var downloaded int64
	totalBytes := w.SizeBytes
	if totalBytes <= 0 {
		totalBytes = resp.ContentLength
	}

	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := writer.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progress != nil && totalBytes > 0 {
				pct := float64(downloaded) / float64(totalBytes) * 100.0
				progress(downloaded, totalBytes, pct)
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return rerr
		}
	}

	outFile.Close()

	// Verify SHA-256
	if w.SHA256 != "" {
		calculatedHex := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(calculatedHex, w.SHA256) {
			return fmt.Errorf("SHA-256 mismatch for %s: expected %s, got %s", w.Filename, w.SHA256, calculatedHex)
		}
	}

	_ = os.Remove(destPath)
	return os.Rename(tmpDest, destPath)
}

func checkFileSHA256(filePath, expectedSHA string) bool {
	if expectedSHA == "" {
		return true
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expectedSHA)
}
