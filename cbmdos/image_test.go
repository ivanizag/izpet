package cbmdos

import (
	"bytes"
	"testing"
)

func TestBlankImages(t *testing.T) {
	cases := []struct {
		g      *geometry
		size   int
		free   int
		header string
	}{
		{geometryD64(), 174848, 664, "2A"},
		{geometryD80(), 533248, 2052, "2C"},
		{geometryD82(), 1066496, 4133, "2C"},
	}
	for _, c := range cases {
		d := newBlankImage(c.g, func([]byte) error { return nil })
		if len(d.data) != c.size {
			t.Errorf("%v: size %v, expected %v", c.g.name, len(d.data), c.size)
		}
		if free := d.blocksFree(); free != c.free {
			t.Errorf("%v: %v blocks free, expected %v", c.g.name, free, c.free)
		}
		name, id, dosType := d.header()
		if string(trimName(name)) != "BLANK" || string(id) != "IZ" || string(dosType) != c.header {
			t.Errorf("%v: header %q %q %q", c.g.name, name, id, dosType)
		}
		if g, err := geometryForSize(c.size); err != nil || g.name != c.g.name {
			t.Errorf("%v: geometry for the size: %v %v", c.g.name, g, err)
		}
	}
}

func TestImageFiles(t *testing.T) {
	for _, g := range []*geometry{geometryD64(), geometryD80(), geometryD82()} {
		saves := 0
		d := newBlankImage(g, func([]byte) error { saves++; return nil })
		free := d.blocksFree()

		// A file of 3 blocks and an empty one
		data := bytes.Repeat([]byte("PET"), 200)
		if err := d.writeFile([]byte("BIG"), fileTypePRG, data, false); err != nil {
			t.Fatal(err)
		}
		if err := d.writeFile([]byte("EMPTY"), fileTypeSEQ, nil, false); err != nil {
			t.Fatal(err)
		}
		if d.blocksFree() != free-4 {
			t.Errorf("%v: %v blocks free, expected %v", g.name, d.blocksFree(), free-4)
		}
		got, typ, err := d.readFile([]byte("BIG"))
		if err != nil || !bytes.Equal(got, data) || typ != fileTypePRG {
			t.Errorf("%v: read %v bytes, type %v, %v", g.name, len(got), typ, err)
		}
		if got, _, err := d.readFile([]byte("EMPTY")); err != nil || len(got) != 0 {
			t.Errorf("%v: the empty file has %v bytes, %v", g.name, len(got), err)
		}
		if err := d.writeFile([]byte("BIG"), fileTypePRG, data, false); err != errFileExists {
			t.Errorf("%v: writing over a file: %v", g.name, err)
		}

		entries := d.entries()
		if len(entries) != 2 || entries[0].blocks != 3 || entries[1].typ != fileTypeClosed|fileTypeSEQ {
			t.Errorf("%v: entries %+v", g.name, entries)
		}

		if err := d.rename([]byte("BIG"), []byte("LARGE")); err != nil {
			t.Fatal(err)
		}
		if err := d.scratch([]byte("LARGE")); err != nil {
			t.Fatal(err)
		}
		if _, _, err := d.readFile([]byte("LARGE")); err != errFileNotFound {
			t.Errorf("%v: reading a scratched file: %v", g.name, err)
		}
		if d.blocksFree() != free-1 {
			t.Errorf("%v: %v blocks free after scratch, expected %v", g.name, d.blocksFree(), free-1)
		}
		if saves == 0 {
			t.Errorf("%v: the image was never saved", g.name)
		}
	}
}

func TestDirectoryGrows(t *testing.T) {
	d := newBlankImage(geometryD64(), func([]byte) error { return nil })
	for i := 0; i < 20; i++ {
		name := []byte{'F', byte('A' + i)}
		if err := d.writeFile(name, fileTypePRG, []byte{1, 4, 0}, false); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(d.entries()); n != 20 {
		t.Errorf("%v entries, expected 20", n)
	}
}

func TestDiskFull(t *testing.T) {
	d := newBlankImage(geometryD64(), func([]byte) error { return nil })
	big := make([]byte, 700*blockDataSize)
	if err := d.writeFile([]byte("HUGE"), fileTypePRG, big, false); err != errDiskFull {
		t.Errorf("writing more than fits: %v", err)
	}
	fits := make([]byte, 664*blockDataSize)
	if err := d.writeFile([]byte("ALL"), fileTypePRG, fits, false); err != nil {
		t.Errorf("writing what fits: %v", err)
	}
	if d.blocksFree() != 0 {
		t.Errorf("%v blocks free", d.blocksFree())
	}
}

func TestReadOnlyImage(t *testing.T) {
	d := newBlankImage(geometryD64(), func([]byte) error { return nil })
	d.save = nil
	if err := d.writeFile([]byte("X"), fileTypePRG, nil, false); err != errWriteProtect {
		t.Errorf("writing a read only image: %v", err)
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		pattern, name string
		match         bool
	}{
		{"HELLO", "HELLO", true},
		{"HELLO", "HELLO2", false},
		{"HEL*", "HELLO", true},
		{"*", "ANY", true},
		{"H?LLO", "HALLO", true},
		{"H?LLO", "HALL", false},
	}
	for _, c := range cases {
		if matches([]byte(c.pattern), []byte(c.name)) != c.match {
			t.Errorf("%q against %q should be %v", c.pattern, c.name, c.match)
		}
	}
}

func TestParseName(t *testing.T) {
	p := parseName([]byte("@1:DATA,S,W"))
	if !p.replace || p.drive != 1 || string(p.name) != "DATA" || p.typ != fileTypeSEQ || p.mode != 'W' {
		t.Errorf("parsed %+v", p)
	}
}

func TestDirectoryListing(t *testing.T) {
	d := newBlankImage(geometryD64(), func([]byte) error { return nil })
	d.writeFile([]byte("HELLO"), fileTypePRG, []byte{1, 4, 0, 0}, false)
	listing := directoryListing(d, 0, nil, 0)
	if listing[0] != 0x01 || listing[1] != 0x04 {
		t.Fatalf("load address %02x%02x", listing[1], listing[0])
	}
	if !bytes.Contains(listing, []byte("\"HELLO\"")) || !bytes.Contains(listing, []byte("BLOCKS FREE.")) {
		t.Errorf("listing %q", listing)
	}
	// The links point to the next line
	address := 0x0401
	pos := 2
	for {
		link := int(listing[pos]) | int(listing[pos+1])<<8
		if link == 0 {
			break
		}
		end := bytes.IndexByte(listing[pos+4:], 0)
		address += 4 + end + 1
		if link != address {
			t.Fatalf("link $%04x, expected $%04x", link, address)
		}
		pos += 4 + end + 1
	}
}
