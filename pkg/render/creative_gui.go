package render

import (
	"strconv"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

const (
	creativeColumns = 9
	creativeRows    = 5
	creativePerPage = creativeColumns * creativeRows
	creativePanelW  = 195
	creativePanelH  = 136
	creativeHotbarY = 112
)

// creativeGUIOrigin follows AbstractContainerScreen's 195x136 image bounds.
// Top and bottom tab rows extend beyond those bounds by their vanilla sprite sizes.
func creativeGUIOrigin(w, h int) (scale, px, py int) {
	scale = CalculateGUIScaleFromDimensions(w, h)
	px = w/2 - creativePanelW*scale/2
	py = h/2 - creativePanelH*scale/2
	return scale, px, py
}

type creativeTabLayout struct {
	category inventory.CreativeCategory
	column   int
	bottom   bool
	icon     int32
	name     string
}

// Positions and spacing match Java 26.3 CreativeModeInventoryScreen's seven
// 27-pixel columns: five categories, Hotbar/Operator, then Search/Inventory.
var creativeTabs = [...]creativeTabLayout{
	{category: inventory.CreativeBlocks, column: 0, icon: 1, name: "建筑方块"},
	{category: inventory.CreativeDecoration, column: 1, icon: 35, name: "有色方块"},
	{category: inventory.CreativeNatural, column: 2, icon: 6, name: "自然方块"},
	{category: inventory.CreativeFunctional, column: 3, icon: 54, name: "功能方块"},
	{category: inventory.CreativeRedstone, column: 4, icon: 331, name: "红石方块"},
	{category: inventory.CreativeHotbar, column: 5, icon: 54, name: "快捷栏预设"},
	{category: inventory.CreativeAll, column: 6, icon: 345, name: "搜索物品"},
	{category: inventory.CreativeTools, column: 0, bottom: true, icon: 270, name: "工具与实用物品"},
	{category: inventory.CreativeCombat, column: 1, bottom: true, icon: 268, name: "战斗"},
	{category: inventory.CreativeFood, column: 2, bottom: true, icon: 260, name: "食物与饮料"},
	{category: inventory.CreativeMaterials, column: 3, bottom: true, icon: 296, name: "材料"},
	{category: inventory.CreativeSpawnEggs, column: 4, bottom: true, icon: 11002, name: "刷怪蛋"},
	{category: inventory.CreativeInventory, column: 6, bottom: true, icon: 54, name: "物品栏"},
}

func creativeTabX(tab creativeTabLayout) int {
	return 27 * tab.column
}

func creativeTabY(tab creativeTabLayout) int {
	if tab.bottom {
		return creativePanelH - 4
	}
	return -28
}

func creativeTabSpriteVariant(tab creativeTabLayout) int {
	return tab.column + 1
}

// DrawCreativeInventoryGUI draws the Java 26.3 creative item picker: a 195x136
// panel with a 9x5 catalog and the player's hotbar below it.
func (fb *Framebuffer) DrawCreativeInventoryGUI(inv *inventory.PlayerInventory, cursor inventory.ItemStack, category inventory.CreativeCategory, query string, scrollRow, mouseX, mouseY int, presetValues ...inventory.CreativeHotbarPresets) {
	if fb == nil || inv == nil {
		return
	}
	var presets inventory.CreativeHotbarPresets
	if len(presetValues) > 0 {
		presets = presetValues[0]
	}
	scale, px, py := creativeGUIOrigin(fb.Width, fb.Height)
	for i, c := range fb.Color {
		r, g, b := (c>>16)&0xFF, (c>>8)&0xFF, c&0xFF
		fb.Color[i] = 0xFF000000 | ((r / 3) << 16) | ((g / 3) << 8) | (b / 3)
	}
	creativeTexture := "gui/container/creative_inventory/tab_items.png"
	if category == inventory.CreativeAll {
		creativeTexture = "gui/container/creative_inventory/tab_item_search.png"
	} else if category == inventory.CreativeInventory {
		creativeTexture = "gui/container/creative_inventory/tab_inventory.png"
	}
	if image, ok := GetOfficialImage(creativeTexture); ok {
		drawCreativeTextureRegion(fb, px, py, scale, image, creativePanelW, creativePanelH)
	}
	drawTab := func(tab creativeTabLayout, selected bool) {
		row := "top"
		if tab.bottom {
			row = "bottom"
		}
		state := "unselected"
		if selected {
			state = "selected"
		}
		variant := creativeTabSpriteVariant(tab)
		texturePath := "gui/sprites/container/creative_inventory/tab_" + row + "_" + state + "_" + strconv.Itoa(variant) + ".png"
		if image, ok := GetOfficialImage(texturePath); ok {
			tabX := px + creativeTabX(tab)*scale
			tabY := py + creativeTabY(tab)*scale
			fb.DrawRawImageScaled(tabX, tabY, scale, image)
			iconY := tabY + 8*scale
			if tab.bottom {
				iconY -= 2 * scale
			}
			fb.DrawItemIconScaled(tabX+5*scale, iconY, tab.icon, scale)
		}
	}
	for _, tab := range creativeTabs {
		if tab.category == category {
			continue
		}
		drawTab(tab, false)
	}
	for _, tab := range creativeTabs {
		if tab.category == category {
			drawTab(tab, true)
			break
		}
	}

	items := inventory.SearchCreativeCatalog(category, query)
	if category == inventory.CreativeHotbar {
		items = creativeHotbarItems(presets)
	}
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
		sx := px + (9+col*18)*scale
		sy := py + (18+row*18)*scale
		fb.DrawItemIconScaled(sx+scale, sy+scale, items[start+i].ID, scale)
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			fb.DrawRect(sx+scale, sy+scale, 16*scale, 16*scale, 0x50FFFFFF)
			hoveredName = items[start+i].Name
		}
	}
	drawCreativeTooltip(fb, hoveredName, mouseX, mouseY, scale)
	if category != inventory.CreativeInventory {
		trackX, trackY, trackH := px+175*scale, py+18*scale, 112*scale
		thumbY := trackY
		if maxScroll > 0 {
			thumbY += (trackH - 17*scale) * scrollRow / maxScroll
		}
		thumbName := "scroller_disabled.png"
		if maxScroll > 0 {
			thumbName = "scroller.png"
		}
		if thumb, ok := GetOfficialImage("gui/sprites/container/creative_inventory/" + thumbName); ok {
			fb.DrawRawImageScaled(trackX, thumbY, scale, thumb)
		} else {
			fb.DrawRect(trackX, thumbY, 12*scale, 15*scale, 0xFFAAAAAA)
		}
	}

	if category == inventory.CreativeAll {
		searchLabel := "搜索物品"
		if query != "" {
			searchLabel = query
		}
		drawCreativeSearchText(fb, px+82*scale, py+6*scale, searchLabel, 0xFF404040, scale, 80*scale)
	}

	invX, invY := px+9*scale, py+creativeHotbarY*scale
	hoveredInventoryName := ""
	if category == inventory.CreativeInventory {
		// The inventory tab exposes the 27 main slots and the hotbar. Armor slots
		// are drawn empty until armor/equipment data is represented by PlayerInventory.
		for slot := 9; slot < 36; slot++ {
			col, row := (slot-9)%9, (slot-9)/9
			name := drawCreativePlayerSlot(fb, inv, slot, px+(9+col*18)*scale, py+(54+row*18)*scale, scale, mouseX, mouseY)
			if name != "" {
				hoveredInventoryName = name
			}
		}
		for col := 0; col < 9; col++ {
			name := drawCreativePlayerSlot(fb, inv, col, invX+col*18*scale, invY, scale, mouseX, mouseY)
			if name != "" {
				hoveredInventoryName = name
			}
		}
	} else {
		for col := 0; col < 9; col++ {
			name := drawCreativePlayerSlot(fb, inv, col, invX+col*18*scale, invY, scale, mouseX, mouseY)
			if name != "" {
				hoveredInventoryName = name
			}
		}
	}
	if hoveredInventoryName != "" {
		drawCreativeTooltip(fb, hoveredInventoryName, mouseX, mouseY, scale)
	}
	for _, tab := range creativeTabs {
		x, y := px+creativeTabX(tab)*scale, py+(creativeTabY(tab)-4)*scale
		if mouseX >= x && mouseX < x+26*scale && mouseY >= y && mouseY < y+32*scale {
			drawCreativeTooltip(fb, tab.name, mouseX, mouseY, scale)
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

func creativeHotbarItems(presets inventory.CreativeHotbarPresets) []inventory.CreativeItem {
	items := make([]inventory.CreativeItem, 0, 81)
	for _, preset := range presets {
		for _, stack := range preset {
			if stack.IsEmpty() {
				items = append(items, inventory.CreativeItem{ID: 339, Name: "空快捷栏槽位", Category: inventory.CreativeHotbar})
				continue
			}
			items = append(items, inventory.CreativeItem{ID: stack.ItemID, Name: inventory.CreativeItemDisplayName(stack.ItemID), Category: inventory.CreativeHotbar})
		}
	}
	return items
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

func drawCreativeSearchText(fb *Framebuffer, x, y int, text string, color uint32, scale, maxWidth int) {
	if fb == nil || scale < 1 || maxWidth <= 0 {
		return
	}
	drawPass := func(originX, originY int, drawColor uint32) {
		cursorX := originX
		for _, ch := range text {
			if ch > 0x7F {
				glyph := GetCJKGlyph(ch)
				const glyphSize = 9
				for gy := 0; gy < glyphSize; gy++ {
					sourceY := gy * len(glyph) / glyphSize
					for gx := 0; gx < glyphSize; gx++ {
						sourceX := gx * 12 / glyphSize
						if glyph[sourceY]&(1<<(11-sourceX)) != 0 {
							drawCreativeSearchPixel(fb, cursorX+gx*scale, originY+gy*scale, scale, drawColor, x, x+maxWidth)
						}
					}
				}
				cursorX += glyphSize * scale
				continue
			}
			if ch >= 'a' && ch <= 'z' {
				ch = ch - 'a' + 'A'
			}
			glyph, ok := font5x7[ch]
			if !ok {
				glyph = font5x7['?']
			}
			for gy, bits := range glyph {
				for gx := 0; gx < 5; gx++ {
					if bits&(1<<(4-gx)) != 0 {
						drawCreativeSearchPixel(fb, cursorX+gx*scale, originY+gy*scale, scale, drawColor, x, x+maxWidth)
					}
				}
			}
			cursorX += 6 * scale
		}
	}
	drawPass(x+scale, y+scale, 0xFF222222)
	drawPass(x, y, color)
}

func drawCreativeSearchPixel(fb *Framebuffer, x, y, scale int, color uint32, clipLeft, clipRight int) {
	for dy := 0; dy < scale; dy++ {
		py := y + dy
		if py < 0 || py >= fb.Height {
			continue
		}
		for dx := 0; dx < scale; dx++ {
			px := x + dx
			if px >= clipLeft && px < clipRight && px >= 0 && px < fb.Width {
				fb.Color[py*fb.Width+px] = color
			}
		}
	}
}

// HitTestCreativeInventory maps mouse coordinates to a tab, catalog item,
// scrollbar, or player inventory slot.
func HitTestCreativeInventory(w, h int, category inventory.CreativeCategory, query string, scrollRow, mouseX, mouseY int, presetValues ...inventory.CreativeHotbarPresets) GUISlotHit {
	scale, px, py := creativeGUIOrigin(w, h)
	if category == inventory.CreativeAll && mouseX >= px+82*scale && mouseX < px+162*scale && mouseY >= py+4*scale && mouseY < py+15*scale {
		return GUISlotHit{Type: SlotCreativeSearch}
	}
	for _, tab := range creativeTabs {
		x, y := px+creativeTabX(tab)*scale, py+(creativeTabY(tab)-4)*scale
		if mouseX >= x && mouseX <= x+26*scale && mouseY >= y && mouseY <= y+32*scale {
			return GUISlotHit{Type: SlotCreativeTab, Index: int(tab.category)}
		}
	}
	trackX, trackY, trackH := px+175*scale, py+18*scale, 112*scale
	if mouseX >= trackX && mouseX < trackX+12*scale && mouseY >= trackY && mouseY < trackY+trackH {
		return GUISlotHit{Type: SlotCreativeScroll, Index: (mouseY - trackY) * 100 / max(1, trackH-1)}
	}
	items := inventory.SearchCreativeCatalog(category, query)
	if category == inventory.CreativeHotbar {
		var presets inventory.CreativeHotbarPresets
		if len(presetValues) > 0 {
			presets = presetValues[0]
		}
		items = creativeHotbarItems(presets)
	}
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
		sx, sy := px+(9+col*18)*scale, py+(18+row*18)*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale && start+i < len(items) {
			return GUISlotHit{Type: SlotCreativeItem, Index: start + i}
		}
	}
	if category == inventory.CreativeInventory {
		for slot := 9; slot < 36; slot++ {
			col, row := (slot-9)%9, (slot-9)/9
			sx, sy := px+(9+col*18)*scale, py+(54+row*18)*scale
			if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
				return GUISlotHit{Type: SlotInventory, Index: slot}
			}
		}
		sx, sy := px+173*scale, py+112*scale
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			return GUISlotHit{Type: SlotCreativeTrash, Index: 0}
		}
	}
	invX, invY := px+9*scale, py+creativeHotbarY*scale
	for col := 0; col < 9; col++ {
		sx, sy := invX+col*18*scale, invY
		if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
			return GUISlotHit{Type: SlotInventory, Index: col}
		}
	}
	return GUISlotHit{Type: SlotNone, Index: -1}
}
