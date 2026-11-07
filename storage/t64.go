package storage

import (
	"encoding/binary"
	"errors"
	"strings"
)

/*
T64 files, a container of the files of a tape with no pulses, as the program
files they hold.

Header of 64 bytes, starting with "C64" and with the count of entries at $22.
Entries of 32 bytes from $40: the type, 1 for a normal file, the file type,
the start and end addresses, the offset of the data in the file, and the
name.

See: https://vice-emu.sourceforge.io/vice_17.html#SEC386
*/

const (
	t64HeaderSize = 0x40
	t64EntrySize  = 0x20
)

// IsT64File returns true for the data of a T64 file
func IsT64File(data []uint8) bool {
	return len(data) >= t64HeaderSize && strings.HasPrefix(string(data), "C64")
}

// T64Programs returns the files of a T64 file as program files, with the
// load address on the first two bytes
func T64Programs(data []uint8) ([][]uint8, error) {
	if !IsT64File(data) {
		return nil, errors.New("not a T64 file")
	}
	maxEntries := int(binary.LittleEndian.Uint16(data[0x22:]))
	var programs [][]uint8
	for i := 0; i < maxEntries; i++ {
		entry := t64HeaderSize + i*t64EntrySize
		if entry+t64EntrySize > len(data) {
			break
		}
		if data[entry] != 1 {
			continue
		}
		start := binary.LittleEndian.Uint16(data[entry+2:])
		end := binary.LittleEndian.Uint16(data[entry+4:])
		offset := int(binary.LittleEndian.Uint32(data[entry+8:]))
		length := int(end) - int(start)
		if length <= 0 || offset >= len(data) {
			continue
		}
		// Some T64 files have a wrong end address, take what there is
		length = min(length, len(data)-offset)
		program := append([]uint8{uint8(start), uint8(start >> 8)}, data[offset:offset+length]...)
		programs = append(programs, program)
	}
	if len(programs) == 0 {
		return nil, errors.New("no files in the T64 file")
	}
	return programs, nil
}
