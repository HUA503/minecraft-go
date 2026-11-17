package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	ErrVarIntTooBig  = errors.New("varint is too big (exceeds 5 bytes)")
	ErrVarLongTooBig = errors.New("varlong is too big (exceeds 10 bytes)")
	ErrStringTooLong = errors.New("string length exceeds maximum allowed")
)

// WriteVarInt serializes an int32 as a 7-bit LEB128 VarInt.
func WriteVarInt(w io.Writer, val int32) (int, error) {
	uval := uint32(val)
	buf := make([]byte, 0, 5)
	for {
		b := byte(uval & 0x7F)
		uval >>= 7
		if uval != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if uval == 0 {
			break
		}
	}
	return w.Write(buf)
}

// ReadVarInt deserializes a 7-bit LEB128 VarInt into an int32.
func ReadVarInt(r io.Reader) (int32, int, error) {
	var val uint32
	var count int
	buf := make([]byte, 1)

	for {
		if count >= 5 {
			return 0, count, ErrVarIntTooBig
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, count, err
		}
		b := buf[0]
		val |= uint32(b&0x7F) << (7 * count)
		count++
		if (b & 0x80) == 0 {
			break
		}
	}
	return int32(val), count, nil
}

// WriteVarLong serializes an int64 as a 7-bit LEB128 VarLong.
func WriteVarLong(w io.Writer, val int64) (int, error) {
	uval := uint64(val)
	buf := make([]byte, 0, 10)
	for {
		b := byte(uval & 0x7F)
		uval >>= 7
		if uval != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if uval == 0 {
			break
		}
	}
	return w.Write(buf)
}

// ReadVarLong deserializes a 7-bit LEB128 VarLong into an int64.
func ReadVarLong(r io.Reader) (int64, int, error) {
	var val uint64
	var count int
	buf := make([]byte, 1)

	for {
		if count >= 10 {
			return 0, count, ErrVarLongTooBig
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, count, err
		}
		b := buf[0]
		val |= uint64(b&0x7F) << (7 * count)
		count++
		if (b & 0x80) == 0 {
			break
		}
	}
	return int64(val), count, nil
}

// WriteString encodes a string prefixed with its length as a VarInt.
func WriteString(w io.Writer, s string) error {
	data := []byte(s)
	if _, err := WriteVarInt(w, int32(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadString decodes a string prefixed with its length as a VarInt.
func ReadString(r io.Reader, maxLen int) (string, error) {
	strLen, _, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}
	if strLen < 0 || (maxLen > 0 && int(strLen) > maxLen) {
		return "", fmt.Errorf("%w: length %d (max %d)", ErrStringTooLong, strLen, maxLen)
	}
	buf := make([]byte, strLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// WritePosition encodes a block position (x: 26 bits, z: 26 bits, y: 12 bits) as an int64.
func WritePosition(w io.Writer, x, y, z int32) error {
	encoded := (uint64(x&0x3FFFFFF) << 38) | (uint64(z&0x3FFFFFF) << 12) | uint64(y&0xFFF)
	return binary.Write(w, binary.BigEndian, encoded)
}

// ReadPosition decodes a block position from an int64.
func ReadPosition(r io.Reader) (x, y, z int32, err error) {
	var encoded uint64
	if err = binary.Read(r, binary.BigEndian, &encoded); err != nil {
		return 0, 0, 0, err
	}
	rawX := int32(encoded >> 38)
	rawZ := int32((encoded >> 12) & 0x3FFFFFF)
	rawY := int32(encoded & 0xFFF)

	// Sign extend 26-bit coordinates
	if rawX >= 1<<25 {
		rawX -= 1 << 26
	}
	if rawZ >= 1<<25 {
		rawZ -= 1 << 26
	}
	// Sign extend 12-bit Y
	if rawY >= 1<<11 {
		rawY -= 1 << 12
	}
	return rawX, rawY, rawZ, nil
}
