package main

import (
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestSpearPrimaryActionDoesNotMineBlocks(t *testing.T) {
	engine := sim.NewEngine()
	engine.SetBlock(2, 64, 2, inventory.ItemStone)
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemIronSpear, Count: 1}
	client := &Client{
		player:         &sim.PlayerEntity{Inventory: inv},
		engine:         engine,
		hasTarget:      true,
		targetedPos:    world.BlockPos{X: 2, Y: 64, Z: 2},
		hasMiningBlock: true,
		miningPos:      world.BlockPos{X: 2, Y: 64, Z: 2},
		breakProgress:  0.75,
		isMining:       true,
	}

	client.UpdateMining(true, 1)
	if got := engine.GetBlockIfLoaded(2, 64, 2); got != inventory.ItemStone {
		t.Fatalf("left-click with a spear broke the target block: %d", got)
	}
	if client.hasMiningBlock || client.isMining || client.breakProgress != 0 {
		t.Fatalf("spear left a block-breaking state active: %+v", client)
	}
}

func TestSpearChargeReleaseWithoutCameraClearsState(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemIronSpear, Count: 1}
	client := &Client{
		player:             &sim.PlayerEntity{Inventory: inv},
		engine:             sim.NewEngine(),
		isLocal:            true,
		spearChargeItem:    inventory.ItemIronSpear,
		spearChargeStarted: time.Now().Add(-time.Second),
	}
	if !client.updateSpearCharge(false) {
		t.Fatal("spear release was not consumed")
	}
	if client.spearChargeItem != 0 || !client.spearChargeStarted.IsZero() {
		t.Fatalf("release left spear charge active: item=%d started=%v", client.spearChargeItem, client.spearChargeStarted)
	}
}
