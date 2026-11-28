package inventory

import "testing"

func TestAddItemStackPreservesDamageAndDoesNotMergeDifferentWear(t *testing.T) {
	inv := NewPlayerInventory()
	if remaining := inv.AddItemStack(ItemStack{ItemID: ItemTrident, Count: 1, Damage: 13}); remaining != 0 {
		t.Fatalf("adding a worn trident left %d items", remaining)
	}
	if got := inv.Slots[0]; got != (ItemStack{ItemID: ItemTrident, Count: 1, Damage: 13}) {
		t.Fatalf("stored trident = %+v, want damage 13", got)
	}
	if remaining := inv.AddItemStack(ItemStack{ItemID: ItemTrident, Count: 1, Damage: 13}); remaining != 0 {
		t.Fatalf("adding a second non-stackable trident left %d items", remaining)
	}
	if got := inv.Slots[1]; got != (ItemStack{ItemID: ItemTrident, Count: 1, Damage: 13}) {
		t.Fatalf("second trident = %+v, want a separate damaged stack", got)
	}
	if remaining := inv.AddItemStack(ItemStack{ItemID: ItemTrident, Count: 1, Damage: 14}); remaining != 0 {
		t.Fatalf("adding a differently worn trident left %d items", remaining)
	}
	if got := inv.Slots[2]; got != (ItemStack{ItemID: ItemTrident, Count: 1, Damage: 14}) {
		t.Fatalf("third trident = %+v, want damage 14", got)
	}
}
