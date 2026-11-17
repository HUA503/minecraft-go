package redstone

import (
	"fmt"
)

// ComponentType identifies the redstone device.
type ComponentType int

const (
	TypeWire ComponentType = iota
	TypeTorch
	TypeRepeater
	TypeComparator
	TypeObserver
	TypePiston
	TypeStickyPiston
	TypeBlock
	TypeLever
)

// Direction represents 3D orientation for directional components.
type Direction int

const (
	DirNorth Direction = iota
	DirSouth
	DirWest
	DirEast
	DirUp
	DirDown
)

// Position in integer block coordinates.
type Pos struct {
	X, Y, Z int32
}

// ScheduledUpdate represents an action to execute at a specific game tick.
type ScheduledUpdate struct {
	TargetTick int64
	TargetPos  Pos
	Action     func(sim *Engine)
}

// Component represents a placed redstone device.
type Component struct {
	Type        ComponentType
	Position    Pos
	Facing      Direction
	PowerLevel  int // 0 to 15
	DelayTicks  int // for repeaters (1, 2, 3, 4 ticks)
	IsSubMode   bool // for comparators (compare vs subtract)
	IsExtended  bool // for pistons
	PushedBlock Pos  // position of attached block for sticky piston
}

// Engine manages the tick-driven redstone state and propagation.
type Engine struct {
	CurrentTick int64
	Components  map[Pos]*Component
	Scheduled   []ScheduledUpdate
}

func NewEngine() *Engine {
	return &Engine{
		CurrentTick: 0,
		Components:  make(map[Pos]*Component),
		Scheduled:   make([]ScheduledUpdate, 0),
	}
}

func (e *Engine) AddComponent(c *Component) {
	e.Components[c.Position] = c
}

func (e *Engine) GetComponent(p Pos) *Component {
	return e.Components[p]
}

func (e *Engine) ScheduleUpdate(delayTicks int, target Pos, action func(sim *Engine)) {
	e.Scheduled = append(e.Scheduled, ScheduledUpdate{
		TargetTick: e.CurrentTick + int64(delayTicks),
		TargetPos:  target,
		Action:     action,
	})
}

// Step advances the redstone engine by 1 tick (50ms).
func (e *Engine) Step() {
	e.CurrentTick++

	// 1. Process scheduled tick updates for current tick
	var remaining []ScheduledUpdate
	ready := make([]ScheduledUpdate, 0)
	for _, up := range e.Scheduled {
		if up.TargetTick <= e.CurrentTick {
			ready = append(ready, up)
		} else {
			remaining = append(remaining, up)
		}
	}
	e.Scheduled = remaining

	for _, up := range ready {
		up.Action(e)
	}

	// 2. Propagate wire and immediate signals
	e.propagateWires()
}

func (e *Engine) propagateWires() {
	// Find all power sources and calculate Manhattan distance signal decay
	powerMap := make(map[Pos]int)

	// Direct power sources (Torches, Levers, Redstone Blocks)
	for pos, comp := range e.Components {
		switch comp.Type {
		case TypeBlock, TypeLever:
			powerMap[pos] = comp.PowerLevel
		case TypeTorch:
			if comp.PowerLevel > 0 {
				powerMap[pos] = 15
			}
		case TypeRepeater:
			if comp.PowerLevel > 0 {
				powerMap[pos] = 15
			}
		}
	}

	// BFS signal spread for wires
	type node struct {
		p   Pos
		pwr int
	}
	queue := make([]node, 0)
	for pos, pwr := range powerMap {
		queue = append(queue, node{p: pos, pwr: pwr})
	}

	wirePower := make(map[Pos]int)
	neighbors := []Pos{
		{X: 1, Y: 0, Z: 0},
		{X: -1, Y: 0, Z: 0},
		{X: 0, Y: 0, Z: 1},
		{X: 0, Y: 0, Z: -1},
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.pwr <= 0 {
			continue
		}

		for _, offset := range neighbors {
			adj := Pos{X: curr.p.X + offset.X, Y: curr.p.Y + offset.Y, Z: curr.p.Z + offset.Z}
			comp := e.Components[adj]
			if comp != nil && comp.Type == TypeWire {
				newPwr := curr.pwr - 1
				if newPwr > wirePower[adj] {
					wirePower[adj] = newPwr
					queue = append(queue, node{p: adj, pwr: newPwr})
				}
			}
		}
	}

	// Update wire components
	for pos, pwr := range wirePower {
		if comp := e.Components[pos]; comp != nil && comp.Type == TypeWire {
			comp.PowerLevel = pwr
		}
	}
}

// TriggerRepeater simulates a repeater receiving an input.
func (e *Engine) TriggerRepeater(pos Pos, inputPower int) {
	rep := e.Components[pos]
	if rep == nil || rep.Type != TypeRepeater {
		return
	}
	e.ScheduleUpdate(rep.DelayTicks, pos, func(sim *Engine) {
		if inputPower > 0 {
			rep.PowerLevel = 15
		} else {
			rep.PowerLevel = 0
		}
	})
}

// TriggerObserver detects block state change at observed pos and triggers a 1-tick pulse.
func (e *Engine) TriggerObserver(pos Pos) {
	obs := e.Components[pos]
	if obs == nil || obs.Type != TypeObserver {
		return
	}
	// Observer emits 1-tick pulse of power 15 after 1 redstone tick delay
	e.ScheduleUpdate(1, pos, func(sim *Engine) {
		obs.PowerLevel = 15
		// Turns off after 1 tick pulse
		sim.ScheduleUpdate(1, pos, func(s *Engine) {
			obs.PowerLevel = 0
		})
	})
}

// TriggerPiston powers or unpowers a piston.
func (e *Engine) TriggerPiston(pos Pos, powered bool) {
	piston := e.Components[pos]
	if piston == nil || (piston.Type != TypePiston && piston.Type != TypeStickyPiston) {
		return
	}
	e.ScheduleUpdate(1, pos, func(sim *Engine) {
		piston.IsExtended = powered
		if powered {
			piston.PowerLevel = 15
		} else {
			piston.PowerLevel = 0
		}
	})
}

// TriggerComparator updates comparator output based on rear and side inputs.
func (e *Engine) TriggerComparator(pos Pos, rearInput int, sideInput int) {
	comp := e.Components[pos]
	if comp == nil || comp.Type != TypeComparator {
		return
	}
	e.ScheduleUpdate(1, pos, func(sim *Engine) {
		var out int
		if comp.IsSubMode {
			out = rearInput - sideInput
			if out < 0 {
				out = 0
			}
		} else {
			if rearInput >= sideInput {
				out = rearInput
			} else {
				out = 0
			}
		}
		comp.PowerLevel = out
	})
}

func (c *Component) String() string {
	return fmt.Sprintf("Component(Type=%d, Pos=%+v, Power=%d, Ext=%v)", c.Type, c.Position, c.PowerLevel, c.IsExtended)
}
