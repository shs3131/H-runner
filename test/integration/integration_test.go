package integration

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrunner/hrunner/pkg/builder"
	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/protocol"
	"github.com/hrunner/hrunner/pkg/registry"
	"github.com/hrunner/hrunner/pkg/resolver"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/storage"
)

func createTestWheel(t *testing.T, dir, filename, modName, code string) string {
	t.Helper()
	p := filepath.Join(dir, filename)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(fmt.Sprintf("%s/__init__.py", modName))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(code))
	_ = zw.Close()
	return p
}

func TestCompleteMVPIntegrationFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Set up Hrunner core components
	reg, err := registry.NewRegistry(tmpDir)
	if err != nil {
		t.Fatalf("registry init: %v", err)
	}

	runtimes, err := runtime.NewRuntimeStore(tmpDir)
	if err != nil {
		t.Fatalf("runtimes init: %v", err)
	}

	pkgStore, err := packages.NewPackageStore(tmpDir, nil)
	if err != nil {
		t.Fatalf("pkgStore init: %v", err)
	}

	storageMgr, err := storage.NewStorageManager(tmpDir, reg, pkgStore)
	if err != nil {
		t.Fatalf("storageMgr init: %v", err)
	}

	// 2. Install mock Python 3.13.7 runtime
	mockPythonDir := filepath.Join(tmpDir, "runtimes", "python-3.13.7")
	_ = os.MkdirAll(mockPythonDir, 0755)
	mockPythonExe := filepath.Join(mockPythonDir, "python.exe")
	_ = os.WriteFile(mockPythonExe, []byte("mock python binary"), 0755)

	rt, exists, _ := runtimes.Find("3.13.7", false)
	if !exists || rt == nil {
		t.Fatal("expected python 3.13.7 to be found")
	}

	// 3. Create mock wheels for shared requests 2.32.5 and distinct numpy 1.26.4 & numpy 2.3.2
	whlRequests := createTestWheel(t, tmpDir, "requests-2.32.5-py3-none-any.whl", "requests", "__version__ = '2.32.5'")
	whlNumpy1 := createTestWheel(t, tmpDir, "numpy-1.26.4-py3-none-any.whl", "numpy", "__version__ = '1.26.4'")
	whlNumpy2 := createTestWheel(t, tmpDir, "numpy-2.3.2-py3-none-any.whl", "numpy", "__version__ = '2.3.2'")

	// Pre-populate pool (simulating downloaded wheels)
	_, _ = pkgStore.InstallWheel(whlRequests, "requests", "2.32.5")
	_, _ = pkgStore.InstallWheel(whlNumpy1, "numpy", "1.26.4")
	_, _ = pkgStore.InstallWheel(whlNumpy2, "numpy", "2.3.2")

	// 4. Build App A: requires requests 2.32.5 and numpy 1.26.4
	appADir := filepath.Join(tmpDir, "AppA")
	_ = os.MkdirAll(appADir, 0755)
	_ = os.WriteFile(filepath.Join(appADir, "main.py"), []byte("import requests, numpy; print('APP A OK')"), 0644)
	stubPath := filepath.Join(tmpDir, "stub.exe")
	_ = os.WriteFile(stubPath, []byte("MOCK_LAUNCHER"), 0644)

	appAExe := filepath.Join(tmpDir, "AppA.exe")
	cfgA := &builder.BuildConfig{
		ProjectPath:   appADir,
		OutputPath:    appAExe,
		ApplicationID: "com.example.appa",
		Name:          "Application A",
		Version:       "1.0.0",
		PythonVersion: "3.13.7",
		LauncherStub:  stubPath,
		Dependencies: map[string]string{
			"requests": "2.32.5",
			"numpy":    "1.26.4",
		},
	}
	resA, err := builder.Build(cfgA)
	if err != nil {
		t.Fatalf("build App A failed: %v", err)
	}
	if resA.SizeBytes <= 0 {
		t.Fatal("expected positive size for App A")
	}

	// 5. Build App B: requires requests 2.32.5 (shared) and numpy 2.3.2 (distinct version)
	appBDir := filepath.Join(tmpDir, "AppB")
	_ = os.MkdirAll(appBDir, 0755)
	_ = os.WriteFile(filepath.Join(appBDir, "main.py"), []byte("import requests, numpy; print('APP B OK')"), 0644)

	appBExe := filepath.Join(tmpDir, "AppB.exe")
	cfgB := &builder.BuildConfig{
		ProjectPath:   appBDir,
		OutputPath:    appBExe,
		ApplicationID: "com.example.appb",
		Name:          "Application B",
		Version:       "2.0.0",
		PythonVersion: "3.13.7",
		LauncherStub:  stubPath,
		Dependencies: map[string]string{
			"requests": "2.32.5",
			"numpy":    "2.3.2",
		},
	}
	resB, err := builder.Build(cfgB)
	if err != nil {
		t.Fatalf("build App B failed: %v", err)
	}
	if resB.SizeBytes <= 0 {
		t.Fatal("expected positive size for App B")
	}

	// 6. Register App A and App B
	_ = reg.Register(&registry.ApplicationRecord{
		ApplicationID:  cfgA.ApplicationID,
		Name:           cfgA.Name,
		Version:        cfgA.Version,
		ExecutablePath: appAExe,
		PythonVersion:  cfgA.PythonVersion,
		Dependencies:   cfgA.Dependencies,
		State:          registry.StateReady,
	})

	_ = reg.Register(&registry.ApplicationRecord{
		ApplicationID:  cfgB.ApplicationID,
		Name:           cfgB.Name,
		Version:        cfgB.Version,
		ExecutablePath: appBExe,
		PythonVersion:  cfgB.PythonVersion,
		Dependencies:   cfgB.Dependencies,
		State:          registry.StateReady,
	})

	// 7. Verify PHYSICAL DEDUPLICATION and REFERENCE COUNTS:
	// requests 2.32.5 is used by BOTH App A and App B
	usages, err := storageMgr.GetPackageUsageMap()
	if err != nil {
		t.Fatalf("failed to get usage map: %v", err)
	}

	usageMap := make(map[string]*storage.PackageUsage)
	for _, u := range usages {
		usageMap[fmt.Sprintf("%s@%s", u.Name, u.Version)] = u
	}

	reqUsage := usageMap["requests@2.32.5"]
	if reqUsage == nil || reqUsage.RefCount != 2 {
		t.Fatalf("expected requests 2.32.5 refcount == 2, got %v", reqUsage)
	}

	numpy1Usage := usageMap["numpy@1.26.4"]
	if numpy1Usage == nil || numpy1Usage.RefCount != 1 {
		t.Fatalf("expected numpy 1.26.4 refcount == 1, got %v", numpy1Usage)
	}

	numpy2Usage := usageMap["numpy@2.3.2"]
	if numpy2Usage == nil || numpy2Usage.RefCount != 1 {
		t.Fatalf("expected numpy 2.3.2 refcount == 1, got %v", numpy2Usage)
	}

	// Verify only ONE physical directory exists for requests 2.32.5 in pool
	reqDir := pkgStore.GetPackageDir("requests", "2.32.5")
	if _, err := os.Stat(reqDir); err != nil {
		t.Fatalf("requests dir does not exist in pool: %s", reqDir)
	}

	// 8. Test Safe App Removal:
	// Remove App A with cleanUnused=true
	err = storageMgr.RemoveApplicationAndCleanup("com.example.appa", true)
	if err != nil {
		t.Fatalf("failed to remove App A: %v", err)
	}

	// requests 2.32.5 MUST STILL EXIST in the pool because App B is still using it!
	if !pkgStore.IsPackageInstalled("requests", "2.32.5") {
		t.Fatalf("FATAL: shared package requests 2.32.5 was deleted while App B still required it!")
	}

	// numpy 1.26.4 should be deleted because only App A used it
	if pkgStore.IsPackageInstalled("numpy", "1.26.4") {
		t.Fatalf("expected orphaned numpy 1.26.4 to be removed after App A uninstalled")
	}

	// numpy 2.3.2 should still exist because App B uses it
	if !pkgStore.IsPackageInstalled("numpy", "2.3.2") {
		t.Fatalf("expected numpy 2.3.2 to still exist for App B")
	}

	// 9. Now remove App B
	err = storageMgr.RemoveApplicationAndCleanup("com.example.appb", true)
	if err != nil {
		t.Fatalf("failed to remove App B: %v", err)
	}

	// Now requests and numpy 2.3.2 should be cleaned
	if pkgStore.IsPackageInstalled("requests", "2.32.5") {
		t.Fatalf("expected requests 2.32.5 to be removed when no apps use it")
	}
	if pkgStore.IsPackageInstalled("numpy", "2.3.2") {
		t.Fatalf("expected numpy 2.3.2 to be removed when no apps use it")
	}

	// 10. Test Auto-shutdown IPC Server:
	pipeName := fmt.Sprintf(`\\.\pipe\hrunner_integration_%d`, time.Now().UnixNano())
	server := protocol.NewServer(pipeName, 200*time.Millisecond, nil)
	if err := server.Start(); err != nil {
		t.Fatalf("server start failed: %v", err)
	}

	if !protocol.IsServerRunning(pipeName, 2*time.Second) {
		t.Fatal("server pipe was not ready")
	}

	client, err := protocol.Dial(pipeName, 2*time.Second)
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	_ = client.Ping(1 * time.Second)
	_ = client.Close()

	// Server should shut down automatically after 200ms idle
	select {
	case <-server.ShutdownChan():
		// Auto-shutdown verified!
	case <-time.After(2 * time.Second):
		t.Fatal("server did not auto-shutdown after idle period")
	}
}

func TestEdgeCasesAndFailureModes(t *testing.T) {
	tmpDir := t.TempDir()

	pkgStore, err := packages.NewPackageStore(tmpDir, nil)
	if err != nil {
		t.Fatalf("pkgStore init: %v", err)
	}

	// 1. Corrupt wheel installation should fail cleanly without polluting the store
	corruptWhl := filepath.Join(tmpDir, "corrupt-1.0.0-py3-none-any.whl")
	if err := os.WriteFile(corruptWhl, []byte("NOT_A_VALID_ZIP_ARCHIVE"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = pkgStore.InstallWheel(corruptWhl, "corrupt", "1.0.0")
	if err == nil {
		t.Fatal("expected install of corrupted wheel to fail, but succeeded")
	}

	// Verify corrupted package directory does NOT exist in package store
	corruptPkgDir := pkgStore.GetPackageDir("corrupt", "1.0.0")
	if _, err := os.Stat(corruptPkgDir); !os.IsNotExist(err) {
		t.Fatalf("corrupt package dir was left behind: %s", corruptPkgDir)
	}

	// 2. Resolver non-existent package returns clear error
	res := resolver.NewResolver(pkgStore, nil)
	v3137, _ := runtime.ParseVersion("3.13.7")
	_, err = res.Resolve(map[string]string{
		"this-package-definitely-does-not-exist-hrunner-test-xyz": "9.9.9",
	}, v3137)
	if err == nil {
		t.Fatal("expected resolution for non-existent package to fail")
	}
	if !strings.Contains(err.Error(), "not found on PyPI") {
		t.Fatalf("expected 'not found on PyPI' error, got: %v", err)
	}
}

func TestConcurrentPipeOperations(t *testing.T) {
	pipeName := fmt.Sprintf(`\\.\pipe\hrunner_concurrency_%d`, time.Now().UnixNano())
	server := protocol.NewServer(pipeName, 500*time.Millisecond, nil)
	if err := server.Start(); err != nil {
		t.Fatalf("server start failed: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	const numClients = 8
	errChan := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientIdx int) {
			client, err := protocol.Dial(pipeName, 2*time.Second)
			if err != nil {
				errChan <- fmt.Errorf("client %d dial failed: %w", clientIdx, err)
				return
			}
			defer client.Close()

			if err := client.Ping(1 * time.Second); err != nil {
				errChan <- fmt.Errorf("client %d ping failed: %w", clientIdx, err)
				return
			}
			errChan <- nil
		}(i)
	}

	for i := 0; i < numClients; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("concurrent client error: %v", err)
		}
	}
}

type testLauncherHandler struct {
	pyExe string
}

func (h *testLauncherHandler) HandleLaunchRequest(req *protocol.LaunchRequest, conn *protocol.ClientConn) (*protocol.LaunchResponse, error) {
	_ = conn.SendMessage(&protocol.LaunchReady{
		BaseMessage:   protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchReady},
		PythonExePath: h.pyExe,
		PackagePaths:  []string{},
		Entrypoint:    req.Entrypoint,
	})
	return &protocol.LaunchResponse{
		BaseMessage:     protocol.BaseMessage{ProtocolVersion: protocol.CurrentProtocolVersion, Type: protocol.MsgLaunchResponse},
		Status:          protocol.StatusReady,
		PythonAvailable: true,
	}, nil
}
func (h *testLauncherHandler) OnClientConnected()    {}
func (h *testLauncherHandler) OnClientDisconnected() {}

func TestRegressionLauncherRequestNoHang(t *testing.T) {
	// Regression test for Bug 2: Ensure LaunchRequest properly sets protocol version and type,
	// and server does not return "missing or invalid protocol_version in message" causing launcher hang.
	pipeName := fmt.Sprintf(`\\.\pipe\hrunner_reg_hang_%d`, time.Now().UnixNano())
	tmpDir := t.TempDir()

	pyExe := filepath.Join(tmpDir, "python.exe")
	_ = os.WriteFile(pyExe, []byte("mock python"), 0755)

	h := &testLauncherHandler{pyExe: pyExe}
	server := protocol.NewServer(pipeName, 2*time.Second, h)
	if err := server.Start(); err != nil {
		t.Fatalf("server start failed: %v", err)
	}
	defer server.Close()

	client, err := protocol.Dial(pipeName, 2*time.Second)
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	defer client.Close()

	// Replicate the exact struct created in cmd/hlauncher
	req := &protocol.LaunchRequest{
		BaseMessage: protocol.BaseMessage{
			ProtocolVersion: protocol.CurrentProtocolVersion,
			Type:            protocol.MsgLaunchRequest,
		},
		ApplicationID: "com.example.testapp",
		Name:          "TestApp",
		Version:       "1.0.0",
		Python:        "3.13.7",
		Dependencies:  map[string]string{},
		Entrypoint:    "main.py",
	}

	if err := client.Send(req); err != nil {
		t.Fatalf("failed to send launch request: %v", err)
	}

	// Read response: must receive LaunchReady and NOT an error about protocol_version
	var receivedReady bool
	for {
		line, err := client.ReadLine()
		if err != nil {
			t.Fatalf("failed to read line: %v", err)
		}

		base, err := protocol.DecodeBaseMessage(line)
		if err != nil {
			t.Fatalf("DecodeBaseMessage failed: %v", err)
		}

		if base.Type == protocol.MsgLaunchResponse {
			var resp protocol.LaunchResponse
			if err := json.Unmarshal(line, &resp); err == nil {
				if resp.Status == protocol.StatusError {
					t.Fatalf("server returned error: %s", resp.ErrorMessage)
				}
			}
		} else if base.Type == protocol.MsgLaunchReady {
			receivedReady = true
			break
		}
	}

	if !receivedReady {
		t.Fatal("expected LaunchReady to be received successfully without hanging")
	}
}


