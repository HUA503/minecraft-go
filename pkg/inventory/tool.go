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
	ItemIronPickaxe:    {Type: ToolPickaxe, Tier: TierIron, MaxDurability: 250, MiningSpeed: 6.0, HarvestLevel: 2, AttackDamage: 4.0},
	ItemDiamondPickaxe: {Type: ToolPickaxe, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 8.0, HarvestLevel: 3, AttackDamage: 5.0},

	// Swords (ID: 268 Wood, 272 Stone, 267 Iron, 276 Diamond)
	268: {Type: ToolSword, Tier: TierWood, MaxDurability: 59, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 4.0},
	272: {Type: ToolSword, Tier: TierStone, MaxDurability: 131, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 5.0},
	267: {Type: ToolSword, Tier: TierIron, MaxDurability: 250, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 6.0},
	276: {Type: ToolSword, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 1.5, HarvestLevel: 0, AttackDamage: 7.0},

	// Axes (ID: 271 Wood, 275 Stone, 258 Iron, 279 Diamond)
	271: {Type: ToolAxe, Tier: TierWood, MaxDurability: 59, MiningSpeed: 2.0, HarvestLevel: 0, AttackDamage: 7.0},
	275: {Type: ToolAxe, Tier: TierStone, MaxDurability: 131, MiningSpeed: 4.0, HarvestLevel: 1, AttackDamage: 9.0},
	258: {Type: ToolAxe, Tier: TierIron, MaxDurability: 250, MiningSpeed: 6.0, HarvestLevel: 2, AttackDamage: 9.0},
	279: {Type: ToolAxe, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 8.0, HarvestLevel: 3, AttackDamage: 9.0},

	// Shovels (ID: 269 Wood, 273 Stone, 256 Iron, 277 Diamond)
	269: {Type: ToolShovel, Tier: TierWood, MaxDurability: 59, MiningSpeed: 2.0, HarvestLevel: 0, AttackDamage: 2.5},
	273: {Type: ToolShovel, Tier: TierStone, MaxDurability: 131, MiningSpeed: 4.0, HarvestLevel: 1, AttackDamage: 3.5},
	256: {Type: ToolShovel, Tier: TierIron, MaxDurability: 250, MiningSpeed: 6.0, HarvestLevel: 2, AttackDamage: 4.5},
	277: {Type: ToolShovel, Tier: TierDiamond, MaxDurability: 1561, MiningSpeed: 8.0, HarvestLevel: 3, AttackDamage: 5.5},
}

// GetToolProperties returns tool specs if item is an authentic tool.
func GetToolProperties(itemID int32) (ToolProperties, bool) {
	props, ok := ToolRegistry[itemID]
	return props, ok
}

// BlockRequiredHarvestLevel specifies the minimum tool tier needed to harvest item drops.
var BlockRequiredHarvestLevel = map[int32]int{
	ItemStone:         0, // Any pickaxe
	ItemCobblestone:   0, // Any pickaxe
	16:                0, // Coal ore: Any pickaxe
	15:                1, // Iron ore: Stone pickaxe or better
	14:                2, // Gold ore: Iron pickaxe or better
	56:                2, // Diamond ore: Iron pickaxe or better
	49:                3, // Obsidian: Diamond pickaxe only
}

// IsPickaxeEffective returns true if block is best mined by a pickaxe.
func IsPickaxeEffective(blockID int32) bool {
	switch blockID {
	case ItemStone, ItemCobblestone, ItemFurnace, 14, 15, 16, 56, 49:
		return true
	default:
		return false
	}
}

// IsAxeEffective returns true if block is best mined by an axe.
func IsAxeEffective(blockID int32) bool {
	switch blockID {
	case ItemOakLog, ItemOakPlanks, ItemCraftingTable:
		return true
	default:
		return false
	}
}

// IsShovelEffective returns true if block is best mined by a shovel.
func IsShovelEffective(blockID int32) bool {
	switch blockID {
	case ItemDirt, ItemGrass, 12, 13: // Dirt, Grass, Sand, Gravel
		return true
	default:
		return false
	}
}

// CanHarvestBlock checks if current tool tier is sufficient to drop items.
func CanHarvestBlock(toolID, blockID int32) bool {
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

// ApplyToolWear deducts 1 durability point. Returns true if tool broke.
func ApplyToolWear(stack *ItemStack) bool {
	if stack == nil || stack.IsEmpty() {
		return false
	}
	props, isTool := GetToolProperties(stack.ItemID)
	if !isTool {
		return false
	}

	stack.Damage++
	if stack.Damage >= props.MaxDurability {
		// Tool broken! Reset to empty air
		stack.ItemID = ItemAir
		stack.Count = 0
		stack.Damage = 0
		return true
	}
	return false
}
