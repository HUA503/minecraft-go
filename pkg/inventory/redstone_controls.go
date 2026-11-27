package inventory

import "fmt"

const (
	ItemStonePressurePlate         int32 = 70
	ItemOakPressurePlate           int32 = 72
	ItemLightWeightedPressurePlate int32 = 19402
	ItemHeavyWeightedPressurePlate int32 = 19403
	ItemStoneButton                int32 = 77
	ItemOakButton                  int32 = 143

	FirstWoodPressurePlateItem          int32 = 19200
	FirstWoodButtonItem                 int32 = 19300
	ItemPolishedBlackstonePressurePlate int32 = 19400
	ItemPolishedBlackstoneButton        int32 = 19401

	PressurePlateStateBase int32 = 46000
	ButtonStateBase        int32 = 47000
)

type controlBlockDefinition struct {
	itemID   int32
	name     string
	display  string
	texture  string
	source   string
	kind     string
	legacy   bool
	weighted bool
}

var pressurePlateDefinitions = buildControlBlockDefinitions("pressure_plate")
var weightedPressurePlateDefinitions = []controlBlockDefinition{
	{itemID: ItemLightWeightedPressurePlate, name: "minecraft:light_weighted_pressure_plate", display: "轻质测重压力板", texture: "gold_block", source: "minecraft:gold_ingot", kind: "pressure_plate", weighted: true},
	{itemID: ItemHeavyWeightedPressurePlate, name: "minecraft:heavy_weighted_pressure_plate", display: "重质测重压力板", texture: "iron_block", source: "minecraft:iron_ingot", kind: "pressure_plate", weighted: true},
}
var buttonDefinitions = buildControlBlockDefinitions("button")

func buildControlBlockDefinitions(kind string) []controlBlockDefinition {
	woods := []struct{ name, display string }{
		{"oak", "橡木"}, {"spruce", "云杉木"}, {"birch", "白桦木"}, {"jungle", "丛林木"},
		{"acacia", "金合欢木"}, {"dark_oak", "深色橡木"}, {"mangrove", "红树木"}, {"cherry", "樱花木"},
		{"pale_oak", "苍白橡木"}, {"poplar", "杨木"}, {"bamboo", "竹"}, {"crimson", "绯红菌柄"}, {"warped", "诡异菌柄"},
	}
	definitions := make([]controlBlockDefinition, 0, len(woods)+3)
	legacyStoneID, legacyOakID := ItemStonePressurePlate, ItemOakPressurePlate
	firstWoodID := FirstWoodPressurePlateItem
	displaySuffix := "压力板"
	if kind == "button" {
		legacyStoneID, legacyOakID = ItemStoneButton, ItemOakButton
		firstWoodID = FirstWoodButtonItem
		displaySuffix = "按钮"
	}
	stoneName, stoneDisplay, stoneTexture := "stone_"+kind, "石质"+displaySuffix, "stone"
	definitions = append(definitions, controlBlockDefinition{
		itemID: legacyStoneID, name: "minecraft:" + stoneName, display: stoneDisplay,
		texture: stoneTexture, source: "minecraft:stone", kind: kind, legacy: true,
	})
	for index, wood := range woods {
		itemID := firstWoodID + int32(index)
		legacy := false
		if index == 0 {
			itemID, legacy = legacyOakID, true
		} else {
			itemID-- // Oak keeps its legacy ID; keep all new variants compact.
		}
		texture := wood.name + "_planks"
		definitions = append(definitions, controlBlockDefinition{
			itemID: itemID, name: fmt.Sprintf("minecraft:%s_%s", wood.name, kind),
			display: wood.display + displaySuffix, texture: texture,
			source: "minecraft:" + texture, kind: kind, legacy: legacy,
		})
	}
	blackstoneID := ItemPolishedBlackstonePressurePlate
	if kind == "button" {
		blackstoneID = ItemPolishedBlackstoneButton
	}
	definitions = append(definitions, controlBlockDefinition{
		itemID: blackstoneID, name: "minecraft:polished_blackstone_" + kind,
		display: "磨制黑石" + displaySuffix, texture: "polished_blackstone",
		source: "minecraft:polished_blackstone", kind: kind,
	})
	return definitions
}

func controlBlockExtendedDefinitions(definitions []controlBlockDefinition) []ExtendedBlockDefinition {
	blocks := make([]ExtendedBlockDefinition, 0, len(definitions))
	for _, item := range definitions {
		extendedBlockDisplayNames[item.name] = item.display
		if item.legacy {
			continue
		}
		hardness := 0.5
		tool := ToolAxe
		if item.source == "minecraft:stone" || item.source == "minecraft:polished_blackstone" || item.weighted {
			tool = ToolPickaxe
		}
		blocks = append(blocks, ExtendedBlockDefinition{
			RuntimeID: item.itemID, Name: item.name,
			TextureTop: item.texture, TextureBottom: item.texture, TextureSide: item.texture,
			Category: CreativeRedstone, Hardness: hardness, HarvestTool: tool, DropRuntimeID: item.itemID,
		})
	}
	return blocks
}

func allPressurePlateDefinitions() []controlBlockDefinition {
	return append(append([]controlBlockDefinition(nil), pressurePlateDefinitions...), weightedPressurePlateDefinitions...)
}

func IsPressurePlateItem(id int32) bool {
	return controlDefinitionByID(allPressurePlateDefinitions(), id) != nil
}
func IsWeightedPressurePlate(id int32) bool {
	if controlDefinitionByID(weightedPressurePlateDefinitions, id) != nil {
		return true
	}
	itemID, _, ok := PressurePlateStateProperties(id)
	return ok && controlDefinitionByID(weightedPressurePlateDefinitions, itemID) != nil
}
func IsWoodenPressurePlate(id int32) bool {
	definition := controlDefinitionByID(pressurePlateDefinitions, id)
	return definition != nil && definition.source != "minecraft:stone" && definition.source != "minecraft:polished_blackstone"
}
func IsButtonItem(id int32) bool { return controlDefinitionByID(buttonDefinitions, id) != nil }

func IsPressurePlateBlock(id int32) bool {
	if IsPressurePlateItem(id) {
		return true
	}
	_, _, ok := PressurePlateStateProperties(id)
	return ok
}

func IsButtonBlock(id int32) bool {
	if IsButtonItem(id) {
		return true
	}
	_, _, _, ok := ButtonStateProperties(id)
	return ok
}

func IsRedstoneControlBlock(id int32) bool { return IsPressurePlateBlock(id) || IsButtonBlock(id) }

func controlDefinitionByID(definitions []controlBlockDefinition, id int32) *controlBlockDefinition {
	for index := range definitions {
		if definitions[index].itemID == id {
			return &definitions[index]
		}
	}
	return nil
}

func PressurePlateBlockState(itemID int32, powered bool) int32 {
	if controlDefinitionByID(weightedPressurePlateDefinitions, itemID) != nil {
		if powered {
			return WeightedPressurePlateBlockState(itemID, 15)
		}
		return WeightedPressurePlateBlockState(itemID, 0)
	}
	index := controlDefinitionIndex(pressurePlateDefinitions, itemID)
	if index < 0 {
		return 0
	}
	props := int32(index * 2)
	if powered {
		props++
	}
	return PressurePlateStateBase + props
}

func PressurePlateStateProperties(state int32) (itemID int32, powered, ok bool) {
	offset := state - PressurePlateStateBase
	if offset >= 0 && offset < int32(len(pressurePlateDefinitions)*2) {
		return pressurePlateDefinitions[offset/2].itemID, offset%2 != 0, true
	}
	weightedOffset := offset - int32(len(pressurePlateDefinitions)*2)
	if weightedOffset < 0 || weightedOffset >= int32(len(weightedPressurePlateDefinitions)*16) {
		return 0, false, false
	}
	return weightedPressurePlateDefinitions[weightedOffset/16].itemID, weightedOffset%16 > 0, true
}

func WeightedPressurePlateBlockState(itemID int32, signal int) int32 {
	index := controlDefinitionIndex(weightedPressurePlateDefinitions, itemID)
	if index < 0 {
		return 0
	}
	if signal < 0 {
		signal = 0
	} else if signal > 15 {
		signal = 15
	}
	return PressurePlateStateBase + int32(len(pressurePlateDefinitions)*2+index*16+signal)
}

func PressurePlateSignalStrength(state int32) int {
	offset := state - PressurePlateStateBase
	if offset >= 0 && offset < int32(len(pressurePlateDefinitions)*2) {
		if offset%2 != 0 {
			return 15
		}
		return 0
	}
	weightedOffset := offset - int32(len(pressurePlateDefinitions)*2)
	if weightedOffset < 0 || weightedOffset >= int32(len(weightedPressurePlateDefinitions)*16) {
		return 0
	}
	return int(weightedOffset % 16)
}

func WeightedPressurePlateSignal(itemID int32, entityCount int) int {
	if entityCount <= 0 {
		return 0
	}
	if itemID == ItemLightWeightedPressurePlate {
		if entityCount > 15 {
			return 15
		}
		return entityCount
	}
	if itemID == ItemHeavyWeightedPressurePlate {
		signal := (entityCount + 9) / 10
		if signal > 15 {
			return 15
		}
		return signal
	}
	return 0
}

func PressurePlatePlacementState(itemID int32) (int32, bool) {
	if !IsPressurePlateItem(itemID) {
		return 0, false
	}
	if controlDefinitionByID(weightedPressurePlateDefinitions, itemID) != nil {
		return WeightedPressurePlateBlockState(itemID, 0), true
	}
	return PressurePlateBlockState(itemID, false), true
}

func ButtonBlockState(itemID int32, facing int, powered bool) int32 {
	index := controlDefinitionIndex(buttonDefinitions, itemID)
	if index < 0 || facing < 0 || facing > 5 {
		return 0
	}
	props := int32(index*12 + facing*2)
	if powered {
		props++
	}
	return ButtonStateBase + props
}

func ButtonStateProperties(state int32) (itemID int32, facing int, powered, ok bool) {
	offset := state - ButtonStateBase
	if offset < 0 || offset >= int32(len(buttonDefinitions)*12) {
		return 0, 0, false, false
	}
	props := int(offset % 12)
	return buttonDefinitions[offset/12].itemID, props / 2, props%2 != 0, true
}

func ButtonPlacementState(itemID, faceX, faceY, faceZ int32) (int32, bool) {
	if !IsButtonItem(itemID) {
		return 0, false
	}
	facing := 1 // A top face is the fallback for calls without ray face data.
	switch {
	case faceY < 0:
		facing = 0
	case faceY > 0:
		facing = 1
	case faceZ < 0:
		facing = 2
	case faceZ > 0:
		facing = 3
	case faceX < 0:
		facing = 4
	case faceX > 0:
		facing = 5
	}
	return ButtonBlockState(itemID, facing, false), true
}

// ButtonSupportOffset returns the neighbor that must support a placed button.
// Facing points away from the supporting wall.
func ButtonSupportOffset(state int32) (x, y, z int32, ok bool) {
	_, facing, _, ok := ButtonStateProperties(state)
	if !ok {
		return 0, 0, 0, false
	}
	switch facing {
	case 0:
		return 0, 1, 0, true
	case 1:
		return 0, -1, 0, true
	case 2:
		return 0, 0, 1, true
	case 3:
		return 0, 0, -1, true
	case 4:
		return 1, 0, 0, true
	default:
		return -1, 0, 0, true
	}
}

func controlDefinitionIndex(definitions []controlBlockDefinition, id int32) int {
	for index, definition := range definitions {
		if definition.itemID == id {
			return index
		}
	}
	return -1
}

func PressurePlateModelBoxes(state int32) ([]BlockBox, bool) {
	_, powered, ok := PressurePlateStateProperties(state)
	if !ok {
		if IsPressurePlateItem(state) {
			state = PressurePlateBlockState(state, false)
			_, powered, ok = PressurePlateStateProperties(state)
		}
	}
	if !ok {
		return nil, false
	}
	height := float32(0.0625)
	if powered {
		height = float32(0.03125)
	}
	return []BlockBox{{MinX: .125, MinY: 0, MinZ: .125, MaxX: .875, MaxY: height, MaxZ: .875}}, true
}

func ButtonModelBoxes(state int32) ([]BlockBox, bool) {
	_, facing, powered, ok := ButtonStateProperties(state)
	if !ok {
		if IsButtonItem(state) {
			state = ButtonBlockState(state, 1, false)
			_, facing, powered, ok = ButtonStateProperties(state)
		}
	}
	if !ok {
		return nil, false
	}
	depth := float32(.125)
	if powered {
		depth = float32(.0625)
	}
	switch facing {
	case 0:
		return []BlockBox{{MinX: .375, MinY: 0, MinZ: .375, MaxX: .625, MaxY: depth, MaxZ: .625}}, true
	case 1:
		return []BlockBox{{MinX: .375, MinY: 1 - depth, MinZ: .375, MaxX: .625, MaxY: 1, MaxZ: .625}}, true
	case 2:
		return []BlockBox{{MinX: .375, MinY: .375, MinZ: 0, MaxX: .625, MaxY: .625, MaxZ: depth}}, true
	case 3:
		return []BlockBox{{MinX: .375, MinY: .375, MinZ: 1 - depth, MaxX: .625, MaxY: .625, MaxZ: 1}}, true
	case 4:
		return []BlockBox{{MinX: 0, MinY: .375, MinZ: .375, MaxX: depth, MaxY: .625, MaxZ: .625}}, true
	default:
		return []BlockBox{{MinX: 1 - depth, MinY: .375, MinZ: .375, MaxX: 1, MaxY: .625, MaxZ: .625}}, true
	}
}

func ControlBlockDefinitions() []ExtendedBlockDefinition {
	definitions := append(controlBlockExtendedDefinitions(allPressurePlateDefinitions()), controlBlockExtendedDefinitions(buttonDefinitions)...)
	return append([]ExtendedBlockDefinition(nil), definitions...)
}

func IsPressurePlateState(state int32) bool {
	_, _, ok := PressurePlateStateProperties(state)
	return ok
}
