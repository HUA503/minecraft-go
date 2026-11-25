package render

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

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
// 27-pixel columns: five categories, Saved Hotbars, then Search/Inventory.
var creativeTabs = [...]creativeTabLayout{
	{category: inventory.CreativeBlocks, column: 0, icon: 45, name: "建筑方块"},
	{category: inventory.CreativeDecoration, column: 1, icon: inventory.ItemCyanWool, name: "有色方块"},
	{category: inventory.CreativeNatural, column: 2, icon: 6, name: "自然方块"},
	{category: inventory.CreativeFunctional, column: 3, icon: 323, name: "功能方块"},
	{category: inventory.CreativeRedstone, column: 4, icon: 331, name: "红石方块"},
	{category: inventory.CreativeHotbar, column: 5, icon: 47, name: "已保存的快捷栏"},
	{category: inventory.CreativeAll, column: 6, icon: 345, name: "搜索物品"},
	{category: inventory.CreativeTools, column: 0, bottom: true, icon: inventory.ItemDiamondPickaxe, name: "工具与实用物品"},
	{category: inventory.CreativeCombat, column: 1, bottom: true, icon: inventory.ItemNetheriteSword, name: "战斗"},
	{category: inventory.CreativeFood, column: 2, bottom: true, icon: inventory.ItemGoldenApple, name: "食物与饮料"},
	{category: inventory.CreativeMaterials, column: 3, bottom: true, icon: inventory.ItemIronIngot, name: "材料"},
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
		trackX := px + 175*scale
		thumbY := CreativeScrollThumbY(fb.Width, fb.Height, scrollRow, maxScroll)
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
		previewX := px + 81*scale
		previewTop := py + 8*scale
		drawCreativePlayerPreview(fb, inv, previewX, previewTop, scale, mouseX, mouseY)
		for slot := 0; slot < 4; slot++ {
			x := px + (54+(slot/2)*54)*scale
			y := py + (6+(slot%2)*27)*scale
			name := drawCreativeArmorSlot(fb, inv, slot, x, y, scale, mouseX, mouseY)
			if name != "" {
				hoveredInventoryName = name
			}
		}
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

type creativePreviewFace struct {
	points  [4]ScreenPoint
	uvs     [4][2]float32
	depth   float64
	face    int
	color   uint32
	texture *RawImage
}

type creativePreviewBox struct {
	center [3]float64
	half   [3]float64
	colors [6]uint32
	uvs    [6][4][2]float32
}

var creativePlayerSkinCache struct {
	sync.Once
	texture *RawImage
}

func creativePlayerSkinTexture() *RawImage {
	creativePlayerSkinCache.Do(func() {
		if configDir, err := os.UserConfigDir(); err == nil {
			if texture, err := loadCreativePlayerSkinFromConfig(configDir); err == nil {
				creativePlayerSkinCache.texture = texture
				return
			}
		}
		creativePlayerSkinCache.texture, _ = GetOfficialImage("entity/player/steve.png")
	})
	return creativePlayerSkinCache.texture
}

func loadCreativePlayerSkinFromConfig(configDir string) (*RawImage, error) {
	file, err := os.Open(filepath.Join(configDir, "minecraft-go", "skin.png"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	texture, err := LoadRawImage(file)
	if err != nil {
		return nil, err
	}
	if texture.Width == 64 && texture.Height == 32 {
		return expandLegacyCreativeSkin(texture), nil
	}
	if texture.Width != 64 || texture.Height != 64 {
		return nil, fmt.Errorf("player skin must be 64x64 or legacy 64x32 pixels, got %dx%d", texture.Width, texture.Height)
	}
	return texture, nil
}

func expandLegacyCreativeSkin(texture *RawImage) *RawImage {
	if texture == nil || texture.Width != 64 || texture.Height != 32 || len(texture.Pixels) != 64*32 {
		return texture
	}
	expanded := &RawImage{Width: 64, Height: 64, Pixels: make([]uint32, 64*64)}
	copy(expanded.Pixels[:64*32], texture.Pixels)
	copyRegion := func(srcX, srcY, dstX, dstY, width, height int) {
		for y := 0; y < height; y++ {
			src := (srcY+y)*texture.Width + srcX
			dst := (dstY+y)*expanded.Width + dstX
			copy(expanded.Pixels[dst:dst+width], texture.Pixels[src:src+width])
		}
	}
	// Legacy skins contain only the right limbs. Mirror those UV islands into
	// the modern left-arm and left-leg regions used by the preview model.
	copyRegion(40, 16, 32, 48, 16, 16)
	copyRegion(0, 16, 16, 48, 16, 16)
	return expanded
}

// drawCreativePlayerPreview draws the same six-part block model used by the
// in-world player renderer into the open player-inventory viewport. The mouse
// position controls the model's yaw, matching the original inventory preview
// interaction while keeping the preview inside the inventory panel.
func drawCreativePlayerPreview(fb *Framebuffer, inv *inventory.PlayerInventory, centerX, topY, scale, mouseX, mouseY int) {
	if fb == nil || scale <= 0 {
		return
	}
	viewport := [4]int{centerX - 24*scale, topY, centerX + 24*scale, topY + 45*scale}
	if mouseX < 0 || mouseY < 0 {
		mouseX, mouseY = centerX, topY+22*scale
	}
	yaw := math.Atan2(float64(mouseX-centerX), float64(48*scale)) * 0.9
	cosYaw, sinYaw := math.Cos(yaw), math.Sin(yaw)
	pitchInput := float64(mouseY-(topY+22*scale)) / float64(24*scale)
	pitchInput = math.Max(-1, math.Min(1, pitchInput))
	pitch := pitchInput * 0.65
	cosPitch, sinPitch := math.Cos(pitch), math.Sin(pitch)

	// The model uses the official classic Steve skin as the local default. The
	// six-part geometry remains the preview's existing software-rendered model.
	skin := creativePlayerSkinTexture()
	const (
		headSkin = uint32(0xFFD6A07E)
		headHair = uint32(0xFF4D3B26)
		shirt    = uint32(0xFF2E8690)
		pants    = uint32(0xFF2A3663)
		shoes    = uint32(0xFF555555)
	)
	boxes := []creativePreviewBox{
		{[3]float64{0, 1.67, 0}, [3]float64{.25, .25, .25}, [6]uint32{headHair, headSkin, headHair, headSkin, headSkin, headSkin}, creativeSkinBoxUV([4]int{8, 8, 16, 16}, [4]int{24, 8, 32, 16}, [4]int{0, 8, 8, 16}, [4]int{16, 8, 24, 16}, [4]int{8, 0, 16, 8}, [4]int{16, 0, 24, 8})},
		{[3]float64{0, 1.125, 0}, [3]float64{.25, .375, .125}, [6]uint32{shirt, shirt, shirt, shirt, shirt, shirt}, creativeSkinBoxUV([4]int{20, 20, 28, 32}, [4]int{32, 20, 40, 32}, [4]int{28, 20, 32, 32}, [4]int{16, 20, 20, 32}, [4]int{20, 16, 28, 20}, [4]int{28, 16, 36, 20})},
		{[3]float64{-.375, 1.125, 0}, [3]float64{.125, .375, .125}, [6]uint32{shirt, headSkin, shirt, headSkin, shirt, shirt}, creativeSkinBoxUV([4]int{44, 20, 48, 32}, [4]int{52, 20, 56, 32}, [4]int{48, 20, 52, 32}, [4]int{40, 20, 44, 32}, [4]int{44, 16, 48, 20}, [4]int{48, 16, 52, 20})},
		{[3]float64{.375, 1.125, 0}, [3]float64{.125, .375, .125}, [6]uint32{shirt, headSkin, shirt, headSkin, shirt, shirt}, creativeSkinBoxUV([4]int{36, 52, 40, 64}, [4]int{44, 52, 48, 64}, [4]int{40, 52, 44, 64}, [4]int{32, 52, 36, 64}, [4]int{36, 48, 40, 52}, [4]int{40, 48, 44, 52})},
		{[3]float64{-.125, .375, 0}, [3]float64{.125, .375, .125}, [6]uint32{pants, shoes, pants, pants, pants, pants}, creativeSkinBoxUV([4]int{4, 20, 8, 32}, [4]int{12, 20, 16, 32}, [4]int{8, 20, 12, 32}, [4]int{0, 20, 4, 32}, [4]int{4, 16, 8, 20}, [4]int{8, 16, 12, 20})},
		{[3]float64{.125, .375, 0}, [3]float64{.125, .375, .125}, [6]uint32{pants, shoes, pants, pants, pants, pants}, creativeSkinBoxUV([4]int{20, 52, 24, 64}, [4]int{28, 52, 32, 64}, [4]int{24, 52, 28, 64}, [4]int{16, 52, 20, 64}, [4]int{20, 48, 24, 52}, [4]int{24, 48, 28, 52})},
	}
	// Minecraft skins have a separate transparent outer layer for the hat,
	// jacket, sleeves, and pant cuffs. Keep it just outside the base mesh so the
	// alpha texture draws only the player's configured clothing pixels.
	boxes = append(boxes,
		creativePreviewBox{[3]float64{0, 1.67, 0}, [3]float64{.257, .257, .257}, [6]uint32{headHair, headSkin, headHair, headSkin, headSkin, headSkin}, creativeSkinBoxUV([4]int{40, 8, 48, 16}, [4]int{56, 8, 64, 16}, [4]int{32, 8, 40, 16}, [4]int{48, 8, 56, 16}, [4]int{40, 0, 48, 8}, [4]int{48, 0, 56, 8})},
		creativePreviewBox{[3]float64{0, 1.125, 0}, [3]float64{.257, .377, .132}, [6]uint32{shirt, shirt, shirt, shirt, shirt, shirt}, creativeSkinBoxUV([4]int{20, 36, 28, 48}, [4]int{32, 36, 40, 48}, [4]int{28, 36, 32, 48}, [4]int{16, 36, 20, 48}, [4]int{20, 32, 28, 36}, [4]int{28, 32, 36, 36})},
		creativePreviewBox{[3]float64{-.375, 1.125, 0}, [3]float64{.127, .377, .132}, [6]uint32{shirt, headSkin, shirt, headSkin, shirt, shirt}, creativeSkinBoxUV([4]int{44, 36, 48, 48}, [4]int{52, 36, 56, 48}, [4]int{48, 36, 52, 48}, [4]int{40, 36, 44, 48}, [4]int{44, 32, 48, 36}, [4]int{48, 32, 52, 36})},
		creativePreviewBox{[3]float64{.375, 1.125, 0}, [3]float64{.127, .377, .132}, [6]uint32{shirt, headSkin, shirt, headSkin, shirt, shirt}, creativeSkinBoxUV([4]int{52, 52, 56, 64}, [4]int{60, 52, 64, 64}, [4]int{56, 52, 60, 64}, [4]int{48, 52, 52, 64}, [4]int{52, 48, 56, 52}, [4]int{56, 48, 60, 52})},
		creativePreviewBox{[3]float64{-.125, .375, 0}, [3]float64{.127, .377, .132}, [6]uint32{pants, shoes, pants, pants, pants, pants}, creativeSkinBoxUV([4]int{4, 36, 8, 48}, [4]int{12, 36, 16, 48}, [4]int{8, 36, 12, 48}, [4]int{0, 36, 4, 48}, [4]int{4, 32, 8, 36}, [4]int{8, 32, 12, 36})},
		creativePreviewBox{[3]float64{.125, .375, 0}, [3]float64{.127, .377, .132}, [6]uint32{pants, shoes, pants, pants, pants, pants}, creativeSkinBoxUV([4]int{4, 52, 8, 64}, [4]int{12, 52, 16, 64}, [4]int{8, 52, 12, 64}, [4]int{0, 52, 4, 64}, [4]int{4, 48, 8, 52}, [4]int{8, 48, 12, 52})},
	)
	boxTextures := make([]*RawImage, len(boxes))
	for i := range boxTextures {
		boxTextures[i] = skin
	}
	armorBoxes, armorTextures := creativeArmorPreviewBoxes(inv)
	boxes = append(boxes, armorBoxes...)
	boxTextures = append(boxTextures, armorTextures...)
	corners := [8][3]float64{
		{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
		{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
	}
	faceIndices := [6][4]int{
		{3, 2, 6, 7}, {4, 5, 1, 0}, {2, 3, 0, 1},
		{7, 6, 5, 4}, {3, 7, 4, 0}, {6, 2, 1, 5},
	}
	faces := make([]creativePreviewFace, 0, len(boxes)*6)
	unit := float64(18 * scale)
	for boxIndex, box := range boxes {
		var projected [8]ScreenPoint
		for i, corner := range corners {
			x := box.center[0] + corner[0]*box.half[0]
			y := box.center[1] + corner[1]*box.half[1]
			z := box.center[2] + corner[2]*box.half[2]
			rx := x*cosYaw + z*sinYaw
			rz := -x*sinYaw + z*cosYaw
			relY := y - 0.9
			ry := relY*cosPitch - rz*sinPitch + 0.9
			rz = relY*sinPitch + rz*cosPitch
			projected[i] = ScreenPoint{
				X:     float32(float64(centerX) + rx*unit),
				Y:     float32(float64(topY+41*scale) - ry*unit + rz*3*float64(scale)),
				Depth: float32(rz + 2),
			}
		}
		for face, indices := range faceIndices {
			var points [4]ScreenPoint
			depth := 0.0
			for i, index := range indices {
				points[i] = projected[index]
				depth += float64(projected[index].Depth)
			}
			color := shadeCreativePreviewFace(box.colors[face], face)
			faces = append(faces, creativePreviewFace{points: points, uvs: box.uvs[face], depth: depth / 4, face: face, color: color, texture: boxTextures[boxIndex]})
		}
	}
	sort.SliceStable(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, face := range faces {
		if face.texture != nil {
			drawCreativePreviewTexturedTriangle(fb, face.points[0], face.points[1], face.points[2], face.uvs[0], face.uvs[1], face.uvs[2], face.texture, face.face, viewport)
			drawCreativePreviewTexturedTriangle(fb, face.points[0], face.points[2], face.points[3], face.uvs[0], face.uvs[2], face.uvs[3], face.texture, face.face, viewport)
		} else {
			drawCreativePreviewTriangle(fb, face.points[0], face.points[1], face.points[2], face.color, viewport)
			drawCreativePreviewTriangle(fb, face.points[0], face.points[2], face.points[3], face.color, viewport)
		}
	}
}

func creativeSkinBoxUV(front, back, left, right, top, bottom [4]int) [6][4][2]float32 {
	return creativeBoxUV(front, back, left, right, top, bottom, 64, 64)
}

func creativeBoxUV(front, back, left, right, top, bottom [4]int, width, height int) [6][4][2]float32 {
	tlbr := func(rect [4]int, order [4]int) [4][2]float32 {
		x0, y0, x1, y1 := rect[0], rect[1], rect[2]-1, rect[3]-1
		corners := [4][2]float32{
			{float32(x0) / float32(width-1), float32(y0) / float32(height-1)}, {float32(x1) / float32(width-1), float32(y0) / float32(height-1)},
			{float32(x1) / float32(width-1), float32(y1) / float32(height-1)}, {float32(x0) / float32(width-1), float32(y1) / float32(height-1)},
		}
		var uv [4][2]float32
		for i, corner := range order {
			uv[i] = corners[corner]
		}
		return uv
	}
	return [6][4][2]float32{
		tlbr(top, [4]int{0, 1, 2, 3}),
		tlbr(bottom, [4]int{1, 0, 3, 2}),
		tlbr(back, [4]int{1, 0, 3, 2}),
		tlbr(front, [4]int{3, 2, 1, 0}),
		tlbr(left, [4]int{1, 0, 3, 2}),
		tlbr(right, [4]int{0, 1, 2, 3}),
	}
}

func creativeArmorPreviewBoxes(inv *inventory.PlayerInventory) ([]creativePreviewBox, []*RawImage) {
	if inv == nil {
		return nil, nil
	}
	var boxes []creativePreviewBox
	var textures []*RawImage
	appendBox := func(material string, leggings bool, box creativePreviewBox) {
		kind := "humanoid"
		if leggings {
			kind = "humanoid_leggings"
		}
		texture, ok := GetOfficialImage("entity/equipment/" + kind + "/" + material + ".png")
		if !ok || texture.Width != 64 || texture.Height != 32 {
			return
		}
		boxes = append(boxes, box)
		textures = append(textures, texture)
	}
	for slot, stack := range inv.Armor {
		if stack.IsEmpty() {
			continue
		}
		material := creativeArmorMaterial(stack.ItemID)
		if material == "" {
			continue
		}
		white := [6]uint32{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF}
		switch slot {
		case 3: // helmet
			appendBox(material, false, creativePreviewBox{[3]float64{0, 1.67, 0}, [3]float64{.267, .267, .267}, white, creativeBoxUV([4]int{8, 8, 16, 16}, [4]int{24, 8, 32, 16}, [4]int{0, 8, 8, 16}, [4]int{16, 8, 24, 16}, [4]int{8, 0, 16, 8}, [4]int{16, 0, 24, 8}, 64, 32)})
		case 2: // chestplate and sleeves
			appendBox(material, false, creativePreviewBox{[3]float64{0, 1.125, 0}, [3]float64{.267, .387, .142}, white, creativeBoxUV([4]int{20, 20, 28, 32}, [4]int{32, 20, 40, 32}, [4]int{28, 20, 32, 32}, [4]int{16, 20, 20, 32}, [4]int{20, 16, 28, 20}, [4]int{28, 16, 36, 20}, 64, 32)})
			for _, side := range []float64{-1, 1} {
				x := side * .375
				uv := creativeBoxUV([4]int{44, 20, 48, 32}, [4]int{52, 20, 56, 32}, [4]int{48, 20, 52, 32}, [4]int{40, 20, 44, 32}, [4]int{44, 16, 48, 20}, [4]int{48, 16, 52, 20}, 64, 32)
				appendBox(material, false, creativePreviewBox{[3]float64{x, 1.125, 0}, [3]float64{.142, .387, .142}, white, uv})
			}
		case 1: // leggings: waist and both legs
			appendBox(material, true, creativePreviewBox{[3]float64{0, .94, 0}, [3]float64{.267, .19, .142}, white, creativeBoxUV([4]int{20, 4, 28, 16}, [4]int{32, 4, 40, 16}, [4]int{28, 4, 32, 16}, [4]int{16, 4, 20, 16}, [4]int{20, 0, 28, 4}, [4]int{28, 0, 36, 4}, 64, 32)})
			for index, side := range []float64{-.125, .125} {
				x0 := 0
				if index == 1 {
					x0 = 16
				}
				uv := creativeBoxUV([4]int{x0 + 4, 20, x0 + 8, 32}, [4]int{x0 + 12, 20, x0 + 16, 32}, [4]int{x0 + 8, 20, x0 + 12, 32}, [4]int{x0, 20, x0 + 4, 32}, [4]int{x0 + 4, 16, x0 + 8, 20}, [4]int{x0 + 8, 16, x0 + 12, 20}, 64, 32)
				appendBox(material, true, creativePreviewBox{[3]float64{side, .375, 0}, [3]float64{.142, .387, .142}, white, uv})
			}
		case 0: // boots cover the lower part of each leg
			for index, side := range []float64{-.125, .125} {
				x0 := 0
				if index == 1 {
					x0 = 16
				}
				uv := creativeBoxUV([4]int{x0 + 4, 27, x0 + 8, 32}, [4]int{x0 + 12, 27, x0 + 16, 32}, [4]int{x0 + 8, 27, x0 + 12, 32}, [4]int{x0, 27, x0 + 4, 32}, [4]int{x0 + 4, 24, x0 + 8, 27}, [4]int{x0 + 8, 24, x0 + 12, 27}, 64, 32)
				appendBox(material, false, creativePreviewBox{[3]float64{side, .105, 0}, [3]float64{.142, .117, .142}, white, uv})
			}
		}
	}
	return boxes, textures
}

func creativeArmorMaterial(itemID int32) string {
	switch itemID {
	case 298, 299, 300, 301:
		return "leather"
	case 302, 303, 304, 305:
		return "chainmail"
	case 306, 307, 308, 309:
		return "iron"
	case 310, 311, 312, 313:
		return "diamond"
	case 314, 315, 316, 317:
		return "gold"
	case inventory.ItemCopperHelmet, inventory.ItemCopperChestplate, inventory.ItemCopperLeggings, inventory.ItemCopperBoots:
		return "copper"
	case inventory.ItemNetheriteHelmet, inventory.ItemNetheriteChestplate, inventory.ItemNetheriteLeggings, inventory.ItemNetheriteBoots:
		return "netherite"
	default:
		return ""
	}
}

func drawCreativePreviewTexturedTriangle(fb *Framebuffer, a, b, c ScreenPoint, ua, ub, uc [2]float32, texture *RawImage, face int, clip [4]int) {
	if fb == nil || texture == nil || texture.Width <= 0 || texture.Height <= 0 {
		return
	}
	minX := max(clip[0], int(math.Floor(math.Min(float64(a.X), math.Min(float64(b.X), float64(c.X))))))
	maxX := min(clip[2]-1, int(math.Ceil(math.Max(float64(a.X), math.Max(float64(b.X), float64(c.X))))))
	minY := max(clip[1], int(math.Floor(math.Min(float64(a.Y), math.Min(float64(b.Y), float64(c.Y))))))
	maxY := min(clip[3]-1, int(math.Ceil(math.Max(float64(a.Y), math.Max(float64(b.Y), float64(c.Y))))))
	if minX < 0 || minY < 0 || maxX >= fb.Width || maxY >= fb.Height || minX > maxX || minY > maxY {
		return
	}
	edge := func(p, q ScreenPoint, x, y float64) float64 {
		return (x-float64(p.X))*(float64(q.Y)-float64(p.Y)) - (y-float64(p.Y))*(float64(q.X)-float64(p.X))
	}
	area := edge(a, b, float64(c.X), float64(c.Y))
	if area == 0 {
		return
	}
	shade := [...]float64{1.12, .62, .78, 1, .78, .86}[face]
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+.5, float64(y)+.5
			w0, w1, w2 := edge(b, c, px, py), edge(c, a, px, py), edge(a, b, px, py)
			if !((w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0)) {
				continue
			}
			w0, w1, w2 = w0/area, w1/area, w2/area
			u := w0*float64(ua[0]) + w1*float64(ub[0]) + w2*float64(uc[0])
			v := w0*float64(ua[1]) + w1*float64(ub[1]) + w2*float64(uc[1])
			color := texture.Sample(float32(u), float32(v))
			if color>>24 == 0 {
				continue
			}
			red := uint32(math.Min(255, float64((color>>16)&0xFF)*shade))
			green := uint32(math.Min(255, float64((color>>8)&0xFF)*shade))
			blue := uint32(math.Min(255, float64(color&0xFF)*shade))
			fb.Color[y*fb.Width+x] = color&0xFF000000 | red<<16 | green<<8 | blue
		}
	}
}

func shadeCreativePreviewFace(color uint32, face int) uint32 {
	factor := [...]float64{1.12, .62, .78, 1, .78, .86}[face]
	r := uint32(math.Min(255, float64((color>>16)&0xFF)*factor))
	g := uint32(math.Min(255, float64((color>>8)&0xFF)*factor))
	b := uint32(math.Min(255, float64(color&0xFF)*factor))
	return color&0xFF000000 | r<<16 | g<<8 | b
}

func drawCreativePreviewTriangle(fb *Framebuffer, a, b, c ScreenPoint, color uint32, clip [4]int) {
	minX := max(clip[0], int(math.Floor(math.Min(float64(a.X), math.Min(float64(b.X), float64(c.X))))))
	maxX := min(clip[2]-1, int(math.Ceil(math.Max(float64(a.X), math.Max(float64(b.X), float64(c.X))))))
	minY := max(clip[1], int(math.Floor(math.Min(float64(a.Y), math.Min(float64(b.Y), float64(c.Y))))))
	maxY := min(clip[3]-1, int(math.Ceil(math.Max(float64(a.Y), math.Max(float64(b.Y), float64(c.Y))))))
	if minX < 0 || minY < 0 || maxX >= fb.Width || maxY >= fb.Height {
		minX, maxX = max(0, minX), min(fb.Width-1, maxX)
		minY, maxY = max(0, minY), min(fb.Height-1, maxY)
	}
	edge := func(p, q ScreenPoint, x, y float64) float64 {
		return (x-float64(p.X))*(float64(q.Y)-float64(p.Y)) - (y-float64(p.Y))*(float64(q.X)-float64(p.X))
	}
	area := edge(a, b, float64(c.X), float64(c.Y))
	if area == 0 || minX > maxX || minY > maxY {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+.5, float64(y)+.5
			w0, w1, w2 := edge(b, c, px, py), edge(c, a, px, py), edge(a, b, px, py)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				fb.Color[y*fb.Width+x] = color
			}
		}
	}
}

// CreativeCatalogMaxScroll returns the number of item rows that fit below the
// last visible row. It is shared by rendering and input so the scrollbar,
// wheel, and hit testing always agree about the final reachable item.
func CreativeCatalogMaxScroll(itemCount int) int {
	return max(0, (itemCount+creativeColumns-1)/creativeColumns-creativeRows)
}

// CreativeScrollThumbY returns the scrollbar thumb's top edge in framebuffer pixels.
func CreativeScrollThumbY(w, h, scrollRow, maxScroll int) int {
	scale, _, py := creativeGUIOrigin(w, h)
	trackY, trackH := py+18*scale, 112*scale
	thumbH := 17 * scale
	if maxScroll <= 0 {
		return trackY
	}
	scrollRow = max(0, min(maxScroll, scrollRow))
	return trackY + (trackH-thumbH)*scrollRow/maxScroll
}

// CreativeScrollRowAtPointer maps a scrollbar pointer position to the catalog row.
// thumbOffset is the pointer's vertical offset from the thumb's top edge.
func CreativeScrollRowAtPointer(w, h, mouseY, maxScroll, thumbOffset int) int {
	if maxScroll <= 0 {
		return 0
	}
	scale, _, py := creativeGUIOrigin(w, h)
	trackY, trackH := py+18*scale, 112*scale
	thumbH := 17 * scale
	travel := trackH - thumbH
	if travel <= 0 {
		return 0
	}
	thumbOffset = max(0, min(thumbH, thumbOffset))
	thumbTop := mouseY - thumbOffset
	thumbTop = max(trackY, min(trackY+travel, thumbTop))
	offset := thumbTop - trackY
	return (offset*maxScroll + travel/2) / travel
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

func drawCreativeArmorSlot(fb *Framebuffer, inv *inventory.PlayerInventory, index, x, y, scale, mouseX, mouseY int) string {
	if index < 0 || index >= len(inv.Armor) {
		return ""
	}
	item := inv.Armor[index]
	if !item.IsEmpty() {
		fb.DrawItemIconScaled(x+scale, y+scale, item.ItemID, scale)
		if item.Count > 1 {
			fb.DrawNumberScaled(x+11*scale, y+11*scale, item.Count, 0xFFFFFFFF, scale)
		}
		drawCreativeDurabilityBar(fb, item, x, y, scale)
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
		for slot := 0; slot < 4; slot++ {
			sx := px + (54+(slot/2)*54)*scale
			sy := py + (6+(slot%2)*27)*scale
			if mouseX >= sx && mouseX < sx+18*scale && mouseY >= sy && mouseY < sy+18*scale {
				return GUISlotHit{Type: SlotCreativeArmor, Index: slot}
			}
		}
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
