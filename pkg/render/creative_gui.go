package render

import "github.com/HUA503/minecraft-go/pkg/inventory"

const (
	creativeColumns = 9
	creativeRows    = 5
	creativePerPage = creativeColumns * creativeRows
	creativePanelW  = 204
	creativePanelH  = 234
	creativeTabsH   = 26
	creativeSlotsH  = 227
)

// creativeGUIOrigin chooses a scale that leaves room for the creative tabs as
// well as the panel. The player inventory can be nudged up on short windows so
// its hotbar remains visible; drawing and hit testing share this calculation.
func creativeGUIOrigin(w, h int) (scale, px, py, inventoryOffsetY int) {
	scale = min(w/creativePanelW, h/(creativePanelH+creativeTabsH))
	if scale < 1 {
		scale = 1
	}
	px = w/2 - creativePanelW*scale/2
	py = (h-(creativePanelH+creativeTabsH)*scale)/2 + creativeTabsH*scale
	if py < creativeTabsH*scale {
		py = creativeTabsH * scale
	}
	if bottom := py + creativeSlotsH*scale; bottom > h {
		inventoryOffsetY = h - bottom
	}
	return scale, px, py, inventoryOffsetY
}

// Icon order follows the compact tab strip used by the Java creative screen.
// Java 26.3 uses a Creeper spawn egg for the Spawn Eggs tab icon.
var creativeTabIcons = [...]int32{345, 1, 35, 6, 54, 331, 270, 268, 260, 296, 11002}
var creativeTabNames = [...]string{"搜索物品", "建筑方块", "有色方块", "自然方块", "功能方块", "红石方块", "工具与实用物品", "战斗", "食物与饮料", "材料", "刷怪蛋"}

// DrawCreativeInventoryGUI draws a scrollable item catalog above the player's
// inventory. The player inventory remains interactive so items can be moved
// from the catalog to the hotbar using the usual cursor clicks.
func (fb *Framebuffer) DrawCreativeInventoryGUI(inv *inventory.PlayerInventory, cursor inventory.ItemStack, category inventory.CreativeCategory, query string, scrollRow, mouseX, mouseY int) {
	if fb == nil || inv == nil {
		return
	}
	scale, px, py, inventoryOffsetY := creativeGUIOrigin(fb.Width, fb.Height)
	for i, c := range fb.Color {
		r, g, b := (c>>16)&0xFF, (c>>8)&0xFF, c&0xFF
		fb.Color[i] = 0xFF000000 | ((r / 3) << 16) | ((g / 3) << 8) | (b / 3)
	}
	if image, ok := GetOfficialImage("gui/container/inventory.png"); ok {
		drawCreativeTextureRegion(fb, px-scale, py+(61*scale)+inventoryOffsetY, scale, image, 176, 166)
	}
	creativeTexture := "gui/container/creative_inventory/tab_items.png"
	if category == inventory.CreativeAll {
		creativeTexture = "gui/container/creative_inventory/tab_item_search.png"
	}
	if image, ok := GetOfficialImage(creativeTexture); ok {
		drawCreativeTextureRegion(fb, px-scale, py, scale, image, 195, 136)
	}
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
			fb.DrawRawImageScaled(x, py-26*scale, scale, tab)
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
			fb.DrawRawImageScaled(x, py-26*scale, scale, tab)
		}
	}
	for i, icon := range creativeTabIcons {
		x := px + (7+i*18)*scale
		fb.DrawItemIconScaled(x, py-20*scale, icon, scale)
	}

	items := inventory.SearchCreativeCatalog(category, query)
	maxScroll := CreativeCatalogMaxScroll(len(items))
	if scrollRow < 0 {
		scrollRow = 0
	}
	if scrollRow > maxScroll {
		scrollRow = maxScroll
	}
	start := scrollRow * creativeColumns
	hoveredName := ""
	for i := 0; i < creativePerPage && start+i < len(items); i++ {
		row, col := i/creativeColumns, i%creativeColumns
		sx := px + (7+col*18)*scale
		sy := py + (16+row*18)*scale
		fb.DrawItemIconScaled(sx+scale, sy+scale, items[start+i].ID, scale)
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0x50FFFFFF)
			hoveredName = items[start+i].Name
		}
	}
	drawCreativeTooltip(fb, hoveredName, mouseX, mouseY, scale)
	trackX, trackY, trackH := px+174*scale, py+16*scale, 90*scale
	thumbY := trackY
	if maxScroll > 0 {
		thumbY += (trackH - 15*scale) * scrollRow / maxScroll
	}
	if thumb, ok := GetOfficialImage("gui/sprites/container/creative_inventory/scroller.png"); ok {
		fb.DrawRawImageScaled(trackX, thumbY, scale, thumb)
	} else {
		fb.DrawRect(trackX, thumbY, 12*scale, 15*scale, 0xFFAAAAAA)
	}

	if category == inventory.CreativeAll {
		searchLabel := "搜索物品"
		if query != "" {
			searchLabel = query
		}
		fb.DrawText5x7(px+83*scale, py+7*scale, searchLabel, 0xFF404040, scale)
	}

	invX, invY := px+7*scale, py+144*scale+inventoryOffsetY
	hoveredInventoryName := ""
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			hoveredInventoryName = drawCreativePlayerSlot(fb, inv, 9+row*9+col, invX+col*18*scale, invY+row*18*scale, scale, mouseX, mouseY)
		}
	}
	for col := 0; col < 9; col++ {
		name := drawCreativePlayerSlot(fb, inv, col, invX+col*18*scale, py+203*scale+inventoryOffsetY, scale, mouseX, mouseY)
		if name != "" {
			hoveredInventoryName = name
		}
	}
	if hoveredInventoryName != "" {
		drawCreativeTooltip(fb, hoveredInventoryName, mouseX, mouseY, scale)
	}
	for i := range creativeTabIcons {
		x, y := px+(2+i*18)*scale, py-26*scale
		if mouseX >= x && mouseX < x+18*scale && mouseY >= y && mouseY < y+32*scale {
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

// CreativeCatalogMaxScroll returns the number of item rows that fit below the
// last visible row. It is shared by rendering and input so the scrollbar,
// wheel, and hit testing always agree about the final reachable item.
func CreativeCatalogMaxScroll(itemCount int) int {
	return max(0, (itemCount+creativeColumns-1)/creativeColumns-creativeRows)
}

func drawCreativePlayerSlot(fb *Framebuffer, inv *inventory.PlayerInventory, index, x, y, scale, mouseX, mouseY int) string {
	item := inv.Slots[index]
	if !item.IsEmpty() {
		fb.DrawItemIconScaled(x+scale, y+scale, item.ItemID, scale)
		if item.Count > 1 {
			fb.DrawNumberScaled(x+11*scale, y+11*scale, item.Count, 0xFFFFFFFF, scale)
		}
		drawCreativeDurabilityBar(fb, item, x, y, scale)
	}
	if index == inv.SelectedHotbar {
		fb.DrawRect(x, y, 18*scale, scale, 0xFFFFFFFF)
		fb.DrawRect(x, y+17*scale, 18*scale, scale, 0xFFFFFFFF)
		fb.DrawRect(x, y, scale, 18*scale, 0xFFFFFFFF)
		fb.DrawRect(x+17*scale, y, scale, 18*scale, 0xFFFFFFFF)
	}
	if mouseX >= x && mouseX < x+18*scale && mouseY >= y && mouseY < y+18*scale {
		fb.DrawRect(x+scale, y+scale, 16*scale, 16*scale, 0x50FFFFFF)
		if !item.IsEmpty() {
			return inventory.CreativeItemDisplayName(item.ItemID)
		}
	}
	return ""
}

func drawCreativeDurabilityBar(fb *Framebuffer, item inventory.ItemStack, x, y, scale int) {
	props, ok := inventory.GetToolProperties(item.ItemID)
	if !ok || item.Damage <= 0 || props.MaxDurability <= 0 || scale <= 0 {
		return
	}
	ratio := float32(props.MaxDurability-item.Damage) / float32(props.MaxDurability)
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	const barWidth = 13
	fb.DrawRect(x+2*scale, y+15*scale, barWidth*scale, 2*scale, 0xFF000000)
	fillWidth := int(ratio * barWidth * float32(scale))
	red := uint32((1 - ratio) * 255)
	green := uint32(ratio * 255)
	color := 0xFF000000 | (red << 16) | (green << 8)
	fb.DrawRect(x+2*scale, y+15*scale, fillWidth, scale, color)
}

func drawCreativeTextureRegion(fb *Framebuffer, x, y, scale int, image *RawImage, width, height int) {
	if fb == nil || image == nil || scale <= 0 || width <= 0 || height <= 0 ||
		image.Width <= 0 || image.Height <= 0 || image.Height > len(image.Pixels)/image.Width {
		return
	}
	if width > image.Width {
		width = image.Width
	}
	if height > image.Height {
		height = image.Height
	}
	for iy := 0; iy < height; iy++ {
		for ix := 0; ix < width; ix++ {
			color := image.Pixels[iy*image.Width+ix]
			if color>>24 != 0 {
				fb.DrawRect(x+ix*scale, y+iy*scale, scale, scale, color)
			}
		}
	}
}

func drawCreativeTooltip(fb *Framebuffer, label string, mouseX, mouseY, scale int) {
	if label == "" {
		return
	}
	width := creativeTooltipWidth(label, scale)
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

func creativeTooltipWidth(label string, scale int) int {
	return MeasureText5x7(label, scale) + 8*scale
}

// HitTestCreativeInventory maps mouse coordinates to a tab, catalog item,
// scrollbar, or player inventory slot.
func HitTestCreativeInventory(w, h int, category inventory.CreativeCategory, query string, scrollRow, mouseX, mouseY int) GUISlotHit {
	scale, px, py, inventoryOffsetY := creativeGUIOrigin(w, h)
	if category == inventory.CreativeAll && mouseX >= px+80*scale && mouseX < px+170*scale && mouseY >= py+4*scale && mouseY < py+15*scale {
		return GUISlotHit{Type: SlotCreativeSearch}
	}
	for i := range creativeTabIcons {
		x := px + (2+i*18)*scale
		if mouseX >= x && mouseX < x+18*scale && mouseY >= py-26*scale && mouseY < py+6*scale {
			return GUISlotHit{Type: SlotCreativeTab, Index: i}
		}
	}
	trackX, trackY, trackH := px+174*scale, py+16*scale, 90*scale
	if mouseX >= trackX && mouseX < trackX+12*scale && mouseY >= trackY && mouseY < trackY+trackH {
		return GUISlotHit{Type: SlotCreativeScroll, Index: (mouseY - trackY) * 100 / max(1, trackH-1)}
	}
	items := inventory.SearchCreativeCatalog(category, query)
	maxScroll := CreativeCatalogMaxScroll(len(items))
	if scrollRow < 0 {
		scrollRow = 0
	}
	if scrollRow > maxScroll {
		scrollRow = maxScroll
	}
	start := scrollRow * creativeColumns
	for i := 0; i < creativePerPage; i++ {
		row, col := i/creativeColumns, i%creativeColumns
		sx, sy := px+(7+col*18)*scale, py+(16+row*18)*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale && start+i < len(items) {
			return GUISlotHit{Type: SlotCreativeItem, Index: start + i}
		}
	}
	invX, invY := px+7*scale, py+144*scale+inventoryOffsetY
	for row := 0; row < 3; row++ {
		for col := 0; col < 9; col++ {
			sx, sy := invX+col*18*scale, invY+row*18*scale
			if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
				return GUISlotHit{Type: SlotInventory, Index: 9 + row*9 + col}
			}
		}
	}
	for col := 0; col < 9; col++ {
		sx, sy := invX+col*18*scale, py+203*scale+inventoryOffsetY
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}
	return GUISlotHit{Type: SlotNone, Index: -1}
}
