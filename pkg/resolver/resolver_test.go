package resolver

import (
	"archive/zip"
	"os"
	"strings"
	"testing"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/runtime"
)

func TestParseRequiresDist(t *testing.T) {
	tests := []struct {
		input       string
		expectedPkg string
		expectedVer string
		expectedOK  bool
	}{
		{
			input:       "urllib3 (<3,>=1.21.1)",
			expectedPkg: "urllib3",
			expectedVer: "<3,>=1.21.1",
			expectedOK:  true,
		},
		{
			input:       "certifi >=2017.4.17",
			expectedPkg: "certifi",
			expectedVer: ">=2017.4.17",
			expectedOK:  true,
		},
		{
			input:       "Pillow == 11.3.0",
			expectedPkg: "pillow",
			expectedVer: "== 11.3.0",
			expectedOK:  true,
		},
		{
			// Optional extra should be skipped
			input:      "socks ; extra == 'socks'",
			expectedOK: false,
		},
		{
			// Non-windows platform should be skipped
			input:      "appnope ; sys_platform == 'darwin'",
			expectedOK: false,
		},
		{
			// Windows platform should be accepted
			input:       "win-inet-pton ; sys_platform == 'win32'",
			expectedPkg: "win-inet-pton",
			expectedOK:  true,
		},
	}

	for _, tc := range tests {
		pkg, ver, ok := ParseRequiresDist(tc.input)
		if ok != tc.expectedOK {
			t.Errorf("input '%s': expected ok=%v, got %v", tc.input, tc.expectedOK, ok)
			continue
		}
		if ok {
			if pkg != tc.expectedPkg {
				t.Errorf("input '%s': expected pkg %s, got %s", tc.input, tc.expectedPkg, pkg)
			}
			if tc.expectedVer != "" && !strings.Contains(ver, strings.TrimSpace(tc.expectedVer)) {
				t.Errorf("input '%s': expected ver %s, got %s", tc.input, tc.expectedVer, ver)
			}
		}
	}
}

func TestResolverLocalAndConflict(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := packages.NewPackageStore(tmpDir, nil)
	r := NewResolver(store, nil)
	pyVer, _ := runtime.ParseVersion("3.13.7")

	// Pre-install a package in store
	targetDir := store.GetPackageDir("requests", "2.32.5")
	_, _ = store.InstallWheel(createTestWheel(t, tmpDir, "requests-2.32.5-py3-none-any.whl"), "requests", "2.32.5")

	if !store.IsPackageInstalled("requests", "2.32.5") {
		t.Fatalf("expected requests 2.32.5 installed in %s", targetDir)
	}

	plan, err := r.Resolve(map[string]string{
		"requests": "2.32.5",
	}, pyVer)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if plan.AllPackages["requests"] != "2.32.5" {
		t.Errorf("expected requests 2.32.5 in plan, got %s", plan.AllPackages["requests"])
	}
}

func createTestWheel(t *testing.T, dir, filename string) string {
	t.Helper()
	whlPath := dir + "/" + filename
	f, err := os.Create(whlPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, _ := zw.Create("pkg/__init__.py")
	_, _ = w.Write([]byte("# mock"))
	_ = zw.Close()
	return whlPath
}
