package world

import (
	"testing"
)

func TestCelestialCycleCalculations(t *testing.T) {
	// Noon (6000 ticks): Full daylight
	noon := ComputeCelestialState(6000)
	if !noon.IsDay || noon.SkyLightLevel != 15 {
		t.Fatalf("noon should be day with light 15: %+v", noon)
	}
	if noon.ZenithColor != 0xFF78A8F5 {
		t.Fatalf("expected sky blue at noon, got 0x%X", noon.ZenithColor)
	}

	// Midnight (18000 ticks): Nighttime with dim light
	midnight := ComputeCelestialState(18000)
	if midnight.IsDay || midnight.SkyLightLevel != 4 {
		t.Fatalf("midnight should be night with light 4: %+v", midnight)
	}
	if midnight.ZenithColor != 0xFF080C1E {
		t.Fatalf("expected midnight navy at midnight, got 0x%X", midnight.ZenithColor)
	}

	// Sunset (12500 ticks): Sunset twilight
	sunset := ComputeCelestialState(12500)
	if sunset.IsDay {
		t.Fatalf("sunset (12500) should transition out of day")
	}
	if sunset.SkyLightLevel <= 4 || sunset.SkyLightLevel >= 15 {
		t.Fatalf("sunset light level should be between 4 and 15, got %d", sunset.SkyLightLevel)
	}
}
