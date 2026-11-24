package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestAxeStripsWoodAndConsumesWearOnlyInSurvival(t *testing.T) {
	for _, tc := range []struct {
		name         string
		creative     bool
		wantToolWear int
	}{
		{name: "survival", wantToolWear: 1},
		{name: "creative", creative: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := sim.NewEngine()
			player := engine.AddPlayer("test", [16]byte{}, sim.Vec3{X: 0.5, Y: 64, Z: 0.5})
			engine.SetBlock(0, 64, 0, inventory.ItemOakLog)
			player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenAxe, Count: 1}
			client := &Client{
				engine: engine, player: player, isLocal: true, isCreative: tc.creative,
				hasTarget: true, targetedPos: world.BlockPos{X: 0, Y: 64, Z: 0},
			}

			client.UpdateRightClick(true, 0.05)

			wantBlock, ok := inventory.StrippedWoodBlockVariant(inventory.ItemOakLog)
			if !ok || engine.GetBlock(0, 64, 0) != wantBlock {
				t.Fatalf("oak log after axe use = %d, want stripped block %d", engine.GetBlock(0, 64, 0), wantBlock)
			}
			axe := player.Inventory.Slots[0]
			if axe.Count != 1 || axe.Damage != tc.wantToolWear {
				t.Fatalf("axe after stripping = %+v, want count 1 and damage %d", axe, tc.wantToolWear)
			}
		})
	}
}

func TestAxeDoesNotStripUnsupportedBlocks(t *testing.T) {
	engine := sim.NewEngine()
	player := engine.AddPlayer("test", [16]byte{}, sim.Vec3{X: 0.5, Y: 64, Z: 0.5})
	engine.SetBlock(0, 64, 0, inventory.ItemStone)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenAxe, Count: 1}
	client := &Client{
		engine: engine, player: player, isLocal: true,
		hasTarget: true, targetedPos: world.BlockPos{X: 0, Y: 64, Z: 0},
	}

	client.UpdateRightClick(true, 0.05)

	if got := engine.GetBlock(0, 64, 0); got != inventory.ItemStone {
		t.Fatalf("stone changed to block %d while using an axe", got)
	}
	if axe := player.Inventory.Slots[0]; axe.Damage != 0 {
		t.Fatalf("axe lost durability on unsupported block: %+v", axe)
	}
}
