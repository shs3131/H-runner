package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hrunner/hrunner/pkg/manifest"
	"github.com/hrunner/hrunner/pkg/protocol"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/ui"
	winreg "golang.org/x/sys/windows/registry"
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to locate executable: %v\n", err)
		os.Exit(1)
	}

	// 1. Read embedded or appended zip payload
	zr, err := openPayloadZip(exePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load application payload: %v\n", err)
		os.Exit(1)
	}
	defer zr.Close()

	// 2. Extract manifest.json
	manifestBytes, err := readFileFromZip(zr, "manifest.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Missing manifest.json in application payload: %v\n", err)
		os.Exit(1)
	}

	appManifest, err := manifest.Parse(manifestBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid application manifest: %v\n", err)
		os.Exit(1)
	}

	// 3. Extract python files to local application runtime cache
	appCacheDir := filepath.Join(os.TempDir(), fmt.Sprintf("hrunner_app_%s_%s", appManifest.ApplicationID, appManifest.Version))
	if err := extractPayloadFiles(zr, appCacheDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract application files: %v\n", err)
		os.Exit(1)
	}

	// 4. Check if Hrunner is running, or start it
	if !protocol.IsServerRunning(protocol.DefaultPipeName, 200*time.Millisecond) {
		hrunnerExe, found := findHrunnerExe(exePath)
		if !found {
			// Show Hrunner is required dialog
			u := ui.NewUI()
			install := u.ShowMissingHrunner()
			if install {
				// Open Hrunner website / repo
				_ = exec.Command("cmd", "/c", "start", "https://github.com/hrunner/hrunner").Start()
			}
			os.Exit(1)
		}

		// Spawn Hrunner in pipe-server mode
		cmd := exec.Command(hrunnerExe, "--pipe-server")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to launch Hrunner: %v\n", err)
			os.Exit(1)
		}

		// Wait for pipe to be ready (up to 5 seconds)
		ready := false
		for i := 0; i < 50; i++ {
			time.Sleep(100 * time.Millisecond)
			if protocol.IsServerRunning(protocol.DefaultPipeName, 100*time.Millisecond) {
				ready = true
				break
			}
		}

		if !ready {
			fmt.Fprintf(os.Stderr, "Timed out waiting for Hrunner IPC pipe\n")
			os.Exit(1)
		}
	}

	// 5. Connect to Hrunner pipe
	client, err := protocol.Dial(protocol.DefaultPipeName, 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to Hrunner: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// 6. Send launch request
	req := &protocol.LaunchRequest{
		BaseMessage: protocol.BaseMessage{
			ProtocolVersion: protocol.CurrentProtocolVersion,
			Type:            protocol.MsgLaunchRequest,
		},
		ApplicationID:         appManifest.ApplicationID,
		Name:                  appManifest.Name,
		Version:               appManifest.Version,
		Python:                appManifest.Python.Version,
		AllowCompatiblePython: appManifest.Python.AllowCompatible,
		Dependencies:          appManifest.Dependencies,
		Entrypoint:            appManifest.Entrypoint,
		ExecutablePath:        exePath,
	}

	if err := client.Send(req); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send launch request: %v\n", err)
		os.Exit(1)
	}

	// 7. Await LaunchReady from Hrunner
	var readyInfo *protocol.LaunchReady
	for {
		line, err := client.ReadLine()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error receiving Hrunner message: %v\n", err)
			os.Exit(1)
		}

		base, err := protocol.DecodeBaseMessage(line)
		if err != nil {
			continue
		}

		if base.Type == protocol.MsgLaunchResponse {
			var resp protocol.LaunchResponse
			if err := jsonUnmarshal(line, &resp); err == nil {
				if resp.Status == protocol.StatusError {
					fmt.Fprintf(os.Stderr, "Launch failed: %s\n", resp.ErrorMessage)
					os.Exit(1)
				}
			}
		} else if base.Type == protocol.MsgLaunchReady {
			var r protocol.LaunchReady
			if err := jsonUnmarshal(line, &r); err == nil {
				readyInfo = &r
				break
			}
		}
	}

	if readyInfo == nil {
		fmt.Fprintf(os.Stderr, "Launch cancelled or failed\n")
		os.Exit(1)
	}

	// 8. Execute isolated Python environment
	entrypointPath := filepath.Join(appCacheDir, readyInfo.Entrypoint)
	bootstrapCode := runtime.BuildBootstrapScript(appCacheDir, entrypointPath, readyInfo.PackagePaths)

	pyArgs := []string{"-c", bootstrapCode}
	if len(os.Args) > 1 {
		pyArgs = append(pyArgs, os.Args[1:]...)
	}
	cmd := exec.Command(readyInfo.PythonExePath, pyArgs...)
	cmd.Dir = appCacheDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func openPayloadZip(exePath string) (*zip.ReadCloser, error) {
	// 1. Try opening appended zip from exe overlay
	zr, err := zip.OpenReader(exePath)
	if err == nil {
		// Check if it has manifest.json
		for _, f := range zr.File {
			if f.Name == "manifest.json" {
				return zr, nil
			}
		}
		zr.Close()
	}

	return nil, fmt.Errorf("no application payload found in %s", exePath)
}

func readFileFromZip(zr *zip.ReadCloser, filename string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == filename {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file %s not found in payload", filename)
}

func extractPayloadFiles(zr *zip.ReadCloser, targetDir string) error {
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			continue
		}
		fpath := filepath.Join(targetDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
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

func findHrunnerExe(currentExe string) (string, bool) {
	// 1. Check Windows Registry HKCU\Software\Hrunner\InstallPath (written by NSIS installer)
	if k, err := winreg.OpenKey(winreg.CURRENT_USER, `Software\Hrunner`, winreg.QUERY_VALUE); err == nil {
		if val, _, err := k.GetStringValue("InstallPath"); err == nil && val != "" {
			k.Close()
			candidate := filepath.Join(val, "hrunner.exe")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, true
			}
		}
		k.Close()
	}

	// 2. Check Windows App Paths registration
	if k, err := winreg.OpenKey(winreg.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\App Paths\hrunner.exe`, winreg.QUERY_VALUE); err == nil {
		if val, _, err := k.GetStringValue(""); err == nil && val != "" {
			k.Close()
			if _, err := os.Stat(val); err == nil {
				return val, true
			}
		}
		k.Close()
	}

	// 3. Check official NSIS user install path: %LOCALAPPDATA%\Programs\Hrunner\hrunner.exe
	localApp := os.Getenv("LOCALAPPDATA")
	if localApp != "" {
		nsisPath := filepath.Join(localApp, "Programs", "Hrunner", "hrunner.exe")
		if _, err := os.Stat(nsisPath); err == nil {
			return nsisPath, true
		}
		p := filepath.Join(localApp, "Hrunner", "bin", "hrunner.exe")
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}

	// 4. Adjacent to app exe
	adj := filepath.Join(filepath.Dir(currentExe), "hrunner.exe")
	if _, err := os.Stat(adj); err == nil {
		return adj, true
	}

	// 5. In PATH
	if p, err := exec.LookPath("hrunner.exe"); err == nil {
		return p, true
	}

	// 6. Current working directory
	if cwd, err := os.Getwd(); err == nil {
		p := filepath.Join(cwd, "hrunner.exe")
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}

	return "", false
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
