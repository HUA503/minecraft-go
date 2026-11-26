package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestJava263WoodFenceGatesUseOfficialTexturesAndRender(t *testing.T) {
	for _, gate := range inventory.ModernWoodFenceGateItemDefinitions() {
		if _, ok := GetOfficialTexture("block", gate.TextureSide); !ok {
			t.Errorf("missing official planks texture %q for %s", gate.TextureSide, gate.Name)
		}
		icon := GetItemSprite16x16(gate.RuntimeID)
		visible := 0
		for _, row := range icon {
			for _, pixel := range row {
				if pixel>>24 != 0 {
					visible++
				}
			}
		}
		if visible < 8 {
			t.Errorf("fence gate icon %s has only %d visible pixels", gate.Name, visible)
		}
		closed := inventory.FenceGateBlockState(gate.RuntimeID, 0, false, false)
		open := inventory.FenceGateBlockState(gate.RuntimeID, 0, true, false)
		for _, state := range []int32{closed, open} {
			mesh := BuildSectionMesh(func(x, y, z int32) int32 {
				if x == 4 && y == 4 && z == 4 {
					return state
				}
				return inventory.ItemAir
			}, 0, 0, 0)
			if len(mesh.Vertices) < 8 {
				t.Errorf("gate %s state %d produced only %d vertices", gate.Name, state, len(mesh.Vertices))
			}
			for _, vertex := range mesh.Vertices {
				if vertex.BlockID != gate.RuntimeID {
					t.Errorf("gate state %d vertex uses texture block %d, want item %d", state, vertex.BlockID, gate.RuntimeID)
					break
				}
			}
		}
	}
}
