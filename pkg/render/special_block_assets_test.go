package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestJava263SpecialCubeItemsHaveVanillaSprites(t *testing.T) {
	for _, test := range []struct {
		id      int32
		texture string
	}{
		{14200, "crying_obsidian"},
		{14201, "crimson_nylium_side"},
		{14202, "warped_nylium_side"},
		{14203, "warped_wart_block"},
		{14204, "nether_gold_ore"},
		{14205, "podzol_side"},
		{14206, "mycelium_side"},
		{14207, "magma"},
		{14208, "dried_kelp_side"},
		{14209, "tube_coral_block"},
		{14210, "brain_coral_block"},
		{14211, "bubble_coral_block"},
		{14212, "fire_coral_block"},
		{14213, "horn_coral_block"},
		{14214, "dead_tube_coral_block"},
		{14215, "dead_brain_coral_block"},
		{14216, "dead_bubble_coral_block"},
		{14217, "dead_fire_coral_block"},
		{14218, "dead_horn_coral_block"},
		{14219, "ochre_froglight_side"},
		{14220, "verdant_froglight_side"},
		{14221, "pearlescent_froglight_side"},
	} {
		if got := GetOfficialItemTextureName(test.id); got != test.texture {
			t.Errorf("item %d texture = %q, want %q", test.id, got, test.texture)
		}
		if _, ok := GetOfficialItemSprite(test.id); !ok {
			t.Errorf("item %d has no official sprite", test.id)
		}
		if _, ok := inventory.ExtendedBlockByID(test.id); !ok {
			t.Errorf("item %d has no block placement definition", test.id)
		}
	}
}

func TestJava263IronChainHasOfficialSpriteAndAxisMeshes(t *testing.T) {
	if got := GetOfficialItemTextureName(inventory.ItemIronChain); got != "iron_chain" {
		t.Fatalf("iron chain texture = %q, want iron_chain", got)
	}
	if sprite, ok := GetOfficialItemSprite(inventory.ItemIronChain); !ok || sprite == ([16][16]uint32{}) {
		t.Fatal("iron chain has no visible official item sprite")
	}
	if Textures[inventory.ItemIronChain] == nil {
		t.Fatal("iron chain block texture is not registered")
	}
	for axis := 0; axis < 3; axis++ {
		state, ok := inventory.IronChainBlockState(axis)
		if !ok {
			t.Fatalf("axis %d has no block state", axis)
		}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 {
			if x == 0 && y == 0 && z == 0 {
				return state
			}
			return 0
		}, 0, 0, 0)
		if len(mesh.Vertices) != 24 {
			t.Errorf("axis %d mesh has %d vertices, want one six-face narrow model", axis, len(mesh.Vertices))
		}
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID != inventory.ItemIronChain {
				t.Errorf("axis %d mesh texture block id=%d, want %d", axis, vertex.BlockID, inventory.ItemIronChain)
				break
			}
		}
	}
}

func TestJava263WeightedPressurePlatesHaveVanillaTextures(t *testing.T) {
	for _, tc := range []struct {
		id      int32
		texture string
	}{
		{inventory.ItemLightWeightedPressurePlate, "gold_block"},
		{inventory.ItemHeavyWeightedPressurePlate, "iron_block"},
	} {
		if got := GetOfficialItemTextureName(tc.id); got != tc.texture {
			t.Errorf("weighted plate %d item texture=%q, want %q", tc.id, got, tc.texture)
		}
		if _, ok := GetOfficialItemSprite(tc.id); !ok {
			t.Errorf("weighted plate %d has no official item sprite", tc.id)
		}
		if Textures[tc.id] == nil {
			t.Errorf("weighted plate %d has no official block texture", tc.id)
		}
	}
}
