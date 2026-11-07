package cbmdos

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

/*
A folder of the host as the disk in a drive. The files are named with the
name on the PET and the type as extension: "hello.prg", "data.seq".

The names are converted between PETSCII and the host: the unshifted letters,
uppercase on the PET in the graphics mode, are the lowercase letters of the
host, and the shifted letters the uppercase ones. The characters not allowed
in names on the host are replaced by '_'.
*/

const folderBlocksFree = 9999

type folder struct {
	path string
}

func newFolder(path string) (*folder, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New(path + " is not a folder")
	}
	return &folder{path}, nil
}

func (f *folder) dosName() string {
	return "IZPET DOS V2.5 FOLDER"
}

func (f *folder) writable() bool {
	return true
}

func (f *folder) header() ([]byte, []byte, []byte) {
	name := hostToPetscii(filepath.Base(f.path))
	if len(name) > nameLength {
		name = name[:nameLength]
	}
	return padded(name, nameLength), []byte("IZ"), []byte("2C")
}

func (f *folder) blocksFree() int {
	return folderBlocksFree
}

func fileTypeForExtension(ext string) (byte, bool) {
	switch strings.ToLower(ext) {
	case ".prg":
		return fileTypePRG, true
	case ".seq":
		return fileTypeSEQ, true
	case ".usr":
		return fileTypeUSR, true
	}
	return 0, false
}

func (f *folder) files() map[string]os.DirEntry {
	files := make(map[string]os.DirEntry)
	list, err := os.ReadDir(f.path)
	if err != nil {
		return files
	}
	for _, e := range list {
		if !e.IsDir() {
			files[e.Name()] = e
		}
	}
	return files
}

func (f *folder) entries() []dirEntry {
	var entries []dirEntry
	files := f.files()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ext := filepath.Ext(name)
		typ, ok := fileTypeForExtension(ext)
		if !ok {
			continue
		}
		size := int64(0)
		if info, err := files[name].Info(); err == nil {
			size = info.Size()
		}
		petName := hostToPetscii(strings.TrimSuffix(name, ext))
		if len(petName) > nameLength {
			continue
		}
		entries = append(entries, dirEntry{
			name:   petName,
			typ:    fileTypeClosed | typ,
			blocks: int((size + blockDataSize - 1) / blockDataSize),
		})
	}
	return entries
}

// hostName returns the name of the file on the host for a PET name, the
// existing one if there is one with any case of the extension
func (f *folder) hostName(name []byte) (string, byte, bool) {
	for host := range f.files() {
		ext := filepath.Ext(host)
		typ, ok := fileTypeForExtension(ext)
		if ok && string(hostToPetscii(strings.TrimSuffix(host, ext))) == string(name) {
			return host, typ, true
		}
	}
	return "", 0, false
}

func (f *folder) readFile(name []byte) ([]byte, byte, error) {
	host, typ, ok := f.hostName(name)
	if !ok {
		return nil, 0, errFileNotFound
	}
	data, err := os.ReadFile(filepath.Join(f.path, host))
	return data, typ, err
}

func (f *folder) writeFile(name []byte, typ byte, data []byte, replace bool) error {
	if host, _, exists := f.hostName(name); exists {
		if !replace {
			return errFileExists
		}
		if err := os.Remove(filepath.Join(f.path, host)); err != nil {
			return err
		}
	}
	host := petsciiToHost(name) + "." + strings.ToLower(fileTypeNames()[typ&fileTypeMask])
	return os.WriteFile(filepath.Join(f.path, host), data, 0644)
}

func (f *folder) scratch(name []byte) error {
	host, _, ok := f.hostName(name)
	if !ok {
		return errFileNotFound
	}
	return os.Remove(filepath.Join(f.path, host))
}

func (f *folder) rename(old, name []byte) error {
	if _, _, exists := f.hostName(name); exists {
		return errFileExists
	}
	host, typ, ok := f.hostName(old)
	if !ok {
		return errFileNotFound
	}
	newHost := petsciiToHost(name) + "." + strings.ToLower(fileTypeNames()[typ])
	return os.Rename(filepath.Join(f.path, host), filepath.Join(f.path, newHost))
}

func (f *folder) format(name, id []byte) error {
	return errors.New("a folder can't be formatted")
}

// petsciiToHost converts a PET file name to a name for the host
func petsciiToHost(name []byte) string {
	var sb strings.Builder
	for _, c := range name {
		switch {
		case c >= 0x41 && c <= 0x5a:
			sb.WriteByte(c - 0x41 + 'a')
		case c >= 0xc1 && c <= 0xda:
			sb.WriteByte(c - 0xc1 + 'A')
		case c >= 0x20 && c < 0x41 && !strings.ContainsRune("/\\:*?\"<>|", rune(c)):
			sb.WriteByte(c)
		default:
			sb.WriteByte('_')
		}
	}
	return sb.String()
}

// hostToPetscii converts a name of the host to a PET file name
func hostToPetscii(name string) []byte {
	var b []byte
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b = append(b, byte(r-'a')+0x41)
		case r >= 'A' && r <= 'Z':
			b = append(b, byte(r-'A')+0xc1)
		case r >= 0x20 && r < 0x41:
			b = append(b, byte(r))
		case r == '_':
			b = append(b, 0x5f) // The left arrow
		default:
			b = append(b, '?')
		}
	}
	return b
}
