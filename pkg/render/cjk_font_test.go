package render

import (
	"testing"
)

func TestCJKFontRenderingAndMeasurement(t *testing.T) {
	fb := NewFramebuffer(320, 240)

	text := "单人游戏 Multiplayer 退出"
	w := MeasureText5x7(text, 1)
	if w <= 0 {
		t.Fatalf("expected positive width for CJK text, got %d", w)
	}

	drawnW := fb.DrawText5x7(10, 10, text, 0xFFFFFFFF, 1)
	if drawnW != w {
		t.Fatalf("expected drawn width %d to equal measured width %d", drawnW, w)
	}

	// Verify pixels were written to framebuffer
	hasColor := false
	for _, c := range fb.Color {
		if c != 0 {
			hasColor = true
			break
		}
	}
	if !hasColor {
		t.Fatalf("expected non-zero pixels written to framebuffer for CJK text")
	}

	// Test fallback for glyph not explicitly in map
	fallback := GetCJKGlyph('𢘅')
	allZero := true
	for _, row := range fallback {
		if row != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatalf("expected fallback glyph to have non-zero bits")
	}
}
