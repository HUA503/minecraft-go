//go:build windows

package render

import (
	"fmt"
	"math"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procPeekMessageW        = user32.NewProc("PeekMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procSetCursorPos        = user32.NewProc("SetCursorPos")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procShowCursor          = user32.NewProc("ShowCursor")
	procClientToScreen      = user32.NewProc("ClientToScreen")
	procScreenToClient      = user32.NewProc("ScreenToClient")
	procStretchDIBits        = gdi32.NewProc("StretchDIBits")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetProcessDPIAware  = user32.NewProc("SetProcessDPIAware")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	SW_SHOW             = 5
	PM_REMOVE           = 0x0001
	WM_CLOSE            = 0x0010
	WM_QUIT             = 0x0012
	WM_DESTROY          = 0x0002
	WM_KEYDOWN          = 0x0100
	WM_KEYUP            = 0x0101
	WM_CHAR             = 0x0102
	WM_LBUTTONDOWN      = 0x0201
	WM_RBUTTONDOWN      = 0x0204
	WM_ACTIVATE         = 0x0006
	DIB_RGB_COLORS      = 0
	SRCCOPY             = 0x00CC0020
)

type POINT struct {
	X, Y int32
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
	Hwnd           uintptr
	Hdc            uintptr
	Width          int
	Height         int
	Closed         bool
	KeyDown        [256]bool
	mu             sync.Mutex
	OnKeyClick     func(vkCode int)
	OnMouseClick   func(btn int)
	OnCharInput    func(ch rune)
	MouseCaptured  bool
	centerScreenX  int32
	centerScreenY  int32
	bmi            BITMAPINFO
}

var globalWnd *Window

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_CLOSE:
		if globalWnd != nil {
			globalWnd.Closed = true
		}
		procPostQuitMessage.Call(0)
		return 0
	case WM_DESTROY:
		if globalWnd != nil {
			globalWnd.Closed = true
		}
		procPostQuitMessage.Call(0)
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
	case WM_KEYDOWN:
		vk := int(wparam)
		if vk >= 0 && vk < 256 && globalWnd != nil {
			globalWnd.mu.Lock()
			globalWnd.KeyDown[vk] = true
			globalWnd.mu.Unlock()
			if globalWnd.OnKeyClick != nil {
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
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

// NewNativeWindow creates a real graphical desktop window.
func NewNativeWindow(title string, width, height int) (*Window, error) {
	procSetProcessDPIAware.Call() // Enable 1:1 hardware pixel scaling

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

	// Sub-pixel jitter filter (prevents residual auto-drift when stationary)
	if math.Abs(dx) < 1.0 {
		dx = 0
	}
	if math.Abs(dy) < 1.0 {
		dy = 0
	}

	if dx != 0 || dy != 0 {
		procSetCursorPos.Call(uintptr(center.X), uintptr(center.Y))
	}

	return dx, dy
}

// GetCursorClientPos queries current cursor position mapped to client window coordinates.
func (w *Window) GetCursorClientPos() (int, int) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procScreenToClient.Call(w.Hwnd, uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

// IsMouseDown queries whether the given mouse button (1=Left, 2=Right) is currently physically pressed down.
func (w *Window) IsMouseDown(btn int) bool {
	if !w.MouseCaptured || w.Closed {
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
	// Only respond to key press if window is currently in the active foreground!
	fg, _, _ := procGetForegroundWindow.Call()
	if fg != w.Hwnd {
		return false
	}
	// Real-time physical key press state (high bit set = key physically pressed down right now)
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
	if w.Closed || w.Hdc == 0 {
		return
	}
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

func (w *Window) Close() {
	if !w.Closed {
		w.Closed = true
		if w.MouseCaptured {
			w.SetMouseCaptured(false)
		}
		if w.Hdc != 0 {
			procReleaseDC.Call(w.Hwnd, w.Hdc)
			w.Hdc = 0
		}
		procPostQuitMessage.Call(0)
	}
}
