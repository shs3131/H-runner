package ui

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/hrunner/hrunner/pkg/protocol"
)

var (
	modComctl32             = syscall.NewLazyDLL("comctl32.dll")
	procTaskDialogIndirect  = modComctl32.NewProc("TaskDialogIndirect")
	procInitCommonControls  = modComctl32.NewProc("InitCommonControlsEx")
	modUser32               = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW         = modUser32.NewProc("MessageBoxW")
)

const (
	// TaskDialog flags
	TDF_ENABLE_HYPERLINKS           = 0x0001
	TDF_USE_HICON_MAIN              = 0x0002
	TDF_USE_COMMAND_LINKS           = 0x0010
	TDF_USE_COMMAND_LINKS_NO_ICON   = 0x0020
	TDF_SHOW_PROGRESS_BAR           = 0x0200
	TDF_SHOW_MARQUEE_PROGRESS_BAR   = 0x0400
	TDF_CALLBACK_TIMER              = 0x0800
	TDF_POSITION_RELATIVE_TO_WINDOW = 0x1000
	TDF_CAN_BE_MINIMIZED            = 0x8000

	// Common button flags
	TDCBF_OK_BUTTON     = 0x0001
	TDCBF_YES_BUTTON    = 0x0002
	TDCBF_NO_BUTTON     = 0x0004
	TDCBF_CANCEL_BUTTON = 0x0008
	TDCBF_RETRY_BUTTON  = 0x0010
	TDCBF_CLOSE_BUTTON  = 0x0020

	// Standard Button IDs
	IDOK     = 1
	IDCANCEL = 2
	IDABORT  = 3
	IDRETRY  = 4
	IDIGNORE = 5
	IDYES    = 6
	IDNO     = 7
	IDCLOSE  = 8

	// Custom button IDs
	BtnInstallAndRun = 1001
	BtnLaunchApp     = 1002
	BtnFinish        = 1003
	BtnInstallExact  = 1004
	BtnTryCompatible = 1005
	BtnInstallHrunner= 1006
)

type TASKDIALOG_BUTTON struct {
	nButtonID     int32
	pszButtonText *uint16
}

type TASKDIALOGCONFIG struct {
	cbSize                   uint32
	hwndParent               uintptr
	hInstance                uintptr
	dwFlags                  uint32
	dwCommonButtons          uint32
	pszWindowTitle           *uint16
	hMainIcon                uintptr
	pszMainInstruction       *uint16
	pszContent               *uint16
	cButtons                 uint32
	pButtons                 *TASKDIALOG_BUTTON
	nDefaultButton           int32
	cRadioButtons            uint32
	pRadioButtons            *TASKDIALOG_BUTTON
	nDefaultRadioButton      int32
	pszVerificationText      *uint16
	pszExpandedInformation   *uint16
	pszExpandedControlText   *uint16
	pszCollapsedControlText  *uint16
	hFooterIcon              uintptr
	pszFooter                *uint16
	pfCallback               uintptr
	lpCallbackData           uintptr
	cxWidth                  uint32
}

func init() {
	// Initialize common controls v6
	type INITCOMMONCONTROLSEX struct {
		dwSize uint32
		dwICC  uint32
	}
	icc := INITCOMMONCONTROLSEX{
		dwSize: uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})),
		dwICC:  0x0000ffff,
	}
	_, _, _ = procInitCommonControls.Call(uintptr(unsafe.Pointer(&icc)))
}

// ShowMessageBoxFallback displays a standard Win32 MessageBox.
func ShowMessageBox(title, message string, flags uint32) int {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(message)
	r, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), uintptr(flags))
	return int(r)
}

// ShowTaskDialog displays a native TaskDialog with custom buttons.
func ShowTaskDialog(title, instruction, content string, flags uint32, buttons []TASKDIALOG_BUTTON, defaultBtn int32) (int32, error) {
	if os.Getenv("HRUNNER_HEADLESS") == "1" {
		// Headless testing mode: default button
		return defaultBtn, nil
	}

	tPtr, _ := syscall.UTF16PtrFromString(title)
	iPtr, _ := syscall.UTF16PtrFromString(instruction)
	cPtr, _ := syscall.UTF16PtrFromString(content)

	var pButtons *TASKDIALOG_BUTTON
	if len(buttons) > 0 {
		pButtons = &buttons[0]
	}

	var config TASKDIALOGCONFIG
	config.cbSize = uint32(unsafe.Sizeof(config))
	config.pszWindowTitle = tPtr
	config.pszMainInstruction = iPtr
	config.pszContent = cPtr
	config.dwFlags = flags
	config.cButtons = uint32(len(buttons))
	config.pButtons = pButtons
	config.nDefaultButton = defaultBtn

	var buttonPressed int32
	hr, _, _ := procTaskDialogIndirect.Call(
		uintptr(unsafe.Pointer(&config)),
		uintptr(unsafe.Pointer(&buttonPressed)),
		0,
		0,
	)

	if hr != 0 {
		// Fallback to standard message box if TaskDialogIndirect fails
		mb := ShowMessageBox(title, instruction+"\n\n"+content, 0x00000001 /* MB_OKCANCEL */)
		if mb == 1 {
			return defaultBtn, nil
		}
		return IDCANCEL, nil
	}

	return buttonPressed, nil
}

// FormatBytes formats byte counts into human-readable strings (e.g. 48.7 MB).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// UI manages all installer dialogs.
type UI struct {
	headless bool
}

// NewUI creates a UI manager.
func NewUI() *UI {
	return &UI{
		headless: os.Getenv("HRUNNER_HEADLESS") == "1",
	}
}

// ShowMissingHrunner displays the prompt when Hrunner is not found.
func (u *UI) ShowMissingHrunner() bool {
	if u.headless {
		return false
	}

	btnInstall, _ := syscall.UTF16PtrFromString("Install Hrunner\nDownload and set up Hrunner runtime")
	btnCancel, _ := syscall.UTF16PtrFromString("Cancel")

	buttons := []TASKDIALOG_BUTTON{
		{nButtonID: BtnInstallHrunner, pszButtonText: btnInstall},
		{nButtonID: IDCANCEL, pszButtonText: btnCancel},
	}

	res, _ := ShowTaskDialog(
		"Hrunner is required",
		"This application requires Hrunner to run.",
		"Hrunner manages shared Python environments and dependencies centrally so applications stay lightweight.",
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnInstallHrunner,
	)

	return res == BtnInstallHrunner
}

// AskPythonRuntime prompts user when requested Python version is missing but a compatible one exists.
func (u *UI) AskPythonRuntime(requestedVer, compatibleVer string) (string, bool) {
	if u.headless {
		return "install_exact", true
	}

	btnExact, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Install Python %s\nDownload and install the exact requested version", requestedVer))
	var buttons []TASKDIALOG_BUTTON

	buttons = append(buttons, TASKDIALOG_BUTTON{nButtonID: BtnInstallExact, pszButtonText: btnExact})

	if compatibleVer != "" {
		btnCompat, _ := syscall.UTF16PtrFromString(fmt.Sprintf("Try existing Python %s\nUse already installed compatible version", compatibleVer))
		buttons = append(buttons, TASKDIALOG_BUTTON{nButtonID: BtnTryCompatible, pszButtonText: btnCompat})
	}

	btnCancel, _ := syscall.UTF16PtrFromString("Cancel")
	buttons = append(buttons, TASKDIALOG_BUTTON{nButtonID: IDCANCEL, pszButtonText: btnCancel})

	res, _ := ShowTaskDialog(
		"Python Runtime Required",
		fmt.Sprintf("Python %s is not installed.", requestedVer),
		"How would you like to continue?",
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnInstallExact,
	)

	if res == BtnInstallExact {
		return "install_exact", true
	} else if res == BtnTryCompatible {
		return "use_compatible", true
	}
	return "", false
}

// ConfirmInstallation prompts user before downloading missing runtime/packages.
func (u *UI) ConfirmInstallation(appName, pyVer string, missingPy bool, missingPkgs []protocol.MissingPackageInfo, totalBytes int64) bool {
	if u.headless {
		return true
	}

	var sb strings.Builder
	sb.WriteString("This application requires:\n\n")

	if missingPy {
		sb.WriteString(fmt.Sprintf("• Python %s\n", pyVer))
	}
	for _, p := range missingPkgs {
		sb.WriteString(fmt.Sprintf("• %s %s (%s)\n", p.Name, p.Version, FormatBytes(p.DownloadBytes)))
	}

	sb.WriteString(fmt.Sprintf("\nTotal Download Size: %s", FormatBytes(totalBytes)))

	btnInstall, _ := syscall.UTF16PtrFromString("Install & Run\nDownload missing dependencies and start application")
	btnCancel, _ := syscall.UTF16PtrFromString("Cancel")

	buttons := []TASKDIALOG_BUTTON{
		{nButtonID: BtnInstallAndRun, pszButtonText: btnInstall},
		{nButtonID: IDCANCEL, pszButtonText: btnCancel},
	}

	res, _ := ShowTaskDialog(
		appName,
		fmt.Sprintf("Install dependencies for %s", appName),
		sb.String(),
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnInstallAndRun,
	)

	return res == BtnInstallAndRun
}

// ShowInstallComplete displays the completion dialog with Launch and Finish buttons.
func (u *UI) ShowInstallComplete(appName string) bool {
	if u.headless {
		return true
	}

	btnLaunch, _ := syscall.UTF16PtrFromString("Launch Application\nStart the application now")
	btnFinish, _ := syscall.UTF16PtrFromString("Finish\nClose installer")

	buttons := []TASKDIALOG_BUTTON{
		{nButtonID: BtnLaunchApp, pszButtonText: btnLaunch},
		{nButtonID: BtnFinish, pszButtonText: btnFinish},
	}

	res, _ := ShowTaskDialog(
		"Installation Complete",
		fmt.Sprintf("%s has been installed successfully.", appName),
		"All required Python runtime components and dependencies are ready in the Hrunner shared pool.",
		TDF_USE_COMMAND_LINKS,
		buttons,
		BtnLaunchApp,
	)

	return res == BtnLaunchApp
}
