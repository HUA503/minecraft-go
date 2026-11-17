package sound

import (
	"testing"
)

func TestSoundManagerDispatch(t *testing.T) {
	sm := NewSoundManager()
	var received []SoundEvent

	sm.AddListener(func(ev SoundEvent) {
		received = append(received, ev)
	})

	sm.PlaySound(SoundBlockStoneBreak, CategoryBlocks, 10, 64, 10, 1.0, 1.0)
	sm.PlaySound(SoundPlayerHurt, CategoryPlayers, 0, 64, 0, 0.8, 1.2)

	if len(received) != 2 {
		t.Fatalf("expected 2 received sounds, got %d", len(received))
	}
	if received[0].Name != SoundBlockStoneBreak || received[0].Category != CategoryBlocks {
		t.Fatalf("sound 0 mismatch: %+v", received[0])
	}
	if received[1].Name != SoundPlayerHurt || received[1].Pitch != 1.2 {
		t.Fatalf("sound 1 mismatch: %+v", received[1])
	}

	// Verify block sound lookup
	if GetBlockBreakSound(1) != SoundBlockStoneBreak {
		t.Fatalf("stone block should have stone break sound")
	}
	if GetBlockPlaceSound(5) != SoundBlockWoodPlace {
		t.Fatalf("wood block should have wood place sound")
	}
}
