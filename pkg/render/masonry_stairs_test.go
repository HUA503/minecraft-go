package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestMasonryStairTexturesAndIcons(t *testing.T) {
	for _, stair := range inventory.ModernMasonryStairItemDefinitions() {
		if _, ok := GetOfficialTexture("block", stair.TextureTop); !ok {
			t.Errorf("missing top block texture %q for %s", stair.TextureTop, stair.Name)
		}
		if _, ok := GetOfficialTexture("block", stair.TextureSide); !ok {
			t.Errorf("missing side block texture %q for %s", stair.TextureSide, stair.Name)
		}
		if Textures[stair.RuntimeID] == nil {
			t.Errorf("block face textures were not initialized for %s", stair.Name)
		}
		icon := RenderIsometricBlock16x16(stair.RuntimeID)
		visible := 0
		for _, row := range icon {
			for _, pixel := range row {
				if pixel>>24 != 0 {
					visible++
				}
			}
		}
		if visible < 20 {
			t.Errorf("stair icon %s has only %d visible pixels", stair.Name, visible)
		}
	}
}
