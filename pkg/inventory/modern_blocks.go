package inventory

// ExtendedBlockDefinition provides namespaced identity for recently added
// vanilla blocks while the rest of the game is migrated off legacy numeric IDs.
// RuntimeID is an internal save/protocol value and is not a Mojang registry ID.
type ExtendedBlockDefinition struct {
	RuntimeID     int32
	Name          string
	TextureTop    string
	TextureBottom string
	TextureSide   string
	Category      CreativeCategory
}

var extendedBlockDefinitions = append([]ExtendedBlockDefinition{
	{10000, "minecraft:cinnabar", "cinnabar", "cinnabar", "cinnabar", CreativeBlocks},
	{10001, "minecraft:polished_cinnabar", "polished_cinnabar", "polished_cinnabar", "polished_cinnabar", CreativeBlocks},
	{10002, "minecraft:chiseled_cinnabar", "chiseled_cinnabar", "chiseled_cinnabar", "chiseled_cinnabar", CreativeBlocks},
	{10003, "minecraft:cinnabar_bricks", "cinnabar_bricks", "cinnabar_bricks", "cinnabar_bricks", CreativeBlocks},
	{10004, "minecraft:sulfur", "sulfur", "sulfur", "sulfur", CreativeBlocks},
	{10005, "minecraft:polished_sulfur", "polished_sulfur", "polished_sulfur", "polished_sulfur", CreativeBlocks},
	{10006, "minecraft:chiseled_sulfur", "chiseled_sulfur", "chiseled_sulfur", "chiseled_sulfur", CreativeBlocks},
	{10007, "minecraft:sulfur_bricks", "sulfur_bricks", "sulfur_bricks", "sulfur_bricks", CreativeBlocks},
	{10008, "minecraft:potent_sulfur", "potent_sulfur", "potent_sulfur", "potent_sulfur", CreativeBlocks},
	{10009, "minecraft:poplar_planks", "poplar_planks", "poplar_planks", "poplar_planks", CreativeBlocks},
	{10010, "minecraft:poplar_log", "poplar_log_top", "poplar_log_top", "poplar_log", CreativeBlocks},
	{10011, "minecraft:stripped_poplar_log", "stripped_poplar_log_top", "stripped_poplar_log_top", "stripped_poplar_log", CreativeBlocks},
	{10060, "minecraft:red_poplar_leaves", "red_poplar_leaves", "red_poplar_leaves", "red_poplar_leaves", CreativeNatural},
	{10061, "minecraft:orange_poplar_leaves", "orange_poplar_leaves", "orange_poplar_leaves", "orange_poplar_leaves", CreativeNatural},
	{10062, "minecraft:yellow_poplar_leaves", "yellow_poplar_leaves", "yellow_poplar_leaves", "yellow_poplar_leaves", CreativeNatural},
	{10063, "minecraft:poplar_sapling", "poplar_sapling", "poplar_sapling", "poplar_sapling", CreativeNatural},
}, coloredBlockDefinitions()...)

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

var extendedBlockByID = func() map[int32]ExtendedBlockDefinition {
	definitions := make(map[int32]ExtendedBlockDefinition, len(extendedBlockDefinitions))
	for _, definition := range extendedBlockDefinitions {
		definitions[definition.RuntimeID] = definition
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

func IsExtendedLeaves(runtimeID int32) bool {
	return runtimeID >= 10060 && runtimeID <= 10062
}

func IsExtendedFoliage(runtimeID int32) bool {
	return runtimeID == 10063
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
}
