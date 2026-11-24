package inventory

// FoodItemDefinition is the local registry entry for a Java 26.3 food or drink
// that is represented in the creative Food & Drinks tab. RuntimeID is an
// internal minecraft-go value, not a Mojang protocol or save ID.
type FoodItemDefinition struct {
	RuntimeID       int32
	Name            string
	DisplayName     string
	Texture         string
	Hunger          int
	Saturation      float32
	MaxStack        int
	RemainderItemID int32
	AlwaysEdible    bool
}

const (
	ItemEnchantedGoldenApple int32 = 15000 + iota
	ItemSweetBerries
	ItemGlowBerries
	ItemChorusFruit
	ItemCarrot
	ItemGoldenCarrot
	ItemPotato
	ItemBakedPotato
	ItemPoisonousPotato
	ItemBeetroot
	ItemGoldenDandelion
	ItemDriedKelp
	ItemRabbit
	ItemCookedRabbit
	ItemCod
	ItemCookedCod
	ItemSalmon
	ItemCookedSalmon
	ItemTropicalFish
	ItemPufferfish
	ItemCookie
	ItemCake
	ItemPumpkinPie
	ItemMushroomStew
	ItemBeetrootSoup
	ItemRabbitStew
	ItemMilkBucket
	ItemHoneyBottle
)

// java263FoodItems follows the official Java 26.3 food-and-drinks tab order.
// Cake and golden dandelion are listed there, but are not consumed as ordinary
// food; their placement/animal-growth interactions are not implemented yet.
var java263FoodItems = [...]FoodItemDefinition{
	// Vanilla reuses the golden_apple sprite and applies the enchanted glint as
	// an item render layer; minecraft-go does not render that component layer yet.
	{ItemEnchantedGoldenApple, "minecraft:enchanted_golden_apple", "附魔金苹果", "golden_apple", 4, 9.6, 64, 0, true},
	{ItemSweetBerries, "minecraft:sweet_berries", "甜浆果", "sweet_berries", 2, 0.4, 64, 0, false},
	{ItemGlowBerries, "minecraft:glow_berries", "发光浆果", "glow_berries", 2, 0.4, 64, 0, false},
	{ItemChorusFruit, "minecraft:chorus_fruit", "紫颂果", "chorus_fruit", 4, 2.4, 64, 0, true},
	{ItemCarrot, "minecraft:carrot", "胡萝卜", "carrot", 3, 3.6, 64, 0, false},
	{ItemGoldenCarrot, "minecraft:golden_carrot", "金胡萝卜", "golden_carrot", 6, 14.4, 64, 0, false},
	{ItemPotato, "minecraft:potato", "马铃薯", "potato", 1, 0.6, 64, 0, false},
	{ItemBakedPotato, "minecraft:baked_potato", "烤马铃薯", "baked_potato", 5, 6.0, 64, 0, false},
	{ItemPoisonousPotato, "minecraft:poisonous_potato", "毒马铃薯", "poisonous_potato", 2, 1.2, 64, 0, false},
	{ItemBeetroot, "minecraft:beetroot", "甜菜根", "beetroot", 1, 1.2, 64, 0, false},
	{ItemGoldenDandelion, "minecraft:golden_dandelion", "金蒲公英", "golden_dandelion", 0, 0, 64, 0, false},
	{ItemDriedKelp, "minecraft:dried_kelp", "干海带", "dried_kelp", 1, 0.6, 64, 0, false},
	{ItemRabbit, "minecraft:rabbit", "生兔肉", "rabbit", 3, 1.8, 64, 0, false},
	{ItemCookedRabbit, "minecraft:cooked_rabbit", "熟兔肉", "cooked_rabbit", 5, 6.0, 64, 0, false},
	{ItemCod, "minecraft:cod", "生鳕鱼", "cod", 2, 0.4, 64, 0, false},
	{ItemCookedCod, "minecraft:cooked_cod", "熟鳕鱼", "cooked_cod", 5, 6.0, 64, 0, false},
	{ItemSalmon, "minecraft:salmon", "生鲑鱼", "salmon", 2, 0.4, 64, 0, false},
	{ItemCookedSalmon, "minecraft:cooked_salmon", "熟鲑鱼", "cooked_salmon", 6, 9.6, 64, 0, false},
	{ItemTropicalFish, "minecraft:tropical_fish", "热带鱼", "tropical_fish", 1, 0.2, 64, 0, false},
	{ItemPufferfish, "minecraft:pufferfish", "河豚", "pufferfish", 1, 0.2, 64, 0, false},
	{ItemCookie, "minecraft:cookie", "曲奇", "cookie", 2, 0.4, 64, 0, false},
	{ItemCake, "minecraft:cake", "蛋糕", "cake", 0, 0, 1, 0, false},
	{ItemPumpkinPie, "minecraft:pumpkin_pie", "南瓜派", "pumpkin_pie", 8, 4.8, 64, 0, false},
	{ItemMushroomStew, "minecraft:mushroom_stew", "蘑菇煲", "mushroom_stew", 6, 7.2, 1, 281, false},
	{ItemBeetrootSoup, "minecraft:beetroot_soup", "甜菜汤", "beetroot_soup", 6, 7.2, 1, 281, false},
	{ItemRabbitStew, "minecraft:rabbit_stew", "兔肉煲", "rabbit_stew", 10, 12.0, 1, 281, false},
	{ItemMilkBucket, "minecraft:milk_bucket", "奶桶", "milk_bucket", 0, 0, 1, 325, true},
	{ItemHoneyBottle, "minecraft:honey_bottle", "蜂蜜瓶", "honey_bottle", 6, 1.2, 16, 374, true},
}

// FoodItemByID returns the added Java 26.3 food entry for a runtime ID.
func FoodItemByID(itemID int32) (FoodItemDefinition, bool) {
	for _, item := range java263FoodItems {
		if item.RuntimeID == itemID {
			return item, true
		}
	}
	return FoodItemDefinition{}, false
}
