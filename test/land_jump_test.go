package test

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/sim"
)

func TestPlayerJumpFromGroundDoesNotHitCeiling(t *testing.T) {
	// World with a stone floor at Y=64
	isSolid := func(x, y, z int32) bool {
		return y == 64
	}

	// Player standing firmly on the floor at Y=65.0
	pos := sim.Vec3{X: 8.5, Y: 65.0, Z: 8.5}

	// Jump delta Y = 0.42 (8.4 m/s * 0.05s)
	delta := sim.Vec3{X: 0, Y: 0.42, Z: 0}

	resolvedPos, onGround, hitCeiling := sim.ResolveMovementCollision(pos, delta, isSolid)

	if hitCeiling {
		t.Fatalf("player jumping upward from ground should NEVER trigger hitCeiling from the floor beneath! hitCeiling=%v", hitCeiling)
	}

	if onGround {
		t.Fatalf("player jumping upward should not be onGround! onGround=%v", onGround)
	}

	if resolvedPos.Y <= 65.0 {
		t.Fatalf("player did not move upward! resolvedPos.Y=%f, expected > 65.0", resolvedPos.Y)
	}

	if resolvedPos.Y != 65.42 {
		t.Fatalf("expected resolvedPos.Y=65.42, got %f", resolvedPos.Y)
	}
}

func TestGroundRestingPositionStabilityNoJitter(t *testing.T) {
	// World with a stone floor at Y=64
	isSolid := func(x, y, z int32) bool {
		return y == 64
	}

	// Player resting on ground
	pos := sim.Vec3{X: 8.5, Y: 65.0, Z: 8.5}

	// Gravity pulling down over 10 consecutive ticks while resting on ground
	for tick := 0; tick < 10; tick++ {
		delta := sim.Vec3{X: 0, Y: -0.05, Z: 0}
		resolvedPos, onGround, hitCeiling := sim.ResolveMovementCollision(pos, delta, isSolid)

		if !onGround {
			t.Fatalf("tick %d: player should remain onGround", tick)
		}
		if hitCeiling {
			t.Fatalf("tick %d: hitCeiling should be false on ground", tick)
		}
		if resolvedPos.Y != 65.0 {
			t.Fatalf("tick %d: resting Y position jittered! Expected exactly 65.0, got %f", tick, resolvedPos.Y)
		}
		pos = resolvedPos
	}
}
