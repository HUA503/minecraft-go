package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestJava263MasonryWallTexturesIconsAndMeshes(t *testing.T) {
	for _, wall := range inventory.ModernMasonryWallItemDefinitions() {
		if _, ok := GetOfficialTexture("block", wall.TextureSide); !ok {
			t.Errorf("missing wall texture %q for %s", wall.TextureSide, wall.Name)
		}
		icon := GetItemSprite16x16(wall.RuntimeID)
		visible := 0
		for _, row := range icon {
			for _, pixel := range row {
				if pixel>>24 != 0 {
					visible++
				}
			}
		}
		if visible < 8 {
			t.Errorf("wall icon %s has only %d visible pixels", wall.Name, visible)
		}
		blocks := map[[3]int32]int32{{8, 5, 8}: wall.RuntimeID, {8, 5, 7}: inventory.ItemOakPlanks}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }, 0, 0, 0)
		found := false
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID == wall.RuntimeID {
				found = true
				if vertex.X < 8 || vertex.X > 9 || vertex.Y < 5 || vertex.Y > 6.5 || vertex.Z < 7 || vertex.Z > 9 {
					t.Errorf("wall mesh vertex outside expected wall bounds: %+v", vertex)
					break
				}
			}
		}
		if !found {
			t.Errorf("wall %s produced no mesh", wall.Name)
		}
	}
}
