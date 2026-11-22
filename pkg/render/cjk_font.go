package render

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"io"
	"sync"
)

//go:embed cjk_font.bin.gz
var cjkFontGz []byte

var (
	cjkGlyphs12x12 map[rune][12]uint16
	cjkOnce        sync.Once
)

// Clean fallback glyph: 12x12 missing character box with a centered question mark [?]
var fallbackGlyph12x12 = [12]uint16{
	0b111111111111,
	0b100000000001,
	0b100111100001,
	0b101000010001,
	0b100000100001,
	0b100001000001,
	0b100001000001,
	0b100000000001,
	0b100001000001,
	0b100000000001,
	0b100000000001,
	0b111111111111,
}

func initCJKFont() {
	cjkGlyphs12x12 = make(map[rune][12]uint16, 7500)
	if len(cjkFontGz) == 0 {
		return
	}

	zr, err := gzip.NewReader(bytes.NewReader(cjkFontGz))
	if err != nil {
		return
	}
	defer zr.Close()

	buf := make([]byte, 28) // uint32 (4 bytes) + 12 * uint16 (24 bytes)
	for {
		_, err := io.ReadFull(zr, buf)
		if err != nil {
			break
		}
		r := rune(binary.LittleEndian.Uint32(buf[0:4]))
		var glyph [12]uint16
		for i := 0; i < 12; i++ {
			glyph[i] = binary.LittleEndian.Uint16(buf[4+i*2 : 6+i*2])
		}
		cjkGlyphs12x12[r] = glyph
	}
}

// GetCJKGlyph returns the authentic 12x12 bitmap for a rune, or a clean [?] missing-glyph box.
func GetCJKGlyph(ch rune) [12]uint16 {
	cjkOnce.Do(initCJKFont)
	if g, ok := cjkGlyphs12x12[ch]; ok {
		return g
	}
	return fallbackGlyph12x12
}
