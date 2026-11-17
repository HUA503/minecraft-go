package world

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HUA503/minecraft-go/pkg/nbt"
)

const (
	SectorSize      = 4096
	ChunksPerRegion = 1024
	CompressionGzip = 1
	CompressionZlib = 2
	CompressionNone = 3
)

// Region represents an Anvil MCA region file reader and writer.
type Region struct {
	file *os.File
}

func OpenRegion(path string) (*Region, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	// Initialize empty headers if new file
	if info.Size() < 8192 {
		emptyHeader := make([]byte, 8192)
		if _, err := file.WriteAt(emptyHeader, 0); err != nil {
			file.Close()
			return nil, err
		}
	}
	return &Region{file: file}, nil
}

func (r *Region) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

func chunkIndex(chunkX, chunkZ int32) int {
	rx := chunkX & 31
	rz := chunkZ & 31
	return int((rx & 31) + (rz&31)*32)
}

// ReadChunkData reads raw uncompressed NBT bytes of a chunk at (chunkX, chunkZ).
func (r *Region) ReadChunkData(chunkX, chunkZ int32) (*nbt.CompoundTag, error) {
	idx := chunkIndex(chunkX, chunkZ)
	locBuf := make([]byte, 4)
	if _, err := r.file.ReadAt(locBuf, int64(idx*4)); err != nil {
		return nil, err
	}

	offset := (int64(locBuf[0]) << 16) | (int64(locBuf[1]) << 8) | int64(locBuf[2])
	sectorCount := int(locBuf[3])

	if offset == 0 && sectorCount == 0 {
		return nil, errors.New("chunk not generated in region")
	}

	byteOffset := offset * SectorSize
	var length uint32
	if err := binary.Read(io.NewSectionReader(r.file, byteOffset, 4), binary.BigEndian, &length); err != nil {
		return nil, err
	}

	var compType byte
	if err := binary.Read(io.NewSectionReader(r.file, byteOffset+4, 1), binary.BigEndian, &compType); err != nil {
		return nil, err
	}

	if length <= 1 {
		return nil, errors.New("corrupt chunk length")
	}

	compressedData := make([]byte, length-1)
	if _, err := r.file.ReadAt(compressedData, byteOffset+5); err != nil {
		return nil, err
	}

	switch compType {
	case CompressionZlib:
		zr, err := zlib.NewReader(bytes.NewReader(compressedData))
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		_, tag, err := nbt.NewDecoder(zr).DecodeNamed()
		return tag, err
	case CompressionNone:
		_, tag, err := nbt.NewDecoder(bytes.NewReader(compressedData)).DecodeNamed()
		return tag, err
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", compType)
	}
}

// WriteChunkData writes an NBT chunk tag to the region file.
func (r *Region) WriteChunkData(chunkX, chunkZ int32, chunkNBT *nbt.CompoundTag) error {
	var zlibBuf bytes.Buffer
	zw := zlib.NewWriter(&zlibBuf)
	if err := nbt.NewEncoder(zw).EncodeNamed("", chunkNBT); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	compressed := zlibBuf.Bytes()
	totalPayloadLen := uint32(len(compressed) + 1) // +1 for compression type
	requiredBytes := 4 + totalPayloadLen
	sectorsNeeded := int((requiredBytes + SectorSize - 1) / SectorSize)

	// Seek to end of file to allocate sectors
	stat, err := r.file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()
	if fileSize < 8192 {
		fileSize = 8192
	}
	sectorOffset := int(fileSize / SectorSize)
	if fileSize%SectorSize != 0 {
		sectorOffset++
	}

	// Write payload
	writeOffset := int64(sectorOffset) * SectorSize
	var payload bytes.Buffer
	if err := binary.Write(&payload, binary.BigEndian, totalPayloadLen); err != nil {
		return err
	}
	payload.WriteByte(CompressionZlib)
	payload.Write(compressed)

	// Pad to full sectors
	padding := (sectorsNeeded * SectorSize) - payload.Len()
	if padding > 0 {
		payload.Write(make([]byte, padding))
	}

	if _, err := r.file.WriteAt(payload.Bytes(), writeOffset); err != nil {
		return err
	}

	// Update Location Header
	idx := chunkIndex(chunkX, chunkZ)
	locHeader := []byte{
		byte(sectorOffset >> 16),
		byte(sectorOffset >> 8),
		byte(sectorOffset),
		byte(sectorsNeeded),
	}
	if _, err := r.file.WriteAt(locHeader, int64(idx*4)); err != nil {
		return err
	}

	// Update Timestamp Header
	now := uint32(time.Now().Unix())
	tsHeader := make([]byte, 4)
	binary.BigEndian.PutUint32(tsHeader, now)
	if _, err := r.file.WriteAt(tsHeader, int64(4096+idx*4)); err != nil {
		return err
	}

	return nil
}
