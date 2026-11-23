//go:build windows

package render

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW              = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procUpdateWindow                  = user32.NewProc("UpdateWindow")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procPeekMessageW                  = user32.NewProc("PeekMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procSetCursorPos                  = user32.NewProc("SetCursorPos")
	procGetCursorPos                  = user32.NewProc("GetCursorPos")
	procShowCursor                    = user32.NewProc("ShowCursor")
	procClientToScreen                = user32.NewProc("ClientToScreen")
	procScreenToClient                = user32.NewProc("ScreenToClient")
	procStretchDIBits                 = gdi32.NewProc("StretchDIBits")
	procGetAsyncKeyState              = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procDisableProcessWindowsGhosting = user32.NewProc("DisableProcessWindowsGhosting")
	procBeginPaint                    = user32.NewProc("BeginPaint")
	procEndPaint                      = user32.NewProc("EndPaint")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procGetClientRect                 = user32.NewProc("GetClientRect")
	procGetWindowPlacement            = user32.NewProc("GetWindowPlacement")
	procSetWindowPlacement            = user32.NewProc("SetWindowPlacement")
	procGetWindowLongW                = user32.NewProc("GetWindowLongW")
	procSetWindowLongW                = user32.NewProc("SetWindowLongW")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	procMonitorFromWindow             = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
)

type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_POPUP            = 0x80000000
	GWL_STYLE           = ^uintptr(15) // Sign-extended (int32)-16 for 64-bit Windows
	HWND_TOP            = 0
	SWP_FRAMECHANGED    = 0x0020
	SWP_SHOWWINDOW      = 0x0040
	SWP_NOSIZE          = 0x0001
	SWP_NOMOVE          = 0x0002
	SWP_NOZORDER        = 0x0004
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
	SW_SHOW             = 5
	PM_REMOVE           = 0x0001
	WM_SIZE             = 0x0005
	WM_PAINT            = 0x000F
	WM_CLOSE            = 0x0010
	WM_QUIT             = 0x0012
	WM_ERASEBKGND       = 0x0014
	WM_DESTROY          = 0x0002
	WM_KILLFOCUS        = 0x0008
	WM_KEYDOWN          = 0x0100
	WM_KEYUP            = 0x0101
	WM_CHAR             = 0x0102
	WM_LBUTTONDOWN      = 0x0201
	WM_RBUTTONDOWN      = 0x0204
	WM_ACTIVATE         = 0x0006
	WM_MOUSEWHEEL       = 0x020A
	DIB_RGB_COLORS      = 0
	SRCCOPY             = 0x00CC0020
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

type POINT struct {
	X, Y int32
}

type WINDOWPLACEMENT struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    POINT
	PtMaxPosition    POINT
	RcNormalPosition RECT
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSG struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

// Window represents a native Win32 graphical game window.
type Window struct {
	Hwnd          uintptr
	Hdc           uintptr
	closeOnce     sync.Once
	Width         int
	Height        int
	Closed        bool
	KeyDown       [256]bool
	mu            sync.Mutex
	OnKeyClick    func(vkCode int)
	OnMouseClick  func(btn int)
	OnMouseWheel  func(delta int)
	OnCharInput   func(ch rune)
	MouseCaptured bool
	centerScreenX int32
	centerScreenY int32
	bmi           BITMAPINFO

	IsFullscreen   bool
	savedPlacement WINDOWPLACEMENT
	savedStyle     uint32
}

var globalWnd *Window

func wndProc(hwnd, msg, wparam, lparam uintptr) (result uintptr) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Window] Recovered from panic in wndProc: %v\n%s", r, debug.Stack())
			result = 0
		}
	}()
	switch msg {
	case WM_SIZE:
		if globalWnd != nil {
			var rc RECT
			procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			cw := int(rc.Right - rc.Left)
			ch := int(rc.Bottom - rc.Top)
			if cw > 0 && ch > 0 {
				globalWnd.mu.Lock()
				globalWnd.Width = cw
				globalWnd.Height = ch
				globalWnd.mu.Unlock()
				globalWnd.recenterMouse()
			}
		}
		return 0
	case WM_CLOSE:
		if globalWnd != nil {
			globalWnd.Closed = true
			if globalWnd.Hwnd != 0 {
				procDestroyWindow.Call(globalWnd.Hwnd)
			}
		}
		procPostQuitMessage.Call(0)
		return 0
	case WM_DESTROY:
		if globalWnd != nil {
			globalWnd.Closed = true
		}
		procPostQuitMessage.Call(0)
		return 0
	case WM_KILLFOCUS:
		if globalWnd != nil {
			if globalWnd.MouseCaptured {
				globalWnd.SetMouseCaptured(false)
			}
			globalWnd.mu.Lock()
			globalWnd.KeyDown = [256]bool{}
			globalWnd.mu.Unlock()
		}
		return 0
	case WM_ACTIVATE:
		active := (wparam & 0xFFFF) != 0
		if globalWnd != nil {
			if !active {
				if globalWnd.MouseCaptured {
					globalWnd.SetMouseCaptured(false)
				}
				globalWnd.mu.Lock()
				globalWnd.KeyDown = [256]bool{} // Clear keys on focus loss
				globalWnd.mu.Unlock()
			}
		}
		return 0
	case WM_MOUSEWHEEL:
		if globalWnd != nil && globalWnd.OnMouseWheel != nil {
			delta := int(int16(wparam >> 16))
			globalWnd.OnMouseWheel(delta)
		}
		return 0
	case WM_KEYDOWN:
		vk := int(wparam)
		isRepeat := (lparam & (1 << 30)) != 0
		if vk >= 0 && vk < 256 && globalWnd != nil {
			globalWnd.mu.Lock()
			globalWnd.KeyDown[vk] = true
			globalWnd.mu.Unlock()
			if globalWnd.OnKeyClick != nil && !isRepeat {
				globalWnd.OnKeyClick(vk)
			}
		}
		return 0
	case WM_KEYUP:
		vk := int(wparam)
		if vk >= 0 && vk < 256 && globalWnd != nil {
			globalWnd.mu.Lock()
			globalWnd.KeyDown[vk] = false
			globalWnd.mu.Unlock()
		}
		return 0
	case WM_CHAR:
		if globalWnd != nil && globalWnd.OnCharInput != nil {
			globalWnd.OnCharInput(rune(wparam))
		}
		return 0
	case WM_LBUTTONDOWN:
		if globalWnd != nil && globalWnd.OnMouseClick != nil {
			globalWnd.OnMouseClick(1)
		}
		return 0
	case WM_RBUTTONDOWN:
		if globalWnd != nil && globalWnd.OnMouseClick != nil {
			globalWnd.OnMouseClick(2)
		}
		return 0
	case WM_ERASEBKGND:
		return 1 // Prevent Win32 background erasing flicker and repaint floods
	case WM_PAINT:
		var ps [72]byte
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps[0])))
		if hdc != 0 {
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps[0])))
		}
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

// NewNativeWindow creates a real graphical desktop window.
func NewNativeWindow(title string, width, height int) (*Window, error) {
	runtime.LockOSThread()                   // Pin goroutine to native OS thread for reliable Win32 message pumping
	procSetProcessDPIAware.Call()            // Enable 1:1 hardware pixel scaling
	procDisableProcessWindowsGhosting.Call() // Prevent Windows from falsely marking application as "Not Responding"

	className, _ := syscall.UTF16PtrFromString("MinecraftGoClass")
	windowTitle, _ := syscall.UTF16PtrFromString(title)

	hInstance, _, _ := procGetModuleHandleW.Call(0)

	wndClass := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0x0003, // CS_HREDRAW | CS_VREDRAW
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInstance,
		LpszClassName: className,
	}

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		100, 100,
		uintptr(width+16), uintptr(height+39),
		0, 0, hInstance, 0,
	)

	if hwnd == 0 {
		return nil, fmt.Errorf("failed to create Win32 native window")
	}

	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	hdc, _, _ := procGetDC.Call(hwnd)

	wnd := &Window{
		Hwnd:   hwnd,
		Hdc:    hdc,
		Width:  width,
		Height: height,
	}

	wnd.bmi.BmiHeader = BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		BiWidth:       int32(width),
		BiHeight:      -int32(height),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: DIB_RGB_COLORS,
	}

	globalWnd = wnd

	// Auto capture mouse by default for true FPS gaming
	wnd.SetMouseCaptured(true)

	return wnd, nil
}

// SetMouseCaptured locks or releases the mouse cursor to/from the window center.
func (w *Window) SetMouseCaptured(capture bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.MouseCaptured = capture
	if capture {
		procShowCursor.Call(0) // Hide cursor
		w.recenterMouse()
	} else {
		procShowCursor.Call(1) // Show cursor
	}
}

func (w *Window) recenterMouse() {
	var center POINT
	center.X = int32(w.Width / 2)
	center.Y = int32(w.Height / 2)
	procClientToScreen.Call(w.Hwnd, uintptr(unsafe.Pointer(&center)))
	w.centerScreenX = center.X
	w.centerScreenY = center.Y
	procSetCursorPos.Call(uintptr(w.centerScreenX), uintptr(w.centerScreenY))
}

// PollMouseDelta calculates 1:1 mouse movement delta and re-centers cursor.
func (w *Window) PollMouseDelta() (float64, float64) {
	if !w.MouseCaptured || w.Closed {
		return 0, 0
	}
	// Only track mouse delta if this window is currently in the active foreground!
	fg, _, _ := procGetForegroundWindow.Call()
	if fg != w.Hwnd {
		return 0, 0
	}

	var center POINT
	center.X = int32(w.Width / 2)
	center.Y = int32(w.Height / 2)
	procClientToScreen.Call(w.Hwnd, uintptr(unsafe.Pointer(&center)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	dx := float64(pt.X - center.X)
	dy := float64(pt.Y - center.Y)

	// Sub-pixel jitter filter: clamp deltas under 1.5 pixels to zero
	// Completely eliminates auto-drift caused by Windows display scaling (125%/150%) rounding
	if math.Abs(dx) < 1.5 {
		dx = 0
	}
	if math.Abs(dy) < 1.5 {
		dy = 0
	}

	if dx != 0 || dy != 0 {
		procSetCursorPos.Call(uintptr(center.X), uintptr(center.Y))
	}

	return dx, dy
}

// GetCursorClientPos queries current cursor position mapped accurately to framebuffer pixel coordinates.
func (w *Window) GetCursorClientPos(frameW, frameH int) (int, int) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procScreenToClient.Call(w.Hwnd, uintptr(unsafe.Pointer(&pt)))

	var rect RECT
	procGetClientRect.Call(w.Hwnd, uintptr(unsafe.Pointer(&rect)))
	cw := rect.Right - rect.Left
	ch := rect.Bottom - rect.Top
	if cw <= 0 || ch <= 0 || frameW <= 0 || frameH <= 0 {
		return int(pt.X), int(pt.Y)
	}

	// Map client coordinates to framebuffer pixel space
	fx := int(pt.X) * frameW / int(cw)
	fy := int(pt.Y) * frameH / int(ch)
	return fx, fy
}

// IsMouseDown queries whether the given mouse button (1=Left, 2=Right) is currently physically pressed down.
func (w *Window) IsMouseDown(btn int) bool {
	if w.Closed {
		return false
	}
	fg, _, _ := procGetForegroundWindow.Call()
	if fg != w.Hwnd {
		return false
	}
	var vk int
	if btn == 1 {
		vk = 0x01 // VK_LBUTTON
	} else if btn == 2 {
		vk = 0x02 // VK_RBUTTON
	} else {
		return false
	}
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return (ret & 0x8000) != 0
}

func (w *Window) IsKeyDown(vkCode int) bool {
	if vkCode < 0 || vkCode >= 256 || w.Closed {
		return false
	}
	// 1. Check window message-pump tracked key state (most reliable for focused window)
	w.mu.Lock()
	down := w.KeyDown[vkCode]
	w.mu.Unlock()
	if down {
		return true
	}

	// 2. Real-time physical hardware key press state
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vkCode))
	return (ret & 0x8000) != 0
}

func (w *Window) PollEvents() bool {
	if w.Closed {
		return false
	}
	var msg MSG
	for {
		r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, PM_REMOVE)
		if r == 0 {
			break
		}
		if msg.Message == WM_QUIT {
			w.Closed = true
			return false
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return !w.Closed
}

func (w *Window) Present(fb *Framebuffer) {
	if w == nil || w.Closed || w.Hdc == 0 || fb == nil ||
		w.Width <= 0 || w.Height <= 0 || fb.Width <= 0 || fb.Height <= 0 {
		return
	}
	// StretchDIBits receives a pointer into a Go slice. Reject inconsistent
	// dimensions before crossing the native boundary so GDI cannot read past it.
	const maxBitmapDimension = int(^uint32(0) >> 1)
	if fb.Width > maxBitmapDimension || fb.Height > maxBitmapDimension ||
		fb.Width > len(fb.Color)/fb.Height {
		return
	}
	w.bmi.BmiHeader.BiWidth = int32(fb.Width)
	w.bmi.BmiHeader.BiHeight = -int32(fb.Height)
	procStretchDIBits.Call(
		w.Hdc,
		0, 0, uintptr(w.Width), uintptr(w.Height),
		0, 0, uintptr(fb.Width), uintptr(fb.Height),
		uintptr(unsafe.Pointer(&fb.Color[0])),
		uintptr(unsafe.Pointer(&w.bmi)),
		DIB_RGB_COLORS,
		SRCCOPY,
	)
}

// ToggleFullscreen toggles between standard windowed mode and seamless borderless fullscreen mode.
func (w *Window) ToggleFullscreen() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.IsFullscreen {
		// 1. Save current window placement and window styles
		w.savedPlacement.Length = uint32(unsafe.Sizeof(w.savedPlacement))
		procGetWindowPlacement.Call(w.Hwnd, uintptr(unsafe.Pointer(&w.savedPlacement)))
		style, _, _ := procGetWindowLongW.Call(w.Hwnd, uintptr(GWL_STYLE))
		w.savedStyle = uint32(style)

		// 2. Query monitor dimensions of active window
		hMon, _, _ := procMonitorFromWindow.Call(w.Hwnd, 2) // MONITOR_DEFAULTTONEAREST = 2
		var mi MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		procGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi)))

		monX := int(mi.RcMonitor.Left)
		monY := int(mi.RcMonitor.Top)
		monW := int(mi.RcMonitor.Right - mi.RcMonitor.Left)
		monH := int(mi.RcMonitor.Bottom - mi.RcMonitor.Top)
		if monW <= 0 || monH <= 0 {
			sw, _, _ := procGetSystemMetrics.Call(uintptr(SM_CXSCREEN))
			sh, _, _ := procGetSystemMetrics.Call(uintptr(SM_CYSCREEN))
			monX = 0
			monY = 0
			monW = int(sw)
			monH = int(sh)
		}

		// 3. Switch window style to borderless popup
		newStyle := (w.savedStyle &^ WS_OVERLAPPEDWINDOW) | WS_POPUP | WS_VISIBLE
		procSetWindowLongW.Call(w.Hwnd, uintptr(GWL_STYLE), uintptr(newStyle))

		// 4. Position window to cover entire active monitor
		procSetWindowPos.Call(
			w.Hwnd,
			uintptr(HWND_TOP),
			uintptr(monX), uintptr(monY), uintptr(monW), uintptr(monH),
			uintptr(SWP_FRAMECHANGED|SWP_SHOWWINDOW),
		)

		w.IsFullscreen = true
		w.Width = monW
		w.Height = monH
	} else {
		// 1. Restore standard window style
		procSetWindowLongW.Call(w.Hwnd, uintptr(GWL_STYLE), uintptr(w.savedStyle))

		// 2. Restore previous window placement
		procSetWindowPlacement.Call(w.Hwnd, uintptr(unsafe.Pointer(&w.savedPlacement)))

		// 3. Notify Windows of frame change
		procSetWindowPos.Call(
			w.Hwnd,
			0, 0, 0, 0, 0,
			uintptr(SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED),
		)

		var rc RECT
		procGetClientRect.Call(w.Hwnd, uintptr(unsafe.Pointer(&rc)))
		cw := int(rc.Right - rc.Left)
		ch := int(rc.Bottom - rc.Top)
		if cw > 0 && ch > 0 {
			w.Width = cw
			w.Height = ch
		}

		w.IsFullscreen = false
	}
	w.recenterMouse()
}

func (w *Window) Close() {
	w.closeOnce.Do(func() {
		w.Closed = true
		if w.MouseCaptured {
			w.SetMouseCaptured(false)
		}
		if w.Hdc != 0 {
			procReleaseDC.Call(w.Hwnd, w.Hdc)
			w.Hdc = 0
		}
		if w.Hwnd != 0 {
			procDestroyWindow.Call(w.Hwnd)
			w.Hwnd = 0
		}
		if globalWnd == w {
			globalWnd = nil
		}
		procPostQuitMessage.Call(0)
	})
}
