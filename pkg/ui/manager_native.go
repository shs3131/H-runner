package ui

import (
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/hrunner/hrunner/pkg/packages"
	"github.com/hrunner/hrunner/pkg/registry"
	"github.com/hrunner/hrunner/pkg/runtime"
	"github.com/hrunner/hrunner/pkg/storage"
)

var (
	procRegisterClassExW = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW  = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW   = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow    = modUser32.NewProc("DestroyWindow")
	procPostQuitMessage  = modUser32.NewProc("PostQuitMessage")
	procShowWindow       = modUser32.NewProc("ShowWindow")
	procUpdateWindow     = modUser32.NewProc("UpdateWindow")
	procGetMessageW      = modUser32.NewProc("GetMessageW")
	procTranslateMessage = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW = modUser32.NewProc("DispatchMessageW")
	procSendMessageW     = modUser32.NewProc("SendMessageW")
	procSetWindowTextW   = modUser32.NewProc("SetWindowTextW")
	procGetSystemMetrics = modUser32.NewProc("GetSystemMetrics")

	modGdi32           = syscall.NewLazyDLL("gdi32.dll")
	procGetStockObject = modGdi32.NewProc("GetStockObject")
)

type WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type POINT struct {
	x, y int32
}

type MSG struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      POINT
}

const (
	WM_DESTROY       = 0x0002
	WM_SETFONT       = 0x0030
	WM_COMMAND       = 0x0111
	WS_OVERLAPPED    = 0x00000000
	WS_CAPTION       = 0x00C00000
	WS_SYSMENU       = 0x00080000
	WS_THICKFRAME    = 0x00040000
	WS_MINIMIZEBOX   = 0x00020000
	WS_VISIBLE       = 0x10000000
	WS_CHILD         = 0x40000000
	WS_BORDER        = 0x00800000
	WS_VSCROLL       = 0x00200000
	WS_HSCROLL       = 0x00100000
	ES_MULTILINE     = 0x0004
	ES_READONLY      = 0x0800
	ES_AUTOVSCROLL   = 0x0040
	COLOR_BTNFACE    = 15
	DEFAULT_GUI_FONT = 17
	SW_SHOW          = 5
	SM_CXSCREEN      = 0
	SM_CYSCREEN      = 1

	TabApplications = 101
	TabPackages     = 102
	TabRuntimes     = 103
	TabStorage      = 104

	CmdCleanPackages = 201
	CmdRefresh       = 202
	CmdLaunchApp     = 203
	CmdRemoveApp     = 204
	CmdClose         = 205
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
// It uses native Windows windows, controls, and TaskDialogs with Common Controls.
// It NEVER opens a web browser or localhost HTTP server.
type NativeManager struct {
	reg        *registry.RegistryStore
	runtimes   *runtime.RuntimeStore
	pkgStore   *packages.PackageStore
	storageMgr *storage.StorageManager

	activeTab  int
	hwndEdit   uintptr
	hwndStatus uintptr
	hwndMain   uintptr
}

var activeManagerInstance *NativeManager

// NewNativeManager creates a new native Windows manager instance.
func NewNativeManager(reg *registry.RegistryStore, runtimes *runtime.RuntimeStore, pkgStore *packages.PackageStore, storageMgr *storage.StorageManager) *NativeManager {
	return &NativeManager{
		reg:        reg,
		runtimes:   runtimes,
		pkgStore:   pkgStore,
		storageMgr: storageMgr,
		activeTab:  TabApplications,
	}
}

func managerWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		cmdID := int(wParam & 0xffff)
		if activeManagerInstance != nil {
			activeManagerInstance.handleCommand(cmdID)
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// RunInteractiveLoop presents the native Windows Manager UI and message loop.
func (m *NativeManager) RunInteractiveLoop() {
	if os.Getenv("HRUNNER_HEADLESS") == "1" {
		return
	}

	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()

	activeManagerInstance = m

	className, _ := syscall.UTF16PtrFromString("HrunnerManagerWindow")
	windowTitle, _ := syscall.UTF16PtrFromString("Hrunner Desktop Manager - Python Runtime & Dependency Manager")

	var wc WNDCLASSEXW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = syscall.NewCallback(managerWndProc)
	wc.hbrBackground = uintptr(COLOR_BTNFACE + 1)
	wc.lpszClassName = className

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Center on screen
	scrW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	scrH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	wndW := int32(840)
	wndH := int32(580)
	x := (int32(scrW) - wndW) / 2
	y := (int32(scrH) - wndH) / 2
	if x < 0 {
		x = 50
	}
	if y < 0 {
		y = 50
	}

	style := uint32(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | WS_VISIBLE)
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(wndW), uintptr(wndH),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		return
	}
	m.hwndMain = hwnd

	font, _, _ := procGetStockObject.Call(DEFAULT_GUI_FONT)

	// Header static
	staticClass, _ := syscall.UTF16PtrFromString("STATIC")
	headerText, _ := syscall.UTF16PtrFromString("Hrunner Desktop Manager")
	hHeader, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(staticClass)), uintptr(unsafe.Pointer(headerText)),
		WS_CHILD|WS_VISIBLE,
		20, 15, 780, 24,
		hwnd, 0, 0, 0,
	)
	if hHeader != 0 && font != 0 {
		procSendMessageW.Call(hHeader, WM_SETFONT, font, 1)
	}

	subText, _ := syscall.UTF16PtrFromString("Central management for Python runtimes, shared dependencies, and native applications.")
	hSub, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(staticClass)), uintptr(unsafe.Pointer(subText)),
		WS_CHILD|WS_VISIBLE,
		20, 42, 780, 20,
		hwnd, 0, 0, 0,
	)
	if hSub != 0 && font != 0 {
		procSendMessageW.Call(hSub, WM_SETFONT, font, 1)
	}

	// Tab buttons
	btnClass, _ := syscall.UTF16PtrFromString("BUTTON")
	tabs := []struct {
		id   int
		text string
		x    int32
		w    int32
	}{
		{TabApplications, "1. Applications", 20, 180},
		{TabPackages, "2. Shared Packages", 210, 180},
		{TabRuntimes, "3. Python Runtimes", 400, 180},
		{TabStorage, "4. Storage Overview", 590, 180},
	}

	for _, t := range tabs {
		btnTxt, _ := syscall.UTF16PtrFromString(t.text)
		hBtn, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(btnClass)), uintptr(unsafe.Pointer(btnTxt)),
			WS_CHILD|WS_VISIBLE,
			uintptr(t.x), 72, uintptr(t.w), 32,
			hwnd, uintptr(t.id), 0, 0,
		)
		if hBtn != 0 && font != 0 {
			procSendMessageW.Call(hBtn, WM_SETFONT, font, 1)
		}
	}

	// Main Display: Read-only multiline Edit control
	editClass, _ := syscall.UTF16PtrFromString("EDIT")
	m.hwndEdit, _, _ = procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(editClass)), 0,
		WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_READONLY|ES_AUTOVSCROLL,
		20, 115, 785, 340,
		hwnd, 0, 0, 0,
	)
	if m.hwndEdit != 0 && font != 0 {
		procSendMessageW.Call(m.hwndEdit, WM_SETFONT, font, 1)
	}

	// Action buttons
	actions := []struct {
		id   int
		text string
		x    int32
		w    int32
	}{
		{CmdRefresh, "Refresh", 20, 110},
		{CmdCleanPackages, "Clean Unused Packages", 140, 175},
		{CmdLaunchApp, "Launch Application", 325, 150},
		{CmdRemoveApp, "Remove Application", 485, 150},
		{CmdClose, "Close", 695, 110},
	}

	for _, a := range actions {
		btnTxt, _ := syscall.UTF16PtrFromString(a.text)
		hBtn, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(btnClass)), uintptr(unsafe.Pointer(btnTxt)),
			WS_CHILD|WS_VISIBLE,
			uintptr(a.x), 465, uintptr(a.w), 32,
			hwnd, uintptr(a.id), 0, 0,
		)
		if hBtn != 0 && font != 0 {
			procSendMessageW.Call(hBtn, WM_SETFONT, font, 1)
		}
	}

	// Status label
	statusTxt, _ := syscall.UTF16PtrFromString("Ready")
	m.hwndStatus, _, _ = procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(staticClass)), uintptr(unsafe.Pointer(statusTxt)),
		WS_CHILD|WS_VISIBLE,
		20, 508, 780, 20,
		hwnd, 0, 0, 0,
	)
	if m.hwndStatus != 0 && font != 0 {
		procSendMessageW.Call(m.hwndStatus, WM_SETFONT, font, 1)
	}

	m.refreshView()

	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	// Standard Win32 message loop: remains open until user closes window
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 || int32(r) == -1 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (m *NativeManager) handleCommand(cmdID int) {
	switch cmdID {
	case TabApplications:
		m.activeTab = TabApplications
		m.refreshView()
	case TabPackages:
		m.activeTab = TabPackages
		m.refreshView()
	case TabRuntimes:
		m.activeTab = TabRuntimes
		m.refreshView()
	case TabStorage:
		m.activeTab = TabStorage
		m.refreshView()
	case CmdRefresh:
		m.refreshView()
		m.setStatus("Refreshed successfully.")
	case CmdCleanPackages:
		deleted, freed, err := m.storageMgr.CleanUnusedPackages()
		if err != nil {
			ShowMessageBox("Clean Failed", err.Error(), 0x10)
		} else {
			ShowMessageBox("Clean Complete", fmt.Sprintf("Removed %d unused package(s), freed %s of disk space.", deleted, FormatBytes(freed)), 0x40)
			m.refreshView()
			m.setStatus(fmt.Sprintf("Cleaned %d unused package(s), freed %s.", deleted, FormatBytes(freed)))
		}
	case CmdLaunchApp:
		apps := m.reg.List()
		if len(apps) == 0 {
			ShowMessageBox("Launch Application", "No applications are currently installed.", 0x40)
			return
		}
		m.launchApp(apps[0])
		m.refreshView()
	case CmdRemoveApp:
		apps := m.reg.List()
		if len(apps) == 0 {
			ShowMessageBox("Remove Application", "No applications are currently installed.", 0x40)
			return
		}
		m.removeApp(apps[0])
		m.refreshView()
	case CmdClose:
		if m.hwndMain != 0 {
			procDestroyWindow.Call(m.hwndMain)
		}
	}
}

func (m *NativeManager) setStatus(text string) {
	if m.hwndStatus != 0 {
		p, _ := syscall.UTF16PtrFromString(text)
		procSetWindowTextW.Call(m.hwndStatus, uintptr(unsafe.Pointer(p)))
	}
}

func (m *NativeManager) refreshView() {
	if m.hwndEdit == 0 {
		return
	}

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

	var content strings.Builder
	switch m.activeTab {
	case TabApplications:
		content.WriteString(fmt.Sprintf("=== INSTALLED APPLICATIONS (%d) ===\r\n\r\n", len(apps)))
		if len(apps) == 0 {
			content.WriteString("No applications registered yet.\r\nLaunch an application built with Hrunner to register it here automatically.\r\n")
		} else {
			for idx, app := range apps {
				content.WriteString(fmt.Sprintf("[%d] %s (v%s)\r\n", idx+1, app.Name, app.Version))
				content.WriteString(fmt.Sprintf("    Application ID: %s\r\n", app.ApplicationID))
				content.WriteString(fmt.Sprintf("    Python Runtime: %s\r\n", app.PythonVersion))
				content.WriteString(fmt.Sprintf("    State:          %s\r\n", app.State))
				content.WriteString(fmt.Sprintf("    Executable:     %s\r\n", app.ExecutablePath))
				content.WriteString(fmt.Sprintf("    Dependencies (%d):\r\n", len(app.Dependencies)))
				for p, v := range app.Dependencies {
					content.WriteString(fmt.Sprintf("      • %s == %s\r\n", p, v))
				}
				content.WriteString("\r\n")
			}
		}
	case TabPackages:
		content.WriteString(fmt.Sprintf("=== SHARED PACKAGE POOL (%d packages, %d unused) ===\r\n\r\n", len(usages), unusedCount))
		if len(usages) == 0 {
			content.WriteString("Shared package pool is currently empty.\r\n")
		} else {
			for idx, u := range usages {
				status := "SHARED"
				if u.RefCount == 0 {
					status = "UNUSED"
				}
				content.WriteString(fmt.Sprintf("[%d] %s %s [%s]\r\n", idx+1, u.Name, u.Version, status))
				content.WriteString(fmt.Sprintf("    Reference Count: Used by %d application(s)\r\n", u.RefCount))
				content.WriteString(fmt.Sprintf("    Disk Size:       %s\r\n", FormatBytes(u.SizeBytes)))
				content.WriteString(fmt.Sprintf("    Location:        %s\r\n\r\n", m.pkgStore.GetPackageDir(u.Name, u.Version)))
			}
		}
	case TabRuntimes:
		content.WriteString(fmt.Sprintf("=== INSTALLED PYTHON RUNTIMES (%d) ===\r\n\r\n", len(rts)))
		if len(rts) == 0 {
			content.WriteString("No Python runtimes currently installed in central store.\r\n")
		} else {
			for idx, r := range rts {
				content.WriteString(fmt.Sprintf("[%d] Python %s (Size: %s)\r\n", idx+1, r.Version.String(), FormatBytes(r.SizeBytes)))
				content.WriteString(fmt.Sprintf("    Path: %s\r\n\r\n", r.Path))
			}
		}
	case TabStorage:
		content.WriteString("=== STORAGE BREAKDOWN ===\r\n\r\n")
		content.WriteString(fmt.Sprintf("  • Python Runtimes:  %s\r\n", FormatBytes(sum.RuntimesBytes)))
		content.WriteString(fmt.Sprintf("  • Shared Packages:  %s\r\n", FormatBytes(sum.PackagesBytes)))
		content.WriteString(fmt.Sprintf("  • Applications:     %s\r\n", FormatBytes(sum.ApplicationsBytes)))
		content.WriteString(fmt.Sprintf("  • Download Cache:   %s\r\n", FormatBytes(sum.CacheBytes)))
		content.WriteString("  ─────────────────────────────────────\r\n")
		content.WriteString(fmt.Sprintf("  Total Disk Usage:   %s\r\n\r\n", FormatBytes(sum.TotalBytes)))
		content.WriteString(fmt.Sprintf("Central Store Location: %s\r\n", registry.DefaultHrunnerDir()))
	}

	p, _ := syscall.UTF16PtrFromString(content.String())
	procSetWindowTextW.Call(m.hwndEdit, uintptr(unsafe.Pointer(p)))

	m.setStatus(fmt.Sprintf("Ready | %d application(s) | %d runtime(s) | %d package(s) (%d unused)",
		len(apps), len(rts), len(usages), unusedCount))
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
