package nbt

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// TagType defines the NBT tag type identifier.
type TagType byte

const (
	TagEnd TagType = iota
	TagByte
	TagShort
	TagInt
	TagLong
	TagFloat
	TagDouble
	TagByteArray
	TagString
	TagList
	TagCompound
	TagIntArray
	TagLongArray
)

func (t TagType) String() string {
	switch t {
	case TagEnd:
		return "TAG_End"
	case TagByte:
		return "TAG_Byte"
	case TagShort:
		return "TAG_Short"
	case TagInt:
		return "TAG_Int"
	case TagLong:
		return "TAG_Long"
	case TagFloat:
		return "TAG_Float"
	case TagDouble:
		return "TAG_Double"
	case TagByteArray:
		return "TAG_ByteArray"
	case TagString:
		return "TAG_String"
	case TagList:
		return "TAG_List"
	case TagCompound:
		return "TAG_Compound"
	case TagIntArray:
		return "TAG_IntArray"
	case TagLongArray:
		return "TAG_LongArray"
	default:
		return fmt.Sprintf("TAG_Unknown(%d)", byte(t))
	}
}

// Tag represents a generic NBT node.
type Tag interface {
	Type() TagType
}

type ByteTag struct{ Value int8 }
type ShortTag struct{ Value int16 }
type IntTag struct{ Value int32 }
type LongTag struct{ Value int64 }
type FloatTag struct{ Value float32 }
type DoubleTag struct{ Value float64 }
type ByteArrayTag struct{ Value []byte }
type StringTag struct{ Value string }
type ListTag struct {
	ElementType TagType
	Elements    []Tag
}
type CompoundTag struct {
	Value map[string]Tag
}
type IntArrayTag struct{ Value []int32 }
type LongArrayTag struct{ Value []int64 }

func (t *ByteTag) Type() TagType      { return TagByte }
func (t *ShortTag) Type() TagType     { return TagShort }
func (t *IntTag) Type() TagType       { return TagInt }
func (t *LongTag) Type() TagType      { return TagLong }
func (t *FloatTag) Type() TagType     { return TagFloat }
func (t *DoubleTag) Type() TagType    { return TagDouble }
func (t *ByteArrayTag) Type() TagType { return TagByteArray }
func (t *StringTag) Type() TagType    { return TagString }
func (t *ListTag) Type() TagType      { return TagList }
func (t *CompoundTag) Type() TagType  { return TagCompound }
func (t *IntArrayTag) Type() TagType  { return TagIntArray }
func (t *LongArrayTag) Type() TagType { return TagLongArray }

func NewCompound() *CompoundTag {
	return &CompoundTag{Value: make(map[string]Tag)}
}

func (c *CompoundTag) Get(key string) Tag {
	if c == nil || c.Value == nil {
		return nil
	}
	return c.Value[key]
}

func (c *CompoundTag) GetByte(key string) int8 {
	if t, ok := c.Get(key).(*ByteTag); ok {
		return t.Value
	}
	return 0
}

func (c *CompoundTag) GetShort(key string) int16 {
	if t, ok := c.Get(key).(*ShortTag); ok {
		return t.Value
	}
	return 0
}

func (c *CompoundTag) GetString(key string) string {
	if t, ok := c.Get(key).(*StringTag); ok {
		return t.Value
	}
	return ""
}

func (c *CompoundTag) GetInt(key string) int32 {
	if t, ok := c.Get(key).(*IntTag); ok {
		return t.Value
	}
	return 0
}

func (c *CompoundTag) GetLong(key string) int64 {
	if t, ok := c.Get(key).(*LongTag); ok {
		return t.Value
	}
	return 0
}

func (c *CompoundTag) Put(key string, tag Tag) {
	if c.Value == nil {
		c.Value = make(map[string]Tag)
	}
	c.Value[key] = tag
}

// Encoder writes NBT structures to an io.Writer.
type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) EncodeNamed(name string, root *CompoundTag) error {
	if root == nil {
		return errors.New("cannot encode nil compound")
	}
	if err := binary.Write(e.w, binary.BigEndian, byte(TagCompound)); err != nil {
		return err
	}
	if err := e.writeString(name); err != nil {
		return err
	}
	return e.writeCompoundPayload(root)
}

func (e *Encoder) writeString(s string) error {
	data := []byte(s)
	if len(data) > 65535 {
		return errors.New("nbt string too long")
	}
	if err := binary.Write(e.w, binary.BigEndian, uint16(len(data))); err != nil {
		return err
	}
	_, err := e.w.Write(data)
	return err
}

func (e *Encoder) writePayload(t Tag) error {
	switch v := t.(type) {
	case *ByteTag:
		return binary.Write(e.w, binary.BigEndian, v.Value)
	case *ShortTag:
		return binary.Write(e.w, binary.BigEndian, v.Value)
	case *IntTag:
		return binary.Write(e.w, binary.BigEndian, v.Value)
	case *LongTag:
		return binary.Write(e.w, binary.BigEndian, v.Value)
	case *FloatTag:
		return binary.Write(e.w, binary.BigEndian, v.Value)
	case *DoubleTag:
		return binary.Write(e.w, binary.BigEndian, v.Value)
	case *ByteArrayTag:
		if err := binary.Write(e.w, binary.BigEndian, int32(len(v.Value))); err != nil {
			return err
		}
		_, err := e.w.Write(v.Value)
		return err
	case *StringTag:
		return e.writeString(v.Value)
	case *ListTag:
		if err := binary.Write(e.w, binary.BigEndian, byte(v.ElementType)); err != nil {
			return err
		}
		if err := binary.Write(e.w, binary.BigEndian, int32(len(v.Elements))); err != nil {
			return err
		}
		for _, el := range v.Elements {
			if el.Type() != v.ElementType {
				return fmt.Errorf("list element type mismatch: got %v, expected %v", el.Type(), v.ElementType)
			}
			if err := e.writePayload(el); err != nil {
				return err
			}
		}
		return nil
	case *CompoundTag:
		return e.writeCompoundPayload(v)
	case *IntArrayTag:
		buf := make([]byte, 4+len(v.Value)*4)
		binary.BigEndian.PutUint32(buf[0:4], uint32(len(v.Value)))
		for i, item := range v.Value {
			binary.BigEndian.PutUint32(buf[4+i*4:8+i*4], uint32(item))
		}
		_, err := e.w.Write(buf)
		return err
	case *LongArrayTag:
		buf := make([]byte, 4+len(v.Value)*8)
		binary.BigEndian.PutUint32(buf[0:4], uint32(len(v.Value)))
		for i, item := range v.Value {
			binary.BigEndian.PutUint64(buf[4+i*8:12+i*8], uint64(item))
		}
		_, err := e.w.Write(buf)
		return err
	default:
		return fmt.Errorf("unsupported tag type: %T", t)
	}
}

func (e *Encoder) writeCompoundPayload(c *CompoundTag) error {
	for name, tag := range c.Value {
		if err := binary.Write(e.w, binary.BigEndian, byte(tag.Type())); err != nil {
			return err
		}
		if err := e.writeString(name); err != nil {
			return err
		}
		if err := e.writePayload(tag); err != nil {
			return err
		}
	}
	return binary.Write(e.w, binary.BigEndian, byte(TagEnd))
}

// Decoder reads NBT structures from an io.Reader.
type Decoder struct {
	r     io.Reader
	nodes int
	depth int
}

const (
	maxNBTBytes      = 64 * 1024 * 1024
	maxNBTArrayBytes = 64 * 1024 * 1024
	maxNBTListLength = 100_000
	maxNBTNodes      = 100_000
	maxNBTDepth      = 256
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: io.LimitReader(r, maxNBTBytes)}
}

func (d *Decoder) DecodeNamed() (string, *CompoundTag, error) {
	var tagTypeByte byte
	if err := binary.Read(d.r, binary.BigEndian, &tagTypeByte); err != nil {
		return "", nil, err
	}
	if TagType(tagTypeByte) != TagCompound {
		return "", nil, fmt.Errorf("root tag must be TAG_Compound, got %v", TagType(tagTypeByte))
	}
	name, err := d.readString()
	if err != nil {
		return "", nil, err
	}
	root, err := d.readCompoundPayload()
	if err != nil {
		return "", nil, err
	}
	return name, root, nil
}

func (d *Decoder) readString() (string, error) {
	var length uint16
	if err := binary.Read(d.r, binary.BigEndian, &length); err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (d *Decoder) readPayload(tagType TagType) (Tag, error) {
	if d.nodes >= maxNBTNodes {
		return nil, fmt.Errorf("NBT tag count exceeds maximum %d", maxNBTNodes)
	}
	d.nodes++

	switch tagType {
	case TagByte:
		var v int8
		err := binary.Read(d.r, binary.BigEndian, &v)
		return &ByteTag{Value: v}, err
	case TagShort:
		var v int16
		err := binary.Read(d.r, binary.BigEndian, &v)
		return &ShortTag{Value: v}, err
	case TagInt:
		var v int32
		err := binary.Read(d.r, binary.BigEndian, &v)
		return &IntTag{Value: v}, err
	case TagLong:
		var v int64
		err := binary.Read(d.r, binary.BigEndian, &v)
		return &LongTag{Value: v}, err
	case TagFloat:
		var v float32
		err := binary.Read(d.r, binary.BigEndian, &v)
		return &FloatTag{Value: v}, err
	case TagDouble:
		var v float64
		err := binary.Read(d.r, binary.BigEndian, &v)
		return &DoubleTag{Value: v}, err
	case TagByteArray:
		var len int32
		if err := binary.Read(d.r, binary.BigEndian, &len); err != nil {
			return nil, err
		}
		if len < 0 || len > maxNBTArrayBytes {
			return nil, fmt.Errorf("invalid byte array length: %d", len)
		}
		buf := make([]byte, len)
		_, err := io.ReadFull(d.r, buf)
		return &ByteArrayTag{Value: buf}, err
	case TagString:
		s, err := d.readString()
		return &StringTag{Value: s}, err
	case TagList:
		var elemTypeByte byte
		if err := binary.Read(d.r, binary.BigEndian, &elemTypeByte); err != nil {
			return nil, err
		}
		var count int32
		if err := binary.Read(d.r, binary.BigEndian, &count); err != nil {
			return nil, err
		}
		if count < 0 || count > maxNBTListLength {
			return nil, fmt.Errorf("invalid list length: %d", count)
		}
		elemType := TagType(elemTypeByte)
		elements := make([]Tag, 0, count)
		for i := int32(0); i < count; i++ {
			el, err := d.readPayload(elemType)
			if err != nil {
				return nil, err
			}
			elements = append(elements, el)
		}
		return &ListTag{ElementType: elemType, Elements: elements}, nil
	case TagCompound:
		return d.readCompoundPayload()
	case TagIntArray:
		var count int32
		if err := binary.Read(d.r, binary.BigEndian, &count); err != nil {
			return nil, err
		}
		if count < 0 || count > 1000000 {
			return nil, fmt.Errorf("invalid int array length: %d", count)
		}
		buf := make([]byte, count*4)
		if _, err := io.ReadFull(d.r, buf); err != nil {
			return nil, err
		}
		arr := make([]int32, count)
		for i := int32(0); i < count; i++ {
			arr[i] = int32(binary.BigEndian.Uint32(buf[i*4 : (i+1)*4]))
		}
		return &IntArrayTag{Value: arr}, nil
	case TagLongArray:
		var count int32
		if err := binary.Read(d.r, binary.BigEndian, &count); err != nil {
			return nil, err
		}
		if count < 0 || count > 1000000 {
			return nil, fmt.Errorf("invalid long array length: %d", count)
		}
		buf := make([]byte, count*8)
		if _, err := io.ReadFull(d.r, buf); err != nil {
			return nil, err
		}
		arr := make([]int64, count)
		for i := int32(0); i < count; i++ {
			arr[i] = int64(binary.BigEndian.Uint64(buf[i*8 : (i+1)*8]))
		}
		return &LongArrayTag{Value: arr}, nil
	default:
		return nil, fmt.Errorf("unknown tag type: %d", tagType)
	}
}

func (d *Decoder) readCompoundPayload() (*CompoundTag, error) {
	if d.depth >= maxNBTDepth {
		return nil, fmt.Errorf("NBT nesting depth exceeds maximum %d", maxNBTDepth)
	}
	d.depth++
	defer func() { d.depth-- }()

	c := NewCompound()
	for {
		var tagTypeByte byte
		if err := binary.Read(d.r, binary.BigEndian, &tagTypeByte); err != nil {
			return nil, err
		}
		tagType := TagType(tagTypeByte)
		if tagType == TagEnd {
			break
		}
		name, err := d.readString()
		if err != nil {
			return nil, err
		}
		payload, err := d.readPayload(tagType)
		if err != nil {
			return nil, err
		}
		c.Put(name, payload)
	}
	return c, nil
}

// WriteGzipped encodes an NBT compound compressed with gzip.
func WriteGzipped(w io.Writer, name string, root *CompoundTag) error {
	gw := gzip.NewWriter(w)
	encodeErr := NewEncoder(gw).EncodeNamed(name, root)
	closeErr := gw.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

// ReadGzipped decodes an NBT compound from a gzip-compressed stream.
func ReadGzipped(r io.Reader) (string, *CompoundTag, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return "", nil, err
	}
	defer gr.Close()
	limited := &io.LimitedReader{R: gr, N: maxNBTBytes + 1}
	name, root, err := NewDecoder(limited).DecodeNamed()
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(io.Discard, limited); err != nil {
		return "", nil, err
	}
	if limited.N == 0 {
		return "", nil, fmt.Errorf("NBT payload exceeds maximum %d bytes", maxNBTBytes)
	}
	return name, root, nil
}

// WriteZlib encodes an NBT compound compressed with zlib (as in MCA chunks).
func WriteZlib(w io.Writer, name string, root *CompoundTag) error {
	zw := zlib.NewWriter(w)
	encodeErr := NewEncoder(zw).EncodeNamed(name, root)
	closeErr := zw.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

// ReadZlib decodes an NBT compound from a zlib-compressed stream.
func ReadZlib(r io.Reader) (string, *CompoundTag, error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return "", nil, err
	}
	defer zr.Close()
	limited := &io.LimitedReader{R: zr, N: maxNBTBytes + 1}
	name, root, err := NewDecoder(limited).DecodeNamed()
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(io.Discard, limited); err != nil {
		return "", nil, err
	}
	if limited.N == 0 {
		return "", nil, fmt.Errorf("NBT payload exceeds maximum %d bytes", maxNBTBytes)
	}
	return name, root, nil
}

// Marshal returns uncompressed binary NBT bytes.
func Marshal(name string, root *CompoundTag) ([]byte, error) {
	var buf bytes.Buffer
	if err := NewEncoder(&buf).EncodeNamed(name, root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal decodes uncompressed binary NBT bytes.
func Unmarshal(data []byte) (string, *CompoundTag, error) {
	return NewDecoder(bytes.NewReader(data)).DecodeNamed()
}
