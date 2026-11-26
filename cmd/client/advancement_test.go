package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
)

func TestStoryAdvancementsFollowObtainedItems(t *testing.T) {
	client := &Client{advancements: make(map[string]bool)}
	client.recordObtainedItem(1) // Breaking stone is not the inventory criterion.
	if client.advancements["minecraft:story/mine_stone"] {
		t.Fatal("mine_stone unlocked by a block ID instead of obtaining cobblestone")
	}
	client.recordObtainedItem(inventory.ItemCobblestone)
	if !client.advancements["minecraft:story/mine_stone"] {
		t.Fatal("mine_stone did not unlock after obtaining cobblestone")
	}

	client.recordObtainedItem(inventory.ItemStoneAxe)
	if client.advancements["minecraft:story/upgrade_tools"] {
		t.Fatal("upgrade_tools unlocked for a stone axe instead of a stone pickaxe")
	}
	client.recordObtainedItem(inventory.ItemStonePickaxe)
	if !client.advancements["minecraft:story/upgrade_tools"] {
		t.Fatal("upgrade_tools did not unlock after obtaining a stone pickaxe")
	}
	client.recordObtainedItem(inventory.ItemIronPickaxe)
	if !client.advancements["minecraft:story/iron_tools"] {
		t.Fatal("iron_tools did not unlock after obtaining an iron pickaxe")
	}

	if !isLogItem(17) || !isLogItem(22) {
		t.Fatal("legacy oak/spruce log items were not recognized")
	}
	if poplar, ok := inventory.ExtendedBlockByName("minecraft:poplar_log"); !ok || !isLogItem(poplar.RuntimeID) {
		t.Fatal("Java 26.3 poplar log item was not recognized")
	}
	if isLogItem(inventory.ItemOakPlanks) {
		t.Fatal("planks incorrectly satisfy the logs advancement criterion")
	}
}

func TestPickupAdvancementUsesLocalPlayerItemEvent(t *testing.T) {
	engine := sim.NewEngine()
	client := &Client{
		isLocal:      true,
		engine:       engine,
		player:       &sim.PlayerEntity{ID: 7},
		advancements: make(map[string]bool),
	}
	engine.Events <- sim.WorldEvent{Type: "item_pickup", PlayerID: 7, BlockID: inventory.ItemCobblestone}
	engine.Events <- sim.WorldEvent{Type: "item_pickup", PlayerID: 8, BlockID: inventory.ItemDiamond}
	client.processLocalWorldEvents()
	if !client.advancements["minecraft:story/mine_stone"] {
		t.Fatal("local player's cobblestone pickup did not unlock mine_stone")
	}
	if client.advancements["minecraft:story/mine_diamond"] {
		t.Fatal("another player's diamond pickup unlocked a local advancement")
	}
}
