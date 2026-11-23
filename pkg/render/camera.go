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

	dx := math.Sin(radYaw) * math.Cos(radPitch)
	dy := math.Sin(radPitch) // looking up is +Y
	dz := math.Cos(radYaw) * math.Cos(radPitch)

	return dx, dy, dz
}

// GetRightVector computes normalized horizontal right strafe vector (dx, dz).
func (c *Camera) GetRightVector() (float64, float64) {
	radYaw := c.Yaw * math.Pi / 180.0
	return math.Cos(radYaw), -math.Sin(radYaw)
}

// ToViewSpace transforms a world point (wx, wy, wz) into camera view-space coordinates (rx, ry, vz).
func (c *Camera) ToViewSpace(wx, wy, wz float64) (float64, float64, float64) {
	// 1. Translate relative to camera
	tx := wx - c.X
	ty := wy - c.Y
	tz := wz - c.Z

	// 2. Rotate around Y axis (Yaw)
	radYaw := c.Yaw * math.Pi / 180.0
	cosY := math.Cos(radYaw)
	sinY := math.Sin(radYaw)

	rx := tx*cosY - tz*sinY
	rz := tx*sinY + tz*cosY

	// 3. Rotate around X axis (Pitch)
	radPitch := c.Pitch * math.Pi / 180.0
	cosP := math.Cos(radPitch)
	sinP := math.Sin(radPitch)

	ry := ty*cosP - rz*sinP
	vz := ty*sinP + rz*cosP

	return rx, ry, vz
}

// SphereInFrustum reports whether a world-space sphere may intersect the camera
// view frustum. The sphere is a conservative bound: false means it is fully
// outside, while true may include some false positives near the frustum edges.
func (c *Camera) SphereInFrustum(wx, wy, wz, radius float64, screenW, screenH int, near float64) bool {
	if c == nil || screenW <= 0 || screenH <= 0 || radius < 0 || near < 0 ||
		math.IsNaN(wx) || math.IsNaN(wy) || math.IsNaN(wz) || math.IsNaN(radius) || math.IsNaN(near) ||
		math.IsInf(wx, 0) || math.IsInf(wy, 0) || math.IsInf(wz, 0) || math.IsInf(radius, 0) || math.IsInf(near, 0) ||
		math.IsNaN(c.Fov) || math.IsInf(c.Fov, 0) || c.Fov <= 0 || c.Fov >= 180 {
		return true
	}

	x, y, z := c.ToViewSpace(wx, wy, wz)
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		return true
	}
	if z+radius < near {
		return false
	}

	tanHalfY := math.Tan(c.Fov * math.Pi / 360.0)
	tanHalfX := tanHalfY * float64(screenW) / float64(screenH)
	invLenX := 1.0 / math.Hypot(1, tanHalfX)
	invLenY := 1.0 / math.Hypot(1, tanHalfY)
	if (x+z*tanHalfX)*invLenX < -radius || (z*tanHalfX-x)*invLenX < -radius {
		return false
	}
	if (z*tanHalfY+y)*invLenY < -radius || (z*tanHalfY-y)*invLenY < -radius {
		return false
	}
	return true
}

// ProjectViewToScreen transforms view-space (vx, vy, vz) to 2D screen coordinates.
func (c *Camera) ProjectViewToScreen(vx, vy, vz float64, screenW, screenH int) ScreenPoint {
	fovRad := c.Fov * math.Pi / 180.0
	aspect := float64(screenW) / float64(screenH)
	f := 1.0 / math.Tan(fovRad/2.0)

	screenX := (vx*f/(aspect*vz))*float64(screenW/2) + float64(screenW/2)
	screenY := -(vy*f/vz)*float64(screenH/2) + float64(screenH/2)

	return ScreenPoint{
		X:     float32(screenX),
		Y:     float32(screenY),
		Depth: float32(vz), // Linear view depth in meters
	}
}

// ProjectToScreen transforms a world point (wx, wy, wz) to screen coordinates.
func (c *Camera) ProjectToScreen(wx, wy, wz float64, screenW, screenH int) (ScreenPoint, bool) {
	rx, ry, vz := c.ToViewSpace(wx, wy, wz)
	if vz <= 0.12 {
		return ScreenPoint{}, false
	}
	return c.ProjectViewToScreen(rx, ry, vz, screenW, screenH), true
}

// ViewVertex represents a 3D vertex transformed into camera view space.
type ViewVertex struct {
	X, Y, Z      float64
	U, V         float32
	BlockID      int32
	FaceIndex    int
	LightingMult float32
}

// ViewTriangle represents a triangle in camera view space.
type ViewTriangle struct {
	V0, V1, V2 ViewVertex
}

// ClipTriangleNear clips a view-space triangle against the near clipping plane (zNear) without heap allocations.
// Returns an array of up to 2 clipped triangles and the count of valid triangles (0, 1, or 2).
func ClipTriangleNear(v0, v1, v2 ViewVertex, zNear float64) ([2]ViewTriangle, int) {
	in0 := v0.Z >= zNear
	in1 := v1.Z >= zNear
	in2 := v2.Z >= zNear

	numIn := 0
	if in0 {
		numIn++
	}
	if in1 {
		numIn++
	}
	if in2 {
		numIn++
	}

	if numIn == 0 {
		return [2]ViewTriangle{}, 0
	}
	if numIn == 3 {
		return [2]ViewTriangle{{V0: v0, V1: v1, V2: v2}}, 1
	}

	lerp := func(a, b ViewVertex) ViewVertex {
		denom := b.Z - a.Z
		var t float32 = 0.5
		if math.Abs(denom) > 1e-7 {
			t = float32((zNear - a.Z) / denom)
		}
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		return ViewVertex{
			X:            a.X + float64(t)*(b.X-a.X),
			Y:            a.Y + float64(t)*(b.Y-a.Y),
			Z:            zNear,
			U:            a.U + t*(b.U-a.U),
			V:            a.V + t*(b.V-a.V),
			BlockID:      a.BlockID,
			FaceIndex:    a.FaceIndex,
			LightingMult: a.LightingMult,
		}
	}

	if numIn == 1 {
		// Exactly 1 vertex in front of near plane -> 1 smaller triangle
		if in0 {
			return [2]ViewTriangle{{V0: v0, V1: lerp(v0, v1), V2: lerp(v0, v2)}}, 1
		} else if in1 {
			return [2]ViewTriangle{{V0: v1, V1: lerp(v1, v2), V2: lerp(v1, v0)}}, 1
		} else {
			return [2]ViewTriangle{{V0: v2, V1: lerp(v2, v0), V2: lerp(v2, v1)}}, 1
		}
	}

	// Exactly 2 vertices in front of near plane -> quad clipped to 2 triangles
	if !in0 {
		p10 := lerp(v1, v0)
		p20 := lerp(v2, v0)
		return [2]ViewTriangle{
			{V0: v1, V1: v2, V2: p10},
			{V0: v2, V1: p20, V2: p10},
		}, 2
	} else if !in1 {
		p21 := lerp(v2, v1)
		p01 := lerp(v0, v1)
		return [2]ViewTriangle{
			{V0: v2, V1: v0, V2: p21},
			{V0: v0, V1: p01, V2: p21},
		}, 2
	} else {
		p02 := lerp(v0, v2)
		p12 := lerp(v1, v2)
		return [2]ViewTriangle{
			{V0: v0, V1: v1, V2: p02},
			{V0: v1, V1: p12, V2: p02},
		}, 2
	}
}

