package render

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

// DrawHUD renders crosshair, health bar (10 hearts), hunger bar (10 drumsticks), and 9 hotbar slots.
func (fb *Framebuffer) DrawHUD(health float32, hunger int, selectedSlot int) {
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
			fb.DrawRect(hx+1, hudBaseY+1, 6, 6, 0xFFFF1111) // Minecraft Authentic Red (0xAARRGGBB)
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

	// 4. Hotbar (9 slots centered at bottom, Y = Height - 40)
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

		// Selected slot white border highlight
		if i == selectedSlot {
			fb.DrawRect(slotX-1, slotY-1, 20, 2, 0xFFFFFFFF)
			fb.DrawRect(slotX-1, slotY+17, 20, 2, 0xFFFFFFFF)
			fb.DrawRect(slotX-1, slotY-1, 2, 20, 0xFFFFFFFF)
			fb.DrawRect(slotX+17, slotY-1, 2, 20, 0xFFFFFFFF)
		}
	}
}
