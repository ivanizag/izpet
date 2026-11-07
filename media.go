package izpet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ivanizag/izpet/storage"
)

/*
The media given to the PET: programs, tapes and disks, by the type of the
file.

	.prg        a program, loaded and run once BASIC is ready
	.t64        the programs of a tape, the first is loaded
	.tap, .wav  a tape for the deck #1
	.d64, .d80, .d82, a folder  a disk for the drive 0 of the unit 8, or
	            the drive 1 if the drive 0 has one already
*/

const diskUnit = 8

type mediaKind int

const (
	mediaProgram mediaKind = iota
	mediaT64
	mediaTape
	mediaDisk
)

func mediaKindOf(path string) (mediaKind, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return mediaDisk, nil
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".prg", ".p00":
		return mediaProgram, nil
	case ".t64":
		return mediaT64, nil
	case ".tap", ".wav":
		return mediaTape, nil
	case ".d64", ".d80", ".d82":
		return mediaDisk, nil
	}
	return 0, fmt.Errorf("unknown type of file %v, the types are .prg, .t64, .tap, .wav, .d64, .d80, .d82 and folders", path)
}

// LoadFile puts a file where it belongs by its type: a program to run, a
// tape in the deck #1 or a disk in the drive
func (p *PET) LoadFile(path string) error {
	kind, err := mediaKindOf(path)
	if err != nil {
		return err
	}
	switch kind {
	case mediaT64:
		data, err := LoadResource(path)
		if err != nil {
			return err
		}
		programs, err := storage.T64Programs(data)
		if err != nil {
			return err
		}
		return p.loadProgramData(path, programs[0])
	case mediaTape:
		return p.InsertTape(1, path)
	case mediaDisk:
		drive := 0
		if p.drive.DiskName(0) != "" {
			drive = 1
		}
		return p.InsertDisk(drive, path)
	default:
		return p.LoadProgram(path)
	}
}

// InsertDisk puts a disk image, or a folder of the host, in the drive 0 or
// 1 of the unit 8. A disk image that does not exist is created, blank, when
// the PET writes on it.
func (p *PET) InsertDisk(drive int, path string) error {
	path = normalizeFilename(path)
	var data []byte
	writable := false
	info, err := os.Stat(path)
	switch {
	case err == nil && info.IsDir():
		writable = true
	case err == nil:
		data, err = LoadResource(path)
		if err != nil {
			return err
		}
		writable = info.Mode().Perm()&0200 != 0
	case isInternalResource(path) || isHTTPResource(path):
		data, err = LoadResource(path)
		if err != nil {
			return err
		}
	case os.IsNotExist(err):
		// A new disk
		writable = true
	default:
		return err
	}
	return p.drive.Insert(drive, path, data, writable)
}

// DiskName returns the name and id of the disk in a drive, empty if none
func (p *PET) DiskName(drive int) string {
	return p.drive.DiskName(drive)
}

// DriveStatus returns the error status of the disk drive
func (p *PET) DriveStatus() string {
	return p.drive.Status()
}

// InsertTape puts a tape in the deck 1 or 2
func (p *PET) InsertTape(deck int, path string) error {
	if deck != 1 && deck != 2 {
		return fmt.Errorf("there is no tape deck %v", deck)
	}
	return p.cassettes[deck-1].insert(normalizeFilename(path))
}

// isTapeMoving returns true while a deck plays or records
func (p *PET) isTapeMoving() bool {
	return p.cassettes[0].isMoving() || p.cassettes[1].isMoving()
}

// Close saves what is pending: the tapes recorded and the printer output
func (p *PET) Close() error {
	for _, c := range p.cassettes {
		c.save()
	}
	if p.printer != nil {
		return p.printer.close()
	}
	return nil
}
