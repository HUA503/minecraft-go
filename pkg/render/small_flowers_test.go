package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestSmallFlowerCreativeIconsAndCrossedQuads(t *testing.T) {
	ids := make([]int32, 0, inventory.LastSmallFlower-inventory.FirstSmallFlower+11)
	for id := inventory.FirstSmallFlower; id <= inventory.LastSmallFlower; id++ {
		ids = append(ids, id)
	}
	ids = append(ids, inventory.ItemDandelion, inventory.ItemPoppy, inventory.ItemWitherRose, inventory.ItemBrownMushroom, inventory.ItemRedMushroom,
		inventory.ItemCrimsonFungus, inventory.ItemWarpedFungus, inventory.ItemCrimsonRoots, inventory.ItemWarpedRoots, inventory.ItemNetherSprouts)
	for _, id := range ids {
		if _, ok := GetOfficialItemSprite(id); !ok {
			t.Errorf("official small flower icon is missing for block %d", id)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return id
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 8 || len(mesh.Indices) != 24 {
			t.Errorf("flower %d mesh has %d vertices/%d indices, want 8/24 crossed planes", id, len(mesh.Vertices), len(mesh.Indices))
		}
	}
}
