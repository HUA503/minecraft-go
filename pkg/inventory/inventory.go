package inventory

import (
	"errors"
	"fmt"
)

// Item definition.
type Item struct {
	ID           int32
	Name         string
	MaxStackSize int
	IsTool       bool
	MaxDamage    int
}

// Common item catalog
const (
	ItemAir            int32 = 0
	ItemStone          int32 = 1
	ItemGrass          int32 = 2
	ItemDirt           int32 = 3
	ItemCobblestone    int32 = 4
	ItemOakPlanks      int32 = 5
	ItemOakLog         int32 = 17
	ItemCraftingTable  int32 = 58
	ItemFurnace        int32 = 61
	ItemTorch          int32 = 50
	ItemStick          int32 = 280
	ItemWoodenPickaxe  int32 = 270
	ItemWoodenSword    int32 = 268
	ItemWoodenShovel   int32 = 269
	ItemWoodenAxe      int32 = 271
	ItemStonePickaxe   int32 = 274
	ItemStoneSword     int32 = 272
	ItemStoneShovel    int32 = 273
	ItemStoneAxe       int32 = 275
	ItemIronPickaxe    int32 = 257
	ItemIronSword      int32 = 267
	ItemIronShovel     int32 = 256
	ItemIronAxe        int32 = 258
	ItemDiamondPickaxe int32 = 278
	ItemDiamondSword   int32 = 276
	ItemDiamondShovel  int32 = 277
	ItemDiamondAxe     int32 = 279
	ItemIronIngot      int32 = 265
	ItemGoldIngot      int32 = 266
	ItemDiamond        int32 = 264
	ItemCoal           int32 = 263
	ItemApple          int32 = 260
	ItemBread          int32 = 297
	ItemIronBlock      int32 = 42
	ItemGoldBlock      int32 = 41
	ItemDiamondBlock   int32 = 57
	ItemChest          int32 = 54
	ItemRawPorkchop    int32 = 319
	ItemCookedPorkchop int32 = 320
	ItemRawBeef        int32 = 363
	ItemSteak          int32 = 364
	ItemRottenFlesh    int32 = 367
	ItemEnderPearl     int32 = 368
	ItemEyeOfEnder     int32 = 381
	ItemString         int32 = 287
	ItemSpiderEye      int32 = 375
	ItemGunpowder      int32 = 289
	ItemBone           int32 = 352
	ItemBoneMeal       int32 = 351
	ItemArrow          int32 = 262
	ItemBow            int32 = 261
	ItemFlintAndSteel  int32 = 259
	ItemGoldNugget     int32 = 371
	ItemNetherQuartz   int32 = 406

	// Gold Tools
	ItemGoldenSword   int32 = 283
	ItemGoldenShovel  int32 = 284
	ItemGoldenPickaxe int32 = 285
	ItemGoldenAxe     int32 = 286
	ItemGoldenHoe     int32 = 294

	// Hoes
	ItemWoodenHoe  int32 = 290
	ItemStoneHoe   int32 = 291
	ItemIronHoe    int32 = 292
	ItemDiamondHoe int32 = 293

	// Armor
	ItemLeatherHelmet     int32 = 298
	ItemLeatherChestplate int32 = 299
	ItemLeatherLeggings   int32 = 300
	ItemLeatherBoots      int32 = 301

	ItemIronHelmet     int32 = 306
	ItemIronChestplate int32 = 307
	ItemIronLeggings   int32 = 308
	ItemIronBoots      int32 = 309

	ItemDiamondHelmet     int32 = 310
	ItemDiamondChestplate int32 = 311
	ItemDiamondLeggings   int32 = 312
	ItemDiamondBoots      int32 = 313

	ItemGoldenHelmet     int32 = 314
	ItemGoldenChestplate int32 = 315
	ItemGoldenLeggings   int32 = 316
	ItemGoldenBoots      int32 = 317

	// Copper armor uses project-local IDs because the legacy item-ID space is
	// occupied by the existing compatibility catalog.
	ItemCopperHelmet        int32 = 13000
	ItemCopperChestplate    int32 = 13001
	ItemCopperLeggings      int32 = 13002
	ItemCopperBoots         int32 = 13003
	ItemRawCopper           int32 = 13004
	ItemRawIron             int32 = 13005
	ItemRawGold             int32 = 13006
	ItemEmerald             int32 = 13007
	ItemCopperIngot         int32 = 13008
	ItemNetheriteScrap      int32 = 13009
	ItemLapisLazuli         int32 = 13010
	ItemNetheriteIngot      int32 = 13011
	ItemNetheriteHelmet     int32 = 13012
	ItemNetheriteChestplate int32 = 13013
	ItemNetheriteLeggings   int32 = 13014
	ItemNetheriteBoots      int32 = 13015

	// Utility, materials, and blocks
	ItemBucket        int32 = 325
	ItemWaterBucket   int32 = 326
	ItemLavaBucket    int32 = 327
	ItemWheat         int32 = 296
	ItemWheatSeeds    int32 = 295
	ItemFeather       int32 = 288
	ItemFlint         int32 = 318
	ItemRawChicken    int32 = 365
	ItemCookedChicken int32 = 366
	ItemRawMutton     int32 = 423
	ItemCookedMutton  int32 = 424
	ItemGoldenApple   int32 = 322
	ItemShears        int32 = 359
	ItemOakDoor       int32 = 324
	ItemIronDoor      int32 = 330
	ItemOakSlab       int32 = 126
	ItemStoneSlab     int32 = 44
	ItemLadder        int32 = 65
	ItemSnowball      int32 = 332
	ItemBrick         int32 = 336
	ItemClayBall      int32 = 337
	ItemBook          int32 = 340
	ItemSugarCane     int32 = 83
	ItemPumpkin       int32 = 86
	ItemMelon         int32 = 103
)

// IsPlaceableBlock reports whether an item ID corresponds to a placeable voxel block.
func IsPlaceableBlock(itemID int32) bool {
	if _, ok := ExtendedBlockByID(itemID); ok {
		return true
	}
	switch itemID {
	case ItemStone, ItemGrass, ItemDirt, ItemCobblestone, ItemOakPlanks,
		6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, ItemOakLog, 18, 19, 20, 21, 22, 23, 24, 35, 43,
		31, 32, 33, 37, 38, ItemGoldBlock, ItemIronBlock, ItemStoneSlab, 45, 46, 47, 48, 49, ItemTorch, 52, ItemChest, 56, ItemDiamondBlock, ItemCraftingTable, ItemFurnace,
		25, 65, 73, 74, 79, 80, 81, 82, 83, 84, 86, 87, 88, 89, 90, 91, 98, 103, 111, 112, 119, 120, 121, 122, 123, 124, 126, 129, 133, 138, 150, 152, 153, 154, 155, 159, 165, 168, 169, 170, 172, 173, 174, 179, 324, 330:
		return true
	default:
		return false
	}
}

// IsReplaceableBlock reports whether a block can be replaced by placing another block into its space.
func IsReplaceableBlock(blockID int32) bool {
	switch blockID {
	case 0, 6, 8, 9, 10, 11, 31, 32, 33, 37, 38, 50, 83, 111, 10063, 10100:
		return true
	default:
		return false
	}
}

// FoodProperties defines nutrition restored by edible items.
type FoodProperties struct {
	HungerRestored   int
	Saturation       float32
	RemainderItemID  int32
	AlwaysEdible     bool
	Effects          []FoodStatusEffect
	RemoveEffects    []string
	ClearAllEffects  bool
	TeleportRandomly bool
}

// FoodStatusEffect describes an effect added by a consumed item.
type FoodStatusEffect struct {
	ID            string
	DurationTicks int
	Amplifier     int
	Chance        float64
}

const (
	EffectHunger         = "minecraft:hunger"
	EffectPoison         = "minecraft:poison"
	EffectRegeneration   = "minecraft:regeneration"
	EffectResistance     = "minecraft:resistance"
	EffectFireResistance = "minecraft:fire_resistance"
	EffectAbsorption     = "minecraft:absorption"
	EffectNausea         = "minecraft:nausea"
)

// FoodRegistry defines vanilla Minecraft Java food nutrition and saturation parameters.
var FoodRegistry = map[int32]FoodProperties{
	ItemApple:          {HungerRestored: 4, Saturation: 2.4},
	ItemBread:          {HungerRestored: 5, Saturation: 6.0},
	ItemRawPorkchop:    {HungerRestored: 3, Saturation: 1.8},
	ItemCookedPorkchop: {HungerRestored: 8, Saturation: 12.8},
	ItemRawBeef:        {HungerRestored: 3, Saturation: 1.8},
	ItemSteak:          {HungerRestored: 8, Saturation: 12.8},
	ItemRottenFlesh:    {HungerRestored: 4, Saturation: 0.8, Effects: []FoodStatusEffect{{ID: EffectHunger, DurationTicks: 600, Chance: 0.8}}},
	360:                {HungerRestored: 2, Saturation: 1.2},
	ItemRawChicken:     {HungerRestored: 2, Saturation: 1.2, Effects: []FoodStatusEffect{{ID: EffectHunger, DurationTicks: 600, Chance: 0.3}}},
	ItemCookedChicken:  {HungerRestored: 6, Saturation: 7.2},
	ItemRawMutton:      {HungerRestored: 2, Saturation: 1.2},
	ItemCookedMutton:   {HungerRestored: 6, Saturation: 9.6},
	ItemGoldenApple:    {HungerRestored: 4, Saturation: 9.6, AlwaysEdible: true, Effects: []FoodStatusEffect{{ID: EffectRegeneration, DurationTicks: 100, Amplifier: 1}, {ID: EffectAbsorption, DurationTicks: 2400}}},
	ItemMelon:          {HungerRestored: 2, Saturation: 1.2},
	ItemSpiderEye:      {HungerRestored: 2, Saturation: 3.2, Effects: []FoodStatusEffect{{ID: EffectPoison, DurationTicks: 100}}},
}

func init() {
	for _, item := range java263FoodItems {
		// Cake is eaten as a placed block and golden dandelion is used on young
		// animals, so neither is an inventory-consumable food item.
		if item.Hunger == 0 && item.RuntimeID != ItemMilkBucket {
			continue
		}
		FoodRegistry[item.RuntimeID] = FoodProperties{
			HungerRestored:  item.Hunger,
			Saturation:      item.Saturation,
			RemainderItemID: item.RemainderItemID,
			AlwaysEdible:    item.AlwaysEdible,
		}
	}
	FoodRegistry[ItemEnchantedGoldenApple] = FoodProperties{
		HungerRestored: 4, Saturation: 9.6, AlwaysEdible: true,
		Effects: []FoodStatusEffect{
			{ID: EffectRegeneration, DurationTicks: 400, Amplifier: 1},
			{ID: EffectResistance, DurationTicks: 6000},
			{ID: EffectFireResistance, DurationTicks: 6000},
			{ID: EffectAbsorption, DurationTicks: 2400, Amplifier: 3},
		},
	}
	FoodRegistry[ItemChorusFruit] = FoodProperties{HungerRestored: 4, Saturation: 2.4, AlwaysEdible: true, TeleportRandomly: true}
	FoodRegistry[ItemPoisonousPotato] = FoodProperties{HungerRestored: 2, Saturation: 1.2, Effects: []FoodStatusEffect{{ID: EffectPoison, DurationTicks: 100, Chance: 0.6}}}
	FoodRegistry[ItemPufferfish] = FoodProperties{
		HungerRestored: 1, Saturation: 0.2,
		Effects: []FoodStatusEffect{
			{ID: EffectPoison, DurationTicks: 1200, Amplifier: 1},
			{ID: EffectHunger, DurationTicks: 300, Amplifier: 2},
			{ID: EffectNausea, DurationTicks: 300},
		},
	}
	FoodRegistry[ItemMilkBucket] = FoodProperties{RemainderItemID: 325, AlwaysEdible: true, ClearAllEffects: true}
	FoodRegistry[ItemHoneyBottle] = FoodProperties{
		HungerRestored: 6, Saturation: 1.2, RemainderItemID: 374, AlwaysEdible: true,
		RemoveEffects: []string{EffectPoison},
	}
}

// IsFood checks if given item ID is edible.
func IsFood(itemID int32) bool {
	_, ok := FoodRegistry[itemID]
	return ok
}

// ItemStack represents a stack of items in an inventory slot.
type ItemStack struct {
	ItemID int32
	Count  int
	Damage int
}

func (s ItemStack) IsEmpty() bool {
	return s.ItemID == ItemAir || s.Count <= 0
}

func (s ItemStack) String() string {
	if s.IsEmpty() {
		return "Empty"
	}
	return fmt.Sprintf("Item(ID=%d, Count=%d)", s.ItemID, s.Count)
}

// PlayerInventory manages hotbar, main storage, armor, and active slot.
type PlayerInventory struct {
	Slots          [36]ItemStack // 0..8 Hotbar, 9..35 Main Inventory
	Armor          [4]ItemStack  // boots, leggings, chestplate, helmet
	Offhand        ItemStack
	SelectedHotbar int // 0..8
}

// ArmorProtection summarizes the physical protection carried by equipped armor.
type ArmorProtection struct {
	Defense   int
	Toughness float32
}

// CalculateArmorProtection returns the summed vanilla armor and toughness values.
func CalculateArmorProtection(armor [4]ItemStack) ArmorProtection {
	var protection ArmorProtection
	for slot, stack := range armor {
		if stack.IsEmpty() || !CanEquipArmor(slot, stack.ItemID) {
			continue
		}
		switch stack.ItemID {
		case 301, 305, 317, 298:
			protection.Defense += 1 // leather boots, chain boots, gold boots, leather helmet
		case 313:
			protection.Defense += 3 // diamond boots
			protection.Toughness += 2
		case ItemNetheriteBoots:
			protection.Defense += 3
			protection.Toughness += 3
		case 309:
			protection.Defense += 2 // iron boots
		case 300:
			protection.Defense += 2 // leather leggings
		case 304:
			protection.Defense += 4 // chain leggings
		case 308:
			protection.Defense += 5 // iron leggings
		case 312:
			protection.Defense += 6 // diamond leggings
			protection.Toughness += 2
		case ItemNetheriteLeggings:
			protection.Defense += 6
			protection.Toughness += 3
		case 316:
			protection.Defense += 3 // gold leggings
		case 299:
			protection.Defense += 3 // leather chestplate
		case 303:
			protection.Defense += 5 // chain chestplate
		case 307:
			protection.Defense += 6 // iron chestplate
		case 311:
			protection.Defense += 8 // diamond chestplate
			protection.Toughness += 2
		case ItemNetheriteChestplate:
			protection.Defense += 8
			protection.Toughness += 3
		case 315:
			protection.Defense += 5 // gold chestplate
		case 302:
			protection.Defense += 2 // chain helmet
		case 306:
			protection.Defense += 2 // iron helmet
		case 310:
			protection.Defense += 3 // diamond helmet
			protection.Toughness += 2
		case ItemNetheriteHelmet:
			protection.Defense += 3
			protection.Toughness += 3
		case 314:
			protection.Defense += 2 // gold helmet
		case ItemCopperBoots:
			protection.Defense += 2
		case ItemCopperLeggings:
			protection.Defense += 4
		case ItemCopperChestplate:
			protection.Defense += 3
		case ItemCopperHelmet:
			protection.Defense += 1
		}
	}
	return protection
}

// ArmorSlotForItem maps the supported legacy armor IDs to vanilla equipment slots.
// Slot order is boots, leggings, chestplate, helmet.
func ArmorSlotForItem(itemID int32) (int, bool) {
	switch itemID {
	case 301, 305, 309, 313, 317, ItemCopperBoots, ItemNetheriteBoots:
		return 0, true
	case 300, 304, 308, 312, 316, ItemCopperLeggings, ItemNetheriteLeggings:
		return 1, true
	case 299, 303, 307, 311, 315, ItemCopperChestplate, ItemNetheriteChestplate:
		return 2, true
	case 298, 302, 306, 310, 314, ItemCopperHelmet, ItemNetheriteHelmet:
		return 3, true
	default:
		return 0, false
	}
}

// CanEquipArmor reports whether an item belongs in the specified equipment slot.
func CanEquipArmor(slot int, itemID int32) bool {
	itemSlot, ok := ArmorSlotForItem(itemID)
	return ok && slot == itemSlot
}

func NewPlayerInventory() *PlayerInventory {
	return &PlayerInventory{
		SelectedHotbar: 0,
	}
}

func (inv *PlayerInventory) GetSelectedItem() ItemStack {
	if inv.SelectedHotbar < 0 || inv.SelectedHotbar >= 9 {
		return ItemStack{}
	}
	return inv.Slots[inv.SelectedHotbar]
}

// SwapSelectedWithOffhand performs the Java F-key main-hand/offhand swap.
func (inv *PlayerInventory) SwapSelectedWithOffhand() bool {
	if inv == nil || inv.SelectedHotbar < 0 || inv.SelectedHotbar >= 9 {
		return false
	}
	selected := &inv.Slots[inv.SelectedHotbar]
	*selected, inv.Offhand = inv.Offhand, *selected
	return true
}

// GetMaxStackSize returns the canonical maximum stack size for the given item.
func GetMaxStackSize(itemID int32) int {
	if item, ok := FoodItemByID(itemID); ok {
		return item.MaxStack
	}
	if _, isTool := GetToolProperties(itemID); isTool {
		return 1
	}
	// Armor (Leather 298-301, Chainmail 302-305, Iron 306-309, Diamond 310-313, Gold 314-317)
	if (itemID >= 298 && itemID <= 317) || (itemID >= ItemCopperHelmet && itemID <= ItemCopperBoots) || (itemID >= ItemNetheriteHelmet && itemID <= ItemNetheriteBoots) {
		return 1
	}
	// Non-stackable tools, weapons, and accessories
	switch itemID {
	case 259, 261, 328, 333, 342, 343, 346, 359: // Flint and Steel, Bow, minecarts, boat, fishing rod, shears
		return 1
	case 325: // Empty bucket
		return 16
	case 326, 327: // Filled buckets are not stackable.
		return 1
	case 332, 344, 368: // Snowball, Egg, Ender Pearl
		return 16
	default:
		return 64
	}
}

// CreativeItemStack returns the stack created by a click on a creative catalog
// entry: left and middle click give a full stack, while right click gives one.
func CreativeItemStack(itemID int32, button int) ItemStack {
	if itemID <= 0 || button < 1 || button > 3 {
		return ItemStack{}
	}
	count := GetMaxStackSize(itemID)
	if button == 2 {
		count = 1
	}
	return ItemStack{ItemID: itemID, Count: count}
}

// AddItem tries to stack or insert items into first available slot.
func (inv *PlayerInventory) AddItem(itemID int32, count int) int {
	if inv == nil || itemID == ItemAir || count <= 0 {
		return count
	}

	remaining := count
	maxStack := GetMaxStackSize(itemID)

	// First pass: stack onto existing matching items only if item is stackable (maxStack > 1)
	if maxStack > 1 {
		for i := 0; i < len(inv.Slots); i++ {
			slot := &inv.Slots[i]
			if slot.ItemID == itemID && slot.Damage == 0 && slot.Count < maxStack {
				space := maxStack - slot.Count
				if remaining <= space {
					slot.Count += remaining
					return 0
				}
				slot.Count = maxStack
				remaining -= space
			}
		}
	}

	// Second pass: fill empty slots respecting maxStack
	for i := 0; i < len(inv.Slots); i++ {
		slot := &inv.Slots[i]
		if slot.IsEmpty() {
			if remaining <= maxStack {
				*slot = ItemStack{ItemID: itemID, Count: remaining}
				return 0
			}
			*slot = ItemStack{ItemID: itemID, Count: maxStack}
			remaining -= maxStack
		}
	}

	return remaining // returned overflow if inventory was full
}

// RemoveItem decrements count from a specific slot.
func (inv *PlayerInventory) RemoveItem(slotIdx int, count int) error {
	if inv == nil {
		return errors.New("inventory is nil")
	}
	if count <= 0 {
		return errors.New("item count must be positive")
	}

	if slotIdx < 0 || slotIdx >= len(inv.Slots) {
		return errors.New("slot index out of range")
	}
	slot := &inv.Slots[slotIdx]
	if slot.IsEmpty() || slot.Count < count {
		return errors.New("not enough items in slot")
	}
	slot.Count -= count
	if slot.Count == 0 {
		*slot = ItemStack{}
	}
	return nil
}

// HandleSlotLeftClick handles authentic Minecraft left-click item interaction on a slot.
func HandleSlotLeftClick(cursor *ItemStack, slot *ItemStack) {
	if cursor.IsEmpty() {
		if !slot.IsEmpty() {
			*cursor = *slot
			*slot = ItemStack{}
		}
		return
	}
	if slot.IsEmpty() {
		*slot = *cursor
		*cursor = ItemStack{}
		return
	}
	maxStack := GetMaxStackSize(slot.ItemID)
	if cursor.ItemID == slot.ItemID && cursor.Damage == slot.Damage {
		space := maxStack - slot.Count
		if space > 0 {
			toAdd := cursor.Count
			if toAdd > space {
				toAdd = space
			}
			slot.Count += toAdd
			cursor.Count -= toAdd
			if cursor.Count <= 0 {
				*cursor = ItemStack{}
			}
		}
	} else {
		tmp := *slot
		*slot = *cursor
		*cursor = tmp
	}
}

// HandleSlotRightClick handles authentic Minecraft right-click item interaction on a slot.
func HandleSlotRightClick(cursor *ItemStack, slot *ItemStack) {
	if cursor.IsEmpty() {
		if !slot.IsEmpty() {
			half := (slot.Count + 1) / 2
			*cursor = ItemStack{
				ItemID: slot.ItemID,
				Count:  half,
				Damage: slot.Damage,
			}
			slot.Count -= half
			if slot.Count <= 0 {
				*slot = ItemStack{}
			}
		}
		return
	}
	maxStack := GetMaxStackSize(cursor.ItemID)
	if slot.IsEmpty() {
		*slot = ItemStack{
			ItemID: cursor.ItemID,
			Count:  1,
			Damage: cursor.Damage,
		}
		cursor.Count--
		if cursor.Count <= 0 {
			*cursor = ItemStack{}
		}
		return
	}
	if slot.ItemID == cursor.ItemID && slot.Damage == cursor.Damage && slot.Count < maxStack {
		slot.Count++
		cursor.Count--
		if cursor.Count <= 0 {
			*cursor = ItemStack{}
		}
	}
}

// HandleArmorSlotClick applies creative inventory mouse clicks to one armor slot.
// Armor types are restricted to their vanilla equipment position.
func HandleArmorSlotClick(cursor, slot *ItemStack, armorSlot, button int) {
	if cursor == nil || slot == nil || armorSlot < 0 || armorSlot >= 4 {
		return
	}
	if button == 3 {
		if !slot.IsEmpty() {
			*cursor = CreativeItemStack(slot.ItemID, 1)
		}
		return
	}
	if button == 2 {
		if cursor.IsEmpty() {
			HandleSlotRightClick(cursor, slot)
		} else if slot.IsEmpty() && CanEquipArmor(armorSlot, cursor.ItemID) {
			HandleSlotRightClick(cursor, slot)
		}
		return
	}
	if button != 1 {
		return
	}
	if cursor.IsEmpty() || CanEquipArmor(armorSlot, cursor.ItemID) {
		HandleSlotLeftClick(cursor, slot)
	}
}
