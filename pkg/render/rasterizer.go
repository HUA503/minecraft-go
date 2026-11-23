package render

import (
	"image"
	"image/color"
	"math"

	"github.com/HUA503/minecraft-go/pkg/world"
)

// Framebuffer holds 2D pixel color and depth buffers.
type Framebuffer struct {
	Width           int
	Height          int
	Color           []uint32  // RGBA packed 0xAABBGGRR
	Depth           []float32 // Z buffer: 0.0 (near) to 1.0 (far)
	HorizonFogColor uint32    // Current celestial horizon color for distance fog
}

func NewFramebuffer(w, h int) *Framebuffer {
	if w <= 0 || h <= 0 {
		w, h = 800, 600
	}
	// The software renderer keeps both a color and a depth buffer. Bound the
	// allocation so malformed CLI dimensions cannot overflow w*h or exhaust RAM.
	const maxPixels = 4 * 1024 * 1024
	pixels := float64(w) * float64(h)
	if pixels > maxPixels {
		scale := math.Sqrt(float64(maxPixels) / pixels)
		w = max(1, int(float64(w)*scale))
		h = max(1, int(float64(h)*scale))
	}
	return &Framebuffer{
		Width:           w,
		Height:          h,
		Color:           make([]uint32, w*h),
		Depth:           make([]float32, w*h),
		HorizonFogColor: 0xFFC3DEFF,
	}
}

func (fb *Framebuffer) Clear(clearColor uint32) {
	for i := range fb.Color {
		fb.Color[i] = clearColor
		fb.Depth[i] = 1e9 // Far plane in linear view meters
	}
}

// ClearSky renders daytime Minecraft sky with horizon fog.
func (fb *Framebuffer) ClearSky() {
	fb.ClearSkyWithCamera(6000, nil) // Default: noon daylight
}

// ClearSkyWithTime renders authentic day/sunset/night sky, stars, sun & moon.
func (fb *Framebuffer) ClearSkyWithTime(worldTime int64) {
	fb.ClearSkyWithCamera(worldTime, nil)
}

// ClearSkyWithCamera renders authentic Minecraft sky dome with dynamic day/sunset/night colors
// realistically oriented according to camera pitch, so looking up/down correctly rotates the sky dome.
func (fb *Framebuffer) ClearSkyWithCamera(worldTime int64, cam *Camera) {
	state := world.ComputeCelestialState(worldTime)
	fb.HorizonFogColor = state.HorizonColor
	zr := float32((state.ZenithColor >> 16) & 0xFF)
	zg := float32((state.ZenithColor >> 8) & 0xFF)
	zb := float32(state.ZenithColor & 0xFF)

	hr := float32((state.HorizonColor >> 16) & 0xFF)
	hg := float32((state.HorizonColor >> 8) & 0xFF)
	hb := float32(state.HorizonColor & 0xFF)

	camPitchRad := 0.0
	fovHalfTan := math.Tan(35.0 * math.Pi / 180.0) // Default 70 FOV half
	if cam != nil {
		camPitchRad = cam.Pitch * math.Pi / 180.0
		fovHalfTan = math.Tan((cam.Fov * math.Pi / 180.0) / 2.0)
	}

	cy := float64(fb.Height) / 2.0

	for y := 0; y < fb.Height; y++ {
		normY := (cy - float64(y)) / cy
		rayPitch := camPitchRad + math.Atan(normY*fovHalfTan)

		var r, g, b uint32
		if rayPitch >= 0 {
			// Above horizon: smooth dome blend from horizon fog to deep zenith sky (0 to ~60 deg)
			t := float32(rayPitch / (math.Pi * 0.33))
			if t > 1.0 {
				t = 1.0
			}
			r = uint32(hr*(1-t) + zr*t)
			g = uint32(hg*(1-t) + zg*t)
			b = uint32(hb*(1-t) + zb*t)
		} else {
			// Below horizon: smooth blend into ground/void horizon haze
			tDown := float32(-rayPitch / (math.Pi * 0.25))
			if tDown > 1.0 {
				tDown = 1.0
			}
			r = uint32(hr * (1.0 - tDown*0.35))
			g = uint32(hg * (1.0 - tDown*0.35))
			b = uint32(hb * (1.0 - tDown*0.35))
		}

		col := 0xFF000000 | (r << 16) | (g << 8) | b
		rowOffset := y * fb.Width
		for x := 0; x < fb.Width; x++ {
			fb.Color[rowOffset+x] = col
			fb.Depth[rowOffset+x] = 1e9
		}
	}
}

var fixedStars [64][3]float64

func init() {
	// Deterministic spherical Fibonacci distribution across celestial dome
	for i := 0; i < len(fixedStars); i++ {
		fi := float64(i)
		theta := fi * 2.399963229728653 // Golden ratio angle
		y := 0.15 + (float64(i)/float64(len(fixedStars)))*0.85 // Upper hemisphere
		r := math.Sqrt(1.0 - y*y)
		fixedStars[i] = [3]float64{
			math.Cos(theta) * r,
			y,
			math.Sin(theta) * r,
		}
	}
}

// DrawCelestialBodies renders 3D celestial bodies (Sun, Moon, Stars) accurately anchored at infinity in 3D space.
func (fb *Framebuffer) DrawCelestialBodies(cam *Camera, worldTime int64) {
	if cam == nil {
		return
	}
	state := world.ComputeCelestialState(worldTime)
	dayProgress := float64(worldTime%24000) / 24000.0
	// In Minecraft: tick 0 is sunrise (+X East), tick 6000 is noon (+Y Zenith), tick 12000 is sunset (-X West)
	angle := dayProgress * 2.0 * math.Pi

	// Celestial orbit direction vectors (infinite distance, rotation only, zero translation parallax)
	sunDirX := math.Cos(angle)
	sunDirY := math.Sin(angle)
	sunDirZ := 0.0

	radYaw := cam.Yaw * math.Pi / 180.0
	cosY, sinY := math.Cos(radYaw), math.Sin(radYaw)
	radPitch := cam.Pitch * math.Pi / 180.0
	cosP, sinP := math.Cos(radPitch), math.Sin(radPitch)

	aspect := float64(fb.Width) / float64(fb.Height)
	fovRad := cam.Fov * math.Pi / 180.0
	f := 1.0 / math.Tan(fovRad/2.0)

	projectDir := func(dx, dy, dz float64) (int, int, bool) {
		// 1. Yaw rotation around Y axis
		rx := dx*cosY - dz*sinY
		rz := dx*sinY + dz*cosY

		// 2. Pitch rotation around X axis
		ry := dy*cosP - rz*sinP
		vz := dy*sinP + rz*cosP

		if vz <= 0.05 {
			return 0, 0, false
		}

		sx := (rx * f / (aspect * vz)) * float64(fb.Width/2) + float64(fb.Width/2)
		sy := -(ry * f / vz) * float64(fb.Height/2) + float64(fb.Height/2)
		return int(sx), int(sy), true
	}

	const celestialDepth = float32(1e8)

	// 1. Draw 3D Sun if above or near horizon
	if sunDirY > -0.25 {
		if sx, sy, ok := projectDir(sunDirX, sunDirY, sunDirZ); ok {
			sunRadius := int(float64(fb.Height) * 0.045 * (70.0 / cam.Fov))
			if sunRadius < 18 {
				sunRadius = 18
			}
			coronaRadius := int(float64(sunRadius) * 1.5)
			// Sun corona (outer glow)
			fb.drawBillboard(sx, sy, coronaRadius, coronaRadius, celestialDepth, 0x50FFE57F)
			// Sun core (bright Minecraft golden square)
			fb.drawBillboard(sx, sy, sunRadius, sunRadius, celestialDepth, 0xFFFFFFCC)
		}
	}

	// 2. Draw 3D Moon (opposite direction)
	if -sunDirY > -0.25 {
		if mx, my, ok := projectDir(-sunDirX, -sunDirY, -sunDirZ); ok {
			moonRadius := int(float64(fb.Height) * 0.035 * (70.0 / cam.Fov))
			if moonRadius < 14 {
				moonRadius = 14
			}
			glowRadius := int(float64(moonRadius) * 1.5)
			// Moon glow
			fb.drawBillboard(mx, my, glowRadius, glowRadius, celestialDepth, 0x4099BBEE)
			// Moon core (pale silver square)
			fb.drawBillboard(mx, my, moonRadius, moonRadius, celestialDepth, 0xFFF0F4FF)
		}
	}

	// 3. Draw 3D Stars at night / twilight
	starAlpha := float32(0.0)
	if !state.IsDay {
		starAlpha = 1.0 - float32(state.SkyLightLevel-4)/11.0
	}
	if starAlpha > 0.05 {
		for _, s := range fixedStars {
			if sx, sy, ok := projectDir(s[0], s[1], s[2]); ok {
				if sx >= 0 && sx < fb.Width && sy >= 0 && sy < fb.Height {
					idx := sy*fb.Width + sx
					if celestialDepth < fb.Depth[idx] {
						lum := uint32(starAlpha * 255.0)
						col := 0xFF000000 | (lum << 16) | (lum << 8) | lum
						fb.Color[idx] = col
						if sx+1 < fb.Width {
							fb.Color[idx+1] = col
						}
					}
				}
			}
		}
	}
}

// DrawClouds renders 3D procedural drifting clouds at Y=128 with authentic Minecraft Java aesthetics.
func (fb *Framebuffer) DrawClouds(cam *Camera, worldTime int64) {
	if cam == nil {
		return
	}
	state := world.ComputeCelestialState(worldTime)
	cloudY := 128.0

	// Determine cloud color based on celestial time (day, sunset/sunrise, night)
	cloudAlpha := uint32(0xD0)
	var cloudR, cloudG, cloudB uint32
	if state.IsDay {
		if worldTime >= 11500 && worldTime <= 12500 { // Sunset/twilight
			cloudR, cloudG, cloudB = 255, 180, 160
		} else if worldTime <= 1000 { // Sunrise
			cloudR, cloudG, cloudB = 255, 200, 170
		} else { // Daytime bright white
			cloudR, cloudG, cloudB = 245, 250, 255
		}
	} else { // Night slate blue
		cloudR, cloudG, cloudB = 35, 45, 65
	}
	cloudCol := (cloudAlpha << 24) | (cloudR << 16) | (cloudG << 8) | cloudB

	// Wind drift speed along X axis
	cloudDrift := float64(worldTime) * 0.2
	cellW := 12.0 // Cloud cell dimension: 12x12 blocks

	baseX := math.Floor((cam.X - cloudDrift) / cellW)
	baseZ := math.Floor(cam.Z / cellW)

	const radius = 8 // 17x17 grid around player (approx 200m radius)

	for dz := -radius; dz <= radius; dz++ {
		cz := int(baseZ) + dz
		for dx := -radius; dx <= radius; dx++ {
			cx := int(baseX) + dx

			// 2-octave integer noise for continuous cloud bands
			h1 := (math.Sin(float64(cx)*0.45)+math.Cos(float64(cz)*0.45))*50.0 + 50.0
			h2 := (math.Sin(float64(cx+cz)*0.25))*25.0
			if h1+h2 < 55.0 {
				continue // Clear sky here
			}

			x0 := float64(cx)*cellW + cloudDrift
			z0 := float64(cz) * cellW
			x1 := x0 + cellW
			z1 := z0 + cellW

			vx0, vy0, vz0 := cam.ToViewSpace(x0, cloudY, z0)
			vx1, vy1, vz1 := cam.ToViewSpace(x1, cloudY, z0)
			vx2, vy2, vz2 := cam.ToViewSpace(x0, cloudY, z1)
			vx3, vy3, vz3 := cam.ToViewSpace(x1, cloudY, z1)

			vv0 := ViewVertex{X: vx0, Y: vy0, Z: vz0}
			vv1 := ViewVertex{X: vx1, Y: vy1, Z: vz1}
			vv2 := ViewVertex{X: vx2, Y: vy2, Z: vz2}
			vv3 := ViewVertex{X: vx3, Y: vy3, Z: vz3}

			renderTri := func(va, vb, vc ViewVertex) {
				clipped, count := ClipTriangleNear(va, vb, vc, 0.10)
				for ci := 0; ci < count; ci++ {
					tri := clipped[ci]
					p0 := cam.ProjectViewToScreen(tri.V0.X, tri.V0.Y, tri.V0.Z, fb.Width, fb.Height)
					p1 := cam.ProjectViewToScreen(tri.V1.X, tri.V1.Y, tri.V1.Z, fb.Width, fb.Height)
					p2 := cam.ProjectViewToScreen(tri.V2.X, tri.V2.Y, tri.V2.Z, fb.Width, fb.Height)
					fb.drawFlatTriangleTranslucent(p0, p1, p2, cloudCol)
				}
			}

			renderTri(vv0, vv1, vv2)
			renderTri(vv1, vv3, vv2)
		}
	}
}

func (fb *Framebuffer) drawFlatTriangleTranslucent(p0, p1, p2 ScreenPoint, col uint32) {
	minX := int(math.Min(float64(p0.X), math.Min(float64(p1.X), float64(p2.X))))
	maxX := int(math.Max(float64(p0.X), math.Max(float64(p1.X), float64(p2.X))) + 0.999999)
	minY := int(math.Min(float64(p0.Y), math.Min(float64(p1.Y), float64(p2.Y))))
	maxY := int(math.Max(float64(p0.Y), math.Max(float64(p1.Y), float64(p2.Y))) + 0.999999)

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

	v0x, v0y := float64(p0.X), float64(p0.Y)
	v1x, v1y := float64(p1.X), float64(p1.Y)
	v2x, v2y := float64(p2.X), float64(p2.Y)

	dx12, dy12 := v2x-v1x, v2y-v1y
	dx20, dy20 := v0x-v2x, v0y-v2y
	dx01, dy01 := v1x-v0x, v1y-v0y

	area := (v2x-v0x)*dy01 - (v2y-v0y)*dx01
	absArea := math.Abs(area)
	if absArea < 1e-4 {
		return
	}
	invArea := 1.0 / area

	invZ0 := 1.0 / float64(p0.Depth)
	invZ1 := 1.0 / float64(p1.Depth)
	invZ2 := 1.0 / float64(p2.Depth)

	alpha := (col >> 24) & 0xFF
	r := (col >> 16) & 0xFF
	g := (col >> 8) & 0xFF
	b := col & 0xFF
	a := float32(alpha) / 255.0

	px0 := float64(minX) + 0.5
	py0 := float64(minY) + 0.5

	w0Row := (px0-v1x)*dy12 - (py0-v1y)*dx12
	w1Row := (px0-v2x)*dy20 - (py0-v2y)*dx20
	w2Row := (px0-v0x)*dy01 - (py0-v0y)*dx01

	isPositiveArea := area > 0

	for y := minY; y <= maxY; y++ {
		rowOffset := y * fb.Width
		w0, w1, w2 := w0Row, w1Row, w2Row

		for x := minX; x <= maxX; x++ {
			inside := false
			if isPositiveArea {
				inside = (w0 >= 0 && w1 >= 0 && w2 >= 0)
			} else {
				inside = (w0 <= 0 && w1 <= 0 && w2 <= 0)
			}

			if inside {
				nw0 := float32(w0 * invArea)
				nw1 := float32(w1 * invArea)
				nw2 := float32(w2 * invArea)

				invZ := float64(nw0)*invZ0 + float64(nw1)*invZ1 + float64(nw2)*invZ2
				depth := float32(1.0 / invZ)

				idx := rowOffset + x
				if depth < fb.Depth[idx] {
					bg := fb.Color[idx]
					bgr := (bg >> 16) & 0xFF
					bgg := (bg >> 8) & 0xFF
					bgb := bg & 0xFF
					outR := uint32(float32(r)*a + float32(bgr)*(1-a))
					outG := uint32(float32(g)*a + float32(bgg)*(1-a))
					outB := uint32(float32(b)*a + float32(bgb)*(1-a))
					fb.Color[idx] = 0xFF000000 | (outR << 16) | (outG << 8) | outB
					fb.Depth[idx] = depth
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

func (fb *Framebuffer) drawBillboard(cx, cy, w, h int, depth float32, col uint32) {
	halfW := w / 2
	halfH := h / 2
	alpha := (col >> 24) & 0xFF
	if alpha == 0 {
		return
	}
	r := (col >> 16) & 0xFF
	g := (col >> 8) & 0xFF
	b := col & 0xFF

	for y := cy - halfH; y <= cy+halfH; y++ {
		if y < 0 || y >= fb.Height {
			continue
		}
		rowOffset := y * fb.Width
		for x := cx - halfW; x <= cx+halfW; x++ {
			if x < 0 || x >= fb.Width {
				continue
			}
			idx := rowOffset + x
			if depth < fb.Depth[idx] {
				if alpha == 255 {
					fb.Color[idx] = col
				} else {
					bg := fb.Color[idx]
					bgr := (bg >> 16) & 0xFF
					bgg := (bg >> 8) & 0xFF
					bgb := bg & 0xFF
					a := float32(alpha) / 255.0
					outR := uint32(float32(r)*a + float32(bgr)*(1-a))
					outG := uint32(float32(g)*a + float32(bgg)*(1-a))
					outB := uint32(float32(b)*a + float32(bgb)*(1-a))
					fb.Color[idx] = 0xFF000000 | (outR << 16) | (outG << 8) | outB
				}
			}
		}
	}
}

func (fb *Framebuffer) drawStars3D(cam *Camera, starAlpha float32, dist float64) {
	if starAlpha <= 0 {
		return
	}
	intensity := uint32(starAlpha * 255)
	if intensity > 255 {
		intensity = 255
	}
	col := 0xFF000000 | (intensity << 16) | (intensity << 8) | intensity
	depth := float32(dist)

	for _, s := range fixedStars {
		wx := cam.X + s[0]*dist
		wy := cam.Y + s[1]*dist
		wz := cam.Z + s[2]*dist
		p, ok := cam.ProjectToScreen(wx, wy, wz, fb.Width, fb.Height)
		if ok && p.Depth > 0 {
			sx := int(p.X)
			sy := int(p.Y)
			if sx >= 0 && sx < fb.Width && sy >= 0 && sy < fb.Height {
				idx := sy*fb.Width + sx
				if depth < fb.Depth[idx] {
					fb.Color[idx] = col
				}
			}
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
	if depth < 0.01 {
		depth = 0.01
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

// DrawTexturedTriangle renders a perspective depth-tested triangle textured by a RawImage with normalized UV coordinates [0.0, 1.0].
// If tint is non-zero, it blends or overrides the color (e.g. hurt red flash 0xFFFF3333).
func (fb *Framebuffer) DrawTexturedTriangle(p0, p1, p2 ScreenPoint, u0, v0, u1, v1, u2, v2 float32, img *RawImage, tint uint32) {
	if img == nil || img.Width == 0 || img.Height == 0 {
		fb.DrawTriangle(p0, p1, p2, tint)
		return
	}

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
	if minX > maxX || minY > maxY {
		return
	}

	edge := func(a, b ScreenPoint, px, py float32) float32 {
		return (px-a.X)*(b.Y-a.Y) - (py-a.Y)*(b.X-a.X)
	}

	area := edge(p0, p1, p2.X, p2.Y)
	if area == 0 {
		return
	}

	for y := minY; y <= maxY; y++ {
		rowOffset := y * fb.Width
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

				idx := rowOffset + x
				if z < fb.Depth[idx] {
					u := w0*u0 + w1*u1 + w2*u2
					v := w0*v0 + w1*v1 + w2*v2
					col := img.Sample(u, v)
					if (col >> 24) == 0 {
						continue
					}
					if tint != 0 {
						tr := (tint >> 16) & 0xFF
						tg := (tint >> 8) & 0xFF
						tb := tint & 0xFF
						cr := (col >> 16) & 0xFF
						cg := (col >> 8) & 0xFF
						cb := col & 0xFF
						outR := (cr*60 + tr*195) / 255
						outG := (cg*60 + tg*195) / 255
						outB := (cb*60 + tb*195) / 255
						col = 0xFF000000 | (outR << 16) | (outG << 8) | outB
					}
					fb.Depth[idx] = z
					fb.Color[idx] = col
				}
			}
		}
	}
}

// DrawVoxelTrianglePerspective performs mathematically exact perspective-correct texture mapping & distance fog.
func (fb *Framebuffer) DrawVoxelTrianglePerspective(v0, v1, v2 ScreenVertex) {
	fMinX := v0.X
	if v1.X < fMinX {
		fMinX = v1.X
	}
	if v2.X < fMinX {
		fMinX = v2.X
	}
	fMaxX := v0.X
	if v1.X > fMaxX {
		fMaxX = v1.X
	}
	if v2.X > fMaxX {
		fMaxX = v2.X
	}
	fMinY := v0.Y
	if v1.Y < fMinY {
		fMinY = v1.Y
	}
	if v2.Y < fMinY {
		fMinY = v2.Y
	}
	fMaxY := v0.Y
	if v1.Y > fMaxY {
		fMaxY = v1.Y
	}
	if v2.Y > fMaxY {
		fMaxY = v2.Y
	}

	minX := int(fMinX)
	maxX := int(fMaxX + 0.999999)
	minY := int(fMinY)
	maxY := int(fMaxY + 0.999999)

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

	v0x := float64(v0.X)
	v0y := float64(v0.Y)
	v1x := float64(v1.X)
	v1y := float64(v1.Y)
	v2x := float64(v2.X)
	v2y := float64(v2.Y)

	dx12 := v2x - v1x
	dy12 := v2y - v1y
	dx20 := v0x - v2x
	dy20 := v0y - v2y
	dx01 := v1x - v0x
	dy01 := v1y - v0y

	area := (v2x-v0x)*dy01 - (v2y-v0y)*dx01
	absArea := area
	if absArea < 0 {
		absArea = -absArea
	}
	if absArea < 1e-4 {
		return
	}
	invArea := 1.0 / area

	const fogStart = 20.0 // Distance fog start (in meters, tailored for 2-chunk radius horizon)
	const fogEnd = 33.0   // Distance fog end (full horizon haze)
	hCol := fb.HorizonFogColor
	if hCol == 0 {
		hCol = 0xFFC3DEFF
	}
	skyHazeR := float32((hCol >> 16) & 0xFF)
	skyHazeG := float32((hCol >> 8) & 0xFF)
	skyHazeB := float32(hCol & 0xFF)

	px0 := float64(minX) + 0.5
	py0 := float64(minY) + 0.5

	w0Row := (px0-v1x)*dy12 - (py0-v1y)*dx12
	w1Row := (px0-v2x)*dy20 - (py0-v2y)*dx20
	w2Row := (px0-v0x)*dy01 - (py0-v0y)*dx01

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
				nw0 := float32(w0 * invArea)
				nw1 := float32(w1 * invArea)
				nw2 := float32(w2 * invArea)

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
					alpha := (pixelCol >> 24) & 0xFF
					if alpha == 0 {
						// Full alpha cutout for leaves and clear glass
						goto nextPixel
					}

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
						pixelCol = (uint32(alpha) << 24) | (finalR << 16) | (finalG << 8) | finalB
					}

					if alpha == 255 {
						fb.Depth[idx] = depth
						fb.Color[idx] = pixelCol
					} else {
						// Translucent alpha blending (e.g. water surface)
						// Do not overwrite depth buffer for translucent pixels to eliminate all z-fighting and flickering
						bg := fb.Color[idx]
						bgr := (bg >> 16) & 0xFF
						bgg := (bg >> 8) & 0xFF
						bgb := bg & 0xFF
						aNorm := float32(alpha) / 255.0
						pr := (pixelCol >> 16) & 0xFF
						pg := (pixelCol >> 8) & 0xFF
						pb := pixelCol & 0xFF
						outR := uint32(float32(pr)*aNorm + float32(bgr)*(1-aNorm))
						outG := uint32(float32(pg)*aNorm + float32(bgg)*(1-aNorm))
						outB := uint32(float32(pb)*aNorm + float32(bgb)*(1-aNorm))
						fb.Color[idx] = 0xFF000000 | (outR << 16) | (outG << 8) | outB
						fb.Depth[idx] = depth
					}
				}
			}

		nextPixel:
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

// DrawPlacementGhostOutline draws a subtle cyan wireframe outline at the targeted placement location.
func (fb *Framebuffer) DrawPlacementGhostOutline(cam *Camera, bx, by, bz int32) {
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

	const ghostCol = 0xFF55FFFF // Luminous cyan wireframe
	for _, e := range edges {
		if valid[e[0]] && valid[e[1]] {
			fb.DrawLine(int(projected[e[0]].X), int(projected[e[0]].Y), int(projected[e[1]].X), int(projected[e[1]].Y), ghostCol)
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
