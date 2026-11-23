package inventory

import "strings"

// CreativeCategory groups the catalog shown by the creative inventory.
type CreativeCategory int

const (
	CreativeAll CreativeCategory = iota
	CreativeBlocks
	CreativeDecoration
	CreativeRedstone
	CreativeTransportation
	CreativeMisc
	CreativeFood
	CreativeTools
	CreativeCombat
	CreativeMaterials
)

// CreativeItem describes an item offered by the local creative inventory.
type CreativeItem struct {
	ID       int32
	Name     string
	Category CreativeCategory
}

// CreativeCatalog returns the currently implemented creative items in a
// stable, vanilla-like order. The catalog is intentionally limited to items
// with a corresponding local block, item behavior, or sprite implementation.
func CreativeCatalog(category CreativeCategory) []CreativeItem {
	items := creativeCatalog
	if category == CreativeAll {
		return append([]CreativeItem(nil), items...)
	}
	filtered := make([]CreativeItem, 0, len(items))
	for _, item := range items {
		if item.Category == category {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// SearchCreativeCatalog filters the selected category by its localized item name.
func SearchCreativeCatalog(category CreativeCategory, query string) []CreativeItem {
	items := CreativeCatalog(category)
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return items
	}
	filtered := make([]CreativeItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

var creativeCatalog = []CreativeItem{
	{1, "石头", CreativeBlocks}, {2, "草方块", CreativeBlocks}, {3, "泥土", CreativeBlocks}, {4, "圆石", CreativeBlocks},
	{5, "橡木木板", CreativeBlocks}, {6, "橡木树苗", CreativeDecoration}, {7, "基岩", CreativeBlocks}, {8, "水", CreativeBlocks}, {9, "静止的水", CreativeBlocks}, {10, "熔岩", CreativeBlocks}, {11, "静止的熔岩", CreativeBlocks},
	{12, "沙子", CreativeBlocks}, {13, "沙砾", CreativeBlocks}, {14, "金矿石", CreativeBlocks}, {15, "铁矿石", CreativeBlocks},
	{16, "煤矿石", CreativeBlocks}, {17, "橡木原木", CreativeBlocks}, {18, "橡木树叶", CreativeDecoration}, {19, "海绵", CreativeBlocks}, {20, "玻璃", CreativeDecoration},
	{21, "白桦树叶", CreativeDecoration}, {22, "云杉原木", CreativeBlocks}, {23, "云杉树叶", CreativeDecoration}, {24, "砂岩", CreativeBlocks}, {31, "高草", CreativeDecoration},
	{35, "白色羊毛", CreativeBlocks}, {43, "双石台阶", CreativeBlocks},
	{32, "枯萎灌木", CreativeDecoration}, {33, "蕨", CreativeDecoration}, {37, "蒲公英", CreativeDecoration}, {38, "虞美人", CreativeDecoration},
	{41, "金块", CreativeBlocks}, {42, "铁块", CreativeBlocks}, {44, "石台阶", CreativeBlocks}, {45, "砖块", CreativeBlocks}, {46, "TNT", CreativeRedstone},
	{47, "书架", CreativeBlocks}, {48, "苔石", CreativeBlocks}, {49, "黑曜石", CreativeBlocks}, {50, "火把", CreativeRedstone}, {52, "刷怪笼", CreativeBlocks},
	{54, "箱子", CreativeBlocks}, {56, "钻石矿石", CreativeBlocks}, {57, "钻石块", CreativeBlocks}, {58, "工作台", CreativeBlocks},
	{61, "熔炉", CreativeRedstone}, {65, "梯子", CreativeDecoration}, {73, "红石矿石", CreativeRedstone}, {74, "青金石矿石", CreativeBlocks},
	{81, "仙人掌", CreativeDecoration}, {83, "甘蔗", CreativeDecoration}, {86, "南瓜", CreativeDecoration}, {87, "下界岩", CreativeBlocks},
	{79, "冰", CreativeBlocks}, {80, "雪块", CreativeBlocks}, {82, "黏土块", CreativeBlocks}, {88, "灵魂沙", CreativeBlocks}, {89, "荧石", CreativeBlocks}, {90, "下界传送门", CreativeBlocks}, {98, "石砖", CreativeBlocks}, {103, "西瓜", CreativeDecoration},
	{111, "睡莲", CreativeDecoration}, {112, "下界砖块", CreativeBlocks}, {119, "末地传送门", CreativeBlocks}, {120, "末地传送门框架", CreativeBlocks}, {121, "末地石", CreativeBlocks},
	{122, "龙蛋", CreativeBlocks}, {126, "橡木台阶", CreativeBlocks}, {150, "深板岩", CreativeBlocks}, {153, "投掷器", CreativeBlocks}, {154, "漏斗", CreativeBlocks},
	{155, "石英块", CreativeBlocks}, {159, "白色陶瓦", CreativeBlocks}, {172, "陶瓦", CreativeBlocks}, {173, "煤炭块", CreativeBlocks}, {174, "浮冰", CreativeBlocks}, {179, "红砂岩", CreativeBlocks},

	{268, "木剑", CreativeCombat}, {272, "石剑", CreativeCombat}, {267, "铁剑", CreativeCombat}, {276, "钻石剑", CreativeCombat},
	{283, "金剑", CreativeCombat}, {270, "木镐", CreativeTools}, {274, "石镐", CreativeTools}, {257, "铁镐", CreativeTools},
	{278, "钻石镐", CreativeTools}, {285, "金镐", CreativeTools}, {271, "木斧", CreativeTools}, {275, "石斧", CreativeTools},
	{258, "铁斧", CreativeTools}, {279, "钻石斧", CreativeTools}, {286, "金斧", CreativeTools}, {269, "木锹", CreativeTools},
	{273, "石锹", CreativeTools}, {256, "铁锹", CreativeTools}, {277, "钻石锹", CreativeTools}, {284, "金锹", CreativeTools},
	{290, "木锄", CreativeTools}, {291, "石锄", CreativeTools}, {292, "铁锄", CreativeTools}, {293, "钻石锄", CreativeTools},
	{294, "金锄", CreativeTools}, {261, "弓", CreativeCombat}, {259, "打火石", CreativeTools}, {359, "剪刀", CreativeTools},
	{298, "皮革帽子", CreativeCombat}, {299, "皮革外套", CreativeCombat}, {300, "皮革裤子", CreativeCombat}, {301, "皮革靴子", CreativeCombat},
	{306, "铁头盔", CreativeCombat}, {307, "铁胸甲", CreativeCombat}, {308, "铁护腿", CreativeCombat}, {309, "铁靴子", CreativeCombat},
	{310, "钻石头盔", CreativeCombat}, {311, "钻石胸甲", CreativeCombat}, {312, "钻石护腿", CreativeCombat}, {313, "钻石靴子", CreativeCombat},
	{302, "锁链头盔", CreativeCombat}, {303, "锁链胸甲", CreativeCombat}, {304, "锁链护腿", CreativeCombat}, {305, "锁链靴子", CreativeCombat},
	{314, "金头盔", CreativeCombat}, {315, "金胸甲", CreativeCombat}, {316, "金护腿", CreativeCombat}, {317, "金靴子", CreativeCombat},

	{262, "箭", CreativeMaterials}, {263, "煤炭", CreativeMaterials}, {264, "钻石", CreativeMaterials}, {265, "铁锭", CreativeMaterials}, {266, "金锭", CreativeMaterials},
	{280, "木棍", CreativeMaterials}, {331, "红石粉", CreativeRedstone}, {352, "骨头", CreativeMaterials}, {288, "羽毛", CreativeMaterials},
	{289, "火药", CreativeMaterials}, {287, "线", CreativeMaterials}, {334, "皮革", CreativeMaterials}, {318, "燧石", CreativeMaterials}, {295, "小麦种子", CreativeMaterials}, {296, "小麦", CreativeMaterials},
	{322, "金苹果", CreativeFood}, {325, "铁桶", CreativeMisc}, {326, "水桶", CreativeMisc}, {327, "熔岩桶", CreativeMisc}, {346, "钓鱼竿", CreativeTools},
	{332, "雪球", CreativeMisc}, {344, "鸡蛋", CreativeMisc}, {368, "末影珍珠", CreativeMisc}, {324, "橡木门", CreativeDecoration},
	{330, "铁门", CreativeDecoration}, {336, "砖", CreativeMaterials}, {337, "黏土球", CreativeMaterials}, {340, "书", CreativeMisc}, {345, "指南针", CreativeMisc}, {371, "金粒", CreativeMaterials}, {375, "蜘蛛眼", CreativeMisc}, {381, "末影之眼", CreativeMisc}, {406, "下界石英", CreativeMaterials},
	{328, "矿车", CreativeTransportation}, {333, "橡木船", CreativeTransportation}, {342, "运输矿车", CreativeTransportation}, {343, "熔炉矿车", CreativeTransportation},

	{260, "苹果", CreativeFood}, {297, "面包", CreativeFood}, {319, "生猪排", CreativeFood}, {320, "熟猪排", CreativeFood},
	{363, "生牛肉", CreativeFood}, {364, "牛排", CreativeFood}, {367, "腐肉", CreativeFood}, {365, "生鸡肉", CreativeFood},
	{366, "熟鸡肉", CreativeFood}, {423, "生羊肉", CreativeFood}, {424, "熟羊肉", CreativeFood}, {360, "西瓜片", CreativeFood},
}

