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
