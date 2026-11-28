package inventory

import "testing"

func TestSpearMaterialCombatProperties(t *testing.T) {
	tests := []struct {
		id               int32
		cooldown         int
		damage           float32
		durability       int
		forwardMovement  float32
		chargeDelay      int
		dismountWindow   int
		dismountMinSpeed float32
		knockbackWindow  int
		damageWindow     int
	}{
		{ItemWoodenSpear, 13, 1, 59, 0.7, 15, 100, 14, 200, 300},
		{ItemStoneSpear, 15, 2, 131, 0.82, 14, 90, 13, 180, 275},
		{ItemCopperSpear, 17, 2, 190, 0.82, 13, 80, 12, 165, 250},
		{ItemIronSpear, 19, 3, 250, 0.95, 12, 50, 11, 135, 225},
		{ItemGoldenSpear, 19, 1, 32, 0.7, 14, 70, 13, 170, 275},
		{ItemDiamondSpear, 21, 4, 1561, 1.075, 13, 60, 10, 130, 200},
		{ItemNetheriteSpear, 23, 5, 2031, 1.2, 12, 50, 9, 110, 175},
	}
	for _, test := range tests {
		properties, ok := GetSpearProperties(test.id)
		if !ok || properties.JabCooldownTicks != test.cooldown || properties.ForwardMovement != test.forwardMovement || properties.ChargeDamageMultiplier != 0.38 || properties.ChargeDelayTicks != test.chargeDelay || properties.DismountWindowTicks != test.dismountWindow || properties.DismountMinSpeed != test.dismountMinSpeed || properties.KnockbackWindowTicks != test.knockbackWindow || properties.KnockbackMinSpeed != 5.1 || properties.DamageWindowTicks != test.damageWindow || properties.DamageMinRelativeSpeed != 4.6 || properties.MinAttackReach != 2 || properties.MaxAttackReach != 4.5 || properties.CreativeMaxReach != 6.5 || properties.HitboxMargin != 0.125 || properties.MobAttackReachFactor != 0.5 {
			t.Errorf("spear %d combat properties = %+v, found=%t", test.id, properties, ok)
		}
		tool, ok := GetToolProperties(test.id)
		if !ok || tool.Type != ToolSpear || tool.AttackDamage != test.damage || tool.MaxDurability != test.durability {
			t.Errorf("spear %d tool properties = %+v, found=%t", test.id, tool, ok)
		}
	}
}
