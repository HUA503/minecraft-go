package render

import "github.com/HUA503/minecraft-go/pkg/inventory"

// Vertex represents a 3D textured, lit vertex in the chunk mesh.
type Vertex struct {
	X, Y, Z      float32
	U, V         float32
	NormalX      float32
	NormalY      float32
	NormalZ      float32
	BlockID      int32
	FaceIndex    int // 0: +X, 1: -X, 2: +Y, 3: -Y, 4: +Z, 5: -Z
	LightLevel   uint8
	LightingMult float32
}

// Mesh holds generated geometry buffers for rendering.
type Mesh struct {
	Vertices []Vertex
	Indices  []uint32
}

// Face directions for 6 sides of a voxel cube.
var FaceNormals = [6][3]float32{
	{1, 0, 0},  // +X (East)
	{-1, 0, 0}, // -X (West)
	{0, 1, 0},  // +Y (Up)
	{0, -1, 0}, // -Y (Down)
	{0, 0, 1},  // +Z (South)
	{0, 0, -1}, // -Z (North)
}

// Directional lighting multiplier per face (Minecraft classic standard).
var FaceLighting = [6]float32{
	0.6, // +X (East)
	0.6, // -X (West)
	1.0, // +Y (Top sunlight)
	0.5, // -Y (Bottom shadow)
	0.8, // +Z (South)
	0.8, // -Z (North)
}

// Neighbor offsets in block coordinates.
var NeighborOffsets = [6][3]int32{
	{1, 0, 0},
	{-1, 0, 0},
	{0, 1, 0},
	{0, -1, 0},
	{0, 0, 1},
	{0, 0, -1},
}

// IsOpaqueCube returns whether a block is a solid, fully opaque 1x1x1 cube that hides neighbor faces.
func IsOpaqueCube(blockID int32) bool {
	if inventory.IsExtendedLeaves(blockID) || inventory.IsExtendedFoliage(blockID) {
		return false
	}
	if boxes, ok := inventory.BlockCollisionBoxes(blockID); ok {
		for _, box := range boxes {
			if box.MinX > 0 || box.MinY > 0 || box.MinZ > 0 || box.MaxX < 1 || box.MaxY < 1 || box.MaxZ < 1 {
				return false
			}
		}
		return len(boxes) > 0
	}
	switch blockID {
	case 0: // Air
		return false
	case 8, 9: // Water
		return false
	case 10, 11: // Lava
		return false
	case 18: // Oak Leaves (cutout alpha)
		return false
	case 20: // Glass (transparent)
		return false
	case 90: // Nether portal (translucent)
		return false
	case 6, 31, 32: // Sapling, tall grass, dead bush
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

// IsTranslucentBlock identifies blocks that must be sorted and blended after
// opaque terrain. Cutout textures such as leaves stay in the opaque pass.
func IsTranslucentBlock(blockID int32) bool {
	switch blockID {
	case 8, 9, 20, 90: // Water, still water, glass, nether portal
		return true
	default:
		return false
	}
}

// IsFoliage returns true if the block is rendered as crossed diagonal quads.
func IsFoliage(blockID int32) bool {
	if inventory.IsExtendedFoliage(blockID) {
		return true
	}
	switch blockID {
	case 6, 31, 32, 37, 38, 50, 65, 83:
		return true
	default:
		return false
	}
}

func isCutoutCube(blockID int32) bool {
	if inventory.IsExtendedLeaves(blockID) {
		return true
	}
	return blockID == 18 || blockID == 20 || blockID == 90
}

// BuildSectionMesh generates face-culled geometry for a 16x16x16 section.
func BuildSectionMesh(getBlock func(x, y, z int32) int32, originX, originY, originZ int32) *Mesh {
	mesh := &Mesh{
		Vertices: make([]Vertex, 0, 1024),
		Indices:  make([]uint32, 0, 1536),
	}

	for y := int32(0); y < 16; y++ {
		for z := int32(0); z < 16; z++ {
			for x := int32(0); x < 16; x++ {
				blockID := getBlock(originX+x, originY+y, originZ+z)
				if blockID == 0 {
					continue // Air: no mesh
				}
				if blockID == 65 {
					addLadderQuad(mesh, originX+x, originY+y, originZ+z, getBlock)
					continue
				}
				if _, _, _, isStair := inventory.StairStateProperties(blockID); isStair {
					addStairBlockMesh(mesh, originX+x, originY+y, originZ+z, blockID, getBlock)
					continue
				}
				if inventory.IsWoodFenceBlock(blockID) {
					boxes, _ := inventory.WoodFenceCollisionBoxesAt(blockID, originX+x, originY+y, originZ+z, getBlock)
					addBlockBoxesMesh(mesh, originX+x, originY+y, originZ+z, blockID, boxes, getBlock)
					continue
				}
				if inventory.IsMasonryWallBlock(blockID) {
					boxes, _ := inventory.MasonryWallCollisionBoxesAt(blockID, originX+x, originY+y, originZ+z, getBlock)
					addBlockBoxesMesh(mesh, originX+x, originY+y, originZ+z, blockID, boxes, getBlock)
					continue
				}
				if boxes, ok := inventory.BlockCollisionBoxes(blockID); ok && !IsOpaqueCube(blockID) {
					addPartialBlockMesh(mesh, originX+x, originY+y, originZ+z, blockID, boxes, getBlock)
					continue
				}

				// Special crossed diagonal quads for foliage and the torch sprite.
				if IsFoliage(blockID) {
					addCrossedQuads(mesh, originX+x, originY+y, originZ+z, blockID)
					continue
				}

				// Check each of 6 faces
				for face := 0; face < 6; face++ {
					off := NeighborOffsets[face]
					neighborID := getBlock(originX+x+off[0], originY+y+off[1], originZ+z+off[2])

					// If neighbor is an opaque full cube, this face is hidden
					if IsOpaqueCube(neighborID) {
						continue
					}
					// Internal face culling between transparent/translucent blocks (both water blocks 8 and 9 cull mutually)
					isWaterA := (blockID == 8 || blockID == 9)
					isWaterB := (neighborID == 8 || neighborID == 9)
					if (isWaterA && isWaterB) || (blockID == neighborID && isCutoutCube(blockID)) {
						continue
					}
					addFaceQuad(mesh, originX+x, originY+y, originZ+z, face, blockID, getBlock)
				}
			}
		}
	}

	return mesh
}

func addBlockBoxesMesh(mesh *Mesh, x, y, z, blockID int32, boxes []inventory.BlockBox, getBlock func(x, y, z int32) int32) {
	for _, box := range boxes {
		for face, offset := range NeighborOffsets {
			neighbor := getBlock(x+offset[0], y+offset[1], z+offset[2])
			if IsOpaqueCube(neighbor) {
				continue
			}
			if inventory.IsWoodFenceBlock(neighbor) && (face == 0 || face == 1 || face == 4 || face == 5) {
				continue
			}
			base := len(mesh.Vertices)
			addFaceQuad(mesh, x, y, z, face, blockID, getBlock)
			for i := base; i < len(mesh.Vertices); i++ {
				vertex := &mesh.Vertices[i]
				vertex.X = float32(x) + box.MinX + (vertex.X-float32(x))*(box.MaxX-box.MinX)
				vertex.Y = float32(y) + box.MinY + (vertex.Y-float32(y))*(box.MaxY-box.MinY)
				vertex.Z = float32(z) + box.MinZ + (vertex.Z-float32(z))*(box.MaxZ-box.MinZ)
				vertex.BlockID = inventory.BlockItemForState(blockID)
			}
		}
	}
}

func addStairBlockMesh(mesh *Mesh, x, y, z, blockID int32, getBlock func(x, y, z int32) int32) {
	shape := inventory.StairShapeAt(blockID, x, y, z, getBlock)
	for cy := 0; cy < 2; cy++ {
		for cz := 0; cz < 2; cz++ {
			for cx := 0; cx < 2; cx++ {
				if !inventory.StairVoxelOccupiedWithShape(blockID, cx, cy, cz, shape) {
					continue
				}
				for face, offset := range [6][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
					nx, ny, nz := cx+offset[0], cy+offset[1], cz+offset[2]
					if nx >= 0 && nx < 2 && ny >= 0 && ny < 2 && nz >= 0 && nz < 2 {
						if inventory.StairVoxelOccupiedWithShape(blockID, nx, ny, nz, shape) {
							continue
						}
					} else {
						neighbor := getBlock(x+int32(offset[0]), y+int32(offset[1]), z+int32(offset[2]))
						if IsOpaqueCube(neighbor) {
							continue
						}
					}

					base := len(mesh.Vertices)
					addFaceQuad(mesh, x, y, z, face, blockID, getBlock)
					for i := base; i < len(mesh.Vertices); i++ {
						vertex := &mesh.Vertices[i]
						vertex.X = float32(x) + float32(cx)*.5 + (vertex.X-float32(x))*.5
						vertex.Y = float32(y) + float32(cy)*.5 + (vertex.Y-float32(y))*.5
						vertex.Z = float32(z) + float32(cz)*.5 + (vertex.Z-float32(z))*.5
						lx, ly, lz := vertex.X-float32(x), vertex.Y-float32(y), vertex.Z-float32(z)
						switch face {
						case 0, 1:
							vertex.U, vertex.V = lz, 1-ly
						case 2, 3:
							vertex.U, vertex.V = lx, lz
						case 4:
							vertex.U, vertex.V = 1-lx, 1-ly
						case 5:
							vertex.U, vertex.V = lx, 1-ly
						}
						vertex.BlockID = inventory.BlockItemForState(blockID)
					}
				}
			}
		}
	}
}

func addPartialBlockMesh(mesh *Mesh, x, y, z int32, blockID int32, boxes []inventory.BlockBox, getBlock func(x, y, z int32) int32) {
	for _, box := range boxes {
		for face, offset := range NeighborOffsets {
			neighborID := getBlock(x+offset[0], y+offset[1], z+offset[2])
			if IsOpaqueCube(neighborID) {
				continue
			}
			if face != 2 && face != 3 && neighborID == blockID {
				continue
			}

			baseVertex := len(mesh.Vertices)
			addFaceQuad(mesh, x, y, z, face, blockID, getBlock)
			fy := float32(y)
			for i := baseVertex; i < len(mesh.Vertices); i++ {
				vertex := &mesh.Vertices[i]
				vertex.Y = fy + box.MinY + (vertex.Y-fy)*(box.MaxY-box.MinY)
				vertex.BlockID = inventory.BlockItemForState(blockID)
			}
		}
	}
}

// calcAO returns an authentic Minecraft ambient occlusion factor (0.5 to 1.0) for a corner vertex based on adjacent blocks.
func calcAO(s1, s2, c bool) float32 {
	if s1 && s2 {
		return 0.5
	}
	count := 0
	if s1 {
		count++
	}
	if s2 {
		count++
	}
	if c {
		count++
	}
	switch count {
	case 1:
		return 0.8
	case 2:
		return 0.65
	case 3:
		return 0.5
	default:
		return 1.0
	}
}

func addFaceQuad(mesh *Mesh, x, y, z int32, face int, blockID int32, getBlock func(x, y, z int32) int32) {
	fx, fy, fz := float32(x), float32(y), float32(z)
	norm := FaceNormals[face]
	baseIdx := uint32(len(mesh.Vertices))

	isSolid := func(bx, by, bz int32) bool {
		return IsOpaqueCube(getBlock(bx, by, bz))
	}

	var corners [4][3]float32
	var ao [4]float32

	switch face {
	case 0: // +X (East)
		corners = [4][3]float32{
			{fx + 1, fy, fz},
			{fx + 1, fy + 1, fz},
			{fx + 1, fy + 1, fz + 1},
			{fx + 1, fy, fz + 1},
		}
		ao[0] = calcAO(isSolid(x+1, y-1, z), isSolid(x+1, y, z-1), isSolid(x+1, y-1, z-1))
		ao[1] = calcAO(isSolid(x+1, y+1, z), isSolid(x+1, y, z-1), isSolid(x+1, y+1, z-1))
		ao[2] = calcAO(isSolid(x+1, y+1, z), isSolid(x+1, y, z+1), isSolid(x+1, y+1, z+1))
		ao[3] = calcAO(isSolid(x+1, y-1, z), isSolid(x+1, y, z+1), isSolid(x+1, y-1, z+1))

	case 1: // -X (West)
		corners = [4][3]float32{
			{fx, fy, fz + 1},
			{fx, fy + 1, fz + 1},
			{fx, fy + 1, fz},
			{fx, fy, fz},
		}
		ao[0] = calcAO(isSolid(x-1, y-1, z), isSolid(x-1, y, z+1), isSolid(x-1, y-1, z+1))
		ao[1] = calcAO(isSolid(x-1, y+1, z), isSolid(x-1, y, z+1), isSolid(x-1, y+1, z+1))
		ao[2] = calcAO(isSolid(x-1, y+1, z), isSolid(x-1, y, z-1), isSolid(x-1, y+1, z-1))
		ao[3] = calcAO(isSolid(x-1, y-1, z), isSolid(x-1, y, z-1), isSolid(x-1, y-1, z-1))

	case 2: // +Y (Top)
		corners = [4][3]float32{
			{fx, fy + 1, fz},
			{fx, fy + 1, fz + 1},
			{fx + 1, fy + 1, fz + 1},
			{fx + 1, fy + 1, fz},
		}
		ao[0] = calcAO(isSolid(x-1, y+1, z), isSolid(x, y+1, z-1), isSolid(x-1, y+1, z-1))
		ao[1] = calcAO(isSolid(x-1, y+1, z), isSolid(x, y+1, z+1), isSolid(x-1, y+1, z+1))
		ao[2] = calcAO(isSolid(x+1, y+1, z), isSolid(x, y+1, z+1), isSolid(x+1, y+1, z+1))
		ao[3] = calcAO(isSolid(x+1, y+1, z), isSolid(x, y+1, z-1), isSolid(x+1, y+1, z-1))

	case 3: // -Y (Bottom)
		corners = [4][3]float32{
			{fx, fy, fz + 1},
			{fx, fy, fz},
			{fx + 1, fy, fz},
			{fx + 1, fy, fz + 1},
		}
		ao[0] = calcAO(isSolid(x-1, y-1, z), isSolid(x, y-1, z+1), isSolid(x-1, y-1, z+1))
		ao[1] = calcAO(isSolid(x-1, y-1, z), isSolid(x, y-1, z-1), isSolid(x-1, y-1, z-1))
		ao[2] = calcAO(isSolid(x+1, y-1, z), isSolid(x, y-1, z-1), isSolid(x+1, y-1, z-1))
		ao[3] = calcAO(isSolid(x+1, y-1, z), isSolid(x, y-1, z+1), isSolid(x+1, y-1, z+1))

	case 4: // +Z (South)
		corners = [4][3]float32{
			{fx + 1, fy, fz + 1},
			{fx + 1, fy + 1, fz + 1},
			{fx, fy + 1, fz + 1},
			{fx, fy, fz + 1},
		}
		ao[0] = calcAO(isSolid(x+1, y, z+1), isSolid(x, y-1, z+1), isSolid(x+1, y-1, z+1))
		ao[1] = calcAO(isSolid(x+1, y, z+1), isSolid(x, y+1, z+1), isSolid(x+1, y+1, z+1))
		ao[2] = calcAO(isSolid(x-1, y, z+1), isSolid(x, y+1, z+1), isSolid(x-1, y+1, z+1))
		ao[3] = calcAO(isSolid(x-1, y, z+1), isSolid(x, y-1, z+1), isSolid(x-1, y-1, z+1))

	case 5: // -Z (North)
		corners = [4][3]float32{
			{fx, fy, fz},
			{fx, fy + 1, fz},
			{fx + 1, fy + 1, fz},
			{fx + 1, fy, fz},
		}
		ao[0] = calcAO(isSolid(x-1, y, z-1), isSolid(x, y-1, z-1), isSolid(x-1, y-1, z-1))
		ao[1] = calcAO(isSolid(x-1, y, z-1), isSolid(x, y+1, z-1), isSolid(x-1, y+1, z-1))
		ao[2] = calcAO(isSolid(x+1, y, z-1), isSolid(x, y+1, z-1), isSolid(x+1, y+1, z-1))
		ao[3] = calcAO(isSolid(x+1, y, z-1), isSolid(x, y-1, z-1), isSolid(x+1, y-1, z-1))
	}

	var uvs [4][2]float32
	if face == 2 { // Top (+Y)
		uvs = [4][2]float32{
			{0, 0},
			{0, 1},
			{1, 1},
			{1, 0},
		}
	} else if face == 3 { // Bottom (-Y)
		uvs = [4][2]float32{
			{0, 1},
			{0, 0},
			{1, 0},
			{1, 1},
		}
	} else { // Sides (0, 1, 4, 5): Corners 0 & 3 are bottom (V=1), corners 1 & 2 are top (V=0)
		uvs = [4][2]float32{
			{0, 1},
			{0, 0},
			{1, 0},
			{1, 1},
		}
	}

	for i := 0; i < 4; i++ {
		mesh.Vertices = append(mesh.Vertices, Vertex{
			X:            corners[i][0],
			Y:            corners[i][1],
			Z:            corners[i][2],
			U:            uvs[i][0],
			V:            uvs[i][1],
			NormalX:      norm[0],
			NormalY:      norm[1],
			NormalZ:      norm[2],
			BlockID:      inventory.BlockItemForState(blockID),
			FaceIndex:    face,
			LightingMult: FaceLighting[face] * ao[i],
		})
	}

	// Minecraft Java Edition classic diagonal flip to prevent anisotropic shading creases:
	if ao[0]+ao[2] < ao[1]+ao[3] {
		mesh.Indices = append(mesh.Indices,
			baseIdx+1, baseIdx+2, baseIdx+3,
			baseIdx+1, baseIdx+3, baseIdx,
		)
	} else {
		// Two triangles: (0, 1, 2) and (0, 2, 3)
		mesh.Indices = append(mesh.Indices,
			baseIdx, baseIdx+1, baseIdx+2,
			baseIdx, baseIdx+2, baseIdx+3,
		)
	}
}

// addCrossedQuads generates two diagonal double-sided quads for foliage (tall grass, flowers, torches).
func addCrossedQuads(mesh *Mesh, x, y, z int32, blockID int32) {
	fx, fy, fz := float32(x), float32(y), float32(z)
	baseIdx := uint32(len(mesh.Vertices))

	lightMult := float32(1.0)
	if blockID != 50 {
		lightMult = 0.95
	}

	var x0, z0, x1, z1, yTop float32
	if blockID == 50 {
		// Torches are slightly narrower/tighter footprint in block center
		x0, z0 = fx+0.15, fz+0.15
		x1, z1 = fx+0.85, fz+0.85
		yTop = fy + 0.85
	} else {
		x0, z0 = fx, fz
		x1, z1 = fx+1, fz+1
		yTop = fy + 1
	}

	// Diagonal 1: (x0, fy, z0) [bottom, V=1] -> (x0, yTop, z0) [top, V=0] -> (x1, yTop, z1) [top, V=0] -> (x1, fy, z1) [bottom, V=1]
	mesh.Vertices = append(mesh.Vertices,
		Vertex{X: x0, Y: fy, Z: z0, U: 0, V: 1, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
		Vertex{X: x0, Y: yTop, Z: z0, U: 0, V: 0, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
		Vertex{X: x1, Y: yTop, Z: z1, U: 1, V: 0, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
		Vertex{X: x1, Y: fy, Z: z1, U: 1, V: 1, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
	)

	// Diagonal 2: (x0, fy, z1) [bottom, V=1] -> (x0, yTop, z1) [top, V=0] -> (x1, yTop, z0) [top, V=0] -> (x1, fy, z0) [bottom, V=1]
	mesh.Vertices = append(mesh.Vertices,
		Vertex{X: x0, Y: fy, Z: z1, U: 0, V: 1, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
		Vertex{X: x0, Y: yTop, Z: z1, U: 0, V: 0, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
		Vertex{X: x1, Y: yTop, Z: z0, U: 1, V: 0, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
		Vertex{X: x1, Y: fy, Z: z0, U: 1, V: 1, BlockID: blockID, FaceIndex: 6, LightingMult: lightMult},
	)

	// Add double-sided indices (front and back) for both diagonals
	// Diagonal 1: front (0, 1, 2) and (0, 2, 3), back (2, 1, 0) and (3, 2, 0)
	mesh.Indices = append(mesh.Indices,
		baseIdx+0, baseIdx+1, baseIdx+2,
		baseIdx+0, baseIdx+2, baseIdx+3,
		baseIdx+2, baseIdx+1, baseIdx+0,
		baseIdx+3, baseIdx+2, baseIdx+0,
	)

	// Diagonal 2: front (4, 5, 6) and (4, 6, 7), back (6, 5, 4) and (7, 6, 4)
	mesh.Indices = append(mesh.Indices,
		baseIdx+4, baseIdx+5, baseIdx+6,
		baseIdx+4, baseIdx+6, baseIdx+7,
		baseIdx+6, baseIdx+5, baseIdx+4,
		baseIdx+7, baseIdx+6, baseIdx+4,
	)
}

func addLadderQuad(mesh *Mesh, x, y, z int32, getBlock func(x, y, z int32) int32) {
	fx, fy, fz := float32(x), float32(y), float32(z)
	const inset = float32(0.02)
	var vertices [4][3]float32
	switch {
	case IsOpaqueCube(getBlock(x+1, y, z)):
		planeX := fx + 1 - inset
		vertices = [4][3]float32{{planeX, fy, fz}, {planeX, fy + 1, fz}, {planeX, fy + 1, fz + 1}, {planeX, fy, fz + 1}}
	case IsOpaqueCube(getBlock(x-1, y, z)):
		planeX := fx + inset
		vertices = [4][3]float32{{planeX, fy, fz + 1}, {planeX, fy + 1, fz + 1}, {planeX, fy + 1, fz}, {planeX, fy, fz}}
	case IsOpaqueCube(getBlock(x, y, z+1)):
		planeZ := fz + 1 - inset
		vertices = [4][3]float32{{fx + 1, fy, planeZ}, {fx + 1, fy + 1, planeZ}, {fx, fy + 1, planeZ}, {fx, fy, planeZ}}
	case IsOpaqueCube(getBlock(x, y, z-1)):
		planeZ := fz + inset
		vertices = [4][3]float32{{fx, fy, planeZ}, {fx, fy + 1, planeZ}, {fx + 1, fy + 1, planeZ}, {fx + 1, fy, planeZ}}
	default:
		// Keep unsupported/floating ladder blocks visible with a stable default orientation.
		planeZ := fz + inset
		vertices = [4][3]float32{{fx, fy, planeZ}, {fx, fy + 1, planeZ}, {fx + 1, fy + 1, planeZ}, {fx + 1, fy, planeZ}}
	}

	base := uint32(len(mesh.Vertices))
	uv := [4][2]float32{{0, 1}, {0, 0}, {1, 0}, {1, 1}}
	for i, vertex := range vertices {
		mesh.Vertices = append(mesh.Vertices, Vertex{
			X: vertex[0], Y: vertex[1], Z: vertex[2], U: uv[i][0], V: uv[i][1],
			BlockID: 65, FaceIndex: 6, LightingMult: 0.9,
		})
	}
	mesh.Indices = append(mesh.Indices,
		base, base+1, base+2,
		base, base+2, base+3,
		base+2, base+1, base,
		base+3, base+2, base,
	)
}
