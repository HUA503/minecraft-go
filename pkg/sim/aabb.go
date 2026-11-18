package sim

import (
	"math"
)

// Vec3 represents a 3D double vector for physics calculations.
type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return Vec3{X: v.X / l, Y: v.Y / l, Z: v.Z / l}
}

// AABB represents an Axis-Aligned Bounding Box.
type AABB struct {
	Min Vec3
	Max Vec3
}

func NewPlayerAABB(pos Vec3) AABB {
	const halfWidth = 0.3
	const height = 1.8
	return AABB{
		Min: Vec3{X: pos.X - halfWidth, Y: pos.Y, Z: pos.Z - halfWidth},
		Max: Vec3{X: pos.X + halfWidth, Y: pos.Y + height, Z: pos.Z + halfWidth},
	}
}

func NewBlockAABB(x, y, z int32) AABB {
	return AABB{
		Min: Vec3{X: float64(x), Y: float64(y), Z: float64(z)},
		Max: Vec3{X: float64(x + 1), Y: float64(y + 1), Z: float64(z + 1)},
	}
}

func (a AABB) Intersects(b AABB) bool {
	return a.Min.X < b.Max.X && a.Max.X > b.Min.X &&
		a.Min.Y < b.Max.Y && a.Max.Y > b.Min.Y &&
		a.Min.Z < b.Max.Z && a.Max.Z > b.Min.Z
}

// RaycastResult contains intersection details.
type RaycastResult struct {
	Hit    bool
	BlockX int32
	BlockY int32
	BlockZ int32
	FaceX  int32
	FaceY  int32
	FaceZ  int32
}

// Raycast performs 3D digital differential analysis (DDA) through voxels.
func Raycast(origin Vec3, dir Vec3, maxDistance float64, isSolid func(x, y, z int32) bool) RaycastResult {
	dir = dir.Normalize()
	if dir.Length() == 0 {
		return RaycastResult{}
	}

	currX := int32(math.Floor(origin.X))
	currY := int32(math.Floor(origin.Y))
	currZ := int32(math.Floor(origin.Z))

	var stepX, stepY, stepZ int32
	var tDeltaX, tDeltaY, tDeltaZ float64
	var tMaxX, tMaxY, tMaxZ float64

	if dir.X > 0 {
		stepX = 1
		tDeltaX = 1.0 / dir.X
		tMaxX = (float64(currX+1) - origin.X) / dir.X
	} else if dir.X < 0 {
		stepX = -1
		tDeltaX = -1.0 / dir.X
		tMaxX = (origin.X - float64(currX)) / -dir.X
	} else {
		tDeltaX = math.MaxFloat64
		tMaxX = math.MaxFloat64
	}

	if dir.Y > 0 {
		stepY = 1
		tDeltaY = 1.0 / dir.Y
		tMaxY = (float64(currY+1) - origin.Y) / dir.Y
	} else if dir.Y < 0 {
		stepY = -1
		tDeltaY = -1.0 / dir.Y
		tMaxY = (origin.Y - float64(currY)) / -dir.Y
	} else {
		tDeltaY = math.MaxFloat64
		tMaxY = math.MaxFloat64
	}

	if dir.Z > 0 {
		stepZ = 1
		tDeltaZ = 1.0 / dir.Z
		tMaxZ = (float64(currZ+1) - origin.Z) / dir.Z
	} else if dir.Z < 0 {
		stepZ = -1
		tDeltaZ = -1.0 / dir.Z
		tMaxZ = (origin.Z - float64(currZ)) / -dir.Z
	} else {
		tDeltaZ = math.MaxFloat64
		tMaxZ = math.MaxFloat64
	}

	var faceX, faceY, faceZ int32
	distTraveled := 0.0

	for distTraveled <= maxDistance {
		if isSolid(currX, currY, currZ) {
			return RaycastResult{
				Hit:    true,
				BlockX: currX,
				BlockY: currY,
				BlockZ: currZ,
				FaceX:  faceX,
				FaceY:  faceY,
				FaceZ:  faceZ,
			}
		}

		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				currX += stepX
				distTraveled = tMaxX
				tMaxX += tDeltaX
				faceX = -stepX
				faceY = 0
				faceZ = 0
			} else {
				currZ += stepZ
				distTraveled = tMaxZ
				tMaxZ += tDeltaZ
				faceX = 0
				faceY = 0
				faceZ = -stepZ
			}
		} else {
			if tMaxY < tMaxZ {
				currY += stepY
				distTraveled = tMaxY
				tMaxY += tDeltaY
				faceX = 0
				faceY = -stepY
				faceZ = 0
			} else {
				currZ += stepZ
				distTraveled = tMaxZ
				tMaxZ += tDeltaZ
				faceX = 0
				faceY = 0
				faceZ = -stepZ
			}
		}
	}

	return RaycastResult{Hit: false}
}

// ResolveMovementCollision implements Minecraft standard Axis-Aligned Bounding Box (AABB)
// swept collision resolution. Resolves Y, X, and Z axes independently so players slide
// smoothly along walls, stand firmly on ground, and never clip through blocks.
// Returns resolved position, whether player is on ground, and whether ceiling was hit.
func ResolveMovementCollision(pos Vec3, delta Vec3, isSolid func(x, y, z int32) bool) (Vec3, bool, bool) {
	const halfW = 0.3
	const height = 1.8
	const epsilon = 1e-4

	onGround := false
	hitCeiling := false

	// Search bounding box for nearby candidate solid blocks
	minX := int32(math.Floor(pos.X + math.Min(0, delta.X) - halfW - 1))
	maxX := int32(math.Ceil(pos.X + math.Max(0, delta.X) + halfW + 1))
	minY := int32(math.Floor(pos.Y + math.Min(0, delta.Y) - 1))
	maxY := int32(math.Ceil(pos.Y + math.Max(0, delta.Y) + height + 1))
	minZ := int32(math.Floor(pos.Z + math.Min(0, delta.Z) - halfW - 1))
	maxZ := int32(math.Ceil(pos.Z + math.Max(0, delta.Z) + halfW + 1))

	var solidBoxes []AABB
	for by := minY; by <= maxY; by++ {
		for bz := minZ; bz <= maxZ; bz++ {
			for bx := minX; bx <= maxX; bx++ {
				if isSolid(bx, by, bz) {
					solidBoxes = append(solidBoxes, NewBlockAABB(bx, by, bz))
				}
			}
		}
	}

	pBox := NewPlayerAABB(pos)

	// 1. Resolve Y axis first (gravity / jump / landing)
	dy := delta.Y
	for _, b := range solidBoxes {
		if pBox.Min.X < b.Max.X-epsilon && pBox.Max.X > b.Min.X+epsilon &&
			pBox.Min.Z < b.Max.Z-epsilon && pBox.Max.Z > b.Min.Z+epsilon {
			if dy < 0 && pBox.Min.Y >= b.Max.Y-epsilon {
				maxDown := b.Max.Y - pBox.Min.Y
				if maxDown > dy {
					dy = maxDown
					onGround = true
				}
			} else if dy > 0 && pBox.Max.Y <= b.Min.Y+epsilon {
				maxUp := b.Min.Y - pBox.Max.Y
				if maxUp < dy {
					dy = maxUp
					hitCeiling = true
				}
			}
		}
	}
	pos.Y += dy
	pBox = NewPlayerAABB(pos)

	// 2. Resolve X axis (sliding along X walls)
	dx := delta.X
	for _, b := range solidBoxes {
		if pBox.Min.Y < b.Max.Y-epsilon && pBox.Max.Y > b.Min.Y+epsilon &&
			pBox.Min.Z < b.Max.Z-epsilon && pBox.Max.Z > b.Min.Z+epsilon {
			if dx > 0 && pBox.Max.X <= b.Min.X+epsilon {
				maxRight := b.Min.X - pBox.Max.X
				if maxRight < dx {
					dx = maxRight
				}
			} else if dx < 0 && pBox.Min.X >= b.Max.X-epsilon {
				maxLeft := b.Max.X - pBox.Min.X
				if maxLeft > dx {
					dx = maxLeft
				}
			}
		}
	}
	pos.X += dx
	pBox = NewPlayerAABB(pos)

	// 3. Resolve Z axis (sliding along Z walls)
	dz := delta.Z
	for _, b := range solidBoxes {
		if pBox.Min.X < b.Max.X-epsilon && pBox.Max.X > b.Min.X+epsilon &&
			pBox.Min.Y < b.Max.Y-epsilon && pBox.Max.Y > b.Min.Y+epsilon {
			if dz > 0 && pBox.Max.Z <= b.Min.Z+epsilon {
				maxFwd := b.Min.Z - pBox.Max.Z
				if maxFwd < dz {
					dz = maxFwd
				}
			} else if dz < 0 && pBox.Min.Z >= b.Max.Z-epsilon {
				maxBack := b.Max.Z - pBox.Min.Z
				if maxBack > dz {
					dz = maxBack
				}
			}
		}
	}
	pos.Z += dz

	// Extra ground stability check (feet resting on solid block, only when not jumping upward)
	if !onGround && delta.Y <= 0 {
		feetX := int32(math.Floor(pos.X))
		feetY := int32(math.Floor(pos.Y - 0.05))
		feetZ := int32(math.Floor(pos.Z))
		if isSolid(feetX, feetY, feetZ) && math.Abs(pos.Y-float64(feetY+1)) < 0.15 {
			pos.Y = float64(feetY + 1)
			onGround = true
		}
	}

	return pos, onGround, hitCeiling
}
