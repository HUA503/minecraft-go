package inventory

import "testing"

func TestShelfMushroomCatalogStatesAndShapes(t *testing.T) {
	definition, ok := ExtendedBlockByName("minecraft:shelf_mushroom")
	if !ok || definition.RuntimeID != ItemShelfMushroom || definition.Category != CreativeNatural || definition.DropRuntimeID != ItemShelfMushroom {
		t.Fatalf("shelf mushroom definition = %+v, found=%v", definition, ok)
	}
	found := false
	for _, item := range CreativeCatalog(CreativeNatural) {
		if item.ID == ItemShelfMushroom {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("shelf mushroom missing from Natural creative tab")
	}
	for facing := 0; facing < 4; facing++ {
		for age := 0; age < 2; age++ {
			state := ShelfMushroomBlockState(facing, age)
			item, gotFacing, gotAge, ok := ShelfMushroomStateProperties(state)
			if !ok || item != ItemShelfMushroom || gotFacing != facing || gotAge != age {
				t.Fatalf("state %d decoded as (%d,%d,%d,%v)", state, item, gotFacing, gotAge, ok)
			}
			boxes, ok := BlockCollisionBoxes(state)
			if !ok || len(boxes) != 2 {
				t.Fatalf("facing=%d age=%d boxes=%+v found=%v", facing, age, boxes, ok)
			}
			wantTexture := ShelfMushroomTextureBase + int32(age)
			if got := BlockTextureForState(state); got != wantTexture {
				t.Fatalf("state texture=%d want %d", got, wantTexture)
			}
		}
	}
	if min, max := BlockDropCountRange(ShelfMushroomBlockState(0, 1)); min != 2 || max != 2 {
		t.Fatalf("large shelf mushroom drops [%d,%d], want [2,2]", min, max)
	}
	crafted, matched := NewCraftingManager().MatchCrafting([]int32{
		ItemBakedPotato, ItemCookedRabbit, ItemModernBowl,
		ItemCarrot, ItemShelfMushroom, ItemAir,
		ItemAir, ItemAir, ItemAir,
	}, 3)
	if !matched || crafted != (ItemStack{ItemID: ItemRabbitStew, Count: 1}) {
		t.Fatalf("Java 26.3 shelf mushroom rabbit stew recipe = %+v, matched=%v", crafted, matched)
	}
	if state, ok := ShelfMushroomPlacementState(ItemShelfMushroom, 1, 0, 0); !ok {
		t.Fatal("east-facing placement state was rejected")
	} else if _, facing, age, valid := ShelfMushroomStateProperties(state); !valid || facing != 1 || age != 0 {
		t.Fatalf("east placement state decoded as (%d,%d,%v)", facing, age, valid)
	}
}
