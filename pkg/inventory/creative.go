package inventory

import (
	"fmt"
	"strings"
)

// CreativeCategory groups the catalog shown by the creative inventory.
type CreativeCategory int

const (
	CreativeAll CreativeCategory = iota
	CreativeBlocks
	CreativeDecoration
	CreativeNatural
	CreativeFunctional
	CreativeRedstone
	CreativeTools
	CreativeCombat
	CreativeFood
	CreativeMaterials
	CreativeSpawnEggs
	CreativeHotbar
	CreativeInventory
)

// CreativeHotbarPresets stores nine named player hotbars, each containing nine slots.
type CreativeHotbarPresets [9][9]ItemStack

// CreativeItem describes an item offered by the local creative inventory.
type CreativeItem struct {
	ID       int32
	Name     string
	Category CreativeCategory
}

// CreativeCatalog returns the currently implemented creative items in a
// stable, vanilla-like order. The catalog is intentionally limited to items
// with a corresponding local block, item behavior, or sprite implementation.
func CreativeCatalog(category CreativeCategory) []CreativeItem {
	if category == CreativeHotbar || category == CreativeInventory {
		return nil
	}
	items := make([]CreativeItem, 0, len(creativeCatalog)+len(extendedBlockDefinitions)+len(spawnEggDefinitions))
	items = append(items, creativeCatalog...)
	for _, definition := range extendedBlockDefinitions {
		items = append(items, CreativeItem{ID: definition.RuntimeID, Name: extendedBlockDisplayNames[definition.Name], Category: definition.Category})
	}
	for _, definition := range spawnEggDefinitions {
		items = append(items, CreativeItem{ID: definition.RuntimeID, Name: definition.DisplayName, Category: CreativeSpawnEggs})
	}
	for _, definition := range java263FoodItems {
		items = append(items, CreativeItem{ID: definition.RuntimeID, Name: definition.DisplayName, Category: CreativeFood})
	}
	if category == CreativeAll {
		return items
	}
	filtered := make([]CreativeItem, 0, len(items))
	for _, item := range items {
		if item.Category == category {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// SearchCreativeCatalog filters the selected category by its localized item name.
func SearchCreativeCatalog(category CreativeCategory, query string) []CreativeItem {
	items := CreativeCatalog(category)
	query = normalizeCreativeSearch(query)
	if query == "" {
		return items
	}
	filtered := make([]CreativeItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(normalizeCreativeSearch(item.Name), query) || strings.Contains(normalizeCreativeSearch(creativeItemRegistryName(item.ID)), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// CreativeItemDisplayName returns the localized catalog label for an item.
// Items outside the implemented creative catalog still receive a stable ID label.
func CreativeItemDisplayName(itemID int32) string {
	for _, item := range CreativeCatalog(CreativeAll) {
		if item.ID == itemID {
			return item.Name
		}
	}
	return fmt.Sprintf("物品 #%d", itemID)
}

func normalizeCreativeSearch(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "minecraft:")
	return strings.ReplaceAll(value, "_", " ")
}

func creativeItemRegistryName(itemID int32) string {
	if item, ok := FoodItemByID(itemID); ok {
		return item.Name
	}
	if block, ok := ExtendedBlockByID(itemID); ok {
		return block.Name
	}
	if egg, ok := SpawnEggByID(itemID); ok {
		return "minecraft:" + egg.EntityName + "_spawn_egg"
	}
	if itemID == ItemBoneMeal {
		return "minecraft:bone_meal"
	}
	if name, ok := legacyCreativeRegistryNames[itemID]; ok {
		return "minecraft:" + name
	}
	switch itemID {
	case ItemCopperHelmet:
		return "minecraft:copper_helmet"
	case ItemCopperChestplate:
		return "minecraft:copper_chestplate"
	case ItemCopperLeggings:
		return "minecraft:copper_leggings"
	case ItemCopperBoots:
		return "minecraft:copper_boots"
	case ItemRawCopper:
		return "minecraft:raw_copper"
	case ItemRawIron:
		return "minecraft:raw_iron"
	case ItemRawGold:
		return "minecraft:raw_gold"
	case ItemEmerald:
		return "minecraft:emerald"
	case ItemLapisLazuli:
		return "minecraft:lapis_lazuli"
	case ItemCopperIngot:
		return "minecraft:copper_ingot"
	case ItemNetheriteScrap:
		return "minecraft:netherite_scrap"
	}
	return ""
}

// legacyCreativeRegistryNames bridges the project's pre-registry numeric item
// IDs to modern vanilla names for creative search. These IDs remain internal
// runtime IDs; this map does not claim protocol or save-format compatibility.
var legacyCreativeRegistryNames = map[int32]string{
	1: "stone", 2: "grass_block", 3: "dirt", 4: "cobblestone", 5: "oak_planks", 6: "oak_sapling", 7: "bedrock",
	12: "sand", 13: "gravel", 14: "gold_ore", 15: "iron_ore",
	16: "coal_ore", 17: "oak_log", 18: "oak_leaves", 19: "sponge", 20: "glass", 21: "birch_leaves", 22: "spruce_log",
	23: "spruce_leaves", 24: "sandstone", 25: "note_block", 31: "short_grass", 32: "dead_bush", 33: "fern",
	35: "white_wool", 37: "dandelion", 38: "poppy", 41: "gold_block", 42: "iron_block",
	44: "stone_slab", 45: "bricks", 46: "tnt", 47: "bookshelf", 48: "mossy_cobblestone", 49: "obsidian", 50: "torch",
	52: "spawner", 54: "chest", 56: "diamond_ore", 57: "diamond_block", 58: "crafting_table", 61: "furnace",
	65: "ladder", 73: "redstone_ore", 74: "lapis_ore", 79: "ice", 80: "snow_block", 81: "cactus", 82: "clay",
	83: "sugar_cane", 84: "jukebox", 86: "pumpkin", 87: "netherrack", 88: "soul_sand", 89: "glowstone",
	91: "jack_o_lantern", 98: "stone_bricks", 103: "melon", 111: "lily_pad", 112: "nether_bricks",
	120: "end_portal_frame", 121: "end_stone", 122: "dragon_egg", 124: "redstone_lamp", 126: "oak_slab", 129: "emerald_ore", 133: "emerald_block", 138: "beacon", 150: "deepslate",
	152: "redstone_block", 153: "dropper", 154: "hopper", 155: "quartz_block", 159: "white_terracotta",
	165: "slime_block", 168: "prismarine", 169: "sea_lantern", 170: "hay_block", 172: "terracotta", 173: "coal_block",
	174: "packed_ice", 179: "red_sandstone", 256: "iron_shovel", 257: "iron_pickaxe", 258: "iron_axe", 259: "flint_and_steel",
	260: "apple", 261: "bow", 262: "arrow", 263: "coal", 264: "diamond", 265: "iron_ingot", 266: "gold_ingot",
	267: "iron_sword", 268: "wooden_sword", 269: "wooden_shovel", 270: "wooden_pickaxe", 271: "wooden_axe", 272: "stone_sword",
	273: "stone_shovel", 274: "stone_pickaxe", 275: "stone_axe", 276: "diamond_sword", 277: "diamond_shovel", 278: "diamond_pickaxe",
	279: "diamond_axe", 280: "stick", 281: "bowl", 282: "mushroom_stew", 283: "golden_sword", 284: "golden_shovel",
	285: "golden_pickaxe", 286: "golden_axe", 287: "string", 288: "feather", 289: "gunpowder", 290: "wooden_hoe",
	291: "stone_hoe", 292: "iron_hoe", 293: "diamond_hoe", 294: "golden_hoe", 295: "wheat_seeds", 296: "wheat",
	297: "bread", 298: "leather_helmet", 299: "leather_chestplate", 300: "leather_leggings", 301: "leather_boots",
	302: "chainmail_helmet", 303: "chainmail_chestplate", 304: "chainmail_leggings", 305: "chainmail_boots", 306: "iron_helmet",
	307: "iron_chestplate", 308: "iron_leggings", 309: "iron_boots", 310: "diamond_helmet", 311: "diamond_chestplate",
	312: "diamond_leggings", 313: "diamond_boots", 314: "golden_helmet", 315: "golden_chestplate", 316: "golden_leggings",
	317: "golden_boots", 318: "flint", 319: "porkchop", 320: "cooked_porkchop", 322: "golden_apple", 324: "oak_door",
	325: "bucket", 326: "water_bucket", 327: "lava_bucket", 328: "minecart", 330: "iron_door", 331: "redstone",
	332: "snowball", 333: "oak_boat", 334: "leather", 336: "brick", 337: "clay_ball", 340: "book", 342: "chest_minecart",
	343: "furnace_minecart", 344: "egg", 345: "compass", 346: "fishing_rod", 351: "bone_meal", 352: "bone",
	359: "shears", 360: "melon_slice", 363: "beef", 364: "cooked_beef", 365: "chicken", 366: "cooked_chicken",
	367: "rotten_flesh", 368: "ender_pearl", 371: "gold_nugget", 375: "spider_eye", 381: "ender_eye", 383: "slime_spawn_egg",
	406: "quartz", 423: "mutton", 424: "cooked_mutton",
}

var creativeCatalog = []CreativeItem{
	{1, "石头", CreativeBlocks}, {2, "草方块", CreativeBlocks}, {3, "泥土", CreativeBlocks}, {4, "圆石", CreativeBlocks},
	{5, "橡木木板", CreativeBlocks}, {6, "橡木树苗", CreativeNatural}, {7, "基岩", CreativeBlocks},
	{12, "沙子", CreativeBlocks}, {13, "沙砾", CreativeBlocks}, {14, "金矿石", CreativeBlocks}, {15, "铁矿石", CreativeBlocks},
	{16, "煤矿石", CreativeBlocks}, {17, "橡木原木", CreativeBlocks}, {18, "橡木树叶", CreativeNatural}, {19, "海绵", CreativeBlocks}, {20, "玻璃", CreativeDecoration},
	{21, "白桦树叶", CreativeNatural}, {22, "云杉原木", CreativeBlocks}, {23, "云杉树叶", CreativeNatural}, {24, "砂岩", CreativeBlocks}, {31, "高草", CreativeNatural},
	{25, "音符盒", CreativeRedstone}, {35, "白色羊毛", CreativeBlocks},
	{32, "枯萎灌木", CreativeNatural}, {33, "蕨", CreativeNatural}, {37, "蒲公英", CreativeNatural}, {38, "虞美人", CreativeNatural},
	{41, "金块", CreativeBlocks}, {42, "铁块", CreativeBlocks}, {44, "石台阶", CreativeBlocks}, {45, "砖块", CreativeBlocks}, {46, "TNT", CreativeRedstone},
	{47, "书架", CreativeBlocks}, {48, "苔石", CreativeBlocks}, {49, "黑曜石", CreativeBlocks}, {50, "火把", CreativeRedstone}, {52, "刷怪笼", CreativeBlocks},
	{54, "箱子", CreativeBlocks}, {56, "钻石矿石", CreativeBlocks}, {57, "钻石块", CreativeBlocks}, {58, "工作台", CreativeBlocks},
	{61, "熔炉", CreativeRedstone}, {65, "梯子", CreativeFunctional}, {73, "红石矿石", CreativeRedstone}, {74, "青金石矿石", CreativeBlocks},
	{81, "仙人掌", CreativeNatural}, {83, "甘蔗", CreativeNatural}, {84, "唱片机", CreativeFunctional}, {86, "南瓜", CreativeNatural}, {87, "下界岩", CreativeBlocks},
	{79, "冰", CreativeNatural}, {80, "雪块", CreativeNatural}, {82, "黏土块", CreativeBlocks}, {88, "灵魂沙", CreativeNatural}, {89, "荧石", CreativeBlocks}, {98, "石砖", CreativeBlocks}, {103, "西瓜", CreativeNatural},
	{91, "南瓜灯", CreativeFunctional}, {111, "睡莲", CreativeNatural}, {112, "下界砖块", CreativeBlocks}, {120, "末地传送门框架", CreativeFunctional}, {121, "末地石", CreativeBlocks},
	{122, "龙蛋", CreativeFunctional}, {124, "红石灯", CreativeRedstone}, {126, "橡木台阶", CreativeBlocks}, {129, "绿宝石矿石", CreativeBlocks}, {133, "绿宝石块", CreativeBlocks}, {138, "信标", CreativeFunctional},
	{150, "深板岩", CreativeBlocks}, {152, "红石块", CreativeRedstone}, {153, "投掷器", CreativeBlocks}, {154, "漏斗", CreativeBlocks},
	{155, "石英块", CreativeBlocks}, {159, "白色陶瓦", CreativeDecoration}, {165, "黏液块", CreativeBlocks}, {168, "海晶石", CreativeBlocks}, {169, "海晶灯", CreativeFunctional}, {170, "干草块", CreativeBlocks}, {172, "陶瓦", CreativeDecoration}, {173, "煤炭块", CreativeBlocks}, {174, "浮冰", CreativeNatural}, {179, "红砂岩", CreativeBlocks},

	{268, "木剑", CreativeCombat}, {272, "石剑", CreativeCombat}, {267, "铁剑", CreativeCombat}, {276, "钻石剑", CreativeCombat},
	{283, "金剑", CreativeCombat}, {270, "木镐", CreativeTools}, {274, "石镐", CreativeTools}, {257, "铁镐", CreativeTools},
	{278, "钻石镐", CreativeTools}, {285, "金镐", CreativeTools}, {271, "木斧", CreativeTools}, {275, "石斧", CreativeTools},
	{258, "铁斧", CreativeTools}, {279, "钻石斧", CreativeTools}, {286, "金斧", CreativeTools}, {269, "木锹", CreativeTools},
	{273, "石锹", CreativeTools}, {256, "铁锹", CreativeTools}, {277, "钻石锹", CreativeTools}, {284, "金锹", CreativeTools},
	{290, "木锄", CreativeTools}, {291, "石锄", CreativeTools}, {292, "铁锄", CreativeTools}, {293, "钻石锄", CreativeTools},
	{294, "金锄", CreativeTools}, {261, "弓", CreativeCombat}, {259, "打火石", CreativeTools}, {359, "剪刀", CreativeTools},
	{298, "皮革帽子", CreativeCombat}, {299, "皮革外套", CreativeCombat}, {300, "皮革裤子", CreativeCombat}, {301, "皮革靴子", CreativeCombat},
	{306, "铁头盔", CreativeCombat}, {307, "铁胸甲", CreativeCombat}, {308, "铁护腿", CreativeCombat}, {309, "铁靴子", CreativeCombat},
	{310, "钻石头盔", CreativeCombat}, {311, "钻石胸甲", CreativeCombat}, {312, "钻石护腿", CreativeCombat}, {313, "钻石靴子", CreativeCombat},
	{302, "锁链头盔", CreativeCombat}, {303, "锁链胸甲", CreativeCombat}, {304, "锁链护腿", CreativeCombat}, {305, "锁链靴子", CreativeCombat},
	{314, "金头盔", CreativeCombat}, {315, "金胸甲", CreativeCombat}, {316, "金护腿", CreativeCombat}, {317, "金靴子", CreativeCombat},
	{ItemCopperHelmet, "铜头盔", CreativeCombat}, {ItemCopperChestplate, "铜胸甲", CreativeCombat}, {ItemCopperLeggings, "铜护腿", CreativeCombat}, {ItemCopperBoots, "铜靴子", CreativeCombat},
	{ItemRawCopper, "粗铜", CreativeMaterials}, {ItemRawIron, "粗铁", CreativeMaterials}, {ItemRawGold, "粗金", CreativeMaterials}, {ItemEmerald, "绿宝石", CreativeMaterials}, {ItemLapisLazuli, "青金石", CreativeMaterials}, {ItemCopperIngot, "铜锭", CreativeMaterials}, {ItemNetheriteScrap, "下界合金碎片", CreativeMaterials},

	{262, "箭", CreativeMaterials}, {263, "煤炭", CreativeMaterials}, {264, "钻石", CreativeMaterials}, {265, "铁锭", CreativeMaterials}, {266, "金锭", CreativeMaterials},
	{280, "木棍", CreativeMaterials}, {331, "红石粉", CreativeRedstone}, {352, "骨头", CreativeMaterials}, {351, "骨粉", CreativeMaterials}, {288, "羽毛", CreativeMaterials},
	{289, "火药", CreativeMaterials}, {287, "线", CreativeMaterials}, {334, "皮革", CreativeMaterials}, {318, "燧石", CreativeMaterials}, {295, "小麦种子", CreativeMaterials}, {296, "小麦", CreativeMaterials},
	{322, "金苹果", CreativeFood}, {325, "铁桶", CreativeTools}, {326, "水桶", CreativeTools}, {327, "熔岩桶", CreativeTools}, {346, "钓鱼竿", CreativeTools},
	{332, "雪球", CreativeTools}, {344, "鸡蛋", CreativeTools}, {368, "末影珍珠", CreativeTools}, {324, "橡木门", CreativeFunctional},
	{330, "铁门", CreativeFunctional}, {336, "砖", CreativeMaterials}, {337, "黏土球", CreativeMaterials}, {340, "书", CreativeTools}, {345, "指南针", CreativeTools}, {371, "金粒", CreativeMaterials}, {375, "蜘蛛眼", CreativeFood}, {381, "末影之眼", CreativeTools}, {406, "下界石英", CreativeMaterials},
	{328, "矿车", CreativeFunctional}, {333, "橡木船", CreativeFunctional}, {342, "运输矿车", CreativeFunctional}, {343, "熔炉矿车", CreativeFunctional},

	{260, "苹果", CreativeFood}, {297, "面包", CreativeFood}, {319, "生猪排", CreativeFood}, {320, "熟猪排", CreativeFood},
	{363, "生牛肉", CreativeFood}, {364, "牛排", CreativeFood}, {367, "腐肉", CreativeFood}, {365, "生鸡肉", CreativeFood},
	{366, "熟鸡肉", CreativeFood}, {423, "生羊肉", CreativeFood}, {424, "熟羊肉", CreativeFood}, {360, "西瓜片", CreativeFood},
}
