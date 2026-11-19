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

func TestFullJumpArcAndBunnyHopCycle(t *testing.T) {
	eng := sim.NewEngine()
	// Flat floor at Y=64
	for x := int32(-2); x <= 2; x++ {
		for z := int32(-2); z <= 2; z++ {
			eng.SetBlock(x, 64, z, 1)
		}
	}

	p := eng.AddPlayer("Jumper", [16]byte{9}, sim.Vec3{X: 0.5, Y: 65.0, Z: 0.5})
	p.ClientDriven = true
	p.OnGround = true

	dt := 0.02 // 50 FPS
	jumpRequested := true

	// Step 1: Initial jump impulse
	if jumpRequested && p.OnGround {
		p.Velocity.Y = 8.4
		p.OnGround = false
	}

	var peakY float64 = 65.0
	var landed bool = false

	for step := 0; step < 100; step++ {
		if !p.OnGround {
			p.Velocity.Y -= 32.0 * dt
			if p.Velocity.Y < -78.4 {
				p.Velocity.Y = -78.4
			}
		}

		dy := p.Velocity.Y * dt
		delta := sim.Vec3{X: 0, Y: dy, Z: 0}
		resolvedPos, onGround, hitCeiling := eng.MovePlayerRelative(p.ID, delta, 0, 0)
		p.Position = resolvedPos
		p.OnGround = onGround

		if onGround {
			p.Velocity.Y = 0
			landed = true
			break
		} else if hitCeiling && p.Velocity.Y > 0 {
			p.Velocity.Y = 0
		}

		if resolvedPos.Y > peakY {
			peakY = resolvedPos.Y
		}
	}

	if !landed {
		t.Fatalf("player did not land back on ground after jump")
	}

	// Verify apex height reaches ~1.10 to 1.25 blocks above ground
	if peakY < 66.0 || peakY > 66.5 {
		t.Fatalf("jump apex height out of bounds: reached peak Y=%f (expected ~66.10-66.25)", peakY)
	}

	// Step 2: Bunny-hop on the very next frame while holding space
	if jumpRequested && p.OnGround {
		p.Velocity.Y = 8.4
		p.OnGround = false
	}
	if p.Velocity.Y != 8.4 || p.OnGround {
		t.Fatalf("bunny-hop failed to re-launch: velY=%f, onGround=%v", p.Velocity.Y, p.OnGround)
	}
}
