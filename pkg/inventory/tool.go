package inventory

// ToolType categorizes Minecraft tool categories.
type ToolType int

const (
	ToolNone ToolType = iota
	ToolPickaxe
	ToolAxe
	ToolShovel
	ToolSword
	ToolHoe
)

// ToolTier defines the material quality and capabilities.
type ToolTier int

const (
	TierHand ToolTier = iota
	TierWood
	TierStone
	TierIron
	TierDiamond
	TierNetherite
	TierGold
	TierCopper
)

// ToolProperties defines durability, mining speed multiplier, and harvest level.
type ToolProperties struct {
	Type          ToolType
	Tier          ToolTier
	MaxDurability int
	MiningSpeed   float64
	HarvestLevel  int // 0: Wood, 1: Stone, 2: Iron, 3: Diamond
	AttackDamage  float32
}

// Tool Registry for Minecraft standard tools.
var ToolRegistry = map[int32]ToolProperties{
	// Pickaxes
	ItemWoodenPickaxe:  {Type: ToolPickaxe, Tier: TierWood, MaxDurability: 59, MiningSpeed: 2.0, HarvestLevel: 0, AttackDamage: 2.0},
	ItemStonePickaxe:   {Type: ToolPickaxe, Tier: TierStone, MaxDurability: 131, MiningSpeed: 4.0, HarvestLevel: 1, AttackDamage: 3.0},
	ItemCopperPickaxe:  {Type: ToolPickaxe, Tier: TierCopper, MaxDurability: 190, MiningSpeed: 5.0, HarvestLevel: 1, AttackDamage: 2.0},
	ItemIronPickaxe:    {Type: ToolPickaxe, Tier: TierIron, MaxDurability: 250, MiningSpeed: 6.0, HarvestLevel: 2, AttackDamage: 4.0},
	ItemDiamondPickaxe: {Type: ToolPickaxe, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 8.0, HarvestLevel: 3, AttackDamage: 5.0},
	285:                {Type: ToolPickaxe, Tier: TierGold, MaxDurability: 32, MiningSpeed: 12.0, HarvestLevel: 0, AttackDamage: 2.0}, // Golden Pickaxe

	// Swords (ID: 268 Wood, 272 Stone, 267 Iron, 276 Diamond, 283 Gold)
	268:                  {Type: ToolSword, Tier: TierWood, MaxDurability: 59, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 4.0},
	272:                  {Type: ToolSword, Tier: TierStone, MaxDurability: 131, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 5.0},
	ItemCopperSword:      {Type: ToolSword, Tier: TierCopper, MaxDurability: 190, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 4.0},
	267:                  {Type: ToolSword, Tier: TierIron, MaxDurability: 250, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 6.0},
	276:                  {Type: ToolSword, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 7.0},
	283:                  {Type: ToolSword, Tier: TierGold, MaxDurability: 32, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 4.0}, // Golden Sword
	ItemNetheriteSword:   {Type: ToolSword, Tier: TierNetherite, MaxDurability: 2031, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 8.0},
	ItemNetheritePickaxe: {Type: ToolPickaxe, Tier: TierNetherite, MaxDurability: 2031, MiningSpeed: 9.0, HarvestLevel: 3, AttackDamage: 6.0},
	ItemNetheriteAxe:     {Type: ToolAxe, Tier: TierNetherite, MaxDurability: 2031, MiningSpeed: 9.0, HarvestLevel: 3, AttackDamage: 10.0},
	ItemNetheriteShovel:  {Type: ToolShovel, Tier: TierNetherite, MaxDurability: 2031, MiningSpeed: 9.0, HarvestLevel: 3, AttackDamage: 6.5},
	ItemNetheriteHoe:     {Type: ToolHoe, Tier: TierNetherite, MaxDurability: 2031, MiningSpeed: 9.0, HarvestLevel: 3, AttackDamage: 1.0},

	// Axes (ID: 271 Wood, 275 Stone, 258 Iron, 279 Diamond, 286 Gold)
	271:           {Type: ToolAxe, Tier: TierWood, MaxDurability: 59, MiningSpeed: 2.0, HarvestLevel: 0, AttackDamage: 7.0},
	275:           {Type: ToolAxe, Tier: TierStone, MaxDurability: 131, MiningSpeed: 4.0, HarvestLevel: 1, AttackDamage: 9.0},
	ItemCopperAxe: {Type: ToolAxe, Tier: TierCopper, MaxDurability: 190, MiningSpeed: 5.0, HarvestLevel: 1, AttackDamage: 8.0},
	258:           {Type: ToolAxe, Tier: TierIron, MaxDurability: 250, MiningSpeed: 6.0, HarvestLevel: 2, AttackDamage: 9.0},
	279:           {Type: ToolAxe, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 8.0, HarvestLevel: 3, AttackDamage: 9.0},
	286:           {Type: ToolAxe, Tier: TierGold, MaxDurability: 32, MiningSpeed: 12.0, HarvestLevel: 0, AttackDamage: 7.0}, // Golden Axe

	// Shovels (ID: 269 Wood, 273 Stone, 256 Iron, 277 Diamond, 284 Gold)
	269:              {Type: ToolShovel, Tier: TierWood, MaxDurability: 59, MiningSpeed: 2.0, HarvestLevel: 0, AttackDamage: 2.5},
	273:              {Type: ToolShovel, Tier: TierStone, MaxDurability: 131, MiningSpeed: 4.0, HarvestLevel: 1, AttackDamage: 3.5},
	ItemCopperShovel: {Type: ToolShovel, Tier: TierCopper, MaxDurability: 190, MiningSpeed: 5.0, HarvestLevel: 1, AttackDamage: 2.5},
	256:              {Type: ToolShovel, Tier: TierIron, MaxDurability: 250, MiningSpeed: 6.0, HarvestLevel: 2, AttackDamage: 4.5},
	277:              {Type: ToolShovel, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 8.0, HarvestLevel: 3, AttackDamage: 5.5},
	284:              {Type: ToolShovel, Tier: TierGold, MaxDurability: 32, MiningSpeed: 12.0, HarvestLevel: 0, AttackDamage: 2.5}, // Golden Shovel

	// Hoes (ID: 290 Wood, 291 Stone, 292 Iron, 293 Diamond, 294 Gold)
	290:           {Type: ToolHoe, Tier: TierWood, MaxDurability: 59, MiningSpeed: 1.0, HarvestLevel: 0, AttackDamage: 1.0},
	291:           {Type: ToolHoe, Tier: TierStone, MaxDurability: 131, MiningSpeed: 1.0, HarvestLevel: 0, AttackDamage: 1.0},
	ItemCopperHoe: {Type: ToolHoe, Tier: TierCopper, MaxDurability: 190, MiningSpeed: 5.0, HarvestLevel: 0, AttackDamage: 0.0},
	292:           {Type: ToolHoe, Tier: TierIron, MaxDurability: 250, MiningSpeed: 1.0, HarvestLevel: 0, AttackDamage: 1.0},
	293:           {Type: ToolHoe, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 1.0, HarvestLevel: 0, AttackDamage: 1.0},
	294:           {Type: ToolHoe, Tier: TierGold, MaxDurability: 32, MiningSpeed: 1.0, HarvestLevel: 0, AttackDamage: 1.0},

	// Shears (ID: 359)
	359: {Type: ToolNone, Tier: TierIron, MaxDurability: 238, MiningSpeed: 15.0, HarvestLevel: 0, AttackDamage: 1.0},
}

// GetToolProperties returns tool specs if item is an authentic tool.
func GetToolProperties(itemID int32) (ToolProperties, bool) {
	props, ok := ToolRegistry[itemID]
	return props, ok
}

// BlockRequiredHarvestLevel specifies the minimum tool tier needed to harvest item drops.
var BlockRequiredHarvestLevel = map[int32]int{
	ItemStone:       0, // Any pickaxe
	ItemStoneSlab:   0, // Any pickaxe
	ItemCobblestone: 0, // Any pickaxe
	16:              0, // Coal ore: Any pickaxe
	15:              1, // Iron ore: Stone pickaxe or better
	14:              2, // Gold ore: Iron pickaxe or better
	56:              2, // Diamond ore: Iron pickaxe or better
	73:              2, // Redstone ore: Iron pickaxe or better
	74:              1, // Lapis lazuli ore: Stone pickaxe or better
	129:             2, // Emerald ore: Iron pickaxe or better
	49:              3, // Obsidian: Diamond pickaxe only
}

// IsPickaxeEffective returns true if block is best mined by a pickaxe.
func IsPickaxeEffective(blockID int32) bool {
	blockID = BlockItemForState(blockID)
	if definition, ok := ExtendedBlockByID(blockID); ok {
		return definition.HarvestTool == ToolPickaxe
	}
	switch blockID {
	case ItemStone, ItemStoneSlab, ItemCobblestone, ItemFurnace, 14, 15, 16, 56, 73, 74, 129, 49:
		return true
	default:
		return false
	}
}

// IsAxeEffective returns true if block is best mined by an axe.
func IsAxeEffective(blockID int32) bool {
	blockID = BlockItemForState(blockID)
	if definition, ok := ExtendedBlockByID(blockID); ok {
		return definition.HarvestTool == ToolAxe
	}
	switch blockID {
	case ItemOakLog, 22, ItemOakPlanks, ItemOakSlab, ItemCraftingTable:
		return true
	default:
		return false
	}
}

// IsShovelEffective returns true if block is best mined by a shovel.
func IsShovelEffective(blockID int32) bool {
	blockID = BlockItemForState(blockID)
	if definition, ok := ExtendedBlockByID(blockID); ok {
		return definition.HarvestTool == ToolShovel
	}
	switch blockID {
	case ItemDirt, ItemGrass, 12, 13: // Dirt, Grass, Sand, Gravel
		return true
	default:
		return false
	}
}

// BlockHardness defines canonical Minecraft Java Edition block hardness values.
// Negative values (-1.0) indicate unbreakable blocks such as Bedrock.
var BlockHardness = map[int32]float64{
	0:   0.0,   // Air
	1:   1.5,   // Stone
	2:   0.6,   // Grass Block
	3:   0.5,   // Dirt
	4:   2.0,   // Cobblestone
	5:   2.0,   // Oak Planks
	44:  2.0,   // Stone Slab
	126: 2.0,   // Oak Slab
	7:   -1.0,  // Bedrock (unbreakable)
	8:   100.0, // Water
	9:   100.0, // Flowing Water
	10:  100.0, // Lava
	11:  100.0, // Flowing Lava
	12:  0.5,   // Sand
	13:  0.6,   // Gravel
	14:  3.0,   // Gold Ore
	15:  3.0,   // Iron Ore
	16:  3.0,   // Coal Ore
	17:  2.0,   // Oak Log
	18:  0.2,   // Oak Leaves
	20:  0.3,   // Glass
	31:  0.0,   // Tall Grass (instant)
	32:  0.0,   // Dead Bush (instant)
	37:  0.0,   // Dandelion (instant)
	38:  0.0,   // Poppy (instant)
	49:  50.0,  // Obsidian
	50:  0.0,   // Torch (instant)
	56:  3.0,   // Diamond Ore
	58:  2.5,   // Crafting Table
	61:  3.5,   // Furnace
	87:  0.4,   // Netherrack
	89:  0.3,   // Glowstone
	121: 3.0,   // End Stone
	150: 3.0,   // Deepslate
}

// GetBlockHardness returns the hardness of a block ID.
func GetBlockHardness(blockID int32) float64 {
	blockID = BlockItemForState(blockID)
	if IsDecorativePlantBlock(blockID) {
		return 0 // Vanilla small decorative plants break instantly.
	}
	if definition, ok := ExtendedBlockByID(blockID); ok && definition.Hardness > 0 {
		return definition.Hardness
	}
	h, ok := BlockHardness[blockID]
	if !ok {
		return 1.5 // Default standard block hardness
	}
	return h
}

// CalculateBlockDamagePerSecond computes fractional damage (0.0 to 1.0) inflicted per second.
// Returns 0.0 for unbreakable blocks, and >=100.0 for instantly breakable blocks.
func CalculateBlockDamagePerSecond(toolID, blockID int32) float64 {
	hardness := GetBlockHardness(blockID)
	if hardness < 0 {
		return 0.0 // Bedrock is unbreakable
	}
	if hardness == 0 {
		return 100.0 // Instant break (e.g. torch, flowers)
	}

	canHarvest := CanHarvestBlock(toolID, blockID)
	speedMultiplier := CalculateMiningSpeedMultiplier(toolID, blockID)

	canHarvestFactor := 30.0
	if !canHarvest {
		canHarvestFactor = 100.0
	}

	damagePerTick := speedMultiplier / (hardness * canHarvestFactor)
	return damagePerTick * 20.0 // 20 ticks per second
}

// CanHarvestBlock checks if current tool tier is sufficient to drop items.
func CanHarvestBlock(toolID, blockID int32) bool {
	blockID = BlockItemForState(blockID)
	if definition, ok := ExtendedBlockByID(blockID); ok && definition.RequiresHarvestTool {
		tool, isTool := GetToolProperties(toolID)
		return isTool && tool.Type == definition.HarvestTool && tool.HarvestLevel >= definition.RequiredHarvestLevel
	}
	reqTier, requiresTool := BlockRequiredHarvestLevel[blockID]
	if !requiresTool {
		return true // Dirt, wood, etc. can be harvested by hand
	}

	tool, isTool := GetToolProperties(toolID)
	if !isTool || tool.Type != ToolPickaxe {
		return false // Rock/ores strictly require a pickaxe
	}

	return tool.HarvestLevel >= reqTier
}

// CalculateMiningSpeedMultiplier computes speed scaling factor.
func CalculateMiningSpeedMultiplier(toolID, blockID int32) float64 {
	tool, isTool := GetToolProperties(toolID)
	if !isTool {
		return 1.0 // Bare hand base speed
	}

	effective := false
	switch tool.Type {
	case ToolPickaxe:
		effective = IsPickaxeEffective(blockID)
	case ToolAxe:
		effective = IsAxeEffective(blockID)
	case ToolShovel:
		effective = IsShovelEffective(blockID)
	case ToolSword:
		if blockID == 30 { // Cobweb
			return 15.0
		}
	}

	if effective {
		return tool.MiningSpeed
	}
	return 1.0
}

// ApplyToolWear deducts 1 durability point. Returns true if the tool broke.
func ApplyToolWear(stack *ItemStack) bool {
	return ApplyToolWearAmount(stack, 1)
}

// ApplyToolWearAmount deducts the requested durability amount. Returns true if
// the tool broke.
func ApplyToolWearAmount(stack *ItemStack, amount int) bool {
	if stack == nil || stack.IsEmpty() {
		return false
	}
	if amount <= 0 {
		return false
	}
	props, isTool := GetToolProperties(stack.ItemID)
	if !isTool {
		return false
	}

	stack.Damage += amount
	if stack.Damage >= props.MaxDurability {
		// Tool broken! Reset to empty air
		stack.ItemID = ItemAir
		stack.Count = 0
		stack.Damage = 0
		return true
	}
	return false
}
