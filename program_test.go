package izpet

import (
	"bytes"
	"strings"
	"testing"
)

// testBasicLines makes a BASIC program saved at an address, a line per text
// already tokenized, with its links and the end of program
func testBasicLines(address uint16, lines ...[]uint8) []uint8 {
	var data []uint8
	for i, line := range lines {
		next := int(address) + len(data) + 4 + len(line) + 1
		data = append(data, uint8(next), uint8(next>>8), uint8(10*(i+1)), 0)
		data = append(data, line...)
		data = append(data, 0)
	}
	return append(data, 0, 0)
}

func TestIsBasicProgram(t *testing.T) {
	program := testBasicLines(0x0801, []uint8{0x99, ' ', '1'}, []uint8{0x80})
	if !isBasicProgram(0x0801, program) {
		t.Error("a program at $0801 should be BASIC")
	}
	if isBasicProgram(0x0401, program) {
		t.Error("the links of a program at $0801 do not chain from $0401")
	}
	if isBasicProgram(0x033a, []uint8{0xa9, 0x51, 0x8d, 0x00, 0x80, 0x60}) {
		t.Error("machine code is not BASIC")
	}
}

func TestRelink(t *testing.T) {
	saved := testBasicLines(0x0801, []uint8{0x99, ' ', '1'}, []uint8{0x80})
	relink(0x0401, saved)
	if expected := testBasicLines(0x0401, []uint8{0x99, ' ', '1'}, []uint8{0x80}); !bytes.Equal(saved, expected) {
		t.Errorf("relinked % x, expected % x", saved, expected)
	}
}

// 10 PRINT "MOVED";6*7
func testMovedLine() []uint8 {
	return []uint8{0x99, ' ', '"', 'M', 'O', 'V', 'E', 'D', '"', ';', '6', 0xac, '7'}
}

func TestLoadBasicProgramSavedElsewhere(t *testing.T) {
	data := testBasicLines(0x0801, testMovedLine())
	p := newTestPET(t, "3032")
	err := p.LoadProgram(testWritePrg(t, 0x0801, data))
	if err != nil {
		t.Fatal(err)
	}
	testWaitText(t, p, "MOVED 42")
	testType(t, p, "LIST\r")
	testWaitText(t, p, "10 PRINT \"MOVED\";6*7")
}

func TestLoadP00(t *testing.T) {
	data := testBasicLines(0x0401, testMovedLine())
	header := make([]uint8, p00HeaderSize)
	copy(header, p00Magic)
	copy(header[8:], "MOVED")
	prg := append([]uint8{0x01, 0x04}, data...)
	p := newTestPET(t, "3032")
	err := p.loadProgramData("test.p00", append(header, prg...))
	if err != nil {
		t.Fatal(err)
	}
	testWaitText(t, p, "MOVED 42")
}

func TestLoadMachineCode(t *testing.T) {
	code := []uint8{0xa9, 0x51, 0x8d, 0x00, 0x80, 0x60}
	p := newTestPET(t, "3032")
	err := p.LoadProgram(testWritePrg(t, 0x033a, code))
	if err != nil {
		t.Fatal(err)
	}
	testWaitText(t, p, "READY.")
	p.RunFrames(10)
	for i, b := range code {
		if got := p.Peek(0x033a + uint16(i)); got != b {
			t.Errorf("$%04x is $%02x, expected $%02x", 0x033a+i, got, b)
		}
	}
	if p.IsTyping() || strings.Contains(p.GetScreenText(), "RUN") {
		t.Error("machine code should not be run")
	}
}

func TestLoadProgramTooBig(t *testing.T) {
	p := newTestPET(t, "2001") // 8 KB
	err := p.LoadProgram(testWritePrg(t, 0x0401, make([]uint8, 9000)))
	if err == nil || !strings.Contains(err.Error(), "8 KB") {
		t.Errorf("a program of 9000 bytes should not fit in 8 KB: %v", err)
	}
	err = p.LoadProgram(testWritePrg(t, 0x8000, make([]uint8, 1000)))
	if err != nil {
		t.Errorf("a program on the screen memory should load: %v", err)
	}
}
