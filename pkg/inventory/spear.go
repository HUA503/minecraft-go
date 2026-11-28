package inventory

// SpearProperties stores the material-specific combat values from the Java
// 26.3 item definitions. Durability and attack damage are also exposed through
// ToolRegistry so existing inventory wear and damage systems can use them.
type SpearProperties struct {
	JabCooldownTicks       int
	ChargeDamageMultiplier float32
	ForwardMovement        float32
	ChargeDelayTicks       int
	DismountWindowTicks    int
	DismountMinSpeed       float32
	KnockbackWindowTicks   int
	KnockbackMinSpeed      float32
	DamageWindowTicks      int
	DamageMinRelativeSpeed float32
	MinAttackReach         float64
	MaxAttackReach         float64
	CreativeMaxReach       float64
	HitboxMargin           float64
	MobAttackReachFactor   float64
}

var spearPropertiesByID = map[int32]SpearProperties{
	ItemWoodenSpear: {
		JabCooldownTicks: 13, ChargeDamageMultiplier: 0.38, ForwardMovement: 0.70,
		ChargeDelayTicks: 15, DismountWindowTicks: 100, DismountMinSpeed: 14,
		KnockbackWindowTicks: 200, KnockbackMinSpeed: 5.1, DamageWindowTicks: 300,
		DamageMinRelativeSpeed: 4.6,
	},
	ItemStoneSpear: {
		JabCooldownTicks: 15, ChargeDamageMultiplier: 0.38, ForwardMovement: 0.82,
		ChargeDelayTicks: 14, DismountWindowTicks: 90, DismountMinSpeed: 13,
		KnockbackWindowTicks: 180, KnockbackMinSpeed: 5.1, DamageWindowTicks: 275,
		DamageMinRelativeSpeed: 4.6,
	},
	ItemCopperSpear: {
		JabCooldownTicks: 17, ChargeDamageMultiplier: 0.38, ForwardMovement: 0.82,
		ChargeDelayTicks: 13, DismountWindowTicks: 80, DismountMinSpeed: 12,
		KnockbackWindowTicks: 165, KnockbackMinSpeed: 5.1, DamageWindowTicks: 250,
		DamageMinRelativeSpeed: 4.6,
	},
	ItemIronSpear: {
		JabCooldownTicks: 19, ChargeDamageMultiplier: 0.38, ForwardMovement: 0.95,
		ChargeDelayTicks: 12, DismountWindowTicks: 50, DismountMinSpeed: 11,
		KnockbackWindowTicks: 135, KnockbackMinSpeed: 5.1, DamageWindowTicks: 225,
		DamageMinRelativeSpeed: 4.6,
	},
	ItemGoldenSpear: {
		JabCooldownTicks: 19, ChargeDamageMultiplier: 0.38, ForwardMovement: 0.70,
		ChargeDelayTicks: 14, DismountWindowTicks: 70, DismountMinSpeed: 13,
		KnockbackWindowTicks: 170, KnockbackMinSpeed: 5.1, DamageWindowTicks: 275,
		DamageMinRelativeSpeed: 4.6,
	},
	ItemDiamondSpear: {
		JabCooldownTicks: 21, ChargeDamageMultiplier: 0.38, ForwardMovement: 1.075,
		ChargeDelayTicks: 13, DismountWindowTicks: 60, DismountMinSpeed: 10,
		KnockbackWindowTicks: 130, KnockbackMinSpeed: 5.1, DamageWindowTicks: 200,
		DamageMinRelativeSpeed: 4.6,
	},
	ItemNetheriteSpear: {
		JabCooldownTicks: 23, ChargeDamageMultiplier: 0.38, ForwardMovement: 1.20,
		ChargeDelayTicks: 12, DismountWindowTicks: 50, DismountMinSpeed: 9,
		KnockbackWindowTicks: 110, KnockbackMinSpeed: 5.1, DamageWindowTicks: 175,
		DamageMinRelativeSpeed: 4.6,
	},
}

func init() {
	for id, properties := range spearPropertiesByID {
		properties.MinAttackReach = 2.0
		properties.MaxAttackReach = 4.5
		properties.CreativeMaxReach = 6.5
		properties.HitboxMargin = 0.125
		properties.MobAttackReachFactor = 0.5
		spearPropertiesByID[id] = properties
	}
}

// GetSpearProperties returns the authentic material-specific spear settings.
func GetSpearProperties(itemID int32) (SpearProperties, bool) {
	properties, ok := spearPropertiesByID[itemID]
	return properties, ok
}
