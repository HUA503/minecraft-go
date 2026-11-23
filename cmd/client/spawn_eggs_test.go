package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestSpawnEggMobTypeMapsSupportedEntities(t *testing.T) {
	cases := []struct {
		name string
		want sim.MobType
	}{
		{"zombie", sim.MobZombie},
		{"skeleton", sim.MobSkeleton},
		{"creeper", sim.MobCreeper},
		{"cow", sim.MobCow},
		{"pig", sim.MobPig},
		{"sheep", sim.MobSheep},
		{"chicken", sim.MobChicken},
		{"enderman", sim.MobEnderman},
		{"spider", sim.MobSpider},
	}
	for _, tc := range cases {
		got, ok := spawnEggMobType(tc.name)
		if !ok || got != tc.want {
			t.Errorf("spawnEggMobType(%q) = (%v, %v), want (%v, true)", tc.name, got, ok, tc.want)
		}
	}
	if _, ok := spawnEggMobType("warden"); ok {
		t.Fatal("unsupported warden egg resolved to a simulated mob")
	}
}

func TestSpawnEggUseCreatesMobAndConsumesOnlyInSurvival(t *testing.T) {
	for _, tc := range []struct {
		name      string
		creative  bool
		wantCount int
	}{
		{"creative", true, 2},
		{"survival", false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := sim.NewEngine()
			player := engine.AddPlayer("test", [16]byte{}, sim.Vec3{X: 0.5, Y: 64, Z: 0.5})
			engine.SetBlock(0, 63, 0, 1)
			player.Inventory.Slots[0] = inventory.ItemStack{ItemID: 11003, Count: 2} // Cow egg
			client := &Client{
				engine:      engine,
				player:      player,
				isLocal:     true,
				hasTarget:   true,
				targetedPos: world.BlockPos{X: 0, Y: 63, Z: 0},
				isCreative:  tc.creative,
			}

			client.UpdateRightClick(true, 0.05)

			mobs := engine.GetMobs()
			if len(mobs) != 1 || mobs[0].Type != sim.MobCow {
				t.Fatalf("spawned mobs = %+v, want one cow", mobs)
			}
			if got := player.Inventory.Slots[0].Count; got != tc.wantCount {
				t.Fatalf("egg count = %d, want %d", got, tc.wantCount)
			}
		})
	}
}

func TestSpawnEggUseRejectsBlockedSpawnSpace(t *testing.T) {
	engine := sim.NewEngine()
	player := engine.AddPlayer("test", [16]byte{}, sim.Vec3{X: 0.5, Y: 64, Z: 0.5})
	engine.SetBlock(0, 63, 0, 1)
	engine.SetBlock(0, 64, 0, 1)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: 11003, Count: 2}
	client := &Client{
		engine:      engine,
		player:      player,
		isLocal:     true,
		hasTarget:   true,
		targetedPos: world.BlockPos{X: 0, Y: 63, Z: 0},
		isCreative:  false,
	}

	client.UpdateRightClick(true, 0.05)
	if got := len(engine.GetMobs()); got != 0 {
		t.Fatalf("spawned %d mobs into a blocked space", got)
	}
	if got := player.Inventory.Slots[0].Count; got != 2 {
		t.Fatalf("blocked spawn consumed an egg: count=%d", got)
	}
}
