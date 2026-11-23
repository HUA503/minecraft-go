//go:build !windows

package render

import "fmt"

// Window represents a window stub on non-Windows platforms.
type Window struct {
	Closed        bool
	Width         int
	Height        int
	OnKeyClick    func(vkCode int)
	OnMouseClick  func(btn int)
	OnMouseWheel  func(delta int)
	OnCharInput   func(ch rune)
	OnPanic       func(message string, recovered any, stack []byte)
	MouseCaptured bool
	IsFullscreen  bool
}

// NewNativeWindow returns an error on non-Windows platforms advising users to use CLI mode.
func NewNativeWindow(title string, width, height int) (*Window, error) {
	return nil, fmt.Errorf("native graphical desktop window is currently only supported on Windows (use -cli flag for terminal CLI mode)")
}

func (w *Window) SetMouseCaptured(capture bool)                                             {}
func (w *Window) SetPanicHandler(handler func(message string, recovered any, stack []byte)) {}
func (w *Window) ToggleFullscreen()                                                         {}
func (w *Window) PollMouseDelta() (float64, float64)                                        { return 0, 0 }
func (w *Window) GetCursorClientPos(frameW, frameH int) (int, int)                          { return 0, 0 }
func (w *Window) IsMouseDown(btn int) bool                                                  { return false }
func (w *Window) IsKeyDown(vkCode int) bool                                                 { return false }
func (w *Window) PollEvents() bool                                                          { return false }
func (w *Window) Present(fb *Framebuffer)                                                   {}
func (w *Window) Close()                                                                    { w.Closed = true }
