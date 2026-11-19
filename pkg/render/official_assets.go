package render

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

//go:embed assets/textures/item/*.png assets/textures/block/*.png
var EmbeddedTextures embed.FS

var (
	officialSpriteCache   = make(map[string][16][16]uint32)
	officialSpriteCacheMu sync.RWMutex
)

// LoadPNG16x16 decodes a 16x16 PNG image from an io.Reader into a [16][16]uint32 RGBA pixel buffer.
func LoadPNG16x16(r io.Reader) ([16][16]uint32, error) {
	var s [16][16]uint32
	img, _, err := image.Decode(r)
	if err != nil {
		return s, err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			srcX := bounds.Min.X + (x*w)/16
			srcY := bounds.Min.Y + (y*h)/16
			r, g, b, a := img.At(srcX, srcY).RGBA()
			if a == 0 {
				s[y][x] = 0
			} else {
				a8 := uint32(a >> 8)
				r8 := uint32((r * 255) / a)
				g8 := uint32((g * 255) / a)
				b8 := uint32((b * 255) / a)
				s[y][x] = (a8 << 24) | (r8 << 16) | (g8 << 8) | b8
			}
		}
	}
	return s, nil
}

// ReadOfficialTextureBytes attempts to read texture bytes from external assets/ directory first,
// falling back to embedded official vanilla assets.
func ReadOfficialTextureBytes(category, name string) ([]byte, error) {
	if !strings.HasSuffix(name, ".png") {
		name += ".png"
	}

	// 1. Check external directories
	candidates := []string{
		filepath.Join("assets", "minecraft", "textures", category, name),
		filepath.Join("..", "assets", "minecraft", "textures", category, name),
		filepath.Join("..", "..", "assets", "minecraft", "textures", category, name),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	// 2. Fallback to embedded textures
	embedPath := "assets/textures/" + category + "/" + name
	return EmbeddedTextures.ReadFile(embedPath)
}

// GetOfficialTexture loads and caches a 16x16 texture from category ("item" or "block").
func GetOfficialTexture(category, name string) ([16][16]uint32, bool) {
	cacheKey := category + ":" + name
	officialSpriteCacheMu.RLock()
	if cached, ok := officialSpriteCache[cacheKey]; ok {
		officialSpriteCacheMu.RUnlock()
		return cached, true
	}
	officialSpriteCacheMu.RUnlock()

	data, err := ReadOfficialTextureBytes(category, name)
	if err != nil {
		return [16][16]uint32{}, false
	}

	tex, err := LoadPNG16x16(bytes.NewReader(data))
	if err != nil {
		return [16][16]uint32{}, false
	}

	officialSpriteCacheMu.Lock()
	officialSpriteCache[cacheKey] = tex
	officialSpriteCacheMu.Unlock()
	return tex, true
}

// GetOfficialItemSprite retrieves the 100% authentic official vanilla item texture for an item ID.
func GetOfficialItemSprite(itemID int32) ([16][16]uint32, bool) {
	texName := GetOfficialItemTextureName(itemID)
	if texName != "" {
		if tex, ok := GetOfficialTexture("item", texName); ok {
			return tex, true
		}
		// Also check block category (e.g. torch, flowers, saplings)
		if tex, ok := GetOfficialTexture("block", texName); ok {
			return tex, true
		}
	}
	return [16][16]uint32{}, false
}

// GetOfficialItemTextureName maps item IDs to official Mojang vanilla texture filenames.
func GetOfficialItemTextureName(itemID int32) string {
	switch itemID {
	// --- Tools ---
	// Pickaxes
	case 270:
		return "wooden_pickaxe"
	case 274:
		return "stone_pickaxe"
	case 257:
		return "iron_pickaxe"
	case 278:
		return "diamond_pickaxe"
	case 285:
		return "golden_pickaxe"
	// Swords
	case 268:
		return "wooden_sword"
	case 272:
		return "stone_sword"
	case 267:
		return "iron_sword"
	case 276:
		return "diamond_sword"
	case 283:
		return "golden_sword"
	// Axes
	case 271:
		return "wooden_axe"
	case 275:
		return "stone_axe"
	case 258:
		return "iron_axe"
	case 279:
		return "diamond_axe"
	case 286:
		return "golden_axe"
	// Shovels
	case 269:
		return "wooden_shovel"
	case 273:
		return "stone_shovel"
	case 256:
		return "iron_shovel"
	case 277:
		return "diamond_shovel"
	case 284:
		return "golden_shovel"
	// Hoes
	case 290:
		return "wooden_hoe"
	case 291:
		return "stone_hoe"
	case 292:
		return "iron_hoe"
	case 293:
		return "diamond_hoe"
	case 294:
		return "golden_hoe"

	// --- Food Items ---
	case inventory.ItemApple: // 260
		return "apple"
	case inventory.ItemBread: // 297
		return "bread"
	case inventory.ItemRawPorkchop: // 319
		return "porkchop"
	case inventory.ItemCookedPorkchop: // 320
		return "cooked_porkchop"
	case inventory.ItemRawBeef: // 363
		return "beef"
	case inventory.ItemSteak: // 364
		return "cooked_beef"
	case inventory.ItemRottenFlesh: // 367
		return "rotten_flesh"
	case 365:
		return "chicken"
	case 366:
		return "cooked_chicken"
	case 423:
		return "mutton"
	case 424:
		return "cooked_mutton"

	// --- Minerals & Crafting ---
	case inventory.ItemDiamond: // 264
		return "diamond"
	case inventory.ItemCoal: // 263
		return "coal"
	case inventory.ItemIronIngot: // 265
		return "iron_ingot"
	case 266:
		return "gold_ingot"
	case inventory.ItemStick: // 280
		return "stick"
	case inventory.ItemTorch: // 50
		return "torch"
	case 331:
		return "redstone"
	case 352:
		return "bone"
	case 288:
		return "feather"
	case 289:
		return "gunpowder"
	case 334:
		return "leather"
	case 287:
		return "string"
	case 261:
		return "bow"
	case 262:
		return "arrow"

	// --- Armor Sets ---
	// Leather
	case 298:
		return "leather_helmet"
	case 299:
		return "leather_chestplate"
	case 300:
		return "leather_leggings"
	case 301:
		return "leather_boots"
	// Iron
	case 306:
		return "iron_helmet"
	case 307:
		return "iron_chestplate"
	case 308:
		return "iron_leggings"
	case 309:
		return "iron_boots"
	// Diamond
	case 310:
		return "diamond_helmet"
	case 311:
		return "diamond_chestplate"
	case 312:
		return "diamond_leggings"
	case 313:
		return "diamond_boots"
	// Gold
	case 314:
		return "golden_helmet"
	case 315:
		return "golden_chestplate"
	case 316:
		return "golden_leggings"
	case 317:
		return "golden_boots"

	// --- Other Essentials ---
	case 325:
		return "bucket"
	case 326:
		return "water_bucket"
	case 327:
		return "lava_bucket"
	case 332:
		return "snowball"
	case 344:
		return "egg"
	case 368:
		return "ender_pearl"
	case 359:
		return "shears"

	// Flat foliage / flora
	case 37:
		return "dandelion"
	case 38:
		return "poppy"
	case 6:
		return "oak_sapling"
	}
	return ""
}
