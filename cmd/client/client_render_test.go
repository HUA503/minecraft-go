package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
)

func TestClientMobAndHUDRender(t *testing.T) {
	width := 320
	height := 240
	fb := render.NewFramebuffer(width, height)
	cam := render.NewCamera(8.0, 66.62, 8.0)
	cam.Yaw = 35.0
	cam.Pitch = 10.0

	client := &Client{
		framebuf: fb,
		camera:   cam,
		isLocal:  true,
		engine:   sim.NewEngine(),
	}

	pigPos := sim.Vec3{X: 11.0, Y: 65.0, Z: 12.0}
	pig := client.engine.SpawnMob(sim.MobPig, pigPos)
	pig.Yaw = 135.0
	pig.Pitch = 0

	fb.Clear(0xFF87CEEB)
	client.drawMob(pig)

	// Verify mob rendered pixels
	var mobPixels int
	for _, c := range fb.Color {
		if c != 0xFF87CEEB {
			mobPixels++
		}
	}
	if mobPixels < 100 {
		t.Errorf("expected textured pig to render at least 100 pixels, got %d", mobPixels)
	}

	// Verify HUD rendering
	var slots [9]inventory.ItemStack
	slots[0] = inventory.ItemStack{ItemID: 270, Count: 1} // Wooden pickaxe
	slots[1] = inventory.ItemStack{ItemID: 1, Count: 64}   // Stone
	client.framebuf.DrawFullHUD(0, 14.0, 16, 0.5, 5, 0, slots)

	var hudPixels int
	for _, c := range fb.Color {
		if c != 0xFF87CEEB {
			hudPixels++
		}
	}
	if hudPixels <= mobPixels+200 {
		t.Errorf("expected HUD to add at least 200 pixels, total=%d mob=%d", hudPixels, mobPixels)
	}
}
