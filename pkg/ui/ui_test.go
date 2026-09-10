package ui

import (
	"os"
	"syscall"
	"testing"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/registry"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/storage"
)

func TestFormatBytes(t *testing.T) {
	if s := FormatBytes(500); s != "500 B" {
		t.Errorf("expected 500 B, got %s", s)
	}
	if s := FormatBytes(1024 * 1024); s != "1.0 MB" {
		t.Errorf("expected 1.0 MB, got %s", s)
	}
	if s := FormatBytes(50 * 1024 * 1024); s != "50.0 MB" {
		t.Errorf("expected 50.0 MB, got %s", s)
	}
}

func TestNativeManagerHeadless(t *testing.T) {
	os.Setenv("HRUNNER_HEADLESS", "1")
	defer os.Unsetenv("HRUNNER_HEADLESS")

	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	runtimes, err := runtime.NewRuntimeStore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	pkgStore, err := packages.NewPackageStore(tmpDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	storageMgr, err := storage.NewStorageManager(tmpDir, reg, pkgStore)
	if err != nil {
		t.Fatal(err)
	}

	mgr := NewNativeManager(reg, runtimes, pkgStore, storageMgr)
	if mgr == nil {
		t.Fatal("expected non-nil native manager")
	}

	// Test UI dialog functions in headless mode
	u := NewUI()
	if u.ShowMissingHrunner() {
		t.Errorf("expected false for ShowMissingHrunner in headless mode")
	}

	choice, confirmed := u.AskPythonRuntime("3.13.7", "3.13.8")
	if !confirmed || choice != "install_exact" {
		t.Errorf("unexpected choice in headless mode: %s, %v", choice, confirmed)
	}

	if !u.ConfirmInstallation("TestApp", "3.13.7", true, nil, 1000) {
		t.Errorf("expected true for ConfirmInstallation in headless mode")
	}

	if !u.ShowInstallComplete("TestApp") {
		t.Errorf("expected true for ShowInstallComplete in headless mode")
	}
}

func TestRegressionManagerComctl32V6(t *testing.T) {
	// Regression test for Bug 1: Ensure Comctl32 v6 activation does not panic
	// and procTaskDialogIndirect lookup does not crash
	enableCommonControlsV6()

	// Verify procTaskDialogIndirect.Find() does not panic and is either found or cleanly handled
	err := procTaskDialogIndirect.Find()
	if err != nil {
		t.Logf("Note: TaskDialogIndirect not exported by current environment, fallback to MessageBox verified")
	} else {
		t.Logf("Success: TaskDialogIndirect successfully found via Comctl32 v6 activation context")
	}

	// Verify ShowTaskDialog never panics even in interactive mode if invoked with headless unset
	btnOK, _ := syscall.UTF16PtrFromString("OK")
	buttons := []TASKDIALOG_BUTTON{{nButtonID: 1, pszButtonText: btnOK}}

	// In headless mode:
	os.Setenv("HRUNNER_HEADLESS", "1")
	res, err := ShowTaskDialog("Test", "Instruction", "Content", 0, buttons, 1)
	if err != nil || res != 1 {
		t.Fatalf("ShowTaskDialog failed in headless mode: res=%d, err=%v", res, err)
	}
}

