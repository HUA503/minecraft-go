package inventory

import "testing"

func TestTallFlowerRegistryAndPairedStates(t *testing.T) {
	names := []string{"sunflower", "lilac", "rose_bush", "peony"}
	for index, name := range names {
		id := FirstTallFlower + int32(index)
		definition, ok := ExtendedBlockByName("minecraft:" + name)
		if !ok || definition.RuntimeID != id || definition.Category != CreativeNatural || definition.DropRuntimeID != id {
			t.Errorf("tall flower %s definition = %+v, found=%v", name, definition, ok)
		}
		upper, ok := TallFlowerBlockState(id, true)
		if !ok {
			t.Fatalf("could not encode upper half for %s", name)
		}
		if got, isUpper, valid := TallFlowerStateProperties(upper); !valid || got != id || !isUpper {
			t.Errorf("upper state %d decoded as item=%d upper=%v valid=%v", upper, got, isUpper, valid)
		}
		if got := BlockItemForState(upper); got != id {
			t.Errorf("upper state %d maps to item %d, want %d", upper, got, id)
		}
		if got := BlockTextureForState(upper); got != upper {
			t.Errorf("upper state %d maps to texture state %d", upper, got)
		}
		if !IsPlaceableBlock(id) || !IsExtendedFoliage(id) || !IsReplaceableBlock(id) {
			t.Errorf("tall flower %s missing placeable/foliage/replaceable behavior", name)
		}
		if got := GetBlockHardness(upper); got != 0 {
			t.Errorf("upper half hardness = %v, want instant break", got)
		}
		found := false
		for _, item := range CreativeCatalog(CreativeNatural) {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tall flower %s missing from Natural creative tab", name)
		}
	}
}
