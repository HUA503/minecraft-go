package inventory

// ExtendedBlockDefinition provides namespaced identity for recently added
// vanilla blocks while the rest of the game is migrated off legacy numeric IDs.
// RuntimeID is an internal save/protocol value and is not a Mojang registry ID.
type ExtendedBlockDefinition struct {
	RuntimeID            int32
	Name                 string
	TextureTop           string
	TextureBottom        string
	TextureSide          string
	Category             CreativeCategory
	Hardness             float64
	HarvestTool          ToolType
	RequiresHarvestTool  bool
	RequiredHarvestLevel int
	DropRuntimeID        int32
	DropsNothing         bool
	DropCountMin         int
	DropCountMax         int
	StrippedVariantID    int32
}

var modernOreDropRanges = map[string][2]int{
	"minecraft:copper_ore":             {2, 5},
	"minecraft:deepslate_copper_ore":   {2, 5},
	"minecraft:deepslate_lapis_ore":    {4, 9},
	"minecraft:deepslate_redstone_ore": {4, 5},
}

var extendedBlockDefinitions = buildExtendedBlockDefinitions()

const (
	FirstCushionBlock int32 = 16200
	LastCushionBlock  int32 = FirstCushionBlock + 15
	FirstCandleItem   int32 = 17133
	LastCandleItem    int32 = FirstCandleItem + 16
	CandleStateBase   int32 = 40000
)

var cushionColors = [...]struct {
	name, display string
}{
	{"white", "\u767d\u8272"}, {"orange", "\u6a59\u8272"}, {"magenta", "\u54c1\u7ea2\u8272"}, {"light_blue", "\u6d45\u84dd\u8272"},
	{"yellow", "\u9ec4\u8272"}, {"lime", "\u9ec4\u7eff\u8272"}, {"pink", "\u7c89\u8272"}, {"gray", "\u7070\u8272"},
	{"light_gray", "\u6d45\u7070\u8272"}, {"cyan", "\u9752\u8272"}, {"purple", "\u7d2b\u8272"}, {"blue", "\u84dd\u8272"},
	{"brown", "\u68d5\u8272"}, {"green", "\u7eff\u8272"}, {"red", "\u7ea2\u8272"}, {"black", "\u9ed1\u8272"},
}

func IsCushionBlock(runtimeID int32) bool {
	return runtimeID >= FirstCushionBlock && runtimeID <= LastCushionBlock
}

func CushionItemID(colorIndex int) (int32, bool) {
	if colorIndex < 0 || colorIndex >= len(cushionColors) {
		return 0, false
	}
	return FirstCushionBlock + int32(colorIndex), true
}

func buildExtendedBlockDefinitions() []ExtendedBlockDefinition {
	definitions := []ExtendedBlockDefinition{
		baseExtendedBlock(10000, "minecraft:cinnabar", "cinnabar", "cinnabar", "cinnabar", CreativeBlocks),
		baseExtendedBlock(10001, "minecraft:polished_cinnabar", "polished_cinnabar", "polished_cinnabar", "polished_cinnabar", CreativeBlocks),
		baseExtendedBlock(10002, "minecraft:chiseled_cinnabar", "chiseled_cinnabar", "chiseled_cinnabar", "chiseled_cinnabar", CreativeBlocks),
		baseExtendedBlock(10003, "minecraft:cinnabar_bricks", "cinnabar_bricks", "cinnabar_bricks", "cinnabar_bricks", CreativeBlocks),
		baseExtendedBlock(10004, "minecraft:sulfur", "sulfur", "sulfur", "sulfur", CreativeBlocks),
		baseExtendedBlock(10005, "minecraft:polished_sulfur", "polished_sulfur", "polished_sulfur", "polished_sulfur", CreativeBlocks),
		baseExtendedBlock(10006, "minecraft:chiseled_sulfur", "chiseled_sulfur", "chiseled_sulfur", "chiseled_sulfur", CreativeBlocks),
		baseExtendedBlock(10007, "minecraft:sulfur_bricks", "sulfur_bricks", "sulfur_bricks", "sulfur_bricks", CreativeBlocks),
		baseExtendedBlock(10008, "minecraft:potent_sulfur", "potent_sulfur", "potent_sulfur", "potent_sulfur", CreativeBlocks),
		baseExtendedBlock(10009, "minecraft:poplar_planks", "poplar_planks", "poplar_planks", "poplar_planks", CreativeBlocks),
		{RuntimeID: 10010, Name: "minecraft:poplar_log", TextureTop: "poplar_log_top", TextureBottom: "poplar_log_top", TextureSide: "poplar_log", Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe, StrippedVariantID: 10011},
		baseExtendedBlock(10011, "minecraft:stripped_poplar_log", "stripped_poplar_log_top", "stripped_poplar_log_top", "stripped_poplar_log", CreativeBlocks),
		{RuntimeID: 10101, Name: "minecraft:poplar_wood", TextureTop: "poplar_log", TextureBottom: "poplar_log", TextureSide: "poplar_log", Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe, StrippedVariantID: 10102},
		{RuntimeID: 10102, Name: "minecraft:stripped_poplar_wood", TextureTop: "stripped_poplar_log", TextureBottom: "stripped_poplar_log", TextureSide: "stripped_poplar_log", Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe},
		baseExtendedBlock(10060, "minecraft:red_poplar_leaves", "red_poplar_leaves", "red_poplar_leaves", "red_poplar_leaves", CreativeNatural),
		baseExtendedBlock(10061, "minecraft:orange_poplar_leaves", "orange_poplar_leaves", "orange_poplar_leaves", "orange_poplar_leaves", CreativeNatural),
		baseExtendedBlock(10062, "minecraft:yellow_poplar_leaves", "yellow_poplar_leaves", "yellow_poplar_leaves", "yellow_poplar_leaves", CreativeNatural),
		baseExtendedBlock(10063, "minecraft:poplar_sapling", "poplar_sapling", "poplar_sapling", "poplar_sapling", CreativeNatural),
		{RuntimeID: 10100, Name: "minecraft:red_shrub", TextureTop: "red_shrub", TextureBottom: "red_shrub", TextureSide: "red_shrub", Category: CreativeNatural, Hardness: 0.1, DropRuntimeID: 10100},
	}
	definitions = append(definitions, modernFullCubeDefinitions()...)
	definitions = append(definitions, woodCubeDefinitions()...)
	definitions = append(definitions, coloredBlockDefinitions()...)
	definitions = append(definitions, stainedGlassBlockDefinitions()...)
	definitions = append(definitions, glassPaneBlockDefinitions()...)
	definitions = append(definitions, candleBlockDefinitions()...)
	definitions = append(definitions, glazedTerracottaDefinitions()...)
	definitions = append(definitions, additionalFullCubeDefinitions()...)
	definitions = append(definitions, additionalCommonCubeDefinitions()...)
	definitions = append(definitions, additionalSpecialCubeDefinitions()...)
	definitions = append(definitions, woodSlabBlockDefinitions()...)
	definitions = append(definitions, colorSlabBlockDefinitions()...)
	definitions = append(definitions, colorStairBlockDefinitions()...)
	definitions = append(definitions, woodStairBlockDefinitions()...)
	definitions = append(definitions, masonrySlabBlockDefinitions()...)
	definitions = append(definitions, masonryStairBlockDefinitions()...)
	definitions = append(definitions, woodFenceBlockDefinitions()...)
	definitions = append(definitions, netherBrickFenceBlockDefinition())
	definitions = append(definitions, woodFenceGateBlockDefinitions()...)
	definitions = append(definitions, woodDoorBlockDefinitions()...)
	definitions = append(definitions, woodTrapdoorBlockDefinitions()...)
	definitions = append(definitions, ironTrapdoorDefinition())
	definitions = append(definitions, ironChainDefinition())
	definitions = append(definitions, ControlBlockDefinitions()...)
	definitions = append(definitions, masonryWallBlockDefinitions()...)
	definitions = append(definitions, cushionBlockDefinitions()...)
	return definitions
}

func additionalSpecialCubeDefinitions() []ExtendedBlockDefinition {
	blocks := []struct {
		id                int32
		name, display     string
		top, bottom, side string
		category          CreativeCategory
		hardness          float64
		tool              ToolType
		requiresTool      bool
		requiredTier      int
		drop              int32
		minDrop           int
		maxDrop           int
	}{
		{14200, "crying_obsidian", "哭泣的黑曜石", "crying_obsidian", "crying_obsidian", "crying_obsidian", CreativeBlocks, 50, ToolPickaxe, true, 3, 14200, 1, 1},
		{14201, "crimson_nylium", "绯红菌岩", "crimson_nylium", "netherrack", "crimson_nylium_side", CreativeNatural, 0.4, ToolNone, false, 0, 87, 1, 1},
		{14202, "warped_nylium", "诡异菌岩", "warped_nylium", "netherrack", "warped_nylium_side", CreativeNatural, 0.4, ToolNone, false, 0, 87, 1, 1},
		{14203, "warped_wart_block", "诡异疣块", "warped_wart_block", "warped_wart_block", "warped_wart_block", CreativeNatural, 1, ToolHoe, false, 0, 14203, 1, 1},
		{14204, "nether_gold_ore", "下界金矿石", "nether_gold_ore", "nether_gold_ore", "nether_gold_ore", CreativeNatural, 3, ToolPickaxe, true, 0, ItemGoldNugget, 2, 6},
		{14205, "podzol", "灰化土", "podzol_top", "dirt", "podzol_side", CreativeNatural, 0.5, ToolShovel, false, 0, ItemDirt, 1, 1},
		{14206, "mycelium", "菌丝", "mycelium_top", "dirt", "mycelium_side", CreativeNatural, 0.6, ToolShovel, false, 0, ItemDirt, 1, 1},
		{14207, "magma_block", "岩浆块", "magma", "magma", "magma", CreativeNatural, 0.5, ToolPickaxe, false, 0, 14207, 1, 1},
		{14208, "dried_kelp_block", "干海带块", "dried_kelp_top", "dried_kelp_bottom", "dried_kelp_side", CreativeNatural, 0.5, ToolHoe, false, 0, 14208, 1, 1},
		{14209, "tube_coral_block", "管珊瑚块", "tube_coral_block", "tube_coral_block", "tube_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14214, 1, 1},
		{14210, "brain_coral_block", "脑纹珊瑚块", "brain_coral_block", "brain_coral_block", "brain_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14215, 1, 1},
		{14211, "bubble_coral_block", "气泡珊瑚块", "bubble_coral_block", "bubble_coral_block", "bubble_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14216, 1, 1},
		{14212, "fire_coral_block", "火珊瑚块", "fire_coral_block", "fire_coral_block", "fire_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14217, 1, 1},
		{14213, "horn_coral_block", "鹿角珊瑚块", "horn_coral_block", "horn_coral_block", "horn_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14218, 1, 1},
		{14214, "dead_tube_coral_block", "死管珊瑚块", "dead_tube_coral_block", "dead_tube_coral_block", "dead_tube_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14214, 1, 1},
		{14215, "dead_brain_coral_block", "死脑纹珊瑚块", "dead_brain_coral_block", "dead_brain_coral_block", "dead_brain_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14215, 1, 1},
		{14216, "dead_bubble_coral_block", "死气泡珊瑚块", "dead_bubble_coral_block", "dead_bubble_coral_block", "dead_bubble_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14216, 1, 1},
		{14217, "dead_fire_coral_block", "死火珊瑚块", "dead_fire_coral_block", "dead_fire_coral_block", "dead_fire_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14217, 1, 1},
		{14218, "dead_horn_coral_block", "死鹿角珊瑚块", "dead_horn_coral_block", "dead_horn_coral_block", "dead_horn_coral_block", CreativeNatural, 1.5, ToolPickaxe, false, 0, 14218, 1, 1},
		{14219, "ochre_froglight", "赭黄蛙明灯", "ochre_froglight_top", "ochre_froglight_top", "ochre_froglight_side", CreativeNatural, 0.3, ToolNone, false, 0, 14219, 1, 1},
		{14220, "verdant_froglight", "青翠蛙明灯", "verdant_froglight_top", "verdant_froglight_top", "verdant_froglight_side", CreativeNatural, 0.3, ToolNone, false, 0, 14220, 1, 1},
		{14221, "pearlescent_froglight", "珠光蛙明灯", "pearlescent_froglight_top", "pearlescent_froglight_top", "pearlescent_froglight_side", CreativeNatural, 0.3, ToolNone, false, 0, 14221, 1, 1},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(blocks))
	for _, block := range blocks {
		name := "minecraft:" + block.name
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: block.id, Name: name,
			TextureTop: block.top, TextureBottom: block.bottom, TextureSide: block.side,
			Category: block.category, Hardness: block.hardness, HarvestTool: block.tool,
			RequiresHarvestTool: block.requiresTool, RequiredHarvestLevel: block.requiredTier,
			DropRuntimeID: block.drop, DropCountMin: block.minDrop, DropCountMax: block.maxDrop,
		})
		extendedBlockDisplayNames[name] = block.display
	}
	name := "minecraft:reinforced_deepslate"
	definitions = append(definitions, ExtendedBlockDefinition{
		RuntimeID: 14222, Name: name,
		TextureTop: "reinforced_deepslate_top", TextureBottom: "reinforced_deepslate_bottom", TextureSide: "reinforced_deepslate_side",
		Category: CreativeBlocks, Hardness: 55, DropsNothing: true,
	})
	extendedBlockDisplayNames[name] = "强化深板岩"
	return definitions
}

func cushionBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(cushionColors))
	for i, color := range cushionColors {
		name := "minecraft:" + color.name + "_cushion"
		texture := color.name + "_cushion"
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: FirstCushionBlock + int32(i), Name: name,
			TextureTop: texture, TextureBottom: texture, TextureSide: texture,
			Category: CreativeDecoration, Hardness: 0.2, DropRuntimeID: FirstCushionBlock + int32(i),
		})
		extendedBlockDisplayNames[name] = color.display + "\u5750\u57ab"
	}
	return definitions
}

func colorSlabBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(colorSlabDefinitions))
	for _, slab := range colorSlabDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: slab.ItemID, Name: slab.Name,
			TextureTop: slab.Texture, TextureBottom: slab.Texture, TextureSide: slab.Texture,
			Category: CreativeDecoration, Hardness: slab.Hardness,
			HarvestTool: slab.HarvestTool, RequiresHarvestTool: slab.RequiresTool,
		})
	}
	return definitions
}

func masonrySlabBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(masonrySlabDefinitions))
	for _, slab := range masonrySlabDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: slab.ItemID, Name: slab.Name,
			TextureTop: slab.TextureTop, TextureBottom: slab.TextureBottom, TextureSide: slab.TextureSide,
			Category: CreativeBlocks, Hardness: slab.Hardness, HarvestTool: ToolPickaxe,
			RequiresHarvestTool: true,
		})
	}
	return definitions
}

func woodSlabBlockDefinitions() []ExtendedBlockDefinition {
	definitions := make([]ExtendedBlockDefinition, 0, len(woodSlabDefinitions))
	for _, slab := range woodSlabDefinitions {
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: slab.ItemID, Name: slab.Name,
			TextureTop: slab.Texture, TextureBottom: slab.Texture, TextureSide: slab.Texture,
			Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe,
		})
	}
	return definitions
}

func baseExtendedBlock(id int32, name, top, bottom, side string, category CreativeCategory) ExtendedBlockDefinition {
	return ExtendedBlockDefinition{
		RuntimeID: id, Name: name, TextureTop: top, TextureBottom: bottom,
		TextureSide: side, Category: category,
	}
}

// additionalFullCubeDefinitions adds vanilla 26.3 full-cube blocks that use
// ordinary placement and self-drop rules. Stateful and special-drop blocks
// stay out of this table until their behavior can be represented accurately.
func additionalFullCubeDefinitions() []ExtendedBlockDefinition {
	type spec struct {
		name, display string
		hardness      float64
		tool          ToolType
		category      CreativeCategory
	}
	blocks := [...]spec{
		{"amethyst_block", "紫水晶块", 1.5, ToolPickaxe, CreativeBlocks},
		{"bamboo_mosaic", "竹马赛克", 2, ToolAxe, CreativeBlocks},
		{"chiseled_nether_bricks", "錾制下界砖块", 2, ToolPickaxe, CreativeBlocks},
		{"chiseled_polished_blackstone", "錾制磨制黑石", 1.5, ToolPickaxe, CreativeBlocks},
		{"chiseled_stone_bricks", "錾制石砖", 1.5, ToolPickaxe, CreativeBlocks},
		{"cracked_nether_bricks", "裂纹下界砖块", 2, ToolPickaxe, CreativeBlocks},
		{"cracked_stone_bricks", "裂纹石砖", 1.5, ToolPickaxe, CreativeBlocks},
		{"dark_prismarine", "暗海晶石", 1.5, ToolPickaxe, CreativeBlocks},
		{"mossy_stone_bricks", "苔石砖", 1.5, ToolPickaxe, CreativeBlocks},
		{"nether_wart_block", "下界疣块", 1, ToolHoe, CreativeNatural},
		{"netherite_block", "下界合金块", 50, ToolPickaxe, CreativeBlocks},
		{"pale_moss_block", "苍白苔藓块", 0.1, ToolHoe, CreativeNatural},
		{"prismarine", "海晶石", 1.5, ToolPickaxe, CreativeBlocks},
		{"prismarine_bricks", "海晶石砖", 1.5, ToolPickaxe, CreativeBlocks},
		{"purpur_block", "紫珀块", 1.5, ToolPickaxe, CreativeBlocks},
		{"quartz_bricks", "石英砖", 0.8, ToolPickaxe, CreativeBlocks},
		{"red_nether_bricks", "红色下界砖块", 2, ToolPickaxe, CreativeBlocks},
		{"resin_block", "树脂块", 5, ToolNone, CreativeBlocks},
		{"resin_bricks", "树脂砖", 1.5, ToolPickaxe, CreativeBlocks},
		{"chiseled_resin_bricks", "錾制树脂砖", 1.5, ToolPickaxe, CreativeBlocks},
		{"lapis_block", "青金石块", 3, ToolPickaxe, CreativeBlocks},
		{"cut_sandstone", "切制砂岩", 0.8, ToolPickaxe, CreativeBlocks},
		{"cut_red_sandstone", "切制红砂岩", 0.8, ToolPickaxe, CreativeBlocks},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(blocks))
	for i, block := range blocks {
		id := int32(14000 + i)
		texture := block.name
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: id, Name: "minecraft:" + block.name,
			TextureTop: texture, TextureBottom: texture, TextureSide: texture,
			Category: block.category, Hardness: block.hardness,
			HarvestTool: block.tool, RequiresHarvestTool: block.name == "netherite_block" || block.name == "lapis_block" || block.name == "cut_sandstone" || block.name == "cut_red_sandstone",
			DropRuntimeID: id, DropCountMin: 1, DropCountMax: 1,
		})
		if block.name == "netherite_block" {
			definitions[len(definitions)-1].RequiredHarvestLevel = 3 // The official loot table requires a diamond-tier tool.
		} else if block.name == "lapis_block" {
			definitions[len(definitions)-1].RequiredHarvestLevel = 1 // Lapis blocks require a stone-tier pickaxe for drops.
		}
		extendedBlockDisplayNames["minecraft:"+block.name] = block.display
	}
	return definitions
}

// additionalCommonCubeDefinitions uses the 26.3 block-model face textures for
// another group of ordinary cube blocks. Pillars retain the vanilla vertical
// end texture; block-axis state rotation is handled separately from this list.
func additionalCommonCubeDefinitions() []ExtendedBlockDefinition {
	type spec struct {
		name, display, top, bottom, side string
		hardness                         float64
		tool                             ToolType
		category                         CreativeCategory
	}
	blocks := [...]spec{
		{"bone_block", "骨块", "bone_block_top", "bone_block_top", "bone_block_side", 2, ToolPickaxe, CreativeBlocks},
		{"chiseled_quartz_block", "錾制石英块", "chiseled_quartz_block_top", "chiseled_quartz_block_top", "chiseled_quartz_block", 0.8, ToolPickaxe, CreativeBlocks},
		{"chiseled_red_sandstone", "錾制红砂岩", "red_sandstone_top", "red_sandstone_top", "chiseled_red_sandstone", 0.8, ToolPickaxe, CreativeBlocks},
		{"chiseled_sandstone", "錾制砂岩", "sandstone_top", "sandstone_top", "chiseled_sandstone", 0.8, ToolPickaxe, CreativeBlocks},
		{"coarse_dirt", "砂土", "coarse_dirt", "coarse_dirt", "coarse_dirt", 0.5, ToolShovel, CreativeNatural},
		{"end_stone_bricks", "末地石砖", "end_stone_bricks", "end_stone_bricks", "end_stone_bricks", 3, ToolPickaxe, CreativeBlocks},
		{"honeycomb_block", "蜂巢块", "honeycomb_block", "honeycomb_block", "honeycomb_block", 0.6, ToolNone, CreativeBlocks},
		{"lodestone", "磁石", "lodestone_top", "lodestone_top", "lodestone_side", 3.5, ToolPickaxe, CreativeFunctional},
		{"muddy_mangrove_roots", "沾泥的红树根", "muddy_mangrove_roots_top", "muddy_mangrove_roots_top", "muddy_mangrove_roots_side", 1, ToolShovel, CreativeNatural},
		{"purpur_pillar", "紫珀柱", "purpur_pillar_top", "purpur_pillar_top", "purpur_pillar_side", 1.5, ToolPickaxe, CreativeBlocks},
		{"quartz_pillar", "石英柱", "quartz_pillar_top", "quartz_pillar_top", "quartz_pillar_side", 0.8, ToolPickaxe, CreativeBlocks},
		{"rooted_dirt", "缠根泥土", "rooted_dirt", "rooted_dirt", "rooted_dirt", 0.5, ToolShovel, CreativeNatural},
		{"shroomlight", "菌光体", "shroomlight", "shroomlight", "shroomlight", 1, ToolHoe, CreativeNatural},
		{"smooth_quartz", "平滑石英块", "quartz_block_bottom", "quartz_block_bottom", "quartz_block_bottom", 0.8, ToolPickaxe, CreativeBlocks},
		{"smooth_red_sandstone", "平滑红砂岩", "red_sandstone_top", "red_sandstone_top", "red_sandstone_top", 0.8, ToolPickaxe, CreativeBlocks},
		{"smooth_sandstone", "平滑砂岩", "sandstone_top", "sandstone_top", "sandstone_top", 0.8, ToolPickaxe, CreativeBlocks},
		{"smooth_stone", "平滑石头", "smooth_stone", "smooth_stone", "smooth_stone", 2, ToolPickaxe, CreativeBlocks},
		{"soul_soil", "灵魂土", "soul_soil", "soul_soil", "soul_soil", 0.5, ToolShovel, CreativeNatural},
		{"wet_sponge", "湿海绵", "wet_sponge", "wet_sponge", "wet_sponge", 0.6, ToolHoe, CreativeNatural},
		{"red_sand", "红沙", "red_sand", "red_sand", "red_sand", 0.5, ToolShovel, CreativeNatural},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(blocks))
	for i, block := range blocks {
		id := int32(14100 + i)
		name := "minecraft:" + block.name
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: id, Name: name, TextureTop: block.top, TextureBottom: block.bottom, TextureSide: block.side,
			Category: block.category, Hardness: block.hardness, HarvestTool: block.tool,
			DropRuntimeID: id,
		})
		extendedBlockDisplayNames[name] = block.display
	}
	return definitions
}

func woodCubeDefinitions() []ExtendedBlockDefinition {
	type woodType struct {
		name, texture string
	}
	woods := [...]woodType{
		{"oak_wood", "oak_log"},
		{"spruce_wood", "spruce_log"},
		{"birch_wood", "birch_log"},
		{"jungle_wood", "jungle_log"},
		{"acacia_wood", "acacia_log"},
		{"dark_oak_wood", "dark_oak_log"},
		{"mangrove_wood", "mangrove_log"},
		{"cherry_wood", "cherry_log"},
		{"pale_oak_wood", "pale_oak_log"},
		{"crimson_hyphae", "crimson_stem"},
		{"warped_hyphae", "warped_stem"},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(woods)*2)
	for i, wood := range woods {
		id := int32(10200 + i*2)
		name := "minecraft:" + wood.name
		strippedName := "minecraft:stripped_" + wood.name
		strippedTexture := "stripped_" + wood.texture
		definitions = append(definitions,
			ExtendedBlockDefinition{
				RuntimeID: id, Name: name, TextureTop: wood.texture, TextureBottom: wood.texture, TextureSide: wood.texture,
				Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe, DropRuntimeID: id,
				StrippedVariantID: id + 1,
			},
			ExtendedBlockDefinition{
				RuntimeID: id + 1, Name: strippedName, TextureTop: strippedTexture, TextureBottom: strippedTexture, TextureSide: strippedTexture,
				Category: CreativeBlocks, Hardness: 2, HarvestTool: ToolAxe, DropRuntimeID: id + 1,
			},
		)
	}
	return definitions
}

func modernFullCubeDefinitions() []ExtendedBlockDefinition {
	type spec struct {
		name, display, top, bottom, side string
		category                         CreativeCategory
		hardness                         float64
		harvestTool                      ToolType
		requiresTool                     bool
		requiredLevel                    int
		drop                             string
	}
	blocks := []spec{
		{"andesite", "安山岩", "andesite", "andesite", "andesite", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"polished_andesite", "磨制安山岩", "polished_andesite", "polished_andesite", "polished_andesite", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"diorite", "闪长岩", "diorite", "diorite", "diorite", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"polished_diorite", "磨制闪长岩", "polished_diorite", "polished_diorite", "polished_diorite", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"granite", "花岗岩", "granite", "granite", "granite", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"polished_granite", "磨制花岗岩", "polished_granite", "polished_granite", "polished_granite", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"deepslate", "深板岩", "deepslate_top", "deepslate_top", "deepslate", CreativeNatural, 3.0, ToolPickaxe, true, 1, "cobbled_deepslate"},
		{"cobbled_deepslate", "深板岩圆石", "cobbled_deepslate", "cobbled_deepslate", "cobbled_deepslate", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"polished_deepslate", "磨制深板岩", "polished_deepslate", "polished_deepslate", "polished_deepslate", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"chiseled_deepslate", "錾制深板岩", "chiseled_deepslate", "chiseled_deepslate", "chiseled_deepslate", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"deepslate_bricks", "深板岩砖", "deepslate_bricks", "deepslate_bricks", "deepslate_bricks", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"cracked_deepslate_bricks", "裂纹深板岩砖", "cracked_deepslate_bricks", "cracked_deepslate_bricks", "cracked_deepslate_bricks", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"deepslate_tiles", "深板岩瓦", "deepslate_tiles", "deepslate_tiles", "deepslate_tiles", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"cracked_deepslate_tiles", "裂纹深板岩瓦", "cracked_deepslate_tiles", "cracked_deepslate_tiles", "cracked_deepslate_tiles", CreativeBlocks, 3.5, ToolPickaxe, true, 1, ""},
		{"calcite", "方解石", "calcite", "calcite", "calcite", CreativeNatural, 0.75, ToolPickaxe, true, 0, ""},
		{"tuff", "凝灰岩", "tuff", "tuff", "tuff", CreativeNatural, 1.5, ToolPickaxe, true, 0, ""},
		{"polished_tuff", "磨制凝灰岩", "polished_tuff", "polished_tuff", "polished_tuff", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"chiseled_tuff", "錾制凝灰岩", "chiseled_tuff", "chiseled_tuff", "chiseled_tuff", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"tuff_bricks", "凝灰岩砖", "tuff_bricks", "tuff_bricks", "tuff_bricks", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"chiseled_tuff_bricks", "錾制凝灰岩砖", "chiseled_tuff_bricks", "chiseled_tuff_bricks", "chiseled_tuff_bricks", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"dripstone_block", "滴水石块", "dripstone_block", "dripstone_block", "dripstone_block", CreativeNatural, 1.5, ToolPickaxe, true, 0, ""},
		{"moss_block", "苔藓块", "moss_block", "moss_block", "moss_block", CreativeNatural, 0.1, ToolHoe, false, 0, ""},
		{"mud", "泥巴块", "mud", "mud", "mud", CreativeNatural, 0.5, ToolShovel, false, 0, ""},
		{"packed_mud", "黏土块", "packed_mud", "packed_mud", "packed_mud", CreativeBlocks, 1.0, ToolShovel, false, 0, ""},
		{"mud_bricks", "泥砖", "mud_bricks", "mud_bricks", "mud_bricks", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"raw_iron_block", "粗铁块", "raw_iron_block", "raw_iron_block", "raw_iron_block", CreativeNatural, 5.0, ToolPickaxe, true, 1, ""},
		{"raw_copper_block", "粗铜块", "raw_copper_block", "raw_copper_block", "raw_copper_block", CreativeNatural, 5.0, ToolPickaxe, true, 1, ""},
		{"raw_gold_block", "粗金块", "raw_gold_block", "raw_gold_block", "raw_gold_block", CreativeNatural, 5.0, ToolPickaxe, true, 2, ""},
		{"copper_block", "铜块", "copper_block", "copper_block", "copper_block", CreativeBlocks, 3.0, ToolPickaxe, true, 1, ""},
		{"exposed_copper", "斑驳的铜块", "exposed_copper", "exposed_copper", "exposed_copper", CreativeBlocks, 3.0, ToolPickaxe, true, 1, ""},
		{"weathered_copper", "锈蚀的铜块", "weathered_copper", "weathered_copper", "weathered_copper", CreativeBlocks, 3.0, ToolPickaxe, true, 1, ""},
		{"oxidized_copper", "氧化的铜块", "oxidized_copper", "oxidized_copper", "oxidized_copper", CreativeBlocks, 3.0, ToolPickaxe, true, 1, ""},
		{"blackstone", "黑石", "blackstone", "blackstone", "blackstone", CreativeNatural, 1.5, ToolPickaxe, true, 0, ""},
		{"polished_blackstone", "磨制黑石", "polished_blackstone", "polished_blackstone", "polished_blackstone", CreativeBlocks, 2.0, ToolPickaxe, true, 0, ""},
		{"polished_blackstone_bricks", "磨制黑石砖", "polished_blackstone_bricks", "polished_blackstone_bricks", "polished_blackstone_bricks", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"cracked_polished_blackstone_bricks", "裂纹磨制黑石砖", "cracked_polished_blackstone_bricks", "cracked_polished_blackstone_bricks", "cracked_polished_blackstone_bricks", CreativeBlocks, 1.5, ToolPickaxe, true, 0, ""},
		{"basalt", "玄武岩", "basalt_top", "basalt_top", "basalt_side", CreativeNatural, 1.25, ToolPickaxe, true, 0, ""},
		{"polished_basalt", "磨制玄武岩", "polished_basalt_top", "polished_basalt_top", "polished_basalt_side", CreativeBlocks, 1.25, ToolPickaxe, true, 0, ""},
		{"smooth_basalt", "平滑玄武岩", "smooth_basalt", "smooth_basalt", "smooth_basalt", CreativeNatural, 1.25, ToolPickaxe, true, 0, ""},
		{"crimson_planks", "绯红木板", "crimson_planks", "crimson_planks", "crimson_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"warped_planks", "诡异木板", "warped_planks", "warped_planks", "warped_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"mangrove_planks", "红树木板", "mangrove_planks", "mangrove_planks", "mangrove_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"cherry_planks", "樱花木板", "cherry_planks", "cherry_planks", "cherry_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"bamboo_planks", "竹板", "bamboo_planks", "bamboo_planks", "bamboo_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"pale_oak_planks", "苍白橡木木板", "pale_oak_planks", "pale_oak_planks", "pale_oak_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"spruce_planks", "云杉木板", "spruce_planks", "spruce_planks", "spruce_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"birch_planks", "白桦木板", "birch_planks", "birch_planks", "birch_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"jungle_planks", "丛林木板", "jungle_planks", "jungle_planks", "jungle_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"acacia_planks", "金合欢木板", "acacia_planks", "acacia_planks", "acacia_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"dark_oak_planks", "深色橡木木板", "dark_oak_planks", "dark_oak_planks", "dark_oak_planks", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"birch_log", "白桦原木", "birch_log_top", "birch_log_top", "birch_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"jungle_log", "丛林原木", "jungle_log_top", "jungle_log_top", "jungle_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"acacia_log", "金合欢原木", "acacia_log_top", "acacia_log_top", "acacia_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"dark_oak_log", "深色橡木原木", "dark_oak_log_top", "dark_oak_log_top", "dark_oak_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_oak_log", "去皮橡木原木", "stripped_oak_log_top", "stripped_oak_log_top", "stripped_oak_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"stripped_spruce_log", "去皮云杉原木", "stripped_spruce_log_top", "stripped_spruce_log_top", "stripped_spruce_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"stripped_birch_log", "去皮白桦原木", "stripped_birch_log_top", "stripped_birch_log_top", "stripped_birch_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"stripped_jungle_log", "去皮丛林原木", "stripped_jungle_log_top", "stripped_jungle_log_top", "stripped_jungle_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"stripped_acacia_log", "去皮金合欢原木", "stripped_acacia_log_top", "stripped_acacia_log_top", "stripped_acacia_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"stripped_dark_oak_log", "去皮深色橡木原木", "stripped_dark_oak_log_top", "stripped_dark_oak_log_top", "stripped_dark_oak_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"mangrove_log", "红树原木", "mangrove_log_top", "mangrove_log_top", "mangrove_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_mangrove_log", "去皮红树原木", "stripped_mangrove_log_top", "stripped_mangrove_log_top", "stripped_mangrove_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"cherry_log", "樱花原木", "cherry_log_top", "cherry_log_top", "cherry_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_cherry_log", "去皮樱花原木", "stripped_cherry_log_top", "stripped_cherry_log_top", "stripped_cherry_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"pale_oak_log", "苍白橡木原木", "pale_oak_log_top", "pale_oak_log_top", "pale_oak_log", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_pale_oak_log", "去皮苍白橡木原木", "stripped_pale_oak_log_top", "stripped_pale_oak_log_top", "stripped_pale_oak_log", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"crimson_stem", "绯红菌柄", "crimson_stem_top", "crimson_stem_top", "crimson_stem", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_crimson_stem", "去皮绯红菌柄", "stripped_crimson_stem_top", "stripped_crimson_stem_top", "stripped_crimson_stem", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"warped_stem", "诡异菌柄", "warped_stem_top", "warped_stem_top", "warped_stem", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_warped_stem", "去皮诡异菌柄", "stripped_warped_stem_top", "stripped_warped_stem_top", "stripped_warped_stem", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"bamboo_block", "竹块", "bamboo_block_top", "bamboo_block_top", "bamboo_block", CreativeNatural, 2.0, ToolAxe, false, 0, ""},
		{"stripped_bamboo_block", "去皮竹块", "stripped_bamboo_block_top", "stripped_bamboo_block_top", "stripped_bamboo_block", CreativeBlocks, 2.0, ToolAxe, false, 0, ""},
		{"copper_ore", "铜矿石", "copper_ore", "copper_ore", "copper_ore", CreativeNatural, 3.0, ToolPickaxe, true, 1, "raw_copper"},
		{"deepslate_copper_ore", "深层铜矿石", "deepslate_copper_ore", "deepslate_copper_ore", "deepslate_copper_ore", CreativeNatural, 4.5, ToolPickaxe, true, 1, "raw_copper"},
		{"deepslate_coal_ore", "深层煤矿石", "deepslate_coal_ore", "deepslate_coal_ore", "deepslate_coal_ore", CreativeNatural, 4.5, ToolPickaxe, true, 0, "coal"},
		{"deepslate_iron_ore", "深层铁矿石", "deepslate_iron_ore", "deepslate_iron_ore", "deepslate_iron_ore", CreativeNatural, 4.5, ToolPickaxe, true, 1, "raw_iron"},
		{"deepslate_gold_ore", "深层金矿石", "deepslate_gold_ore", "deepslate_gold_ore", "deepslate_gold_ore", CreativeNatural, 4.5, ToolPickaxe, true, 2, "raw_gold"},
		{"deepslate_diamond_ore", "深层钻石矿石", "deepslate_diamond_ore", "deepslate_diamond_ore", "deepslate_diamond_ore", CreativeNatural, 4.5, ToolPickaxe, true, 2, "diamond"},
		{"deepslate_emerald_ore", "深层绿宝石矿石", "deepslate_emerald_ore", "deepslate_emerald_ore", "deepslate_emerald_ore", CreativeNatural, 4.5, ToolPickaxe, true, 2, "emerald"},
		{"deepslate_lapis_ore", "深层青金石矿石", "deepslate_lapis_ore", "deepslate_lapis_ore", "deepslate_lapis_ore", CreativeNatural, 4.5, ToolPickaxe, true, 1, "lapis_lazuli"},
		{"deepslate_redstone_ore", "深层红石矿石", "deepslate_redstone_ore", "deepslate_redstone_ore", "deepslate_redstone_ore", CreativeNatural, 4.5, ToolPickaxe, true, 2, "redstone"},
		{"nether_quartz_ore", "下界石英矿石", "nether_quartz_ore", "nether_quartz_ore", "nether_quartz_ore", CreativeNatural, 3.0, ToolPickaxe, true, 0, "quartz"},
		{"ancient_debris", "远古残骸", "ancient_debris_top", "ancient_debris_top", "ancient_debris_side", CreativeNatural, 30.0, ToolPickaxe, true, 3, ""},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(blocks))
	idsByName := make(map[string]int32, len(blocks))
	for i, block := range blocks {
		idsByName[block.name] = 12000 + int32(i)
	}
	for i, block := range blocks {
		definition := ExtendedBlockDefinition{
			RuntimeID: 12000 + int32(i), Name: "minecraft:" + block.name,
			TextureTop: block.top, TextureBottom: block.bottom, TextureSide: block.side,
			Category: block.category, Hardness: block.hardness, HarvestTool: block.harvestTool,
			RequiresHarvestTool: block.requiresTool, RequiredHarvestLevel: block.requiredLevel,
		}
		if block.drop != "" {
			definition.DropRuntimeID = idsByName[block.drop]
			if definition.DropRuntimeID == 0 {
				definition.DropRuntimeID = map[string]int32{
					"raw_copper": ItemRawCopper, "raw_iron": ItemRawIron, "raw_gold": ItemRawGold,
					"coal": ItemCoal, "diamond": ItemDiamond, "emerald": ItemEmerald, "quartz": ItemNetherQuartz,
					"lapis_lazuli": ItemLapisLazuli, "redstone": 331,
				}[block.drop]
			}
		}
		if dropRange, ok := modernOreDropRanges[definition.Name]; ok {
			definition.DropCountMin = dropRange[0]
			definition.DropCountMax = dropRange[1]
		}
		definitions = append(definitions, definition)
		extendedBlockDisplayNames[definition.Name] = block.display
	}
	for i := range definitions {
		if strippedName := strippedWoodNames[definitions[i].Name]; strippedName != "" {
			definitions[i].StrippedVariantID = idsByName[strippedName]
		}
	}
	return definitions
}

var strippedWoodNames = map[string]string{
	"minecraft:birch_log":    "stripped_birch_log",
	"minecraft:jungle_log":   "stripped_jungle_log",
	"minecraft:acacia_log":   "stripped_acacia_log",
	"minecraft:dark_oak_log": "stripped_dark_oak_log",
	"minecraft:mangrove_log": "stripped_mangrove_log",
	"minecraft:cherry_log":   "stripped_cherry_log",
	"minecraft:pale_oak_log": "stripped_pale_oak_log",
	"minecraft:crimson_stem": "stripped_crimson_stem",
	"minecraft:warped_stem":  "stripped_warped_stem",
	"minecraft:bamboo_block": "stripped_bamboo_block",
}

func coloredBlockDefinitions() []ExtendedBlockDefinition {
	colors := []struct{ id, name string }{
		{"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(colors)*3)
	nextID := int32(10012)
	for _, color := range colors {
		for _, blockType := range []struct {
			suffix, texture string
		}{
			{"wool", color.id + "_wool"},
			{"concrete", color.id + "_concrete"},
			{"terracotta", color.id + "_terracotta"},
		} {
			name := "minecraft:" + color.id + "_" + blockType.suffix
			if name == "minecraft:white_wool" {
				// White wool already has the legacy runtime ID 35; keep its
				// modern-colored-block slot reserved without exposing a duplicate item.
				nextID++
				continue
			}
			definitions = append(definitions, ExtendedBlockDefinition{
				RuntimeID: nextID, Name: name, TextureTop: blockType.texture,
				TextureBottom: blockType.texture, TextureSide: blockType.texture,
				Category: CreativeDecoration,
			})
			extendedBlockDisplayNames[name] = color.name + map[string]string{"wool": "色羊毛", "concrete": "色混凝土", "terracotta": "色陶瓦"}[blockType.suffix]
			nextID++
		}
	}
	return definitions
}

func stainedGlassBlockDefinitions() []ExtendedBlockDefinition {
	colors := []struct{ id, name string }{
		{"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(colors))
	for i, color := range colors {
		texture := color.id + "_stained_glass"
		name := "minecraft:" + texture
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: 17100 + int32(i), Name: name,
			TextureTop: texture, TextureBottom: texture, TextureSide: texture,
			Category: CreativeDecoration, Hardness: 0.3,
			DropsNothing: true,
		})
		extendedBlockDisplayNames[name] = color.name + "色染色玻璃"
	}
	return definitions
}

// IsStainedGlassBlock reports whether the runtime ID is one of Java's colored glass blocks.
func IsStainedGlassBlock(runtimeID int32) bool {
	definition, ok := ExtendedBlockByID(runtimeID)
	return ok && len(definition.Name) >= len("_stained_glass") &&
		definition.Name[len(definition.Name)-len("_stained_glass"):] == "_stained_glass"
}

func glassPaneBlockDefinitions() []ExtendedBlockDefinition {
	colors := []struct{ id, name string }{
		{"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(colors)+1)
	definitions = append(definitions, ExtendedBlockDefinition{
		RuntimeID: 17116, Name: "minecraft:glass_pane",
		TextureTop: "glass_pane_top", TextureBottom: "glass_pane_top", TextureSide: "glass_pane_top",
		Category: CreativeDecoration, Hardness: 0.3, DropsNothing: true,
	})
	extendedBlockDisplayNames["minecraft:glass_pane"] = "玻璃板"
	for i, color := range colors {
		name := "minecraft:" + color.id + "_stained_glass_pane"
		texture := color.id + "_stained_glass_pane_top"
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: 17117 + int32(i), Name: name,
			TextureTop: texture, TextureBottom: texture, TextureSide: texture,
			Category: CreativeDecoration, Hardness: 0.3, DropsNothing: true,
		})
		extendedBlockDisplayNames[name] = color.name + "色染色玻璃板"
	}
	definitions = append(definitions, ExtendedBlockDefinition{
		RuntimeID: ItemIronBars, Name: "minecraft:iron_bars",
		TextureTop: "iron_bars", TextureBottom: "iron_bars", TextureSide: "iron_bars",
		Category: CreativeBlocks, Hardness: 5, HarvestTool: ToolPickaxe,
		RequiresHarvestTool: true, DropRuntimeID: ItemIronBars,
	})
	extendedBlockDisplayNames["minecraft:iron_bars"] = "铁栏杆"
	return definitions
}

// IsGlassPaneBlock reports whether a runtime ID is a clear or colored glass pane.
func IsGlassPaneBlock(runtimeID int32) bool {
	definition, ok := ExtendedBlockByID(runtimeID)
	if !ok {
		return false
	}
	return definition.Name == "minecraft:glass_pane" ||
		(len(definition.Name) >= len("_stained_glass_pane") && definition.Name[len(definition.Name)-len("_stained_glass_pane"):] == "_stained_glass_pane")
}

// IsIronBarsBlock reports whether a runtime ID is the iron bars block.
func IsIronBarsBlock(runtimeID int32) bool { return runtimeID == ItemIronBars }

// IsPaneBlock includes glass panes and iron bars, which share a connected
// center-post footprint while retaining separate material and render behavior.
func IsPaneBlock(runtimeID int32) bool {
	return IsGlassPaneBlock(runtimeID) || IsIronBarsBlock(runtimeID)
}

func IsIronChainBlock(runtimeID int32) bool {
	_, ok := IronChainStateProperties(runtimeID)
	return ok
}

func candleBlockDefinitions() []ExtendedBlockDefinition {
	colors := []struct{ id, name string }{
		{"", ""}, {"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(colors))
	for i, color := range colors {
		name := "candle"
		display := "蜡烛"
		if color.id != "" {
			name = color.id + "_candle"
			display = color.name + "色蜡烛"
		}
		registryName := "minecraft:" + name
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: FirstCandleItem + int32(i), Name: registryName,
			TextureTop: name, TextureBottom: name, TextureSide: name,
			Category: CreativeDecoration, Hardness: 0.1, DropRuntimeID: FirstCandleItem + int32(i),
		})
		extendedBlockDisplayNames[registryName] = display
	}
	return definitions
}

// IsCandleBlock reports whether a runtime ID belongs to a candle item/block.
func IsCandleBlock(runtimeID int32) bool {
	if _, _, _, ok := CandleStateProperties(runtimeID); ok {
		return true
	}
	return IsCandleItem(runtimeID)
}

func IsCandleItem(itemID int32) bool {
	return itemID >= FirstCandleItem && itemID <= LastCandleItem
}

// CandleBlockState encodes the vanilla candle count and lit properties in an
// internal runtime ID, keeping candle items separate from placed block states.
func CandleBlockState(itemID int32, count int, lit bool) int32 {
	if !IsCandleItem(itemID) || count < 1 || count > 4 {
		return 0
	}
	state := CandleStateBase + (itemID-FirstCandleItem)*8 + int32((count-1)*2)
	if lit {
		state++
	}
	return state
}

// CandleStateProperties decodes the candle item, count, and lit flag.
func CandleStateProperties(stateID int32) (itemID int32, count int, lit bool, ok bool) {
	offset := stateID - CandleStateBase
	if offset < 0 || offset >= (LastCandleItem-FirstCandleItem+1)*8 {
		return 0, 0, false, false
	}
	itemID = FirstCandleItem + offset/8
	properties := offset % 8
	return itemID, int(properties/2) + 1, properties%2 == 1, true
}

// BlockTextureForState preserves candle lighting in render texture lookup.
func BlockTextureForState(blockID int32) int32 {
	if _, _, _, ok := CandleStateProperties(blockID); ok {
		return blockID
	}
	if _, _, _, _, _, ok := WoodDoorStateProperties(blockID); ok {
		return blockID
	}
	return BlockItemForState(blockID)
}

func glazedTerracottaDefinitions() []ExtendedBlockDefinition {
	colors := []struct{ id, name string }{
		{"white", "白"}, {"orange", "橙"}, {"magenta", "品红"}, {"light_blue", "淡蓝"},
		{"yellow", "黄"}, {"lime", "黄绿"}, {"pink", "粉红"}, {"gray", "灰"},
		{"light_gray", "淡灰"}, {"cyan", "青"}, {"purple", "紫"}, {"blue", "蓝"},
		{"brown", "棕"}, {"green", "绿"}, {"red", "红"}, {"black", "黑"},
	}
	definitions := make([]ExtendedBlockDefinition, 0, len(colors))
	for i, color := range colors {
		name := "minecraft:" + color.id + "_glazed_terracotta"
		texture := color.id + "_glazed_terracotta"
		definitions = append(definitions, ExtendedBlockDefinition{
			RuntimeID: 10064 + int32(i), Name: name,
			TextureTop: texture, TextureBottom: texture, TextureSide: texture,
			Category: CreativeDecoration, Hardness: 1.4, HarvestTool: ToolPickaxe,
		})
		extendedBlockDisplayNames[name] = color.name + "色带釉陶瓦"
	}
	return definitions
}

var extendedBlockByID = func() map[int32]ExtendedBlockDefinition {
	definitions := make(map[int32]ExtendedBlockDefinition, len(extendedBlockDefinitions))
	for _, definition := range extendedBlockDefinitions {
		definitions[definition.RuntimeID] = definition
	}
	return definitions
}()

var extendedBlockByName = func() map[string]ExtendedBlockDefinition {
	definitions := make(map[string]ExtendedBlockDefinition, len(extendedBlockDefinitions))
	for _, definition := range extendedBlockDefinitions {
		definitions[definition.Name] = definition
	}
	return definitions
}()

// ExtendedBlockDefinitions returns the supported namespaced blocks in stable order.
func ExtendedBlockDefinitions() []ExtendedBlockDefinition {
	return append([]ExtendedBlockDefinition(nil), extendedBlockDefinitions...)
}

// ExtendedBlockByID resolves an internal runtime ID to its namespaced definition.
func ExtendedBlockByID(runtimeID int32) (ExtendedBlockDefinition, bool) {
	definition, ok := extendedBlockByID[runtimeID]
	return definition, ok
}

// ExtendedBlockByName resolves a namespaced block name to its internal runtime definition.
func ExtendedBlockByName(name string) (ExtendedBlockDefinition, bool) {
	definition, ok := extendedBlockByName[name]
	return definition, ok
}

// BlockDropCountRange returns the base quantity range before enchantment bonuses.
func BlockDropCountRange(blockID int32) (int, int) {
	if definition, ok := ExtendedBlockByID(blockID); ok {
		if definition.DropsNothing {
			return 0, 0
		}
		if definition.DropCountMin > 0 && definition.DropCountMax >= definition.DropCountMin {
			return definition.DropCountMin, definition.DropCountMax
		}
	}
	if IsDoubleSlabState(blockID) {
		return 2, 2
	}
	switch blockID {
	case 73: // Redstone ore
		return 4, 5
	case 74: // Lapis lazuli ore
		return 4, 9
	default:
		return 1, 1
	}
}

// StrippedWoodBlockVariant returns the stripped-log or stripped-stem runtime ID
// corresponding to a wood block, including legacy oak and spruce log IDs.
func StrippedWoodBlockVariant(runtimeID int32) (int32, bool) {
	switch runtimeID {
	case 17:
		runtimeID = extendedBlockByName["minecraft:stripped_oak_log"].RuntimeID
	case 22:
		runtimeID = extendedBlockByName["minecraft:stripped_spruce_log"].RuntimeID
	default:
		definition, ok := ExtendedBlockByID(runtimeID)
		if !ok || definition.StrippedVariantID == 0 {
			return 0, false
		}
		return definition.StrippedVariantID, true
	}
	return runtimeID, runtimeID != 0
}

func IsExtendedLeaves(runtimeID int32) bool {
	return runtimeID >= 10060 && runtimeID <= 10062
}

func IsExtendedFoliage(runtimeID int32) bool {
	return runtimeID == 10063 || runtimeID == 10100
}

func ExtendedBlockItemTextureName(runtimeID int32) string {
	if IsCushionBlock(runtimeID) {
		return cushionColors[runtimeID-FirstCushionBlock].name + "_cushion"
	}
	switch runtimeID {
	case 10060:
		return "red_poplar_leaves"
	case 10061:
		return "orange_poplar_leaves"
	case 10062:
		return "yellow_poplar_leaves"
	case 10063:
		return "poplar_sapling"
	case 10100:
		return "red_shrub"
	default:
		if definition, ok := ExtendedBlockByID(runtimeID); ok {
			return definition.TextureSide
		}
		return ""
	}
}

var extendedBlockDisplayNames = map[string]string{
	"minecraft:spruce_slab":             "云杉木台阶",
	"minecraft:birch_slab":              "白桦木台阶",
	"minecraft:jungle_slab":             "丛林木台阶",
	"minecraft:acacia_slab":             "金合欢木台阶",
	"minecraft:dark_oak_slab":           "深色橡木台阶",
	"minecraft:mangrove_slab":           "红树木台阶",
	"minecraft:cherry_slab":             "樱花木台阶",
	"minecraft:pale_oak_slab":           "苍白橡木台阶",
	"minecraft:poplar_slab":             "杨木台阶",
	"minecraft:bamboo_slab":             "竹台阶",
	"minecraft:bamboo_mosaic_slab":      "竹马赛克台阶",
	"minecraft:crimson_slab":            "绯红菌柄台阶",
	"minecraft:warped_slab":             "诡异菌柄台阶",
	"minecraft:cinnabar":                "朱砂块",
	"minecraft:polished_cinnabar":       "磨制朱砂块",
	"minecraft:chiseled_cinnabar":       "錾制朱砂块",
	"minecraft:cinnabar_bricks":         "朱砂砖",
	"minecraft:sulfur":                  "硫磺块",
	"minecraft:polished_sulfur":         "磨制硫磺块",
	"minecraft:chiseled_sulfur":         "錾制硫磺块",
	"minecraft:sulfur_bricks":           "硫磺砖",
	"minecraft:potent_sulfur":           "强效硫磺块",
	"minecraft:poplar_planks":           "杨木木板",
	"minecraft:poplar_log":              "杨木原木",
	"minecraft:stripped_poplar_log":     "去皮杨木原木",
	"minecraft:poplar_wood":             "杨木",
	"minecraft:stripped_poplar_wood":    "去皮杨木",
	"minecraft:oak_wood":                "橡木",
	"minecraft:stripped_oak_wood":       "去皮橡木",
	"minecraft:spruce_wood":             "云杉木",
	"minecraft:stripped_spruce_wood":    "去皮云杉木",
	"minecraft:birch_wood":              "白桦木",
	"minecraft:stripped_birch_wood":     "去皮白桦木",
	"minecraft:jungle_wood":             "丛林木",
	"minecraft:stripped_jungle_wood":    "去皮丛林木",
	"minecraft:acacia_wood":             "金合欢木",
	"minecraft:stripped_acacia_wood":    "去皮金合欢木",
	"minecraft:dark_oak_wood":           "深色橡木",
	"minecraft:stripped_dark_oak_wood":  "去皮深色橡木",
	"minecraft:mangrove_wood":           "红树木",
	"minecraft:stripped_mangrove_wood":  "去皮红树木",
	"minecraft:cherry_wood":             "樱花木",
	"minecraft:stripped_cherry_wood":    "去皮樱花木",
	"minecraft:pale_oak_wood":           "苍白橡木",
	"minecraft:stripped_pale_oak_wood":  "去皮苍白橡木",
	"minecraft:crimson_hyphae":          "绯红菌核",
	"minecraft:stripped_crimson_hyphae": "去皮绯红菌核",
	"minecraft:warped_hyphae":           "诡异菌核",
	"minecraft:stripped_warped_hyphae":  "去皮诡异菌核",
	"minecraft:red_poplar_leaves":       "红色杨木树叶",
	"minecraft:orange_poplar_leaves":    "橙色杨木树叶",
	"minecraft:yellow_poplar_leaves":    "黄色杨木树叶",
	"minecraft:poplar_sapling":          "杨木树苗",
	"minecraft:red_shrub":               "红灌木",
}

func init() {
	for _, slab := range colorSlabDefinitions {
		extendedBlockDisplayNames[slab.Name] = slab.DisplayName
	}
	for _, slab := range masonrySlabDefinitions {
		extendedBlockDisplayNames[slab.Name] = slab.DisplayName
	}
	for _, stair := range colorStairDefinitions {
		extendedBlockDisplayNames[stair.Name] = stair.DisplayName
	}
}
