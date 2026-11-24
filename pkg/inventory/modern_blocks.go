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
	StrippedVariantID    int32
}

var extendedBlockDefinitions = append(append(append([]ExtendedBlockDefinition{
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
	baseExtendedBlock(10060, "minecraft:red_poplar_leaves", "red_poplar_leaves", "red_poplar_leaves", "red_poplar_leaves", CreativeNatural),
	baseExtendedBlock(10061, "minecraft:orange_poplar_leaves", "orange_poplar_leaves", "orange_poplar_leaves", "orange_poplar_leaves", CreativeNatural),
	baseExtendedBlock(10062, "minecraft:yellow_poplar_leaves", "yellow_poplar_leaves", "yellow_poplar_leaves", "yellow_poplar_leaves", CreativeNatural),
	baseExtendedBlock(10063, "minecraft:poplar_sapling", "poplar_sapling", "poplar_sapling", "poplar_sapling", CreativeNatural),
	{RuntimeID: 10100, Name: "minecraft:red_shrub", TextureTop: "red_shrub", TextureBottom: "red_shrub", TextureSide: "red_shrub", Category: CreativeNatural, Hardness: 0.1, DropRuntimeID: 10100},
}, modernFullCubeDefinitions()...), coloredBlockDefinitions()...), glazedTerracottaDefinitions()...)

func baseExtendedBlock(id int32, name, top, bottom, side string, category CreativeCategory) ExtendedBlockDefinition {
	return ExtendedBlockDefinition{
		RuntimeID: id, Name: name, TextureTop: top, TextureBottom: bottom,
		TextureSide: side, Category: category,
	}
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
		return ""
	}
}

var extendedBlockDisplayNames = map[string]string{
	"minecraft:cinnabar":             "朱砂块",
	"minecraft:polished_cinnabar":    "磨制朱砂块",
	"minecraft:chiseled_cinnabar":    "錾制朱砂块",
	"minecraft:cinnabar_bricks":      "朱砂砖",
	"minecraft:sulfur":               "硫磺块",
	"minecraft:polished_sulfur":      "磨制硫磺块",
	"minecraft:chiseled_sulfur":      "錾制硫磺块",
	"minecraft:sulfur_bricks":        "硫磺砖",
	"minecraft:potent_sulfur":        "强效硫磺块",
	"minecraft:poplar_planks":        "杨木木板",
	"minecraft:poplar_log":           "杨木原木",
	"minecraft:stripped_poplar_log":  "去皮杨木原木",
	"minecraft:red_poplar_leaves":    "红色杨木树叶",
	"minecraft:orange_poplar_leaves": "橙色杨木树叶",
	"minecraft:yellow_poplar_leaves": "黄色杨木树叶",
	"minecraft:poplar_sapling":       "杨木树苗",
	"minecraft:red_shrub":            "红灌木",
}
