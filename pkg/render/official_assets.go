package render

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

const (
	maxTextureEncodedBytes = 16 << 20
	maxTexturePixels       = 4 * 1024 * 1024
)

//go:embed assets/textures/item/*.png assets/textures/block/*.png assets/textures/font/*.png assets/textures/misc/*.png assets/textures/mob_effect/*.png assets/textures/gui/container/*.png assets/textures/gui/container/creative_inventory/*.png assets/textures/gui/title/*.png assets/textures/gui/title/background/*.png assets/textures/gui/sprites/widget/*.png assets/textures/gui/sprites/widget/*.png.mcmeta assets/textures/gui/sprites/container/slot.png assets/textures/gui/sprites/container/inventory/*.png assets/textures/gui/sprites/container/inventory/*.png.mcmeta assets/textures/gui/sprites/container/furnace/*.png assets/textures/gui/sprites/container/creative_inventory/*.png assets/textures/gui/sprites/tooltip/*.png assets/textures/gui/sprites/tooltip/*.png.mcmeta assets/textures/gui/sprites/hud/*.png assets/textures/gui/sprites/hud/heart/*.png assets/textures/entity/*/*.png assets/textures/entity/*/*/*.png
var EmbeddedTextures embed.FS

var (
	officialSpriteCache   = make(map[string][16][16]uint32)
	officialSpriteCacheMu sync.RWMutex

	officialRawCache   = make(map[string]*RawImage)
	officialRawCacheMu sync.RWMutex
)

// RawImage represents an authentic decoded RGBA texture.
type RawImage struct {
	Width  int
	Height int
	Pixels []uint32 // 0xAARRGGBB in row-major order
}

// Sample returns the pixel color at normalized UV coordinates [0.0, 1.0].
func (img *RawImage) Sample(u, v float32) uint32 {
	if img == nil || img.Width <= 0 || img.Height <= 0 || img.Height > len(img.Pixels)/img.Width {
		return 0
	}
	if u != u || v != v { // NaN UVs can otherwise convert to an invalid pixel index.
		return 0
	}
	if u < 0 {
		u = 0
	} else if u > 1.0 {
		u = 1.0
	}
	if v < 0 {
		v = 0
	} else if v > 1.0 {
		v = 1.0
	}
	x := int(u * float32(img.Width-1))
	y := int(v * float32(img.Height-1))
	if x >= img.Width {
		x = img.Width - 1
	}
	if y >= img.Height {
		y = img.Height - 1
	}
	return img.Pixels[y*img.Width+x]
}

// SampleUVRect returns pixel color within a sub-rectangle of the texture [u0..u1, v0..v1].
func (img *RawImage) SampleUVRect(u0, v0, u1, v1 float32, subU, subV float32) uint32 {
	u := u0 + (u1-u0)*subU
	v := v0 + (v1-v0)*subV
	return img.Sample(u, v)
}

func decodeBoundedImage(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxTextureEncodedBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTextureEncodedBytes {
		return nil, fmt.Errorf("texture file exceeds %d bytes", maxTextureEncodedBytes)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxTexturePixels/config.Height {
		return nil, fmt.Errorf("texture dimensions %dx%d exceed supported limit", config.Width, config.Height)
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := decoded.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 || bounds.Dx() > maxTexturePixels/bounds.Dy() {
		return nil, fmt.Errorf("decoded texture dimensions %dx%d exceed supported limit", bounds.Dx(), bounds.Dy())
	}
	return decoded, nil
}

func readTextureFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil {
		return nil, err
	} else if info.Size() > maxTextureEncodedBytes {
		return nil, fmt.Errorf("texture file exceeds %d bytes", maxTextureEncodedBytes)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxTextureEncodedBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTextureEncodedBytes {
		return nil, fmt.Errorf("texture file exceeds %d bytes", maxTextureEncodedBytes)
	}
	return data, nil
}

// LoadRawImage decodes any PNG image into a RawImage.
func LoadRawImage(r io.Reader) (*RawImage, error) {
	img, err := decodeBoundedImage(r)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	raw := &RawImage{
		Width:  w,
		Height: h,
		Pixels: make([]uint32, w*h),
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a == 0 {
				raw.Pixels[y*w+x] = 0
			} else {
				a8 := uint32(a >> 8)
				r8 := uint32((r * 255) / a)
				g8 := uint32((g * 255) / a)
				b8 := uint32((b * 255) / a)
				raw.Pixels[y*w+x] = (a8 << 24) | (r8 << 16) | (g8 << 8) | b8
			}
		}
	}
	return raw, nil
}

// ReadOfficialAssetBytes reads bytes from file or embedded FS for any relative path.
func ReadOfficialAssetBytes(relPath string) ([]byte, error) {
	data, err := readSingleAssetCandidate(relPath)
	if err == nil && len(data) > 0 {
		return data, nil
	}

	// Try biome temperate/alternative variants if standard name was queried
	slashPath := filepath.ToSlash(relPath)
	if strings.Contains(slashPath, "entity/pig/pig.png") {
		alt := strings.Replace(slashPath, "entity/pig/pig.png", "entity/pig/pig_temperate.png", 1)
		if d, e := readSingleAssetCandidate(alt); e == nil && len(d) > 0 {
			return d, nil
		}
	} else if strings.Contains(slashPath, "entity/cow/cow.png") {
		alt := strings.Replace(slashPath, "entity/cow/cow.png", "entity/cow/cow_temperate.png", 1)
		if d, e := readSingleAssetCandidate(alt); e == nil && len(d) > 0 {
			return d, nil
		}
	} else if strings.Contains(slashPath, "entity/chicken/chicken.png") {
		alt := strings.Replace(slashPath, "entity/chicken/chicken.png", "entity/chicken/chicken_temperate.png", 1)
		if d, e := readSingleAssetCandidate(alt); e == nil && len(d) > 0 {
			return d, nil
		}
	}

	return nil, err
}

func readSingleAssetCandidate(relPath string) ([]byte, error) {
	if !strings.HasSuffix(relPath, ".png") {
		relPath += ".png"
	}
	normPath := filepath.Clean(relPath)
	candidates := []string{
		filepath.Join("assets", "minecraft", "textures", normPath),
		filepath.Join("..", "assets", "minecraft", "textures", normPath),
		filepath.Join("..", "..", "assets", "minecraft", "textures", normPath),
		filepath.Join("pkg", "render", "assets", "textures", normPath),
		filepath.Join("..", "pkg", "render", "assets", "textures", normPath),
		filepath.Join("..", "..", "pkg", "render", "assets", "textures", normPath),
	}
	for _, p := range candidates {
		if data, err := readTextureFile(p); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	// Fallback to embedded textures
	embedPath := "assets/textures/" + filepath.ToSlash(normPath)
	return EmbeddedTextures.ReadFile(embedPath)
}

// GetOfficialImage loads and caches any authentic PNG image from embedded or disk assets.
func GetOfficialImage(relPath string) (*RawImage, bool) {
	relPath = filepath.ToSlash(relPath)
	officialRawCacheMu.RLock()
	if cached, ok := officialRawCache[relPath]; ok {
		officialRawCacheMu.RUnlock()
		return cached, true
	}
	officialRawCacheMu.RUnlock()

	data, err := ReadOfficialAssetBytes(relPath)
	if err != nil {
		return nil, false
	}
	raw, err := LoadRawImage(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	officialRawCacheMu.Lock()
	officialRawCache[relPath] = raw
	officialRawCacheMu.Unlock()
	return raw, true
}

// LoadPNG16x16 decodes a 16x16 PNG image from an io.Reader into a [16][16]uint32 RGBA pixel buffer.
func LoadPNG16x16(r io.Reader) ([16][16]uint32, error) {
	var s [16][16]uint32
	img, err := decodeBoundedImage(r)
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
		if data, err := readTextureFile(p); err == nil && len(data) > 0 {
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
	// Pressure plates are thin block models; their vanilla item appearance is
	// the corresponding block texture mapped onto the official plate geometry.
	if inventory.IsPressurePlateItem(itemID) {
		return RenderIsometricBlock16x16(itemID), true
	}
	// Panes have no standalone item PNG in the vanilla atlas: the client derives
	// their icon from the pane block model and its official pane texture.
	if inventory.IsPaneBlock(itemID) {
		return RenderIsometricBlock16x16(itemID), true
	}
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
	if item, ok := inventory.ModernCreativeItemByID(itemID); ok {
		return item.Texture
	}
	if item, ok := inventory.FoodItemByID(itemID); ok {
		return item.Texture
	}
	if texture := inventory.ExtendedBlockItemTextureName(itemID); texture != "" {
		return texture
	}
	if texture := inventory.SpawnEggTextureName(itemID); texture != "" {
		return texture
	}
	switch itemID {
	case inventory.ItemFlintAndSteel:
		return "flint_and_steel"
	case 302:
		return "chainmail_helmet"
	case 303:
		return "chainmail_chestplate"
	case 304:
		return "chainmail_leggings"
	case 305:
		return "chainmail_boots"
	case 323:
		return "oak_sign"
	case 345:
		return "compass_00"
	case 346:
		return "fishing_rod"
	case 262:
		return "arrow"
	case 295:
		return "wheat_seeds"
	case 296:
		return "wheat"
	case 371:
		return "gold_nugget"
	case inventory.ItemBoneMeal:
		return "bone_meal"
	case inventory.ItemRawCopper:
		return "raw_copper"
	case inventory.ItemRawIron:
		return "raw_iron"
	case inventory.ItemRawGold:
		return "raw_gold"
	case inventory.ItemEmerald:
		return "emerald"
	case inventory.ItemLapisLazuli:
		return "lapis_lazuli"
	case inventory.ItemCopperIngot:
		return "copper_ingot"
	case inventory.ItemNetheriteScrap:
		return "netherite_scrap"
	case 375:
		return "spider_eye"
	case 381:
		return "ender_eye"
	case 383:
		return "slime_spawn_egg"
	case 406:
		return "quartz"
	case 328:
		return "minecart"
	case 333:
		return "oak_boat"
	case 342:
		return "chest_minecart"
	case 343:
		return "furnace_minecart"
	case 19:
		return "sponge"
	case 44:
		return "stone"
	case 52:
		return "spawner"
	case 153:
		return "dropper_front"
	case 154:
		return "hopper"
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
	case inventory.ItemCopperHelmet:
		return "copper_helmet"
	case inventory.ItemCopperChestplate:
		return "copper_chestplate"
	case inventory.ItemCopperLeggings:
		return "copper_leggings"
	case inventory.ItemCopperBoots:
		return "copper_boots"
	case inventory.ItemCopperSword:
		return "copper_sword"
	case inventory.ItemCopperShovel:
		return "copper_shovel"
	case inventory.ItemCopperPickaxe:
		return "copper_pickaxe"
	case inventory.ItemCopperAxe:
		return "copper_axe"
	case inventory.ItemCopperHoe:
		return "copper_hoe"
	case inventory.ItemNetheriteIngot:
		return "netherite_ingot"
	case inventory.ItemNetheriteHelmet:
		return "netherite_helmet"
	case inventory.ItemNetheriteChestplate:
		return "netherite_chestplate"
	case inventory.ItemNetheriteLeggings:
		return "netherite_leggings"
	case inventory.ItemNetheriteBoots:
		return "netherite_boots"
	case inventory.ItemNetheriteSword:
		return "netherite_sword"
	case inventory.ItemNetheriteAxe:
		return "netherite_axe"
	case inventory.ItemNetheritePickaxe:
		return "netherite_pickaxe"
	case inventory.ItemNetheriteShovel:
		return "netherite_shovel"
	case inventory.ItemNetheriteHoe:
		return "netherite_hoe"

	// --- Other Essentials ---
	case 322:
		return "golden_apple"
	case 325:
		return "bucket"
	case 326:
		return "water_bucket"
	case 327:
		return "lava_bucket"
	case 318:
		return "flint"
	case 332:
		return "snowball"
	case 336:
		return "brick"
	case 337:
		return "clay_ball"
	case 340:
		return "book"
	case 344:
		return "egg"
	case 368:
		return "ender_pearl"
	case 359:
		return "shears"
	case 360:
		return "melon_slice"
	case 324:
		return "oak_door"
	case 330:
		return "iron_door"
	case 83:
		return "sugar_cane"
	case 86:
		return "pumpkin"
	case 103:
		return "melon_slice"

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
