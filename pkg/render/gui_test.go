package render

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestGUIHitTestingAndDrawing(t *testing.T) {
	fb := NewFramebuffer(320, 240)
	inv := inventory.NewPlayerInventory()

	// 1. Test Crafting GUI Hit Testing
	// Center of screen
	scale := CalculateGUIScaleFromDimensions(320, 240)
	gw := 176 * scale
	gh := 166 * scale
	gx := 320/2 - gw/2
	gy := 240/2 - gh/2

	// Hit inside hotbar slot 0
	hitHotbar := HitTestCraftingGUI(320, 240, gx+8+2, gy+142+2, false)
	if hitHotbar.Type != SlotInventory || hitHotbar.Index != 0 {
		t.Fatalf("expected hotbar slot 0, got %+v", hitHotbar)
	}

	// Hit inside main inventory slot 9
	hitMain := HitTestCraftingGUI(320, 240, gx+8+2, gy+84+2, false)
	if hitMain.Type != SlotInventory || hitMain.Index != 9 {
		t.Fatalf("expected main slot 9, got %+v", hitMain)
	}

	// Hit inside 2x2 crafting grid slot 0
	hitCraft := HitTestCraftingGUI(320, 240, gx+98+2, gy+18+2, false)
	if hitCraft.Type != SlotCraftGrid || hitCraft.Index != 0 {
		t.Fatalf("expected craft grid slot 0, got %+v", hitCraft)
	}

	// Hit inside 2x2 result slot
	hitRes := HitTestCraftingGUI(320, 240, gx+154+2, gy+36+2, false)
	if hitRes.Type != SlotCraftResult {
		t.Fatalf("expected craft result slot, got %+v", hitRes)
	}
	hitHelmet := HitTestCraftingGUI(320, 240, gx+8+2, gy+8+2, false)
	if hitHelmet.Type != SlotCreativeArmor || hitHelmet.Index != 3 {
		t.Fatalf("expected helmet slot 3, got %+v", hitHelmet)
	}
	hitOffhand := HitTestCraftingGUI(320, 240, gx+77+2, gy+62+2, false)
	if hitOffhand.Type != SlotOffhand {
		t.Fatalf("expected offhand slot, got %+v", hitOffhand)
	}

	// Hit outside
	hitOut := HitTestCraftingGUI(320, 240, 2, 2, false)
	if hitOut.Type != SlotNone {
		t.Fatalf("expected slot none for outside, got %+v", hitOut)
	}

	// 2. Test Furnace GUI Hit Testing
	hitIn := HitTestFurnaceGUI(320, 240, gx+56+2, gy+17+2)
	if hitIn.Type != SlotFurnaceInput {
		t.Fatalf("expected furnace input slot, got %+v", hitIn)
	}
	hitFuel := HitTestFurnaceGUI(320, 240, gx+56+2, gy+53+2)
	if hitFuel.Type != SlotFurnaceFuel {
		t.Fatalf("expected furnace fuel slot, got %+v", hitFuel)
	}
	hitFurnaceOut := HitTestFurnaceGUI(320, 240, gx+116+2, gy+35+2)
	if hitFurnaceOut.Type != SlotFurnaceOutput {
		t.Fatalf("expected furnace output slot, got %+v", hitFurnaceOut)
	}

	// 3. Test Drawing without panics
	craftGrid := make([]inventory.ItemStack, 4)
	craftGrid[0] = inventory.ItemStack{ItemID: inventory.ItemOakLog, Count: 1}
	craftResult := inventory.ItemStack{ItemID: inventory.ItemOakPlanks, Count: 4}

	fb.DrawCraftingGUI(inv, craftGrid, craftResult, false, inventory.ItemStack{}, 100, 100)

	furnace := inventory.NewFurnaceBlockEntity()
	furnace.InputSlot = inventory.ItemStack{ItemID: 15, Count: 5}
	furnace.FuelSlot = inventory.ItemStack{ItemID: inventory.ItemCoal, Count: 2}
	furnace.BurnTimeRemaining = 800
	furnace.MaxBurnTime = 1600
	furnace.CookTimeProgress = 100
	fb.DrawFurnaceGUI(furnace, inv, inventory.ItemStack{}, 120, 120)

	spBtn, mpBtn, qBtn := fb.DrawTitleScreen(50, 50)
	if spBtn[2] <= 0 || mpBtn[2] <= 0 || qBtn[2] <= 0 {
		t.Fatalf("title screen buttons invalid")
	}

	worlds := []WorldInfo{
		{DirName: "world1", LevelName: "Survival World", GameMode: 0, LastTime: time.Now()},
	}
	pBtn, nBtn, bBtn, wRects := fb.DrawWorldSelectScreen(worlds, 0, 100, 100)
	if pBtn[2] <= 0 || nBtn[2] <= 0 || bBtn[2] <= 0 || len(wRects) != 1 {
		t.Fatalf("world select screen elements invalid")
	}

	cBtn, canBtn, mBtn, _, _ := fb.DrawNewWorldScreen("New World", 0, "12345", 0, 100, 100)
	if cBtn[2] <= 0 || canBtn[2] <= 0 || mBtn[2] <= 0 {
		t.Fatalf("new world screen elements invalid")
	}
}

func TestCreativeInventoryCatalogAndHitTesting(t *testing.T) {
	if tab := creativeTabForCategory(inventory.CreativeFood); tab.icon != inventory.ItemGoldenApple {
		t.Fatalf("Java 26.3 food tab icon = %d, want golden apple (%d)", tab.icon, inventory.ItemGoldenApple)
	}
	items := inventory.CreativeCatalog(inventory.CreativeAll)
	if len(items) < 150 {
		t.Fatalf("creative catalog unexpectedly small: %d items", len(items))
	}
	for _, category := range []inventory.CreativeCategory{
		inventory.CreativeBlocks, inventory.CreativeDecoration, inventory.CreativeNatural,
		inventory.CreativeFunctional, inventory.CreativeRedstone, inventory.CreativeTools,
		inventory.CreativeCombat, inventory.CreativeFood, inventory.CreativeMaterials,
	} {
		if got := inventory.CreativeCatalog(category); len(got) == 0 {
			t.Errorf("creative category %d is empty", category)
		}
	}
	if got := inventory.CreativeCatalog(inventory.CreativeSpawnEggs); len(got) != len(inventory.SpawnEggDefinitions())+1 || len(got) == 0 {
		t.Fatalf("spawn-egg tab has %d items, want %d supported mob eggs plus the vanilla spawner entry", len(got), len(inventory.SpawnEggDefinitions())+1)
	}

	w, h := 400, 300
	scale, px, py := creativeGUIOrigin(w, h)
	hitItem := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+10*scale, py+19*scale)
	if hitItem.Type != SlotCreativeItem || hitItem.Index != 0 {
		t.Fatalf("first creative item slot hit = %+v", hitItem)
	}
	toolsTab := creativeTabForCategory(inventory.CreativeTools)
	hitTools := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+(creativeTabX(toolsTab)+10)*scale, py+140*scale)
	if hitTools.Type != SlotCreativeTab || hitTools.Index != int(inventory.CreativeTools) {
		t.Fatalf("tools tab hit = %+v", hitTools)
	}
	hitSearch := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+100*scale, py+8*scale)
	if hitSearch.Type != SlotCreativeSearch {
		t.Fatalf("creative search field hit = %+v", hitSearch)
	}
	if hit := HitTestCreativeInventory(w, h, inventory.CreativeBlocks, "", 0, px+100*scale, py+8*scale); hit.Type == SlotCreativeSearch {
		t.Fatalf("search field should not be active outside the search tab: %+v", hit)
	}
	hitLaterItem := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 1, px+10*scale, py+19*scale)
	if hitLaterItem.Type != SlotCreativeItem || hitLaterItem.Index != 9 {
		t.Fatalf("creative second-page item hit = %+v", hitLaterItem)
	}
	hitScroll := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+178*scale, py+72*scale)
	if hitScroll.Type != SlotCreativeScroll || hitScroll.Index < 40 || hitScroll.Index > 65 {
		t.Fatalf("creative scrollbar hit = %+v", hitScroll)
	}
	if hit := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+10*scale, py+113*scale); hit.Type != SlotInventory || hit.Index != 0 {
		t.Fatalf("creative hotbar hit = %+v", hit)
	}

	fb := NewFramebuffer(w, h)
	fb.DrawCreativeInventoryGUI(inventory.NewPlayerInventory(), inventory.ItemStack{}, inventory.CreativeAll, "", 0, -1, -1)
	nonZero := 0
	for _, pixel := range fb.Color {
		if pixel != 0 {
			nonZero++
		}
	}
	if nonZero < 1000 {
		t.Fatalf("creative inventory rendered too few pixels: %d", nonZero)
	}
}

func TestCreativeSearchCaretUsesRenderedGlyphWidths(t *testing.T) {
	if got := creativeSearchTextAdvance("ab中文", 3); got != 21 {
		t.Fatalf("search caret advance = %d, want 21 pixels", got)
	}
	if got := creativeSearchTextAdvance("ab中文", 99); got != 30 {
		t.Fatalf("clamped search caret advance = %d, want 30 pixels", got)
	}
	if got := creativeSearchTextAdvance("ab中文", -1); got != 0 {
		t.Fatalf("negative search caret advance = %d, want 0", got)
	}
}

func TestCreativeSearchUsesTheSameVanillaASCIIAtlasAsOtherGUIText(t *testing.T) {
	const query = "Minecraft 26.3"
	vanilla := NewFramebuffer(180, 24)
	search := NewFramebuffer(180, 24)
	vanilla.DrawText5x7(2, 2, query, 0xFFFFFFFF, 1)
	drawCreativeSearchText(search, 2, 2, query, 0xFFFFFFFF, 1, 170)
	for i := range vanilla.Color {
		if vanilla.Color[i] != search.Color[i] {
			t.Fatalf("creative search glyph raster differs from vanilla GUI text at pixel (%d,%d)", i%vanilla.Width, i/vanilla.Width)
		}
	}
	queryWidth := vanillaASCIICharWidth(mustVanillaASCIIAtlas(t), 'W') + vanillaASCIICharWidth(mustVanillaASCIIAtlas(t), 'i')
	if got := creativeSearchTextAdvance("Wi", 2); got != queryWidth {
		t.Fatalf("creative search caret width for Wi = %d, want atlas width %d", got, queryWidth)
	}
}

func TestCreativeSearchCursorAtPointerUsesRuneBoundaries(t *testing.T) {
	width, height := 640, 480
	scale, panelX, _ := creativeGUIOrigin(width, height)
	query := "ab中"
	for _, tc := range []struct {
		name string
		x    int
		want int
	}{
		{"before text", panelX + 82*scale, 0},
		{"first rune", panelX + 83*scale + 1*scale, 0},
		{"after first rune", panelX + 83*scale + 5*scale, 1},
		{"after all runes", panelX + 83*scale + 30*scale, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CreativeSearchCursorAt(width, height, query, tc.x); got != tc.want {
				t.Fatalf("cursor at x=%d = %d, want %d", tc.x, got, tc.want)
			}
		})
	}
}

func TestCreativeSearchDisplayStartKeepsCaretVisible(t *testing.T) {
	for _, tc := range []struct {
		query string
		caret int
		want  int
	}{
		{"abcdefghijklmnopqrstuvwxyz", 20, 7},
		{"短い検索語", 5, 0},
		{"abcdefghijklmnop", 16, 3},
	} {
		if got := CreativeSearchDisplayStart(tc.query, tc.caret, 80); got != tc.want {
			t.Errorf("display start for %q at caret %d = %d, want %d", tc.query, tc.caret, got, tc.want)
		}
	}
}

func TestCreativeSearchCursorAtPointerAccountsForHorizontalScroll(t *testing.T) {
	width, height := 640, 480
	scale, panelX, _ := creativeGUIOrigin(width, height)
	query := "abcdefghijklmnopqrstuvwxyz"
	if got := CreativeSearchCursorAt(width, height, query, panelX+82*scale, 20); got != 7 {
		t.Fatalf("click at left of scrolled query = %d, want visible start 7", got)
	}
	if got := CreativeSearchCursorAt(width, height, query, panelX+83*scale+5*scale, 20); got != 8 {
		t.Fatalf("click after first visible rune = %d, want 8", got)
	}
}

func TestCreativeTabsMatchJava263VanillaRows(t *testing.T) {
	for _, tc := range []struct {
		category inventory.CreativeCategory
		column   int
		bottom   bool
	}{
		{inventory.CreativeBlocks, 0, false},
		{inventory.CreativeDecoration, 1, false},
		{inventory.CreativeNatural, 2, false},
		{inventory.CreativeFunctional, 3, false},
		{inventory.CreativeRedstone, 4, false},
		{inventory.CreativeHotbar, 5, false},
		{inventory.CreativeAll, 6, false},
		{inventory.CreativeTools, 0, true},
		{inventory.CreativeCombat, 1, true},
		{inventory.CreativeFood, 2, true},
		{inventory.CreativeMaterials, 3, true},
		{inventory.CreativeSpawnEggs, 4, true},
		{inventory.CreativeInventory, 6, true},
	} {
		tab := creativeTabForCategory(tc.category)
		if tab.column != tc.column || tab.bottom != tc.bottom {
			t.Errorf("creative tab %d position = bottom:%v column:%d, want bottom:%v column:%d", tc.category, tab.bottom, tab.column, tc.bottom, tc.column)
		}
	}
	hotbar := creativeTabForCategory(inventory.CreativeHotbar)
	if hotbar.icon != 47 || hotbar.name != "已保存的快捷栏" {
		t.Fatalf("saved hotbars tab appearance = icon %d label %q, want bookshelf icon and vanilla label", hotbar.icon, hotbar.name)
	}
	if got := creativeTabSpriteVariant(hotbar); got != 6 {
		t.Fatalf("saved hotbars sprite variant = %d, want column variant 6", got)
	}

	const width, height = 400, 300
	scale, px, py := creativeGUIOrigin(width, height)
	hit := HitTestCreativeInventory(width, height, inventory.CreativeAll, "", 0,
		px+(creativeTabX(hotbar)+8)*scale, py+(creativeTabY(hotbar)+8)*scale)
	if hit.Type != SlotCreativeTab || hit.Index != int(inventory.CreativeHotbar) {
		t.Fatalf("saved hotbars tab hit = %+v, want top-row saved hotbars tab", hit)
	}
}

func TestCreativeInventoryEquipmentSlotsMatchOfficialPanel(t *testing.T) {
	const width, height = 400, 300
	scale, px, py := creativeGUIOrigin(width, height)
	wantArmorPositions := [...][2]int{{106, 27}, {52, 27}, {106, 0}, {52, 0}}
	for index, position := range wantArmorPositions {
		x := px + (position[0]+8)*scale
		y := py + (position[1]+8)*scale
		hit := HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, x, y)
		if hit.Type != SlotCreativeArmor || hit.Index != index {
			t.Errorf("official armor slot at %v hit = %+v, want armor slot %d", position, hit, index)
		}
	}
	offhandX := px + (creativeOffhandSlotPosition[0]+8)*scale
	offhandY := py + (creativeOffhandSlotPosition[1]+8)*scale
	if hit := HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, offhandX, offhandY); hit.Type != SlotOffhand || hit.Index != 0 {
		t.Fatalf("official offhand slot hit = %+v, want SlotOffhand", hit)
	}

	inv := inventory.NewPlayerInventory()
	inv.Offhand = inventory.ItemStack{ItemID: inventory.ItemDiamond, Count: 1}
	fb := NewFramebuffer(width, height)
	x, y := px+creativeOffhandSlotPosition[0]*scale, py+creativeOffhandSlotPosition[1]*scale
	if name := drawCreativeOffhandSlot(fb, inv, x, y, scale, x+2*scale, y+2*scale); name != "钻石" {
		t.Fatalf("offhand hover label = %q, want diamond display name", name)
	}
}

func TestCreativeTabIconsUseJava263ReferenceItems(t *testing.T) {
	for _, tc := range []struct {
		category  inventory.CreativeCategory
		icon      int32
		texture   string
		blockIcon bool
	}{
		{inventory.CreativeBlocks, 45, "bricks", true},
		{inventory.CreativeDecoration, inventory.ItemCyanWool, "cyan_wool", true},
		{inventory.CreativeFunctional, 323, "oak_sign", false},
		{inventory.CreativeTools, inventory.ItemDiamondPickaxe, "diamond_pickaxe", false},
		{inventory.CreativeCombat, inventory.ItemNetheriteSword, "netherite_sword", false},
		{inventory.CreativeFood, inventory.ItemGoldenApple, "golden_apple", false},
		{inventory.CreativeMaterials, inventory.ItemIronIngot, "iron_ingot", false},
		{inventory.CreativeSpawnEggs, 11002, "creeper_spawn_egg", false},
		{inventory.CreativeHotbar, 47, "bookshelf", true},
		{inventory.CreativeAll, 345, "compass_00", false},
		{inventory.CreativeInventory, 54, "chest", true},
	} {
		tab := creativeTabForCategory(tc.category)
		if tab.icon != tc.icon {
			t.Errorf("creative tab %d icon = %d, want %d", tc.category, tab.icon, tc.icon)
		}
		if tc.category == inventory.CreativeInventory {
			// Chest item rendering still uses the project's generic block sprite;
			// the vanilla chest atlas has not yet been wired into item rendering.
			continue
		} else if tc.blockIcon {
			if _, ok := GetOfficialTexture("block", tc.texture); !ok {
				t.Errorf("creative tab %d block icon %q has no official texture", tc.category, tc.texture)
			}
			sprite := GetItemSprite16x16(tab.icon)
			if sprite[8][8] == 0 {
				t.Errorf("creative tab %d block icon %q rendered empty", tc.category, tc.texture)
			}
		} else {
			if got := GetOfficialItemTextureName(tab.icon); got != tc.texture {
				t.Errorf("creative tab %d texture = %q, want %q", tc.category, got, tc.texture)
			}
			if _, ok := GetOfficialItemSprite(tab.icon); !ok {
				t.Errorf("creative tab %d icon %q has no official item sprite", tc.category, tc.texture)
			}
		}
	}
}

func TestInventoryTabUsesOfficialChestModelTexture(t *testing.T) {
	texture, ok := GetOfficialImage("entity/chest/normal.png")
	if !ok || texture.Width != 64 || texture.Height != 64 {
		t.Fatalf("official chest model texture = (%v, %dx%d), want embedded 64x64 texture", ok, texture.Width, texture.Height)
	}
	got := GetItemSprite16x16(inventory.ItemChest)
	if got == RenderIsometricBlock16x16(inventory.ItemChest) {
		t.Fatal("chest item icon still uses the generic full-cube block renderer")
	}
	colored := 0
	for y := range got {
		for _, pixel := range got[y] {
			if pixel>>24 != 0 {
				colored++
			}
		}
	}
	if colored < 30 {
		t.Fatalf("official chest model icon has only %d nontransparent pixels", colored)
	}
}

func TestCreativeChineseTooltipsUseCJKTextWidth(t *testing.T) {
	label := "工具与实用物品"
	want := MeasureText5x7(label, 2) + 16
	if got := creativeTooltipWidth(label, 2); got != want {
		t.Fatalf("Chinese creative tooltip width = %d, want %d", got, want)
	}
	if MeasureText5x7(label, 1) != 12*len([]rune(label)) {
		t.Fatal("creative tooltip text metrics do not account for 12-pixel CJK glyphs")
	}
}

func TestCreativeSearchTextFitsAndClipsToVanillaField(t *testing.T) {
	fb := NewFramebuffer(96, 40)
	drawCreativeSearchText(fb, 4, 3, "搜索物品以及超长查询", 0xFFFFFFFF, 2, 36)
	visible := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			if fb.Color[y*fb.Width+x] == 0 {
				continue
			}
			if x < 4 || x >= 40 {
				t.Fatalf("search text escaped its 36px field at (%d,%d)", x, y)
			}
			visible++
		}
	}
	if visible == 0 {
		t.Fatal("search text rendered no visible pixels")
	}
}

func TestCreativeInventoryFitsMinimumGUIResolution(t *testing.T) {
	for _, size := range [][2]int{{320, 240}, {400, 300}, {1280, 720}, {1920, 1080}} {
		w, h := size[0], size[1]
		scale, px, py := creativeGUIOrigin(w, h)
		if px < 0 || px+creativePanelW*scale > w {
			t.Errorf("creative panel exceeds framebuffer horizontally at %dx%d: scale=%d x=%d", w, h, scale, px)
		}
		if py-32*scale < 0 {
			t.Errorf("creative top tabs are clipped at %dx%d: scale=%d y=%d", w, h, scale, py-32*scale)
		}
		if py+creativePanelH*scale+32*scale > h {
			t.Errorf("creative bottom tabs are clipped at %dx%d: scale=%d y=%d", w, h, scale, py+creativePanelH*scale+32*scale)
		}
		if py+(creativeHotbarY+18)*scale > h {
			t.Errorf("creative hotbar is clipped at %dx%d: scale=%d y=%d", w, h, scale, py+(creativeHotbarY+18)*scale)
		}
		if w == 320 && h == 240 {
			tab := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+(creativeTabX(creativeTabs[0])+10)*scale, py-20*scale)
			if tab.Type != SlotCreativeTab || tab.Index != int(inventory.CreativeBlocks) {
				t.Errorf("minimum-size visible tab hit = %+v", tab)
			}
			hotbar := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", 0, px+10*scale, py+113*scale)
			if hotbar.Type != SlotInventory || hotbar.Index != 0 {
				t.Errorf("minimum-size hotbar hit = %+v", hotbar)
			}
		}
		fb := NewFramebuffer(w, h)
		fb.DrawCreativeInventoryGUI(inventory.NewPlayerInventory(), inventory.ItemStack{}, inventory.CreativeAll, "", 0, -1, -1)
	}
}

func TestCreativeInventoryScrollUsesRowsAndReachesLastCatalogItem(t *testing.T) {
	for _, test := range []struct {
		items, want int
	}{{0, 0}, {45, 0}, {46, 1}, {54, 1}, {55, 2}} {
		if got := CreativeCatalogMaxScroll(test.items); got != test.want {
			t.Errorf("CreativeCatalogMaxScroll(%d) = %d, want %d", test.items, got, test.want)
		}
	}

	items := inventory.CreativeCatalog(inventory.CreativeAll)
	maxRow := CreativeCatalogMaxScroll(len(items))
	start := maxRow * creativeColumns
	last := len(items) - 1
	if last < start || last-start >= creativePerPage {
		t.Fatalf("last catalog item %d is outside final visible window starting at %d", last, start)
	}

	w, h := 400, 300
	scale, px, py := creativeGUIOrigin(w, h)
	visible := last - start
	row, col := visible/creativeColumns, visible%creativeColumns
	x := px + (10+col*18)*scale
	y := py + (19+row*18)*scale
	hit := HitTestCreativeInventory(w, h, inventory.CreativeAll, "", maxRow, x, y)
	if hit.Type != SlotCreativeItem || hit.Index != last {
		t.Fatalf("last item hit at max scroll = %+v, want catalog index %d", hit, last)
	}
}

func TestCreativeScrollbarThumbDragMapsEveryRow(t *testing.T) {
	for _, name := range []string{"scroller.png", "scroller_disabled.png"} {
		image, ok := GetOfficialImage("gui/sprites/container/creative_inventory/" + name)
		if !ok || image.Width != 12 || image.Height != 15 {
			t.Fatalf("official creative scrollbar %s = (%v, %dx%d), want 12x15", name, ok, image.Width, image.Height)
		}
	}
	for _, size := range [][2]int{{320, 240}, {400, 300}, {1280, 720}, {1920, 1080}} {
		const maxScroll = 37
		width, height := size[0], size[1]
		scale := CalculateGUIScaleFromDimensions(width, height)
		offset := 8 * scale
		thumbTop := CreativeScrollThumbY(width, height, maxScroll, maxScroll)
		_, _, py := creativeGUIOrigin(width, height)
		if thumbTop+15*scale != py+130*scale {
			t.Fatalf("size %dx%d: final scrollbar thumb bottom = %d, want track end %d", width, height, thumbTop+15*scale, py+130*scale)
		}
		for row := 0; row <= maxScroll; row++ {
			mouseY := CreativeScrollThumbY(width, height, row, maxScroll) + offset
			got := CreativeScrollRowAtPointer(width, height, mouseY, maxScroll, offset)
			if got != row {
				t.Fatalf("size %dx%d: dragging thumb at row %d returned %d", width, height, row, got)
			}
		}
	}
}

func TestCreativeInventoryShowsPlayerItemTooltipAndDurability(t *testing.T) {
	const width, height = 400, 300
	scale, px, py := creativeGUIOrigin(width, height)
	mouseX, mouseY := px+10*scale, py+113*scale

	inv := inventory.NewPlayerInventory()
	inv.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 1}
	fb := NewFramebuffer(width, height)
	fb.DrawCreativeInventoryGUI(inv, inventory.ItemStack{}, inventory.CreativeAll, "", 0, mouseX, mouseY)
	tooltipX, tooltipY := mouseX+10*scale, mouseY+10*scale
	if got := fb.Color[tooltipY*width+tooltipX]; got == 0xF0100010 || got == 0 {
		t.Fatalf("player inventory tooltip did not use the official nine-slice art: %#08x", got)
	}

	bar := NewFramebuffer(20, 20)
	drawCreativeDurabilityBar(bar, inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Damage: 30}, 0, 0, 1)
	if got := bar.Color[15*bar.Width+2]; got == 0 || got == 0xFF000000 {
		t.Fatalf("damaged tool durability fill was not rendered: %#08x", got)
	}
	if got := bar.Color[15*bar.Width+12]; got != 0xFF000000 {
		t.Fatalf("durability track background = %#08x, want black", got)
	}
}

func TestCreativeInventoryUsesEmbeddedVanillaGUITextures(t *testing.T) {
	spawnTab := creativeTabForCategory(inventory.CreativeSpawnEggs)
	if _, ok := GetOfficialItemSprite(spawnTab.icon); !ok {
		t.Fatal("creative Spawn Eggs tab icon has no official creeper spawn egg sprite")
	}
	for _, path := range []string{
		"assets/textures/gui/sprites/container/slot.png",
		"assets/textures/gui/sprites/container/creative_inventory/tab_top_selected_1.png",
		"assets/textures/gui/sprites/container/furnace/burn_progress.png",
		"assets/textures/gui/sprites/container/furnace/lit_progress.png",
		"assets/textures/gui/sprites/container/creative_inventory/tab_top_unselected_4.png",
		"assets/textures/gui/sprites/container/creative_inventory/tab_bottom_selected_1.png",
		"assets/textures/gui/sprites/container/creative_inventory/tab_bottom_unselected_5.png",
		"assets/textures/gui/container/inventory.png",
		"assets/textures/gui/container/crafting_table.png",
		"assets/textures/gui/container/furnace.png",
		"assets/textures/gui/container/creative_inventory/tab_items.png",
		"assets/textures/gui/container/creative_inventory/tab_item_search.png",
		"assets/textures/gui/container/creative_inventory/tab_inventory.png",
		"assets/textures/gui/sprites/tooltip/background.png",
		"assets/textures/gui/sprites/tooltip/background.png.mcmeta",
		"assets/textures/gui/sprites/tooltip/frame.png",
		"assets/textures/gui/sprites/tooltip/frame.png.mcmeta",
	} {
		if _, err := EmbeddedTextures.ReadFile(path); err != nil {
			t.Errorf("missing embedded vanilla GUI texture %s: %v", path, err)
		}
	}
	for _, path := range []string{"gui/sprites/tooltip/background.png", "gui/sprites/tooltip/frame.png"} {
		if image, ok := GetOfficialImage(path); !ok || image.Width != 100 || image.Height != 100 {
			t.Errorf("official tooltip sprite %s = (%v, %t), want 100x100", path, image, ok)
		}
	}
	for _, path := range []string{"gui/container/inventory.png", "gui/container/crafting_table.png", "gui/container/furnace.png"} {
		if image, ok := GetOfficialImage(path); !ok || image.Width != 256 || image.Height != 256 {
			t.Errorf("official container texture %s = (%v, %t), want 256x256", path, image, ok)
		}
	}
	for path, want := range map[string][2]int{
		"gui/sprites/container/furnace/burn_progress.png": {24, 16},
		"gui/sprites/container/furnace/lit_progress.png":  {14, 14},
	} {
		if image, ok := GetOfficialImage(path); !ok || image.Width != want[0] || image.Height != want[1] {
			t.Errorf("official furnace sprite %s = (%v, %t), want %dx%d", path, image, ok, want[0], want[1])
		}
	}
}

func TestCreativePlayerInventoryPreviewRendersAndTracksMouse(t *testing.T) {
	skin, ok := GetOfficialImage("entity/player/steve.png")
	if !ok || skin.Width != 64 || skin.Height != 64 {
		t.Fatalf("embedded official Steve skin = (%v,%t), want a 64x64 texture", skin, ok)
	}
	const width, height = 160, 120
	const centerX, topY, scale = 80, 20, 1
	left := NewFramebuffer(width, height)
	right := NewFramebuffer(width, height)
	vertical := NewFramebuffer(width, height)
	drawCreativePlayerPreview(left, nil, centerX, topY, scale, centerX-20, topY+22)
	drawCreativePlayerPreview(right, nil, centerX, topY, scale, centerX+20, topY+22)
	drawCreativePlayerPreview(vertical, nil, centerX, topY, scale, centerX, topY+40)

	colored := 0
	different := false
	verticalDifferent := false
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			if left.Color[index] != 0 {
				if x < centerX-24*scale || x >= centerX+24*scale || y < topY || y >= topY+45*scale {
					t.Fatalf("preview painted outside its viewport at (%d,%d)", x, y)
				}
				colored++
			}
			if left.Color[index] != right.Color[index] {
				different = true
			}
			if left.Color[index] != vertical.Color[index] {
				verticalDifferent = true
			}
		}
	}
	if colored < 80 {
		t.Fatalf("player preview rendered only %d pixels", colored)
	}
	if !different {
		t.Fatal("player preview did not rotate with the mouse position")
	}
	if !verticalDifferent {
		t.Fatal("player preview did not pitch with the vertical mouse position")
	}
}

func TestCreativePlayerPreviewRendersOfficialArmorMaterials(t *testing.T) {
	for _, material := range []string{"chainmail", "copper", "diamond", "gold", "iron", "leather", "netherite"} {
		for _, layer := range []string{"humanoid", "humanoid_leggings"} {
			texture, ok := GetOfficialImage("entity/equipment/" + layer + "/" + material + ".png")
			if !ok || texture.Width != 64 || texture.Height != 32 {
				t.Errorf("official %s %s texture = (%v,%t), want 64x32", material, layer, texture, ok)
			}
		}
	}
	const width, height = 160, 120
	const centerX, topY, scale = 80, 20, 1
	naked := NewFramebuffer(width, height)
	armored := NewFramebuffer(width, height)
	drawCreativePlayerPreview(naked, nil, centerX, topY, scale, centerX, topY+22)
	inv := &inventory.PlayerInventory{}
	for slot, itemID := range []int32{313, 312, 311, 310} { // boots, leggings, chestplate, helmet
		inv.Armor[slot] = inventory.ItemStack{ItemID: itemID, Count: 1}
	}
	drawCreativePlayerPreview(armored, inv, centerX, topY, scale, centerX, topY+22)
	different := false
	for i := range naked.Color {
		if naked.Color[i] != armored.Color[i] {
			different = true
			break
		}
	}
	if !different {
		t.Fatal("equipped diamond armor did not change the player inventory preview")
	}
}

func TestCreativePlayerSkinLoadsUserConfigAndNormalizesLegacyDimensions(t *testing.T) {
	configDir := t.TempDir()
	skinPath := filepath.Join(configDir, "minecraft-go", "skin.png")
	if err := os.MkdirAll(filepath.Dir(skinPath), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	img.SetRGBA(12, 12, color.RGBA{R: 20, G: 90, B: 180, A: 255})
	file, err := os.Create(skinPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	texture, err := loadCreativePlayerSkinFromConfig(configDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := texture.Sample(12.0/63, 12.0/63); got != 0xFF145AB4 {
		t.Fatalf("custom skin pixel = %#08x, want %#08x", got, uint32(0xFF145AB4))
	}
	legacy := image.NewRGBA(image.Rect(0, 0, 64, 32))
	legacy.SetRGBA(42, 18, color.RGBA{R: 220, G: 40, B: 60, A: 255})
	file, err = os.Create(skinPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, legacy); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	legacyTexture, err := loadCreativePlayerSkinFromConfig(configDir)
	if err != nil || legacyTexture.Width != 64 || legacyTexture.Height != 64 {
		t.Fatalf("legacy skin normalization = (%v,%v), want 64x64", legacyTexture, err)
	}
	if got := legacyTexture.Sample(34.0/63, 50.0/63); got != 0xFFDC283C {
		t.Fatalf("mirrored left arm skin pixel = %#08x, want %#08x", got, uint32(0xFFDC283C))
	}
	invalid := image.NewRGBA(image.Rect(0, 0, 64, 30))
	file, err = os.Create(skinPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, invalid); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCreativePlayerSkinFromConfig(configDir); err == nil {
		t.Fatal("accepted a skin with unsupported dimensions")
	}
}

func TestCreativeSpecialTabsRenderAndHitPlayerInventory(t *testing.T) {
	const width, height = 400, 300
	scale, px, py := creativeGUIOrigin(width, height)
	inv := inventory.NewPlayerInventory()
	presets := inventory.CreativeHotbarPresets{}
	presets[0][0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 64}

	hotbarTab := creativeTabForCategory(inventory.CreativeHotbar)
	hit := HitTestCreativeInventory(width, height, inventory.CreativeAll, "", 0,
		px+(creativeTabX(hotbarTab)+8)*scale, py+(creativeTabY(hotbarTab)+8)*scale)
	if hit.Type != SlotCreativeTab || hit.Index != int(inventory.CreativeHotbar) {
		t.Fatalf("hotbar preset tab hit = %+v", hit)
	}
	hit = HitTestCreativeInventory(width, height, inventory.CreativeHotbar, "", 0, px+10*scale, py+19*scale, presets)
	if hit.Type != SlotCreativeItem || hit.Index != 0 {
		t.Fatalf("first hotbar preset item hit = %+v", hit)
	}
	if got := len(creativeHotbarItems(presets)); got != 81 {
		t.Fatalf("hotbar preset tab has %d cells, want 81", got)
	}

	invTab := creativeTabForCategory(inventory.CreativeInventory)
	hit = HitTestCreativeInventory(width, height, inventory.CreativeAll, "", 0,
		px+(creativeTabX(invTab)+8)*scale, py+(creativeTabY(invTab)+8)*scale)
	if hit.Type != SlotCreativeTab || hit.Index != int(inventory.CreativeInventory) {
		t.Fatalf("player inventory tab hit = %+v", hit)
	}
	hit = HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, px+10*scale, py+55*scale)
	if hit.Type != SlotInventory || hit.Index != 9 {
		t.Fatalf("first main inventory slot hit = %+v, want slot 9", hit)
	}
	hit = HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, px+(creativeArmorSlotPositions[0][0]+8)*scale, py+(creativeArmorSlotPositions[0][1]+8)*scale)
	if hit.Type != SlotCreativeArmor || hit.Index != 0 {
		t.Fatalf("boots armor slot hit = %+v", hit)
	}
	hit = HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, px+174*scale, py+113*scale)
	if hit.Type != SlotCreativeTrash {
		t.Fatalf("trash slot hit = %+v", hit)
	}

	fb := NewFramebuffer(width, height)
	fb.DrawCreativeInventoryGUI(inv, inventory.ItemStack{}, inventory.CreativeHotbar, "", 0, -1, -1, presets)
	fb.DrawCreativeInventoryGUI(inv, inventory.ItemStack{}, inventory.CreativeInventory, "", 0, -1, -1, presets)
}

func TestCreativeTabHitboxesMatchOfficialSpriteBounds(t *testing.T) {
	const width, height = 400, 300
	scale, px, py := creativeGUIOrigin(width, height)
	for _, category := range []inventory.CreativeCategory{inventory.CreativeBlocks, inventory.CreativeInventory} {
		tab := creativeTabForCategory(category)
		x := px + (creativeTabX(tab)+13)*scale
		for _, offset := range []int{0, 31} {
			y := py + (creativeTabY(tab)+offset)*scale
			hit := HitTestCreativeInventory(width, height, inventory.CreativeAll, "", 0, x, y)
			if hit.Type != SlotCreativeTab || hit.Index != int(category) {
				t.Errorf("tab %d sprite-edge hit at y offset %d = %+v", category, offset, hit)
			}
		}
	}
}

func creativeTabForCategory(category inventory.CreativeCategory) creativeTabLayout {
	for _, tab := range creativeTabs {
		if tab.category == category {
			return tab
		}
	}
	return creativeTabLayout{}
}

func TestVisualScreenRenders(t *testing.T) {
	fb := NewFramebuffer(400, 300)
	worlds := []WorldInfo{
		{DirName: "BENCHMARK_WORLD", LevelName: "BENCHMARK", GameMode: 0, LastTime: time.Now()},
	}
	fb.DrawWorldSelectScreen(worlds, 0, 200, 280)

	// Verify framebuffer is not blank
	nonZero := 0
	for _, c := range fb.Color {
		if c != 0 {
			nonZero++
		}
	}
	if nonZero < 1000 {
		t.Fatalf("framebuffer has too few rendered pixels: %d", nonZero)
	}

	// 2. Render Title Screen
	fbTitle := NewFramebuffer(400, 300)
	fbTitle.DrawTitleScreen(200, 160)
	nonZeroTitle := 0
	for _, c := range fbTitle.Color {
		if c != 0 {
			nonZeroTitle++
		}
	}
	if nonZeroTitle < 1000 {
		t.Fatalf("title screen framebuffer has too few rendered pixels: %d", nonZeroTitle)
	}

	// 3. Render New World Screen
	fbNew := NewFramebuffer(400, 300)
	fbNew.DrawNewWorldScreen("新世界", 0, "", 0, 200, 200)
	nonZeroNew := 0
	for _, c := range fbNew.Color {
		if c != 0 {
			nonZeroNew++
		}
	}
	if nonZeroNew < 1000 {
		t.Fatalf("new world screen framebuffer has too few rendered pixels: %d", nonZeroNew)
	}
}
