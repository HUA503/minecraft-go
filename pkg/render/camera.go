package render

import (
	"math"
)

// Camera represents a first-person 3D camera.
type Camera struct {
	X, Y, Z float64
	Yaw     float64 // in degrees
	Pitch   float64 // in degrees
	Fov     float64 // field of view in degrees (e.g. 70.0)
}

func NewCamera(x, y, z float64) *Camera {
	return &Camera{
		X:     x,
		Y:     y,
		Z:     z,
		Yaw:   0,
		Pitch: 0,
		Fov:   70.0,
	}
}

// GetLookVector computes normalized forward direction vector (dx, dy, dz).
func (c *Camera) GetLookVector() (float64, float64, float64) {
	radYaw := c.Yaw * math.Pi / 180.0
	radPitch := c.Pitch * math.Pi / 180.0

	dx := -math.Sin(radYaw) * math.Cos(radPitch)
	dy := -math.Sin(radPitch)
	dz := math.Cos(radYaw) * math.Cos(radPitch)

	return dx, dy, dz
}

// ProjectToScreen transforms a world point (wx, wy, wz) to screen coordinates.
func (c *Camera) ProjectToScreen(wx, wy, wz float64, screenW, screenH int) (ScreenPoint, bool) {
	// 1. Translate relative to camera
	tx := wx - c.X
	ty := wy - c.Y
	tz := wz - c.Z

	// 2. Rotate around Y axis (Yaw)
	radYaw := -c.Yaw * math.Pi / 180.0
	cosY := math.Cos(radYaw)
	sinY := math.Sin(radYaw)

	rx := tx*cosY - tz*sinY
	rz := tx*sinY + tz*cosY

	// 3. Rotate around X axis (Pitch)
	radPitch := -c.Pitch * math.Pi / 180.0
	cosP := math.Cos(radPitch)
	sinP := math.Sin(radPitch)

	ry := ty*cosP - rz*sinP
	finalZ := ty*sinP + rz*cosP

	// Behind camera or near clipping plane
	if finalZ <= 0.1 {
		return ScreenPoint{}, false
	}

	// 4. Perspective projection
	fovRad := c.Fov * math.Pi / 180.0
	aspect := float64(screenW) / float64(screenH)
	f := 1.0 / math.Tan(fovRad/2.0)

	screenX := (rx * f / (aspect * finalZ)) * float64(screenW/2) + float64(screenW/2)
	screenY := -(ry * f / finalZ) * float64(screenH/2) + float64(screenH/2)

	return ScreenPoint{
		X:     float32(screenX),
		Y:     float32(screenY),
		Depth: float32(finalZ / 100.0), // normalized depth
	}, true
}
