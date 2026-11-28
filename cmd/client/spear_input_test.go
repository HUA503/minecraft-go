package main

import (
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
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

func TestSpyglassHeldUseConsumesRightClickAndStopsOnRelease(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemSpyglass, Count: 1}
	client := &Client{player: &sim.PlayerEntity{Inventory: inv}, isLocal: true}
	if !client.updateSpyglass(true) || !client.isUsingSpyglass {
		t.Fatal("holding right-click with a spyglass did not start scope use")
	}
	if !client.updateSpyglass(false) || client.isUsingSpyglass {
		t.Fatal("releasing right-click did not stop scope use")
	}
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 1}
	client.isUsingSpyglass = true
	if client.updateSpyglass(true) || client.isUsingSpyglass {
		t.Fatal("changing away from the spyglass left scope use active")
	}
}

func TestSpyglassFOVOverridesSprintAndRestoresNormalFOV(t *testing.T) {
	if got := targetPlayerFOV(true, true, true, true); got != 7 {
		t.Fatalf("spyglass sprint FOV = %.1f, want 7", got)
	}
	if got := targetPlayerFOV(false, true, true, true); got != 82 {
		t.Fatalf("sprint FOV = %.1f, want 82", got)
	}
	if got := targetPlayerFOV(false, false, false, true); got != 70 {
		t.Fatalf("normal FOV = %.1f, want 70", got)
	}
}

func TestTridentChargeReleaseLaunchesAndWearsRecoverableItem(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemTrident, Count: 1}
	engine := sim.NewEngine()
	client := &Client{
		player:            &sim.PlayerEntity{ID: 1, Inventory: inv},
		engine:            engine,
		camera:            render.NewCamera(0, 65, 0),
		isLocal:           true,
		tridentUseItem:    inventory.ItemTrident,
		tridentUseStarted: time.Now().Add(-time.Second),
	}
	if !client.updateTridentUse(false) {
		t.Fatal("trident release was not consumed")
	}
	if got := inv.Slots[inv.SelectedHotbar]; got.Damage != 1 || got.ItemID != inventory.ItemTrident {
		t.Fatalf("thrown trident inventory stack = %+v, want one durability used", got)
	}
	projectiles := engine.GetTridentProjectiles()
	if len(projectiles) != 1 || projectiles[0].Stack.Damage != 1 || projectiles[0].OwnerID != 1 {
		t.Fatalf("launched trident projectile = %+v", projectiles)
	}
}

func TestTridentReleaseBeforeChargeThresholdDoesNotThrow(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemTrident, Count: 1}
	engine := sim.NewEngine()
	client := &Client{
		player:            &sim.PlayerEntity{Inventory: inv},
		engine:            engine,
		camera:            render.NewCamera(0, 65, 0),
		isLocal:           true,
		tridentUseItem:    inventory.ItemTrident,
		tridentUseStarted: time.Now().Add(-100 * time.Millisecond),
	}
	if !client.updateTridentUse(false) || len(engine.GetTridentProjectiles()) != 0 {
		t.Fatal("early trident release launched a projectile")
	}
	if got := inv.Slots[inv.SelectedHotbar]; got.Damage != 0 {
		t.Fatalf("early release damaged trident: %+v", got)
	}
}
