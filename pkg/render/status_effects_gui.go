package render

import (
	"fmt"
	"strings"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

type statusEffectVisual struct {
	texture string
	name    string
}

var statusEffectVisuals = map[string]statusEffectVisual{
	"minecraft:absorption":      {"mob_effect/absorption.png", "伤害吸收"},
	"minecraft:fire_resistance": {"mob_effect/fire_resistance.png", "抗火"},
	"minecraft:hunger":          {"mob_effect/hunger.png", "饥饿"},
	"minecraft:nausea":          {"mob_effect/nausea.png", "反胃"},
	"minecraft:poison":          {"mob_effect/poison.png", "中毒"},
	"minecraft:regeneration":    {"mob_effect/regeneration.png", "生命恢复"},
	"minecraft:resistance":      {"mob_effect/resistance.png", "抗性提升"},
	"minecraft:wither":          {"mob_effect/wither.png", "凋零"},
}

// DrawInventoryStatusEffects draws the vanilla inventory's right-side active
// effect list. x/y are framebuffer pixels; availableWidth is in GUI units.
func (fb *Framebuffer) DrawInventoryStatusEffects(effects []world.PlayerStatusEffect, x, y, availableWidth, scale int) {
	if fb == nil || scale < 1 || availableWidth < 32 {
		return
	}
	visible := make([]struct {
		effect world.PlayerStatusEffect
		visual statusEffectVisual
		icon   *RawImage
	}, 0, len(effects))
	for _, effect := range effects {
		if effect.ID == "" || effect.DurationTicks <= 0 {
			continue
		}
		visual, ok := statusEffectVisuals[effect.ID]
		if !ok {
			continue
		}
		icon, ok := GetOfficialImage(visual.texture)
		if !ok {
			continue
		}
		visible = append(visible, struct {
			effect world.PlayerStatusEffect
			visual statusEffectVisual
			icon   *RawImage
		}{effect, visual, icon})
	}
	if len(visible) == 0 {
		return
	}
	spacing := 33
	if len(visible) > 5 {
		spacing = 132 / (len(visible) - 1)
	}
	textAllowed := availableWidth >= 120
	maxWidth := 32
	if textAllowed {
		maxWidth = availableWidth - 7
	}
	background, ok := GetOfficialImage("gui/sprites/container/inventory/effect_background.png")
	if !ok {
		return
	}
	for i, entry := range visible {
		rowY := y + i*spacing*scale
		width := 32
		if textAllowed {
			name := effectDisplayName(entry.visual.name, entry.effect.Amplifier)
			seconds := entry.effect.DurationTicks / 20
			duration := fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
			width = min(maxWidth, max(32+MeasureText5x7(name, scale)/scale+7, 32+MeasureText5x7(duration, scale)/scale+7))
		}
		drawNineSliceRawImage(fb, background, x, rowY, scale, width*scale, 32*scale, 4, true)
		fb.DrawRawImageScaled(x+7*scale, rowY+7*scale, scale, entry.icon)
		if width > 32 {
			name := effectDisplayName(entry.visual.name, entry.effect.Amplifier)
			seconds := entry.effect.DurationTicks / 20
			fb.DrawText5x7(x+32*scale, rowY+7*scale, name, 0xFFFFFFFF, scale)
			fb.DrawText5x7(x+32*scale, rowY+17*scale, fmt.Sprintf("%d:%02d", seconds/60, seconds%60), 0xFFAAAAAA, scale)
		}
	}
}

// DrawCreativeInventoryGUIWithEffects preserves the creative screen's search
// caret and saved-hotbar state while rendering the vanilla side panel.
func (fb *Framebuffer) DrawCreativeInventoryGUIWithEffects(inv *inventory.PlayerInventory, cursor inventory.ItemStack, category inventory.CreativeCategory, query string, scrollRow, mouseX, mouseY, searchCursor, searchAnchor int, searchFocused bool, presets inventory.CreativeHotbarPresets, effects []world.PlayerStatusEffect) {
	fb.DrawCreativeInventoryGUIWithSearchState(inv, cursor, category, query, scrollRow, mouseX, mouseY, searchCursor, searchAnchor, searchFocused, presets)
	if fb == nil {
		return
	}
	scale, px, py := creativeGUIOrigin(fb.Width, fb.Height)
	x := px + (creativePanelW+2)*scale
	available := (fb.Width - x) / scale
	fb.DrawInventoryStatusEffects(effects, x, py, available, scale)
}

// DrawCraftingGUIWithEffects renders the normal inventory/crafting screen and
// the same right-side status panel shown by vanilla.
func (fb *Framebuffer) DrawCraftingGUIWithEffects(inv *inventory.PlayerInventory, craftGrid []inventory.ItemStack, craftResult inventory.ItemStack, is3x3 bool, cursorItem inventory.ItemStack, mouseX, mouseY int, effects []world.PlayerStatusEffect) {
	fb.DrawCraftingGUI(inv, craftGrid, craftResult, is3x3, cursorItem, mouseX, mouseY)
	if fb == nil {
		return
	}
	scale := fb.CalculateGUIScale()
	panelW, panelH := 176, 166
	px := fb.Width/2 - panelW*scale/2
	py := fb.Height/2 - panelH*scale/2
	x := px + (panelW+2)*scale
	available := (fb.Width - x) / scale
	fb.DrawInventoryStatusEffects(effects, x, py, available, scale)
}

func effectDisplayName(name string, amplifier int) string {
	if amplifier <= 0 {
		return name
	}
	level := amplifier + 1
	if level <= 10 {
		return name + " " + []string{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X"}[level]
	}
	return name + " " + strings.Repeat("I", min(level, 16))
}
