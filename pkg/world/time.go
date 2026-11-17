package world

import "math"

// TimeConstants matching Minecraft Java 26.3 specifications.
const (
	TicksPerDay      int64 = 24000
	TimeSunrise      int64 = 0
	TimeNoon         int64 = 6000
	TimeSunset       int64 = 12000
	TimeNight        int64 = 13000
	TimeMidnight     int64 = 18000
	TimeSunriseDawn  int64 = 23000
)

// CelestialState contains lighting parameters, celestial angle, and sky colors.
type CelestialState struct {
	WorldTime     int64
	TimeOfDay     int64
	SunAngle      float64 // 0 to 2*Pi
	SkyLightLevel int     // 4 (midnight) to 15 (noon)
	IsDay         bool
	ZenithColor   uint32
	HorizonColor  uint32
}

// ComputeCelestialState calculates celestial angles and exact sky colors.
func ComputeCelestialState(worldTime int64) CelestialState {
	timeOfDay := worldTime % TicksPerDay
	if timeOfDay < 0 {
		timeOfDay += TicksPerDay
	}

	// Normalized day fraction [0.0, 1.0)
	dayFrac := float64(timeOfDay) / float64(TicksPerDay)
	sunAngle := dayFrac * 2.0 * math.Pi

	var zenith, horizon uint32
	var skyLight int
	isDay := false

	if timeOfDay >= 0 && timeOfDay < 11500 {
		// Daytime (Full daylight)
		isDay = true
		skyLight = 15
		zenith = 0xFF78A8F5  // Authentic Sky Blue
		horizon = 0xFFC3DEFF // Horizon haze
	} else if timeOfDay >= 11500 && timeOfDay < 13500 {
		// Sunset transition
		t := float64(timeOfDay-11500) / 2000.0
		skyLight = int(15.0*(1.0-t) + 4.0*t)
		isDay = false
		// Blend day sky to golden twilight sunset (0xFFD86030)
		rZ := uint32(float64(120)*(1-t) + float64(40)*t)
		gZ := uint32(float64(168)*(1-t) + float64(30)*t)
		bZ := uint32(float64(245)*(1-t) + float64(70)*t)
		zenith = 0xFF000000 | (rZ << 16) | (gZ << 8) | bZ

		rH := uint32(float64(195)*(1-t) + float64(220)*t)
		gH := uint32(float64(222)*(1-t) + float64(110)*t)
		bH := uint32(float64(255)*(1-t) + float64(50)*t)
		horizon = 0xFF000000 | (rH << 16) | (gH << 8) | bH
	} else if timeOfDay >= 13500 && timeOfDay < 22500 {
		// Nighttime (Deep starry sky)
		isDay = false
		skyLight = 4
		zenith = 0xFF080C1E  // Midnight Navy Black
		horizon = 0xFF14192E // Dim night horizon
	} else {
		// Sunrise dawn transition
		t := float64(timeOfDay-22500) / 1500.0
		skyLight = int(4.0*(1.0-t) + 15.0*t)
		isDay = true
		rZ := uint32(float64(40)*(1-t) + float64(120)*t)
		gZ := uint32(float64(30)*(1-t) + float64(168)*t)
		bZ := uint32(float64(70)*(1-t) + float64(245)*t)
		zenith = 0xFF000000 | (rZ << 16) | (gZ << 8) | bZ

		rH := uint32(float64(220)*(1-t) + float64(195)*t)
		gH := uint32(float64(110)*(1-t) + float64(222)*t)
		bH := uint32(float64(50)*(1-t) + float64(255)*t)
		horizon = 0xFF000000 | (rH << 16) | (gH << 8) | bH
	}

	return CelestialState{
		WorldTime:     worldTime,
		TimeOfDay:     timeOfDay,
		SunAngle:      sunAngle,
		SkyLightLevel: skyLight,
		IsDay:         isDay,
		ZenithColor:   zenith,
		HorizonColor:  horizon,
	}
}
