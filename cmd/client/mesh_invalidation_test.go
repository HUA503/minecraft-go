package main

import (
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestAffectedMeshSectionsIncludesThreeAxisCornerNeighbors(t *testing.T) {
	keys := affectedMeshSections(world.BlockPos{X: 16, Y: 16, Z: 16})
	got := make(map[[3]int32]bool, len(keys))
	for _, key := range keys {
		got[key] = true
	}
	if len(got) != 8 {
		t.Fatalf("corner block should invalidate 8 sections, got %d: %v", len(got), keys)
	}
	for _, x := range []int32{0, 1} {
		for _, y := range []int32{0, 1} {
			for _, z := range []int32{0, 1} {
				key := [3]int32{x, y, z}
				if !got[key] {
					t.Errorf("corner block did not invalidate section %v", key)
				}
			}
		}
	}
}

func TestAffectedMeshSectionsUsesAbsoluteBoundariesForNegativeCoordinates(t *testing.T) {
	keys := affectedMeshSections(world.BlockPos{X: -16, Y: -2, Z: -32})
	got := make(map[[3]int32]bool, len(keys))
	for _, key := range keys {
		got[key] = true
	}
	// X=-16 and Z=-32 sit on chunk-aligned section boundaries; Y=-2 is
	// inside section -1 and must not spuriously invalidate section -2 or 0.
	for _, key := range [][3]int32{{-1, -1, -2}, {-2, -1, -2}, {-1, -1, -3}, {-2, -1, -3}} {
		if !got[key] {
			t.Errorf("negative boundary did not invalidate section %v; got %v", key, keys)
		}
	}
	if len(got) != 4 {
		t.Fatalf("two-axis boundary should invalidate 4 sections, got %d: %v", len(got), keys)
	}
}

func TestChunkMeshInvalidationIncludesDiagonalColumns(t *testing.T) {
	client := &Client{
		meshCache:     make(map[[3]int32]*render.Mesh),
		meshRevisions: make(map[[3]int32]uint64),
	}
	center := world.ChunkPos{X: 4, Z: -3}
	for dz := int32(-1); dz <= 1; dz++ {
		for dx := int32(-1); dx <= 1; dx++ {
			for sy := int32(world.MinSectionY); sy <= world.MaxSectionY; sy++ {
				key := [3]int32{center.X + dx, sy, center.Z + dz}
				client.meshCache[key] = &render.Mesh{}
			}
		}
	}

	client.meshMu.Lock()
	client.invalidateChunkMeshNeighborhoodLocked(center)
	client.meshMu.Unlock()

	if len(client.meshCache) != 0 {
		t.Fatalf("chunk load/unload left %d stale meshes cached", len(client.meshCache))
	}
	want := 9 * world.NumSections
	if len(client.meshRevisions) != want {
		t.Fatalf("expected %d invalidated section revisions, got %d", want, len(client.meshRevisions))
	}
	corner := [3]int32{center.X + 1, world.MaxSectionY, center.Z - 1}
	if client.meshRevisions[corner] != 1 {
		t.Errorf("diagonal section %v revision = %d, want 1", corner, client.meshRevisions[corner])
	}
}

func TestRebuildAffectedMeshesInitializesZeroValueCacheBeforeWorkerRuns(t *testing.T) {
	client := &Client{engine: sim.NewEngine()}
	client.RebuildAffectedMeshes(world.BlockPos{X: 8, Y: 64, Z: 8})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		client.meshJobsMu.Lock()
		pending, running := len(client.meshJobsPending), len(client.meshJobsRunning)
		client.meshJobsMu.Unlock()
		if pending == 0 && running == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	client.meshJobsMu.Lock()
	pending, running := len(client.meshJobsPending), len(client.meshJobsRunning)
	client.meshJobsMu.Unlock()
	if pending != 0 || running != 0 {
		t.Fatalf("mesh jobs did not drain: pending=%d running=%d", pending, running)
	}
	client.meshMu.RLock()
	cacheInitialized := client.meshCache != nil
	client.meshMu.RUnlock()
	if !cacheInitialized {
		t.Fatal("background mesh job completed without an initialized cache map")
	}
}
