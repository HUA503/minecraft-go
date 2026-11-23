package render

import "github.com/HUA503/minecraft-go/pkg/inventory"

const (
	creativeColumns = 9
	creativeRows    = 3
	creativePerPage = creativeColumns * creativeRows
)

var creativeTabLabels = [...]string{"全部", "方块", "工具", "材料", "食物"}

// DrawCreativeInventoryGUI draws a paged item catalog above the player's
// inventory. The player inventory remains interactive so items can be moved
// from the catalog to the hotbar using the usual cursor clicks.
func (fb *Framebuffer) DrawCreativeInventoryGUI(inv *inventory.PlayerInventory, cursor inventory.ItemStack, category inventory.CreativeCategory, page, mouseX, mouseY int) {
	if fb == nil || inv == nil {
		return
	}
	scale := CalculateGUIScaleFromDimensions(fb.Width, fb.Height)
	panelW, panelH := 230*scale, 210*scale
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
	fb.DrawText5x7(px+8*scale, py+5*scale, "创造模式物品栏", 0xFF404040, scale)

	for i, label := range creativeTabLabels {
		x := px + (7+i*43)*scale
		col := uint32(0xFF555555)
		if int(category) == i {
			col = 0xFF202020
		}
		fb.DrawRect(x, py+17*scale, 41*scale, 15*scale, 0xFF8B8B8B)
		fb.DrawText5x7(x+2*scale, py+21*scale, label, col, scale)
	}

	items := inventory.CreativeCatalog(category)
	maxPage := max(0, (len(items)-1)/creativePerPage)
	if page < 0 {
		page = 0
	}
	if page > maxPage {
		page = maxPage
	}
	start := page * creativePerPage
	for i := 0; i < creativePerPage && start+i < len(items); i++ {
		row, col := i/creativeColumns, i%creativeColumns
		sx := px + (7+col*24)*scale
		sy := py + (37+row*20)*scale
		fb.DrawRect(sx, sy, 18*scale, 18*scale, 0xFF373737)
		fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0xFF8B8B8B)
		fb.DrawItemIconScaled(sx+scale, sy+scale, items[start+i].ID, scale)
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0x50FFFFFF)
		}
	}

	pageText := "第 " + itoa(page+1) + " / " + itoa(maxPage+1) + " 页"
	fb.DrawText5x7(px+88*scale, py+101*scale, pageText, 0xFF404040, scale)
	fb.DrawButton(px+7*scale, py+98*scale, 54*scale, 16*scale, "上一页", page > 0 && mouseX >= px+7*scale && mouseX < px+61*scale && mouseY >= py+98*scale && mouseY < py+114*scale)
	fb.DrawButton(px+169*scale, py+98*scale, 54*scale, 16*scale, "下一页", page < maxPage && mouseX >= px+169*scale && mouseX < px+223*scale && mouseY >= py+98*scale && mouseY < py+114*scale)
	fb.DrawText5x7(px+8*scale, py+119*scale, "物品栏", 0xFF404040, scale)

	invX, invY := px+7*scale, py+132*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			drawCreativePlayerSlot(fb, inv, 9+row*9+col, invX+col*20*scale, invY+row*20*scale, scale, mouseX, mouseY)
		}
	}
	for col := 0; col < 9; col++ {
		drawCreativePlayerSlot(fb, inv, col, invX+col*20*scale, py+191*scale, scale, mouseX, mouseY)
	}
	if !cursor.IsEmpty() {
		fb.DrawItemIconScaled(mouseX-8*scale, mouseY-8*scale, cursor.ItemID, scale)
		if cursor.Count > 1 {
			fb.DrawNumberScaled(mouseX+8*scale, mouseY+3*scale, cursor.Count, 0xFFFFFFFF, scale)
		}
	}
}

func drawCreativePlayerSlot(fb *Framebuffer, inv *inventory.PlayerInventory, index, x, y, scale, mouseX, mouseY int) {
	fb.DrawRect(x, y, 18*scale, 18*scale, 0xFF373737)
	fb.DrawRect(x+scale, y+scale, 16*scale, 16*scale, 0xFF8B8B8B)
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

// HitTestCreativeInventory maps mouse coordinates to a tab, catalog item,
// page button, or player inventory slot.
func HitTestCreativeInventory(w, h int, category inventory.CreativeCategory, page, mouseX, mouseY int) GUISlotHit {
	scale := CalculateGUIScaleFromDimensions(w, h)
	px, py := w/2-230*scale/2, h/2-210*scale/2
	for i := range creativeTabLabels {
		x := px + (7+i*43)*scale
		if mouseX >= x && mouseX < x+41*scale && mouseY >= py+17*scale && mouseY < py+32*scale {
			return GUISlotHit{Type: SlotCreativeTab, Index: i}
		}
	}
	if mouseX >= px+7*scale && mouseX < px+61*scale && mouseY >= py+98*scale && mouseY < py+114*scale {
		return GUISlotHit{Type: SlotCreativePrevious}
	}
	if mouseX >= px+169*scale && mouseX < px+223*scale && mouseY >= py+98*scale && mouseY < py+114*scale {
		return GUISlotHit{Type: SlotCreativeNext}
	}
	items := inventory.CreativeCatalog(category)
	for i := 0; i < creativePerPage; i++ {
		row, col := i/creativeColumns, i%creativeColumns
		sx, sy := px+(7+col*24)*scale, py+(37+row*20)*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale && page*creativePerPage+i < len(items) {
			return GUISlotHit{Type: SlotCreativeItem, Index: page*creativePerPage + i}
		}
	}
	invX, invY := px+7*scale, py+132*scale
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx, sy := invX+col*20*scale, invY+row*20*scale
			if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}
	for col := 0; col < 9; col++ {
		sx, sy := invX+col*20*scale, py+191*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}
	return GUISlotHit{Type: SlotNone, Index: -1}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}

