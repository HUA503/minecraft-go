package render

import "github.com/HUA503/minecraft-go/pkg/inventory"

const (
	creativeColumns = 9
	creativeRows    = 5
	creativePerPage = creativeColumns * creativeRows
)

// Icon order follows the compact tab strip used by the Java creative screen.
var creativeTabIcons = [...]int32{345, 1, 35, 6, 54, 331, 270, 268, 260, 296, 383}
var creativeTabNames = [...]string{"Search Items", "Building Blocks", "Colored Blocks", "Natural Blocks", "Functional Blocks", "Redstone Blocks", "Tools & Utilities", "Combat", "Food & Drinks", "Ingredients", "Spawn Eggs"}

// DrawCreativeInventoryGUI draws a paged item catalog above the player's
// inventory. The player inventory remains interactive so items can be moved
// from the catalog to the hotbar using the usual cursor clicks.
func (fb *Framebuffer) DrawCreativeInventoryGUI(inv *inventory.PlayerInventory, cursor inventory.ItemStack, category inventory.CreativeCategory, query string, page, mouseX, mouseY int) {
	if fb == nil || inv == nil {
		return
	}
	scale := CalculateGUIScaleFromDimensions(fb.Width, fb.Height)
	panelW, panelH := 204*scale, 234*scale
	px, py := fb.Width/2-panelW/2, fb.Height/2-panelH/2
	for i, c := range fb.Color {
		r, g, b := (c>>16)&0xFF, (c>>8)&0xFF, c&0xFF
		fb.Color[i] = 0xFF000000 | ((r / 3) << 16) | ((g / 3) << 8) | (b / 3)
	}
	fb.DrawRect(px, py, panelW, panelH, 0xFF373737)
	fb.DrawRect(px+scale, py+scale, panelW-2*scale, panelH-2*scale, 0xFFC6C6C6)
	fb.DrawRect(px+scale, py+scale, panelW-2*scale, scale, 0xFFFFFFFF)
	fb.DrawRect(px+scale, py+scale, scale, panelH-2*scale, 0xFFFFFFFF)
	fb.DrawRect(px+panelW-2*scale, py+scale, scale, panelH-2*scale, 0xFF555555)
	fb.DrawRect(px+scale, py+panelH-2*scale, panelW-2*scale, scale, 0xFF555555)
	for i := range creativeTabIcons {
		if int(category) == i {
			continue
		}
		x := px + (2+i*18)*scale
		variant := "4"
		if i == 0 {
			variant = "1"
		} else if i == len(creativeTabIcons)-1 {
			variant = "7"
		}
		if tab, ok := GetOfficialImage("gui/sprites/container/creative_inventory/tab_top_unselected_" + variant + ".png"); ok {
			fb.DrawRawImageScaled(x, py-2*scale, scale, tab)
		}
	}
	selected := int(category)
	if selected >= 0 && selected < len(creativeTabIcons) {
		variant := "4"
		if selected == 0 {
			variant = "1"
		} else if selected == len(creativeTabIcons)-1 {
			variant = "7"
		}
		x := px + (2+selected*18)*scale
		if tab, ok := GetOfficialImage("gui/sprites/container/creative_inventory/tab_top_selected_" + variant + ".png"); ok {
			fb.DrawRawImageScaled(x, py-2*scale, scale, tab)
		}
	}
	for i, icon := range creativeTabIcons {
		x := px + (7+i*18)*scale
		fb.DrawItemIconScaled(x, py+6*scale, icon, scale)
	}

	items := inventory.SearchCreativeCatalog(category, query)
	maxScroll := max(0, (len(items)-creativePerPage+creativeColumns-1)/creativeColumns)
	if page < 0 {
		page = 0
	}
	if page > maxScroll {
		page = maxScroll
	}
	start := page * creativeColumns
	hoveredName := ""
	for i := 0; i < creativePerPage && start+i < len(items); i++ {
		row, col := i/creativeColumns, i%creativeColumns
		sx := px + (7+col*18)*scale
		sy := py + (27+row*18)*scale
		drawOfficialCreativeSlot(fb, sx, sy, scale)
		fb.DrawItemIconScaled(sx+scale, sy+scale, items[start+i].ID, scale)
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0x50FFFFFF)
			hoveredName = items[start+i].Name
		}
	}
	drawCreativeTooltip(fb, hoveredName, mouseX, mouseY, scale)
	trackX, trackY, trackH := px+174*scale, py+27*scale, 90*scale
	fb.DrawRect(trackX+4*scale, trackY, 4*scale, trackH, 0xFF777777)
	thumbY := trackY
	if maxScroll > 0 {
		thumbY += (trackH - 15*scale) * page / maxScroll
	}
	if thumb, ok := GetOfficialImage("gui/sprites/container/creative_inventory/scroller.png"); ok {
		fb.DrawRawImageScaled(trackX, thumbY, scale, thumb)
	} else {
		fb.DrawRect(trackX, thumbY, 12*scale, 15*scale, 0xFFAAAAAA)
	}

	searchX, searchY := px+7*scale, py+120*scale
	fb.DrawRect(searchX, searchY, 186*scale, 16*scale, 0xFF555555)
	fb.DrawRect(searchX+scale, searchY+scale, 184*scale, 14*scale, 0xFFFFFFFF)
	searchLabel := "Search Items..."
	if query != "" {
		searchLabel = query
	}
	fb.DrawText5x7(searchX+3*scale, searchY+5*scale, searchLabel, 0xFF404040, scale)

	invX, invY := px+7*scale, py+144*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			drawCreativePlayerSlot(fb, inv, 9+row*9+col, invX+col*18*scale, invY+row*18*scale, scale, mouseX, mouseY)
		}
	}
	for col := 0; col < 9; col++ {
		drawCreativePlayerSlot(fb, inv, col, invX+col*18*scale, py+208*scale, scale, mouseX, mouseY)
	}
	for i := range creativeTabIcons {
		x, y := px+(2+i*18)*scale, py+3*scale
		if mouseX >= x && mouseX < x+18*scale && mouseY >= y && mouseY < y+22*scale {
			drawCreativeTooltip(fb, creativeTabNames[i], mouseX, mouseY, scale)
			break
		}
	}
	if !cursor.IsEmpty() {
		fb.DrawItemIconScaled(mouseX-8*scale, mouseY-8*scale, cursor.ItemID, scale)
		if cursor.Count > 1 {
			fb.DrawNumberScaled(mouseX+8*scale, mouseY+3*scale, cursor.Count, 0xFFFFFFFF, scale)
		}
	}
}

func drawCreativePlayerSlot(fb *Framebuffer, inv *inventory.PlayerInventory, index, x, y, scale, mouseX, mouseY int) {
	drawOfficialCreativeSlot(fb, x, y, scale)
	item := inv.Slots[index]
	if !item.IsEmpty() {
		fb.DrawItemIconScaled(x+scale, y+scale, item.ItemID, scale)
		if item.Count > 1 {
			fb.DrawNumberScaled(x+11*scale, y+11*scale, item.Count, 0xFFFFFFFF, scale)
		}
	}
	if index == inv.SelectedHotbar {
		fb.DrawRect(x, y, 18*scale, scale, 0xFFFFFFFF)
		fb.DrawRect(x, y+17*scale, 18*scale, scale, 0xFFFFFFFF)
		fb.DrawRect(x, y, scale, 18*scale, 0xFFFFFFFF)
		fb.DrawRect(x+17*scale, y, scale, 18*scale, 0xFFFFFFFF)
	}
	if mouseX >= x && mouseX < x+18*scale && mouseY >= y && mouseY < y+18*scale {
		fb.DrawRect(x+scale, y+scale, 16*scale, 16*scale, 0x50FFFFFF)
	}
}

func drawOfficialCreativeSlot(fb *Framebuffer, x, y, scale int) {
	if slot, ok := GetOfficialImage("gui/sprites/container/slot.png"); ok && slot.Width == 18 && slot.Height == 18 {
		fb.DrawRawImageScaled(x, y, scale, slot)
		return
	}
	fb.DrawRect(x, y, 18*scale, 18*scale, 0xFF373737)
	fb.DrawRect(x+scale, y+scale, 16*scale, 16*scale, 0xFF8B8B8B)
}

func drawCreativeTooltip(fb *Framebuffer, label string, mouseX, mouseY, scale int) {
	if label == "" {
		return
	}
	width := len([]rune(label))*6*scale + 8*scale
	x, y := mouseX+10*scale, mouseY+10*scale
	if x+width > fb.Width {
		x = mouseX - width - 8*scale
	}
	if y+13*scale > fb.Height {
		y = mouseY - 18*scale
	}
	fb.DrawRect(x, y, width, 13*scale, 0xF0100010)
	fb.DrawRect(x+scale, y+scale, width-2*scale, 11*scale, 0xF0100010)
	fb.DrawText5x7(x+4*scale, y+3*scale, label, 0xFFFFFFFF, scale)
}

// HitTestCreativeInventory maps mouse coordinates to a tab, catalog item,
// page button, or player inventory slot.
func HitTestCreativeInventory(w, h int, category inventory.CreativeCategory, query string, page, mouseX, mouseY int) GUISlotHit {
	scale := CalculateGUIScaleFromDimensions(w, h)
	px, py := w/2-204*scale/2, h/2-234*scale/2
	if mouseX >= px+7*scale && mouseX < px+193*scale && mouseY >= py+120*scale && mouseY < py+136*scale {
		return GUISlotHit{Type: SlotCreativeSearch}
	}
	for i := range creativeTabIcons {
		x := px + (2+i*18)*scale
		if mouseX >= x && mouseX < x+18*scale && mouseY >= py+3*scale && mouseY < py+25*scale {
			return GUISlotHit{Type: SlotCreativeTab, Index: i}
		}
	}
	trackX, trackY, trackH := px+174*scale, py+27*scale, 90*scale
	if mouseX >= trackX && mouseX < trackX+12*scale && mouseY >= trackY && mouseY < trackY+trackH {
		return GUISlotHit{Type: SlotCreativeScroll, Index: (mouseY - trackY) * 100 / max(1, trackH-1)}
	}
	items := inventory.SearchCreativeCatalog(category, query)
	maxScroll := max(0, (len(items)-creativePerPage+creativeColumns-1)/creativeColumns)
	if page < 0 {
		page = 0
	}
	if page > maxScroll {
		page = maxScroll
	}
	start := page * creativeColumns
	for i := 0; i < creativePerPage; i++ {
		row, col := i/creativeColumns, i%creativeColumns
		sx, sy := px+(7+col*18)*scale, py+(27+row*18)*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale && start+i < len(items) {
			return GUISlotHit{Type: SlotCreativeItem, Index: start + i}
		}
	}
	invX, invY := px+7*scale, py+144*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx, sy := invX+col*18*scale, invY+row*18*scale
			if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}
	for col := 0; col < 9; col++ {
		sx, sy := invX+col*18*scale, py+208*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}
	return GUISlotHit{Type: SlotNone, Index: -1}
}
