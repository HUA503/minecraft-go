package world

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
)

const (
	regionJournalRecordSize = 28
	regionJournalPrepared   = 1
	regionJournalCommitted  = 2
)

type regionTransaction struct {
	index        int
	oldLocation  [4]byte
	newLocation  [4]byte
	oldTimestamp [4]byte
	newTimestamp [4]byte
}

func encodeRegionJournalRecord(state byte, transaction regionTransaction) []byte {
	record := make([]byte, regionJournalRecordSize)
	copy(record[:4], "MCT1")
	record[4] = state
	binary.BigEndian.PutUint16(record[6:8], uint16(transaction.index))
	copy(record[8:12], transaction.oldLocation[:])
	copy(record[12:16], transaction.newLocation[:])
	copy(record[16:20], transaction.oldTimestamp[:])
	copy(record[20:24], transaction.newTimestamp[:])
	binary.BigEndian.PutUint32(record[24:28], crc32.ChecksumIEEE(record[:24]))
	return record
}

func decodeRegionJournalRecord(record []byte) (byte, regionTransaction, bool) {
	if len(record) != regionJournalRecordSize || string(record[:4]) != "MCT1" ||
		binary.BigEndian.Uint32(record[24:28]) != crc32.ChecksumIEEE(record[:24]) {
		return 0, regionTransaction{}, false
	}
	state := record[4]
	if state != regionJournalPrepared && state != regionJournalCommitted {
		return 0, regionTransaction{}, false
	}
	transaction := regionTransaction{index: int(binary.BigEndian.Uint16(record[6:8]))}
	if transaction.index >= ChunksPerRegion {
		return 0, regionTransaction{}, false
	}
	copy(transaction.oldLocation[:], record[8:12])
	copy(transaction.newLocation[:], record[12:16])
	copy(transaction.oldTimestamp[:], record[16:20])
	copy(transaction.newTimestamp[:], record[20:24])
	return state, transaction, true
}

func sameRegionTransaction(a, b regionTransaction) bool {
	return a.index == b.index && a.oldLocation == b.oldLocation && a.newLocation == b.newLocation &&
		a.oldTimestamp == b.oldTimestamp && a.newTimestamp == b.newTimestamp
}

func writeRegionJournal(path string, state byte, transaction regionTransaction) error {
	flags := os.O_WRONLY
	if state == regionJournalPrepared {
		flags |= os.O_CREATE | os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return err
	}
	record := encodeRegionJournalRecord(state, transaction)
	offset := int64(0)
	if state == regionJournalCommitted {
		offset = regionJournalRecordSize
	}
	writeErr := writeAtFull(file, record, offset)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func applyRegionTransaction(file regionFile, transaction regionTransaction, committed bool) error {
	location := transaction.oldLocation[:]
	timestamp := transaction.oldTimestamp[:]
	if committed {
		location = transaction.newLocation[:]
		timestamp = transaction.newTimestamp[:]
	}
	if err := writeAtFull(file, location, int64(transaction.index*4)); err != nil {
		return err
	}
	if err := writeAtFull(file, timestamp, int64(4096+transaction.index*4)); err != nil {
		return err
	}
	return file.Sync()
}

// recoverRegionJournal replays a complete committed record or rolls back to the
// old location when only the prepared record exists. A partial journal record
// cannot have modified the region because the region is touched only after the
// prepared record has been fully synced.
func recoverRegionJournal(path string, file regionFile) error {
	journalPath := path + ".txn"
	data, err := os.ReadFile(journalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) > regionJournalRecordSize*2 {
		return fmt.Errorf("region transaction journal has unexpected size %d", len(data))
	}
	if len(data) < regionJournalRecordSize {
		return removeRegionJournal(journalPath)
	}
	state, transaction, ok := decodeRegionJournalRecord(data[:regionJournalRecordSize])
	if !ok {
		return fmt.Errorf("region transaction journal has an invalid prepared record; preserved at %s", journalPath)
	}
	if state != regionJournalPrepared {
		return fmt.Errorf("region transaction journal starts in invalid state %d; preserved at %s", state, journalPath)
	}
	committed := false
	if len(data) >= regionJournalRecordSize*2 {
		commitState, commitTransaction, valid := decodeRegionJournalRecord(data[regionJournalRecordSize : regionJournalRecordSize*2])
		committed = valid && commitState == regionJournalCommitted && sameRegionTransaction(transaction, commitTransaction)
	}
	if err := applyRegionTransaction(file, transaction, committed); err != nil {
		return fmt.Errorf("recover region transaction for chunk slot %d: %w", transaction.index, err)
	}
	return removeRegionJournal(journalPath)
}

func removeRegionJournal(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("sync region journal removal: %w", err)
	}
	return nil
}

func (r *Region) recoverPendingJournal() error {
	if r.journalDirectorySyncPending {
		if err := syncDirectory(filepath.Dir(r.path)); err != nil {
			return fmt.Errorf("sync region directory after journal removal: %w", err)
		}
		r.journalDirectorySyncPending = false
	}
	if !r.journalPresent {
		return nil
	}
	err := recoverRegionJournal(r.path, r.file)
	if err == nil {
		r.journalPresent = false
		return nil
	}
	if _, statErr := os.Stat(r.path + ".txn"); os.IsNotExist(statErr) {
		r.journalPresent = false
		r.journalDirectorySyncPending = true
	}
	return err
}
