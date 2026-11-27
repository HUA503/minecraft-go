package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestColoredCarpetItemsUseBundledWoolTexturesAndRenderAsThinBlocks(t *testing.T) {
	seen := 0
	for _, block := range inventory.ExtendedBlockDefinitions() {
		if !inventory.IsCarpetBlock(block.RuntimeID) {
			continue
		}
		seen++
		if _, ok := GetOfficialTexture("block", block.TextureTop); !ok {
			t.Errorf("%s texture %q is missing from bundled vanilla assets", block.Name, block.TextureTop)
		}
		icon := RenderIsometricBlock16x16(block.RuntimeID)
		pixels := 0
		for _, row := range icon {
			for _, pixel := range row {
				if pixel>>24 != 0 {
					pixels++
				}
			}
		}
		if pixels == 0 {
			t.Errorf("%s produced an empty creative icon", block.Name)
		}
	}
	if seen != 16 {
		t.Fatalf("rendered colored carpet items = %d, want 16", seen)
	}
}
