package sim

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/inventory"
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

func (v Vec3) DistanceTo(other Vec3) float64 {
	return v.Sub(other).Length()
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

// IsSolidBlock returns whether a block impedes physical player movement.
// Air (0), liquids (8, 9, 10, 11), foliage (31, 32, 37, 38), and torches (50) are passable.
func IsSolidBlock(blockID int32) bool {
	if inventory.IsExtendedFoliage(blockID) {
		return false
	}
	switch blockID {
	case 0: // Air
		return false
	case 6, 10063: // Oak/poplar saplings
		return false
	case 8, 9: // Water
		return false
	case 10, 11: // Lava
		return false
	case 31, 32: // Tall grass, Dead bush
		return false
	case 37, 38: // Dandelion, Poppy
		return false
	case 50: // Torch
		return false
	case 65: // Ladder
		return false
	case 83: // Sugar Cane
		return false
	default:
		return true
	}
}

// PlacementIntersects checks if a block intersects a player bounding box for block placement.
// Uses a subtle tolerance margin (0.06) so standing adjacent to a block or jumping to pillar up
// does not trigger false intersection rejections.
func PlacementIntersects(playerBox AABB, blockBox AABB) bool {
	const eps = 0.06
	return (playerBox.Min.X+eps) < blockBox.Max.X && (playerBox.Max.X-eps) > blockBox.Min.X &&
		(playerBox.Min.Y+eps) < blockBox.Max.Y && (playerBox.Max.Y-eps) > blockBox.Min.Y &&
		(playerBox.Min.Z+eps) < blockBox.Max.Z && (playerBox.Max.Z-eps) > blockBox.Min.Z
}

// RaycastResult contains intersection details.
type RaycastResult struct {
	Hit      bool
	BlockX   int32
	BlockY   int32
	BlockZ   int32
	FaceX    int32
	FaceY    int32
	FaceZ    int32
	Distance float64
	HitPos   Vec3
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
		if distTraveled > 0 && isSolid(currX, currY, currZ) {
			return RaycastResult{
				Hit:      true,
				BlockX:   currX,
				BlockY:   currY,
				BlockZ:   currZ,
				FaceX:    faceX,
				FaceY:    faceY,
				FaceZ:    faceZ,
				Distance: distTraveled,
				HitPos: Vec3{
					X: origin.X + dir.X*distTraveled,
					Y: origin.Y + dir.Y*distTraveled,
					Z: origin.Z + dir.Z*distTraveled,
				},
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
// swept collision resolution with continuous sub-stepping. Resolves Y, X, and Z axes independently
// so players slide smoothly along walls, stand firmly on ground, and never clip through blocks or terrain.
// Returns resolved position, whether player is on ground, and whether ceiling was hit.
func ResolveMovementCollision(pos Vec3, delta Vec3, isSolid func(x, y, z int32) bool) (Vec3, bool, bool) {
	const halfW = 0.3
	const height = 1.8
	const epsilon = 1e-4

	onGround := false
	hitCeiling := false

	// Substep large movements to prevent high-velocity tunneling / wall clipping
	maxComponent := math.Max(math.Abs(delta.X), math.Max(math.Abs(delta.Y), math.Abs(delta.Z)))
	substeps := 1
	if maxComponent > 0.20 {
		substeps = int(math.Ceil(maxComponent / 0.18))
		if substeps > 10 {
			substeps = 10
		}
	}

	stepDelta := Vec3{
		X: delta.X / float64(substeps),
		Y: delta.Y / float64(substeps),
		Z: delta.Z / float64(substeps),
	}

	for s := 0; s < substeps; s++ {
		// 1. Gather all solid candidate boxes in expanded neighborhood of current step
		minX := int32(math.Floor(pos.X + math.Min(0, stepDelta.X) - halfW - 1))
		maxX := int32(math.Ceil(pos.X + math.Max(0, stepDelta.X) + halfW + 1))
		minY := int32(math.Floor(pos.Y + math.Min(0, stepDelta.Y) - 1))
		maxY := int32(math.Ceil(pos.Y + math.Max(0, stepDelta.Y) + height + 1))
		minZ := int32(math.Floor(pos.Z + math.Min(0, stepDelta.Z) - halfW - 1))
		maxZ := int32(math.Ceil(pos.Z + math.Max(0, stepDelta.Z) + halfW + 1))

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

		// 2. Resolve Y axis swept collision (gravity / landing / ceiling)
		dy := stepDelta.Y
		for _, b := range solidBoxes {
			// Check horizontal overlap between player and block box
			if pBox.Min.X < b.Max.X-epsilon && pBox.Max.X > b.Min.X+epsilon &&
				pBox.Min.Z < b.Max.Z-epsilon && pBox.Max.Z > b.Min.Z+epsilon {
				if dy < 0 {
					// Moving down: can only collide with floor blocks where top surface is at or below player's feet
					if b.Max.Y <= pBox.Min.Y+epsilon {
						maxDown := b.Max.Y - pBox.Min.Y
						if maxDown > dy {
							dy = maxDown
							onGround = true
						}
					}
				} else if dy > 0 {
					// Moving up: can only collide with ceiling blocks where bottom surface is at or above player's head
					if b.Min.Y >= pBox.Max.Y-epsilon {
						maxUp := b.Min.Y - pBox.Max.Y
						if maxUp < dy {
							dy = maxUp
							hitCeiling = true
						}
					}
				}
			}
		}
		pos.Y += dy
		pBox = NewPlayerAABB(pos)

		// 3. Resolve X axis swept collision (sliding along X walls)
		dx := stepDelta.X
		for _, b := range solidBoxes {
			// Check vertical and Z overlap
			if pBox.Min.Y < b.Max.Y-epsilon && pBox.Max.Y > b.Min.Y+epsilon &&
				pBox.Min.Z < b.Max.Z-epsilon && pBox.Max.Z > b.Min.Z+epsilon {
				if dx > 0 {
					// Moving right (+X): block is to the right
					if b.Min.X >= pBox.Max.X-epsilon {
						maxRight := b.Min.X - pBox.Max.X
						if maxRight < dx {
							dx = maxRight
						}
					}
				} else if dx < 0 {
					// Moving left (-X): block is to the left
					if b.Max.X <= pBox.Min.X+epsilon {
						maxLeft := b.Max.X - pBox.Min.X
						if maxLeft > dx {
							dx = maxLeft
						}
					}
				}
			}
		}
		pos.X += dx
		pBox = NewPlayerAABB(pos)

		// 4. Resolve Z axis swept collision (sliding along Z walls)
		dz := stepDelta.Z
		for _, b := range solidBoxes {
			// Check vertical and X overlap
			if pBox.Min.X < b.Max.X-epsilon && pBox.Max.X > b.Min.X+epsilon &&
				pBox.Min.Y < b.Max.Y-epsilon && pBox.Max.Y > b.Min.Y+epsilon {
				if dz > 0 {
					// Moving forward (+Z): block is ahead
					if b.Min.Z >= pBox.Max.Z-epsilon {
						maxFwd := b.Min.Z - pBox.Max.Z
						if maxFwd < dz {
							dz = maxFwd
						}
					}
				} else if dz < 0 {
					// Moving back (-Z): block is behind
					if b.Max.Z <= pBox.Min.Z+epsilon {
						maxBack := b.Max.Z - pBox.Min.Z
						if maxBack > dz {
							dz = maxBack
						}
					}
				}
			}
		}
		pos.Z += dz
		pBox = NewPlayerAABB(pos)

		// 5. Anti-penetration safety push-out: if player box penetrates any solid block, push along shallowest axis
		for _, b := range solidBoxes {
			if pBox.Min.X < b.Max.X-epsilon && pBox.Max.X > b.Min.X+epsilon &&
				pBox.Min.Y < b.Max.Y-epsilon && pBox.Max.Y > b.Min.Y+epsilon &&
				pBox.Min.Z < b.Max.Z-epsilon && pBox.Max.Z > b.Min.Z+epsilon {

				overlapX1 := b.Max.X - pBox.Min.X
				overlapX2 := pBox.Max.X - b.Min.X
				overlapY1 := b.Max.Y - pBox.Min.Y
				overlapY2 := pBox.Max.Y - b.Min.Y
				overlapZ1 := b.Max.Z - pBox.Min.Z
				overlapZ2 := pBox.Max.Z - b.Min.Z

				minPushX, signX := overlapX1, 1.0
				if overlapX2 < minPushX {
					minPushX, signX = overlapX2, -1.0
				}
				minPushY, signY := overlapY1, 1.0
				if overlapY2 < minPushY {
					minPushY, signY = overlapY2, -1.0
				}
				minPushZ, signZ := overlapZ1, 1.0
				if overlapZ2 < minPushZ {
					minPushZ, signZ = overlapZ2, -1.0
				}

				if minPushY <= minPushX && minPushY <= minPushZ && minPushY < 0.5 {
					pos.Y += signY * minPushY
					pBox = NewPlayerAABB(pos)
					if signY > 0 && stepDelta.Y <= 0 {
						onGround = true
					}
				} else if minPushX <= minPushZ && minPushX < 0.5 {
					pos.X += signX * minPushX
					pBox = NewPlayerAABB(pos)
				} else if minPushZ < 0.5 {
					pos.Z += signZ * minPushZ
					pBox = NewPlayerAABB(pos)
				}
			}
		}
	}

	// 6. Ground contact stability check
	if !onGround && delta.Y <= 0 {
		checkFoot := func(px, pz float64) bool {
			fx := int32(math.Floor(px))
			fy := int32(math.Floor(pos.Y - 0.05))
			fz := int32(math.Floor(pz))
			return isSolid(fx, fy, fz) && math.Abs(pos.Y-float64(fy+1)) < 0.10
		}
		if checkFoot(pos.X-halfW*0.8, pos.Z-halfW*0.8) ||
			checkFoot(pos.X+halfW*0.8, pos.Z-halfW*0.8) ||
			checkFoot(pos.X-halfW*0.8, pos.Z+halfW*0.8) ||
			checkFoot(pos.X+halfW*0.8, pos.Z+halfW*0.8) ||
			checkFoot(pos.X, pos.Z) {
			feetY := int32(math.Floor(pos.Y - 0.05))
			pos.Y = float64(feetY + 1)
			onGround = true
		}
	}

	return pos, onGround, hitCeiling
}
