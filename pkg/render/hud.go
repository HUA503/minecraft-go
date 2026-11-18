package render

import (
	"strconv"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// DrawRect fills a 2D rectangle in the framebuffer.
func (fb *Framebuffer) DrawRect(x, y, w, h int, col uint32) {
	for dy := 0; dy < h; dy++ {
		py := y + dy
		if py < 0 || py >= fb.Height {
			continue
		}
		for dx := 0; dx < w; dx++ {
			px := x + dx
			if px < 0 || px >= fb.Width {
				continue
			}
			fb.Color[py*fb.Width+px] = col
		}
	}
}

// DrawCrosshair renders the standard centered "+" crosshair.
func (fb *Framebuffer) DrawCrosshair() {
	cx := fb.Width / 2
	cy := fb.Height / 2
	crosshairColor := uint32(0xFFFFFFFF) // White

	// Horizontal line: 9px
	fb.DrawRect(cx-4, cy, 9, 1, crosshairColor)
	// Vertical line: 9px
	fb.DrawRect(cx, cy-4, 1, 9, crosshairColor)
}

// 3x5 bitmap font for digits 0-9
var font3x5 = [10][5]uint8{
	{0b111, 0b101, 0b101, 0b101, 0b111}, // 0
	{0b010, 0b110, 0b010, 0b010, 0b111}, // 1
	{0b111, 0b001, 0b111, 0b100, 0b111}, // 2
	{0b111, 0b001, 0b111, 0b001, 0b111}, // 3
	{0b101, 0b101, 0b111, 0b001, 0b001}, // 4
	{0b111, 0b100, 0b111, 0b001, 0b111}, // 5
	{0b111, 0b100, 0b111, 0b101, 0b111}, // 6
	{0b111, 0b001, 0b010, 0b010, 0b010}, // 7
	{0b111, 0b101, 0b111, 0b101, 0b111}, // 8
	{0b111, 0b101, 0b111, 0b001, 0b111}, // 9
}

// DrawDigit renders a single 3x5 font character.
func (fb *Framebuffer) DrawDigit(x, y int, digit int, col uint32) {
	if digit < 0 || digit > 9 {
		return
	}
	glyph := font3x5[digit]
	for row := 0; row < 5; row++ {
		bits := glyph[row]
		for colIdx := 0; colIdx < 3; colIdx++ {
			if (bits & (1 << (2 - colIdx))) != 0 {
				px := x + colIdx
				py := y + row
				if px >= 0 && px < fb.Width && py >= 0 && py < fb.Height {
					fb.Color[py*fb.Width+px] = col
				}
			}
		}
	}
}

// DrawNumber renders a right-aligned integer number with shadow.
func (fb *Framebuffer) DrawNumber(x, y int, num int, col uint32) {
	if num < 0 {
		return
	}
	s := strconv.Itoa(num)
	startX := x - len(s)*4 + 1
	for i, ch := range s {
		d := int(ch - '0')
		// Drop shadow
		fb.DrawDigit(startX+i*4+1, y+1, d, 0xFF000000)
		// Character
		fb.DrawDigit(startX+i*4, y, d, col)
	}
}

// DrawItemIcon renders a 12x12 mini item or block icon inside a hotbar slot.
func (fb *Framebuffer) DrawItemIcon(x, y int, itemID int32) {
	switch itemID {
	case inventory.ItemStone:
		// Stone: 12x12 textured gray block
		fb.DrawRect(x, y, 12, 12, 0xFF7D7D7D)
		fb.DrawRect(x+2, y+2, 3, 2, 0xFF969696)
		fb.DrawRect(x+7, y+6, 3, 3, 0xFF636363)
		fb.DrawRect(x+3, y+8, 2, 2, 0xFF969696)

	case inventory.ItemGrass:
		// Grass: 12x12 block with vibrant green top and rich earth brown bottom
		fb.DrawRect(x, y+4, 12, 8, 0xFF866043)
		fb.DrawRect(x, y, 12, 4, 0xFF5AA53C)
		fb.DrawRect(x+1, y+3, 2, 2, 0xFF46862E)
		fb.DrawRect(x+6, y+3, 2, 2, 0xFF46862E)
		fb.DrawRect(x+9, y+3, 2, 2, 0xFF46862E)

	case inventory.ItemDirt:
		// Dirt: 12x12 warm earth brown
		fb.DrawRect(x, y, 12, 12, 0xFF866043)
		fb.DrawRect(x+2, y+3, 2, 2, 0xFF6F4F37)
		fb.DrawRect(x+7, y+7, 2, 2, 0xFF9B6F4E)
		fb.DrawRect(x+3, y+8, 2, 2, 0xFF6F4F37)

	case inventory.ItemCobblestone:
		// Cobblestone: 12x12 mottled stone bricks
		fb.DrawRect(x, y, 12, 12, 0xFF6C6C6C)
		fb.DrawRect(x+1, y+1, 4, 3, 0xFF888888)
		fb.DrawRect(x+7, y+2, 4, 3, 0xFF505050)
		fb.DrawRect(x+2, y+6, 5, 4, 0xFF505050)
		fb.DrawRect(x+8, y+7, 3, 3, 0xFF888888)

	case inventory.ItemOakPlanks:
		// Oak Planks: warm tan boards with grain
		fb.DrawRect(x, y, 12, 12, 0xFFB8945F)
		fb.DrawRect(x, y+3, 12, 1, 0xFF8F6E3E)
		fb.DrawRect(x, y+7, 12, 1, 0xFF8F6E3E)
		fb.DrawRect(x, y+11, 12, 1, 0xFF8F6E3E)
		fb.DrawRect(x+5, y, 1, 3, 0xFF8F6E3E)
		fb.DrawRect(x+8, y+4, 1, 3, 0xFF8F6E3E)

	case inventory.ItemOakLog:
		// Oak Log: tree trunk bark edge with lighter wood rings
		fb.DrawRect(x, y, 12, 12, 0xFF4D3B26)
		fb.DrawRect(x+2, y+2, 8, 8, 0xFF9E7C4D)
		fb.DrawRect(x+4, y+4, 4, 4, 0xFFB8945F)

	case inventory.ItemWoodenPickaxe, inventory.ItemStonePickaxe, inventory.ItemIronPickaxe, inventory.ItemDiamondPickaxe:
		// Diagonal pickaxe
		headCol := uint32(0xFFB8945F) // Wood
		if itemID == inventory.ItemStonePickaxe {
			headCol = 0xFF888888
		} else if itemID == inventory.ItemIronPickaxe {
			headCol = 0xFFEEEEEE
		} else if itemID == inventory.ItemDiamondPickaxe {
			headCol = 0xFF4AEDD7
		}
		// Stick
		for d := 0; d < 7; d++ {
			fb.DrawRect(x+2+d, y+10-d, 1, 1, 0xFF6F4F37)
		}
		// Pickaxe head
		fb.DrawRect(x+6, y+2, 5, 2, headCol)
		fb.DrawRect(x+9, y+4, 2, 3, headCol)
		fb.DrawRect(x+4, y+3, 2, 2, headCol)

	case 268, 272, 267, 276:
		// Sword (268 Wood, 272 Stone, 267 Iron, 276 Diamond)
		bladeCol := uint32(0xFFB8945F)
		if itemID == 272 {
			bladeCol = 0xFF888888
		} else if itemID == 267 {
			bladeCol = 0xFFEEEEEE
		} else if itemID == 276 {
			bladeCol = 0xFF4AEDD7
		}
		// Blade
		for d := 0; d < 6; d++ {
			fb.DrawRect(x+5+d, y+6-d, 2, 2, bladeCol)
		}
		// Crossguard
		fb.DrawRect(x+3, y+8, 4, 2, 0xFF4D3B26)
		// Handle
		fb.DrawRect(x+2, y+9, 2, 2, 0xFF6F4F37)

	case 271, 275, 258, 279:
		// Axe (Wood, Stone, Iron, Diamond)
		headCol := uint32(0xFFB8945F)
		if itemID == 275 {
			headCol = 0xFF888888
		} else if itemID == 258 {
			headCol = 0xFFEEEEEE
		} else if itemID == 279 {
			headCol = 0xFF4AEDD7
		}
		for d := 0; d < 8; d++ {
			fb.DrawRect(x+2+d, y+10-d, 1, 1, 0xFF6F4F37)
		}
		fb.DrawRect(x+6, y+1, 4, 5, headCol)

	case 269, 273, 256, 277:
		// Shovel (Wood, Stone, Iron, Diamond)
		headCol := uint32(0xFFB8945F)
		if itemID == 273 {
			headCol = 0xFF888888
		} else if itemID == 256 {
			headCol = 0xFFEEEEEE
		} else if itemID == 277 {
			headCol = 0xFF4AEDD7
		}
		for d := 0; d < 7; d++ {
			fb.DrawRect(x+2+d, y+10-d, 1, 1, 0xFF6F4F37)
		}
		fb.DrawRect(x+7, y+2, 4, 4, headCol)

	case inventory.ItemCoal:
		// Coal lump
		fb.DrawRect(x+3, y+3, 6, 6, 0xFF242424)
		fb.DrawRect(x+4, y+2, 4, 8, 0xFF181818)

	case inventory.ItemDiamond:
		// Diamond gem
		fb.DrawRect(x+4, y+2, 4, 8, 0xFF4AEDD7)
		fb.DrawRect(x+2, y+4, 8, 4, 0xFF35C0AD)
		fb.DrawRect(x+4, y+4, 4, 4, 0xFFE0FFFF)

	case inventory.ItemCraftingTable:
		fb.DrawRect(x, y, 12, 12, 0xFFB8945F)
		fb.DrawRect(x+2, y+2, 8, 8, 0xFF8F6E3E)

	case inventory.ItemFurnace:
		fb.DrawRect(x, y, 12, 12, 0xFF606060)
		fb.DrawRect(x+4, y+5, 4, 4, 0xFF151515)

	default:
		// Generic item square
		fb.DrawRect(x+1, y+1, 10, 10, 0xFF888888)
	}
}

// DrawHUD renders crosshair, health bar, hunger bar, and 9 empty hotbar slots.
func (fb *Framebuffer) DrawHUD(health float32, hunger int, selectedSlot int) {
	var emptySlots [9]inventory.ItemStack
	fb.DrawHUDWithInventory(health, hunger, selectedSlot, emptySlots)
}

// DrawHUDWithInventory renders crosshair, health, hunger, 9 hotbar slots with item icons and counts.
func (fb *Framebuffer) DrawHUDWithInventory(health float32, hunger int, selectedSlot int, slots [9]inventory.ItemStack) {
	// 1. Center Crosshair
	fb.DrawCrosshair()

	// 2. Health bar (bottom-left above hotbar, e.g. Y = Height - 55)
	hudBaseY := fb.Height - 55
	healthStartX := fb.Width/2 - 180

	numHearts := int(health / 2.0)
	for i := 0; i < 10; i++ {
		hx := healthStartX + i*10
		// Dark background border
		fb.DrawRect(hx, hudBaseY, 8, 8, 0xFF330000)
		if i < numHearts {
			// Red heart
			fb.DrawRect(hx+1, hudBaseY+1, 6, 6, 0xFFFF1111) // Minecraft Authentic Red
		}
	}

	// 3. Hunger bar (bottom-right above hotbar)
	hungerStartX := fb.Width/2 + 80
	numDrumsticks := hunger / 2
	for i := 0; i < 10; i++ {
		hx := hungerStartX + i*10
		fb.DrawRect(hx, hudBaseY, 8, 8, 0xFF2A1E14)
		if i < numDrumsticks {
			// Orange/brown drumstick
			fb.DrawRect(hx+1, hudBaseY+1, 6, 6, 0xFFD47C2F) // Minecraft Authentic Cooked Meat
		}
	}

	// 4. Hotbar (9 slots centered at bottom, Y = Height - 35)
	hotbarW := 182
	hotbarH := 22
	hotbarX := fb.Width/2 - hotbarW/2
	hotbarY := fb.Height - 35

	// Hotbar background
	fb.DrawRect(hotbarX, hotbarY, hotbarW, hotbarH, 0x88404040)

	// Draw each slot
	for i := 0; i < 9; i++ {
		slotX := hotbarX + i*20 + 2
		slotY := hotbarY + 2
		// Slot border
		fb.DrawRect(slotX, slotY, 18, 18, 0xFF222222)

		// Render item icon inside slot
		item := slots[i]
		if !item.IsEmpty() {
			fb.DrawItemIcon(slotX+3, slotY+3, item.ItemID)
			// Render count text in lower right corner
			if item.Count > 1 {
				fb.DrawNumber(slotX+16, slotY+12, item.Count, 0xFFFFFFFF)
			}
		}

		// Selected slot white border highlight
		if i == selectedSlot {
			fb.DrawRect(slotX-1, slotY-1, 20, 2, 0xFFFFFFFF)
			fb.DrawRect(slotX-1, slotY+17, 20, 2, 0xFFFFFFFF)
			fb.DrawRect(slotX-1, slotY-1, 2, 20, 0xFFFFFFFF)
			fb.DrawRect(slotX+17, slotY-1, 2, 20, 0xFFFFFFFF)
		}
	}
}

// DrawInventoryGUI renders an authentic full-screen Minecraft Inventory and Crafting dialog.
func (fb *Framebuffer) DrawInventoryGUI(inv *inventory.PlayerInventory) {
	if inv == nil {
		return
	}

	// 1. Dark semi-transparent background overlay
	for y := 0; y < fb.Height; y++ {
		rowOffset := y * fb.Width
		for x := 0; x < fb.Width; x++ {
			c := fb.Color[rowOffset+x]
			r := ((c >> 16) & 0xFF) / 3
			g := ((c >> 8) & 0xFF) / 3
			b := (c & 0xFF) / 3
			fb.Color[rowOffset+x] = 0xFF000000 | (r << 16) | (g << 8) | b
		}
	}

	// 2. Centered dialog box (280x210)
	gw := 280
	gh := 210
	gx := fb.Width/2 - gw/2
	gy := fb.Height/2 - gh/2

	// Dialog outer border and background
	fb.DrawRect(gx, gy, gw, gh, 0xFF373737)
	fb.DrawRect(gx+2, gy+2, gw-4, gh-4, 0xFFC6C6C6)
	fb.DrawRect(gx+2, gy+2, gw-4, 2, 0xFFFFFFFF)
	fb.DrawRect(gx+2, gy+2, 2, gh-4, 0xFFFFFFFF)
	fb.DrawRect(gx+gw-4, gy+2, 2, gh-4, 0xFF555555)
	fb.DrawRect(gx+2, gy+gh-4, gw-4, 2, 0xFF555555)

	// Crafting section (Top-Left)
	craftX := gx + 20
	craftY := gy + 15
	fb.DrawRect(craftX, craftY, 40, 40, 0xFF888888)
	// 2x2 Crafting slots
	for cy := 0; cy < 2; cy++ {
		for cx := 0; cx < 2; cx++ {
			sx := craftX + 2 + cx*19
			sy := craftY + 2 + cy*19
			fb.DrawRect(sx, sy, 17, 17, 0xFF373737)
			fb.DrawRect(sx+1, sy+1, 15, 15, 0xFF8B8B8B)
		}
	}
	// Crafting arrow ->
	arrowX := craftX + 46
	arrowY := craftY + 16
	fb.DrawRect(arrowX, arrowY+3, 12, 3, 0xFF555555)
	fb.DrawRect(arrowX+8, arrowY, 3, 9, 0xFF555555)
	fb.DrawRect(arrowX+10, arrowY+1, 2, 7, 0xFF555555)

	// Result slot
	resX := arrowX + 18
	resY := craftY + 9
	fb.DrawRect(resX, resY, 24, 24, 0xFF373737)
	fb.DrawRect(resX+1, resY+1, 22, 22, 0xFF8B8B8B)

	// Quick Craft Help text area (Top-Right)
	helpX := gx + 120
	helpY := gy + 12
	fb.DrawRect(helpX, helpY, 145, 55, 0xFF8F8F8F)
	fb.DrawRect(helpX+1, helpY+1, 143, 53, 0xFFB0B0B0)

	// Draw miniature recipe guide icons
	recipes := []struct {
		key    int
		inID   int32
		outID  int32
		outCnt int
	}{
		{1, inventory.ItemOakLog, inventory.ItemOakPlanks, 4},
		{2, inventory.ItemOakPlanks, 280, 4}, // Sticks
		{3, inventory.ItemOakPlanks, inventory.ItemCraftingTable, 1},
		{4, inventory.ItemOakPlanks, inventory.ItemWoodenPickaxe, 1},
		{5, inventory.ItemOakPlanks, 268, 1}, // Wooden Sword
		{6, inventory.ItemCobblestone, inventory.ItemStonePickaxe, 1},
		{7, inventory.ItemCobblestone, inventory.ItemFurnace, 1},
	}

	for i, r := range recipes {
		rx := helpX + 4 + (i%4)*35
		ry := helpY + 4 + (i/4)*26
		fb.DrawDigit(rx, ry+5, r.key, 0xFF000000)
		fb.DrawRect(rx+5, ry+6, 2, 2, 0xFF444444)
		fb.DrawItemIcon(rx+9, ry+1, r.outID)
		if r.outCnt > 1 {
			fb.DrawDigit(rx+23, ry+8, r.outCnt, 0xFFFFFFFF)
		}
	}

	// 3. Main Storage Inventory (3 rows of 9 slots: Slots 9..35)
	invStartX := gx + 18
	invStartY := gy + 75

	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			slotIdx := 9 + row*9 + col
			sx := invStartX + col*27
			sy := invStartY + row*24

			fb.DrawRect(sx, sy, 22, 22, 0xFF373737)
			fb.DrawRect(sx+1, sy+1, 20, 20, 0xFF8B8B8B)

			item := inv.Slots[slotIdx]
			if !item.IsEmpty() {
				fb.DrawItemIcon(sx+4, sy+4, item.ItemID)
				if item.Count > 1 {
					fb.DrawNumber(sx+19, sy+14, item.Count, 0xFFFFFFFF)
				}
			}
		}
	}

	// 4. Hotbar (1 row of 9 slots: Slots 0..8)
	hotbarY := gy + 160
	for col := 0; col < 9; col++ {
		sx := invStartX + col*27
		sy := hotbarY

		fb.DrawRect(sx, sy, 22, 22, 0xFF373737)
		fb.DrawRect(sx+1, sy+1, 20, 20, 0xFF8B8B8B)

		item := inv.Slots[col]
		if !item.IsEmpty() {
			fb.DrawItemIcon(sx+4, sy+4, item.ItemID)
			if item.Count > 1 {
				fb.DrawNumber(sx+19, sy+14, item.Count, 0xFFFFFFFF)
			}
		}

		if col == inv.SelectedHotbar {
			fb.DrawRect(sx, sy, 22, 2, 0xFFFFFFFF)
			fb.DrawRect(sx, sy+20, 22, 2, 0xFFFFFFFF)
			fb.DrawRect(sx, sy, 2, 22, 0xFFFFFFFF)
			fb.DrawRect(sx+20, sy, 2, 22, 0xFFFFFFFF)
		}
	}
}
