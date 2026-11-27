package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestMossCarpetTexturesAndPartialMesh(t *testing.T) {
	for _, id := range []int32{inventory.ItemMossCarpet, inventory.ItemPaleMossCarpet} {
		if _, ok := GetOfficialItemSprite(id); !ok {
			t.Errorf("official moss carpet item texture missing for %d", id)
		}
		if Textures[id] == nil {
			t.Errorf("block texture missing for moss carpet %d", id)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return id
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 24 || len(mesh.Indices) != 36 {
			t.Errorf("moss carpet %d mesh has %d vertices/%d indices, want 24/36 partial-block mesh", id, len(mesh.Vertices), len(mesh.Indices))
		}
	}
}
