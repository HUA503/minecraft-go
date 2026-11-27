package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestMasonryStairTexturesAndIcons(t *testing.T) {
	for _, stair := range inventory.ModernMasonryStairItemDefinitions() {
		if _, ok := GetOfficialTexture("block", stair.TextureTop); !ok {
			t.Errorf("missing top block texture %q for %s", stair.TextureTop, stair.Name)
		}
		if _, ok := GetOfficialTexture("block", stair.TextureSide); !ok {
			t.Errorf("missing side block texture %q for %s", stair.TextureSide, stair.Name)
		}
		if Textures[stair.RuntimeID] == nil {
			t.Errorf("block face textures were not initialized for %s", stair.Name)
		}
		icon := RenderIsometricBlock16x16(stair.RuntimeID)
		visible := 0
		for _, row := range icon {
			for _, pixel := range row {
				if pixel>>24 != 0 {
					visible++
				}
			}
		}
		if visible < 20 {
			t.Errorf("stair icon %s has only %d visible pixels", stair.Name, visible)
		}
	}
}

func TestJava263WoodFenceTexturesIconsAndMesh(t *testing.T) {
	for _, fence := range inventory.ModernWoodFenceItemDefinitions() {
		if _, ok := GetOfficialTexture("block", fence.TextureSide); !ok {
			t.Errorf("missing fence texture %q for %s", fence.TextureSide, fence.Name)
		}
		icon := GetItemSprite16x16(fence.RuntimeID)
		visible := 0
		for _, row := range icon {
			for _, pixel := range row {
				if pixel>>24 != 0 {
					visible++
				}
			}
		}
		if visible < 8 {
			t.Errorf("fence icon %s has only %d visible pixels", fence.Name, visible)
		}
		blocks := map[[3]int32]int32{{8, 5, 8}: fence.RuntimeID, {8, 5, 7}: inventory.ItemOakPlanks}
		mesh := BuildSectionMesh(func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }, 0, 0, 0)
		found := false
		for _, vertex := range mesh.Vertices {
			if vertex.BlockID == fence.RuntimeID {
				found = true
				if vertex.X < 8 || vertex.X > 9 || vertex.Y < 5 || vertex.Y > 6.5 || vertex.Z < 7 || vertex.Z > 9 {
					t.Errorf("fence mesh vertex outside fence bounds: %+v", vertex)
					break
				}
			}
		}
		if !found {
			t.Errorf("fence %s produced no mesh", fence.Name)
		}
	}
}

func TestJava263NetherBrickFenceTextureIconAndMesh(t *testing.T) {
	definition, ok := inventory.ExtendedBlockByID(inventory.ItemNetherBrickFence)
	if !ok {
		t.Fatal("nether brick fence is not registered as a block")
	}
	if _, ok := GetOfficialTexture("block", definition.TextureSide); !ok {
		t.Fatalf("missing official fence texture %q", definition.TextureSide)
	}
	icon := GetItemSprite16x16(inventory.ItemNetherBrickFence)
	visible := 0
	for _, row := range icon {
		for _, pixel := range row {
			if pixel>>24 != 0 {
				visible++
			}
		}
	}
	if visible < 8 {
		t.Fatalf("nether brick fence icon has only %d visible pixels", visible)
	}
	blocks := map[[3]int32]int32{
		{8, 5, 8}: inventory.ItemNetherBrickFence,
		{9, 5, 8}: inventory.ItemOakPlanks,
	}
	mesh := BuildSectionMesh(func(x, y, z int32) int32 { return blocks[[3]int32{x, y, z}] }, 0, 0, 0)
	found := false
	for _, vertex := range mesh.Vertices {
		if vertex.BlockID == inventory.ItemNetherBrickFence {
			found = true
			if vertex.X < 8 || vertex.X > 9 || vertex.Y < 5 || vertex.Y > 6.5 || vertex.Z < 8 || vertex.Z > 9 {
				t.Errorf("nether brick fence mesh vertex outside fence bounds: %+v", vertex)
				break
			}
		}
	}
	if !found {
		t.Fatal("nether brick fence produced no mesh")
	}
}
