package inventory

import "strings"

// SmeltingRecipe holds input item, output product, and experience earned.
type SmeltingRecipe struct {
	InputID    int32
	OutputID   int32
	Experience float32
}

// SmeltingRecipes registry for Minecraft standard furnace recipes.
var SmeltingRecipes = map[int32]SmeltingRecipe{
	15:              {InputID: 15, OutputID: ItemIronIngot, Experience: 0.7},          // Raw Iron / Iron Ore -> Iron Ingot
	14:              {InputID: 14, OutputID: 266, Experience: 1.0},                    // Gold Ore -> Gold Ingot (ID 266)
	ItemCobblestone: {InputID: ItemCobblestone, OutputID: ItemStone, Experience: 0.1}, // Cobblestone -> Smooth Stone
	ItemOakLog:      {InputID: ItemOakLog, OutputID: ItemCharcoal, Experience: 0.15},  // Oak Log -> Charcoal
	12:              {InputID: 12, OutputID: 20, Experience: 0.1},                     // Sand -> Glass (ID 20)
	319:             {InputID: 319, OutputID: 320, Experience: 0.35},                  // Raw Porkchop -> Cooked Porkchop
	363:             {InputID: 363, OutputID: 364, Experience: 0.35},                  // Raw Beef -> Steak
}

func init() {
	SmeltingRecipes[ItemRawIron] = SmeltingRecipe{InputID: ItemRawIron, OutputID: ItemIronIngot, Experience: 0.7}
	SmeltingRecipes[ItemRawGold] = SmeltingRecipe{InputID: ItemRawGold, OutputID: ItemGoldIngot, Experience: 1.0}
	SmeltingRecipes[ItemRawCopper] = SmeltingRecipe{InputID: ItemRawCopper, OutputID: ItemCopperIngot, Experience: 0.7}
	if ancientDebris, ok := ExtendedBlockByName("minecraft:ancient_debris"); ok {
		SmeltingRecipes[ancientDebris.RuntimeID] = SmeltingRecipe{InputID: ancientDebris.RuntimeID, OutputID: ItemNetheriteScrap, Experience: 2.0}
	}
	for _, block := range ExtendedBlockDefinitions() {
		name := strings.TrimPrefix(block.Name, "minecraft:")
		if (strings.HasSuffix(name, "_log") || strings.HasSuffix(name, "_wood")) &&
			!strings.HasPrefix(name, "crimson_") && !strings.HasPrefix(name, "warped_") {
			SmeltingRecipes[block.RuntimeID] = SmeltingRecipe{InputID: block.RuntimeID, OutputID: ItemCharcoal, Experience: 0.15}
		}
	}
	for _, recipe := range []struct {
		inputID int32
		output  string
	}{
		{ItemStone, "minecraft:smooth_stone"},
		{155, "minecraft:smooth_quartz"},
		{24, "minecraft:smooth_sandstone"},
		{179, "minecraft:smooth_red_sandstone"},
	} {
		if output, ok := ExtendedBlockByName(recipe.output); ok {
			SmeltingRecipes[recipe.inputID] = SmeltingRecipe{
				InputID: recipe.inputID, OutputID: output.RuntimeID, Experience: 0.1,
			}
		}
	}
}

// MatchSmelting looks up the smelted output product for an input item ID.
func MatchSmelting(inputID int32) (SmeltingRecipe, bool) {
	recipe, ok := SmeltingRecipes[inputID]
	return recipe, ok
}

// Fuel burn duration in ticks (20 ticks = 1 second)
var FuelBurnTimes = map[int32]int{
	ItemCoal:          1600,  // Coal: 80 seconds (smelts 8 items)
	ItemCharcoal:      1600,  // Charcoal: 80 seconds (smelts 8 items)
	ItemBlazeRod:      2400,  // Blaze rod: 120 seconds (smelts 12 items)
	ItemOakLog:        300,   // Wood log: 15 seconds (smelts 1.5 items)
	ItemOakPlanks:     300,   // Wood plank: 15 seconds (smelts 1.5 items)
	ItemStick:         100,   // Stick: 5 seconds (smelts 0.5 items)
	ItemCraftingTable: 300,   // Crafting table: 15 seconds
	ItemWoodenPickaxe: 200,   // Wooden tools: 10 seconds
	325:               20000, // Lava Bucket: 1000 seconds (smelts 100 items)
}

// GetFuelBurnTime returns how many ticks the fuel burns for.
func GetFuelBurnTime(fuelID int32) int {
	return FuelBurnTimes[fuelID]
}

// FurnaceBlockEntity represents a functional Minecraft furnace.
type FurnaceBlockEntity struct {
	InputSlot         ItemStack
	FuelSlot          ItemStack
	OutputSlot        ItemStack
	BurnTimeRemaining int // Ticks left on currently active fuel
	MaxBurnTime       int // Total ticks of currently active fuel
	CookTimeProgress  int // Progress towards current smelted item (0..200)
	CookTimeTotal     int // Standard 200 ticks (10 seconds)
}

// NewFurnaceBlockEntity initializes an empty furnace block entity.
func NewFurnaceBlockEntity() *FurnaceBlockEntity {
	return &FurnaceBlockEntity{
		CookTimeTotal: 200, // 200 ticks = 10.0 seconds per item
	}
}

// IsBurning returns true if the furnace is currently lit with fuel.
func (f *FurnaceBlockEntity) IsBurning() bool {
	return f.BurnTimeRemaining > 0
}

// Tick processes one game tick (1/20 second) of furnace operation.
// Returns true if state changed (e.g. cooked item or fuel consumed).
func (f *FurnaceBlockEntity) Tick() bool {
	changed := false

	// 1. Decrement fuel burn time if lit
	if f.BurnTimeRemaining > 0 {
		f.BurnTimeRemaining--
		changed = true
	}

	// 2. Check if input can be smelted
	recipe, canSmelt := MatchSmelting(f.InputSlot.ItemID)
	canOutput := false
	if canSmelt && !f.InputSlot.IsEmpty() {
		// Can output if output slot is empty or matching item below stack limit 64
		if f.OutputSlot.IsEmpty() {
			canOutput = true
		} else if f.OutputSlot.ItemID == recipe.OutputID && f.OutputSlot.Count < 64 {
			canOutput = true
		}
	}

	// 3. If furnace is not lit, try to ignite fuel if cooking is needed
	if f.BurnTimeRemaining <= 0 && canOutput {
		fuelDuration := GetFuelBurnTime(f.FuelSlot.ItemID)
		if fuelDuration > 0 && f.FuelSlot.Count > 0 {
			f.BurnTimeRemaining = fuelDuration
			f.MaxBurnTime = fuelDuration
			f.FuelSlot.Count--
			if f.FuelSlot.Count <= 0 {
				f.FuelSlot = ItemStack{}
			}
			changed = true
		}
	}

	// 4. Progress cooking if currently lit
	if f.IsBurning() && canOutput {
		f.CookTimeProgress++
		changed = true

		if f.CookTimeProgress >= f.CookTimeTotal {
			f.CookTimeProgress = 0

			// Consume 1 input item
			f.InputSlot.Count--
			if f.InputSlot.Count <= 0 {
				f.InputSlot = ItemStack{}
			}

			// Add 1 output product
			if f.OutputSlot.IsEmpty() {
				f.OutputSlot = ItemStack{ItemID: recipe.OutputID, Count: 1}
			} else {
				f.OutputSlot.Count++
			}
		}
	} else if !canOutput && f.CookTimeProgress > 0 {
		// Slowly decay cook progress if recipe became invalid
		f.CookTimeProgress--
		if f.CookTimeProgress < 0 {
			f.CookTimeProgress = 0
		}
		changed = true
	}

	return changed
}
