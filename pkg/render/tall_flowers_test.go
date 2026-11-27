package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestTallFlowerIconsMeshesAndUpperTextures(t *testing.T) {
	for id := inventory.FirstTallFlower; id <= inventory.LastTallFlower; id++ {
		if _, ok := GetOfficialItemSprite(id); !ok {
			t.Errorf("official tall flower icon is missing for block %d", id)
		}
		upper, _ := inventory.TallFlowerBlockState(id, true)
		if Textures[upper] == nil {
			t.Errorf("upper-half texture is missing for tall flower %d (state %d)", id, upper)
		}
		for _, blockID := range []int32{id, upper} {
			mesh := BuildSectionMesh(func(x, y, z int32) int32 {
				if x == 0 && y == 0 && z == 0 {
					return blockID
				}
				return 0
			}, 0, 0, 0)
			if len(mesh.Vertices) != 8 || len(mesh.Indices) != 24 {
				t.Errorf("tall flower state %d mesh has %d vertices/%d indices, want crossed planes (8/24)", blockID, len(mesh.Vertices), len(mesh.Indices))
			}
			if len(mesh.Vertices) > 0 && mesh.Vertices[0].BlockID != blockID {
				t.Errorf("tall flower state %d rendered with texture state %d", blockID, mesh.Vertices[0].BlockID)
			}
		}
	}
}
