//go:build windows

package render

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procSetWindowTextW   = user32.NewProc("SetWindowTextW")
	procStretchDIBits     = gdi32.NewProc("StretchDIBits")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	SW_SHOW             = 5
	PM_REMOVE           = 0x0001
	WM_QUIT             = 0x0012
	WM_DESTROY          = 0x0002
	WM_KEYDOWN          = 0x0100
	WM_KEYUP            = 0x0101
	WM_LBUTTONDOWN      = 0x0201
	WM_RBUTTONDOWN      = 0x0204
	WM_MOUSEMOVE        = 0x0200
	DIB_RGB_COLORS      = 0
	SRCCOPY             = 0x00CC0020
)

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
	Pt       struct{ X, Y int32 }
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
	Hwnd      uintptr
	Hdc       uintptr
	Width     int
	Height    int
	Closed    bool
	OnKey     func(vkCode int, isDown bool)
	OnMouse   func(x, y int, btn int)
	bmi       BITMAPINFO
}

var globalWnd *Window

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_DESTROY:
		if globalWnd != nil {
			globalWnd.Closed = true
		}
		procPostQuitMessage.Call(0)
		return 0
	case WM_KEYDOWN:
		if globalWnd != nil && globalWnd.OnKey != nil {
			globalWnd.OnKey(int(wparam), true)
		}
		return 0
	case WM_KEYUP:
		if globalWnd != nil && globalWnd.OnKey != nil {
			globalWnd.OnKey(int(wparam), false)
		}
		return 0
	case WM_LBUTTONDOWN:
		if globalWnd != nil && globalWnd.OnMouse != nil {
			x := int(int16(lparam & 0xFFFF))
			y := int(int16((lparam >> 16) & 0xFFFF))
			globalWnd.OnMouse(x, y, 1) // Left Click
		}
		return 0
	case WM_RBUTTONDOWN:
		if globalWnd != nil && globalWnd.OnMouse != nil {
			x := int(int16(lparam & 0xFFFF))
			y := int(int16((lparam >> 16) & 0xFFFF))
			globalWnd.OnMouse(x, y, 2) // Right Click
		}
		return 0
	case WM_MOUSEMOVE:
		if globalWnd != nil && globalWnd.OnMouse != nil {
			x := int(int16(lparam & 0xFFFF))
			y := int(int16((lparam >> 16) & 0xFFFF))
			globalWnd.OnMouse(x, y, 0)
		}
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

// NewNativeWindow creates a real graphical desktop window.
func NewNativeWindow(title string, width, height int) (*Window, error) {
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

	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass))); r == 0 {
		// Class might already be registered
	}

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		100, 100,
		uintptr(width+16), uintptr(height+39), // adjust for window border & title bar
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
		BiHeight:      -int32(height), // top-down bitmap
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: DIB_RGB_COLORS,
	}

	globalWnd = wnd
	return wnd, nil
}

// PollEvents pumps OS message queue; returns false if user closed window.
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

// Present blits the software framebuffer directly into the native window.
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

// Close destroys the window and releases DC.
func (w *Window) Close() {
	if !w.Closed {
		w.Closed = true
		if w.Hdc != 0 {
			procReleaseDC.Call(w.Hwnd, w.Hdc)
			w.Hdc = 0
		}
	}
}
