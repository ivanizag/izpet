// Package cbmdos is a Commodore disk drive emulated at the level of the
// commands of its DOS, on disk images or folders of the host. It knows
// nothing of the PET.
package cbmdos

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/ivanizag/izpet/ieee488"
)

/*
A Commodore disk drive, 4040, 8050 or 8250, as it answers on the IEEE-488
bus. The drive has 16 channels, opened with a name:

	0      LOAD, reads a program
	1      SAVE, writes a program
	2-14   data files, "0:NAME,S,R" to read a SEQ file, ",W" to write, ",A"
	       to append. "@0:NAME" replaces an existing file.
	15     the command channel. A name or the data sent to it are commands,
	       reading it returns the error status, "00, OK,00,00".

The name "$" reads the directory as a BASIC program, as LOAD"$",8 and the
DIRECTORY of BASIC 4 want it. "$0:A*" lists the files that match.

The names can have the wildcards * and ?, and a drive number, "0:" or "1:".

Commands of the command channel:

	I      initialize                  V      validate
	N0:NAME,ID  format                 N0:NAME  clear the directory
	S0:NAME     scratch, delete        R0:NEW=OLD  rename
	C0:NEW=0:OLD,0:OLD2  copy and concatenate
	UJ     reset

The files are read whole when opened and written whole when closed. There
are no relative files.

A channel with nothing to read does not answer, the PET times out and
reports the file not found.
*/

// File types
const (
	fileTypeDEL = 0
	fileTypeSEQ = 1
	fileTypePRG = 2
	fileTypeUSR = 3
	fileTypeREL = 4
)

func fileTypeNames() []string {
	return []string{"DEL", "SEQ", "PRG", "USR", "REL", "???", "???", "???"}
}

type dirEntry struct {
	name   []byte
	typ    byte
	blocks int
}

// fileSystem is what is in a drive: a disk image or a folder
type fileSystem interface {
	dosName() string
	writable() bool
	header() (name, id, dosType []byte)
	entries() []dirEntry
	blocksFree() int
	readFile(name []byte) ([]byte, byte, error)
	writeFile(name []byte, typ byte, data []byte, replace bool) error
	scratch(name []byte) error
	rename(old, name []byte) error
	format(name, id []byte) error
}

// Error codes of the DOS
const (
	statusOK             = 0
	statusScratched      = 1
	statusWriteProtect   = 26
	statusSyntax         = 31
	statusNoFileGiven    = 34
	statusFileNotFound   = 62
	statusFileExists     = 63
	statusFileType       = 64
	statusNoChannel      = 70
	statusDiskFull       = 72
	statusDOSVersion     = 73
	statusDriveNotReady  = 74
	commandChannel       = 15
	loadChannel          = 0
	saveChannel          = 1
	secondaryOpen        = 0xf0
	secondaryClose       = 0xe0
	secondaryChannelMask = 0x0f
	maxDrives            = 2
	basicStart           = 0x0401
)

func statusMessages() map[int]string {
	return map[int]string{
		statusOK:            " OK",
		statusScratched:     " FILES SCRATCHED",
		statusWriteProtect:  "WRITE PROTECT ON",
		statusSyntax:        "SYNTAX ERROR",
		statusNoFileGiven:   "SYNTAX ERROR",
		statusFileNotFound:  "FILE NOT FOUND",
		statusFileExists:    "FILE EXISTS",
		statusFileType:      "FILE TYPE MISMATCH",
		statusNoChannel:     "NO CHANNEL",
		statusDiskFull:      "DISK FULL",
		statusDriveNotReady: "DRIVE NOT READY",
	}
}

type channel struct {
	data    []byte // To read
	pos     int
	writing bool
	fs      fileSystem
	name    []byte
	typ     byte
	replace bool
}

// Drive is a disk drive on the IEEE-488 bus
type Drive struct {
	*ieee488.Peripheral

	drives   [maxDrives]fileSystem
	channels [16]*channel

	listenChannel int
	talkChannel   int
	opening       bool
	openName      []byte
	command       []byte

	status     []byte // The status message being read
	statusPos  int
	statusRead bool
	trace      bool
}

// NewDrive returns a drive with an address, usually 8
func NewDrive(address uint8, trace bool) *Drive {
	d := &Drive{trace: trace}
	d.Peripheral = ieee488.NewPeripheral(address, d)
	d.Reset()
	return d
}

// Reset resets the drive, as after power on
func (d *Drive) Reset() {
	d.Peripheral.Reset()
	d.channels = [16]*channel{}
	d.listenChannel = -1
	d.talkChannel = -1
	d.opening = false
	d.command = nil
	d.setStatus(statusDOSVersion, 0, 0)
}

// Insert puts a disk image or a folder of the host in one of the drives. A
// file that does not exist is created as a blank image of the type of its
// extension, D64, D80 or D82, when written.
func (d *Drive) Insert(drive int, path string, data []byte, writable bool) error {
	if drive < 0 || drive >= maxDrives {
		return fmt.Errorf("there is no drive %v", drive)
	}
	fs, err := openFileSystem(path, data, writable)
	if err != nil {
		return err
	}
	d.drives[drive] = fs
	return nil
}

// Eject removes the disk of a drive
func (d *Drive) Eject(drive int) {
	if drive >= 0 && drive < maxDrives {
		d.drives[drive] = nil
	}
}

// DiskName returns a description of the disk in a drive, empty if none
func (d *Drive) DiskName(drive int) string {
	if drive < 0 || drive >= maxDrives || d.drives[drive] == nil {
		return ""
	}
	name, id, _ := d.drives[drive].header()
	return fmt.Sprintf("%s %s", petsciiToHost(trimName(name)), petsciiToHost(id))
}

func openFileSystem(path string, data []byte, writable bool) (fileSystem, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return newFolder(path)
	}

	save := func(image []byte) error {
		return os.WriteFile(path, image, 0644)
	}
	if !writable {
		save = nil
	}
	if data == nil {
		// A new disk
		if save == nil {
			return nil, errors.New("can't create the disk image " + path)
		}
		return newBlankImage(geometryForName(path), save), nil
	}
	return newDiskImage(data, save)
}

// Status returns the error status of the drive, as read on the command
// channel
func (d *Drive) Status() string {
	return string(bytes.TrimRight(d.status, "\r"))
}

func (d *Drive) setStatus(code int, track int, sector int) {
	message := statusMessages()[code]
	if code == statusDOSVersion {
		message = d.dosName()
	}
	d.status = []byte(fmt.Sprintf("%02d,%s,%02d,%02d\r", code, message, track, sector))
	d.statusPos = 0
	if d.trace {
		fmt.Printf("[drive %v] %s\n", d.Address(), d.Status())
	}
}

func (d *Drive) dosName() string {
	if d.drives[0] != nil {
		return d.drives[0].dosName()
	}
	return "CBM DOS V2.5 8050"
}

func (d *Drive) setError(err error) {
	switch {
	case err == nil:
		d.setStatus(statusOK, 0, 0)
	case errors.Is(err, errFileNotFound):
		d.setStatus(statusFileNotFound, 0, 0)
	case errors.Is(err, errFileExists):
		d.setStatus(statusFileExists, 0, 0)
	case errors.Is(err, errDiskFull), errors.Is(err, errDirectoryFull):
		d.setStatus(statusDiskFull, 0, 0)
	case errors.Is(err, errWriteProtect):
		d.setStatus(statusWriteProtect, 0, 0)
	default:
		d.setStatus(statusDriveNotReady, 0, 0)
	}
}

// The ieee488.Handler

// Listen is told the secondary address while listening
func (d *Drive) Listen(secondary uint8) {
	ch := int(secondary & secondaryChannelMask)
	switch secondary & 0xf0 {
	case secondaryOpen:
		d.opening = true
		d.openName = nil
		d.listenChannel = ch
	case secondaryClose:
		d.close(ch)
		d.listenChannel = -1
	default:
		d.listenChannel = ch
	}
}

// Talk is told the secondary address while talking
func (d *Drive) Talk(secondary uint8) {
	d.talkChannel = int(secondary & secondaryChannelMask)
}

// Receive gets a byte while listening
func (d *Drive) Receive(b uint8, eoi bool) {
	switch {
	case d.opening:
		d.openName = append(d.openName, b)
	case d.listenChannel == commandChannel:
		d.command = append(d.command, b)
	case d.listenChannel >= 0:
		if c := d.channels[d.listenChannel]; c != nil && c.writing {
			c.data = append(c.data, b)
		}
	}
}

// Unlisten ends an OPEN or a command
func (d *Drive) Unlisten() {
	if d.opening {
		d.opening = false
		d.open(d.listenChannel, d.openName)
	} else if d.listenChannel == commandChannel && len(d.command) > 0 {
		d.execute(d.command)
		d.command = nil
	}
	d.listenChannel = -1
}

// Untalk ends sending
func (d *Drive) Untalk() {
	d.talkChannel = -1
}

// Peek returns the byte to send
func (d *Drive) Peek() (uint8, bool, bool) {
	if d.talkChannel == commandChannel {
		if d.statusPos >= len(d.status) {
			d.statusPos = 0
		}
		return d.status[d.statusPos], d.statusPos == len(d.status)-1, true
	}
	if d.talkChannel < 0 {
		return 0, false, false
	}
	c := d.channels[d.talkChannel]
	if c == nil || c.writing || c.pos >= len(c.data) {
		return 0, false, false
	}
	return c.data[c.pos], c.pos == len(c.data)-1, true
}

// Next moves to the next byte to send
func (d *Drive) Next() {
	if d.talkChannel == commandChannel {
		d.statusPos++
		if d.statusPos >= len(d.status) {
			// Once read, the status is cleared
			d.setStatus(statusOK, 0, 0)
		}
		return
	}
	if d.talkChannel >= 0 {
		if c := d.channels[d.talkChannel]; c != nil {
			c.pos++
		}
	}
}

// Files

// parsedName is a file name with its drive and options
type parsedName struct {
	drive   int
	name    []byte
	typ     byte // 0 for any
	mode    byte // 'R', 'W', 'A', or 0
	replace bool
}

func parseName(raw []byte) parsedName {
	p := parsedName{drive: 0}
	raw = bytes.TrimRight(raw, "\r\xa0")
	if len(raw) > 0 && raw[0] == '@' {
		p.replace = true
		raw = raw[1:]
	}
	if colon := bytes.IndexByte(raw, ':'); colon >= 0 {
		if colon == 1 && (raw[0] == '0' || raw[0] == '1') {
			p.drive = int(raw[0] - '0')
		}
		raw = raw[colon+1:]
	}
	parts := bytes.Split(raw, []byte(","))
	p.name = parts[0]
	for _, opt := range parts[1:] {
		if len(opt) == 0 {
			continue
		}
		switch opt[0] {
		case 'S':
			p.typ = fileTypeSEQ
		case 'P':
			p.typ = fileTypePRG
		case 'U':
			p.typ = fileTypeUSR
		case 'L':
			p.typ = fileTypeREL
		case 'R', 'W', 'A', 'M':
			p.mode = opt[0]
		}
	}
	return p
}

// matches returns true if a name matches a pattern with * and ?
func matches(pattern, name []byte) bool {
	for i := 0; i < len(pattern); i++ {
		switch {
		case pattern[i] == '*':
			return true
		case i >= len(name):
			return false
		case pattern[i] != '?' && pattern[i] != name[i]:
			return false
		}
	}
	return len(pattern) == len(name)
}

func hasWildcards(name []byte) bool {
	return bytes.ContainsAny(name, "*?")
}

// findFile returns the drive and entry of the first file that matches
func (d *Drive) findFile(p parsedName, anyDrive bool) (fileSystem, dirEntry, bool) {
	drives := []int{p.drive}
	if anyDrive && p.drive == 0 {
		drives = []int{0, 1}
	}
	for _, drive := range drives {
		fs := d.drives[drive]
		if fs == nil {
			continue
		}
		for _, e := range fs.entries() {
			if e.typ&fileTypeClosed == 0 || e.typ&fileTypeMask == fileTypeDEL {
				continue
			}
			if p.typ != 0 && e.typ&fileTypeMask != p.typ {
				continue
			}
			if matches(p.name, e.name) {
				return fs, e, true
			}
		}
	}
	return nil, dirEntry{}, false
}

func (d *Drive) open(ch int, raw []byte) {
	if d.trace {
		fmt.Printf("[drive %v] open %v %q\n", d.Address(), ch, petsciiToHost(raw))
	}
	if ch == commandChannel {
		if len(raw) > 0 {
			d.execute(raw)
		}
		return
	}
	d.channels[ch] = nil
	if len(raw) == 0 {
		d.setStatus(statusNoFileGiven, 0, 0)
		return
	}

	if raw[0] == '$' {
		d.openDirectory(ch, raw[1:])
		return
	}

	p := parseName(raw)
	if p.mode == 0 {
		p.mode = 'R'
		if ch == saveChannel {
			p.mode = 'W'
		}
	}
	if p.typ == 0 && (ch == loadChannel || ch == saveChannel) {
		p.typ = fileTypePRG
	}

	fs := d.drives[p.drive]
	if fs == nil {
		d.setStatus(statusDriveNotReady, 0, 0)
		return
	}

	switch p.mode {
	case 'W':
		if len(p.name) == 0 || hasWildcards(p.name) {
			d.setStatus(statusSyntax, 0, 0)
			return
		}
		if !fs.writable() {
			d.setStatus(statusWriteProtect, 0, 0)
			return
		}
		if _, _, exists := d.findFile(parsedName{drive: p.drive, name: p.name}, false); exists && !p.replace {
			d.setStatus(statusFileExists, 0, 0)
			return
		}
		if p.typ == 0 {
			p.typ = fileTypeSEQ
		}
		d.channels[ch] = &channel{writing: true, fs: fs, name: p.name, typ: p.typ, replace: p.replace}
		d.setStatus(statusOK, 0, 0)

	case 'A':
		data, typ, err := fs.readFile(p.name)
		if err != nil {
			d.setError(err)
			return
		}
		d.channels[ch] = &channel{writing: true, fs: fs, name: p.name, typ: typ,
			replace: true, data: data}
		d.setStatus(statusOK, 0, 0)

	default:
		fs, e, ok := d.findFile(p, true)
		if !ok {
			d.setStatus(statusFileNotFound, 0, 0)
			return
		}
		data, _, err := fs.readFile(e.name)
		if err != nil {
			d.setError(err)
			return
		}
		d.channels[ch] = &channel{data: data}
		d.setStatus(statusOK, 0, 0)
	}
}

func (d *Drive) close(ch int) {
	if ch == commandChannel {
		// Closing the command channel closes them all
		for i := range d.channels {
			if i != commandChannel {
				d.close(i)
			}
		}
		return
	}
	c := d.channels[ch]
	d.channels[ch] = nil
	if c == nil || !c.writing {
		return
	}
	err := c.fs.writeFile(c.name, c.typ, c.data, c.replace)
	if d.trace {
		fmt.Printf("[drive %v] write %q, %v bytes: %v\n", d.Address(), petsciiToHost(c.name), len(c.data), err)
	}
	d.setError(err)
}

// openDirectory makes the directory as a BASIC program for a channel
func (d *Drive) openDirectory(ch int, raw []byte) {
	p := parseName(raw)
	if len(raw) > 0 && (raw[0] == '0' || raw[0] == '1') && bytes.IndexByte(raw, ':') < 0 {
		p.drive = int(raw[0] - '0')
		p.name = nil
	}
	fs := d.drives[p.drive]
	if fs == nil {
		d.setStatus(statusDriveNotReady, 0, 0)
		return
	}
	d.channels[ch] = &channel{data: directoryListing(fs, p.drive, p.name, p.typ)}
	d.setStatus(statusOK, 0, 0)
}

/*
directoryListing returns the directory as a BASIC program loaded at $0401:
a line per file with the blocks as line number, a first line with the name
and id of the disk in reverse, and a last one with the free blocks.
*/
func directoryListing(fs fileSystem, drive int, pattern []byte, typ byte) []byte {
	var lines [][]byte
	var numbers []int

	name, id, dosType := fs.header()
	header := []byte{0x12, '"'}
	header = append(header, padded(trimName(name), nameLength)...)
	header = append(header, '"', ' ')
	header = append(header, id...)
	header = append(header, ' ')
	header = append(header, dosType...)
	for i := range header {
		if header[i] == padding {
			header[i] = ' '
		}
	}
	lines = append(lines, header)
	numbers = append(numbers, drive)

	for _, e := range fs.entries() {
		if len(pattern) > 0 && !matches(pattern, e.name) {
			continue
		}
		if typ != 0 && e.typ&fileTypeMask != typ {
			continue
		}
		var line []byte
		switch {
		case e.blocks < 10:
			line = append(line, "   "...)
		case e.blocks < 100:
			line = append(line, "  "...)
		case e.blocks < 1000:
			line = append(line, ' ')
		}
		line = append(line, '"')
		line = append(line, e.name...)
		line = append(line, '"')
		for i := len(e.name); i < nameLength; i++ {
			line = append(line, ' ')
		}
		if e.typ&fileTypeClosed == 0 {
			line = append(line, '*')
		} else {
			line = append(line, ' ')
		}
		line = append(line, fileTypeNames()[e.typ&fileTypeMask]...)
		if e.typ&fileTypeLocked != 0 {
			line = append(line, '<')
		}
		lines = append(lines, line)
		numbers = append(numbers, e.blocks)
	}

	lines = append(lines, []byte("BLOCKS FREE.             "))
	numbers = append(numbers, min(fs.blocksFree(), 0xffff))

	program := []byte{byte(basicStart & 0xff), byte(basicStart >> 8)}
	address := basicStart
	for i, line := range lines {
		address += 4 + len(line) + 1
		program = append(program, byte(address), byte(address>>8),
			byte(numbers[i]), byte(numbers[i]>>8))
		program = append(program, line...)
		program = append(program, 0)
	}
	return append(program, 0, 0)
}

// Commands

func (d *Drive) execute(raw []byte) {
	cmd := bytes.TrimRight(raw, "\r\xa0 ")
	if d.trace {
		fmt.Printf("[drive %v] command %q\n", d.Address(), petsciiToHost(cmd))
	}
	if len(cmd) == 0 {
		return
	}

	switch cmd[0] {
	case 'I', 'V':
		d.setStatus(statusOK, 0, 0)
	case 'U':
		if len(cmd) > 1 && (cmd[1] == 'J' || cmd[1] == ':' || cmd[1] == '9' || cmd[1] == 'I') {
			d.Reset()
		} else {
			d.setStatus(statusSyntax, 0, 0)
		}
	case 'N':
		d.commandNew(cmd)
	case 'S':
		d.commandScratch(cmd)
	case 'R':
		d.commandRename(cmd)
	case 'C':
		d.commandCopy(cmd)
	default:
		d.setStatus(statusSyntax, 0, 0)
	}
}

// argument returns the drive and the text after the colon of a command
func argument(cmd []byte) (int, []byte, bool) {
	colon := bytes.IndexByte(cmd, ':')
	if colon < 0 {
		return 0, nil, false
	}
	drive := 0
	if colon > 0 && (cmd[colon-1] == '1') {
		drive = 1
	}
	return drive, cmd[colon+1:], true
}

func (d *Drive) commandNew(cmd []byte) {
	drive, arg, ok := argument(cmd)
	if !ok {
		d.setStatus(statusSyntax, 0, 0)
		return
	}
	fs := d.drives[drive]
	if fs == nil {
		d.setStatus(statusDriveNotReady, 0, 0)
		return
	}
	parts := bytes.SplitN(arg, []byte(","), 2)
	var id []byte
	if len(parts) == 2 {
		id = parts[1]
	}
	d.setError(fs.format(parts[0], id))
}

func (d *Drive) commandScratch(cmd []byte) {
	drive, arg, ok := argument(cmd)
	if !ok {
		d.setStatus(statusSyntax, 0, 0)
		return
	}
	count := 0
	for _, pattern := range bytes.Split(arg, []byte(",")) {
		p := parseName(pattern)
		if bytes.IndexByte(pattern, ':') < 0 {
			p.drive = drive
		}
		fs := d.drives[p.drive]
		if fs == nil {
			d.setStatus(statusDriveNotReady, 0, 0)
			return
		}
		for _, e := range fs.entries() {
			if matches(p.name, e.name) {
				if err := fs.scratch(e.name); err != nil {
					d.setError(err)
					return
				}
				count++
			}
		}
	}
	d.setStatus(statusScratched, count, 0)
}

// splitAssignment splits "0:NEW=0:OLD,0:OLD2" in the new name and the old
// ones
func splitAssignment(cmd []byte) (parsedName, []parsedName, bool) {
	eq := bytes.IndexByte(cmd, '=')
	if eq < 0 {
		return parsedName{}, nil, false
	}
	target := parseName(cmd[1:eq])
	var sources []parsedName
	for _, s := range bytes.Split(cmd[eq+1:], []byte(",")) {
		sources = append(sources, parseName(s))
	}
	return target, sources, true
}

func (d *Drive) commandRename(cmd []byte) {
	target, sources, ok := splitAssignment(cmd)
	if !ok || len(sources) != 1 {
		d.setStatus(statusSyntax, 0, 0)
		return
	}
	fs := d.drives[target.drive]
	if fs == nil {
		d.setStatus(statusDriveNotReady, 0, 0)
		return
	}
	d.setError(fs.rename(sources[0].name, target.name))
}

func (d *Drive) commandCopy(cmd []byte) {
	target, sources, ok := splitAssignment(cmd)
	if !ok || len(target.name) == 0 {
		// The copy of a whole disk is not supported
		d.setStatus(statusSyntax, 0, 0)
		return
	}
	dst := d.drives[target.drive]
	if dst == nil {
		d.setStatus(statusDriveNotReady, 0, 0)
		return
	}
	var data []byte
	typ := byte(0)
	for _, s := range sources {
		fs, e, found := d.findFile(s, false)
		if !found {
			d.setStatus(statusFileNotFound, 0, 0)
			return
		}
		content, t, err := fs.readFile(e.name)
		if err != nil {
			d.setError(err)
			return
		}
		if typ == 0 {
			typ = t
		}
		data = append(data, content...)
	}
	d.setError(dst.writeFile(target.name, typ, data, false))
}
