package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func countVisiblePixels(sprite [16][16]uint32) int {
	visible := 0
	for _, row := range sprite {
		for _, pixel := range row {
			if pixel>>24 != 0 {
				visible++
			}
		}
	}
	return visible
}

func TestJava263WoodDoorsAndTrapdoorsUseVanillaTexturesAndModels(t *testing.T) {
	for _, door := range inventory.ModernWoodDoorItemDefinitions() {
		for _, texture := range []string{door.TextureTop, door.TextureBottom} {
			if _, ok := GetOfficialTexture("block", texture); !ok {
				t.Errorf("missing official door texture %q for %s", texture, door.Name)
			}
		}
		icon := GetItemSprite16x16(door.RuntimeID)
		if countVisiblePixels(icon) < 8 {
			t.Errorf("door icon for %s is empty", door.Name)
		}
		for _, upper := range []bool{false, true} {
			state := inventory.WoodDoorBlockState(door.RuntimeID, 0, false, false, upper)
			mesh := BuildSectionMesh(func(x, y, z int32) int32 {
				if x == 4 && y == 4 && z == 4 {
					return state
				}
				return 0
			}, 0, 0, 0)
			if len(mesh.Vertices) == 0 {
				t.Errorf("door %s state %d produced no mesh", door.Name, state)
			}
			for _, vertex := range mesh.Vertices {
				if vertex.BlockID != state {
					t.Errorf("door state %d uses texture ID %d, want upper/lower texture state %d", state, vertex.BlockID, state)
					break
				}
			}
		}
	}
	for _, trapdoor := range inventory.ModernWoodTrapdoorItemDefinitions() {
		if _, ok := GetOfficialTexture("block", trapdoor.TextureSide); !ok {
			t.Errorf("missing official trapdoor texture %q for %s", trapdoor.TextureSide, trapdoor.Name)
		}
		icon := GetItemSprite16x16(trapdoor.RuntimeID)
		if countVisiblePixels(icon) < 8 {
			t.Errorf("trapdoor icon for %s is empty", trapdoor.Name)
		}
		for _, state := range []int32{
			inventory.WoodTrapdoorBlockState(trapdoor.RuntimeID, 0, false, false),
			inventory.WoodTrapdoorBlockState(trapdoor.RuntimeID, 1, true, true),
		} {
			mesh := BuildSectionMesh(func(x, y, z int32) int32 {
				if x == 4 && y == 4 && z == 4 {
					return state
				}
				return 0
			}, 0, 0, 0)
			if len(mesh.Vertices) == 0 {
				t.Errorf("trapdoor %s state %d produced no mesh", trapdoor.Name, state)
			}
			for _, vertex := range mesh.Vertices {
				if vertex.BlockID != trapdoor.RuntimeID {
					t.Errorf("trapdoor state %d uses texture ID %d, want item %d", state, vertex.BlockID, trapdoor.RuntimeID)
					break
				}
			}
		}
	}
	iron, ok := inventory.ExtendedBlockByID(inventory.ItemIronTrapdoor)
	if !ok {
		t.Fatal("iron trapdoor is not registered")
	}
	if _, ok := GetOfficialTexture("block", iron.TextureSide); !ok {
		t.Fatalf("missing official iron trapdoor texture %q", iron.TextureSide)
	}
	if countVisiblePixels(GetItemSprite16x16(inventory.ItemIronTrapdoor)) < 8 {
		t.Fatal("iron trapdoor creative icon is empty")
	}
	state := inventory.IronTrapdoorBlockState(0, true, true)
	mesh := BuildSectionMesh(func(x, y, z int32) int32 {
		if x == 4 && y == 4 && z == 4 {
			return state
		}
		return 0
	}, 0, 0, 0)
	if len(mesh.Vertices) == 0 {
		t.Fatal("iron trapdoor state produced no mesh")
	}
	for _, vertex := range mesh.Vertices {
		if vertex.BlockID != inventory.ItemIronTrapdoor {
			t.Fatalf("iron trapdoor state uses texture ID %d, want item %d", vertex.BlockID, inventory.ItemIronTrapdoor)
		}
	}
}
