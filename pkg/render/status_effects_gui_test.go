package render

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestInventoryStatusEffectAssetsAndPanel(t *testing.T) {
	for _, path := range []string{
		"gui/sprites/container/inventory/effect_background.png",
		"gui/sprites/container/inventory/effect_background_ambient.png",
		"mob_effect/wither.png",
		"mob_effect/poison.png",
	} {
		if _, ok := GetOfficialImage(path); !ok {
			t.Fatalf("missing vanilla status-effect asset %q", path)
		}
	}
	fb := NewFramebuffer(300, 120)
	for i := range fb.Color {
		fb.Color[i] = 0xFF202020
	}
	fb.DrawInventoryStatusEffects([]world.PlayerStatusEffect{{ID: "minecraft:wither", DurationTicks: 20 * 95, Amplifier: 1}}, 10, 10, 140, 1)
	changed := 0
	for _, pixel := range fb.Color {
		if pixel != 0xFF202020 {
			changed++
		}
	}
	if changed < 250 {
		t.Fatalf("status-effect panel did not draw its background, icon and labels: changed pixels=%d", changed)
	}
}

func TestInventoryStatusEffectPanelHonorsNarrowSpaceAndUnknownEffects(t *testing.T) {
	base := NewFramebuffer(100, 80)
	for i := range base.Color {
		base.Color[i] = 0xFF202020
	}
	unchanged := append([]uint32(nil), base.Color...)
	base.DrawInventoryStatusEffects([]world.PlayerStatusEffect{
		{ID: "minecraft:wither", DurationTicks: 200},
		{ID: "minecraft:not_implemented", DurationTicks: 200},
	}, 20, 10, 31, 1)
	for i, pixel := range base.Color {
		if pixel != unchanged[i] {
			t.Fatalf("panel rendered when vanilla has less than 32 logical pixels: pixel %d changed", i)
		}
	}
	base.DrawInventoryStatusEffects([]world.PlayerStatusEffect{{ID: "minecraft:not_implemented", DurationTicks: 200}}, 20, 10, 80, 1)
	for i, pixel := range base.Color {
		if pixel != unchanged[i] {
			t.Fatalf("unsupported effect rendered a placeholder: pixel %d changed", i)
		}
	}
}
