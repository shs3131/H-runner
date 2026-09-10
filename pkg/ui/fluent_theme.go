package ui

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modDwmapi                  = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute  = modDwmapi.NewProc("DwmSetWindowAttribute")

	modShcore                  = syscall.NewLazyDLL("shcore.dll")
	procSetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")
)

const (
	// DWM Window Attributes for Windows 11
	DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	DWMWA_WINDOW_CORNER_PREFERENCE = 33
	DWMWA_BORDER_COLOR            = 34
	DWMWA_CAPTION_COLOR           = 35
	DWMWA_TEXT_COLOR              = 36
	DWMWA_SYSTEMBACKDROP_TYPE     = 38
	DWMWA_MICA_EFFECT             = 1029

	// Window corner preferences
	DWMWCP_DEFAULT    = 0
	DWMWCP_DONOTROUND = 1
	DWMWCP_ROUND      = 2
	DWMWCP_ROUNDSMALL = 3

	// System Backdrop Types (Windows 11 Build 22621+)
	DWMSBT_AUTO            = 0
	DWMSBT_NONE            = 1
	DWMSBT_MAINWINDOW      = 2 // Mica
	DWMSBT_TRANSIENTWINDOW = 3 // Acrylic
	DWMSBT_TABBEDWINDOW    = 4 // Mica Alt
)

// ARGB color representation for GDI+
type ARGB uint32

func MakeARGB(a, r, g, b byte) ARGB {
	return ARGB((uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
}

func MakeRGB(r, g, b byte) ARGB {
	return MakeARGB(255, r, g, b)
}

// FluentTheme provides the Windows 11 Fluent / WinUI 3 color palette.
type FluentTheme struct {
	IsDark bool

	// Backgrounds
	BgWindow     ARGB
	BgSidebar    ARGB
	BgCard       ARGB
	BgCardHover  ARGB
	BgCardBorder ARGB

	// Typography colors
	TextPrimary   ARGB
	TextSecondary ARGB
	TextTertiary  ARGB
	TextOnAccent  ARGB

	// Accent Blue (Windows 11 Accent)
	AccentPrimary ARGB
	AccentHover   ARGB
	AccentPressed ARGB
	AccentSubtle  ARGB

	// Navigation
	NavHover    ARGB
	NavActive   ARGB
	NavActiveBar ARGB

	// Status Badges
	BadgeReadyBg   ARGB
	BadgeReadyText ARGB
	BadgeReadyDot  ARGB

	BadgeWarnBg   ARGB
	BadgeWarnText ARGB
	BadgeWarnDot  ARGB

	BadgeErrorBg   ARGB
	BadgeErrorText ARGB
	BadgeErrorDot  ARGB

	BadgeNeutralBg   ARGB
	BadgeNeutralText ARGB
	BadgeNeutralDot  ARGB

	// Button
	BtnSecondaryBg      ARGB
	BtnSecondaryHover   ARGB
	BtnSecondaryPressed ARGB
	BtnSecondaryBorder  ARGB

	// Storage Bar Colors
	StorageRuntimes ARGB
	StoragePackages ARGB
	StorageApps     ARGB
	StorageCache    ARGB
	StorageTrack    ARGB
}

// DefaultFluentTheme returns the Windows 11 Light theme matching Settings and Lossless Scaling.
func DefaultFluentTheme() *FluentTheme {
	return &FluentTheme{
		IsDark: false,

		// Surfaces
		BgWindow:     MakeRGB(243, 243, 243), // #F3F3F3 Mica-like base
		BgSidebar:    MakeRGB(238, 238, 238), // #EEEEEE Sidebar
		BgCard:       MakeRGB(255, 255, 255), // #FFFFFF Pure white card
		BgCardHover:  MakeRGB(250, 250, 250), // #FAFAFA Hovered card
		BgCardBorder: MakeRGB(229, 229, 229), // #E5E5E5 Subtle card border

		// Text
		TextPrimary:   MakeRGB(26, 26, 26),    // #1A1A1A High contrast
		TextSecondary: MakeRGB(94, 94, 94),    // #5E5E5E Muted subtitle
		TextTertiary:  MakeRGB(140, 140, 140), // #8C8C8C Muted caption
		TextOnAccent:  MakeRGB(255, 255, 255), // #FFFFFF

		// Accent (Win11 Blue)
		AccentPrimary: MakeRGB(0, 103, 192),   // #0067C0
		AccentHover:   MakeRGB(24, 123, 208),  // #187BD0
		AccentPressed: MakeRGB(0, 95, 184),    // #005FB8
		AccentSubtle:  MakeARGB(30, 0, 103, 192),

		// Navigation
		NavHover:     MakeRGB(230, 230, 230), // #E6E6E6
		NavActive:    MakeRGB(255, 255, 255), // #FFFFFF pill on sidebar
		NavActiveBar: MakeRGB(0, 103, 192),   // #0067C0 vertical bar

		// Status Pills
		BadgeReadyBg:   MakeRGB(223, 246, 221), // #DFF6DD Soft green
		BadgeReadyText: MakeRGB(16, 124, 16),   // #107C10 Green text
		BadgeReadyDot:  MakeRGB(16, 124, 16),

		BadgeWarnBg:   MakeRGB(255, 244, 206), // #FFF4CE Soft amber
		BadgeWarnText: MakeRGB(157, 93, 0),    // #9D5D00 Amber text
		BadgeWarnDot:  MakeRGB(157, 93, 0),

		BadgeErrorBg:   MakeRGB(253, 231, 233), // #FDE7E9 Soft red
		BadgeErrorText: MakeRGB(196, 43, 28),   // #C42B1C Red text
		BadgeErrorDot:  MakeRGB(196, 43, 28),

		BadgeNeutralBg:   MakeRGB(240, 240, 240),
		BadgeNeutralText: MakeRGB(94, 94, 94),
		BadgeNeutralDot:  MakeRGB(140, 140, 140),

		// Secondary buttons
		BtnSecondaryBg:      MakeRGB(255, 255, 255),
		BtnSecondaryHover:   MakeRGB(245, 245, 245),
		BtnSecondaryPressed: MakeRGB(235, 235, 235),
		BtnSecondaryBorder:  MakeRGB(224, 224, 224),

		// Storage chart segments
		StorageRuntimes: MakeRGB(0, 103, 192),   // Blue
		StoragePackages: MakeRGB(16, 137, 62),   // Green
		StorageApps:     MakeRGB(135, 100, 184), // Purple
		StorageCache:    MakeRGB(227, 115, 14),  // Orange
		StorageTrack:    MakeRGB(230, 230, 230),
	}
}

// ApplyWindows11DwmEffects enables Mica, rounded corners, and immersive theme styling.
func ApplyWindows11DwmEffects(hwnd uintptr, dark bool) {
	if procDwmSetWindowAttribute.Find() != nil {
		return
	}

	// 1. Set Windows 11 Rounded Window Corners
	cornerPref := uint32(DWMWCP_ROUND)
	_, _, _ = procDwmSetWindowAttribute.Call(
		hwnd,
		DWMWA_WINDOW_CORNER_PREFERENCE,
		uintptr(unsafe.Pointer(&cornerPref)),
		uintptr(unsafe.Sizeof(cornerPref)),
	)

	// 2. Set Immersive Dark/Light Mode
	darkMode := uint32(0)
	if dark {
		darkMode = 1
	}
	_, _, _ = procDwmSetWindowAttribute.Call(
		hwnd,
		DWMWA_USE_IMMERSIVE_DARK_MODE,
		uintptr(unsafe.Pointer(&darkMode)),
		uintptr(unsafe.Sizeof(darkMode)),
	)

	// 3. Set Mica Backdrop (Windows 11 Build 22621+)
	backdropType := uint32(DWMSBT_MAINWINDOW)
	hr, _, _ := procDwmSetWindowAttribute.Call(
		hwnd,
		DWMWA_SYSTEMBACKDROP_TYPE,
		uintptr(unsafe.Pointer(&backdropType)),
		uintptr(unsafe.Sizeof(backdropType)),
	)

	// Fallback for Windows 11 21H2 (Build 22000)
	if hr != 0 {
		micaTrue := uint32(1)
		_, _, _ = procDwmSetWindowAttribute.Call(
			hwnd,
			DWMWA_MICA_EFFECT,
			uintptr(unsafe.Pointer(&micaTrue)),
			uintptr(unsafe.Sizeof(micaTrue)),
		)
	}
}

// EnablePerMonitorHighDPI initializes Windows per-monitor v2 DPI awareness.
func EnablePerMonitorHighDPI() {
	if os.Getenv("HRUNNER_HEADLESS") == "1" {
		return
	}

	// Try SetProcessDpiAwarenessContext (Win 10 Creators Update+)
	procSetCtx := modUser32.NewProc("SetProcessDpiAwarenessContext")
	if procSetCtx.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
		_, _, _ = procSetCtx.Call(uintptr(0xfffffffffffffffc))
		return
	}

	// Fallback to SetProcessDpiAwareness (Win 8.1+)
	if procSetProcessDpiAwareness.Find() == nil {
		// PROCESS_PER_MONITOR_DPI_AWARE = 2
		_, _, _ = procSetProcessDpiAwareness.Call(2)
		return
	}

	// Fallback to SetProcessDPIAware (Vista+)
	procSetAware := modUser32.NewProc("SetProcessDPIAware")
	if procSetAware.Find() == nil {
		_, _, _ = procSetAware.Call()
	}
}
