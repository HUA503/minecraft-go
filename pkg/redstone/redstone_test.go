package redstone

import (
	"testing"
)

func TestWireDecay(t *testing.T) {
	eng := NewEngine()

	// Place power block at (0, 0, 0)
	eng.AddComponent(&Component{
		Type:       TypeBlock,
		Position:   Pos{X: 0, Y: 0, Z: 0},
		PowerLevel: 15,
	})

	// Place 16 wires in a line from X=1 to X=16
	for x := int32(1); x <= 16; x++ {
		eng.AddComponent(&Component{
			Type:     TypeWire,
			Position: Pos{X: x, Y: 0, Z: 0},
		})
	}

	eng.Step()

	// Wire at X=1 should have power 14
	if p := eng.GetComponent(Pos{X: 1, Y: 0, Z: 0}).PowerLevel; p != 14 {
		t.Fatalf("wire at X=1 power mismatch: got %d, expected 14", p)
	}
	// Wire at X=14 should have power 1
	if p := eng.GetComponent(Pos{X: 14, Y: 0, Z: 0}).PowerLevel; p != 1 {
		t.Fatalf("wire at X=14 power mismatch: got %d, expected 1", p)
	}
	// Wire at X=15 should have power 0 (exceeds 15 blocks from source)
	if p := eng.GetComponent(Pos{X: 15, Y: 0, Z: 0}).PowerLevel; p != 0 {
		t.Fatalf("wire at X=15 power mismatch: got %d, expected 0", p)
	}
}

func TestRepeaterDelay(t *testing.T) {
	eng := NewEngine()

	repPos := Pos{X: 2, Y: 0, Z: 0}
	eng.AddComponent(&Component{
		Type:       TypeRepeater,
		Position:   repPos,
		DelayTicks: 2,
	})

	eng.TriggerRepeater(repPos, 5)

	// Tick 1: Still 0
	eng.Step()
	if p := eng.GetComponent(repPos).PowerLevel; p != 0 {
		t.Fatalf("repeater should not be powered at tick 1: got %d", p)
	}

	// Tick 2: Powered to 15
	eng.Step()
	if p := eng.GetComponent(repPos).PowerLevel; p != 15 {
		t.Fatalf("repeater should be powered to 15 at tick 2: got %d", p)
	}
}

func TestObserverPulse(t *testing.T) {
	eng := NewEngine()

	obsPos := Pos{X: 5, Y: 0, Z: 5}
	eng.AddComponent(&Component{
		Type:     TypeObserver,
		Position: obsPos,
	})

	eng.TriggerObserver(obsPos)

	// Tick 1: Observer turns ON (pulse begins)
	eng.Step()
	if p := eng.GetComponent(obsPos).PowerLevel; p != 15 {
		t.Fatalf("observer should output power 15 at tick 1: got %d", p)
	}

	// Tick 2: Observer turns back OFF (1-tick pulse ends)
	eng.Step()
	if p := eng.GetComponent(obsPos).PowerLevel; p != 0 {
		t.Fatalf("observer pulse should end at tick 2: got %d", p)
	}
}

func TestPistonExtension(t *testing.T) {
	eng := NewEngine()

	pistonPos := Pos{X: 0, Y: 0, Z: 10}
	eng.AddComponent(&Component{
		Type:     TypeStickyPiston,
		Position: pistonPos,
	})

	// Power piston
	eng.TriggerPiston(pistonPos, true)
	eng.Step()

	piston := eng.GetComponent(pistonPos)
	if !piston.IsExtended {
		t.Fatalf("piston should be extended after powering")
	}

	// Unpower piston
	eng.TriggerPiston(pistonPos, false)
	eng.Step()

	if piston.IsExtended {
		t.Fatalf("piston should be retracted after unpowering")
	}
}

func TestComparatorModes(t *testing.T) {
	eng := NewEngine()

	compPos := Pos{X: 0, Y: 0, Z: 0}
	eng.AddComponent(&Component{
		Type:      TypeComparator,
		Position:  compPos,
		IsSubMode: false, // Compare mode
	})

	// In compare mode: Rear 10 >= Side 8 -> Output 10
	eng.TriggerComparator(compPos, 10, 8)
	eng.Step()
	if p := eng.GetComponent(compPos).PowerLevel; p != 10 {
		t.Fatalf("compare mode failed: got %d, expected 10", p)
	}

	// Switch to subtraction mode: Rear 10 - Side 8 = 2
	eng.GetComponent(compPos).IsSubMode = true
	eng.TriggerComparator(compPos, 10, 8)
	eng.Step()
	if p := eng.GetComponent(compPos).PowerLevel; p != 2 {
		t.Fatalf("subtraction mode failed: got %d, expected 2", p)
	}
}
