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
	ItemIronNugget         int32 = 16100
	ItemNetherBrick        int32 = 16117
	ItemResinClump         int32 = 16103
	ItemSlimeBall          int32 = 16108
	ItemResinBrick         int32 = 16118
	ItemPaper              int32 = 16119
	ItemNetherWart         int32 = 16121
	ItemGlowstoneDust      int32 = 16122
	ItemSugar              int32 = 16123
	ItemGlisteringMelon    int32 = 16125
	ItemMagmaCream         int32 = 16126
	ItemTurtleHelmet       int32 = 16127
	ItemBreezeRod          int32 = 16113
	ItemMace               int32 = 16216
	ItemHeavyCore          int32 = 16217
	ItemStrawBed           int32 = 16218
	ItemWhiteDye           int32 = 17000
	ItemOrangeDye          int32 = 17001
	ItemMagentaDye         int32 = 17002
	ItemLightBlueDye       int32 = 17003
	ItemYellowDye          int32 = 17004
	ItemLimeDye            int32 = 17005
	ItemPinkDye            int32 = 17006
	ItemGrayDye            int32 = 17007
	ItemLightGrayDye       int32 = 17008
	ItemCyanDye            int32 = 17009
	ItemPurpleDye          int32 = 17010
	ItemBlueDye            int32 = 17011
	ItemBrownDye           int32 = 17012
	ItemGreenDye           int32 = 17013
	ItemRedDye             int32 = 17014
	ItemBlackDye           int32 = 17015
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
	{ItemIronNugget, "minecraft:iron_nugget", "\u94c1\u7c92", "iron_nugget", CreativeMaterials, 64},
	{16101, "minecraft:brown_egg", "\u68d5\u8272\u9e21\u86cb", "brown_egg", CreativeMaterials, 16},
	{16102, "minecraft:blue_egg", "\u84dd\u8272\u9e21\u86cb", "blue_egg", CreativeMaterials, 16},
	{ItemResinClump, "minecraft:resin_clump", "\u6811\u8102\u56e2", "resin_clump", CreativeMaterials, 64},
	{16104, "minecraft:ink_sac", "\u58a8\u56ca", "ink_sac", CreativeMaterials, 64},
	{16105, "minecraft:glow_ink_sac", "\u8367\u5149\u58a8\u56ca", "glow_ink_sac", CreativeMaterials, 64},
	{16106, "minecraft:turtle_scute", "\u6d77\u9f9f\u9cde\u7532", "turtle_scute", CreativeMaterials, 64},
	{16107, "minecraft:armadillo_scute", "\u72b0\u72f3\u9cde\u7532", "armadillo_scute", CreativeMaterials, 64},
	{ItemSlimeBall, "minecraft:slime_ball", "\u9ecf\u6db2\u7403", "slime_ball", CreativeMaterials, 64},
	{16109, "minecraft:prismarine_shard", "\u6d77\u6676\u788e\u7247", "prismarine_shard", CreativeMaterials, 64},
	{16110, "minecraft:prismarine_crystals", "\u6d77\u6676\u7802\u7c92", "prismarine_crystals", CreativeMaterials, 64},
	{16111, "minecraft:heart_of_the_sea", "\u6d77\u6d0b\u4e4b\u5fc3", "heart_of_the_sea", CreativeMaterials, 64},
	{16112, "minecraft:fire_charge", "\u706b\u7130\u5f39", "fire_charge", CreativeMaterials, 64},
	{ItemBreezeRod, "minecraft:breeze_rod", "\u65cb\u98ce\u68d2", "breeze_rod", CreativeMaterials, 64},
	{16114, "minecraft:nether_star", "\u4e0b\u754c\u4e4b\u661f", "nether_star", CreativeMaterials, 64},
	{16115, "minecraft:shulker_shell", "\u6f5c\u5f71\u58f3", "shulker_shell", CreativeMaterials, 64},
	{16116, "minecraft:popped_chorus_fruit", "\u7206\u88c2\u7d2b\u9882\u679c", "popped_chorus_fruit", CreativeMaterials, 64},
	{ItemNetherBrick, "minecraft:nether_brick", "\u4e0b\u754c\u7816", "nether_brick", CreativeMaterials, 64},
	{ItemResinBrick, "minecraft:resin_brick", "\u6811\u8102\u7816", "resin_brick", CreativeMaterials, 64},
	{ItemPaper, "minecraft:paper", "\u7eb8", "paper", CreativeMaterials, 64},
	{16120, "minecraft:firework_star", "\u70df\u706b\u4e4b\u661f", "firework_star", CreativeMaterials, 64},
	{ItemNetherWart, "minecraft:nether_wart", "\u4e0b\u754c\u75a3", "nether_wart", CreativeMaterials, 64},
	{ItemGlowstoneDust, "minecraft:glowstone_dust", "\u8367\u77f3\u7c89", "glowstone_dust", CreativeMaterials, 64},
	{ItemSugar, "minecraft:sugar", "\u7cd6", "sugar", CreativeMaterials, 64},
	{16124, "minecraft:rabbit_foot", "\u5154\u5b50\u811a", "rabbit_foot", CreativeMaterials, 64},
	{ItemGlisteringMelon, "minecraft:glistering_melon_slice", "\u95ea\u70c1\u7684\u897f\u74dc\u7247", "glistering_melon_slice", CreativeMaterials, 64},
	{ItemMagmaCream, "minecraft:magma_cream", "\u5ca9\u6d46\u818f", "magma_cream", CreativeMaterials, 64},
	{ItemTurtleHelmet, "minecraft:turtle_helmet", "\u6d77\u9f9f\u58f3", "turtle_helmet", CreativeMaterials, 1},
	{16128, "minecraft:phantom_membrane", "\u5e7b\u7ffc\u819c", "phantom_membrane", CreativeMaterials, 64},
	{16129, "minecraft:field_masoned_banner_pattern", "\u7816\u7eb9\u65d7\u5e1c\u56fe\u6848", "field_masoned_banner_pattern", CreativeMaterials, 1},
	{16130, "minecraft:bordure_indented_banner_pattern", "\u6ce2\u7eb9\u8fb9\u65d7\u5e1c\u56fe\u6848", "bordure_indented_banner_pattern", CreativeMaterials, 1},
	{16131, "minecraft:flower_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "flower_banner_pattern", CreativeMaterials, 1},
	{16132, "minecraft:creeper_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "creeper_banner_pattern", CreativeMaterials, 1},
	{16133, "minecraft:skull_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "skull_banner_pattern", CreativeMaterials, 1},
	{16134, "minecraft:mojang_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "mojang_banner_pattern", CreativeMaterials, 1},
	{16135, "minecraft:globe_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "globe_banner_pattern", CreativeMaterials, 1},
	{16136, "minecraft:piglin_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "piglin_banner_pattern", CreativeMaterials, 1},
	{16137, "minecraft:flow_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "flow_banner_pattern", CreativeMaterials, 1},
	{16138, "minecraft:guster_banner_pattern", "\u65d7\u5e1c\u56fe\u6848", "guster_banner_pattern", CreativeMaterials, 1},
	{16139, "minecraft:angler_pottery_sherd", "\u5782\u9493\u7eb9\u6837\u9676\u7247", "angler_pottery_sherd", CreativeMaterials, 64},
	{16140, "minecraft:archer_pottery_sherd", "\u5f13\u7bad\u7eb9\u6837\u9676\u7247", "archer_pottery_sherd", CreativeMaterials, 64},
	{16141, "minecraft:arms_up_pottery_sherd", "\u4e3e\u81c2\u7eb9\u6837\u9676\u7247", "arms_up_pottery_sherd", CreativeMaterials, 64},
	{16142, "minecraft:blade_pottery_sherd", "\u5229\u5203\u7eb9\u6837\u9676\u7247", "blade_pottery_sherd", CreativeMaterials, 64},
	{16143, "minecraft:brewer_pottery_sherd", "\u4f73\u917f\u7eb9\u6837\u9676\u7247", "brewer_pottery_sherd", CreativeMaterials, 64},
	{16144, "minecraft:burn_pottery_sherd", "\u70c8\u7130\u7eb9\u6837\u9676\u7247", "burn_pottery_sherd", CreativeMaterials, 64},
	{16145, "minecraft:danger_pottery_sherd", "\u5371\u673a\u7eb9\u6837\u9676\u7247", "danger_pottery_sherd", CreativeMaterials, 64},
	{16146, "minecraft:explorer_pottery_sherd", "\u63a2\u9669\u7eb9\u6837\u9676\u7247", "explorer_pottery_sherd", CreativeMaterials, 64},
	{16147, "minecraft:flow_pottery_sherd", "\u6da1\u6d41\u7eb9\u6837\u9676\u7247", "flow_pottery_sherd", CreativeMaterials, 64},
	{16148, "minecraft:friend_pottery_sherd", "\u631a\u53cb\u7eb9\u6837\u9676\u7247", "friend_pottery_sherd", CreativeMaterials, 64},
	{16149, "minecraft:guster_pottery_sherd", "\u65cb\u98ce\u7eb9\u6837\u9676\u7247", "guster_pottery_sherd", CreativeMaterials, 64},
	{16150, "minecraft:heart_pottery_sherd", "\u7231\u5fc3\u7eb9\u6837\u9676\u7247", "heart_pottery_sherd", CreativeMaterials, 64},
	{16151, "minecraft:heartbreak_pottery_sherd", "\u5fc3\u788e\u7eb9\u6837\u9676\u7247", "heartbreak_pottery_sherd", CreativeMaterials, 64},
	{16152, "minecraft:howl_pottery_sherd", "\u72fc\u568e\u7eb9\u6837\u9676\u7247", "howl_pottery_sherd", CreativeMaterials, 64},
	{16153, "minecraft:miner_pottery_sherd", "\u91c7\u77ff\u7eb9\u6837\u9676\u7247", "miner_pottery_sherd", CreativeMaterials, 64},
	{16154, "minecraft:mourner_pottery_sherd", "\u60b2\u6078\u7eb9\u6837\u9676\u7247", "mourner_pottery_sherd", CreativeMaterials, 64},
	{16155, "minecraft:plenty_pottery_sherd", "\u5bcc\u9976\u7eb9\u6837\u9676\u7247", "plenty_pottery_sherd", CreativeMaterials, 64},
	{16156, "minecraft:prize_pottery_sherd", "\u73cd\u5b9d\u7eb9\u6837\u9676\u7247", "prize_pottery_sherd", CreativeMaterials, 64},
	{16157, "minecraft:scrape_pottery_sherd", "\u522e\u524a\u7eb9\u6837\u9676\u7247", "scrape_pottery_sherd", CreativeMaterials, 64},
	{16158, "minecraft:sheaf_pottery_sherd", "\u9ea6\u6346\u7eb9\u6837\u9676\u7247", "sheaf_pottery_sherd", CreativeMaterials, 64},
	{16159, "minecraft:shelter_pottery_sherd", "\u6811\u836b\u7eb9\u6837\u9676\u7247", "shelter_pottery_sherd", CreativeMaterials, 64},
	{16160, "minecraft:skull_pottery_sherd", "\u5934\u9885\u7eb9\u6837\u9676\u7247", "skull_pottery_sherd", CreativeMaterials, 64},
	{16161, "minecraft:snort_pottery_sherd", "\u55c5\u63a2\u7eb9\u6837\u9676\u7247", "snort_pottery_sherd", CreativeMaterials, 64},
	{16162, "minecraft:netherite_upgrade_smithing_template", "\u953b\u9020\u6a21\u677f", "netherite_upgrade_smithing_template", CreativeMaterials, 64},
	{16163, "minecraft:sentry_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "sentry_armor_trim_smithing_template", CreativeMaterials, 64},
	{16164, "minecraft:vex_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "vex_armor_trim_smithing_template", CreativeMaterials, 64},
	{16165, "minecraft:wild_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "wild_armor_trim_smithing_template", CreativeMaterials, 64},
	{16166, "minecraft:coast_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "coast_armor_trim_smithing_template", CreativeMaterials, 64},
	{16167, "minecraft:dune_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "dune_armor_trim_smithing_template", CreativeMaterials, 64},
	{16168, "minecraft:wayfinder_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "wayfinder_armor_trim_smithing_template", CreativeMaterials, 64},
	{16169, "minecraft:raiser_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "raiser_armor_trim_smithing_template", CreativeMaterials, 64},
	{16170, "minecraft:shaper_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "shaper_armor_trim_smithing_template", CreativeMaterials, 64},
	{16171, "minecraft:host_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "host_armor_trim_smithing_template", CreativeMaterials, 64},
	{16172, "minecraft:ward_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "ward_armor_trim_smithing_template", CreativeMaterials, 64},
	{16173, "minecraft:silence_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "silence_armor_trim_smithing_template", CreativeMaterials, 64},
	{16174, "minecraft:tide_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "tide_armor_trim_smithing_template", CreativeMaterials, 64},
	{16175, "minecraft:snout_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "snout_armor_trim_smithing_template", CreativeMaterials, 64},
	{16176, "minecraft:rib_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "rib_armor_trim_smithing_template", CreativeMaterials, 64},
	{16177, "minecraft:eye_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "eye_armor_trim_smithing_template", CreativeMaterials, 64},
	{16178, "minecraft:spire_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "spire_armor_trim_smithing_template", CreativeMaterials, 64},
	{16179, "minecraft:flow_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "flow_armor_trim_smithing_template", CreativeMaterials, 64},
	{16180, "minecraft:bolt_armor_trim_smithing_template", "\u953b\u9020\u6a21\u677f", "bolt_armor_trim_smithing_template", CreativeMaterials, 64},
	{16181, "minecraft:experience_bottle", "\u9644\u9b54\u4e4b\u74f6", "experience_bottle", CreativeMaterials, 64},
	{16182, "minecraft:trial_key", "\u8bd5\u70bc\u94a5\u5319", "trial_key", CreativeMaterials, 64},
	{16183, "minecraft:ominous_trial_key", "\u4e0d\u7965\u8bd5\u70bc\u94a5\u5319", "ominous_trial_key", CreativeMaterials, 64},
	{ItemHeavyCore, "minecraft:heavy_core", "\u91cd\u578b\u6838\u5fc3", "heavy_core", CreativeMaterials, 64},
	{ItemMace, "minecraft:mace", "\u91cd\u9524", "mace", CreativeCombat, 1},
	{ItemStrawBed, "minecraft:straw_bed", "\u7a3b\u8349\u5e8a", "straw_bed_particle", CreativeFunctional, 1},
	{ItemWhiteDye, "minecraft:white_dye", "\u767d\u8272\u67d3\u6599", "white_dye", CreativeMaterials, 64},
	{ItemOrangeDye, "minecraft:orange_dye", "\u6a59\u8272\u67d3\u6599", "orange_dye", CreativeMaterials, 64},
	{ItemMagentaDye, "minecraft:magenta_dye", "\u54c1\u7ea2\u8272\u67d3\u6599", "magenta_dye", CreativeMaterials, 64},
	{ItemLightBlueDye, "minecraft:light_blue_dye", "\u6de1\u84dd\u8272\u67d3\u6599", "light_blue_dye", CreativeMaterials, 64},
	{ItemYellowDye, "minecraft:yellow_dye", "\u9ec4\u8272\u67d3\u6599", "yellow_dye", CreativeMaterials, 64},
	{ItemLimeDye, "minecraft:lime_dye", "\u9ec5\u7eff\u8272\u67d3\u6599", "lime_dye", CreativeMaterials, 64},
	{ItemPinkDye, "minecraft:pink_dye", "\u7c89\u7ea2\u8272\u67d3\u6599", "pink_dye", CreativeMaterials, 64},
	{ItemGrayDye, "minecraft:gray_dye", "\u7070\u8272\u67d3\u6599", "gray_dye", CreativeMaterials, 64},
	{ItemLightGrayDye, "minecraft:light_gray_dye", "\u6de1\u7070\u8272\u67d3\u6599", "light_gray_dye", CreativeMaterials, 64},
	{ItemCyanDye, "minecraft:cyan_dye", "\u9752\u8272\u67d3\u6599", "cyan_dye", CreativeMaterials, 64},
	{ItemPurpleDye, "minecraft:purple_dye", "\u7d2b\u8272\u67d3\u6599", "purple_dye", CreativeMaterials, 64},
	{ItemBlueDye, "minecraft:blue_dye", "\u84dd\u8272\u67d3\u6599", "blue_dye", CreativeMaterials, 64},
	{ItemBrownDye, "minecraft:brown_dye", "\u68d5\u8272\u67d3\u6599", "brown_dye", CreativeMaterials, 64},
	{ItemGreenDye, "minecraft:green_dye", "\u7eff\u8272\u67d3\u6599", "green_dye", CreativeMaterials, 64},
	{ItemRedDye, "minecraft:red_dye", "\u7ea2\u8272\u67d3\u6599", "red_dye", CreativeMaterials, 64},
	{ItemBlackDye, "minecraft:black_dye", "\u9ed1\u8272\u67d3\u6599", "black_dye", CreativeMaterials, 64},
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
