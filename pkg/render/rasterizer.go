package render

import (
	"image"
	"image/color"
	"math"

	"github.com/HUA503/minecraft-go/pkg/world"
)

// Framebuffer holds 2D pixel color and depth buffers.
type Framebuffer struct {
	Width  int
	Height int
	Color  []uint32  // RGBA packed 0xAABBGGRR
	Depth  []float32 // Z buffer: 0.0 (near) to 1.0 (far)
}

func NewFramebuffer(w, h int) *Framebuffer {
	return &Framebuffer{
		Width:  w,
		Height: h,
		Color:  make([]uint32, w*h),
		Depth:  make([]float32, w*h),
	}
}

func (fb *Framebuffer) Clear(clearColor uint32) {
	for i := range fb.Color {
		fb.Color[i] = clearColor
		fb.Depth[i] = 1.0 // Far plane
	}
}

// ClearSky renders daytime Minecraft sky with horizon fog.
func (fb *Framebuffer) ClearSky() {
	fb.ClearSkyWithTime(6000) // Default: noon daylight
}

// ClearSkyWithTime renders authentic day/sunset/night sky, stars, sun & moon.
func (fb *Framebuffer) ClearSkyWithTime(worldTime int64) {
	state := world.ComputeCelestialState(worldTime)
	zr := float32((state.ZenithColor >> 16) & 0xFF)
	zg := float32((state.ZenithColor >> 8) & 0xFF)
	zb := float32(state.ZenithColor & 0xFF)

	hr := float32((state.HorizonColor >> 16) & 0xFF)
	hg := float32((state.HorizonColor >> 8) & 0xFF)
	hb := float32(state.HorizonColor & 0xFF)

	for y := 0; y < fb.Height; y++ {
		t := float32(y) / float32(fb.Height)
		r := uint32(zr*(1-t) + hr*t)
		g := uint32(zg*(1-t) + hg*t)
		b := uint32(zb*(1-t) + hb*t)
		col := 0xFF000000 | (r << 16) | (g << 8) | b

		rowOffset := y * fb.Width
		for x := 0; x < fb.Width; x++ {
			fb.Color[rowOffset+x] = col
			fb.Depth[rowOffset+x] = 1.0
		}
	}

	if state.IsDay {
		// Draw classic square Minecraft Sun (bright golden square in upper sky)
		sunX := fb.Width/2 + 100
		sunY := 60
		fb.DrawRect(sunX-15, sunY-15, 30, 30, 0xFFFFFF80) // Sun core
		fb.DrawRect(sunX-18, sunY-18, 36, 36, 0x40FFFFCC) // Soft sun corona
	} else {
		// Draw square Minecraft Moon (pale silver-blue square)
		moonX := fb.Width/2 - 100
		moonY := 60
		fb.DrawRect(moonX-12, moonY-12, 24, 24, 0xFFEAEAFF) // Moon core
		fb.DrawRect(moonX-15, moonY-15, 30, 30, 0x30CCCCFF) // Moon glow
	}
}

func (fb *Framebuffer) SetPixel(x, y int, depth float32, col uint32) {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return
	}
	idx := y*fb.Width + x
	if depth < fb.Depth[idx] {
		fb.Depth[idx] = depth
		fb.Color[idx] = col
	}
}

func (fb *Framebuffer) ToRGBAImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, fb.Width, fb.Height))
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.Color[y*fb.Width+x]
			r := uint8((c >> 16) & 0xFF)
			g := uint8((c >> 8) & 0xFF)
			b := uint8(c & 0xFF)
			a := uint8((c >> 24) & 0xFF)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

// ScreenPoint represents 2D projected screen coordinates.
type ScreenPoint struct {
	X, Y  float32
	Depth float32
}

// ScreenVertex represents a textured 2D projected vertex with perspective attributes.
type ScreenVertex struct {
	X, Y      float32
	Depth     float32 // View-space linear Z depth
	U, V      float32
	InvZ      float32 // 1.0 / Depth for perspective correction
	UOverZ    float32 // U / Depth
	VOverZ    float32 // V / Depth
	BlockID   int32
	FaceIndex int
	LightMult float32
}

func NewScreenVertex(p ScreenPoint, u, v float32, blockID int32, faceIndex int, lightMult float32) ScreenVertex {
	depth := p.Depth
	if depth < 0.1 {
		depth = 0.1
	}
	invZ := 1.0 / depth
	return ScreenVertex{
		X:         p.X,
		Y:         p.Y,
		Depth:     depth,
		U:         u,
		V:         v,
		InvZ:      invZ,
		UOverZ:    u * invZ,
		VOverZ:    v * invZ,
		BlockID:   blockID,
		FaceIndex: faceIndex,
		LightMult: lightMult,
	}
}

// DrawTriangle draws a flat-shaded triangle into the framebuffer with depth test.
func (fb *Framebuffer) DrawTriangle(p0, p1, p2 ScreenPoint, col uint32) {
	minX := int(math.Floor(math.Min(float64(p0.X), math.Min(float64(p1.X), float64(p2.X)))))
	maxX := int(math.Ceil(math.Max(float64(p0.X), math.Max(float64(p1.X), float64(p2.X)))))
	minY := int(math.Floor(math.Min(float64(p0.Y), math.Min(float64(p1.Y), float64(p2.Y)))))
	maxY := int(math.Ceil(math.Max(float64(p0.Y), math.Max(float64(p1.Y), float64(p2.Y)))))

	if minX < 0 {
		minX = 0
	}
	if maxX >= fb.Width {
		maxX = fb.Width - 1
	}
	if minY < 0 {
		minY = 0
	}
	if maxY >= fb.Height {
		maxY = fb.Height - 1
	}

	edge := func(a, b ScreenPoint, px, py float32) float32 {
		return (px-a.X)*(b.Y-a.Y) - (py-a.Y)*(b.X-a.X)
	}

	area := edge(p0, p1, p2.X, p2.Y)
	if area == 0 {
		return
	}

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float32(x) + 0.5
			py := float32(y) + 0.5

			w0 := edge(p1, p2, px, py)
			w1 := edge(p2, p0, px, py)
			w2 := edge(p0, p1, px, py)

			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				w0 /= area
				w1 /= area
				w2 /= area
				z := w0*p0.Depth + w1*p1.Depth + w2*p2.Depth
				fb.SetPixel(x, y, z, col)
			}
		}
	}
}

// DrawVoxelTrianglePerspective performs mathematically exact perspective-correct texture mapping & distance fog.
func (fb *Framebuffer) DrawVoxelTrianglePerspective(v0, v1, v2 ScreenVertex) {
	minX := int(math.Floor(math.Min(float64(v0.X), math.Min(float64(v1.X), float64(v2.X)))))
	maxX := int(math.Ceil(math.Max(float64(v0.X), math.Max(float64(v1.X), float64(v2.X)))))
	minY := int(math.Floor(math.Min(float64(v0.Y), math.Min(float64(v1.Y), float64(v2.Y)))))
	maxY := int(math.Ceil(math.Max(float64(v0.Y), math.Max(float64(v1.Y), float64(v2.Y)))))

	if minX < 0 {
		minX = 0
	}
	if maxX >= fb.Width {
		maxX = fb.Width - 1
	}
	if minY < 0 {
		minY = 0
	}
	if maxY >= fb.Height {
		maxY = fb.Height - 1
	}
	if minX > maxX || minY > maxY {
		return
	}

	dx12 := v2.X - v1.X
	dy12 := v2.Y - v1.Y
	dx20 := v0.X - v2.X
	dy20 := v0.Y - v2.Y
	dx01 := v1.X - v0.X
	dy01 := v1.Y - v0.Y

	area := (v2.X-v0.X)*dy01 - (v2.Y-v0.Y)*dx01
	if area == 0 {
		return
	}
	invArea := 1.0 / area

	const fogStart = 0.35 // Normalized depth start
	const fogEnd = 0.85   // Normalized depth end (full fog)
	skyHazeR := float32(195)
	skyHazeG := float32(222)
	skyHazeB := float32(255)

	px0 := float32(minX) + 0.5
	py0 := float32(minY) + 0.5

	w0Row := (px0-v1.X)*dy12 - (py0-v1.Y)*dx12
	w1Row := (px0-v2.X)*dy20 - (py0-v2.Y)*dx20
	w2Row := (px0-v0.X)*dy01 - (py0-v0.Y)*dx01

	isPositiveArea := area > 0

	for y := minY; y <= maxY; y++ {
		rowOffset := y * fb.Width
		w0 := w0Row
		w1 := w1Row
		w2 := w2Row

		for x := minX; x <= maxX; x++ {
			inside := false
			if isPositiveArea {
				inside = (w0 >= 0 && w1 >= 0 && w2 >= 0)
			} else {
				inside = (w0 <= 0 && w1 <= 0 && w2 <= 0)
			}

			if inside {
				nw0 := w0 * invArea
				nw1 := w1 * invArea
				nw2 := w2 * invArea

				// Interpolate 1/Z linearly in screen space
				invZ := nw0*v0.InvZ + nw1*v1.InvZ + nw2*v2.InvZ
				depth := 1.0 / invZ // Recover true perspective depth

				idx := rowOffset + x
				if depth < fb.Depth[idx] {
					// Perspective-correct UV interpolation: (U/Z) / (1/Z)
					u := (nw0*v0.UOverZ + nw1*v1.UOverZ + nw2*v2.UOverZ) * depth
					v := (nw0*v0.VOverZ + nw1*v1.VOverZ + nw2*v2.VOverZ) * depth
					light := nw0*v0.LightMult + nw1*v1.LightMult + nw2*v2.LightMult

					pixelCol := SampleBlockColor(v0.BlockID, v0.FaceIndex, u, v, light)

					// Distance fog calculation
					if depth > fogStart {
						fogT := (depth - fogStart) / (fogEnd - fogStart)
						if fogT > 1.0 {
							fogT = 1.0
						}
						// Blend pixel with sky haze in 0xAARRGGBB order
						pr := float32((pixelCol >> 16) & 0xFF)
						pg := float32((pixelCol >> 8) & 0xFF)
						pb := float32(pixelCol & 0xFF)

						finalR := uint32(pr*(1-fogT) + skyHazeR*fogT)
						finalG := uint32(pg*(1-fogT) + skyHazeG*fogT)
						finalB := uint32(pb*(1-fogT) + skyHazeB*fogT)
						pixelCol = 0xFF000000 | (finalR << 16) | (finalG << 8) | finalB
					}

					fb.Depth[idx] = depth
					fb.Color[idx] = pixelCol
				}
			}

			w0 += dy12
			w1 += dy20
			w2 += dy01
		}

		w0Row -= dx12
		w1Row -= dx20
		w2Row -= dx01
	}
}

// DrawBlockOutline draws wireframe selection outline around targeted block edges.
func (fb *Framebuffer) DrawBlockOutline(cam *Camera, bx, by, bz int32) {
	fx, fy, fz := float64(bx), float64(by), float64(bz)

	corners := [8][3]float64{
		{fx, fy, fz},
		{fx + 1, fy, fz},
		{fx + 1, fy + 1, fz},
		{fx, fy + 1, fz},
		{fx, fy, fz + 1},
		{fx + 1, fy, fz + 1},
		{fx + 1, fy + 1, fz + 1},
		{fx, fy + 1, fz + 1},
	}

	edges := [12][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}

	var projected [8]ScreenPoint
	var valid [8]bool
	for i := 0; i < 8; i++ {
		projected[i], valid[i] = cam.ProjectToScreen(corners[i][0], corners[i][1], corners[i][2], fb.Width, fb.Height)
	}

	for _, e := range edges {
		if valid[e[0]] && valid[e[1]] {
			fb.DrawLine(int(projected[e[0]].X), int(projected[e[0]].Y), int(projected[e[1]].X), int(projected[e[1]].Y), 0xFF000000)
		}
	}
}

func (fb *Framebuffer) DrawLine(x0, y0, x1, y1 int, col uint32) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy

	for {
		if x0 >= 0 && x0 < fb.Width && y0 >= 0 && y0 < fb.Height {
			fb.Color[y0*fb.Width+x0] = col
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Particle represents a 3D textured debris particle emitted during block mining or breaking.
type Particle struct {
	X, Y, Z    float64
	Vx, Vy, Vz float64
	Color      uint32
	Life       float64
	MaxLife    float64
}

// DrawParticles renders 3D physical debris particles with perspective projection.
func (fb *Framebuffer) DrawParticles(cam *Camera, particles []Particle) {
	for _, p := range particles {
		if p.Life <= 0 {
			continue
		}
		sp, ok := cam.ProjectToScreen(p.X, p.Y, p.Z, fb.Width, fb.Height)
		if !ok {
			continue
		}
		sx := int(sp.X)
		sy := int(sp.Y)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				fb.SetPixel(sx+dx, sy+dy, sp.Depth, p.Color)
			}
		}
	}
}

// Canonical Minecraft crack lines per stage (0 through 9) in normalized [0, 1] face coordinates.
var crackSegments = [][4]float64{
	// Stage 0 (early hairline crack)
	{0.45, 0.45, 0.55, 0.55},
	{0.55, 0.45, 0.48, 0.52},
	// Stage 1
	{0.35, 0.35, 0.45, 0.45},
	{0.55, 0.55, 0.65, 0.65},
	// Stage 2
	{0.45, 0.45, 0.40, 0.70},
	{0.55, 0.45, 0.70, 0.40},
	// Stage 3
	{0.25, 0.30, 0.35, 0.35},
	{0.65, 0.65, 0.75, 0.80},
	// Stage 4
	{0.40, 0.70, 0.25, 0.85},
	{0.70, 0.40, 0.85, 0.30},
	// Stage 5
	{0.50, 0.50, 0.50, 0.20},
	{0.50, 0.20, 0.30, 0.15},
	// Stage 6
	{0.50, 0.20, 0.70, 0.10},
	{0.40, 0.70, 0.60, 0.80},
	// Stage 7
	{0.25, 0.85, 0.10, 0.90},
	{0.85, 0.30, 0.95, 0.35},
	// Stage 8
	{0.15, 0.15, 0.25, 0.30},
	{0.85, 0.85, 0.75, 0.80},
	// Stage 9 (full shattered spiderweb)
	{0.05, 0.50, 0.25, 0.45},
	{0.95, 0.50, 0.75, 0.55},
	{0.50, 0.05, 0.50, 0.20},
	{0.50, 0.95, 0.50, 0.80},
}

// DrawBlockDamage renders Minecraft 10-stage destruction cracks across the visible faces of a block.
func (fb *Framebuffer) DrawBlockDamage(cam *Camera, bx, by, bz int32, stage int) {
	if stage < 0 {
		return
	}
	if stage > 9 {
		stage = 9
	}

	// Determine how many crack lines to render for this destruction stage
	numSegments := (stage + 1) * 2 + (stage / 3)
	if numSegments > len(crackSegments) {
		numSegments = len(crackSegments)
	}

	fx, fy, fz := float64(bx), float64(by), float64(bz)
	crackColor := uint32(0xFF181818) // Dark charcoal crack texture

	// Check each face and draw cracks if face is pointing towards camera
	type faceSpec struct {
		visible bool
		pointAt func(u, v float64) (float64, float64, float64)
	}

	faces := [6]faceSpec{
		// 0: +X face
		{
			visible: cam.X > fx+1.0,
			pointAt: func(u, v float64) (float64, float64, float64) {
				return fx + 1.002, fy + v, fz + u
			},
		},
		// 1: -X face
		{
			visible: cam.X < fx,
			pointAt: func(u, v float64) (float64, float64, float64) {
				return fx - 0.002, fy + v, fz + (1.0 - u)
			},
		},
		// 2: +Y face (Top)
		{
			visible: cam.Y > fy+1.0,
			pointAt: func(u, v float64) (float64, float64, float64) {
				return fx + u, fy + 1.002, fz + v
			},
		},
		// 3: -Y face (Bottom)
		{
			visible: cam.Y < fy,
			pointAt: func(u, v float64) (float64, float64, float64) {
				return fx + u, fy - 0.002, fz + (1.0 - v)
			},
		},
		// 4: +Z face
		{
			visible: cam.Z > fz+1.0,
			pointAt: func(u, v float64) (float64, float64, float64) {
				return fx + (1.0 - u), fy + v, fz + 1.002
			},
		},
		// 5: -Z face
		{
			visible: cam.Z < fz,
			pointAt: func(u, v float64) (float64, float64, float64) {
				return fx + u, fy + v, fz - 0.002
			},
		},
	}

	for _, f := range faces {
		if !f.visible {
			continue
		}
		for i := 0; i < numSegments; i++ {
			seg := crackSegments[i]
			x0, y0, z0 := f.pointAt(seg[0], seg[1])
			x1, y1, z1 := f.pointAt(seg[2], seg[3])

			p0, ok0 := cam.ProjectToScreen(x0, y0, z0, fb.Width, fb.Height)
			p1, ok1 := cam.ProjectToScreen(x1, y1, z1, fb.Width, fb.Height)

			if ok0 && ok1 {
				fb.DrawLine(int(p0.X), int(p0.Y), int(p1.X), int(p1.Y), crackColor)
			}
		}
	}
}
