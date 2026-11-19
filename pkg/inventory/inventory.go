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
	ItemAir           int32 = 0
	ItemStone         int32 = 1
	ItemGrass         int32 = 2
	ItemDirt          int32 = 3
	ItemCobblestone   int32 = 4
	ItemOakPlanks     int32 = 5
	ItemOakLog        int32 = 17
	ItemCraftingTable int32 = 58
	ItemFurnace       int32 = 61
	ItemTorch         int32 = 50
	ItemStick         int32 = 280
	ItemWoodenPickaxe int32 = 270
	ItemStonePickaxe  int32 = 274
	ItemIronPickaxe   int32 = 257
	ItemDiamondPickaxe int32 = 278
	ItemIronIngot     int32 = 265
	ItemDiamond       int32 = 264
	ItemCoal          int32 = 263
	ItemApple         int32 = 260
	ItemBread         int32 = 297
	ItemRawPorkchop   int32 = 319
	ItemCookedPorkchop int32 = 320
	ItemRawBeef       int32 = 363
	ItemSteak         int32 = 364
	ItemRottenFlesh   int32 = 367
)

// IsPlaceableBlock reports whether an item ID corresponds to a placeable voxel block.
func IsPlaceableBlock(itemID int32) bool {
	switch itemID {
	case ItemStone, ItemGrass, ItemDirt, ItemCobblestone, ItemOakPlanks,
		7, 8, 9, 10, 11, 12, 13, 14, 15, 16, ItemOakLog, 18, 20, 49,
		ItemTorch, 56, ItemCraftingTable, ItemFurnace, 87, 89, 121, 150:
		return true
	default:
		return false
	}
}

// FoodProperties defines nutrition restored by edible items.
type FoodProperties struct {
	HungerRestored int
	Saturation     float32
}

// FoodRegistry defines vanilla Minecraft Java food nutrition and saturation parameters.
var FoodRegistry = map[int32]FoodProperties{
	ItemApple:          {HungerRestored: 4, Saturation: 2.4},
	ItemBread:          {HungerRestored: 5, Saturation: 6.0},
	ItemRawPorkchop:    {HungerRestored: 3, Saturation: 1.8},
	ItemCookedPorkchop: {HungerRestored: 8, Saturation: 12.8},
	ItemRawBeef:        {HungerRestored: 3, Saturation: 1.8},
	ItemSteak:          {HungerRestored: 8, Saturation: 12.8},
	ItemRottenFlesh:    {HungerRestored: 4, Saturation: 0.8},
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
	Slots       [36]ItemStack // 0..8 Hotbar, 9..35 Main Inventory
	SelectedHotbar int       // 0..8
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

// AddItem tries to stack or insert items into first available slot.
func (inv *PlayerInventory) AddItem(itemID int32, count int) int {
	if itemID == ItemAir || count <= 0 {
		return count
	}

	remaining := count
	maxStack := 64

	// First pass: stack onto existing matching items
	for i := 0; i < len(inv.Slots); i++ {
		slot := &inv.Slots[i]
		if slot.ItemID == itemID && slot.Count < maxStack {
			space := maxStack - slot.Count
			if remaining <= space {
				slot.Count += remaining
				return 0
			}
			slot.Count = maxStack
			remaining -= space
		}
	}

	// Second pass: fill empty slots
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
