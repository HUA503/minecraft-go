package render

import (
	"bytes"
	"math"
	"strconv"
	"sync"
	"time"
)

var titlePanoramaStarted = time.Now()

var titlePanoramaFrameCache struct {
	sync.Mutex
	width, height int
	frame         int64
	pixels        []uint32
}

// drawTitlePanorama projects the official Java 26.3 cubemap into the title
// framebuffer. Mojang ships the six source faces through the version asset
// index; the client JAR itself only contains 1x1 placeholders.
func drawTitlePanorama(fb *Framebuffer) {
	if fb == nil || fb.Width <= 0 || fb.Height <= 0 || fb.Height > len(fb.Color)/fb.Width {
		return
	}
	frame := int64(time.Since(titlePanoramaStarted).Seconds())
	titlePanoramaFrameCache.Lock()
	if titlePanoramaFrameCache.width != fb.Width || titlePanoramaFrameCache.height != fb.Height ||
		titlePanoramaFrameCache.frame != frame || len(titlePanoramaFrameCache.pixels) != fb.Width*fb.Height {
		faces, ok := loadTitlePanoramaFaces()
		if ok {
			// Java's CubeMap starts with a 180 degree X rotation, adds a fixed 10
			// degree X tilt, then slowly spins around Y at 0.1 degrees per tick.
			titlePanoramaFrameCache.pixels = projectTitlePanorama(fb.Width, fb.Height, faces, float64(frame)*2)
		} else {
			titlePanoramaFrameCache.pixels = titlePanoramaFallback(fb.Width, fb.Height)
		}
		titlePanoramaFrameCache.width = fb.Width
		titlePanoramaFrameCache.height = fb.Height
		titlePanoramaFrameCache.frame = frame
	}
	copy(fb.Color[:fb.Width*fb.Height], titlePanoramaFrameCache.pixels)
	titlePanoramaFrameCache.Unlock()
}

func loadTitlePanoramaFaces() ([6]*RawImage, bool) {
	var faces [6]*RawImage
	for i := range faces {
		path := "assets/textures/gui/title/background/panorama_" + strconv.Itoa(i) + ".png"
		data, err := EmbeddedTextures.ReadFile(path)
		if err != nil {
			return faces, false
		}
		image, err := LoadRawImage(bytes.NewReader(data))
		if err != nil || image == nil || image.Width < 2 || image.Height < 2 || image.Height > len(image.Pixels)/image.Width {
			return faces, false
		}
		faces[i] = image
	}
	return faces, true
}

func projectTitlePanorama(width, height int, faces [6]*RawImage, spinDegrees float64) []uint32 {
	pixels := make([]uint32, width*height)
	aspect := float64(width) / float64(height)
	tanHalfFOV := math.Tan(85 * math.Pi / 360) // Match CubeMap's 85 degree perspective FOV.
	rotationX := -190 * math.Pi / 180          // Inverse of vanilla's Rx(180°) followed by Rx(10°).
	rotationY := math.Mod(spinDegrees, 360) * math.Pi / 180
	sinX, cosX := math.Sin(rotationX), math.Cos(rotationX)
	sinY, cosY := math.Sin(rotationY), math.Cos(rotationY)
	for y := 0; y < height; y++ {
		planeY := (1 - 2*(float64(y)+0.5)/float64(height)) * tanHalfFOV
		for x := 0; x < width; x++ {
			planeX := (2*(float64(x)+0.5)/float64(width) - 1) * tanHalfFOV * aspect
			// The cube shader emits the original vertex position as its cubemap
			// direction, so screen rays must be transformed by the inverse model
			// rotation to match CubeMap.render exactly.
			dirX := cosY*planeX + sinY*(sinX*planeY-cosX)
			dirY := cosX*planeY + sinX
			dirZ := -sinY*planeX + cosY*(sinX*planeY-cosX)
			pixels[y*width+x] = sampleTitleCubemap(faces, dirX, dirY, dirZ) | 0xFF000000
		}
	}
	return pixels
}

func sampleTitleCubemap(faces [6]*RawImage, x, y, z float64) uint32 {
	ax, ay, az := math.Abs(x), math.Abs(y), math.Abs(z)
	face := 0
	var sc, tc, major float64
	switch {
	case ax >= ay && ax >= az && x >= 0:
		face, major, sc, tc = 0, ax, -z, -y
	case ax >= ay && ax >= az:
		face, major, sc, tc = 1, ax, z, -y
	case ay >= ax && ay >= az && y >= 0:
		face, major, sc, tc = 2, ay, x, z
	case ay >= ax && ay >= az:
		face, major, sc, tc = 3, ay, x, -z
	case z >= 0:
		face, major, sc, tc = 4, az, x, -y
	default:
		face, major, sc, tc = 5, az, -x, -y
	}
	if major == 0 || faces[face] == nil {
		return 0xFF000000
	}
	u := (sc/major + 1) * 0.5
	// The source PNGs store image rows top-to-bottom while cubemap tc follows
	// the GL convention; reversing the projected tc here keeps sky above ground.
	v := (tc/major + 1) * 0.5
	image := faces[face]
	sx := min(image.Width-1, max(0, int(u*float64(image.Width))))
	sy := min(image.Height-1, max(0, int(v*float64(image.Height))))
	return image.Pixels[sy*image.Width+sx]
}

func titlePanoramaFallback(width, height int) []uint32 {
	pixels := make([]uint32, width*height)
	for y := 0; y < height; y++ {
		shade := uint32(18 + (height-y)*8/max(1, height))
		color := 0xFF000000 | (shade << 16) | (shade << 8) | shade
		for x := 0; x < width; x++ {
			pixels[y*width+x] = color
		}
	}
	return pixels
}
