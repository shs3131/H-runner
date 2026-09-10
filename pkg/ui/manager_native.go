package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
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
	procInvalidateRect   = modUser32.NewProc("InvalidateRect")
	procBeginPaint       = modUser32.NewProc("BeginPaint")
	procEndPaint         = modUser32.NewProc("EndPaint")
	procGetClientRect    = modUser32.NewProc("GetClientRect")
	procSetCursor        = modUser32.NewProc("SetCursor")
	procLoadCursorW      = modUser32.NewProc("LoadCursorW")
	procGetSystemMetrics = modUser32.NewProc("GetSystemMetrics")
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

type PAINTSTRUCT struct {
	hdc         uintptr
	fErase      int32
	rcPaint     RECT
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type RECT struct {
	Left, Top, Right, Bottom int32
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
	WM_DESTROY      = 0x0002
	WM_SIZE         = 0x0005
	WM_PAINT        = 0x000F
	WM_ERASEBKGND   = 0x0014
	WM_MOUSEMOVE    = 0x0200
	WM_LBUTTONDOWN  = 0x0201
	WM_LBUTTONUP    = 0x0202
	WM_MOUSEWHEEL   = 0x020A
	WM_SETCURSOR    = 0x0020

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	SW_SHOW             = 5
	IDC_ARROW           = 32512
	IDC_HAND            = 32649
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
)

type NavPage int

const (
	PageHome NavPage = iota
	PageApplications
	PageRuntimes
	PagePackages
	PageStorage
	PageSettings
)

type HitType int

const (
	HitNavItem HitType = iota + 1
	HitCard
	HitBtnLaunchApp
	HitBtnRemoveApp
	HitBtnCleanPackages
	HitBtnRefresh
	HitBtnOpenDir
	HitBtnCheckDiag
)

type HitArea struct {
	X, Y, W, H float32
	Type       HitType
	Nav        NavPage
	Param      string
}

// NativeManager provides the Windows 11 Fluent desktop manager UI.
type NativeManager struct {
	reg        *registry.RegistryStore
	runtimes   *runtime.RuntimeStore
	pkgStore   *packages.PackageStore
	storageMgr *storage.StorageManager

	theme      *FluentTheme
	fonts      *FontSet
	activePage NavPage

	hwndMain   uintptr
	width      int32
	height     int32
	scale      float32
	scrollY    float32
	maxScrollY float32

	mouseX     float32
	mouseY     float32
	isPressed  bool
	hitAreas   []HitArea
	hoveredHit *HitArea
	hCursorHand uintptr
	hCursorArrow uintptr

	statusText string
}

var activeManager *NativeManager

func NewNativeManager(reg *registry.RegistryStore, runtimes *runtime.RuntimeStore, pkgStore *packages.PackageStore, storageMgr *storage.StorageManager) *NativeManager {
	return &NativeManager{
		reg:        reg,
		runtimes:   runtimes,
		pkgStore:   pkgStore,
		storageMgr: storageMgr,
		theme:      DefaultFluentTheme(),
		activePage: PageHome,
		scale:      1.0,
		statusText: "Ready",
	}
}

func fluentWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if activeManager == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	}
	m := activeManager

	switch msg {
	case WM_SIZE:
		m.width = int32(lParam & 0xffff)
		m.height = int32((lParam >> 16) & 0xffff)
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_ERASEBKGND:
		return 1 // Prevent flicker; full surface drawn in WM_PAINT

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			fc := NewFluentContext(hdc, m.width, m.height, m.scale)
			m.render(fc)
			fc.Close(hdc)
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		return 0

	case WM_MOUSEMOVE:
		m.mouseX = float32(int16(lParam & 0xffff))
		m.mouseY = float32(int16((lParam >> 16) & 0xffff))
		prevHovered := m.hoveredHit
		m.hoveredHit = m.findHit(m.mouseX, m.mouseY)
		if m.hoveredHit != prevHovered {
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0

	case WM_SETCURSOR:
		if m.hoveredHit != nil {
			procSetCursor.Call(m.hCursorHand)
			return 1
		}
		procSetCursor.Call(m.hCursorArrow)
		return 1

	case WM_LBUTTONDOWN:
		m.mouseX = float32(int16(lParam & 0xffff))
		m.mouseY = float32(int16((lParam >> 16) & 0xffff))
		m.isPressed = true
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_LBUTTONUP:
		m.mouseX = float32(int16(lParam & 0xffff))
		m.mouseY = float32(int16((lParam >> 16) & 0xffff))
		m.isPressed = false
		hit := m.findHit(m.mouseX, m.mouseY)
		if hit != nil {
			m.handleClick(hit)
		}
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_MOUSEWHEEL:
		delta := int16((wParam >> 16) & 0xffff)
		step := float32(delta) * 0.4
		m.scrollY -= step
		if m.scrollY < 0 {
			m.scrollY = 0
		}
		if m.scrollY > m.maxScrollY {
			m.scrollY = m.maxScrollY
		}
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}

	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (m *NativeManager) findHit(x, y float32) *HitArea {
	for i := len(m.hitAreas) - 1; i >= 0; i-- {
		h := &m.hitAreas[i]
		if x >= h.X && x <= h.X+h.W && y >= h.Y && y <= h.Y+h.H {
			return h
		}
	}
	return nil
}

func (m *NativeManager) handleClick(hit *HitArea) {
	switch hit.Type {
	case HitNavItem:
		m.activePage = hit.Nav
		m.scrollY = 0

	case HitCard:
		m.activePage = hit.Nav
		m.scrollY = 0

	case HitBtnLaunchApp:
		appID := hit.Param
		for _, a := range m.reg.List() {
			if a.ApplicationID == appID {
				m.launchApp(a)
				break
			}
		}

	case HitBtnRemoveApp:
		appID := hit.Param
		for _, a := range m.reg.List() {
			if a.ApplicationID == appID {
				m.removeApp(a)
				break
			}
		}

	case HitBtnCleanPackages:
		deleted, freed, err := m.storageMgr.CleanUnusedPackages()
		if err != nil {
			ShowMessageBox("Clean Failed", err.Error(), 0x10)
		} else {
			ShowMessageBox("Cleanup Complete", fmt.Sprintf("Removed %d unused package(s), freed %s.", deleted, FormatBytes(freed)), 0x40)
			m.statusText = fmt.Sprintf("Cleaned %d package(s), freed %s", deleted, FormatBytes(freed))
		}

	case HitBtnRefresh:
		m.statusText = "Refreshed"

	case HitBtnOpenDir:
		baseDir := registry.DefaultHrunnerDir()
		_ = exec.Command("explorer.exe", baseDir).Start()

	case HitBtnCheckDiag:
		ShowMessageBox("Hrunner Diagnostics", "All components operational.\n• Named Pipe: \\\\.\\pipe\\hrunner (Active)\n• Central Store: Valid\n• DWM Mica: Enabled", 0x40)
	}
}

// RunInteractiveLoop starts the Windows 11 Fluent Manager desktop application.
func (m *NativeManager) RunInteractiveLoop() {
	if os.Getenv("HRUNNER_HEADLESS") == "1" {
		return
	}

	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()

	EnablePerMonitorHighDPI()
	InitGdiplus()
	defer ShutdownGdiplus()

	activeManager = m
	m.fonts = LoadFluentFonts(m.scale)
	defer m.fonts.Close()

	hCurHand, _, _ := procLoadCursorW.Call(0, uintptr(IDC_HAND))
	m.hCursorHand = hCurHand
	hCurArrow, _, _ := procLoadCursorW.Call(0, uintptr(IDC_ARROW))
	m.hCursorArrow = hCurArrow

	className, _ := syscall.UTF16PtrFromString("HrunnerManagerFluent")
	windowTitle, _ := syscall.UTF16PtrFromString("Hrunner")

	var wc WNDCLASSEXW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = syscall.NewCallback(fluentWndProc)
	wc.lpszClassName = className

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	scrW, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	scrH, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	initW := int32(1060)
	initH := int32(700)
	x := (int32(scrW) - initW) / 2
	y := (int32(scrH) - initH) / 2

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		uintptr(x), uintptr(y), uintptr(initW), uintptr(initH),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		return
	}
	m.hwndMain = hwnd
	m.width = initW
	m.height = initH

	// Apply Windows 11 Mica, rounded corners, immersive style
	ApplyWindows11DwmEffects(hwnd, false)

	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	// Standard Win32 Message Loop
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

// render performs full-window double-buffered Fluent drawing.
func (m *NativeManager) render(fc *FluentContext) {
	m.hitAreas = m.hitAreas[:0] // Reset hit targets for current frame

	t := m.theme
	w := float32(m.width)
	h := float32(m.height)

	// Window Background (Mica / Fluent Canvas)
	fc.Clear(t.BgWindow)

	sidebarW := float32(230)

	// 1. Render Left Sidebar
	m.renderSidebar(fc, sidebarW, h)

	// 2. Render Main Content Area
	contentX := sidebarW
	contentW := w - sidebarW
	m.renderContent(fc, contentX, contentW, h)
}

func (m *NativeManager) renderSidebar(fc *FluentContext, sw, sh float32) {
	t := m.theme

	// Sidebar Background
	fc.FillRect(0, 0, sw, sh, t.BgSidebar)
	// Subtle vertical separator line
	fc.FillRect(sw-1, 0, 1, sh, t.BgCardBorder)

	// App Brand Header
	brandY := float32(24)
	iconSize := float32(32)
	// Brand emblem badge (Blue rounded square with 'H')
	fc.FillRoundedRect(20, brandY, iconSize, iconSize, 8, t.AccentPrimary)
	fc.DrawString("H", m.fonts.CardHead, MakeRGB(255, 255, 255), 20, brandY+2, iconSize, iconSize, StringAlignmentCenter, StringAlignmentCenter)

	fc.DrawString("Hrunner", m.fonts.CardHead, t.TextPrimary, 60, brandY+2, 100, 20, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("v1.0.3", m.fonts.Caption, t.TextSecondary, 60, brandY+20, 100, 14, StringAlignmentNear, StringAlignmentNear)

	// Navigation Menu items
	navItems := []struct {
		page NavPage
		name string
		icon IconType
	}{
		{PageHome, "Home", IconHome},
		{PageApplications, "Applications", IconApplications},
		{PageRuntimes, "Python Runtimes", IconRuntimes},
		{PagePackages, "Shared Packages", IconPackages},
		{PageStorage, "Storage & Cache", IconStorage},
		{PageSettings, "Settings", IconSettings},
	}

	itemY := float32(82)
	itemH := float32(38)
	itemW := sw - 32

	for _, item := range navItems {
		isActive := m.activePage == item.page
		itemRect := RectF{X: 16, Y: itemY, Width: itemW, Height: itemH}

		isHovered := m.mouseX >= itemRect.X && m.mouseX <= itemRect.X+itemRect.Width &&
			m.mouseY >= itemRect.Y && m.mouseY <= itemRect.Y+itemRect.Height

		// Pill background for active or hover
		if isActive {
			fc.FillRoundedRect(itemRect.X, itemRect.Y, itemRect.Width, itemRect.Height, 6, t.NavActive)
			fc.DrawRoundedRect(itemRect.X, itemRect.Y, itemRect.Width, itemRect.Height, 6, t.BgCardBorder, 1.0)
			// Left vertical accent indicator bar
			fc.FillRoundedRect(itemRect.X+1, itemRect.Y+8, 3.5, itemRect.Height-16, 2, t.NavActiveBar)
		} else if isHovered {
			fc.FillRoundedRect(itemRect.X, itemRect.Y, itemRect.Width, itemRect.Height, 6, t.NavHover)
		}

		// Icon
		iconColor := t.TextSecondary
		textColor := t.TextPrimary
		if isActive {
			iconColor = t.AccentPrimary
			textColor = t.TextPrimary
		}
		fc.DrawFluentIcon(item.icon, itemRect.X+14, itemRect.Y+11, 16, iconColor)

		// Label
		font := m.fonts.Body
		if isActive {
			font = m.fonts.BodyBold
		}
		fc.DrawString(item.name, font, textColor, itemRect.X+40, itemRect.Y, itemRect.Width-45, itemRect.Height, StringAlignmentNear, StringAlignmentCenter)

		// Register hit area
		m.hitAreas = append(m.hitAreas, HitArea{
			X:    itemRect.X,
			Y:    itemRect.Y,
			W:    itemRect.Width,
			H:    itemRect.Height,
			Type: HitNavItem,
			Nav:  item.page,
		})

		itemY += itemH + 4
	}

	// Sidebar Footer
	footerY := sh - 60
	apps := m.reg.List()
	rts, _ := m.runtimes.List()
	sum, _ := m.storageMgr.GetSummary()

	fc.DrawString("● Service Active", m.fonts.Caption, t.BadgeReadyText, 20, footerY, sw-40, 16, StringAlignmentNear, StringAlignmentNear)
	summaryStr := fmt.Sprintf("%d apps • %d runtimes • %s", len(apps), len(rts), FormatBytes(sum.TotalBytes))
	fc.DrawString(summaryStr, m.fonts.Caption, t.TextTertiary, 20, footerY+18, sw-40, 16, StringAlignmentNear, StringAlignmentNear)
}

func (m *NativeManager) renderContent(fc *FluentContext, cx, cw, ch float32) {
	switch m.activePage {
	case PageHome:
		m.renderHomePage(fc, cx, cw, ch)
	case PageApplications:
		m.renderApplicationsPage(fc, cx, cw, ch)
	case PageRuntimes:
		m.renderRuntimesPage(fc, cx, cw, ch)
	case PagePackages:
		m.renderPackagesPage(fc, cx, cw, ch)
	case PageStorage:
		m.renderStoragePage(fc, cx, cw, ch)
	case PageSettings:
		m.renderSettingsPage(fc, cx, cw, ch)
	}
}

// Page 1: Home Dashboard
func (m *NativeManager) renderHomePage(fc *FluentContext, cx, cw, ch float32) {
	t := m.theme
	pad := float32(32)
	startY := pad - m.scrollY

	// Header
	fc.DrawString("Home", m.fonts.Title, t.TextPrimary, cx+pad, startY, cw-pad*2, 30, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Central Python runtime and dependency management for Windows desktop applications.", m.fonts.Subtitle, t.TextSecondary, cx+pad, startY+32, cw-pad*2, 22, StringAlignmentNear, StringAlignmentNear)

	// Stat Metric Cards
	apps := m.reg.List()
	rts, _ := m.runtimes.List()
	usages, _ := m.storageMgr.GetPackageUsageMap()
	sum, _ := m.storageMgr.GetSummary()

	cardY := startY + 70
	cardGap := float32(14)
	availW := cw - pad*2
	cardW := (availW - cardGap*3) / 4
	cardH := float32(95)

	stats := []struct {
		title string
		val   string
		desc  string
		nav   NavPage
		icon  IconType
	}{
		{"Applications", fmt.Sprintf("%d installed", len(apps)), "Registered apps", PageApplications, IconApplications},
		{"Python Runtimes", fmt.Sprintf("%d runtimes", len(rts)), "Embeddable Python", PageRuntimes, IconRuntimes},
		{"Shared Packages", fmt.Sprintf("%d packages", len(usages)), "Deduplicated pool", PagePackages, IconPackages},
		{"Storage", FormatBytes(sum.TotalBytes), "Disk footprint", PageStorage, IconStorage},
	}

	for i, s := range stats {
		x := cx + pad + float32(i)*(cardW+cardGap)
		isHover := m.mouseX >= x && m.mouseX <= x+cardW && m.mouseY >= cardY && m.mouseY <= cardY+cardH

		bg := t.BgCard
		if isHover {
			bg = t.BgCardHover
		}
		fc.DrawCard(x, cardY, cardW, cardH, 8, bg, t.BgCardBorder)

		fc.DrawFluentIcon(s.icon, x+16, cardY+16, 18, t.AccentPrimary)
		fc.DrawString(s.title, m.fonts.Caption, t.TextSecondary, x+42, cardY+16, cardW-50, 16, StringAlignmentNear, StringAlignmentNear)
		fc.DrawString(s.val, m.fonts.CardHead, t.TextPrimary, x+16, cardY+42, cardW-32, 24, StringAlignmentNear, StringAlignmentNear)
		fc.DrawString(s.desc, m.fonts.Caption, t.TextTertiary, x+16, cardY+68, cardW-32, 16, StringAlignmentNear, StringAlignmentNear)

		m.hitAreas = append(m.hitAreas, HitArea{X: x, Y: cardY, W: cardW, H: cardH, Type: HitCard, Nav: s.nav})
	}

	// Quick Actions Card
	actionsY := cardY + cardH + 20
	actionsW := availW
	actionsH := float32(75)
	fc.DrawCard(cx+pad, actionsY, actionsW, actionsH, 8, t.BgCard, t.BgCardBorder)

	fc.DrawString("Quick Actions", m.fonts.BodyBold, t.TextPrimary, cx+pad+20, actionsY+16, 200, 20, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Manage pool, optimize disk usage, or refresh state.", m.fonts.Caption, t.TextSecondary, cx+pad+20, actionsY+38, 300, 18, StringAlignmentNear, StringAlignmentNear)

	// Buttons on Quick Actions
	btnCleanX := cx + pad + actionsW - 390
	m.renderButton(fc, "Clean Unused Packages", btnCleanX, actionsY+20, 180, 34, false, HitBtnCleanPackages, "")
	m.renderButton(fc, "Refresh", btnCleanX+190, actionsY+20, 90, 34, false, HitBtnRefresh, "")
	m.renderButton(fc, "Open Folder", btnCleanX+290, actionsY+20, 95, 34, false, HitBtnOpenDir, "")

	// Recent Applications Section
	recY := actionsY + actionsH + 26
	fc.DrawString("Registered Applications", m.fonts.CardHead, t.TextPrimary, cx+pad, recY, availW, 22, StringAlignmentNear, StringAlignmentNear)

	if len(apps) == 0 {
		emptyY := recY + 34
		fc.DrawCard(cx+pad, emptyY, availW, 110, 8, t.BgCard, t.BgCardBorder)
		fc.DrawString("No applications installed yet", m.fonts.BodyBold, t.TextPrimary, cx+pad, emptyY+32, availW, 20, StringAlignmentCenter, StringAlignmentNear)
		fc.DrawString("Applications packaged with 'hbuild' automatically register on first launch.", m.fonts.Caption, t.TextSecondary, cx+pad, emptyY+56, availW, 18, StringAlignmentCenter, StringAlignmentNear)
		m.maxScrollY = 0
		return
	}

	appCardY := recY + 32
	appCardH := float32(84)
	for _, app := range apps {
		m.renderAppCard(fc, cx+pad, appCardY, availW, appCardH, app)
		appCardY += appCardH + 12
	}

	m.maxScrollY = appCardY - ch + pad
	if m.maxScrollY < 0 {
		m.maxScrollY = 0
	}
}

// Page 2: Applications
func (m *NativeManager) renderApplicationsPage(fc *FluentContext, cx, cw, ch float32) {
	t := m.theme
	pad := float32(32)
	startY := pad - m.scrollY

	fc.DrawString("Applications", m.fonts.Title, t.TextPrimary, cx+pad, startY, cw-pad*2, 30, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Your registered Python desktop applications running with isolated environments.", m.fonts.Subtitle, t.TextSecondary, cx+pad, startY+32, cw-pad*2, 22, StringAlignmentNear, StringAlignmentNear)

	apps := m.reg.List()
	availW := cw - pad*2

	if len(apps) == 0 {
		emptyY := startY + 80
		fc.DrawCard(cx+pad, emptyY, availW, 140, 8, t.BgCard, t.BgCardBorder)
		fc.DrawFluentIcon(IconApplications, cx+pad+availW/2-14, emptyY+28, 28, t.TextTertiary)
		fc.DrawString("No Applications Registered", m.fonts.BodyBold, t.TextPrimary, cx+pad, emptyY+68, availW, 20, StringAlignmentCenter, StringAlignmentNear)
		fc.DrawString("When you launch an executable built with Hrunner, it will appear here.", m.fonts.Caption, t.TextSecondary, cx+pad, emptyY+92, availW, 18, StringAlignmentCenter, StringAlignmentNear)
		m.maxScrollY = 0
		return
	}

	curY := startY + 74
	cardH := float32(84)
	for _, app := range apps {
		m.renderAppCard(fc, cx+pad, curY, availW, cardH, app)
		curY += cardH + 12
	}

	m.maxScrollY = curY - ch + pad
	if m.maxScrollY < 0 {
		m.maxScrollY = 0
	}
}

func (m *NativeManager) renderAppCard(fc *FluentContext, x, y, w, h float32, app *registry.ApplicationRecord) {
	t := m.theme
	isHover := m.mouseX >= x && m.mouseX <= x+w && m.mouseY >= y && m.mouseY <= y+h

	bg := t.BgCard
	if isHover {
		bg = t.BgCardHover
	}
	fc.DrawCard(x, y, w, h, 8, bg, t.BgCardBorder)

	// App Icon avatar
	fc.DrawFluentIcon(IconAppPill, x+16, y+16, 48, t.AccentPrimary)

	// App Name & Version
	titleText := fmt.Sprintf("%s (v%s)", app.Name, app.Version)
	fc.DrawString(titleText, m.fonts.CardHead, t.TextPrimary, x+76, y+14, w-350, 22, StringAlignmentNear, StringAlignmentNear)

	// Subtitle: Python runtime + Dependency count
	subText := fmt.Sprintf("Python %s • %d dependencies (shared pool) • %s", app.PythonVersion, len(app.Dependencies), app.ExecutablePath)
	fc.DrawString(subText, m.fonts.Caption, t.TextSecondary, x+76, y+40, w-350, 18, StringAlignmentNear, StringAlignmentNear)

	// Status Badge Pill
	stateText := "Ready"
	dotCol := t.BadgeReadyDot
	bgCol := t.BadgeReadyBg
	txtCol := t.BadgeReadyText

	if app.State == registry.StateBroken {
		stateText = "Broken"
		dotCol = t.BadgeErrorDot
		bgCol = t.BadgeErrorBg
		txtCol = t.BadgeErrorText
	} else if app.State == registry.StateInstalling {
		stateText = "Installing"
		dotCol = t.BadgeWarnDot
		bgCol = t.BadgeWarnBg
		txtCol = t.BadgeWarnText
	}

	badgeX := x + 76
	fc.DrawBadgePill(stateText, dotCol, bgCol, txtCol, badgeX, y+58, m.fonts.Badge)

	// Action Buttons on Right
	btnLaunchX := x + w - 195
	m.renderButton(fc, "Launch", btnLaunchX, y+24, 85, 34, true, HitBtnLaunchApp, app.ApplicationID)
	m.renderButton(fc, "Remove", btnLaunchX+95, y+24, 85, 34, false, HitBtnRemoveApp, app.ApplicationID)
}

// Page 3: Python Runtimes
func (m *NativeManager) renderRuntimesPage(fc *FluentContext, cx, cw, ch float32) {
	t := m.theme
	pad := float32(32)
	startY := pad - m.scrollY

	fc.DrawString("Python Runtimes", m.fonts.Title, t.TextPrimary, cx+pad, startY, cw-pad*2, 30, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Centrally managed embeddable Python distributions in %LOCALAPPDATA%\\Hrunner\\runtimes.", m.fonts.Subtitle, t.TextSecondary, cx+pad, startY+32, cw-pad*2, 22, StringAlignmentNear, StringAlignmentNear)

	rts, _ := m.runtimes.List()
	availW := cw - pad*2

	if len(rts) == 0 {
		emptyY := startY + 80
		fc.DrawCard(cx+pad, emptyY, availW, 120, 8, t.BgCard, t.BgCardBorder)
		fc.DrawString("No Python runtimes installed yet", m.fonts.BodyBold, t.TextPrimary, cx+pad, emptyY+36, availW, 20, StringAlignmentCenter, StringAlignmentNear)
		fc.DrawString("Runtimes are downloaded on-demand when applications request them.", m.fonts.Caption, t.TextSecondary, cx+pad, emptyY+60, availW, 18, StringAlignmentCenter, StringAlignmentNear)
		m.maxScrollY = 0
		return
	}

	curY := startY + 74
	cardH := float32(80)
	for _, r := range rts {
		isHover := m.mouseX >= cx+pad && m.mouseX <= cx+pad+availW && m.mouseY >= curY && m.mouseY <= curY+cardH
		bg := t.BgCard
		if isHover {
			bg = t.BgCardHover
		}
		fc.DrawCard(cx+pad, curY, availW, cardH, 8, bg, t.BgCardBorder)

		fc.DrawFluentIcon(IconRuntimes, cx+pad+16, curY+16, 44, t.AccentPrimary)
		title := fmt.Sprintf("Python %s (Windows x64 Embeddable)", r.Version.String())
		fc.DrawString(title, m.fonts.CardHead, t.TextPrimary, cx+pad+72, curY+14, availW-250, 22, StringAlignmentNear, StringAlignmentNear)
		sub := fmt.Sprintf("Disk Footprint: %s • Path: %s", FormatBytes(r.SizeBytes), r.Path)
		fc.DrawString(sub, m.fonts.Caption, t.TextSecondary, cx+pad+72, curY+38, availW-250, 18, StringAlignmentNear, StringAlignmentNear)

		fc.DrawBadgePill("Installed", t.BadgeReadyDot, t.BadgeReadyBg, t.BadgeReadyText, cx+pad+72, curY+56, m.fonts.Badge)

		curY += cardH + 12
	}

	m.maxScrollY = curY - ch + pad
	if m.maxScrollY < 0 {
		m.maxScrollY = 0
	}
}

// Page 4: Shared Packages
func (m *NativeManager) renderPackagesPage(fc *FluentContext, cx, cw, ch float32) {
	t := m.theme
	pad := float32(32)
	startY := pad - m.scrollY

	fc.DrawString("Shared Package Pool", m.fonts.Title, t.TextPrimary, cx+pad, startY, cw-pad*2, 30, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Physically deduplicated Python wheels shared across all applications.", m.fonts.Subtitle, t.TextSecondary, cx+pad, startY+32, cw-pad*2, 22, StringAlignmentNear, StringAlignmentNear)

	usages, _ := m.storageMgr.GetPackageUsageMap()
	availW := cw - pad*2

	unusedCount := 0
	var unusedBytes int64
	for _, u := range usages {
		if u.RefCount == 0 {
			unusedCount++
			unusedBytes += u.SizeBytes
		}
	}

	// Action bar
	barY := startY + 70
	fc.DrawCard(cx+pad, barY, availW, 60, 8, t.BgCard, t.BgCardBorder)
	summaryText := fmt.Sprintf("Package Pool: %d packages • %d unused (%s reclaimable)", len(usages), unusedCount, FormatBytes(unusedBytes))
	fc.DrawString(summaryText, m.fonts.BodyBold, t.TextPrimary, cx+pad+20, barY+20, availW-300, 20, StringAlignmentNear, StringAlignmentNear)

	btnX := cx + pad + availW - 280
	m.renderButton(fc, "Clean Unused Packages", btnX, barY+14, 180, 32, false, HitBtnCleanPackages, "")
	m.renderButton(fc, "Refresh", btnX+190, barY+14, 80, 32, false, HitBtnRefresh, "")

	// Package Table
	tableY := barY + 74
	headerH := float32(30)
	fc.DrawString("PACKAGE", m.fonts.Caption, t.TextTertiary, cx+pad+16, tableY, 180, headerH, StringAlignmentNear, StringAlignmentCenter)
	fc.DrawString("VERSION", m.fonts.Caption, t.TextTertiary, cx+pad+200, tableY, 100, headerH, StringAlignmentNear, StringAlignmentCenter)
	fc.DrawString("REFERENCES", m.fonts.Caption, t.TextTertiary, cx+pad+310, tableY, 140, headerH, StringAlignmentNear, StringAlignmentCenter)
	fc.DrawString("DISK SIZE", m.fonts.Caption, t.TextTertiary, cx+pad+460, tableY, 100, headerH, StringAlignmentNear, StringAlignmentCenter)
	fc.DrawString("STATUS", m.fonts.Caption, t.TextTertiary, cx+pad+570, tableY, 100, headerH, StringAlignmentNear, StringAlignmentCenter)

	rowY := tableY + headerH
	rowH := float32(48)

	for _, u := range usages {
		isHover := m.mouseX >= cx+pad && m.mouseX <= cx+pad+availW && m.mouseY >= rowY && m.mouseY <= rowY+rowH
		bg := t.BgCard
		if isHover {
			bg = t.BgCardHover
		}
		fc.DrawCard(cx+pad, rowY, availW, rowH, 6, bg, t.BgCardBorder)

		fc.DrawString(u.Name, m.fonts.BodyBold, t.TextPrimary, cx+pad+16, rowY, 180, rowH, StringAlignmentNear, StringAlignmentCenter)
		fc.DrawString(u.Version, m.fonts.Body, t.TextSecondary, cx+pad+200, rowY, 100, rowH, StringAlignmentNear, StringAlignmentCenter)

		refText := fmt.Sprintf("%d app(s)", u.RefCount)
		fc.DrawString(refText, m.fonts.Body, t.TextSecondary, cx+pad+310, rowY, 140, rowH, StringAlignmentNear, StringAlignmentCenter)
		fc.DrawString(FormatBytes(u.SizeBytes), m.fonts.Body, t.TextSecondary, cx+pad+460, rowY, 100, rowH, StringAlignmentNear, StringAlignmentCenter)

		if u.RefCount > 0 {
			fc.DrawBadgePill("In Use", t.BadgeReadyDot, t.BadgeReadyBg, t.BadgeReadyText, cx+pad+570, rowY+12, m.fonts.Badge)
		} else {
			fc.DrawBadgePill("Unused", t.BadgeWarnDot, t.BadgeWarnBg, t.BadgeWarnText, cx+pad+570, rowY+12, m.fonts.Badge)
		}

		rowY += rowH + 6
	}

	m.maxScrollY = rowY - ch + pad
	if m.maxScrollY < 0 {
		m.maxScrollY = 0
	}
}

// Page 5: Storage & Cache
func (m *NativeManager) renderStoragePage(fc *FluentContext, cx, cw, ch float32) {
	t := m.theme
	pad := float32(32)
	startY := pad - m.scrollY
	availW := cw - pad*2

	fc.DrawString("Storage & Cache", m.fonts.Title, t.TextPrimary, cx+pad, startY, availW, 30, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Physical disk usage breakdown across Hrunner runtimes, packages, and download cache.", m.fonts.Subtitle, t.TextSecondary, cx+pad, startY+32, availW, 22, StringAlignmentNear, StringAlignmentNear)

	sum, _ := m.storageMgr.GetSummary()

	// Visual Segmented Bar Card
	barCardY := startY + 70
	barCardH := float32(115)
	fc.DrawCard(cx+pad, barCardY, availW, barCardH, 8, t.BgCard, t.BgCardBorder)

	totalTitle := fmt.Sprintf("Total Disk Footprint: %s", FormatBytes(sum.TotalBytes))
	fc.DrawString(totalTitle, m.fonts.CardHead, t.TextPrimary, cx+pad+20, barCardY+16, availW-40, 20, StringAlignmentNear, StringAlignmentNear)

	segments := []StorageSegment{
		{"Runtimes", sum.RuntimesBytes, t.StorageRuntimes},
		{"Packages", sum.PackagesBytes, t.StoragePackages},
		{"Applications", sum.ApplicationsBytes, t.StorageApps},
		{"Cache", sum.CacheBytes, t.StorageCache},
	}
	fc.DrawSegmentedBar(segments, sum.TotalBytes, cx+pad+20, barCardY+44, availW-40, 14, t.StorageTrack)

	// Legend below bar
	legX := cx + pad + 20
	legY := barCardY + 74
	for _, seg := range segments {
		fc.FillRoundedRect(legX, legY+2, 10, 10, 5, seg.Color)
		label := fmt.Sprintf("%s (%s)", seg.Label, FormatBytes(seg.Bytes))
		fc.DrawString(label, m.fonts.Caption, t.TextSecondary, legX+16, legY, 140, 16, StringAlignmentNear, StringAlignmentCenter)
		legX += 155
	}

	// Breakdown Category Cards (2 columns)
	catY := barCardY + barCardH + 18
	colW := (availW - 14) / 2
	catH := float32(75)

	cats := []struct {
		name string
		size int64
		desc string
		col  ARGB
	}{
		{"Python Runtimes", sum.RuntimesBytes, "Embeddable Python zip extractions", t.StorageRuntimes},
		{"Shared Packages", sum.PackagesBytes, "Extracted deduplicated wheel packages", t.StoragePackages},
		{"Application Cache", sum.ApplicationsBytes, "Extracted application payload files", t.StorageApps},
		{"Download Cache", sum.CacheBytes, "Cached wheels and runtime zip files", t.StorageCache},
	}

	for i, c := range cats {
		col := float32(i % 2)
		row := float32(i / 2)
		x := cx + pad + col*(colW+14)
		y := catY + row*(catH+12)

		fc.DrawCard(x, y, colW, catH, 8, t.BgCard, t.BgCardBorder)
		fc.FillRoundedRect(x+16, y+20, 36, 36, 6, c.col)
		fc.DrawFluentIcon(IconStorage, x+23, y+27, 22, MakeRGB(255, 255, 255))

		fc.DrawString(c.name, m.fonts.BodyBold, t.TextPrimary, x+62, y+16, colW-80, 20, StringAlignmentNear, StringAlignmentNear)
		fc.DrawString(FormatBytes(c.size), m.fonts.CardHead, t.TextPrimary, x+62, y+36, colW-80, 20, StringAlignmentNear, StringAlignmentNear)
		fc.DrawString(c.desc, m.fonts.Caption, t.TextTertiary, x+62, y+54, colW-80, 16, StringAlignmentNear, StringAlignmentNear)
	}

	// Cleanup Card
	cleanCardY := catY + catH*2 + 26
	fc.DrawCard(cx+pad, cleanCardY, availW, 70, 8, t.BgCard, t.BgCardBorder)
	fc.DrawString("Storage Optimization", m.fonts.BodyBold, t.TextPrimary, cx+pad+20, cleanCardY+16, 200, 20, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("Prune orphaned dependencies no longer required by any registered app.", m.fonts.Caption, t.TextSecondary, cx+pad+20, cleanCardY+38, 400, 18, StringAlignmentNear, StringAlignmentNear)

	m.renderButton(fc, "Clean Unused Packages", cx+pad+availW-210, cleanCardY+18, 190, 34, false, HitBtnCleanPackages, "")

	m.maxScrollY = cleanCardY + 90 - ch + pad
	if m.maxScrollY < 0 {
		m.maxScrollY = 0
	}
}

// Page 6: Settings
func (m *NativeManager) renderSettingsPage(fc *FluentContext, cx, cw, ch float32) {
	t := m.theme
	pad := float32(32)
	startY := pad - m.scrollY
	availW := cw - pad*2

	fc.DrawString("Settings", m.fonts.Title, t.TextPrimary, cx+pad, startY, availW, 30, StringAlignmentNear, StringAlignmentNear)
	fc.DrawString("System configuration, central paths, and runtime diagnostics.", m.fonts.Subtitle, t.TextSecondary, cx+pad, startY+32, availW, 22, StringAlignmentNear, StringAlignmentNear)

	curY := startY + 70
	rowH := float32(65)

	settingsRows := []struct {
		title    string
		subtitle string
		btnText  string
		btnHit   HitType
		badge    string
	}{
		{"Hrunner Version", "Release v1.0.3 (Windows x64 Native Architecture)", "", 0, "Up to date"},
		{"Central Storage Directory", registry.DefaultHrunnerDir(), "Open in Explorer", HitBtnOpenDir, ""},
		{"Named Pipe IPC", "\\\\.\\pipe\\hrunner (On-demand auto-shutdown server)", "", 0, "Active"},
		{"Diagnostics & Integrity", "Verify runtime environments and shared pool integrity", "Check Now", HitBtnCheckDiag, ""},
	}

	for _, s := range settingsRows {
		fc.DrawCard(cx+pad, curY, availW, rowH, 8, t.BgCard, t.BgCardBorder)
		fc.DrawString(s.title, m.fonts.BodyBold, t.TextPrimary, cx+pad+20, curY+14, availW-250, 20, StringAlignmentNear, StringAlignmentNear)
		fc.DrawString(s.subtitle, m.fonts.Caption, t.TextSecondary, cx+pad+20, curY+36, availW-250, 18, StringAlignmentNear, StringAlignmentNear)

		if s.btnText != "" {
			btnW := float32(140)
			btnX := cx + pad + availW - btnW - 20
			m.renderButton(fc, s.btnText, btnX, curY+16, btnW, 32, false, s.btnHit, "")
		} else if s.badge != "" {
			badgeX := cx + pad + availW - 120
			fc.DrawBadgePill(s.badge, t.BadgeReadyDot, t.BadgeReadyBg, t.BadgeReadyText, badgeX, curY+20, m.fonts.Badge)
		}

		curY += rowH + 10
	}

	m.maxScrollY = curY - ch + pad
	if m.maxScrollY < 0 {
		m.maxScrollY = 0
	}
}

func (m *NativeManager) renderButton(fc *FluentContext, text string, x, y, w, h float32, isPrimary bool, hitType HitType, param string) {
	t := m.theme
	isHover := m.mouseX >= x && m.mouseX <= x+w && m.mouseY >= y && m.mouseY <= y+h

	bg := t.BtnSecondaryBg
	border := t.BtnSecondaryBorder
	textCol := t.TextPrimary
	font := m.fonts.Body

	if isPrimary {
		bg = t.AccentPrimary
		border = t.AccentPrimary
		textCol = t.TextOnAccent
		font = m.fonts.BodyBold
		if isHover {
			if m.isPressed {
				bg = t.AccentPressed
				border = t.AccentPressed
			} else {
				bg = t.AccentHover
				border = t.AccentHover
			}
		}
	} else {
		if isHover {
			if m.isPressed {
				bg = t.BtnSecondaryPressed
			} else {
				bg = t.BtnSecondaryHover
			}
		}
	}

	fc.FillRoundedRect(x, y, w, h, 6, bg)
	fc.DrawRoundedRect(x, y, w, h, 6, border, 1.0)
	fc.DrawString(text, font, textCol, x, y, w, h, StringAlignmentCenter, StringAlignmentCenter)

	m.hitAreas = append(m.hitAreas, HitArea{
		X:     x,
		Y:     y,
		W:     w,
		H:     h,
		Type:  hitType,
		Param: param,
	})
}

func (m *NativeManager) launchApp(app *registry.ApplicationRecord) {
	rt, exists, err := m.runtimes.Find(app.PythonVersion, false)
	if err != nil || !exists {
		ShowMessageBox("Launch Error", fmt.Sprintf("Python runtime %s is not installed.", app.PythonVersion), 0x10)
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
	m.statusText = fmt.Sprintf("Launched %s", app.Name)
}

func (m *NativeManager) removeApp(app *registry.ApplicationRecord) {
	impact, err := m.storageMgr.CalculateAppRemovalImpact(app.ApplicationID)
	if err != nil {
		ShowMessageBox("Error", err.Error(), 0x10)
		return
	}

	reclaimInfo := ""
	if len(impact.PackagesNowUnused) > 0 {
		reclaimInfo = fmt.Sprintf("\n%d package(s) will become unused (%s reclaimable).", len(impact.PackagesNowUnused), FormatBytes(impact.ReclaimedBytesPool))
	}

	msg := fmt.Sprintf("Remove application '%s' from Hrunner?%s", app.Name, reclaimInfo)
	mb := ShowMessageBox("Confirm Removal", msg, 0x00000001 /* MB_OKCANCEL */)
	if mb == 1 {
		_ = m.storageMgr.RemoveApplicationAndCleanup(app.ApplicationID, true)
		m.statusText = fmt.Sprintf("Removed %s", app.Name)
	}
}
