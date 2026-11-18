package render

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

				// Check each of 6 faces
				for face := 0; face < 6; face++ {
					off := NeighborOffsets[face]
					neighborID := getBlock(originX+x+off[0], originY+y+off[1], originZ+z+off[2])
					// If neighbor is air (0), this face is exposed
					if neighborID == 0 {
						addFaceQuad(mesh, originX+x, originY+y, originZ+z, face, blockID, getBlock)
					}
				}
			}
		}
	}

	return mesh
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
		return getBlock(bx, by, bz) != 0
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

	uvs := [4][2]float32{
		{0, 0},
		{0, 1},
		{1, 1},
		{1, 0},
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
			BlockID:      blockID,
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
