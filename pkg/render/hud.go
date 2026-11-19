package render

import (
	"strconv"
	"time"

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

	case inventory.ItemApple:
		// Apple: bright red fruit with leaf stem
		fb.DrawRect(x+3, y+3, 6, 7, 0xFFFF2222)
		fb.DrawRect(x+4, y+2, 4, 8, 0xFFCC1111)
		fb.DrawRect(x+5, y+1, 2, 2, 0xFF4D3B26) // Stem
		fb.DrawRect(x+7, y+1, 2, 1, 0xFF46862E) // Leaf

	case inventory.ItemBread:
		// Bread: golden brown loaf
		fb.DrawRect(x+2, y+4, 8, 5, 0xFFC99742)
		fb.DrawRect(x+3, y+3, 6, 6, 0xFFDFAB56)
		fb.DrawRect(x+4, y+4, 1, 3, 0xFF8F6322)
		fb.DrawRect(x+7, y+4, 1, 3, 0xFF8F6322)

	case inventory.ItemRawPorkchop:
		// Raw porkchop: pink flesh with bone
		fb.DrawRect(x+3, y+3, 7, 6, 0xFFFFAAAA)
		fb.DrawRect(x+2, y+4, 8, 4, 0xFFFF8888)
		fb.DrawRect(x+7, y+2, 2, 2, 0xFFEEEEEE) // Bone

	case inventory.ItemCookedPorkchop, inventory.ItemSteak:
		// Cooked meat / steak: rich savory brown with sear marks
		meatCol := uint32(0xFF864A2B)
		if itemID == inventory.ItemCookedPorkchop {
			meatCol = 0xFFB0683B
		}
		fb.DrawRect(x+2, y+3, 8, 6, meatCol)
		fb.DrawRect(x+3, y+4, 6, 4, meatCol+0x00111111)
		fb.DrawRect(x+4, y+4, 1, 4, 0xFF4A2513) // Grill mark
		fb.DrawRect(x+7, y+4, 1, 4, 0xFF4A2513)

	case inventory.ItemRawBeef:
		// Raw beef: deep crimson red with marbling
		fb.DrawRect(x+2, y+3, 8, 6, 0xFFC02222)
		fb.DrawRect(x+3, y+4, 6, 4, 0xFF991818)
		fb.DrawRect(x+4, y+4, 2, 1, 0xFFEEEEEE) // Fat marbling

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
		hy := hudBaseY
		if health <= 4.0 {
			// Minecraft Java edition authentic heart shaking at low health
			shake := (i*7 + int(time.Now().UnixMilli()/70)) % 3
			hy += shake - 1
		}
		// Dark background border
		fb.DrawRect(hx, hy, 8, 8, 0xFF330000)
		if i < numHearts {
			// Red heart
			fb.DrawRect(hx+1, hy+1, 6, 6, 0xFFFF1111) // Minecraft Authentic Red
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

// 5x7 ASCII bitmap font for authentic Minecraft GUI text & menus
var font5x7 = map[rune][7]uint8{
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'B': {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'C': {0b01111, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b01111},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'F': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'G': {0b01111, 0b10000, 0b10000, 0b10111, 0b10001, 0b10001, 0b01111},
	'H': {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'I': {0b01110, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'J': {0b00001, 0b00001, 0b00001, 0b00001, 0b10001, 0b10001, 0b01110},
	'K': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'N': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'O': {0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'Q': {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'V': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W': {0b10001, 0b10001, 0b10001, 0b10101, 0b10101, 0b11011, 0b10001},
	'X': {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'Y': {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100, 0b00100},
	'Z': {0b11111, 0b00010, 0b00100, 0b01000, 0b10000, 0b10000, 0b11111},
	'0': {0b01110, 0b10011, 0b10101, 0b10101, 0b11001, 0b10001, 0b01110},
	'1': {0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b01110},
	'2': {0b01110, 0b10001, 0b00001, 0b00110, 0b01000, 0b10000, 0b11111},
	'3': {0b11110, 0b00001, 0b00001, 0b01110, 0b00001, 0b00001, 0b11110},
	'4': {0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010},
	'5': {0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110},
	'6': {0b01110, 0b10000, 0b11110, 0b10001, 0b10001, 0b10001, 0b01110},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8': {0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110},
	'9': {0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00001, 0b01110},
	' ': {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000},
	'-': {0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000},
	':': {0b00000, 0b01100, 0b01100, 0b00000, 0b01100, 0b01100, 0b00000},
	'(': {0b00010, 0b00100, 0b01000, 0b01000, 0b01000, 0b00100, 0b00010},
	')': {0b01000, 0b00100, 0b00010, 0b00010, 0b00010, 0b00100, 0b01000},
	'!': {0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00000, 0b00100},
	'.': {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b01100, 0b01100},
	'/': {0b00001, 0b00010, 0b00100, 0b00100, 0b01000, 0b10000, 0b00000},
	'?': {0b01110, 0b10001, 0b00010, 0b00100, 0b00100, 0b00000, 0b00100},
}

// DrawChar5x7 renders a single 5x7 character scaled with optional drop shadow.
func (fb *Framebuffer) DrawChar5x7(x, y int, ch rune, col uint32, scale int) int {
	if ch >= 'a' && ch <= 'z' {
		ch = ch - 'a' + 'A'
	}
	glyph, ok := font5x7[ch]
	if !ok {
		glyph = font5x7['?']
	}

	for row := 0; row < 7; row++ {
		bits := glyph[row]
		for cIdx := 0; cIdx < 5; cIdx++ {
			if (bits & (1 << (4 - cIdx))) != 0 {
				if scale == 1 {
					px := x + cIdx
					py := y + row
					if px >= 0 && px < fb.Width && py >= 0 && py < fb.Height {
						fb.Color[py*fb.Width+px] = col
					}
				} else {
					fb.DrawRect(x+cIdx*scale, y+row*scale, scale, scale, col)
				}
			}
		}
	}
	return (5 + 1) * scale
}

// DrawText5x7 renders an entire string with Minecraft-style drop shadow.
func (fb *Framebuffer) DrawText5x7(x, y int, text string, col uint32, scale int) int {
	shadowCol := uint32(0xFF222222)
	// Render drop shadow first (offset by scale)
	curX := x + scale
	curY := y + scale
	for _, ch := range text {
		curX += fb.DrawChar5x7(curX, curY, ch, shadowCol, scale)
	}

	// Render crisp foreground text
	curX = x
	curY = y
	for _, ch := range text {
		curX += fb.DrawChar5x7(curX, curY, ch, col, scale)
	}
	return curX - x
}

// MeasureText5x7 returns the pixel width of the text when rendered at the specified scale.
func MeasureText5x7(text string, scale int) int {
	return len(text) * 6 * scale
}

// DrawButton renders an authentic Minecraft beveled button with hover styling.
func (fb *Framebuffer) DrawButton(x, y, w, h int, text string, hovered bool) {
	// Outer border
	fb.DrawRect(x, y, w, h, 0xFF000000)

	// Inner bevel highlight / shadow
	topHighlight := uint32(0xFF8B8B8B)
	bottomShadow := uint32(0xFF373737)
	bodyColor := uint32(0xFF5A5A5A)
	textColor := uint32(0xFFE0E0E0)

	if hovered {
		topHighlight = 0xFFFFFFFF
		bottomShadow = 0xFF4A4A4A
		bodyColor = 0xFF7A8EAE // Minecraft hover blue-gray tint
		textColor = 0xFFFFFFA0 // Classic yellow hover text
	}

	// Fill body
	fb.DrawRect(x+2, y+2, w-4, h-4, bodyColor)

	// Highlights: top and left inner edge
	fb.DrawRect(x+1, y+1, w-2, 1, topHighlight)
	fb.DrawRect(x+1, y+1, 1, h-2, topHighlight)

	// Shadows: bottom and right inner edge
	fb.DrawRect(x+1, y+h-2, w-2, 1, bottomShadow)
	fb.DrawRect(x+w-2, y+1, 1, h-2, bottomShadow)

	// Center text
	textWidth := MeasureText5x7(text, 1)
	textX := x + (w-textWidth)/2
	textY := y + (h-7)/2
	fb.DrawText5x7(textX, textY, text, textColor, 1)
}

// DrawPauseMenu dims the screen and renders the Minecraft Java Edition Game Menu (Pause).
// Returns the clickable bounding rectangles [4]int{x, y, w, h} for "Back to Game" and "Quit Game".
func (fb *Framebuffer) DrawPauseMenu(mouseX, mouseY int) (backBtnRect [4]int, quitBtnRect [4]int) {
	// 1. Dim background by 60% with authentic translucent overlay
	for i := 0; i < len(fb.Color); i++ {
		c := fb.Color[i]
		r := ((c >> 16) & 0xFF) * 4 / 10
		g := ((c >> 8) & 0xFF) * 4 / 10
		b := (c & 0xFF) * 4 / 10
		fb.Color[i] = 0xFF000000 | (r << 16) | (g << 8) | b
	}

	// 2. Title: "GAME PAUSED" (large scale 2)
	title := "GAME PAUSED"
	titleWidth := MeasureText5x7(title, 2)
	titleX := (fb.Width - titleWidth) / 2
	titleY := fb.Height/4 - 20
	fb.DrawText5x7(titleX, titleY, title, 0xFFFFFFFF, 2)

	// 3. Button 1: "BACK TO GAME"
	btnW := 220
	btnH := 28
	btnX := (fb.Width - btnW) / 2
	btn1Y := fb.Height/2 - 30
	hover1 := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= btn1Y && mouseY <= btn1Y+btnH
	fb.DrawButton(btnX, btn1Y, btnW, btnH, "BACK TO GAME", hover1)
	backBtnRect = [4]int{btnX, btn1Y, btnW, btnH}

	// 4. Button 2: "QUIT GAME (PRESS Q)"
	btn2Y := fb.Height/2 + 10
	hover2 := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= btn2Y && mouseY <= btn2Y+btnH
	fb.DrawButton(btnX, btn2Y, btnW, btnH, "QUIT GAME (PRESS Q)", hover2)
	quitBtnRect = [4]int{btnX, btn2Y, btnW, btnH}

	// 5. Bottom Instructions hint
	hint := "PRESS ESC TO RESUME - PRESS Q TO QUIT"
	hintWidth := MeasureText5x7(hint, 1)
	hintX := (fb.Width - hintWidth) / 2
	hintY := fb.Height - 32
	fb.DrawText5x7(hintX, hintY, hint, 0xFFAAAAAA, 1)

	return backBtnRect, quitBtnRect
}

// DebugInfo holds data shown on the Minecraft F3 debug screen.
type DebugInfo struct {
	FPS           int
	FrameTimeMs   float64
	X, Y, Z       float64
	BlockX, BlockY, BlockZ int32
	ChunkX, ChunkZ int32
	Facing        string
	Yaw, Pitch    float64
	TimeOfDay     int64
	TargetedBlock string
	HasTarget     bool
	TargetPos     [3]int32
}

// DrawDebugScreen renders the authentic Minecraft Java Edition F3 debug overlay.
func (fb *Framebuffer) DrawDebugScreen(info DebugInfo) {
	leftLines := []string{
		"Minecraft Go 1.0.0 (Windows pure Go)",
		"FPS: " + strconv.Itoa(info.FPS) + " (" + strconv.FormatFloat(info.FrameTimeMs, 'f', 2, 64) + " ms)",
		"XYZ: " + strconv.FormatFloat(info.X, 'f', 3, 64) + " / " + strconv.FormatFloat(info.Y, 'f', 3, 64) + " / " + strconv.FormatFloat(info.Z, 'f', 3, 64),
		"Block: " + strconv.Itoa(int(info.BlockX)) + " " + strconv.Itoa(int(info.BlockY)) + " " + strconv.Itoa(int(info.BlockZ)),
		"Chunk: " + strconv.Itoa(int(info.ChunkX)) + " " + strconv.Itoa(int(info.ChunkZ)),
		"Facing: " + info.Facing + " (Yaw: " + strconv.FormatFloat(info.Yaw, 'f', 1, 64) + ", Pitch: " + strconv.FormatFloat(info.Pitch, 'f', 1, 64) + ")",
		"Time: " + strconv.FormatInt(info.TimeOfDay, 10),
	}

	curY := 4
	for _, line := range leftLines {
		lineWidth := MeasureText5x7(line, 1)
		fb.DrawRect(4, curY, lineWidth+4, 9, 0xA0151515)
		fb.DrawText5x7(6, curY+1, line, 0xFFFFFFFF, 1)
		curY += 10
	}

	if info.HasTarget {
		rightLines := []string{
			"Targeted Block: " + info.TargetedBlock,
			"Target Pos: " + strconv.Itoa(int(info.TargetPos[0])) + " " + strconv.Itoa(int(info.TargetPos[1])) + " " + strconv.Itoa(int(info.TargetPos[2])),
		}
		curY = 4
		for _, line := range rightLines {
			lineWidth := MeasureText5x7(line, 1)
			startX := fb.Width - lineWidth - 8
			fb.DrawRect(startX-2, curY, lineWidth+4, 9, 0xA0151515)
			fb.DrawText5x7(startX, curY+1, line, 0xFFFFFFFF, 1)
			curY += 10
		}
	}
}

// ChatMessage represents an in-game chat log message with creation timestamp.
type ChatMessage struct {
	Text      string
	Timestamp time.Time
}

// DrawChatOverlay renders the authentic Minecraft Java Edition chat history and command input bar.
func (fb *Framebuffer) DrawChatOverlay(messages []ChatMessage, isTyping bool, inputBuffer string) {
	now := time.Now()
	// Render recent chat log entries from bottom up
	startY := fb.Height - 56
	if isTyping {
		startY = fb.Height - 44
	}

	linesDrawn := 0
	for i := len(messages) - 1; i >= 0 && linesDrawn < 10; i-- {
		msg := messages[i]
		age := now.Sub(msg.Timestamp).Seconds()
		if !isTyping && age > 8.0 {
			continue // Faded out in normal play
		}

		lineY := startY - linesDrawn*12
		if lineY < 20 {
			break
		}

		textWidth := MeasureText5x7(msg.Text, 1)
		fb.DrawRect(4, lineY-1, textWidth+6, 11, 0x90000000)
		fb.DrawText5x7(6, lineY+1, msg.Text, 0xFFFFFFFF, 1)
		linesDrawn++
	}

	// Render interactive command/chat text entry box when user is typing (toggled with T or /)
	if isTyping {
		boxY := fb.Height - 26
		fb.DrawRect(4, boxY, fb.Width-8, 16, 0xCC000000)
		fb.DrawRect(4, boxY, fb.Width-8, 1, 0xFF555555)

		displayText := "> " + inputBuffer
		if (now.UnixMilli()/500)%2 == 0 {
			displayText += "_"
		}
		fb.DrawText5x7(8, boxY+4, displayText, 0xFFFFFFFF, 1)
	}
}

// BlendPixel alpha-blends an ARGB color over the existing pixel in the framebuffer.
func (fb *Framebuffer) BlendPixel(x, y int, argb uint32) {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return
	}
	idx := y*fb.Width + x
	a := (argb >> 24) & 0xFF
	if a == 0 {
		return
	}
	if a == 255 {
		fb.Color[idx] = argb | 0xFF000000
		return
	}
	invA := 255 - a
	sr := (argb >> 16) & 0xFF
	sg := (argb >> 8) & 0xFF
	sb := argb & 0xFF

	dc := fb.Color[idx]
	dr := (dc >> 16) & 0xFF
	dg := (dc >> 8) & 0xFF
	db := dc & 0xFF

	rr := (sr*a + dr*invA) / 255
	rg := (sg*a + dg*invA) / 255
	rb := (sb*a + db*invA) / 255

	fb.Color[idx] = 0xFF000000 | (rr << 16) | (rg << 8) | rb
}

// DrawHurtVignette renders the authentic Minecraft red flash vignette along screen borders when damaged.
func (fb *Framebuffer) DrawHurtVignette(intensity float32) {
	if intensity <= 0 {
		return
	}
	if intensity > 1.0 {
		intensity = 1.0
	}
	vignetteThickness := 28
	alphaMax := uint32(float32(180) * intensity)

	// Top & Bottom gradient strips
	for y := 0; y < vignetteThickness; y++ {
		curAlpha := uint32(float32(alphaMax) * float32(vignetteThickness-y) / float32(vignetteThickness))
		redCol := (curAlpha << 24) | 0x00FF0000
		for x := 0; x < fb.Width; x++ {
			fb.BlendPixel(x, y, redCol)
			fb.BlendPixel(x, fb.Height-1-y, redCol)
		}
	}
	// Left & Right gradient strips
	for x := 0; x < vignetteThickness; x++ {
		curAlpha := uint32(float32(alphaMax) * float32(vignetteThickness-x) / float32(vignetteThickness))
		redCol := (curAlpha << 24) | 0x00FF0000
		for y := 0; y < fb.Height; y++ {
			fb.BlendPixel(x, y, redCol)
			fb.BlendPixel(fb.Width-1-x, y, redCol)
		}
	}
}
