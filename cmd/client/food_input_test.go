package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestCanConsumeSelectedFoodAtFullHunger(t *testing.T) {
	for _, itemID := range []int32{
		inventory.ItemGoldenApple,
		inventory.ItemEnchantedGoldenApple,
		inventory.ItemChorusFruit,
		inventory.ItemMilkBucket,
		inventory.ItemHoneyBottle,
	} {
		if !canConsumeSelectedFood(itemID, 20) {
			t.Errorf("food item %d should be usable at full hunger", itemID)
		}
	}
	if canConsumeSelectedFood(inventory.ItemBread, 20) {
		t.Fatal("ordinary food should not be usable at full hunger")
	}
	if !canConsumeSelectedFood(inventory.ItemBread, 19) {
		t.Fatal("ordinary food should be usable below full hunger")
	}
	if canConsumeSelectedFood(inventory.ItemDiamond, 0) {
		t.Fatal("non-food item should not be consumable")
	}
}

