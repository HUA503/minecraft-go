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

// DrawRawImage draws a RawImage at screen coordinate (x, y) with alpha transparency.
func (fb *Framebuffer) DrawRawImage(x, y int, img *RawImage) {
	if img == nil {
		return
	}
	for iy := 0; iy < img.Height; iy++ {
		py := y + iy
		if py < 0 || py >= fb.Height {
			continue
		}
		imgRow := iy * img.Width
		fbRow := py * fb.Width
		for ix := 0; ix < img.Width; ix++ {
			px := x + ix
			if px < 0 || px >= fb.Width {
				continue
			}
			col := img.Pixels[imgRow+ix]
			a := (col >> 24) & 0xFF
			if a == 0 {
				continue
			}
			if a == 255 {
				fb.Color[fbRow+px] = col
			} else {
				// Alpha blend over existing framebuffer pixel
				bg := fb.Color[fbRow+px]
				bgR := (bg >> 16) & 0xFF
				bgG := (bg >> 8) & 0xFF
				bgB := bg & 0xFF
				fgR := (col >> 16) & 0xFF
				fgG := (col >> 8) & 0xFF
				fgB := col & 0xFF
				invA := 255 - a
				outR := (fgR*a + bgR*invA) / 255
				outG := (fgG*a + bgG*invA) / 255
				outB := (fgB*a + bgB*invA) / 255
				fb.Color[fbRow+px] = 0xFF000000 | (outR << 16) | (outG << 8) | outB
			}
		}
	}
}

// DrawRawImageScaled draws a RawImage scaled by an integer factor.
func (fb *Framebuffer) DrawRawImageScaled(x, y, scale int, img *RawImage) {
	if img == nil || scale <= 0 {
		return
	}
	if scale == 1 {
		fb.DrawRawImage(x, y, img)
		return
	}
	for iy := 0; iy < img.Height; iy++ {
		imgRow := iy * img.Width
		for ix := 0; ix < img.Width; ix++ {
			col := img.Pixels[imgRow+ix]
			if (col >> 24) == 0 {
				continue
			}
			fb.DrawRect(x+ix*scale, y+iy*scale, scale, scale, col)
		}
	}
}

// DrawCrosshair renders the official Mojang crosshair centered on the screen.
func (fb *Framebuffer) DrawCrosshair() {
	cx := fb.Width / 2
	cy := fb.Height / 2
	if crosshairImg, ok := GetOfficialImage("gui/sprites/hud/crosshair.png"); ok {
		fb.DrawRawImage(cx-crosshairImg.Width/2, cy-crosshairImg.Height/2, crosshairImg)
		return
	}
	// Fallback vector crosshair
	crosshairColor := uint32(0xFFFFFFFF)
	fb.DrawRect(cx-4, cy, 9, 1, crosshairColor)
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

// DrawItemIcon renders an authentic 16x16 pixel-art item sprite or 3D isometric voxel block icon.
func (fb *Framebuffer) DrawItemIcon(x, y int, itemID int32) {
	fb.DrawItemSprite16x16(x, y, itemID)
}

// Authentic 9x9 Minecraft Java Edition Bitmaps for Hearts & Food Icons (Used as primary/fallback sprites)
var (
	HeartContainerBitmap = [9][9]uint32{
		{0, 0xFF000000, 0xFF000000, 0, 0, 0, 0xFF000000, 0xFF000000, 0},
		{0xFF000000, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF000000, 0, 0xFF000000, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF000000},
		{0xFF000000, 0xFF4A4A4A, 0xFFFFFFFF, 0xFF4A4A4A, 0xFF000000, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF000000},
		{0xFF000000, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF000000},
		{0, 0xFF000000, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF000000, 0},
		{0, 0, 0xFF000000, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF000000, 0, 0},
		{0, 0, 0, 0xFF000000, 0xFF353535, 0xFF000000, 0, 0, 0},
		{0, 0, 0, 0, 0xFF000000, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	HeartFullBitmap = [9][9]uint32{
		{0, 0xFF000000, 0xFF000000, 0, 0, 0, 0xFF000000, 0xFF000000, 0},
		{0xFF000000, 0xFFFF2222, 0xFFFF2222, 0xFF000000, 0, 0xFF000000, 0xFFFF2222, 0xFFFF2222, 0xFF000000},
		{0xFF000000, 0xFFFF2222, 0xFFFFFFFF, 0xFFFF2222, 0xFF000000, 0xFFFF2222, 0xFFFF2222, 0xFFFF2222, 0xFF000000},
		{0xFF000000, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFF000000},
		{0, 0xFF000000, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFF000000, 0},
		{0, 0, 0xFF000000, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFF000000, 0, 0},
		{0, 0, 0, 0xFF000000, 0xFFBD1010, 0xFF000000, 0, 0, 0},
		{0, 0, 0, 0, 0xFF000000, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	HeartHalfBitmap = [9][9]uint32{
		{0, 0xFF000000, 0xFF000000, 0, 0, 0, 0xFF000000, 0xFF000000, 0},
		{0xFF000000, 0xFFFF2222, 0xFFFF2222, 0xFF000000, 0, 0xFF000000, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF000000},
		{0xFF000000, 0xFFFF2222, 0xFFFFFFFF, 0xFFFF2222, 0xFF000000, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF4A4A4A, 0xFF000000},
		{0xFF000000, 0xFFBD1010, 0xFFBD1010, 0xFFBD1010, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF000000},
		{0, 0xFF000000, 0xFFBD1010, 0xFFBD1010, 0xFF353535, 0xFF353535, 0xFF353535, 0xFF000000, 0},
		{0, 0, 0xFF000000, 0xFFBD1010, 0xFF353535, 0xFF353535, 0xFF000000, 0, 0},
		{0, 0, 0, 0xFF000000, 0xFF353535, 0xFF000000, 0, 0, 0},
		{0, 0, 0, 0, 0xFF000000, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	FoodEmptyBitmap = [9][9]uint32{
		{0, 0, 0, 0, 0, 0xFF4A3828, 0xFF4A3828, 0, 0xFFFFFFFF},
		{0, 0, 0, 0xFF4A3828, 0xFF352518, 0xFF352518, 0xFF352518, 0xFFFFFFFF, 0xFFC0C0C0},
		{0, 0, 0xFF4A3828, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF4A3828, 0xFF4A3828},
		{0, 0xFF4A3828, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF4A3828, 0},
		{0xFF4A3828, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF4A3828, 0, 0},
		{0xFF4A3828, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF4A3828, 0xFF4A3828, 0, 0, 0},
		{0xFF4A3828, 0xFF352518, 0xFF352518, 0xFF4A3828, 0, 0, 0, 0, 0},
		{0, 0xFF4A3828, 0xFF4A3828, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	FoodFullBitmap = [9][9]uint32{
		{0, 0, 0, 0, 0, 0xFF502810, 0xFF502810, 0, 0xFFFFFFFF},
		{0, 0, 0, 0xFF502810, 0xFFB86020, 0xFFD88030, 0xFFD88030, 0xFFFFFFFF, 0xFFC0C0C0},
		{0, 0, 0xFF502810, 0xFFB86020, 0xFFD88030, 0xFFF0A040, 0xFFD88030, 0xFF502810, 0xFF502810},
		{0, 0xFF502810, 0xFFB86020, 0xFFD88030, 0xFFF0A040, 0xFFD88030, 0xFFB86020, 0xFF502810, 0},
		{0xFF502810, 0xFFB86020, 0xFFD88030, 0xFFD88030, 0xFFB86020, 0xFF804018, 0xFF502810, 0, 0},
		{0xFF502810, 0xFFD88030, 0xFFB86020, 0xFF804018, 0xFF502810, 0xFF502810, 0, 0, 0},
		{0xFF502810, 0xFFB86020, 0xFF804018, 0xFF502810, 0, 0, 0, 0, 0},
		{0, 0xFF502810, 0xFF502810, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	FoodHalfBitmap = [9][9]uint32{
		{0, 0, 0, 0, 0, 0xFF502810, 0xFF502810, 0, 0xFFFFFFFF},
		{0, 0, 0, 0xFF502810, 0xFF352518, 0xFFD88030, 0xFFD88030, 0xFFFFFFFF, 0xFFC0C0C0},
		{0, 0, 0xFF502810, 0xFF352518, 0xFF352518, 0xFFF0A040, 0xFFD88030, 0xFF502810, 0xFF502810},
		{0, 0xFF502810, 0xFF352518, 0xFF352518, 0xFF352518, 0xFFD88030, 0xFFB86020, 0xFF502810, 0},
		{0xFF502810, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF352518, 0xFF804018, 0xFF502810, 0, 0},
		{0xFF502810, 0xFF352518, 0xFF352518, 0xFF804018, 0xFF502810, 0xFF502810, 0, 0, 0},
		{0xFF502810, 0xFFB86020, 0xFF804018, 0xFF502810, 0, 0, 0, 0, 0},
		{0, 0xFF502810, 0xFF502810, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
)

func (fb *Framebuffer) draw9x9Bitmap(x, y int, bmp [9][9]uint32) {
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			col := bmp[r][c]
			if col != 0 {
				px := x + c
				py := y + r
				if px >= 0 && px < fb.Width && py >= 0 && py < fb.Height {
					fb.Color[py*fb.Width+px] = col
				}
			}
		}
	}
}

// DrawHUD renders crosshair, health bar, hunger bar, and 9 empty hotbar slots.
func (fb *Framebuffer) DrawHUD(health float32, hunger int, selectedSlot int) {
	var emptySlots [9]inventory.ItemStack
	fb.DrawFullHUD(0, health, hunger, 0.0, 0, selectedSlot, emptySlots)
}

// DrawHUDWithInventory renders crosshair, health, hunger, 9 hotbar slots with item icons and counts.
func (fb *Framebuffer) DrawHUDWithInventory(health float32, hunger int, selectedSlot int, slots [9]inventory.ItemStack) {
	fb.DrawFullHUD(0, health, hunger, 0.0, 0, selectedSlot, slots)
}

// DrawFullHUD renders crosshair, hotbar, and conditional survival HUD (hearts, food, experience bar, level).
func (fb *Framebuffer) DrawFullHUD(
	gameMode int,
	health float32,
	hunger int,
	expProgress float32,
	level int,
	selectedSlot int,
	slots [9]inventory.ItemStack,
) {
	// 1. Center Crosshair
	fb.DrawCrosshair()

	// 2. Hotbar dimensions & placement
	hotbarW := 182
	hotbarH := 22
	hotbarX := fb.Width/2 - hotbarW/2
	hotbarY := fb.Height - 32

	// Render official hotbar texture (182x22)
	if hotbarImg, ok := GetOfficialImage("gui/sprites/hud/hotbar.png"); ok {
		fb.DrawRawImage(hotbarX, hotbarY, hotbarImg)
	} else {
		fb.DrawRect(hotbarX, hotbarY, hotbarW, hotbarH, 0x88404040)
		for i := 0; i < 9; i++ {
			fb.DrawRect(hotbarX+i*20+2, hotbarY+2, 18, 18, 0xFF222222)
		}
	}

	// Render official selected slot highlight (24x23)
	if selImg, ok := GetOfficialImage("gui/sprites/hud/hotbar_selection.png"); ok {
		fb.DrawRawImage(hotbarX+selectedSlot*20-1, hotbarY-1, selImg)
	} else {
		slotX := hotbarX + selectedSlot*20 + 2
		slotY := hotbarY + 2
		fb.DrawRect(slotX-1, slotY-1, 20, 2, 0xFFFFFFFF)
		fb.DrawRect(slotX-1, slotY+17, 20, 2, 0xFFFFFFFF)
		fb.DrawRect(slotX-1, slotY-1, 2, 20, 0xFFFFFFFF)
		fb.DrawRect(slotX+17, slotY-1, 2, 20, 0xFFFFFFFF)
	}

	// 3. Hotbar Item Icons, Count, and Durability
	for i := 0; i < 9; i++ {
		slotX := hotbarX + i*20 + 3
		slotY := hotbarY + 3
		item := slots[i]
		if !item.IsEmpty() {
			fb.DrawItemIcon(slotX, slotY, item.ItemID)
			// Render count text in lower right corner
			if item.Count > 1 {
				fb.DrawNumber(slotX+15, slotY+11, item.Count, 0xFFFFFFFF)
			}
			// Durability bar for damaged tools
			props, isTool := inventory.GetToolProperties(item.ItemID)
			if isTool && item.Damage > 0 && props.MaxDurability > 0 {
				durRatio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
				if durRatio < 0 {
					durRatio = 0
				}
				barW := int(durRatio * 13.0)
				fb.DrawRect(slotX+1, slotY+13, 14, 2, 0xFF000000)
				barCol := uint32(0xFF00FF00)
				if durRatio < 0.25 {
					barCol = 0xFFFF0000
				} else if durRatio < 0.50 {
					barCol = 0xFFFFFF00
				}
				fb.DrawRect(slotX+1, slotY+13, barW, 1, barCol)
			}
		}
	}

	// In Creative Mode (gameMode == 1), health, hunger, exp bar and air bubbles are NOT drawn per vanilla rules
	if gameMode == 1 {
		return
	}

	// 4. Experience Bar (182x5) between hotbar and health/hunger
	expBarW := 182
	expBarH := 5
	expBarX := hotbarX
	expBarY := hotbarY - 7

	if expBg, ok := GetOfficialImage("gui/sprites/hud/experience_bar_background.png"); ok {
		fb.DrawRawImage(expBarX, expBarY, expBg)
	} else {
		// Pixel-accurate Minecraft Experience Bar Background (182x5)
		fb.DrawRect(expBarX, expBarY, expBarW, expBarH, 0xFF000000)
		fb.DrawRect(expBarX+1, expBarY+1, expBarW-2, expBarH-2, 0xFF3B4036)
		for s := 1; s < 10; s++ {
			nx := expBarX + s*18
			fb.DrawRect(nx, expBarY+1, 1, 3, 0xFF222620)
		}
	}

	// Filled Experience Progress Bar
	if expProgress > 0.0 {
		fillW := int(float32(expBarW) * expProgress)
		if fillW > expBarW {
			fillW = expBarW
		}
		if fillW > 2 {
			fb.DrawRect(expBarX+1, expBarY+1, fillW-2, 1, 0xFFB4FF42) // Top bright lime highlight
			fb.DrawRect(expBarX+1, expBarY+2, fillW-2, 2, 0xFF80FF20) // Vibrant Minecraft lime green
			for s := 1; s < 10; s++ {
				nx := expBarX + s*18
				if nx < expBarX+fillW-1 {
					fb.DrawRect(nx, expBarY+1, 1, 3, 0xFF356610)
				}
			}
		}
	}

	// Experience Level Number (Centered right above experience bar)
	if level > 0 {
		lvlStr := strconv.Itoa(level)
		numW := MeasureText5x7(lvlStr, 1)
		lx := hotbarX + 91 - numW/2
		ly := expBarY - 7
		fb.DrawText5x7(lx, ly, lvlStr, 0xFF80FF20, 1) // Vibrant lime green text with drop shadow
	}

	// 5. Health Bar (aligned with left of hotbar, 10 hearts spaced by 8px)
	heartStartX := hotbarX
	hudBaseY := hotbarY - 18
	heartContainer, hasHeartContainer := GetOfficialImage("gui/sprites/hud/heart/container.png")
	heartFull, hasHeartFull := GetOfficialImage("gui/sprites/hud/heart/full.png")
	heartHalf, hasHeartHalf := GetOfficialImage("gui/sprites/hud/heart/half.png")

	fullHearts := int(health) / 2
	hasHalfHeart := (int(health) % 2) == 1

	for i := 0; i < 10; i++ {
		hx := heartStartX + i*8
		hy := hudBaseY
		if health <= 4.0 {
			// Minecraft Java edition authentic heart shaking at low health
			shake := (i*7 + int(time.Now().UnixMilli()/70)) % 3
			hy += shake - 1
		}
		if hasHeartContainer {
			fb.DrawRawImage(hx, hy, heartContainer)
		} else {
			fb.draw9x9Bitmap(hx, hy, HeartContainerBitmap)
		}
		if i < fullHearts {
			if hasHeartFull {
				fb.DrawRawImage(hx, hy, heartFull)
			} else {
				fb.draw9x9Bitmap(hx, hy, HeartFullBitmap)
			}
		} else if i == fullHearts && hasHalfHeart {
			if hasHeartHalf {
				fb.DrawRawImage(hx, hy, heartHalf)
			} else {
				fb.draw9x9Bitmap(hx, hy, HeartHalfBitmap)
			}
		}
	}

	// 6. Hunger Bar (aligned with right of hotbar, drawn right-to-left)
	hungerStartX := hotbarX + 182 - 9
	foodEmpty, hasFoodEmpty := GetOfficialImage("gui/sprites/hud/food_empty.png")
	foodFull, hasFoodFull := GetOfficialImage("gui/sprites/hud/food_full.png")
	foodHalf, hasFoodHalf := GetOfficialImage("gui/sprites/hud/food_half.png")

	fullFood := hunger / 2
	hasHalfFood := (hunger % 2) == 1

	for i := 0; i < 10; i++ {
		hx := hungerStartX - i*8
		hy := hudBaseY
		if hasFoodEmpty {
			fb.DrawRawImage(hx, hy, foodEmpty)
		} else {
			fb.draw9x9Bitmap(hx, hy, FoodEmptyBitmap)
		}
		if i < fullFood {
			if hasFoodFull {
				fb.DrawRawImage(hx, hy, foodFull)
			} else {
				fb.draw9x9Bitmap(hx, hy, FoodFullBitmap)
			}
		} else if i == fullFood && hasHalfFood {
			if hasFoodHalf {
				fb.DrawRawImage(hx, hy, foodHalf)
			} else {
				fb.draw9x9Bitmap(hx, hy, FoodHalfBitmap)
			}
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

// DrawChar5x7 renders a single 5x7 character scaled with optional drop shadow, or 11x11 CJK glyph.
func (fb *Framebuffer) DrawChar5x7(x, y int, ch rune, col uint32, scale int) int {
	if ch > 0x7F {
		glyph := GetCJKGlyph(ch)
		for row := 0; row < 11; row++ {
			bits := glyph[row]
			for cIdx := 0; cIdx < 11; cIdx++ {
				if (bits & (1 << (10 - cIdx))) != 0 {
					py := y + (row-2)*scale
					if scale == 1 {
						px := x + cIdx
						if px >= 0 && px < fb.Width && py >= 0 && py < fb.Height {
							fb.Color[py*fb.Width+px] = col
						}
					} else {
						fb.DrawRect(x+cIdx*scale, py, scale, scale, col)
					}
				}
			}
		}
		return 12 * scale
	}

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
	width := 0
	for _, ch := range text {
		if ch > 0x7F {
			width += 12 * scale
		} else {
			width += 6 * scale
		}
	}
	return width
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

	// 2. Title: "游戏暂停" (large scale 2)
	title := "游戏暂停"
	titleWidth := MeasureText5x7(title, 2)
	titleX := (fb.Width - titleWidth) / 2
	titleY := fb.Height/4 - 20
	fb.DrawText5x7(titleX, titleY, title, 0xFFFFFFFF, 2)

	// 3. Button 1: "返回游戏"
	btnW := 220
	btnH := 28
	btnX := (fb.Width - btnW) / 2
	btn1Y := fb.Height/2 - 30
	hover1 := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= btn1Y && mouseY <= btn1Y+btnH
	fb.DrawButton(btnX, btn1Y, btnW, btnH, "返回游戏", hover1)
	backBtnRect = [4]int{btnX, btn1Y, btnW, btnH}

	// 4. Button 2: "保存并退出"
	btn2Y := fb.Height/2 + 10
	hover2 := mouseX >= btnX && mouseX <= btnX+btnW && mouseY >= btn2Y && mouseY <= btn2Y+btnH
	fb.DrawButton(btnX, btn2Y, btnW, btnH, "保存并退出", hover2)
	quitBtnRect = [4]int{btnX, btn2Y, btnW, btnH}

	// 5. Bottom Instructions hint
	hint := "按 ESC 返回游戏 - 按 Q 立即退出"
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

// DrawUnderwaterOverlay tints the screen with authentic deep aquatic blue when camera is submerged in water.
func (fb *Framebuffer) DrawUnderwaterOverlay() {
	for i := 0; i < len(fb.Color); i++ {
		c := fb.Color[i]
		r := (c >> 16) & 0xFF
		g := (c >> 8) & 0xFF
		b := c & 0xFF
		// Blend with deep aquatic blue (35% tint)
		nr := (r*65 + 10*35) / 100
		ng := (g*65 + 56*35) / 100
		nb := (b*65 + 112*35) / 100
		fb.Color[i] = 0xFF000000 | (nr << 16) | (ng << 8) | nb
	}
}

// DrawBossBar renders an authentic Minecraft purple boss health bar at the top of the screen.
func (fb *Framebuffer) DrawBossBar(title string, health, maxHealth float32) {
	if health < 0 {
		health = 0
	}
	if maxHealth <= 0 {
		maxHealth = 200.0
	}
	barWidth := 200
	barHeight := 6
	barX := (fb.Width - barWidth) / 2
	barY := 22

	// Render title text centered above boss bar
	titleW := len(title) * 6
	textX := (fb.Width - titleW) / 2
	textY := 10
	fb.DrawText5x7(textX, textY, title, 0xFFFFAAFF, 1)

	// Boss bar outer frame and dark background
	fb.DrawRect(barX-1, barY-1, barWidth+2, barHeight+2, 0xFF1A001A)
	fb.DrawRect(barX, barY, barWidth, barHeight, 0xFF3A0D3A)

	// Filled boss bar (magenta / purple)
	pct := float64(health) / float64(maxHealth)
	if pct > 1.0 {
		pct = 1.0
	}
	fillW := int(float64(barWidth) * pct)
	if fillW > 0 {
		fb.DrawRect(barX, barY, fillW, barHeight, 0xFFC800C8)
		// Highlight strip on top row
		fb.DrawRect(barX, barY, fillW, 2, 0xFFEA4FEA)
	}

	// 10 Notch dividers
	for i := 1; i < 10; i++ {
		notchX := barX + (i * barWidth / 10)
		fb.DrawRect(notchX, barY, 1, barHeight, 0xFF1A001A)
	}
}

// DrawNetherPortalOverlay renders the authentic purple swirling portal vignette when standing in a portal.
func (fb *Framebuffer) DrawNetherPortalOverlay(intensity float32) {
	if intensity <= 0 {
		return
	}
	if intensity > 1.0 {
		intensity = 1.0
	}
	alpha := uint32(float32(140) * intensity)
	portalCol := (alpha << 24) | 0x008A2BE2
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			fb.BlendPixel(x, y, portalCol)
		}
	}
}
