package world

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
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

// ErrChunkNotFound means a valid region file has no allocated payload for the
// requested chunk. Callers may generate a new chunk only for this condition;
// other read errors can indicate damaged data that must be preserved.
var ErrChunkNotFound = errors.New("chunk not generated in region")

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
		if err := file.Sync(); err != nil {
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

func writeAtFull(file *os.File, data []byte, offset int64) error {
	n, err := file.WriteAt(data, offset)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

type sectorRange struct {
	start int
	end   int
}

// findFreeSectors validates the location table and finds a free contiguous run.
// The current chunk allocation remains occupied until its replacement header is committed.
func (r *Region) findFreeSectors(sectorsNeeded int, fileSize int64) (int, error) {
	locations := make([]byte, SectorSize)
	if _, err := r.file.ReadAt(locations, 0); err != nil {
		return 0, err
	}
	allocations := make([]sectorRange, 0, ChunksPerRegion)
	for i := 0; i < ChunksPerRegion; i++ {
		base := i * 4
		offset := (int(locations[base]) << 16) | (int(locations[base+1]) << 8) | int(locations[base+2])
		sectors := int(locations[base+3])
		if offset == 0 && sectors == 0 {
			continue
		}
		if offset < 2 || sectors == 0 {
			return 0, fmt.Errorf("corrupt region location entry %d: offset %d, sectors %d", i, offset, sectors)
		}
		end := offset + sectors
		if int64(end)*SectorSize > fileSize {
			return 0, fmt.Errorf("corrupt region location entry %d: sectors [%d, %d) exceed file size %d", i, offset, end, fileSize)
		}
		allocations = append(allocations, sectorRange{start: offset, end: end})
	}
	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].start < allocations[j].start
	})

	cursor := 2
	for i, allocation := range allocations {
		if allocation.start < cursor {
			return 0, fmt.Errorf("corrupt region location table: overlapping allocations at sector %d", allocation.start)
		}
		if allocation.start-cursor >= sectorsNeeded {
			return cursor, nil
		}
		cursor = allocation.end
		if i == len(allocations)-1 {
			break
		}
	}
	fileSectors := int((fileSize + SectorSize - 1) / SectorSize)
	if fileSectors-cursor >= sectorsNeeded {
		return cursor, nil
	}
	if fileSectors < cursor {
		fileSectors = cursor
	}
	return fileSectors, nil
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
		return nil, ErrChunkNotFound
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
		_, tag, err := nbt.ReadGzipped(bytes.NewReader(compressedData))
		return tag, err
	case CompressionZlib:
		_, tag, err := nbt.ReadZlib(bytes.NewReader(compressedData))
		return tag, err
	case CompressionNone:
		payload := bytes.NewReader(compressedData)
		_, tag, err := nbt.NewDecoder(payload).DecodeNamed()
		if err == nil && payload.Len() != 0 {
			return nil, fmt.Errorf("uncompressed chunk has %d trailing bytes", payload.Len())
		}
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
	oldLocation := make([]byte, 4)
	if _, err := r.file.ReadAt(oldLocation, int64(idx*4)); err != nil {
		return err
	}
	info, err := r.file.Stat()
	if err != nil {
		return err
	}
	fileSize := info.Size()
	sectorOffset, err := r.findFreeSectors(sectorsNeeded, fileSize)
	if err != nil {
		return err
	}
	allocSectors := sectorsNeeded
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

	if err := writeAtFull(r.file, payload.Bytes(), writeOffset); err != nil {
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
	if err := writeAtFull(r.file, locHeader, int64(idx*4)); err != nil {
		rollbackErr := writeAtFull(r.file, oldLocation, int64(idx*4))
		if rollbackErr == nil {
			rollbackErr = r.file.Sync()
		}
		if rollbackErr != nil {
			return fmt.Errorf("write chunk location header: %w (restore previous location header: %v)", err, rollbackErr)
		}
		return fmt.Errorf("write chunk location header: %w; previous location header restored", err)
	}
	// The new payload was synced before this pointer update. Sync the pointer
	// before writing the timestamp so the previous allocation stays recoverable
	// if committing the new location fails.
	if err := r.file.Sync(); err != nil {
		rollbackErr := writeAtFull(r.file, oldLocation, int64(idx*4))
		if rollbackErr == nil {
			rollbackErr = r.file.Sync()
		}
		if rollbackErr != nil {
			return fmt.Errorf("sync chunk location header: %w (restore previous location header: %v)", err, rollbackErr)
		}
		return fmt.Errorf("sync chunk location header: %w; previous location header restored", err)
	}

	// Update Timestamp Header
	now := uint32(time.Now().Unix())
	tsHeader := make([]byte, 4)
	binary.BigEndian.PutUint32(tsHeader, now)
	if err := writeAtFull(r.file, tsHeader, int64(4096+idx*4)); err != nil {
		return err
	}
	return r.file.Sync()
}
