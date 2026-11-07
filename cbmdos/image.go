package cbmdos

import (
	"errors"
	"fmt"
	"strings"
)

/*
The disk images of the Commodore drives, the sectors of 256 bytes one after
the other, track by track from track 1:

	D64  4040, 2031  35 tracks, 17 to 21 sectors, 683 blocks
	D80  8050        77 tracks, 23 to 29 sectors, 2083 blocks
	D82  8250        154 tracks, the 77 of the 8050 on each side, 4166 blocks

A file is a chain of blocks: the first two bytes of a block are the track
and sector of the next one, or a zero track and the position of the last
byte used. The other 254 bytes are data.

The directory is a chain of blocks on the directory track with 8 entries of
32 bytes: the file type at 2, the first block at 3, the name at 5, padded
with $a0, and the size in blocks at 30. The type has the bit 7 set for a
closed file, the bit 6 for a locked one, and 0 DEL, 1 SEQ, 2 PRG, 3 USR, 4
REL in the low bits.

The block availability map, BAM, has a bit per sector, set when it is free,
and the count of free sectors of each track. On the D64 it is on 18/0 with
the name of the disk, 4 bytes per track. On the D80 and D82 the header is
on 39/0 and the BAM on 38/0, 38/3... with 50 tracks each, 5 bytes per track.

See:
  - http://unusedino.de/ec64/technical/formats/d64.html
  - http://unusedino.de/ec64/technical/formats/d80-d82.html
*/

type trackSector struct {
	track  int
	sector int
}

type geometry struct {
	name    string
	dosName string
	dosType string
	format  byte // The format byte of the BAM and the header, 'A' or 'C'
	tracks  int

	dirTrack      int
	header        trackSector
	nameOffset    int
	idOffset      int
	dosTypeOffset int
	reserved      []int // Tracks not used for files

	bam           []trackSector
	bamTracks     int // Tracks per BAM sector, 0 for the D64 layout
	bamEntrySize  int
	bamFirstEntry int
}

func geometryD64() *geometry {
	return &geometry{
		name: "D64", dosName: "CBM DOS V2.1 4040", dosType: "2A", format: 'A', tracks: 35,
		dirTrack: 18, header: trackSector{18, 0},
		nameOffset: 0x90, idOffset: 0xa2, dosTypeOffset: 0xa5,
		reserved: []int{18},
		bam:      []trackSector{{18, 0}}, bamTracks: 0, bamEntrySize: 4, bamFirstEntry: 4,
	}
}

func geometryD80() *geometry {
	return &geometry{
		name: "D80", dosName: "CBM DOS V2.5 8050", dosType: "2C", format: 'C', tracks: 77,
		dirTrack: 39, header: trackSector{39, 0},
		nameOffset: 0x06, idOffset: 0x18, dosTypeOffset: 0x1b,
		reserved: []int{39},
		bam:      []trackSector{{38, 0}, {38, 3}}, bamTracks: 50, bamEntrySize: 5, bamFirstEntry: 6,
	}
}

func geometryD82() *geometry {
	g := geometryD80()
	g.name = "D82"
	g.dosName = "CBM DOS V2.7 8250"
	g.tracks = 154
	g.bam = []trackSector{{38, 0}, {38, 3}, {38, 6}, {38, 9}}
	return g
}

func (g *geometry) sectors(track int) int {
	if g.name == "D64" {
		switch {
		case track <= 17:
			return 21
		case track <= 24:
			return 19
		case track <= 30:
			return 18
		default:
			return 17
		}
	}
	if track > 77 {
		track -= 77
	}
	switch {
	case track <= 39:
		return 29
	case track <= 53:
		return 27
	case track <= 64:
		return 25
	default:
		return 23
	}
}

func (g *geometry) blocks() int {
	total := 0
	for t := 1; t <= g.tracks; t++ {
		total += g.sectors(t)
	}
	return total
}

func (g *geometry) isReserved(track int) bool {
	for _, t := range g.reserved {
		if t == track {
			return true
		}
	}
	return false
}

// geometryForSize returns the geometry of an image of a size, a D64 can
// have a byte of errors per block after the blocks
func geometryForSize(size int) (*geometry, error) {
	for _, g := range []*geometry{geometryD64(), geometryD80(), geometryD82()} {
		blocks := g.blocks()
		if size == blocks*sectorSize || (g.name == "D64" && size == blocks*(sectorSize+1)) {
			return g, nil
		}
	}
	return nil, fmt.Errorf("a disk image of %v bytes is not a D64, D80 or D82", size)
}

// geometryForName returns the geometry for the extension of a file name
func geometryForName(name string) *geometry {
	switch {
	case strings.HasSuffix(strings.ToLower(name), ".d80"):
		return geometryD80()
	case strings.HasSuffix(strings.ToLower(name), ".d82"):
		return geometryD82()
	default:
		return geometryD64()
	}
}

const (
	sectorSize      = 256
	blockDataSize   = 254
	dirEntrySize    = 32
	dirEntries      = 8
	maxChainBlocks  = 5000
	nameLength      = 16
	padding         = 0xa0
	fileTypeClosed  = 0x80
	fileTypeLocked  = 0x40
	fileTypeMask    = 0x07
	dirEntryType    = 2
	dirEntryFirst   = 3
	dirEntryName    = 5
	dirEntryBlocks  = 30
	bamFormatOffset = 2
)

var (
	errDiskFull      = errors.New("disk full")
	errFileNotFound  = errors.New("file not found")
	errFileExists    = errors.New("file exists")
	errBadChain      = errors.New("bad block chain")
	errWriteProtect  = errors.New("write protect on")
	errDirectoryFull = errors.New("directory full")
)

// diskImage is a filesystem on a disk image
type diskImage struct {
	g    *geometry
	data []byte
	save func([]byte) error // Called after each change, nil if it can't be saved
}

func newDiskImage(data []byte, save func([]byte) error) (*diskImage, error) {
	g, err := geometryForSize(len(data))
	if err != nil {
		return nil, err
	}
	blocks := g.blocks() * sectorSize
	return &diskImage{g, data[:blocks], save}, nil
}

// newBlankImage returns a formatted image of the geometry
func newBlankImage(g *geometry, save func([]byte) error) *diskImage {
	d := &diskImage{g, make([]byte, g.blocks()*sectorSize), save}
	d.formatBlocks([]byte("BLANK"), []byte("IZ"))
	return d
}

func (d *diskImage) block(ts trackSector) ([]byte, error) {
	if ts.track < 1 || ts.track > d.g.tracks || ts.sector < 0 || ts.sector >= d.g.sectors(ts.track) {
		return nil, fmt.Errorf("illegal track %v or sector %v", ts.track, ts.sector)
	}
	offset := 0
	for t := 1; t < ts.track; t++ {
		offset += d.g.sectors(t)
	}
	offset = (offset + ts.sector) * sectorSize
	return d.data[offset : offset+sectorSize], nil
}

func (d *diskImage) mustBlock(ts trackSector) []byte {
	b, err := d.block(ts)
	if err != nil {
		panic(err)
	}
	return b
}

func (d *diskImage) writable() bool {
	return d.save != nil
}

func (d *diskImage) commit() error {
	if d.save == nil {
		return errWriteProtect
	}
	return d.save(d.data)
}

func (d *diskImage) dosName() string {
	return d.g.dosName
}

// header returns the name of the disk, the id and the DOS type
func (d *diskImage) header() ([]byte, []byte, []byte) {
	h := d.mustBlock(d.g.header)
	return h[d.g.nameOffset : d.g.nameOffset+nameLength],
		h[d.g.idOffset : d.g.idOffset+2],
		h[d.g.dosTypeOffset : d.g.dosTypeOffset+2]
}

// BAM

// bamEntry returns the bytes of the BAM for a track: the free count and
// the bitmap
func (d *diskImage) bamEntry(track int) []byte {
	if d.g.bamTracks == 0 {
		b := d.mustBlock(d.g.bam[0])
		offset := d.g.bamFirstEntry + (track-1)*d.g.bamEntrySize
		return b[offset : offset+d.g.bamEntrySize]
	}
	index := (track - 1) / d.g.bamTracks
	b := d.mustBlock(d.g.bam[index])
	offset := d.g.bamFirstEntry + (track-1-index*d.g.bamTracks)*d.g.bamEntrySize
	return b[offset : offset+d.g.bamEntrySize]
}

func (d *diskImage) isFree(ts trackSector) bool {
	e := d.bamEntry(ts.track)
	return e[1+ts.sector/8]&(1<<(ts.sector%8)) != 0
}

func (d *diskImage) setFree(ts trackSector, free bool) {
	e := d.bamEntry(ts.track)
	mask := byte(1 << (ts.sector % 8))
	was := e[1+ts.sector/8]&mask != 0
	if was == free {
		return
	}
	if free {
		e[1+ts.sector/8] |= mask
		e[0]++
	} else {
		e[1+ts.sector/8] &^= mask
		e[0]--
	}
}

func (d *diskImage) blocksFree() int {
	free := 0
	for t := 1; t <= d.g.tracks; t++ {
		if !d.g.isReserved(t) {
			free += int(d.bamEntry(t)[0])
		}
	}
	return free
}

// allocate takes a free block for a file, from the tracks closest to the
// directory
func (d *diskImage) allocate() (trackSector, error) {
	for distance := 1; distance <= d.g.tracks; distance++ {
		for _, t := range []int{d.g.dirTrack - distance, d.g.dirTrack + distance} {
			if t < 1 || t > d.g.tracks || d.g.isReserved(t) || d.bamEntry(t)[0] == 0 {
				continue
			}
			for s := 0; s < d.g.sectors(t); s++ {
				ts := trackSector{t, s}
				if d.isFree(ts) {
					d.setFree(ts, false)
					return ts, nil
				}
			}
		}
	}
	return trackSector{}, errDiskFull
}

// Formatting

func padded(text []byte, length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = padding
	}
	copy(b, text)
	return b
}

func (d *diskImage) formatBlocks(name, id []byte) {
	for i := range d.data {
		d.data[i] = 0
	}
	g := d.g

	// The BAM sectors
	for i, ts := range g.bam {
		b := d.mustBlock(ts)
		next := trackSector{g.dirTrack, 1}
		if g.name == "D64" {
			next = trackSector{18, 1}
		} else if i+1 < len(g.bam) {
			next = g.bam[i+1]
		}
		b[0], b[1] = byte(next.track), byte(next.sector)
		b[bamFormatOffset] = g.format
		if g.bamTracks != 0 {
			first := i*g.bamTracks + 1
			b[4] = byte(first)
			b[5] = byte(min(first+g.bamTracks, g.tracks+1))
		}
	}
	for t := 1; t <= g.tracks; t++ {
		sectors := g.sectors(t)
		e := d.bamEntry(t)
		e[0] = byte(sectors)
		for s := 0; s < sectors; s++ {
			e[1+s/8] |= 1 << (s % 8)
		}
	}

	// The header
	h := d.mustBlock(g.header)
	if g.name != "D64" {
		h[0], h[1] = byte(g.bam[0].track), byte(g.bam[0].sector)
		h[bamFormatOffset] = g.format
	}
	copy(h[g.nameOffset:], padded(name, nameLength))
	h[g.nameOffset+nameLength] = padding
	h[g.nameOffset+nameLength+1] = padding
	copy(h[g.idOffset:], padded(id, 2))
	h[g.idOffset+2] = padding
	copy(h[g.dosTypeOffset:], g.dosType)
	for i := g.dosTypeOffset + 2; i < g.dosTypeOffset+6; i++ {
		h[i] = padding
	}

	// The first directory sector
	dir := d.mustBlock(trackSector{g.dirTrack, 1})
	dir[0], dir[1] = 0, 0xff

	d.setFree(g.header, false)
	for _, ts := range g.bam {
		d.setFree(ts, false)
	}
	d.setFree(trackSector{g.dirTrack, 1}, false)
}

func (d *diskImage) format(name, id []byte) error {
	if !d.writable() {
		return errWriteProtect
	}
	if id == nil {
		// Without an id the directory is cleared and the id kept
		_, oldID, _ := d.header()
		id = append([]byte(nil), oldID...)
	}
	d.formatBlocks(name, id)
	return d.commit()
}

// Directory

type dirSlot struct {
	ts     trackSector
	offset int
}

func (d *diskImage) entry(slot dirSlot) []byte {
	b := d.mustBlock(slot.ts)
	return b[slot.offset : slot.offset+dirEntrySize]
}

// slots returns all the directory entries, used or not
func (d *diskImage) slots() []dirSlot {
	var slots []dirSlot
	ts := trackSector{d.g.dirTrack, 1}
	for i := 0; i < maxChainBlocks; i++ {
		b, err := d.block(ts)
		if err != nil {
			break
		}
		for e := 0; e < dirEntries; e++ {
			slots = append(slots, dirSlot{ts, e * dirEntrySize})
		}
		if b[0] == 0 {
			break
		}
		ts = trackSector{int(b[0]), int(b[1])}
	}
	return slots
}

func (d *diskImage) entries() []dirEntry {
	var entries []dirEntry
	for _, slot := range d.slots() {
		e := d.entry(slot)
		if e[dirEntryType] == 0 {
			continue
		}
		entries = append(entries, dirEntry{
			name:   trimName(e[dirEntryName : dirEntryName+nameLength]),
			typ:    e[dirEntryType],
			blocks: int(e[dirEntryBlocks]) | int(e[dirEntryBlocks+1])<<8,
		})
	}
	return entries
}

func (d *diskImage) find(name []byte) (dirSlot, bool) {
	for _, slot := range d.slots() {
		e := d.entry(slot)
		if e[dirEntryType]&fileTypeMask != 0 && e[dirEntryType]&fileTypeClosed != 0 &&
			string(trimName(e[dirEntryName:dirEntryName+nameLength])) == string(name) {
			return slot, true
		}
	}
	return dirSlot{}, false
}

// newSlot returns a free directory entry, adding a block to the directory
// if needed
func (d *diskImage) newSlot() (dirSlot, error) {
	slots := d.slots()
	for _, slot := range slots {
		if d.entry(slot)[dirEntryType] == 0 {
			return slot, nil
		}
	}

	// A new block on the directory track
	last := slots[len(slots)-1].ts
	for s := 0; s < d.g.sectors(d.g.dirTrack); s++ {
		ts := trackSector{d.g.dirTrack, s}
		if d.isFree(ts) {
			d.setFree(ts, false)
			b := d.mustBlock(ts)
			for i := range b {
				b[i] = 0
			}
			b[1] = 0xff
			lb := d.mustBlock(last)
			lb[0], lb[1] = byte(ts.track), byte(ts.sector)
			return dirSlot{ts, 0}, nil
		}
	}
	return dirSlot{}, errDirectoryFull
}

// Files

func (d *diskImage) chain(first trackSector) ([]trackSector, error) {
	var blocks []trackSector
	ts := first
	for i := 0; i < maxChainBlocks; i++ {
		b, err := d.block(ts)
		if err != nil {
			return nil, errBadChain
		}
		blocks = append(blocks, ts)
		if b[0] == 0 {
			return blocks, nil
		}
		ts = trackSector{int(b[0]), int(b[1])}
	}
	return nil, errBadChain
}

func (d *diskImage) readFile(name []byte) ([]byte, byte, error) {
	slot, ok := d.find(name)
	if !ok {
		return nil, 0, errFileNotFound
	}
	e := d.entry(slot)
	blocks, err := d.chain(trackSector{int(e[dirEntryFirst]), int(e[dirEntryFirst+1])})
	if err != nil {
		return nil, 0, err
	}
	var data []byte
	for _, ts := range blocks {
		b := d.mustBlock(ts)
		if b[0] == 0 {
			last := int(b[1])
			if last >= 2 {
				data = append(data, b[2:last+1]...)
			}
		} else {
			data = append(data, b[2:]...)
		}
	}
	return data, e[dirEntryType] & fileTypeMask, nil
}

func (d *diskImage) writeFile(name []byte, typ byte, data []byte, replace bool) error {
	if !d.writable() {
		return errWriteProtect
	}
	if _, exists := d.find(name); exists {
		if !replace {
			return errFileExists
		}
		if err := d.scratch(name); err != nil {
			return err
		}
	}

	needed := max(1, (len(data)+blockDataSize-1)/blockDataSize)
	if needed > d.blocksFree() {
		return errDiskFull
	}
	slot, err := d.newSlot()
	if err != nil {
		return err
	}

	blocks := make([]trackSector, needed)
	for i := range blocks {
		blocks[i], err = d.allocate()
		if err != nil {
			return err
		}
	}
	for i, ts := range blocks {
		b := d.mustBlock(ts)
		for j := range b {
			b[j] = 0
		}
		chunk := data[min(i*blockDataSize, len(data)):min((i+1)*blockDataSize, len(data))]
		copy(b[2:], chunk)
		if i+1 < len(blocks) {
			b[0], b[1] = byte(blocks[i+1].track), byte(blocks[i+1].sector)
		} else {
			b[0], b[1] = 0, byte(1+len(chunk))
		}
	}

	e := d.entry(slot)
	for i := 2; i < dirEntrySize; i++ {
		e[i] = 0
	}
	e[dirEntryType] = fileTypeClosed | typ
	e[dirEntryFirst], e[dirEntryFirst+1] = byte(blocks[0].track), byte(blocks[0].sector)
	copy(e[dirEntryName:], padded(name, nameLength))
	e[dirEntryBlocks], e[dirEntryBlocks+1] = byte(needed), byte(needed>>8)
	return d.commit()
}

func (d *diskImage) scratch(name []byte) error {
	if !d.writable() {
		return errWriteProtect
	}
	slot, ok := d.find(name)
	if !ok {
		return errFileNotFound
	}
	e := d.entry(slot)
	if blocks, err := d.chain(trackSector{int(e[dirEntryFirst]), int(e[dirEntryFirst+1])}); err == nil {
		for _, ts := range blocks {
			d.setFree(ts, true)
		}
	}
	e[dirEntryType] = 0
	return d.commit()
}

func (d *diskImage) rename(old, name []byte) error {
	if !d.writable() {
		return errWriteProtect
	}
	if _, exists := d.find(name); exists {
		return errFileExists
	}
	slot, ok := d.find(old)
	if !ok {
		return errFileNotFound
	}
	copy(d.entry(slot)[dirEntryName:], padded(name, nameLength))
	return d.commit()
}

// trimName removes the padding of a name
func trimName(name []byte) []byte {
	end := len(name)
	for end > 0 && name[end-1] == padding {
		end--
	}
	return append([]byte(nil), name[:end]...)
}
