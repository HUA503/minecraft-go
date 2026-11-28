package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestConsumeCraftingInputReturnsContainerItems(t *testing.T) {
	cases := []struct {
		name      string
		stack     inventory.ItemStack
		remainder int32
		wantStack inventory.ItemStack
		wantExtra int32
	}{
		{"bowl is returned to its slot", inventory.ItemStack{ItemID: inventory.ItemModernBowl, Count: 1}, inventory.ItemModernBowl, inventory.ItemStack{ItemID: inventory.ItemModernBowl, Count: 1}, 0},
		{"stacked bowl remains unchanged", inventory.ItemStack{ItemID: inventory.ItemModernBowl, Count: 3}, inventory.ItemModernBowl, inventory.ItemStack{ItemID: inventory.ItemModernBowl, Count: 3}, 0},
		{"ordinary ingredient is consumed", inventory.ItemStack{ItemID: inventory.ItemBrownMushroom, Count: 1}, 0, inventory.ItemStack{}, 0},
		{"different remainder replaces the consumed ingredient", inventory.ItemStack{ItemID: inventory.ItemBrownMushroom, Count: 1}, inventory.ItemModernBowl, inventory.ItemStack{ItemID: inventory.ItemModernBowl, Count: 1}, 0},
		{"different remainder from stacked input goes to inventory", inventory.ItemStack{ItemID: inventory.ItemBrownMushroom, Count: 2}, inventory.ItemModernBowl, inventory.ItemStack{ItemID: inventory.ItemBrownMushroom, Count: 1}, inventory.ItemModernBowl},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack := tc.stack
			if extra := consumeCraftingInput(&stack, tc.remainder); extra != tc.wantExtra || stack != tc.wantStack {
				t.Fatalf("consume result stack=%+v extra=%d, want stack=%+v extra=%d", stack, extra, tc.wantStack, tc.wantExtra)
			}
		})
	}
}
