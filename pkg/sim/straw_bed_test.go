package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestStrawBedPlacementConsumesOneAndBreakingEitherHalfDropsOne(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("bed-test", [16]byte{}, Vec3{X: 0, Y: 70, Z: 0})
	foot := world.BlockPos{X: 10, Y: 64, Z: 10}
	facingState, _ := inventory.StrawBedPlacementState(inventory.ItemStrawBed, 90)
	dx, dz, headState, _ := inventory.StrawBedOtherHalf(facingState)
	head := world.BlockPos{X: foot.X + dx, Y: foot.Y, Z: foot.Z + dz}
	engine.SetBlock(foot.X, foot.Y-1, foot.Z, inventory.ItemStone)
	engine.SetBlock(head.X, head.Y-1, head.Z, inventory.ItemStone)
	player.Inventory.Slots[player.Inventory.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemStrawBed, Count: 1}
	if !engine.PlaceBlock(player.ID, foot, facingState) {
		t.Fatal("could not place the two-part straw bed")
	}
	if engine.GetBlock(foot.X, foot.Y, foot.Z) != facingState || engine.GetBlock(head.X, head.Y, head.Z) != headState {
		t.Fatalf("placed straw bed states = %d/%d, want %d/%d", engine.GetBlock(foot.X, foot.Y, foot.Z), engine.GetBlock(head.X, head.Y, head.Z), facingState, headState)
	}
	if held := player.Inventory.GetSelectedItem(); !held.IsEmpty() {
		t.Fatalf("placing straw bed left held stack %+v, want empty", held)
	}
	if drop, ok := engine.BreakBlockWithTool(player.ID, head, inventory.ItemAir); !ok || drop != inventory.ItemStrawBed {
		t.Fatalf("breaking head half returned item %d, success %t; want straw bed", drop, ok)
	}
	if engine.GetBlock(foot.X, foot.Y, foot.Z) != 0 || engine.GetBlock(head.X, head.Y, head.Z) != 0 {
		t.Fatal("breaking one straw bed half did not remove its pair")
	}
	drops := engine.GetItemDrops()
	if len(drops) != 1 || drops[0].ItemID != inventory.ItemStrawBed || drops[0].Count != 1 {
		t.Fatalf("breaking straw bed created drops %+v, want one straw bed", drops)
	}
}

func TestStrawBedOnlySleepsAtNightAndIsOneUse(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("bed-test", [16]byte{}, Vec3{X: 0, Y: 70, Z: 0})
	foot := world.BlockPos{X: 10, Y: 64, Z: 10}
	footState := inventory.StrawBedBlockState(inventory.ItemStrawBed, 0, false)
	dx, dz, headState, _ := inventory.StrawBedOtherHalf(footState)
	head := world.BlockPos{X: foot.X + dx, Y: foot.Y, Z: foot.Z + dz}
	engine.SetBlock(foot.X, foot.Y, foot.Z, footState)
	engine.SetBlock(head.X, head.Y, head.Z, headState)
	engine.SetWorldTime(6000)
	if slept, destroyed := engine.UseStrawBed(player.ID, foot); slept || destroyed {
		t.Fatalf("daytime use = slept %t destroyed %t, want neither", slept, destroyed)
	}
	if engine.GetBlock(foot.X, foot.Y, foot.Z) != footState || engine.GetBlock(head.X, head.Y, head.Z) != headState {
		t.Fatal("daytime attempt destroyed the straw bed")
	}
	engine.SetWorldTime(18000)
	if slept, destroyed := engine.UseStrawBed(player.ID, head); !slept || !destroyed {
		t.Fatalf("nighttime use = slept %t destroyed %t, want both", slept, destroyed)
	}
	if engine.GetWorldTime() != 0 || engine.GetBlock(foot.X, foot.Y, foot.Z) != 0 || engine.GetBlock(head.X, head.Y, head.Z) != 0 {
		t.Fatal("successful straw bed sleep did not advance to dawn and destroy both halves")
	}
	if slept, destroyed := engine.UseStrawBed(player.ID, foot); slept || destroyed {
		t.Fatalf("second use = slept %t destroyed %t, want no bed", slept, destroyed)
	}
}

func TestStrawBedAttemptInNetherDestroysItWithoutSleeping(t *testing.T) {
	engine := NewEngine()
	engine.SwitchDimension(1)
	player := engine.AddPlayer("bed-test", [16]byte{}, Vec3{X: 0, Y: 70, Z: 0})
	foot := world.BlockPos{X: 10, Y: 64, Z: 10}
	footState := inventory.StrawBedBlockState(inventory.ItemStrawBed, 1, false)
	dx, dz, headState, _ := inventory.StrawBedOtherHalf(footState)
	head := world.BlockPos{X: foot.X + dx, Y: foot.Y, Z: foot.Z + dz}
	engine.SetBlock(foot.X, foot.Y, foot.Z, footState)
	engine.SetBlock(head.X, head.Y, head.Z, headState)
	if slept, destroyed := engine.UseStrawBed(player.ID, foot); slept || !destroyed {
		t.Fatalf("Nether use = slept %t destroyed %t, want destroy without sleeping", slept, destroyed)
	}
	if engine.GetBlock(foot.X, foot.Y, foot.Z) != 0 || engine.GetBlock(head.X, head.Y, head.Z) != 0 {
		t.Fatal("Nether use did not destroy both straw bed halves")
	}
}
