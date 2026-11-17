package nbt

import (
	"bytes"
	"testing"
)

func TestNBTRoundTripAllTypes(t *testing.T) {
	root := NewCompound()
	root.Put("byteVal", &ByteTag{Value: -42})
	root.Put("shortVal", &ShortTag{Value: 12345})
	root.Put("intVal", &IntTag{Value: 987654321})
	root.Put("longVal", &LongTag{Value: 1234567890123456789})
	root.Put("floatVal", &FloatTag{Value: 3.14159})
	root.Put("doubleVal", &DoubleTag{Value: 2.718281828459})
	root.Put("stringVal", &StringTag{Value: "Minecraft in Go!"})
	root.Put("byteArray", &ByteArrayTag{Value: []byte{1, 2, 3, 4, 5, 255}})
	root.Put("intArray", &IntArrayTag{Value: []int32{10, 20, 30, -40, 50}})
	root.Put("longArray", &LongArrayTag{Value: []int64{100, 200, 300, -400, 500}})

	list := &ListTag{
		ElementType: TagString,
		Elements: []Tag{
			&StringTag{Value: "item1"},
			&StringTag{Value: "item2"},
			&StringTag{Value: "item3"},
		},
	}
	root.Put("stringList", list)

	subCompound := NewCompound()
	subCompound.Put("nestedKey", &StringTag{Value: "nestedValue"})
	root.Put("nested", subCompound)

	// Test 1: Direct Marshal / Unmarshal
	data, err := Marshal("Level", root)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	name, decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if name != "Level" {
		t.Fatalf("root name mismatch: got %q, expected %q", name, "Level")
	}

	if decoded.GetString("stringVal") != "Minecraft in Go!" {
		t.Fatalf("stringVal mismatch: got %s", decoded.GetString("stringVal"))
	}
	if decoded.GetInt("intVal") != 987654321 {
		t.Fatalf("intVal mismatch: got %d", decoded.GetInt("intVal"))
	}
	if decoded.GetLong("longVal") != 1234567890123456789 {
		t.Fatalf("longVal mismatch: got %d", decoded.GetLong("longVal"))
	}

	// Test 2: Gzip compression roundtrip
	var gzBuf bytes.Buffer
	if err := WriteGzipped(&gzBuf, "LevelGz", root); err != nil {
		t.Fatalf("WriteGzipped failed: %v", err)
	}

	gzName, gzDecoded, err := ReadGzipped(&gzBuf)
	if err != nil {
		t.Fatalf("ReadGzipped failed: %v", err)
	}
	if gzName != "LevelGz" {
		t.Fatalf("gzip name mismatch: %s", gzName)
	}
	if gzDecoded.GetString("stringVal") != "Minecraft in Go!" {
		t.Fatalf("gzip stringVal mismatch: %s", gzDecoded.GetString("stringVal"))
	}

	// Test 3: Zlib compression roundtrip (MCA chunk format)
	var zlibBuf bytes.Buffer
	if err := WriteZlib(&zlibBuf, "Chunk", root); err != nil {
		t.Fatalf("WriteZlib failed: %v", err)
	}

	zlibName, zlibDecoded, err := ReadZlib(&zlibBuf)
	if err != nil {
		t.Fatalf("ReadZlib failed: %v", err)
	}
	if zlibName != "Chunk" {
		t.Fatalf("zlib name mismatch: %s", zlibName)
	}
	if zlibDecoded.GetString("stringVal") != "Minecraft in Go!" {
		t.Fatalf("zlib stringVal mismatch: %s", zlibDecoded.GetString("stringVal"))
	}
}
