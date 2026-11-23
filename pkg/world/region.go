package world

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
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
	mu   sync.Mutex
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
	// Initialize empty headers if new file; report error if corrupted/truncated
	if info.Size() == 0 {
		emptyHeader := make([]byte, 8192)
		if _, err := file.WriteAt(emptyHeader, 0); err != nil {
			file.Close()
			return nil, err
		}
	} else if info.Size() < 8192 {
		file.Close()
		return nil, fmt.Errorf("corrupt region file: size %d is less than 8192 header bytes", info.Size())
	}
	return &Region{file: file}, nil
}

func (r *Region) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
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

func (r *Region) chunkRangeOverlaps(index, offset, sectors int) (bool, error) {
	locations := make([]byte, 4096)
	if _, err := r.file.ReadAt(locations, 0); err != nil {
		return false, err
	}
	end := offset + sectors
	for other := 0; other < ChunksPerRegion; other++ {
		if other == index {
			continue
		}
		base := other * 4
		otherOffset := (int(locations[base]) << 16) | (int(locations[base+1]) << 8) | int(locations[base+2])
		otherSectors := int(locations[base+3])
		if otherOffset < 2 || otherSectors == 0 {
			continue
		}
		if offset < otherOffset+otherSectors && otherOffset < end {
			return true, nil
		}
	}
	return false, nil
}

// ReadChunkData reads raw uncompressed NBT bytes of a chunk at (chunkX, chunkZ).
func (r *Region) ReadChunkData(chunkX, chunkZ int32) (*nbt.CompoundTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

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
	if offset < 2 || sectorCount == 0 {
		return nil, fmt.Errorf("corrupt chunk location: offset %d, sector count %d", offset, sectorCount)
	}
	info, err := r.file.Stat()
	if err != nil {
		return nil, err
	}

	byteOffset := offset * SectorSize
	allocatedBytes := int64(sectorCount * SectorSize)
	if byteOffset+allocatedBytes > info.Size() {
		return nil, fmt.Errorf("corrupt chunk location: sectors [%d, %d) exceed region size %d", offset, offset+int64(sectorCount), info.Size())
	}
	overlap, err := r.chunkRangeOverlaps(idx, int(offset), sectorCount)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, fmt.Errorf("corrupt chunk location: sectors [%d, %d) overlap another chunk", offset, offset+int64(sectorCount))
	}
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
	if length > uint32(sectorCount*SectorSize-4) {
		return nil, fmt.Errorf("corrupt chunk length %d exceeds allocated sectors %d", length, sectorCount)
	}

	compressedData := make([]byte, length-1)
	if _, err := r.file.ReadAt(compressedData, byteOffset+5); err != nil {
		return nil, err
	}

	switch compType {
	case CompressionGzip:
		gr, err := gzip.NewReader(bytes.NewReader(compressedData))
		if err != nil {
			return nil, err
		}
		defer gr.Close()
		_, tag, err := nbt.NewDecoder(gr).DecodeNamed()
		return tag, err
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

// WriteChunkData writes an NBT chunk tag to the region file with in-place overwrite support.
func (r *Region) WriteChunkData(chunkX, chunkZ int32, chunkNBT *nbt.CompoundTag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

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
	if sectorsNeeded > 255 {
		return fmt.Errorf("chunk requires %d sectors; maximum encodable allocation is 255", sectorsNeeded)
	}

	idx := chunkIndex(chunkX, chunkZ)
	locBuf := make([]byte, 4)
	var existingOffset, existingSectors int
	if _, err := r.file.ReadAt(locBuf, int64(idx*4)); err == nil {
		existingOffset = (int(locBuf[0]) << 16) | (int(locBuf[1]) << 8) | int(locBuf[2])
		existingSectors = int(locBuf[3])
	}
	info, err := r.file.Stat()
	if err != nil {
		return err
	}
	existingRangeValid := existingOffset >= 2 && existingSectors >= sectorsNeeded &&
		int64(existingOffset+existingSectors)*SectorSize <= info.Size()
	if existingRangeValid {
		overlap, err := r.chunkRangeOverlaps(idx, existingOffset, existingSectors)
		if err != nil {
			return err
		}
		existingRangeValid = !overlap
	}

	var sectorOffset int
	var allocSectors int
	if existingRangeValid {
		// Overwrite in-place without expanding file!
		sectorOffset = existingOffset
		allocSectors = existingSectors
	} else {
		// Allocate at end of file
		fileSize := info.Size()
		if fileSize < 8192 {
			fileSize = 8192
		}
		sectorOffset = int((fileSize + SectorSize - 1) / SectorSize)
		if sectorOffset < 2 {
			sectorOffset = 2
		}
		allocSectors = sectorsNeeded
	}
	if sectorOffset > 0xFFFFFF {
		return fmt.Errorf("region file sector offset %d exceeds 24-bit location header", sectorOffset)
	}

	// Write payload
	writeOffset := int64(sectorOffset) * SectorSize
	var payload bytes.Buffer
	if err := binary.Write(&payload, binary.BigEndian, totalPayloadLen); err != nil {
		return err
	}
	payload.WriteByte(CompressionZlib)
	payload.Write(compressed)

	// Pad to full allocated sectors
	padding := (allocSectors * SectorSize) - payload.Len()
	if padding > 0 {
		payload.Write(make([]byte, padding))
	}

	if _, err := r.file.WriteAt(payload.Bytes(), writeOffset); err != nil {
		return err
	}
	if err := r.file.Sync(); err != nil {
		return err
	}

	// Update Location Header
	locHeader := []byte{
		byte(sectorOffset >> 16),
		byte(sectorOffset >> 8),
		byte(sectorOffset),
		byte(allocSectors),
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
	return r.file.Sync()
}
