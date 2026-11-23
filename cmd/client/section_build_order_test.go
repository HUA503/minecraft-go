package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestSectionBuildOrderCoversWorldHeight(t *testing.T) {
	for _, cameraSectionY := range []int32{world.MinSectionY, 4, 16, world.MaxSectionY, 24} {
		order := sectionBuildOrder(cameraSectionY)
		if len(order) != world.NumSections {
			t.Fatalf("camera section %d: got %d sections, want %d", cameraSectionY, len(order), world.NumSections)
		}

		seen := make(map[int32]bool, len(order))
		for _, sectionY := range order {
			if sectionY < world.MinSectionY || sectionY > world.MaxSectionY {
				t.Errorf("camera section %d: out-of-range section %d", cameraSectionY, sectionY)
			}
			if seen[sectionY] {
				t.Errorf("camera section %d: section %d appears more than once", cameraSectionY, sectionY)
			}
			seen[sectionY] = true
		}
		for sectionY := int32(world.MinSectionY); sectionY <= world.MaxSectionY; sectionY++ {
			if !seen[sectionY] {
				t.Errorf("camera section %d: missing section %d", cameraSectionY, sectionY)
			}
		}
	}
}

func TestSectionBuildOrderPrioritizesCamera(t *testing.T) {
	order := sectionBuildOrder(10)
	if len(order) < 3 || order[0] != 10 || order[1] != 11 || order[2] != 9 {
		t.Fatalf("unexpected camera-priority order: %v", order[:3])
	}
}
