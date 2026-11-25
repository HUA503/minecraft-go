package render

// The legacy vanilla ASCII sheet is a 16x16 grid of 8x8 glyph cells. Keeping
// its alpha mask (rather than a hand-authored 5x7 approximation) preserves the
// original Minecraft glyph shapes, including lowercase letters and narrow
// punctuation.
const (
	vanillaASCIIGlyphCell  = 8
	vanillaASCIISpaceWidth = 4
)

func vanillaASCIIAtlas() (*RawImage, bool) {
	return GetOfficialImage("font/ascii")
}

func vanillaASCIICharWidth(atlas *RawImage, ch byte) int {
	if ch == ' ' {
		return vanillaASCIISpaceWidth
	}
	if atlas == nil || atlas.Width < 16*vanillaASCIIGlyphCell || atlas.Height < 16*vanillaASCIIGlyphCell {
		return 0
	}
	x0 := int(ch%16) * vanillaASCIIGlyphCell
	y0 := int(ch/16) * vanillaASCIIGlyphCell
	right := -1
	for y := 0; y < vanillaASCIIGlyphCell; y++ {
		for x := 0; x < vanillaASCIIGlyphCell; x++ {
			if atlas.Pixels[(y0+y)*atlas.Width+x0+x]>>24 != 0 && x > right {
				right = x
			}
		}
	}
	if right < 0 {
		return 0
	}
	// Vanilla's bitmap font leaves one pixel after the rightmost lit pixel.
	return right + 2
}

func (fb *Framebuffer) drawVanillaASCIIChar(x, y int, ch byte, color uint32, scale int) (int, bool) {
	if scale < 1 {
		scale = 1
	}
	atlas, ok := vanillaASCIIAtlas()
	if !ok || atlas.Width < 16*vanillaASCIIGlyphCell || atlas.Height < 16*vanillaASCIIGlyphCell {
		return 0, false
	}
	width := vanillaASCIICharWidth(atlas, ch)
	if width == 0 {
		return 0, false
	}
	x0 := int(ch%16) * vanillaASCIIGlyphCell
	y0 := int(ch/16) * vanillaASCIIGlyphCell
	colorAlpha := (color >> 24) & 0xFF
	for gy := 0; gy < vanillaASCIIGlyphCell; gy++ {
		for gx := 0; gx < width-1 && gx < vanillaASCIIGlyphCell; gx++ {
			alpha := (atlas.Pixels[(y0+gy)*atlas.Width+x0+gx] >> 24) & 0xFF
			if alpha == 0 {
				continue
			}
			a := alpha * colorAlpha / 0xFF
			pixel := (a << 24) | (color & 0x00FFFFFF)
			if scale == 1 {
				px, py := x+gx, y+gy
				if px >= 0 && px < fb.Width && py >= 0 && py < fb.Height {
					fb.Color[py*fb.Width+px] = pixel
				}
			} else {
				fb.DrawRect(x+gx*scale, y+gy*scale, scale, scale, pixel)
			}
		}
	}
	return width * scale, true
}
