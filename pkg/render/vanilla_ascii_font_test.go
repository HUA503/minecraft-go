package render

import "testing"

func TestVanillaTextShadowDerivesDarkenedRGBAndKeepsAlpha(t *testing.T) {
	const foreground uint32 = 0x7FAABBCC
	const want uint32 = 0x7F2A2E33
	if got := vanillaTextShadowColor(foreground); got != want {
		t.Fatalf("vanilla shadow color = %#08x, want %#08x", got, want)
	}
}

func TestVanillaASCIIAtlasIsEmbedded(t *testing.T) {
	data, err := EmbeddedTextures.ReadFile("assets/textures/font/ascii.png")
	if err != nil {
		t.Fatalf("read embedded vanilla ASCII atlas: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("embedded vanilla ASCII atlas is empty")
	}
}

func TestVanillaASCIIAtlasUsesProportionalOfficialGlyphs(t *testing.T) {
	atlas, ok := vanillaASCIIAtlas()
	if !ok {
		t.Fatal("embedded vanilla ASCII atlas not available")
	}
	if got := vanillaASCIICharWidth(atlas, 'i'); got >= vanillaASCIICharWidth(atlas, 'W') {
		t.Fatalf("expected official narrow glyph i to be narrower than W, got i=%d W=%d", got, vanillaASCIICharWidth(atlas, 'W'))
	}
	if got := vanillaASCIICharWidth(atlas, ' '); got != vanillaASCIISpaceWidth {
		t.Fatalf("space width = %d, want %d", got, vanillaASCIISpaceWidth)
	}
}

func TestDrawCharUsesVanillaLowercaseGlyph(t *testing.T) {
	fb := NewFramebuffer(16, 16)
	if width := fb.DrawChar5x7(0, 0, 'a', 0xFFFFFFFF, 1); width != vanillaASCIICharWidth(mustVanillaASCIIAtlas(t), 'a') {
		t.Fatalf("lowercase a width = %d, want official glyph width", width)
	}
	lit := 0
	for _, px := range fb.Color {
		if px != 0 {
			lit++
		}
	}
	if lit == 0 {
		t.Fatal("lowercase glyph did not render")
	}
}

func mustVanillaASCIIAtlas(t *testing.T) *RawImage {
	t.Helper()
	atlas, ok := vanillaASCIIAtlas()
	if !ok {
		t.Fatal("embedded vanilla ASCII atlas not available")
	}
	return atlas
}
