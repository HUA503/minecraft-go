package inventory

// CreativeCategory groups the catalog shown by the creative inventory.
type CreativeCategory int

const (
	CreativeAll CreativeCategory = iota
	CreativeBlocks
	CreativeTools
	CreativeMaterials
	CreativeFood
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

var creativeCatalog = []CreativeItem{
	{1, "石头", CreativeBlocks}, {2, "草方块", CreativeBlocks}, {3, "泥土", CreativeBlocks}, {4, "圆石", CreativeBlocks},
	{5, "橡木木板", CreativeBlocks}, {7, "基岩", CreativeBlocks}, {8, "水", CreativeBlocks}, {10, "熔岩", CreativeBlocks},
	{12, "沙子", CreativeBlocks}, {13, "沙砾", CreativeBlocks}, {14, "金矿石", CreativeBlocks}, {15, "铁矿石", CreativeBlocks},
	{16, "煤矿石", CreativeBlocks}, {17, "橡木原木", CreativeBlocks}, {18, "橡木树叶", CreativeBlocks}, {20, "玻璃", CreativeBlocks},
	{21, "白桦树叶", CreativeBlocks}, {22, "云杉原木", CreativeBlocks}, {23, "云杉树叶", CreativeBlocks}, {31, "高草", CreativeBlocks},
	{32, "枯萎灌木", CreativeBlocks}, {33, "蕨", CreativeBlocks}, {37, "蒲公英", CreativeBlocks}, {38, "虞美人", CreativeBlocks},
	{41, "金块", CreativeBlocks}, {42, "铁块", CreativeBlocks}, {45, "砖块", CreativeBlocks}, {46, "TNT", CreativeBlocks},
	{47, "书架", CreativeBlocks}, {48, "苔石", CreativeBlocks}, {49, "黑曜石", CreativeBlocks}, {50, "火把", CreativeBlocks},
	{54, "箱子", CreativeBlocks}, {56, "钻石矿石", CreativeBlocks}, {57, "钻石块", CreativeBlocks}, {58, "工作台", CreativeBlocks},
	{61, "熔炉", CreativeBlocks}, {65, "梯子", CreativeBlocks}, {73, "红石矿石", CreativeBlocks}, {74, "青金石矿石", CreativeBlocks},
	{81, "仙人掌", CreativeBlocks}, {83, "甘蔗", CreativeBlocks}, {86, "南瓜", CreativeBlocks}, {87, "下界岩", CreativeBlocks},
	{88, "灵魂沙", CreativeBlocks}, {89, "荧石", CreativeBlocks}, {90, "下界传送门", CreativeBlocks}, {103, "西瓜", CreativeBlocks},
	{111, "睡莲", CreativeBlocks}, {119, "末地传送门", CreativeBlocks}, {120, "末地传送门框架", CreativeBlocks}, {121, "末地石", CreativeBlocks},
	{122, "龙蛋", CreativeBlocks}, {126, "橡木台阶", CreativeBlocks}, {150, "深板岩", CreativeBlocks},

	{268, "木剑", CreativeTools}, {272, "石剑", CreativeTools}, {267, "铁剑", CreativeTools}, {276, "钻石剑", CreativeTools},
	{283, "金剑", CreativeTools}, {270, "木镐", CreativeTools}, {274, "石镐", CreativeTools}, {257, "铁镐", CreativeTools},
	{278, "钻石镐", CreativeTools}, {285, "金镐", CreativeTools}, {271, "木斧", CreativeTools}, {275, "石斧", CreativeTools},
	{258, "铁斧", CreativeTools}, {279, "钻石斧", CreativeTools}, {286, "金斧", CreativeTools}, {269, "木锹", CreativeTools},
	{273, "石锹", CreativeTools}, {256, "铁锹", CreativeTools}, {277, "钻石锹", CreativeTools}, {284, "金锹", CreativeTools},
	{290, "木锄", CreativeTools}, {291, "石锄", CreativeTools}, {292, "铁锄", CreativeTools}, {293, "钻石锄", CreativeTools},
	{294, "金锄", CreativeTools}, {261, "弓", CreativeTools}, {259, "打火石", CreativeTools}, {359, "剪刀", CreativeTools},
	{298, "皮革帽子", CreativeTools}, {299, "皮革外套", CreativeTools}, {300, "皮革裤子", CreativeTools}, {301, "皮革靴子", CreativeTools},
	{306, "铁头盔", CreativeTools}, {307, "铁胸甲", CreativeTools}, {308, "铁护腿", CreativeTools}, {309, "铁靴子", CreativeTools},
	{310, "钻石头盔", CreativeTools}, {311, "钻石胸甲", CreativeTools}, {312, "钻石护腿", CreativeTools}, {313, "钻石靴子", CreativeTools},

	{263, "煤炭", CreativeMaterials}, {264, "钻石", CreativeMaterials}, {265, "铁锭", CreativeMaterials}, {266, "金锭", CreativeMaterials},
	{280, "木棍", CreativeMaterials}, {331, "红石粉", CreativeMaterials}, {352, "骨头", CreativeMaterials}, {288, "羽毛", CreativeMaterials},
	{289, "火药", CreativeMaterials}, {287, "线", CreativeMaterials}, {334, "皮革", CreativeMaterials}, {318, "燧石", CreativeMaterials},
	{322, "金苹果", CreativeMaterials}, {325, "铁桶", CreativeMaterials}, {326, "水桶", CreativeMaterials}, {327, "熔岩桶", CreativeMaterials},
	{332, "雪球", CreativeMaterials}, {344, "鸡蛋", CreativeMaterials}, {368, "末影珍珠", CreativeMaterials}, {324, "橡木门", CreativeMaterials},
	{330, "铁门", CreativeMaterials}, {336, "砖", CreativeMaterials}, {337, "黏土球", CreativeMaterials}, {340, "书", CreativeMaterials},

	{260, "苹果", CreativeFood}, {297, "面包", CreativeFood}, {319, "生猪排", CreativeFood}, {320, "熟猪排", CreativeFood},
	{363, "生牛肉", CreativeFood}, {364, "牛排", CreativeFood}, {367, "腐肉", CreativeFood}, {365, "生鸡肉", CreativeFood},
	{366, "熟鸡肉", CreativeFood}, {423, "生羊肉", CreativeFood}, {424, "熟羊肉", CreativeFood}, {360, "西瓜片", CreativeFood},
}

