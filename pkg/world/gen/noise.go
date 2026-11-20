package gen

import (
	"math"
)

// PerlinNoise provides 2D and 3D gradient noise.
type PerlinNoise struct {
	perm [512]int
}

func NewPerlinNoise(seed int64) *PerlinNoise {
	p := &PerlinNoise{}
	permutation := make([]int, 256)
	for i := 0; i < 256; i++ {
		permutation[i] = i
	}

	// Pseudo-random shuffle with seed
	s := seed
	for i := 255; i > 0; i-- {
		s = (s*6364136223846793005 + 1442695040888963407)
		j := int((s >> 32) & 0x7FFFFFFF) % (i + 1)
		permutation[i], permutation[j] = permutation[j], permutation[i]
	}

	for i := 0; i < 512; i++ {
		p.perm[i] = permutation[i&255]
	}
	return p
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

func grad2D(hash int, x, y float64) float64 {
	h := hash & 7
	u := x
	v := y
	if h >= 4 {
		u = y
		v = x
	}
	if (h & 1) != 0 {
		u = -u
	}
	if (h & 2) != 0 {
		v = -v
	}
	return u + v
}

func grad3D(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h >= 8 {
		u = y
	}
	v := y
	if h >= 4 {
		if h == 12 || h == 14 {
			v = x
		} else {
			v = z
		}
	}
	res := 0.0
	if (h & 1) == 0 {
		res += u
	} else {
		res -= u
	}
	if (h & 2) == 0 {
		res += v
	} else {
		res -= v
	}
	return res
}

// Noise2D returns perlin noise value in range [-1.0, 1.0].
func (p *PerlinNoise) Noise2D(x, y float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255

	x -= math.Floor(x)
	y -= math.Floor(y)

	u := fade(x)
	v := fade(y)

	A := p.perm[X] + Y
	B := p.perm[X+1] + Y

	return lerp(v,
		lerp(u, grad2D(p.perm[A], x, y), grad2D(p.perm[B], x-1, y)),
		lerp(u, grad2D(p.perm[A+1], x, y-1), grad2D(p.perm[B+1], x-1, y-1)),
	)
}

// Noise3D returns 3D perlin noise value in range [-1.0, 1.0].
func (p *PerlinNoise) Noise3D(x, y, z float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	Z := int(math.Floor(z)) & 255

	x -= math.Floor(x)
	y -= math.Floor(y)
	z -= math.Floor(z)

	u := fade(x)
	v := fade(y)
	w := fade(z)

	A := p.perm[X] + Y
	AA := p.perm[A] + Z
	AB := p.perm[A+1] + Z
	B := p.perm[X+1] + Y
	BA := p.perm[B] + Z
	BB := p.perm[B+1] + Z

	return lerp(w,
		lerp(v,
			lerp(u, grad3D(p.perm[AA], x, y, z), grad3D(p.perm[BA], x-1, y, z)),
			lerp(u, grad3D(p.perm[AB], x, y-1, z), grad3D(p.perm[BB], x-1, y-1, z))),
		lerp(v,
			lerp(u, grad3D(p.perm[AA+1], x, y, z-1), grad3D(p.perm[BA+1], x-1, y, z-1)),
			lerp(u, grad3D(p.perm[AB+1], x, y-1, z-1), grad3D(p.perm[BB+1], x-1, y-1, z-1))),
	)
}

// FBM2D computes fractional Brownian motion (octaves of noise).
func (p *PerlinNoise) FBM2D(x, y float64, octaves int, persistence, lacunarity float64) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0
	maxValue := 0.0

	for i := 0; i < octaves; i++ {
		total += p.Noise2D(x*frequency, y*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}

	return total / maxValue
}

// FBM3D computes 3D fractional Brownian motion for volumetric noise.
func (p *PerlinNoise) FBM3D(x, y, z float64, octaves int, persistence, lacunarity float64) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0
	maxValue := 0.0

	for i := 0; i < octaves; i++ {
		total += p.Noise3D(x*frequency, y*frequency, z*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}

	return total / maxValue
}
