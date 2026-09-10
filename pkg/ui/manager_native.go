package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/registry"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/storage"
)

const (
	BtnNavApps     = 2001
	BtnNavPackages = 2002
	BtnNavRuntimes = 2003
	BtnNavStorage  = 2004
	BtnNavExit     = 2005

	BtnActionLaunch = 3001
	BtnActionRemove = 3002
	BtnActionClean  = 3003
	BtnActionBack   = 3004
)

// NativeManager provides the 100% native Windows GUI for Hrunner Manager.
// It uses native Windows TaskDialogs with Command Links and Common Controls.
// It NEVER opens a web browser or localhost HTTP server.
type NativeManager struct {
	reg        *registry.RegistryStore
	runtimes   *runtime.RuntimeStore
	pkgStore   *packages.PackageStore
	storageMgr *storage.StorageManager
}

// NewNativeManager creates a new native Windows manager instance.
func NewNativeManager(reg *registry.RegistryStore, runtimes *runtime.RuntimeStore, pkgStore *packages.PackageStore, storageMgr *storage.StorageManager) *NativeManager {
	return &NativeManager{
		reg:        reg,
		runtimes:   runtimes,
		pkgStore:   pkgStore,
		storageMgr: storageMgr,
	}
}

// RunInteractiveLoop presents the native Windows Manager UI.
func (m *NativeManager) RunInteractiveLoop() {
	for {
		apps := m.reg.List()
		rts, _ := m.runtimes.List()
		usages, _ := m.storageMgr.GetPackageUsageMap()
		sum, _ := m.storageMgr.GetSummary()

		unusedCount := 0
		for _, u := range usages {
			if u.RefCount == 0 {
				unusedCount++
			}
		}

		// Main navigation command links
		btnApps, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Installed Applications (%d)\nView, launch, or remove registered Python applications", len(apps)))
		btnPkgs, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Shared Package Pool (%d packages, %d unused)\nInspect deduplicated dependencies and clean unused packages", len(usages), unusedCount))
		btnRts, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Python Runtimes (%d installed)\nInspect installed embeddable Python distributions", len(rts)))
		btnStorage, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Storage Usage (%s total)\nInspect disk space used by runtimes, packages, and cache", FormatBytes(sum.TotalBytes)))
		btnClose, _ := syscall.UTF16PtrFromString("Close Manager\nExit Hrunner Manager")

		buttons := []TASKDIALOG_BUTTON{
			{nButtonID: BtnNavApps, pszButtonText: btnApps},
			{nButtonID: BtnNavPackages, pszButtonText: btnPkgs},
			{nButtonID: BtnNavRuntimes, pszButtonText: btnRts},
			{nButtonID: BtnNavStorage, pszButtonText: btnStorage},
			{nButtonID: BtnNavExit, pszButtonText: btnClose},
		}

		res, err := ShowTaskDialog(
			"Hrunner Manager",
			"Hrunner Desktop Manager",
			"Central management for Python runtimes, shared dependencies, and native desktop applications.",
			TDF_USE_COMMAND_LINKS,
			buttons,
			BtnNavApps,
		)
		if err != nil || res == BtnNavExit || res == IDCANCEL {
			break
		}

		switch res {
		case BtnNavApps:
			m.showApplicationsDialog()
		case BtnNavPackages:
			m.showPackagePoolDialog()
		case BtnNavRuntimes:
			m.showRuntimesDialog()
		case BtnNavStorage:
			m.showStorageDialog()
		}
	}
}

func (m *NativeManager) showApplicationsDialog() {
	apps := m.reg.List()
	if len(apps) == 0 {
		btnOK, _ := syscall.UTF16PtrFromString("Back to Main Menu")
		_, _ = ShowTaskDialog(
			"Installed Applications",
			"No Applications Installed",
			"No applications have been registered with Hrunner yet.\nWhen you launch an application built with Hrunner, it will appear here.",
			0,
			[]TASKDIALOG_BUTTON{{nButtonID: IDOK, pszButtonText: btnOK}},
			IDOK,
		)
		return
	}

	var buttons []TASKDIALOG_BUTTON
	for idx, app := range apps {
		text, _ := syscall.UTF16PtrFromString(fmt.Sprintf("%s (v%s)\nPython %s • %d dependencies • State: %s",
			app.Name, app.Version, app.PythonVersion, len(app.Dependencies), app.State))
		buttons = append(buttons, TASKDIALOG_BUTTON{
			nButtonID: int32(100 + idx),
			pszButtonText: text,
		})
	}
	btnBack, _ := syscall.UTF16PtrFromString("Back to Main Menu")
	buttons = append(buttons, TASKDIALOG_BUTTON{nButtonID: BtnActionBack, pszButtonText: btnBack})

	res, _ := ShowTaskDialog(
		"Installed Applications",
		"Select an Application",
		"Choose an application to launch or remove:",
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnActionBack,
	)

	if res >= 100 && int(res-100) < len(apps) {
		selectedApp := apps[res-100]
		m.showAppDetailsDialog(selectedApp)
	}
}

func (m *NativeManager) showAppDetailsDialog(app *registry.ApplicationRecord) {
	btnLaunch, _ := syscall.UTF16PtrFromString("Launch Application\nStart the application with its isolated environment")
	btnRemove, _ := syscall.UTF16PtrFromString("Remove Application\nUnregister application and calculate orphaned packages")
	btnBack, _ := syscall.UTF16PtrFromString("Back")

	buttons := []TASKDIALOG_BUTTON{
		{nButtonID: BtnActionLaunch, pszButtonText: btnLaunch},
		{nButtonID: BtnActionRemove, pszButtonText: btnRemove},
		{nButtonID: BtnActionBack, pszButtonText: btnBack},
	}

	content := fmt.Sprintf("Application ID: %s\nVersion: %s\nPython Version: %s\nEntrypoint: %s\nExecutable: %s\n\nDependencies:\n",
		app.ApplicationID, app.Version, app.PythonVersion, app.Entrypoint, app.ExecutablePath)
	for p, v := range app.Dependencies {
		content += fmt.Sprintf("  • %s == %s\n", p, v)
	}

	res, _ := ShowTaskDialog(
		app.Name,
		fmt.Sprintf("%s (v%s)", app.Name, app.Version),
		content,
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnActionLaunch,
	)

	if res == BtnActionLaunch {
		m.launchApp(app)
	} else if res == BtnActionRemove {
		m.removeApp(app)
	}
}

func (m *NativeManager) launchApp(app *registry.ApplicationRecord) {
	rt, exists, err := m.runtimes.Find(app.PythonVersion, false)
	if err != nil || !exists {
		ShowMessageBox("Launch Error", fmt.Sprintf("Required Python runtime %s is not installed.", app.PythonVersion), 0x10 /* MB_ICONERROR */)
		return
	}

	var pkgPaths []string
	for pkg, ver := range app.Dependencies {
		pkgPaths = append(pkgPaths, m.pkgStore.GetPackageDir(pkg, ver))
	}

	cmd, err := m.runtimes.RunApplication(rt, filepath.Dir(app.ExecutablePath), app.Entrypoint, pkgPaths, nil)
	if err != nil {
		ShowMessageBox("Launch Error", err.Error(), 0x10)
		return
	}

	if err := cmd.Start(); err != nil {
		ShowMessageBox("Launch Error", err.Error(), 0x10)
		return
	}

	_ = m.reg.UpdateLastLaunched(app.ApplicationID)
	ShowMessageBox("Application Launched", fmt.Sprintf("%s has been started successfully (PID: %d).", app.Name, cmd.Process.Pid), 0x40 /* MB_ICONINFORMATION */)
}

func (m *NativeManager) removeApp(app *registry.ApplicationRecord) {
	impact, err := m.storageMgr.CalculateAppRemovalImpact(app.ApplicationID)
	if err != nil {
		ShowMessageBox("Error", err.Error(), 0x10)
		return
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Are you sure you want to remove %s?\n\n", app.Name))
	if len(impact.PackagesNowUnused) > 0 {
		msg.WriteString("The following packages will become unused:\n")
		for _, u := range impact.PackagesNowUnused {
			msg.WriteString(fmt.Sprintf("  • %s %s (%s)\n", u.Name, u.Version, FormatBytes(u.SizeBytes)))
		}
		msg.WriteString(fmt.Sprintf("\nReclaimable package disk space: %s", FormatBytes(impact.ReclaimedBytesPool)))
	} else {
		msg.WriteString("All packages used by this application are still needed by other applications.")
	}

	btnRemoveClean, _ := syscall.UTF16PtrFromString("Remove and Clean Unused Packages\nDelete application and free orphaned package files")
	btnRemoveOnly, _ := syscall.UTF16PtrFromString("Remove Application Only\nKeep packages in pool for future use")
	btnCancel, _ := syscall.UTF16PtrFromString("Cancel")

	buttons := []TASKDIALOG_BUTTON{
		{nButtonID: 1, pszButtonText: btnRemoveClean},
		{nButtonID: 2, pszButtonText: btnRemoveOnly},
		{nButtonID: IDCANCEL, pszButtonText: btnCancel},
	}

	choice, _ := ShowTaskDialog(
		"Confirm Removal",
		fmt.Sprintf("Remove %s?", app.Name),
		msg.String(),
		TDF_USE_COMMAND_LINKS,
		buttons,
		IDCANCEL,
	)

	if choice == 1 {
		_ = m.storageMgr.RemoveApplicationAndCleanup(app.ApplicationID, true)
		ShowMessageBox("Removed", fmt.Sprintf("%s and its unused packages have been removed.", app.Name), 0x40)
	} else if choice == 2 {
		_ = m.storageMgr.RemoveApplicationAndCleanup(app.ApplicationID, false)
		ShowMessageBox("Removed", fmt.Sprintf("%s has been removed from registry.", app.Name), 0x40)
	}
}

func (m *NativeManager) showPackagePoolDialog() {
	usages, err := m.storageMgr.GetPackageUsageMap()
	if err != nil {
		ShowMessageBox("Error", err.Error(), 0x10)
		return
	}

	var unused []*storage.PackageUsage
	var content strings.Builder
	content.WriteString("Packages in shared pool:\n\n")

	for _, u := range usages {
		status := "SHARED"
		if u.RefCount == 0 {
			status = "UNUSED"
			unused = append(unused, u)
		}
		content.WriteString(fmt.Sprintf("• %s %s [%s] — Used by %d app(s), %s\n",
			u.Name, u.Version, status, u.RefCount, FormatBytes(u.SizeBytes)))
	}

	var buttons []TASKDIALOG_BUTTON
	if len(unused) > 0 {
		btnClean, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Remove Unused Packages (%d)\nFree disk space from packages no longer used by any application", len(unused)))
		buttons = append(buttons, TASKDIALOG_BUTTON{nButtonID: BtnActionClean, pszButtonText: btnClean})
	}

	btnBack, _ := syscall.UTF16PtrFromString("Back to Main Menu")
	buttons = append(buttons, TASKDIALOG_BUTTON{nButtonID: BtnActionBack, pszButtonText: btnBack})

	res, _ := ShowTaskDialog(
		"Shared Package Pool",
		fmt.Sprintf("Package Pool (%d Packages)", len(usages)),
		content.String(),
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnActionBack,
	)

	if res == BtnActionClean {
		deleted, freed, err := m.storageMgr.CleanUnusedPackages()
		if err != nil {
			ShowMessageBox("Clean Failed", err.Error(), 0x10)
		} else {
			ShowMessageBox("Cleanup Complete", fmt.Sprintf("Removed %d unused package(s), freed %s of disk space.", deleted, FormatBytes(freed)), 0x40)
		}
	}
}

func (m *NativeManager) showRuntimesDialog() {
	rts, _ := m.runtimes.List()

	var content strings.Builder
	if len(rts) == 0 {
		content.WriteString("No Python runtimes currently installed.")
	} else {
		for _, r := range rts {
			content.WriteString(fmt.Sprintf("• Python %s (%s)\n  Path: %s\n\n",
				r.Version.String(), FormatBytes(r.SizeBytes), r.Path))
		}
	}

	btnBack, _ := syscall.UTF16PtrFromString("Back to Main Menu")
	_, _ = ShowTaskDialog(
		"Python Runtimes",
		fmt.Sprintf("Installed Python Runtimes (%d)", len(rts)),
		content.String(),
		0,
		[]TASKDIALOG_BUTTON{{nButtonID: BtnActionBack, pszButtonText: btnBack}},
		BtnActionBack,
	)
}

func (m *NativeManager) showStorageDialog() {
	sum, _ := m.storageMgr.GetSummary()

	content := fmt.Sprintf("Disk space consumed by Hrunner components:\n\n"+
		"• Python Runtimes:  %s\n"+
		"• Shared Packages:  %s\n"+
		"• Applications:     %s\n"+
		"• Download Cache:   %s\n"+
		"────────────────────────────\n"+
		"Total Disk Usage:   %s",
		FormatBytes(sum.RuntimesBytes),
		FormatBytes(sum.PackagesBytes),
		FormatBytes(sum.ApplicationsBytes),
		FormatBytes(sum.CacheBytes),
		FormatBytes(sum.TotalBytes),
	)

	btnBack, _ := syscall.UTF16PtrFromString("Back to Main Menu")
	_, _ = ShowTaskDialog(
		"Hrunner Storage",
		"Storage Breakdown",
		content,
		0,
		[]TASKDIALOG_BUTTON{{nButtonID: BtnActionBack, pszButtonText: btnBack}},
		BtnActionBack,
	)
}
