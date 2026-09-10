package ui

import (
	"math"
	"syscall"
	"unsafe"
)

var (
	modGdi32Win              = syscall.NewLazyDLL("gdi32.dll")
	procCreateCompatibleDC   = modGdi32Win.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGdi32Win.NewProc("CreateCompatibleBitmap")
	procSelectObject         = modGdi32Win.NewProc("SelectObject")
	procDeleteObject         = modGdi32Win.NewProc("DeleteObject")
	procDeleteDC             = modGdi32Win.NewProc("DeleteDC")
	procBitBlt               = modGdi32Win.NewProc("BitBlt")

	modGdiplusWin            = syscall.NewLazyDLL("gdiplus.dll")
	procGdiplusStartup       = modGdiplusWin.NewProc("GdiplusStartup")
	procGdiplusShutdown      = modGdiplusWin.NewProc("GdiplusShutdown")
	procGdipCreateFromHDC    = modGdiplusWin.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics   = modGdiplusWin.NewProc("GdipDeleteGraphics")
	procGdipSetSmoothingMode = modGdiplusWin.NewProc("GdipSetSmoothingMode")
	procGdipSetTextRendering = modGdiplusWin.NewProc("GdipSetTextRenderingHint")
	procGdipCreateSolidFill  = modGdiplusWin.NewProc("GdipCreateSolidFill")
	procGdipDeleteBrush      = modGdiplusWin.NewProc("GdipDeleteBrush")
	procGdipCreatePen1       = modGdiplusWin.NewProc("GdipCreatePen1")
	procGdipDeletePen        = modGdiplusWin.NewProc("GdipDeletePen")
	procGdipCreatePath       = modGdiplusWin.NewProc("GdipCreatePath")
	procGdipDeletePath       = modGdiplusWin.NewProc("GdipDeletePath")
	procGdipResetPath        = modGdiplusWin.NewProc("GdipResetPath")
	procGdipAddPathArc       = modGdiplusWin.NewProc("GdipAddPathArc")
	procGdipAddPathLine      = modGdiplusWin.NewProc("GdipAddPathLine")
	procGdipClosePathFigure  = modGdiplusWin.NewProc("GdipClosePathFigure")
	procGdipFillPath         = modGdiplusWin.NewProc("GdipFillPath")
	procGdipDrawPath         = modGdiplusWin.NewProc("GdipDrawPath")
	procGdipFillRectangle    = modGdiplusWin.NewProc("GdipFillRectangle")
	procGdipDrawRectangle    = modGdiplusWin.NewProc("GdipDrawRectangle")
	procGdipDrawEllipse      = modGdiplusWin.NewProc("GdipDrawEllipse")
	procGdipFillEllipse      = modGdiplusWin.NewProc("GdipFillEllipse")
	procGdipDrawLine         = modGdiplusWin.NewProc("GdipDrawLine")
	procGdipCreateFontFamily = modGdiplusWin.NewProc("GdipCreateFontFamilyFromName")
	procGdipDeleteFontFamily = modGdiplusWin.NewProc("GdipDeleteFontFamily")
	procGdipCreateFont       = modGdiplusWin.NewProc("GdipCreateFont")
	procGdipDeleteFont       = modGdiplusWin.NewProc("GdipDeleteFont")
	procGdipDrawString       = modGdiplusWin.NewProc("GdipDrawString")
	procGdipMeasureString    = modGdiplusWin.NewProc("GdipMeasureString")
	procGdipCreateStringFormat = modGdiplusWin.NewProc("GdipCreateStringFormat")
	procGdipDeleteStringFormat = modGdiplusWin.NewProc("GdipDeleteStringFormat")
	procGdipSetStringFormatAlign = modGdiplusWin.NewProc("GdipSetStringFormatAlign")
	procGdipSetStringFormatLineAlign = modGdiplusWin.NewProc("GdipSetStringFormatLineAlign")
)

const (
	SmoothingModeAntiAlias      = 2
	TextRenderingHintClearType  = 5
	FillModeAlternate           = 0
	UnitPixel                   = 2
	FontStyleRegular            = 0
	FontStyleBold               = 1
	FontStyleItalic             = 2
	FontStyleBoldItalic         = 3
	StringAlignmentNear         = 0
	StringAlignmentCenter       = 1
	StringAlignmentFar          = 2
	SRCCOPY                     = 0x00CC0020
)

type GdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread bool
	SuppressExternalCodecs   bool
}

type RectF struct {
	X, Y, Width, Height float32
}

type IconType int

const (
	IconHome IconType = iota + 1
	IconApplications
	IconRuntimes
	IconPackages
	IconStorage
	IconSettings
	IconPlay
	IconTrash
	IconRefresh
	IconFolder
	IconCheck
	IconChevron
	IconClose
	IconAppPill
)

type StorageSegment struct {
	Label string
	Bytes int64
	Color ARGB
}

var gdiplusToken uintptr

func InitGdiplus() {
	if gdiplusToken != 0 {
		return
	}
	input := GdiplusStartupInput{GdiplusVersion: 1}
	_ = procGdiplusStartup.Find()
	_, _, _ = procGdiplusStartup.Call(
		uintptr(unsafe.Pointer(&gdiplusToken)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
}

func ShutdownGdiplus() {
	if gdiplusToken != 0 {
		_, _, _ = procGdiplusShutdown.Call(gdiplusToken)
		gdiplusToken = 0
	}
}

// FluentFont holds GDI+ font handles
type FluentFont struct {
	family uintptr
	font   uintptr
	size   float32
}

func (f *FluentFont) Close() {
	if f.font != 0 {
		procGdipDeleteFont.Call(f.font)
		f.font = 0
	}
	if f.family != 0 {
		procGdipDeleteFontFamily.Call(f.family)
		f.family = 0
	}
}

type FontSet struct {
	Title    *FluentFont
	Subtitle *FluentFont
	CardHead *FluentFont
	Body     *FluentFont
	BodyBold *FluentFont
	Caption  *FluentFont
	Badge    *FluentFont
}

func (fs *FontSet) Close() {
	if fs.Title != nil {
		fs.Title.Close()
	}
	if fs.Subtitle != nil {
		fs.Subtitle.Close()
	}
	if fs.CardHead != nil {
		fs.CardHead.Close()
	}
	if fs.Body != nil {
		fs.Body.Close()
	}
	if fs.BodyBold != nil {
		fs.BodyBold.Close()
	}
	if fs.Caption != nil {
		fs.Caption.Close()
	}
	if fs.Badge != nil {
		fs.Badge.Close()
	}
}

func createFont(names []string, sizePt float32, style int) *FluentFont {
	var family uintptr
	for _, name := range names {
		pName, _ := syscall.UTF16PtrFromString(name)
		st, _, _ := procGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(pName)), 0, uintptr(unsafe.Pointer(&family)))
		if st == 0 && family != 0 {
			break
		}
	}
	if family == 0 {
		pArial, _ := syscall.UTF16PtrFromString("Arial")
		procGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(pArial)), 0, uintptr(unsafe.Pointer(&family)))
	}

	var font uintptr
	// size in pixels = sizePt * 96 / 72
	pixelSize := sizePt * 1.3333
	procGdipCreateFont.Call(family, uintptr(math.Float32bits(pixelSize)), uintptr(style), UnitPixel, uintptr(unsafe.Pointer(&font)))

	return &FluentFont{
		family: family,
		font:   font,
		size:   sizePt,
	}
}

func LoadFluentFonts(scale float32) *FontSet {
	names := []string{"Segoe UI Variable Display", "Segoe UI Variable Text", "Segoe UI", "Arial"}

	return &FontSet{
		Title:    createFont(names, 18.0*scale, FontStyleBold),
		Subtitle: createFont(names, 12.0*scale, FontStyleRegular),
		CardHead: createFont(names, 13.0*scale, FontStyleBold),
		Body:     createFont(names, 10.0*scale, FontStyleRegular),
		BodyBold: createFont(names, 10.0*scale, FontStyleBold),
		Caption:  createFont(names, 8.5*scale, FontStyleRegular),
		Badge:    createFont(names, 8.5*scale, FontStyleBold),
	}
}

// FluentContext manages double-buffered rendering.
type FluentContext struct {
	hdcMem    uintptr
	hbmMem    uintptr
	hbmOld    uintptr
	graphics  uintptr
	width     int32
	height    int32
	scale     float32
	strFormat uintptr
}

func NewFluentContext(hdc uintptr, width, height int32, scale float32) *FluentContext {
	InitGdiplus()

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdc)
	hbmMem, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(width), uintptr(height))
	hbmOld, _, _ := procSelectObject.Call(hdcMem, hbmMem)

	var graphics uintptr
	procGdipCreateFromHDC.Call(hdcMem, uintptr(unsafe.Pointer(&graphics)))
	procGdipSetSmoothingMode.Call(graphics, SmoothingModeAntiAlias)
	procGdipSetTextRendering.Call(graphics, TextRenderingHintClearType)

	var strFormat uintptr
	procGdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&strFormat)))

	return &FluentContext{
		hdcMem:    hdcMem,
		hbmMem:    hbmMem,
		hbmOld:    hbmOld,
		graphics:  graphics,
		width:     width,
		height:    height,
		scale:     scale,
		strFormat: strFormat,
	}
}

func (fc *FluentContext) Close(hdcDst uintptr) {
	if fc.graphics != 0 {
		procBitBlt.Call(hdcDst, 0, 0, uintptr(fc.width), uintptr(fc.height), fc.hdcMem, 0, 0, SRCCOPY)
		procGdipDeleteGraphics.Call(fc.graphics)
	}
	if fc.strFormat != 0 {
		procGdipDeleteStringFormat.Call(fc.strFormat)
	}
	if fc.hbmOld != 0 {
		procSelectObject.Call(fc.hdcMem, fc.hbmOld)
	}
	if fc.hbmMem != 0 {
		procDeleteObject.Call(fc.hbmMem)
	}
	if fc.hdcMem != 0 {
		procDeleteDC.Call(fc.hdcMem)
	}
}

func (fc *FluentContext) Clear(color ARGB) {
	var brush uintptr
	procGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	procGdipFillRectangle.Call(
		fc.graphics, brush,
		0, 0,
		uintptr(math.Float32bits(float32(fc.width))),
		uintptr(math.Float32bits(float32(fc.height))),
	)
	procGdipDeleteBrush.Call(brush)
}

func (fc *FluentContext) FillRect(x, y, w, h float32, color ARGB) {
	var brush uintptr
	procGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	procGdipFillRectangle.Call(
		fc.graphics, brush,
		uintptr(math.Float32bits(x)),
		uintptr(math.Float32bits(y)),
		uintptr(math.Float32bits(w)),
		uintptr(math.Float32bits(h)),
	)
	procGdipDeleteBrush.Call(brush)
}

func (fc *FluentContext) createRoundedPath(x, y, w, h, radius float32) uintptr {
	var path uintptr
	procGdipCreatePath.Call(FillModeAlternate, uintptr(unsafe.Pointer(&path)))

	d := radius * 2
	// Top Left
	procGdipAddPathArc.Call(path, uintptr(math.Float32bits(x)), uintptr(math.Float32bits(y)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(180)), uintptr(math.Float32bits(90)))
	// Top edge & Top Right
	procGdipAddPathArc.Call(path, uintptr(math.Float32bits(x+w-d)), uintptr(math.Float32bits(y)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(270)), uintptr(math.Float32bits(90)))
	// Right edge & Bottom Right
	procGdipAddPathArc.Call(path, uintptr(math.Float32bits(x+w-d)), uintptr(math.Float32bits(y+h-d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(0)), uintptr(math.Float32bits(90)))
	// Bottom edge & Bottom Left
	procGdipAddPathArc.Call(path, uintptr(math.Float32bits(x)), uintptr(math.Float32bits(y+h-d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(90)), uintptr(math.Float32bits(90)))
	procGdipClosePathFigure.Call(path)

	return path
}

func (fc *FluentContext) FillRoundedRect(x, y, w, h, radius float32, color ARGB) {
	path := fc.createRoundedPath(x, y, w, h, radius)
	var brush uintptr
	procGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	procGdipFillPath.Call(fc.graphics, brush, path)
	procGdipDeleteBrush.Call(brush)
	procGdipDeletePath.Call(path)
}

func (fc *FluentContext) DrawRoundedRect(x, y, w, h, radius float32, color ARGB, stroke float32) {
	path := fc.createRoundedPath(x, y, w, h, radius)
	var pen uintptr
	procGdipCreatePen1.Call(uintptr(color), uintptr(math.Float32bits(stroke)), UnitPixel, uintptr(unsafe.Pointer(&pen)))
	procGdipDrawPath.Call(fc.graphics, pen, path)
	procGdipDeletePen.Call(pen)
	procGdipDeletePath.Call(path)
}

func (fc *FluentContext) DrawCard(x, y, w, h, radius float32, bg, border ARGB) {
	fc.FillRoundedRect(x, y, w, h, radius, bg)
	fc.DrawRoundedRect(x, y, w, h, radius, border, 1.0)
}

func (fc *FluentContext) DrawString(text string, font *FluentFont, color ARGB, x, y, w, h float32, alignH, alignV int) {
	if font == nil || font.font == 0 || text == "" {
		return
	}

	pText, _ := syscall.UTF16PtrFromString(text)
	var brush uintptr
	procGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))

	procGdipSetStringFormatAlign.Call(fc.strFormat, uintptr(alignH))
	procGdipSetStringFormatLineAlign.Call(fc.strFormat, uintptr(alignV))

	rect := RectF{X: x, Y: y, Width: w, Height: h}
	procGdipDrawString.Call(
		fc.graphics,
		uintptr(unsafe.Pointer(pText)),
		uintptr(len(text)),
		font.font,
		uintptr(unsafe.Pointer(&rect)),
		fc.strFormat,
		brush,
	)

	procGdipDeleteBrush.Call(brush)
}

func (fc *FluentContext) MeasureString(text string, font *FluentFont, maxW float32) (float32, float32) {
	if font == nil || font.font == 0 || text == "" {
		return 0, 0
	}
	pText, _ := syscall.UTF16PtrFromString(text)
	inRect := RectF{X: 0, Y: 0, Width: maxW, Height: 1000}
	var outRect RectF
	procGdipMeasureString.Call(
		fc.graphics,
		uintptr(unsafe.Pointer(pText)),
		uintptr(len(text)),
		font.font,
		uintptr(unsafe.Pointer(&inRect)),
		fc.strFormat,
		uintptr(unsafe.Pointer(&outRect)),
		0, 0,
	)
	return outRect.Width, outRect.Height
}

// DrawBadgePill draws a Windows 11 status pill badge (e.g. ● Ready).
func (fc *FluentContext) DrawBadgePill(text string, dotColor, bgColor, textColor ARGB, x, y float32, font *FluentFont) float32 {
	tw, _ := fc.MeasureString(text, font, 200)
	pillH := float32(22.0)
	dotD := float32(7.0)
	pillW := tw + 24.0

	// Background pill
	fc.FillRoundedRect(x, y, pillW, pillH, pillH/2.0, bgColor)

	// Status dot
	var brush uintptr
	procGdipCreateSolidFill.Call(uintptr(dotColor), uintptr(unsafe.Pointer(&brush)))
	procGdipFillEllipse.Call(
		fc.graphics, brush,
		uintptr(math.Float32bits(x+8)),
		uintptr(math.Float32bits(y+(pillH-dotD)/2)),
		uintptr(math.Float32bits(dotD)),
		uintptr(math.Float32bits(dotD)),
	)
	procGdipDeleteBrush.Call(brush)

	// Text
	fc.DrawString(text, font, textColor, x+18, y+1, tw+4, pillH, StringAlignmentNear, StringAlignmentCenter)

	return pillW
}

// DrawSegmentedBar renders a Windows 11 multi-segment storage bar.
func (fc *FluentContext) DrawSegmentedBar(segments []StorageSegment, totalBytes int64, x, y, w, h float32, track ARGB) {
	// Draw background track
	fc.FillRoundedRect(x, y, w, h, h/2.0, track)

	if totalBytes <= 0 {
		return
	}

	curX := x
	radius := h / 2.0
	for idx, seg := range segments {
		if seg.Bytes <= 0 {
			continue
		}
		pct := float32(seg.Bytes) / float32(totalBytes)
		segW := w * pct
		if segW < 3.0 {
			segW = 3.0
		}
		if curX+segW > x+w {
			segW = x + w - curX
		}

		if idx == 0 {
			fc.FillRoundedRect(curX, y, segW+radius, h, radius, seg.Color)
		} else {
			fc.FillRect(curX, y, segW, h, seg.Color)
		}
		curX += segW
	}
}

// DrawFluentIcon renders crisp, anti-aliased vector Fluent iconography.
func (fc *FluentContext) DrawFluentIcon(icon IconType, x, y, size float32, color ARGB) {
	var pen uintptr
	stroke := float32(1.8)
	procGdipCreatePen1.Call(uintptr(color), uintptr(math.Float32bits(stroke)), UnitPixel, uintptr(unsafe.Pointer(&pen)))
	defer procGdipDeletePen.Call(pen)

	var brush uintptr
	procGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
	defer procGdipDeleteBrush.Call(brush)

	switch icon {
	case IconHome:
		// House silhouette
		h := size
		midX := x + size/2
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(midX)), uintptr(math.Float32bits(y)), uintptr(math.Float32bits(x+size)), uintptr(math.Float32bits(y+h*0.45)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(midX)), uintptr(math.Float32bits(y)), uintptr(math.Float32bits(x)), uintptr(math.Float32bits(y+h*0.45)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.18)), uintptr(math.Float32bits(y+h*0.40)), uintptr(math.Float32bits(x+size*0.18)), uintptr(math.Float32bits(y+h)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.82)), uintptr(math.Float32bits(y+h*0.40)), uintptr(math.Float32bits(x+size*0.82)), uintptr(math.Float32bits(y+h)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.18)), uintptr(math.Float32bits(y+h)), uintptr(math.Float32bits(x+size*0.82)), uintptr(math.Float32bits(y+h)))
		// Door
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(midX-size*0.15)), uintptr(math.Float32bits(y+h)), uintptr(math.Float32bits(midX-size*0.15)), uintptr(math.Float32bits(y+h*0.65)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(midX+size*0.15)), uintptr(math.Float32bits(y+h)), uintptr(math.Float32bits(midX+size*0.15)), uintptr(math.Float32bits(y+h*0.65)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(midX-size*0.15)), uintptr(math.Float32bits(y+h*0.65)), uintptr(math.Float32bits(midX+size*0.15)), uintptr(math.Float32bits(y+h*0.65)))

	case IconApplications:
		// 4 App tiles (Fluent grid)
		gap := float32(2.5)
		s := (size - gap) / 2
		fc.FillRoundedRect(x, y, s, s, 2, color)
		fc.FillRoundedRect(x+s+gap, y, s, s, 2, color)
		fc.FillRoundedRect(x, y+s+gap, s, s, 2, color)
		fc.FillRoundedRect(x+s+gap, y+s+gap, s, s, 2, color)

	case IconRuntimes:
		// Processor / Chip silhouette
		inset := size * 0.2
		fc.DrawRoundedRect(x+inset, y+inset, size-inset*2, size-inset*2, 2.5, color, stroke)
		// Pins
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.35)), uintptr(math.Float32bits(y)), uintptr(math.Float32bits(x+size*0.35)), uintptr(math.Float32bits(y+inset)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.65)), uintptr(math.Float32bits(y)), uintptr(math.Float32bits(x+size*0.65)), uintptr(math.Float32bits(y+inset)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.35)), uintptr(math.Float32bits(y+size-inset)), uintptr(math.Float32bits(x+size*0.35)), uintptr(math.Float32bits(y+size)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.65)), uintptr(math.Float32bits(y+size-inset)), uintptr(math.Float32bits(x+size*0.65)), uintptr(math.Float32bits(y+size)))
		// Horizontal Pins
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x)), uintptr(math.Float32bits(y+size*0.35)), uintptr(math.Float32bits(x+inset)), uintptr(math.Float32bits(y+size*0.35)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x)), uintptr(math.Float32bits(y+size*0.65)), uintptr(math.Float32bits(x+inset)), uintptr(math.Float32bits(y+size*0.65)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size-inset)), uintptr(math.Float32bits(y+size*0.35)), uintptr(math.Float32bits(x+size)), uintptr(math.Float32bits(y+size*0.35)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size-inset)), uintptr(math.Float32bits(y+size*0.65)), uintptr(math.Float32bits(x+size)), uintptr(math.Float32bits(y+size*0.65)))

	case IconPackages:
		// 3D Package Cube
		mx := x + size/2
		my := y + size*0.1
		bx := x + size/2
		by := y + size*0.9
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(mx)), uintptr(math.Float32bits(my)), uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.32)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(mx)), uintptr(math.Float32bits(my)), uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.32)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.32)), uintptr(math.Float32bits(bx)), uintptr(math.Float32bits(y+size*0.55)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.32)), uintptr(math.Float32bits(bx)), uintptr(math.Float32bits(y+size*0.55)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(bx)), uintptr(math.Float32bits(y+size*0.55)), uintptr(math.Float32bits(bx)), uintptr(math.Float32bits(by)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.32)), uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.70)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.32)), uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.70)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.70)), uintptr(math.Float32bits(bx)), uintptr(math.Float32bits(by)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.70)), uintptr(math.Float32bits(bx)), uintptr(math.Float32bits(by)))

	case IconStorage:
		// Hard drive stack
		h := size * 0.26
		fc.DrawRoundedRect(x, y+size*0.15, size, h, 2, color, stroke)
		fc.DrawRoundedRect(x, y+size*0.55, size, h, 2, color, stroke)
		// Drive lights
		procGdipFillEllipse.Call(fc.graphics, brush, uintptr(math.Float32bits(x+size*0.8)), uintptr(math.Float32bits(y+size*0.24)), uintptr(math.Float32bits(3)), uintptr(math.Float32bits(3)))
		procGdipFillEllipse.Call(fc.graphics, brush, uintptr(math.Float32bits(x+size*0.8)), uintptr(math.Float32bits(y+size*0.64)), uintptr(math.Float32bits(3)), uintptr(math.Float32bits(3)))

	case IconSettings:
		// Modern Gear
		fc.DrawRoundedRect(x+size*0.25, y+size*0.25, size*0.5, size*0.5, size*0.25, color, stroke)
		// 8 teeth around circumference
		for i := 0; i < 8; i++ {
			ang := float64(i) * math.Pi / 4.0
			cx := x + size/2
			cy := y + size/2
			x1 := float32(float64(cx) + math.Cos(ang)*float64(size*0.32))
			y1 := float32(float64(cy) + math.Sin(ang)*float64(size*0.32))
			x2 := float32(float64(cx) + math.Cos(ang)*float64(size*0.48))
			y2 := float32(float64(cy) + math.Sin(ang)*float64(size*0.48))
			procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x1)), uintptr(math.Float32bits(y1)), uintptr(math.Float32bits(x2)), uintptr(math.Float32bits(y2)))
		}

	case IconPlay:
		// Play triangle
		var path uintptr
		procGdipCreatePath.Call(FillModeAlternate, uintptr(unsafe.Pointer(&path)))
		procGdipAddPathLine.Call(path, uintptr(math.Float32bits(x+size*0.25)), uintptr(math.Float32bits(y+size*0.15)), uintptr(math.Float32bits(x+size*0.85)), uintptr(math.Float32bits(y+size*0.5)))
		procGdipAddPathLine.Call(path, uintptr(math.Float32bits(x+size*0.85)), uintptr(math.Float32bits(y+size*0.5)), uintptr(math.Float32bits(x+size*0.25)), uintptr(math.Float32bits(y+size*0.85)))
		procGdipClosePathFigure.Call(path)
		procGdipFillPath.Call(fc.graphics, brush, path)
		procGdipDeletePath.Call(path)

	case IconTrash:
		// Trashcan
		top := y + size*0.22
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(top)), uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(top)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.35)), uintptr(math.Float32bits(y+size*0.12)), uintptr(math.Float32bits(x+size*0.65)), uintptr(math.Float32bits(y+size*0.12)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.2)), uintptr(math.Float32bits(top)), uintptr(math.Float32bits(x+size*0.25)), uintptr(math.Float32bits(y+size*0.9)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.8)), uintptr(math.Float32bits(top)), uintptr(math.Float32bits(x+size*0.75)), uintptr(math.Float32bits(y+size*0.9)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.25)), uintptr(math.Float32bits(y+size*0.9)), uintptr(math.Float32bits(x+size*0.75)), uintptr(math.Float32bits(y+size*0.9)))

	case IconRefresh:
		// Circle arrow
		d := size * 0.75
		cx := x + (size-d)/2
		cy := y + (size-d)/2
		procGdipAddPathArc.Call(0, uintptr(math.Float32bits(cx)), uintptr(math.Float32bits(cy)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(45)), uintptr(math.Float32bits(270)))
		procGdipDrawArc := modGdiplusWin.NewProc("GdipDrawArc")
		procGdipDrawArc.Call(fc.graphics, pen, uintptr(math.Float32bits(cx)), uintptr(math.Float32bits(cy)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(d)), uintptr(math.Float32bits(45)), uintptr(math.Float32bits(270)))
		// Arrowhead
		ax := cx + d - 1
		ay := cy + d/2
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(ax-3)), uintptr(math.Float32bits(ay-4)), uintptr(math.Float32bits(ax+2)), uintptr(math.Float32bits(ay)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(ax-3)), uintptr(math.Float32bits(ay+4)), uintptr(math.Float32bits(ax+2)), uintptr(math.Float32bits(ay)))

	case IconFolder:
		// Folder silhouette
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.25)), uintptr(math.Float32bits(x+size*0.45)), uintptr(math.Float32bits(y+size*0.25)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.45)), uintptr(math.Float32bits(y+size*0.25)), uintptr(math.Float32bits(x+size*0.55)), uintptr(math.Float32bits(y+size*0.35)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.55)), uintptr(math.Float32bits(y+size*0.35)), uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.35)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.35)), uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.85)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.9)), uintptr(math.Float32bits(y+size*0.85)), uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.85)))
		procGdipDrawLine.Call(fc.graphics, pen, uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.85)), uintptr(math.Float32bits(x+size*0.1)), uintptr(math.Float32bits(y+size*0.25)))

	case IconAppPill:
		// Beautiful colored app avatar square
		fc.FillRoundedRect(x, y, size, size, 6, color)
		// Inner Python/terminal glyph
		dot := size * 0.22
		fc.FillRoundedRect(x+size*0.22, y+size*0.22, dot, dot, 2, MakeRGB(255, 255, 255))
		fc.FillRoundedRect(x+size*0.56, y+size*0.56, dot, dot, 2, MakeRGB(255, 255, 255))
	}
}
