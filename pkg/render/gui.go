package render

import (
	"fmt"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// GUISlotType identifies the region/container of a clicked slot.
type GUISlotType int

const (
	SlotNone GUISlotType = iota
	SlotInventory        // Player inventory: 0..8 Hotbar, 9..35 Main Storage
	SlotCraftGrid        // Crafting grid: 0..3 (2x2) or 0..8 (3x3)
	SlotCraftResult      // Crafting output result slot
	SlotFurnaceInput     // Furnace smelting input slot (0)
	SlotFurnaceFuel      // Furnace fuel slot (1)
	SlotFurnaceOutput    // Furnace smelted output slot (2)
)

// GUISlotHit records the detected slot hit during mouse interactions.
type GUISlotHit struct {
	Type  GUISlotType
	Index int
}

// WorldInfo holds metadata of a saved world for the world selection menu.
type WorldInfo struct {
	DirName   string
	LevelName string
	GameMode  int
	LastTime  time.Time
}

// HitTestCraftingGUI calculates which slot the mouse coordinates land on in the crafting/inventory GUI.
func HitTestCraftingGUI(w, h, mouseX, mouseY int, is3x3 bool) GUISlotHit {
	gw := 280
	gh := 220
	gx := w/2 - gw/2
	gy := h/2 - gh/2

	// 1. Check Crafting Grid
	if is3x3 {
		gridStartX := gx + 30
		gridStartY := gy + 20
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				sx := gridStartX + col*20
				sy := gridStartY + row*20
				if mouseX >= sx && mouseX < sx+18 && mouseY >= sy && mouseY < sy+18 {
					return GUISlotHit{Type: SlotCraftGrid, Index: row*3 + col}
				}
			}
		}
		// 3x3 Result slot
		resX := gridStartX + 68 + 28
		resY := gridStartY + 18
		if mouseX >= resX && mouseX < resX+24 && mouseY >= resY && mouseY < resY+24 {
			return GUISlotHit{Type: SlotCraftResult, Index: 0}
		}
	} else {
		gridStartX := gx + 40
		gridStartY := gy + 24
		for row := 0; row < 2; row++ {
			for col := 0; col < 2; col++ {
				sx := gridStartX + col*22
				sy := gridStartY + row*22
				if mouseX >= sx && mouseX < sx+20 && mouseY >= sy && mouseY < sy+20 {
					return GUISlotHit{Type: SlotCraftGrid, Index: row*2 + col}
				}
			}
		}
		// 2x2 Result slot
		resX := gridStartX + 48 + 32
		resY := gridStartY + 10
		if mouseX >= resX && mouseX < resX+24 && mouseY >= resY && mouseY < resY+24 {
			return GUISlotHit{Type: SlotCraftResult, Index: 0}
		}
	}

	// 2. Main Storage Inventory (Slots 9..35)
	invStartX := gx + 18
	invStartY := gy + 86
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx := invStartX + col*27
			sy := invStartY + row*24
			if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}

	// 3. Hotbar (Slots 0..8)
	hotbarY := gy + 164
	for col := 0; col < 9; col++ {
		sx := invStartX + col*27
		sy := hotbarY
		if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}

	return GUISlotHit{Type: SlotNone, Index: -1}
}

// HitTestFurnaceGUI calculates which slot the mouse coordinates land on in the furnace GUI.
func HitTestFurnaceGUI(w, h, mouseX, mouseY int) GUISlotHit {
	gw := 280
	gh := 220
	gx := w/2 - gw/2
	gy := h/2 - gh/2

	// 1. Furnace Slots
	inX := gx + 80
	inY := gy + 20
	if mouseX >= inX && mouseX < inX+22 && mouseY >= inY && mouseY < inY+22 {
		return GUISlotHit{Type: SlotFurnaceInput, Index: 0}
	}

	fuelX := gx + 80
	fuelY := gy + 62
	if mouseX >= fuelX && mouseX < fuelX+22 && mouseY >= fuelY && mouseY < fuelY+22 {
		return GUISlotHit{Type: SlotFurnaceFuel, Index: 1}
	}

	outX := gx + 144
	outY := gy + 36
	if mouseX >= outX && mouseX < outX+26 && mouseY >= outY && mouseY < outY+26 {
		return GUISlotHit{Type: SlotFurnaceOutput, Index: 2}
	}

	// 2. Main Storage Inventory (Slots 9..35)
	invStartX := gx + 18
	invStartY := gy + 86
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx := invStartX + col*27
			sy := invStartY + row*24
			if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}

	// 3. Hotbar (Slots 0..8)
	hotbarY := gy + 164
	for col := 0; col < 9; col++ {
		sx := invStartX + col*27
		sy := hotbarY
		if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}

	return GUISlotHit{Type: SlotNone, Index: -1}
}

// DrawCraftingGUI renders authentic Minecraft 2x2 player inventory or 3x3 crafting table GUI.
func (fb *Framebuffer) DrawCraftingGUI(inv *inventory.PlayerInventory, craftGrid []inventory.ItemStack, craftResult inventory.ItemStack, is3x3 bool, cursorItem inventory.ItemStack, mouseX, mouseY int) {
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

	// 2. Centered dialog box (280x220)
	gw := 280
	gh := 220
	gx := fb.Width/2 - gw/2
	gy := fb.Height/2 - gh/2

	// Dialog outer border and background
	fb.DrawRect(gx, gy, gw, gh, 0xFF373737)
	fb.DrawRect(gx+2, gy+2, gw-4, gh-4, 0xFFC6C6C6)
	fb.DrawRect(gx+2, gy+2, gw-4, 2, 0xFFFFFFFF)
	fb.DrawRect(gx+2, gy+2, 2, gh-4, 0xFFFFFFFF)
	fb.DrawRect(gx+gw-4, gy+2, 2, gh-4, 0xFF555555)
	fb.DrawRect(gx+2, gy+gh-4, gw-4, 2, 0xFF555555)

	// Title
	if is3x3 {
		fb.DrawText5x7(gx+18, gy+8, "工作台", 0xFF404040, 1)
	} else {
		fb.DrawText5x7(gx+18, gy+8, "合成", 0xFF404040, 1)
	}

	// Crafting section
	if is3x3 {
		gridStartX := gx + 30
		gridStartY := gy + 20
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				idx := row*3 + col
				sx := gridStartX + col*20
				sy := gridStartY + row*20
				fb.DrawRect(sx, sy, 18, 18, 0xFF373737)
				fb.DrawRect(sx+1, sy+1, 16, 16, 0xFF8B8B8B)

				if idx < len(craftGrid) && !craftGrid[idx].IsEmpty() {
					fb.DrawItemIcon(sx+1, sy+1, craftGrid[idx].ItemID)
					if craftGrid[idx].Count > 1 {
						fb.DrawNumber(sx+15, sy+11, craftGrid[idx].Count, 0xFFFFFFFF)
					}
				}
				if mouseX >= sx && mouseX < sx+18 && mouseY >= sy && mouseY < sy+18 {
					fb.DrawRect(sx+1, sy+1, 16, 16, 0x40FFFFFF)
				}
			}
		}
		// Arrow
		arrowX := gridStartX + 68 + 6
		arrowY := gridStartY + 24
		fb.DrawRect(arrowX, arrowY+3, 14, 3, 0xFF555555)
		fb.DrawRect(arrowX+10, arrowY, 3, 9, 0xFF555555)
		fb.DrawRect(arrowX+12, arrowY+1, 2, 7, 0xFF555555)

		// 3x3 Result slot
		resX := gridStartX + 68 + 28
		resY := gridStartY + 18
		fb.DrawRect(resX, resY, 24, 24, 0xFF373737)
		fb.DrawRect(resX+1, resY+1, 22, 22, 0xFF8B8B8B)
		if !craftResult.IsEmpty() {
			fb.DrawItemIcon(resX+4, resY+4, craftResult.ItemID)
			if craftResult.Count > 1 {
				fb.DrawNumber(resX+20, resY+15, craftResult.Count, 0xFFFFFFFF)
			}
		}
		if mouseX >= resX && mouseX < resX+24 && mouseY >= resY && mouseY < resY+24 {
			fb.DrawRect(resX+1, resY+1, 22, 22, 0x40FFFFFF)
		}
	} else {
		gridStartX := gx + 40
		gridStartY := gy + 24
		for row := 0; row < 2; row++ {
			for col := 0; col < 2; col++ {
				idx := row*2 + col
				sx := gridStartX + col*22
				sy := gridStartY + row*22
				fb.DrawRect(sx, sy, 20, 20, 0xFF373737)
				fb.DrawRect(sx+1, sy+1, 18, 18, 0xFF8B8B8B)

				if idx < len(craftGrid) && !craftGrid[idx].IsEmpty() {
					fb.DrawItemIcon(sx+2, sy+2, craftGrid[idx].ItemID)
					if craftGrid[idx].Count > 1 {
						fb.DrawNumber(sx+17, sy+13, craftGrid[idx].Count, 0xFFFFFFFF)
					}
				}
				if mouseX >= sx && mouseX < sx+20 && mouseY >= sy && mouseY < sy+20 {
					fb.DrawRect(sx+1, sy+1, 18, 18, 0x40FFFFFF)
				}
			}
		}
		// Arrow
		arrowX := gridStartX + 48 + 10
		arrowY := gridStartY + 16
		fb.DrawRect(arrowX, arrowY+3, 14, 3, 0xFF555555)
		fb.DrawRect(arrowX+10, arrowY, 3, 9, 0xFF555555)
		fb.DrawRect(arrowX+12, arrowY+1, 2, 7, 0xFF555555)

		// Result slot
		resX := gridStartX + 48 + 32
		resY := gridStartY + 10
		fb.DrawRect(resX, resY, 24, 24, 0xFF373737)
		fb.DrawRect(resX+1, resY+1, 22, 22, 0xFF8B8B8B)
		if !craftResult.IsEmpty() {
			fb.DrawItemIcon(resX+4, resY+4, craftResult.ItemID)
			if craftResult.Count > 1 {
				fb.DrawNumber(resX+20, resY+15, craftResult.Count, 0xFFFFFFFF)
			}
		}
		if mouseX >= resX && mouseX < resX+24 && mouseY >= resY && mouseY < resY+24 {
			fb.DrawRect(resX+1, resY+1, 22, 22, 0x40FFFFFF)
		}
	}

	// Inventory Label
	fb.DrawText5x7(gx+18, gy+74, "物品栏", 0xFF404040, 1)

	// Main Storage Inventory (Slots 9..35)
	invStartX := gx + 18
	invStartY := gy + 86
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			slotIdx := 9 + row*9 + col
			sx := invStartX + col*27
			sy := invStartY + row*24

			fb.DrawRect(sx, sy, 22, 22, 0xFF373737)
			fb.DrawRect(sx+1, sy+1, 20, 20, 0xFF8B8B8B)

			item := inv.Slots[slotIdx]
			if !item.IsEmpty() {
				fb.DrawItemIcon(sx+3, sy+3, item.ItemID)
				if item.Count > 1 {
					fb.DrawNumber(sx+19, sy+14, item.Count, 0xFFFFFFFF)
				}
				props, isTool := inventory.GetToolProperties(item.ItemID)
				if isTool && item.Damage > 0 && props.MaxDurability > 0 {
					durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
					if durRatio < 0 {
						durRatio = 0
					}
					barW := int(durRatio * 16.0)
					fb.DrawRect(sx+3, sy+18, 16, 2, 0xFF000000)
					barCol := uint32(0xFF00FF00)
					if durRatio < 0.25 {
						barCol = 0xFFFF0000
					} else if durRatio < 0.50 {
						barCol = 0xFFFFFF00
					}
					fb.DrawRect(sx+3, sy+18, barW, 1, barCol)
				}
			}
			if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
				fb.DrawRect(sx+1, sy+1, 20, 20, 0x40FFFFFF)
			}
		}
	}

	// Hotbar (Slots 0..8)
	hotbarY := gy + 164
	for col := 0; col < 9; col++ {
		sx := invStartX + col*27
		sy := hotbarY

		fb.DrawRect(sx, sy, 22, 22, 0xFF373737)
		fb.DrawRect(sx+1, sy+1, 20, 20, 0xFF8B8B8B)

		item := inv.Slots[col]
		if !item.IsEmpty() {
			fb.DrawItemIcon(sx+3, sy+3, item.ItemID)
			if item.Count > 1 {
				fb.DrawNumber(sx+19, sy+14, item.Count, 0xFFFFFFFF)
			}
			props, isTool := inventory.GetToolProperties(item.ItemID)
			if isTool && item.Damage > 0 && props.MaxDurability > 0 {
				durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
				if durRatio < 0 {
					durRatio = 0
				}
				barW := int(durRatio * 16.0)
				fb.DrawRect(sx+3, sy+18, 16, 2, 0xFF000000)
				barCol := uint32(0xFF00FF00)
				if durRatio < 0.25 {
					barCol = 0xFFFF0000
				} else if durRatio < 0.50 {
					barCol = 0xFFFFFF00
				}
				fb.DrawRect(sx+3, sy+18, barW, 1, barCol)
			}
		}
		if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
			fb.DrawRect(sx+1, sy+1, 20, 20, 0x40FFFFFF)
		}
	}

	// Floating Cursor Item
	if !cursorItem.IsEmpty() && mouseX >= 0 && mouseY >= 0 {
		cx := mouseX - 8
		cy := mouseY - 8
		fb.DrawItemIcon(cx, cy, cursorItem.ItemID)
		if cursorItem.Count > 1 {
			fb.DrawNumber(cx+16, cy+11, cursorItem.Count, 0xFFFFFFFF)
		}
	}
}

// DrawFurnaceGUI renders authentic Minecraft furnace smelting dialog with flame and progress animations.
func (fb *Framebuffer) DrawFurnaceGUI(furnace *inventory.FurnaceBlockEntity, inv *inventory.PlayerInventory, cursorItem inventory.ItemStack, mouseX, mouseY int) {
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

	// 2. Centered dialog box (280x220)
	gw := 280
	gh := 220
	gx := fb.Width/2 - gw/2
	gy := fb.Height/2 - gh/2

	// Dialog outer border and background
	fb.DrawRect(gx, gy, gw, gh, 0xFF373737)
	fb.DrawRect(gx+2, gy+2, gw-4, gh-4, 0xFFC6C6C6)
	fb.DrawRect(gx+2, gy+2, gw-4, 2, 0xFFFFFFFF)
	fb.DrawRect(gx+2, gy+2, 2, gh-4, 0xFFFFFFFF)
	fb.DrawRect(gx+gw-4, gy+2, 2, gh-4, 0xFF555555)
	fb.DrawRect(gx+2, gy+gh-4, gw-4, 2, 0xFF555555)

	// Title
	fb.DrawText5x7(gx+18, gy+8, "熔炉", 0xFF404040, 1)

	// Smelting Input Slot
	inX := gx + 80
	inY := gy + 20
	fb.DrawRect(inX, inY, 22, 22, 0xFF373737)
	fb.DrawRect(inX+1, inY+1, 20, 20, 0xFF8B8B8B)
	if furnace != nil && !furnace.InputSlot.IsEmpty() {
		fb.DrawItemIcon(inX+3, inY+3, furnace.InputSlot.ItemID)
		if furnace.InputSlot.Count > 1 {
			fb.DrawNumber(inX+19, inY+14, furnace.InputSlot.Count, 0xFFFFFFFF)
		}
	}
	if mouseX >= inX && mouseX < inX+22 && mouseY >= inY && mouseY < inY+22 {
		fb.DrawRect(inX+1, inY+1, 20, 20, 0x40FFFFFF)
	}

	// Burning Flame Indicator
	flameX := gx + 84
	flameY := gy + 45
	flameH := 14
	fb.DrawRect(flameX, flameY, 14, flameH, 0xFF555555)
	if furnace != nil && furnace.IsBurning() && furnace.MaxBurnTime > 0 {
		burnRatio := float32(furnace.BurnTimeRemaining) / float32(furnace.MaxBurnTime)
		if burnRatio > 1.0 {
			burnRatio = 1.0
		}
		activeH := int(burnRatio * float32(flameH))
		if activeH > 0 {
			fb.DrawRect(flameX, flameY+flameH-activeH, 14, activeH, 0xFFFF6A00) // Vibrant fire orange
			fb.DrawRect(flameX+3, flameY+flameH-activeH, 8, activeH, 0xFFFFD800) // Yellow inner flame
		}
	}

	// Fuel Slot
	fuelX := gx + 80
	fuelY := gy + 62
	fb.DrawRect(fuelX, fuelY, 22, 22, 0xFF373737)
	fb.DrawRect(fuelX+1, fuelY+1, 20, 20, 0xFF8B8B8B)
	if furnace != nil && !furnace.FuelSlot.IsEmpty() {
		fb.DrawItemIcon(fuelX+3, fuelY+3, furnace.FuelSlot.ItemID)
		if furnace.FuelSlot.Count > 1 {
			fb.DrawNumber(fuelX+19, fuelY+14, furnace.FuelSlot.Count, 0xFFFFFFFF)
		}
	}
	if mouseX >= fuelX && mouseX < fuelX+22 && mouseY >= fuelY && mouseY < fuelY+22 {
		fb.DrawRect(fuelX+1, fuelY+1, 20, 20, 0x40FFFFFF)
	}

	// Cook Progress Arrow
	arrowX := gx + 110
	arrowY := gy + 40
	arrowW := 22
	fb.DrawRect(arrowX, arrowY+3, arrowW, 6, 0xFF555555)
	fb.DrawRect(arrowX+arrowW-6, arrowY, 6, 12, 0xFF555555)
	if furnace != nil && furnace.CookTimeTotal > 0 && furnace.CookTimeProgress > 0 {
		cookRatio := float32(furnace.CookTimeProgress) / float32(furnace.CookTimeTotal)
		if cookRatio > 1.0 {
			cookRatio = 1.0
		}
		progW := int(cookRatio * float32(arrowW))
		if progW > 0 {
			fb.DrawRect(arrowX, arrowY+3, progW, 6, 0xFFFFFFFF)
			if progW >= arrowW-6 {
				fb.DrawRect(arrowX+arrowW-6, arrowY, 6, 12, 0xFFFFFFFF)
			}
		}
	}

	// Output Slot
	outX := gx + 144
	outY := gy + 36
	fb.DrawRect(outX, outY, 26, 26, 0xFF373737)
	fb.DrawRect(outX+1, outY+1, 24, 24, 0xFF8B8B8B)
	if furnace != nil && !furnace.OutputSlot.IsEmpty() {
		fb.DrawItemIcon(outX+5, outY+5, furnace.OutputSlot.ItemID)
		if furnace.OutputSlot.Count > 1 {
			fb.DrawNumber(outX+22, outY+17, furnace.OutputSlot.Count, 0xFFFFFFFF)
		}
	}
	if mouseX >= outX && mouseX < outX+26 && mouseY >= outY && mouseY < outY+26 {
		fb.DrawRect(outX+1, outY+1, 24, 24, 0x40FFFFFF)
	}

	// Inventory Label
	fb.DrawText5x7(gx+18, gy+74, "物品栏", 0xFF404040, 1)

	// Main Storage Inventory (Slots 9..35)
	invStartX := gx + 18
	invStartY := gy + 86
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			slotIdx := 9 + row*9 + col
			sx := invStartX + col*27
			sy := invStartY + row*24

			fb.DrawRect(sx, sy, 22, 22, 0xFF373737)
			fb.DrawRect(sx+1, sy+1, 20, 20, 0xFF8B8B8B)

			item := inv.Slots[slotIdx]
			if !item.IsEmpty() {
				fb.DrawItemIcon(sx+3, sy+3, item.ItemID)
				if item.Count > 1 {
					fb.DrawNumber(sx+19, sy+14, item.Count, 0xFFFFFFFF)
				}
				props, isTool := inventory.GetToolProperties(item.ItemID)
				if isTool && item.Damage > 0 && props.MaxDurability > 0 {
					durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
					if durRatio < 0 {
						durRatio = 0
					}
					barW := int(durRatio * 16.0)
					fb.DrawRect(sx+3, sy+18, 16, 2, 0xFF000000)
					barCol := uint32(0xFF00FF00)
					if durRatio < 0.25 {
						barCol = 0xFFFF0000
					} else if durRatio < 0.50 {
						barCol = 0xFFFFFF00
					}
					fb.DrawRect(sx+3, sy+18, barW, 1, barCol)
				}
			}
			if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
				fb.DrawRect(sx+1, sy+1, 20, 20, 0x40FFFFFF)
			}
		}
	}

	// Hotbar (Slots 0..8)
	hotbarY := gy + 164
	for col := 0; col < 9; col++ {
		sx := invStartX + col*27
		sy := hotbarY

		fb.DrawRect(sx, sy, 22, 22, 0xFF373737)
		fb.DrawRect(sx+1, sy+1, 20, 20, 0xFF8B8B8B)

		item := inv.Slots[col]
		if !item.IsEmpty() {
			fb.DrawItemIcon(sx+3, sy+3, item.ItemID)
			if item.Count > 1 {
				fb.DrawNumber(sx+19, sy+14, item.Count, 0xFFFFFFFF)
			}
			props, isTool := inventory.GetToolProperties(item.ItemID)
			if isTool && item.Damage > 0 && props.MaxDurability > 0 {
				durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
				if durRatio < 0 {
					durRatio = 0
				}
				barW := int(durRatio * 16.0)
				fb.DrawRect(sx+3, sy+18, 16, 2, 0xFF000000)
				barCol := uint32(0xFF00FF00)
				if durRatio < 0.25 {
					barCol = 0xFFFF0000
				} else if durRatio < 0.50 {
					barCol = 0xFFFFFF00
				}
				fb.DrawRect(sx+3, sy+18, barW, 1, barCol)
			}
		}
		if mouseX >= sx && mouseX < sx+22 && mouseY >= sy && mouseY < sy+22 {
			fb.DrawRect(sx+1, sy+1, 20, 20, 0x40FFFFFF)
		}
	}

	// Floating Cursor Item
	if !cursorItem.IsEmpty() && mouseX >= 0 && mouseY >= 0 {
		cx := mouseX - 8
		cy := mouseY - 8
		fb.DrawItemIcon(cx, cy, cursorItem.ItemID)
		if cursorItem.Count > 1 {
			fb.DrawNumber(cx+16, cy+11, cursorItem.Count, 0xFFFFFFFF)
		}
	}
}

// DrawTitleScreen renders the authentic Minecraft main menu.
func (fb *Framebuffer) DrawTitleScreen(mouseX, mouseY int) (spBtn, mpBtn, quitBtn [4]int) {
	// Background dirt pattern
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			// Dark textured dirt vignette
			p := ((x ^ y) & 7) * 4
			r := uint32(35 + p)
			g := uint32(25 + p)
			b := uint32(18 + p)
			fb.Color[y*fb.Width+x] = 0xFF000000 | (r << 16) | (g << 8) | b
		}
	}

	// Title Banner: "MINECRAFT"
	title := "MINECRAFT"
	tw := MeasureText5x7(title, 3)
	tx := (fb.Width - tw) / 2
	ty := fb.Height/4 - 30
	fb.DrawText5x7(tx, ty, title, 0xFFE0E0E0, 3)

	sub := "Go Edition 1.0 (Pure Go)"
	sw := MeasureText5x7(sub, 1)
	fb.DrawText5x7((fb.Width-sw)/2, ty+28, sub, 0xFFFFFF55, 1)

	// Buttons
	btnW := 220
	btnH := 26
	btnX := (fb.Width - btnW) / 2

	// 1. Singleplayer (单人游戏)
	spY := fb.Height/2 - 15
	hoverSP := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= spY && mouseY <= spY+btnH
	fb.DrawButton(btnX, spY, btnW, btnH, "单人游戏", hoverSP)
	spBtn = [4]int{btnX, spY, btnW, btnH}

	// 2. Multiplayer (多人联机)
	mpY := spY + 34
	hoverMP := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= mpY && mouseY <= mpY+btnH
	fb.DrawButton(btnX, mpY, btnW, btnH, "多人联机", hoverMP)
	mpBtn = [4]int{btnX, mpY, btnW, btnH}

	// 3. Quit Game (退出游戏)
	qY := mpY + 34
	hoverQ := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= qY && mouseY <= qY+btnH
	fb.DrawButton(btnX, qY, btnW, btnH, "退出游戏", hoverQ)
	quitBtn = [4]int{btnX, qY, btnW, btnH}

	// Bottom copyright
	copyRight := "Minecraft Go Clone - Pure Go"
	fb.DrawText5x7(6, fb.Height-12, copyRight, 0xFF888888, 1)

	return spBtn, mpBtn, quitBtn
}

// DrawWorldSelectScreen renders the world selection and management dialog.
func (fb *Framebuffer) DrawWorldSelectScreen(worlds []WorldInfo, selectedIdx int, mouseX, mouseY int) (playBtn, newBtn, backBtn [4]int, worldRects [][4]int) {
	// Darkened background
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			p := ((x ^ y) & 7) * 3
			fb.Color[y*fb.Width+x] = 0xFF000000 | (uint32(25+p) << 16) | (uint32(25+p) << 8) | uint32(25+p)
		}
	}

	// Title
	title := "选择世界"
	tw := MeasureText5x7(title, 2)
	fb.DrawText5x7((fb.Width-tw)/2, 16, title, 0xFFFFFFFF, 2)

	// World List Area
	listX := fb.Width/2 - 160
	listY := 45
	listW := 320
	listH := fb.Height - 105
	fb.DrawRect(listX, listY, listW, listH, 0xFF181818)

	worldRects = make([][4]int, len(worlds))
	if len(worlds) == 0 {
		emptyMsg := "没有找到存档，请新建世界！"
		ew := MeasureText5x7(emptyMsg, 1)
		fb.DrawText5x7((fb.Width-ew)/2, listY+listH/2-4, emptyMsg, 0xFFAAAAAA, 1)
	} else {
		for i, wInfo := range worlds {
			wy := listY + 6 + i*38
			if wy+34 > listY+listH {
				break
			}
			rect := [4]int{listX + 4, wy, listW - 8, 34}
			worldRects[i] = rect

			isSelected := i == selectedIdx
			isHovered := mouseX >= rect[0] && mouseX <= rect[0]+rect[2] && mouseY >= rect[1] && mouseY <= rect[1]+rect[3]

			bgCol := uint32(0xFF242424)
			if isSelected {
				bgCol = 0xFF3E4F66
			} else if isHovered {
				bgCol = 0xFF303030
			}
			fb.DrawRect(rect[0], rect[1], rect[2], rect[3], bgCol)
			if isSelected {
				fb.DrawRect(rect[0], rect[1], rect[2], 1, 0xFFFFFFFF)
				fb.DrawRect(rect[0], rect[1]+rect[3]-1, rect[2], 1, 0xFFFFFFFF)
				fb.DrawRect(rect[0], rect[1], 1, rect[3], 0xFFFFFFFF)
				fb.DrawRect(rect[0]+rect[2]-1, rect[1], 1, rect[3], 0xFFFFFFFF)
			}

			// World name
			nameDisp := wInfo.LevelName
			if nameDisp == "" {
				nameDisp = wInfo.DirName
			}
			fb.DrawText5x7(rect[0]+8, rect[1]+6, nameDisp, 0xFFFFFFFF, 1)

			// Sub info (Game mode, folder)
			modeStr := "生存模式"
			if wInfo.GameMode == 1 {
				modeStr = "创造模式"
			}
			infoStr := fmt.Sprintf("%s (%s)", modeStr, wInfo.DirName)
			fb.DrawText5x7(rect[0]+8, rect[1]+20, infoStr, 0xFF888888, 1)
		}
	}

	// Bottom action buttons
	btnW := 96
	btnH := 24
	bottomY := fb.Height - 40
	gap := 12
	totalW := btnW*3 + gap*2
	startX := (fb.Width - totalW) / 2

	// 1. Play Selected World (进入世界)
	pX := startX
	hoverP := mouseX >= pX && mouseX <= pX+btnW && mouseY >= bottomY && mouseY <= bottomY+btnH
	fb.DrawButton(pX, bottomY, btnW, btnH, "进入世界", hoverP && selectedIdx >= 0 && selectedIdx < len(worlds))
	playBtn = [4]int{pX, bottomY, btnW, btnH}

	// 2. Create New World (新建世界)
	nX := pX + btnW + gap
	hoverN := mouseX >= nX && mouseX <= nX+btnW && mouseY >= bottomY && mouseY <= bottomY+btnH
	fb.DrawButton(nX, bottomY, btnW, btnH, "新建世界", hoverN)
	newBtn = [4]int{nX, bottomY, btnW, btnH}

	// 3. Back (返回)
	bX := nX + btnW + gap
	hoverB := mouseX >= bX && mouseX <= bX+btnW && mouseY >= bottomY && mouseY <= bottomY+btnH
	fb.DrawButton(bX, bottomY, btnW, btnH, "返回", hoverB)
	backBtn = [4]int{bX, bottomY, btnW, btnH}

	return playBtn, newBtn, backBtn, worldRects
}

// DrawNewWorldScreen renders the world creation screen.
func (fb *Framebuffer) DrawNewWorldScreen(worldName string, gameMode int, seedText string, activeInput int, mouseX, mouseY int) (createBtn, cancelBtn, modeBtn [4]int, nameBox, seedBox [4]int) {
	// Darkened background
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			p := ((x ^ y) & 7) * 3
			fb.Color[y*fb.Width+x] = 0xFF000000 | (uint32(25+p) << 16) | (uint32(25+p) << 8) | uint32(25+p)
		}
	}

	// Title
	title := "新建世界"
	tw := MeasureText5x7(title, 2)
	fb.DrawText5x7((fb.Width-tw)/2, 16, title, 0xFFFFFFFF, 2)

	centerX := fb.Width / 2
	boxW := 220
	boxH := 24
	boxX := centerX - boxW/2

	// 1. World Name input
	fb.DrawText5x7(boxX, 48, "世界名称:", 0xFFAAAAAA, 1)
	nameY := 62
	nameBox = [4]int{boxX, nameY, boxW, boxH}
	fb.DrawRect(boxX, nameY, boxW, boxH, 0xFF000000)
	nameBorder := uint32(0xFFA0A0A0)
	if activeInput == 0 {
		nameBorder = 0xFFFFFFFF
	}
	fb.DrawRect(boxX, nameY, boxW, 1, nameBorder)
	fb.DrawRect(boxX, nameY+boxH-1, boxW, 1, nameBorder)
	fb.DrawRect(boxX, nameY, 1, boxH, nameBorder)
	fb.DrawRect(boxX+boxW-1, nameY, 1, boxH, nameBorder)
	dispName := worldName
	if activeInput == 0 && (time.Now().UnixMilli()/500)%2 == 0 {
		dispName += "_"
	}
	fb.DrawText5x7(boxX+6, nameY+7, dispName, 0xFFFFFFFF, 1)

	// 2. Game Mode Toggle Button
	modeY := 100
	modeText := "游戏模式: 生存模式"
	if gameMode == 1 {
		modeText = "游戏模式: 创造模式"
	}
	hoverMode := mouseX >= boxX && mouseX <= boxX+boxW && mouseY >= modeY && mouseY <= modeY+boxH
	fb.DrawButton(boxX, modeY, boxW, boxH, modeText, hoverMode)
	modeBtn = [4]int{boxX, modeY, boxW, boxH}

	// 3. Seed input
	fb.DrawText5x7(boxX, 138, "世界种子 (留空则随机):", 0xFFAAAAAA, 1)
	seedY := 152
	seedBox = [4]int{boxX, seedY, boxW, boxH}
	fb.DrawRect(boxX, seedY, boxW, boxH, 0xFF000000)
	seedBorder := uint32(0xFFA0A0A0)
	if activeInput == 1 {
		seedBorder = 0xFFFFFFFF
	}
	fb.DrawRect(boxX, seedY, boxW, 1, seedBorder)
	fb.DrawRect(boxX, seedY+boxH-1, boxW, 1, seedBorder)
	fb.DrawRect(boxX, seedY, 1, boxH, seedBorder)
	fb.DrawRect(boxX+boxW-1, seedY, 1, boxH, seedBorder)
	dispSeed := seedText
	if activeInput == 1 && (time.Now().UnixMilli()/500)%2 == 0 {
		dispSeed += "_"
	}
	fb.DrawText5x7(boxX+6, seedY+7, dispSeed, 0xFFFFFFFF, 1)

	// Bottom action buttons
	btnW := 105
	btnH := 24
	bottomY := fb.Height - 38
	gap := 10
	totalW := btnW*2 + gap
	startX := centerX - totalW/2

	// Create Button
	cX := startX
	hoverC := mouseX >= cX && mouseX <= cX+btnW && mouseY >= bottomY && mouseY <= bottomY+btnH
	fb.DrawButton(cX, bottomY, btnW, btnH, "创建世界", hoverC)
	createBtn = [4]int{cX, bottomY, btnW, btnH}

	// Cancel Button
	canX := cX + btnW + gap
	hoverCan := mouseX >= canX && mouseX <= canX+btnW && mouseY >= bottomY && mouseY <= bottomY+btnH
	fb.DrawButton(canX, bottomY, btnW, btnH, "取消", hoverCan)
	cancelBtn = [4]int{canX, bottomY, btnW, btnH}

	return createBtn, cancelBtn, modeBtn, nameBox, seedBox
}
