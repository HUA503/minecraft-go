package render

import (
	"image"
	"image/color"
	"math"
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

// ClearSky renders a vertical sky gradient (darker blue at zenith, lighter haze at horizon).
func (fb *Framebuffer) ClearSky() {
	zenithColor := [3]float32{120, 165, 240} // R, G, B
	horizonColor := [3]float32{195, 220, 255}

	for y := 0; y < fb.Height; y++ {
		t := float32(y) / float32(fb.Height)
		r := uint32(zenithColor[0]*(1-t) + horizonColor[0]*t)
		g := uint32(zenithColor[1]*(1-t) + horizonColor[1]*t)
		b := uint32(zenithColor[2]*(1-t) + horizonColor[2]*t)
		col := 0xFF000000 | (b << 16) | (g << 8) | r

		rowOffset := y * fb.Width
		for x := 0; x < fb.Width; x++ {
			fb.Color[rowOffset+x] = col
			fb.Depth[rowOffset+x] = 1.0
		}
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
			r := uint8(c & 0xFF)
			g := uint8((c >> 8) & 0xFF)
			b := uint8((c >> 16) & 0xFF)
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

// ScreenVertex represents a textured 2D projected vertex.
type ScreenVertex struct {
	X, Y      float32
	Depth     float32
	U, V      float32
	BlockID   int32
	FaceIndex int
	LightMult float32
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

// DrawVoxelTriangle draws a textured voxel triangle with authentic block texture and directional shading.
func (fb *Framebuffer) DrawVoxelTriangle(v0, v1, v2 ScreenVertex) {
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

	edge := func(ax, ay, bx, by, px, py float32) float32 {
		return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
	}

	area := edge(v0.X, v0.Y, v1.X, v1.Y, v2.X, v2.Y)
	if area == 0 {
		return
	}

	for y := minY; y <= maxY; y++ {
		rowOffset := y * fb.Width
		for x := minX; x <= maxX; x++ {
			px := float32(x) + 0.5
			py := float32(y) + 0.5

			w0 := edge(v1.X, v1.Y, v2.X, v2.Y, px, py)
			w1 := edge(v2.X, v2.Y, v0.X, v0.Y, px, py)
			w2 := edge(v0.X, v0.Y, v1.X, v1.Y, px, py)

			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				w0 /= area
				w1 /= area
				w2 /= area
				z := w0*v0.Depth + w1*v1.Depth + w2*v2.Depth

				idx := rowOffset + x
				if z < fb.Depth[idx] {
					u := w0*v0.U + w1*v1.U + w2*v2.U
					v := w0*v0.V + w1*v1.V + w2*v2.V
					col := SampleBlockColor(v0.BlockID, v0.FaceIndex, u, v, v0.LightMult)
					fb.Depth[idx] = z
					fb.Color[idx] = col
				}
			}
		}
	}
}

// DrawBlockOutline draws a wireframe selection outline around targeted block edges.
func (fb *Framebuffer) DrawBlockOutline(cam *Camera, bx, by, bz int32) {
	fx, fy, fz := float64(bx), float64(by), float64(bz)

	// 8 corners of the voxel
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

	// 12 edges
	edges := [12][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, // back face
		{4, 5}, {5, 6}, {6, 7}, {7, 4}, // front face
		{0, 4}, {1, 5}, {2, 6}, {3, 7}, // connecting edges
	}

	var projected [8]ScreenPoint
	var valid [8]bool
	for i := 0; i < 8; i++ {
		projected[i], valid[i] = cam.ProjectToScreen(corners[i][0], corners[i][1], corners[i][2], fb.Width, fb.Height)
	}

	for _, e := range edges {
		if valid[e[0]] && valid[e[1]] {
			fb.DrawLine(int(projected[e[0]].X), int(projected[e[0]].Y), int(projected[e[1]].X), int(projected[e[1]].Y), 0xFF000000) // Black outline
		}
	}
}

// DrawLine draws a 1px Bresenham line.
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
