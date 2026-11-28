package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestSpyglassScopeUsesOfficialMaskAcrossViewport(t *testing.T) {
	texture, ok := GetOfficialImage("misc/spyglass_scope.png")
	if !ok || texture.Width == 0 || texture.Height == 0 {
		t.Fatal("official spyglass scope texture was not embedded")
	}
	fb := NewFramebuffer(64, 64)
	for i := range fb.Color {
		fb.Color[i] = 0xFF112233
	}
	fb.DrawSpyglassScope()
	if got, want := fb.Color[0], texture.Pixels[0]; got != want {
		t.Fatalf("outer scope pixel = %#08x, want official opaque mask pixel %#08x", got, want)
	}
	center := fb.Color[32*fb.Width+32]
	if center == 0xFF112233 {
		t.Fatal("scope center was not blended over the world image")
	}
}

func TestSpyglassCreativeIconUsesOfficialItemTexture(t *testing.T) {
	sprite, ok := GetOfficialItemSprite(inventory.ItemSpyglass)
	if !ok {
		t.Fatal("official spyglass item texture was not resolved")
	}
	visible := false
	for _, row := range sprite {
		for _, color := range row {
			if color>>24 != 0 {
				visible = true
				break
			}
		}
	}
	if !visible {
		t.Fatal("spyglass creative icon is fully transparent")
	}
}
