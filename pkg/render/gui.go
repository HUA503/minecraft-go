package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// GUISlotType identifies the region/container of a clicked slot.
type GUISlotType int

const (
	SlotNone          GUISlotType = iota
	SlotInventory                 // Player inventory: 0..8 Hotbar, 9..35 Main Storage
	SlotCraftGrid                 // Crafting grid: 0..3 (2x2) or 0..8 (3x3)
	SlotCraftResult               // Crafting output result slot
	SlotFurnaceInput              // Furnace smelting input slot (0)
	SlotFurnaceFuel               // Furnace fuel slot (1)
	SlotFurnaceOutput             // Furnace smelted output slot (2)
	SlotCreativeItem
	SlotCreativeTab
	SlotCreativeScroll
	SlotCreativeSearch
	SlotCreativeTrash
	SlotCreativeArmor
	SlotOffhand
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

// CalculateGUIScaleFromDimensions calculates the authentic Minecraft GUI scale factor for arbitrary dimensions.
func CalculateGUIScaleFromDimensions(w, h int) int {
	sW := w / 320
	sH := h / 240
	scale := sW
	if sH < scale {
		scale = sH
	}
	if scale < 1 {
		scale = 1
	}
	return scale
}

// HitTestCraftingGUI calculates which slot the mouse coordinates land on in the crafting/inventory GUI.
func HitTestCraftingGUI(w, h, mouseX, mouseY int, is3x3 bool) GUISlotHit {
	scale := CalculateGUIScaleFromDimensions(w, h)
	gw := 176 * scale
	gh := 166 * scale
	gx := w/2 - gw/2
	gy := h/2 - gh/2

	slotSize := 18 * scale

	// 1. Check Crafting Grid
	if is3x3 {
		gridStartX := gx + 30*scale
		gridStartY := gy + 17*scale
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				sx := gridStartX + col*slotSize
				sy := gridStartY + row*slotSize
				if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
					return GUISlotHit{Type: SlotCraftGrid, Index: row*3 + col}
				}
			}
		}
		// 3x3 Result slot
		resX := gx + 124*scale
		resY := gy + 35*scale
		resSize := slotSize
		if mouseX >= resX && mouseX < resX+resSize && mouseY >= resY && mouseY < resY+resSize {
			return GUISlotHit{Type: SlotCraftResult, Index: 0}
		}
	} else {
		for visualRow := 0; visualRow < 4; visualRow++ {
			sx, sy := gx+8*scale, gy+(8+visualRow*18)*scale
			if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
				return GUISlotHit{Type: SlotCreativeArmor, Index: 3 - visualRow}
			}
		}
		sx, sy := gx+77*scale, gy+62*scale
		if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
			return GUISlotHit{Type: SlotOffhand, Index: 0}
		}
		gridStartX := gx + 98*scale
		gridStartY := gy + 18*scale
		for row := 0; row < 2; row++ {
			for col := 0; col < 2; col++ {
				sx := gridStartX + col*slotSize
				sy := gridStartY + row*slotSize
				if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
					return GUISlotHit{Type: SlotCraftGrid, Index: row*2 + col}
				}
			}
		}
		// 2x2 Result slot
		resX := gx + 154*scale
		resY := gy + 36*scale
		resSize := slotSize
		if mouseX >= resX && mouseX < resX+resSize && mouseY >= resY && mouseY < resY+resSize {
			return GUISlotHit{Type: SlotCraftResult, Index: 0}
		}
	}

	// 2. Main Storage Inventory (Slots 9..35)
	invStartX := gx + 8*scale
	invStartY := gy + 84*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx := invStartX + col*slotSize
			sy := invStartY + row*slotSize
			if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}

	// 3. Hotbar (Slots 0..8)
	hotbarY := gy + 142*scale
	for col := 0; col < 9; col++ {
		sx := invStartX + col*slotSize
		sy := hotbarY
		if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}

	return GUISlotHit{Type: SlotNone, Index: -1}
}

func drawContainerBackground(fb *Framebuffer, texture string, x, y, scale int) bool {
	image, ok := GetOfficialImage(texture)
	if !ok || image.Width < 176 || image.Height < 166 {
		return false
	}
	drawGUIImageRegion(fb, image, x, y, scale, 176, 166)
	return true
}

func drawPlayerEquipmentSlots(fb *Framebuffer, inv *inventory.PlayerInventory, x, y, scale, mouseX, mouseY int) {
	slot, ok := GetOfficialImage("gui/sprites/container/slot.png")
	if !ok || slot.Width != 18 || slot.Height != 18 {
		return
	}
	for row := 0; row < 4; row++ {
		sx, sy := x+8*scale, y+(8+row*18)*scale
		fb.DrawRawImageScaled(sx, sy, scale, slot)
		if inv != nil {
			item := inv.Armor[3-row]
			if !item.IsEmpty() {
				fb.DrawItemIconScaled(sx+scale, sy+scale, item.ItemID, scale)
				if item.Count > 1 {
					fb.DrawNumberScaled(sx+17*scale, sy+10*scale, item.Count, 0xFFFFFFFF, scale)
				}
			}
		}
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0x40FFFFFF)
		}
	}
	sx, sy := x+77*scale, y+62*scale
	fb.DrawRawImageScaled(sx, sy, scale, slot)
	if inv != nil && !inv.Offhand.IsEmpty() {
		fb.DrawItemIconScaled(sx+scale, sy+scale, inv.Offhand.ItemID, scale)
		if inv.Offhand.Count > 1 {
			fb.DrawNumberScaled(sx+17*scale, sy+10*scale, inv.Offhand.Count, 0xFFFFFFFF, scale)
		}
	}
	if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
		fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0x40FFFFFF)
	}
}

func drawGUIImageRegion(fb *Framebuffer, image *RawImage, x, y, scale, width, height int) {
	if fb == nil || image == nil || scale <= 0 || width <= 0 || height <= 0 || image.Width <= 0 || image.Height <= 0 {
		return
	}
	if width > image.Width {
		width = image.Width
	}
	if height > image.Height {
		height = image.Height
	}
	drawCreativeTextureRegion(fb, x, y, scale, image, width, height)
}

func drawGUIProgressSprite(fb *Framebuffer, texture string, x, y, scale, cropWidth int) {
	image, ok := GetOfficialImage(texture)
	if !ok || image.Width <= 0 || image.Height <= 0 || scale <= 0 {
		return
	}
	if cropWidth < 0 {
		cropWidth = 0
	}
	if cropWidth > image.Width {
		cropWidth = image.Width
	}
	for iy := 0; iy < image.Height; iy++ {
		for ix := 0; ix < cropWidth; ix++ {
			color := image.Pixels[iy*image.Width+ix]
			if color>>24 != 0 {
				fb.DrawRect(x+ix*scale, y+iy*scale, scale, scale, color)
			}
		}
	}
}

// HitTestFurnaceGUI calculates which slot the mouse coordinates land on in the furnace GUI.
func HitTestFurnaceGUI(w, h, mouseX, mouseY int) GUISlotHit {
	scale := CalculateGUIScaleFromDimensions(w, h)
	gw := 176 * scale
	gh := 166 * scale
	gx := w/2 - gw/2
	gy := h/2 - gh/2

	slotSize := 18 * scale

	// 1. Furnace Slots
	inX := gx + 56*scale
	inY := gy + 17*scale
	if mouseX >= inX && mouseX < inX+slotSize && mouseY >= inY && mouseY < inY+slotSize {
		return GUISlotHit{Type: SlotFurnaceInput, Index: 0}
	}

	fuelX := gx + 56*scale
	fuelY := gy + 53*scale
	if mouseX >= fuelX && mouseX < fuelX+slotSize && mouseY >= fuelY && mouseY < fuelY+slotSize {
		return GUISlotHit{Type: SlotFurnaceFuel, Index: 1}
	}

	outX := gx + 116*scale
	outY := gy + 35*scale
	outSize := 18 * scale
	if mouseX >= outX && mouseX < outX+outSize && mouseY >= outY && mouseY < outY+outSize {
		return GUISlotHit{Type: SlotFurnaceOutput, Index: 2}
	}

	// 2. Main Storage Inventory (Slots 9..35)
	invStartX := gx + 8*scale
	invStartY := gy + 84*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx := invStartX + col*slotSize
			sy := invStartY + row*slotSize
			if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}

	// 3. Hotbar (Slots 0..8)
	hotbarY := gy + 142*scale
	for col := 0; col < 9; col++ {
		sx := invStartX + col*slotSize
		sy := hotbarY
		if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
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

	scale := fb.CalculateGUIScale()

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

	// 2. Centered dialog box (176x166 * scale)
	gw := 176 * scale
	gh := 166 * scale
	gx := fb.Width/2 - gw/2
	gy := fb.Height/2 - gh/2

	slotSize := 18 * scale

	texture := "gui/container/crafting_table.png"
	if !is3x3 {
		texture = "gui/container/inventory.png"
	}
	if !drawContainerBackground(fb, texture, gx, gy, scale) {
		fb.DrawRect(gx, gy, gw, gh, 0xFFC6C6C6)
	}
	if !is3x3 {
		fb.DrawRect(gx+8*scale, gy+8*scale, 70*scale, 72*scale, 0xFFC6C6C6)
		drawCreativePlayerPreview(fb, inv, gx+52*scale, gy+12*scale, scale, mouseX, mouseY)
		drawPlayerEquipmentSlots(fb, inv, gx, gy, scale, mouseX, mouseY)
	}

	// Title
	if is3x3 {
		fb.DrawText5x7(gx+8*scale, gy+6*scale, "工作台", 0xFF404040, scale)
	} else {
		fb.DrawText5x7(gx+97*scale, gy+6*scale, "合成", 0xFF404040, scale)
	}

	// Crafting section
	if is3x3 {
		gridStartX := gx + 30*scale
		gridStartY := gy + 17*scale
		for row := 0; row < 3; row++ {
			for col := 0; col < 3; col++ {
				idx := row*3 + col
				sx := gridStartX + col*slotSize
				sy := gridStartY + row*slotSize
				if idx < len(craftGrid) && !craftGrid[idx].IsEmpty() {
					fb.DrawItemIconScaled(sx+scale, sy+scale, craftGrid[idx].ItemID, scale)
					if craftGrid[idx].Count > 1 {
						fb.DrawNumberScaled(sx+17*scale, sy+10*scale, craftGrid[idx].Count, 0xFFFFFFFF, scale)
					}
				}
				if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
					fb.DrawRect(sx+scale, sy+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
				}
			}
		}
		// 3x3 Result slot
		resX := gx + 124*scale
		resY := gy + 35*scale
		resSize := 18 * scale
		if !craftResult.IsEmpty() {
			fb.DrawItemIconScaled(resX+scale, resY+scale, craftResult.ItemID, scale)
			if craftResult.Count > 1 {
				fb.DrawNumberScaled(resX+17*scale, resY+10*scale, craftResult.Count, 0xFFFFFFFF, scale)
			}
		}
		if mouseX >= resX && mouseX < resX+resSize && mouseY >= resY && mouseY < resY+resSize {
			fb.DrawRect(resX+scale, resY+scale, resSize-2*scale, resSize-2*scale, 0x40FFFFFF)
		}
	} else {
		gridStartX := gx + 98*scale
		gridStartY := gy + 18*scale
		for row := 0; row < 2; row++ {
			for col := 0; col < 2; col++ {
				idx := row*2 + col
				sx := gridStartX + col*slotSize
				sy := gridStartY + row*slotSize
				if idx < len(craftGrid) && !craftGrid[idx].IsEmpty() {
					fb.DrawItemIconScaled(sx+scale, sy+scale, craftGrid[idx].ItemID, scale)
					if craftGrid[idx].Count > 1 {
						fb.DrawNumberScaled(sx+17*scale, sy+10*scale, craftGrid[idx].Count, 0xFFFFFFFF, scale)
					}
				}
				if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
					fb.DrawRect(sx+scale, sy+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
				}
			}
		}
		// Result slot
		resX := gx + 154*scale
		resY := gy + 36*scale
		resSize := 18 * scale
		if !craftResult.IsEmpty() {
			fb.DrawItemIconScaled(resX+scale, resY+scale, craftResult.ItemID, scale)
			if craftResult.Count > 1 {
				fb.DrawNumberScaled(resX+17*scale, resY+10*scale, craftResult.Count, 0xFFFFFFFF, scale)
			}
		}
		if mouseX >= resX && mouseX < resX+resSize && mouseY >= resY && mouseY < resY+resSize {
			fb.DrawRect(resX+scale, resY+scale, resSize-2*scale, resSize-2*scale, 0x40FFFFFF)
		}
	}

	// Inventory Label
	fb.DrawText5x7(gx+8*scale, gy+72*scale, "物品栏", 0xFF404040, scale)

	// Main Storage Inventory (Slots 9..35)
	invStartX := gx + 8*scale
	invStartY := gy + 84*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			slotIdx := 9 + row*9 + col
			sx := invStartX + col*slotSize
			sy := invStartY + row*slotSize

			item := inv.Slots[slotIdx]
			if !item.IsEmpty() {
				fb.DrawItemIconScaled(sx+scale, sy+scale, item.ItemID, scale)
				if item.Count > 1 {
					fb.DrawNumberScaled(sx+17*scale, sy+10*scale, item.Count, 0xFFFFFFFF, scale)
				}
				props, isTool := inventory.GetToolProperties(item.ItemID)
				if isTool && item.Damage > 0 && props.MaxDurability > 0 {
					durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
					if durRatio < 0 {
						durRatio = 0
					}
					barW := int(durRatio * float32(14*scale))
					fb.DrawRect(sx+2*scale, sy+15*scale, 14*scale, 2*scale, 0xFF000000)
					barCol := uint32(0xFF00FF00)
					if durRatio < 0.25 {
						barCol = 0xFFFF0000
					} else if durRatio < 0.50 {
						barCol = 0xFFFFFF00
					}
					fb.DrawRect(sx+2*scale, sy+15*scale, barW, scale, barCol)
				}
			}
			if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
				fb.DrawRect(sx+scale, sy+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
			}
		}
	}

	// Hotbar (Slots 0..8)
	hotbarY := gy + 142*scale
	for col := 0; col < 9; col++ {
		sx := invStartX + col*slotSize
		sy := hotbarY

		item := inv.Slots[col]
		if !item.IsEmpty() {
			fb.DrawItemIconScaled(sx+scale, sy+scale, item.ItemID, scale)
			if item.Count > 1 {
				fb.DrawNumberScaled(sx+17*scale, sy+10*scale, item.Count, 0xFFFFFFFF, scale)
			}
			props, isTool := inventory.GetToolProperties(item.ItemID)
			if isTool && item.Damage > 0 && props.MaxDurability > 0 {
				durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
				if durRatio < 0 {
					durRatio = 0
				}
				barW := int(durRatio * float32(14*scale))
				fb.DrawRect(sx+2*scale, sy+15*scale, 14*scale, 2*scale, 0xFF000000)
				barCol := uint32(0xFF00FF00)
				if durRatio < 0.25 {
					barCol = 0xFFFF0000
				} else if durRatio < 0.50 {
					barCol = 0xFFFFFF00
				}
				fb.DrawRect(sx+2*scale, sy+15*scale, barW, scale, barCol)
			}
		}
		if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
			fb.DrawRect(sx+scale, sy+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
		}
	}

	// Floating Cursor Item
	if !cursorItem.IsEmpty() && mouseX >= 0 && mouseY >= 0 {
		cx := mouseX - 8*scale
		cy := mouseY - 8*scale
		fb.DrawItemIconScaled(cx, cy, cursorItem.ItemID, scale)
		if cursorItem.Count > 1 {
			fb.DrawNumberScaled(cx+16*scale, cy+10*scale, cursorItem.Count, 0xFFFFFFFF, scale)
		}
	}
}

// DrawFurnaceGUI renders authentic Minecraft furnace smelting dialog with flame and progress animations.
func (fb *Framebuffer) DrawFurnaceGUI(furnace *inventory.FurnaceBlockEntity, inv *inventory.PlayerInventory, cursorItem inventory.ItemStack, mouseX, mouseY int) {
	if inv == nil {
		return
	}

	scale := fb.CalculateGUIScale()

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

	// 2. Centered dialog box (176x166 * scale)
	gw := 176 * scale
	gh := 166 * scale
	gx := fb.Width/2 - gw/2
	gy := fb.Height/2 - gh/2

	slotSize := 18 * scale

	if !drawContainerBackground(fb, "gui/container/furnace.png", gx, gy, scale) {
		fb.DrawRect(gx, gy, gw, gh, 0xFFC6C6C6)
	}

	// Title
	fb.DrawText5x7(gx+8*scale, gy+6*scale, "熔炉", 0xFF404040, scale)

	// Smelting Input Slot
	inX := gx + 56*scale
	inY := gy + 17*scale
	if furnace != nil && !furnace.InputSlot.IsEmpty() {
		fb.DrawItemIconScaled(inX+scale, inY+scale, furnace.InputSlot.ItemID, scale)
		if furnace.InputSlot.Count > 1 {
			fb.DrawNumberScaled(inX+17*scale, inY+10*scale, furnace.InputSlot.Count, 0xFFFFFFFF, scale)
		}
	}
	if mouseX >= inX && mouseX < inX+slotSize && mouseY >= inY && mouseY < inY+slotSize {
		fb.DrawRect(inX+scale, inY+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
	}

	// Burning Flame Indicator
	flameX := gx + 56*scale
	flameY := gy + 37*scale
	if furnace != nil && furnace.IsBurning() {
		drawGUIProgressSprite(fb, "gui/sprites/container/furnace/lit_progress.png", flameX, flameY, scale, 14)
	}

	// Fuel Slot
	fuelX := gx + 56*scale
	fuelY := gy + 53*scale
	if furnace != nil && !furnace.FuelSlot.IsEmpty() {
		fb.DrawItemIconScaled(fuelX+scale, fuelY+scale, furnace.FuelSlot.ItemID, scale)
		if furnace.FuelSlot.Count > 1 {
			fb.DrawNumberScaled(fuelX+17*scale, fuelY+10*scale, furnace.FuelSlot.Count, 0xFFFFFFFF, scale)
		}
	}
	if mouseX >= fuelX && mouseX < fuelX+slotSize && mouseY >= fuelY && mouseY < fuelY+slotSize {
		fb.DrawRect(fuelX+scale, fuelY+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
	}

	// Cook Progress Arrow
	arrowX := gx + 79*scale
	arrowY := gy + 35*scale
	if furnace != nil && furnace.CookTimeTotal > 0 && furnace.CookTimeProgress > 0 {
		cookRatio := float32(furnace.CookTimeProgress) / float32(furnace.CookTimeTotal)
		if cookRatio > 1.0 {
			cookRatio = 1.0
		}
		progW := int(cookRatio * 24)
		if progW > 0 {
			drawGUIProgressSprite(fb, "gui/sprites/container/furnace/burn_progress.png", arrowX, arrowY, scale, progW)
		}
	}

	// Output Slot
	outX := gx + 116*scale
	outY := gy + 35*scale
	outSize := 18 * scale
	if furnace != nil && !furnace.OutputSlot.IsEmpty() {
		fb.DrawItemIconScaled(outX+scale, outY+scale, furnace.OutputSlot.ItemID, scale)
		if furnace.OutputSlot.Count > 1 {
			fb.DrawNumberScaled(outX+17*scale, outY+10*scale, furnace.OutputSlot.Count, 0xFFFFFFFF, scale)
		}
	}
	if mouseX >= outX && mouseX < outX+outSize && mouseY >= outY && mouseY < outY+outSize {
		fb.DrawRect(outX+scale, outY+scale, outSize-2*scale, outSize-2*scale, 0x40FFFFFF)
	}

	// Inventory Label
	fb.DrawText5x7(gx+8*scale, gy+72*scale, "物品栏", 0xFF404040, scale)

	// Main Storage Inventory (Slots 9..35)
	invStartX := gx + 8*scale
	invStartY := gy + 84*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			slotIdx := 9 + row*9 + col
			sx := invStartX + col*slotSize
			sy := invStartY + row*slotSize

			item := inv.Slots[slotIdx]
			if !item.IsEmpty() {
				fb.DrawItemIconScaled(sx+scale, sy+scale, item.ItemID, scale)
				if item.Count > 1 {
					fb.DrawNumberScaled(sx+17*scale, sy+10*scale, item.Count, 0xFFFFFFFF, scale)
				}
				props, isTool := inventory.GetToolProperties(item.ItemID)
				if isTool && item.Damage > 0 && props.MaxDurability > 0 {
					durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
					if durRatio < 0 {
						durRatio = 0
					}
					barW := int(durRatio * float32(14*scale))
					fb.DrawRect(sx+2*scale, sy+15*scale, 14*scale, 2*scale, 0xFF000000)
					barCol := uint32(0xFF00FF00)
					if durRatio < 0.25 {
						barCol = 0xFFFF0000
					} else if durRatio < 0.50 {
						barCol = 0xFFFFFF00
					}
					fb.DrawRect(sx+2*scale, sy+15*scale, barW, scale, barCol)
				}
			}
			if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
				fb.DrawRect(sx+scale, sy+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
			}
		}
	}

	// Hotbar (Slots 0..8)
	hotbarY := gy + 142*scale
	for col := 0; col < 9; col++ {
		sx := invStartX + col*slotSize
		sy := hotbarY

		item := inv.Slots[col]
		if !item.IsEmpty() {
			fb.DrawItemIconScaled(sx+scale, sy+scale, item.ItemID, scale)
			if item.Count > 1 {
				fb.DrawNumberScaled(sx+17*scale, sy+10*scale, item.Count, 0xFFFFFFFF, scale)
			}
			props, isTool := inventory.GetToolProperties(item.ItemID)
			if isTool && item.Damage > 0 && props.MaxDurability > 0 {
				durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
				if durRatio < 0 {
					durRatio = 0
				}
				barW := int(durRatio * float32(14*scale))
				fb.DrawRect(sx+2*scale, sy+15*scale, 14*scale, 2*scale, 0xFF000000)
				barCol := uint32(0xFF00FF00)
				if durRatio < 0.25 {
					barCol = 0xFFFF0000
				} else if durRatio < 0.50 {
					barCol = 0xFFFFFF00
				}
				fb.DrawRect(sx+2*scale, sy+15*scale, barW, scale, barCol)
			}
		}
		if mouseX >= sx && mouseX < sx+slotSize && mouseY >= sy && mouseY < sy+slotSize {
			fb.DrawRect(sx+scale, sy+scale, slotSize-2*scale, slotSize-2*scale, 0x40FFFFFF)
		}
	}

	// Floating Cursor Item
	if !cursorItem.IsEmpty() && mouseX >= 0 && mouseY >= 0 {
		cx := mouseX - 8*scale
		cy := mouseY - 8*scale
		fb.DrawItemIconScaled(cx, cy, cursorItem.ItemID, scale)
		if cursorItem.Count > 1 {
			fb.DrawNumberScaled(cx+16*scale, cy+10*scale, cursorItem.Count, 0xFFFFFFFF, scale)
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

// DrawAdvancementScreen renders the player's advancement overview.
func (fb *Framebuffer) DrawAdvancementScreen(lines []string) {
	fb.DrawRect(0, 0, fb.Width, fb.Height, 0xB0000000)
	w := min(560, fb.Width-24)
	h := min(420, fb.Height-24)
	if w < 1 || h < 1 {
		return
	}
	x := (fb.Width - w) / 2
	y := (fb.Height - h) / 2
	fb.DrawRect(x, y, w, h, 0xFF282828)
	fb.DrawRect(x+3, y+3, w-6, h-6, 0xFF3C3C3C)
	fb.DrawText5x7(x+16, y+14, "进度", 0xFFFFFFFF, 2)
	fb.DrawText5x7(x+16, y+46, "世界目标", 0xFFBFBFBF, 1)
	rowY := y + 72
	maxRows := max(0, (h-92)/20)
	for i := 0; i < len(lines) && i < maxRows; i++ {
		fb.DrawRect(x+12, rowY-3, w-24, 18, 0xFF303030)
		textColor := uint32(0xFFAAAAAA)
		if strings.HasPrefix(lines[i], "✓") {
			textColor = 0xFFFFFF55
		}
		fb.DrawText5x7(x+20, rowY, lines[i], textColor, 1)
		rowY += 20
	}
	fb.DrawText5x7(x+16, y+h-20, "按 L 或 Esc 关闭", 0xFFAAAAAA, 1)
}

// DrawWorldSelectScreen renders the world selection and management dialog.
func (fb *Framebuffer) DrawWorldSelectScreen(worlds []WorldInfo, selectedIdx int, mouseX, mouseY int) (playBtn, newBtn, backBtn [4]int, worldRects [][4]int) {
	return fb.DrawWorldSelectScreenWithLoading(worlds, selectedIdx, false, mouseX, mouseY)
}

// DrawWorldSelectScreenWithLoading keeps slow save-directory scans off the UI
// thread while showing a distinct state from an actually empty save folder.
func (fb *Framebuffer) DrawWorldSelectScreenWithLoading(worlds []WorldInfo, selectedIdx int, loading bool, mouseX, mouseY int) (playBtn, newBtn, backBtn [4]int, worldRects [][4]int) {
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
		if loading {
			emptyMsg = "正在读取存档..."
		}
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
