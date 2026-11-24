package render

import (
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
	hitCraft := HitTestCraftingGUI(320, 240, gx+40+2, gy+22+2, false)
	if hitCraft.Type != SlotCraftGrid || hitCraft.Index != 0 {
		t.Fatalf("expected craft grid slot 0, got %+v", hitCraft)
	}

	// Hit inside 2x2 result slot
	hitRes := HitTestCraftingGUI(320, 240, gx+124+2, gy+31+2, false)
	if hitRes.Type != SlotCraftResult {
		t.Fatalf("expected craft result slot, got %+v", hitRes)
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
	if got := inventory.CreativeCatalog(inventory.CreativeSpawnEggs); len(got) != len(inventory.SpawnEggDefinitions()) || len(got) == 0 {
		t.Fatalf("spawn-egg tab has %d items, want %d supported mobs", len(got), len(inventory.SpawnEggDefinitions()))
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
	for _, size := range [][2]int{{320, 240}, {400, 300}, {1280, 720}, {1920, 1080}} {
		const maxScroll = 37
		width, height := size[0], size[1]
		scale := CalculateGUIScaleFromDimensions(width, height)
		offset := 8 * scale
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
	if got := fb.Color[tooltipY*width+tooltipX]; got != 0xF0100010 {
		t.Fatalf("player inventory item tooltip background = %#08x, want %#08x", got, uint32(0xF0100010))
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
		"assets/textures/gui/sprites/container/creative_inventory/tab_top_unselected_4.png",
		"assets/textures/gui/sprites/container/creative_inventory/tab_bottom_selected_1.png",
		"assets/textures/gui/sprites/container/creative_inventory/tab_bottom_unselected_5.png",
		"assets/textures/gui/container/inventory.png",
		"assets/textures/gui/container/creative_inventory/tab_items.png",
		"assets/textures/gui/container/creative_inventory/tab_item_search.png",
		"assets/textures/gui/container/creative_inventory/tab_inventory.png",
	} {
		if _, err := EmbeddedTextures.ReadFile(path); err != nil {
			t.Errorf("missing embedded vanilla GUI texture %s: %v", path, err)
		}
	}
}

func TestCreativePlayerInventoryPreviewRendersAndTracksMouse(t *testing.T) {
	const width, height = 160, 120
	const centerX, topY, scale = 80, 20, 1
	left := NewFramebuffer(width, height)
	right := NewFramebuffer(width, height)
	drawCreativePlayerPreview(left, centerX, topY, scale, centerX-20, topY+22)
	drawCreativePlayerPreview(right, centerX, topY, scale, centerX+20, topY+22)

	colored := 0
	different := false
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
		}
	}
	if colored < 80 {
		t.Fatalf("player preview rendered only %d pixels", colored)
	}
	if !different {
		t.Fatal("player preview did not rotate with the mouse position")
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
		px+(creativeTabX(hotbarTab)+8)*scale, py+(creativeTabY(hotbarTab)-4+8)*scale)
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
		px+(creativeTabX(invTab)+8)*scale, py+(creativeTabY(invTab)-4+8)*scale)
	if hit.Type != SlotCreativeTab || hit.Index != int(inventory.CreativeInventory) {
		t.Fatalf("player inventory tab hit = %+v", hit)
	}
	hit = HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, px+10*scale, py+55*scale)
	if hit.Type != SlotInventory || hit.Index != 9 {
		t.Fatalf("first main inventory slot hit = %+v, want slot 9", hit)
	}
	hit = HitTestCreativeInventory(width, height, inventory.CreativeInventory, "", 0, px+55*scale, py+7*scale)
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
