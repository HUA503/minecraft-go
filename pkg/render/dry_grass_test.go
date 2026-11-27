package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestDryGrassOfficialAssetsAndCrossedMeshes(t *testing.T) {
	for _, id := range []int32{inventory.ItemDryShortGrass, inventory.ItemDryTallGrass} {
		if _, ok := GetOfficialItemSprite(id); !ok {
			t.Errorf("official dry grass icon missing for %d", id)
		}
		if Textures[id] == nil {
			t.Errorf("dry grass block texture missing for %d", id)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return id
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 8 || len(mesh.Indices) != 24 {
			t.Errorf("dry grass %d mesh is %d vertices/%d indices, want crossed planes 8/24", id, len(mesh.Vertices), len(mesh.Indices))
		}
	}
}
