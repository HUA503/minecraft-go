package inventory

// ModernCreativeItemDefinition describes a vanilla creative item added beyond
// the project's legacy numeric item table, with its registry name and sprite.
type ModernCreativeItemDefinition struct {
	RuntimeID int32
	Name      string
	Display   string
	Texture   string
	Category  CreativeCategory
	MaxStack  int
}

const (
	ItemAmethystShard      int32 = 16000
	ItemBlazePowder        int32 = 16001
	ItemBlazeRod           int32 = 16002
	ItemCopperNugget       int32 = 16003
	ItemModernBowl         int32 = 281 // Shared with the bowl returned by soups.
	ItemCharcoal           int32 = 16005
	ItemGhastTear          int32 = 16006
	ItemEchoShard          int32 = 16007
	ItemNautilusShell      int32 = 16008
	ItemDiscFragment5      int32 = 16009
	ItemDragonBreath       int32 = 16010
	ItemFermentedSpiderEye int32 = 16011
	ItemGlassBottle        int32 = 16012
	ItemHoneycomb          int32 = 16013
	ItemRabbitHide         int32 = 16014
)

var modernCreativeItemDefinitions = [...]ModernCreativeItemDefinition{
	{ItemAmethystShard, "minecraft:amethyst_shard", "紫水晶碎片", "amethyst_shard", CreativeMaterials, 64},
	{ItemBlazePowder, "minecraft:blaze_powder", "烈焰粉", "blaze_powder", CreativeMaterials, 64},
	{ItemBlazeRod, "minecraft:blaze_rod", "烈焰棒", "blaze_rod", CreativeMaterials, 64},
	{ItemCopperNugget, "minecraft:copper_nugget", "铜粒", "copper_nugget", CreativeMaterials, 64},
	{ItemModernBowl, "minecraft:bowl", "碗", "bowl", CreativeMaterials, 64},
	{ItemCharcoal, "minecraft:charcoal", "木炭", "charcoal", CreativeMaterials, 64},
	{ItemGhastTear, "minecraft:ghast_tear", "恶魂之泪", "ghast_tear", CreativeMaterials, 64},
	{ItemEchoShard, "minecraft:echo_shard", "回响碎片", "echo_shard", CreativeMaterials, 64},
	{ItemNautilusShell, "minecraft:nautilus_shell", "鹦鹉螺壳", "nautilus_shell", CreativeMaterials, 64},
	{ItemDiscFragment5, "minecraft:disc_fragment_5", "唱片残片", "disc_fragment_5", CreativeMaterials, 64},
	{ItemDragonBreath, "minecraft:dragon_breath", "龙息", "dragon_breath", CreativeMaterials, 64},
	{ItemFermentedSpiderEye, "minecraft:fermented_spider_eye", "发酵蜘蛛眼", "fermented_spider_eye", CreativeMaterials, 64},
	{ItemGlassBottle, "minecraft:glass_bottle", "玻璃瓶", "glass_bottle", CreativeMaterials, 64},
	{ItemHoneycomb, "minecraft:honeycomb", "蜂脾", "honeycomb", CreativeMaterials, 64},
	{ItemRabbitHide, "minecraft:rabbit_hide", "兔子皮", "rabbit_hide", CreativeMaterials, 64},
}

var modernCreativeItemByID = func() map[int32]ModernCreativeItemDefinition {
	items := make(map[int32]ModernCreativeItemDefinition, len(modernCreativeItemDefinitions))
	for _, item := range modernCreativeItemDefinitions {
		items[item.RuntimeID] = item
	}
	return items
}()

var modernCreativeItemByName = func() map[string]ModernCreativeItemDefinition {
	items := make(map[string]ModernCreativeItemDefinition, len(modernCreativeItemDefinitions))
	for _, item := range modernCreativeItemDefinitions {
		items[item.Name] = item
	}
	return items
}()

func ModernCreativeItemByID(id int32) (ModernCreativeItemDefinition, bool) {
	item, ok := modernCreativeItemByID[id]
	return item, ok
}

func ModernCreativeItemByName(name string) (ModernCreativeItemDefinition, bool) {
	item, ok := modernCreativeItemByName[name]
	return item, ok
}

func ModernCreativeItemDefinitions() []ModernCreativeItemDefinition {
	return append([]ModernCreativeItemDefinition(nil), modernCreativeItemDefinitions[:]...)
}
