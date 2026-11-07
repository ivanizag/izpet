package izpet

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

const testTapeFrames = 6000 // 100 seconds

func TestTapeSaveLoad(t *testing.T) {
	tape := filepath.Join(t.TempDir(), "test.tap")

	p := newTestPETWith(t, "3032", func(c *Configuration) { c.Tape = tape })
	testWaitText(t, p, "READY.")
	testType(t, p, "10 PRINT \"FROM TAPE\"\r")
	testRun(t, p, "SAVE \"TP\"", "WRITING TP")
	testWaitTextFrames(t, p, "READY.", testTapeFrames)
	p.RunFrames(60) // The motor stops and the tape is saved
	if info, err := os.Stat(tape); err != nil || info.Size() < 10_000 {
		t.Fatalf("the tape was not saved: %v", err)
	}

	p = newTestPETWith(t, "3032", func(c *Configuration) { c.Tape = tape })
	testWaitText(t, p, "READY.")
	testRun(t, p, "LOAD", "SEARCHING")
	testWaitTextFrames(t, p, "FOUND TP", testTapeFrames)
	testWaitTextFrames(t, p, "READY.", testTapeFrames)
	testRun(t, p, "RUN", "FROM TAPE")
}

func TestT64(t *testing.T) {
	// 10 PRINT "FROM T64"
	program := []uint8{0x11, 0x04, 10, 0, 0x99, ' ', '"', 'F', 'R', 'O', 'M', ' ', 'T', '6', '4', '"', 0, 0, 0}
	data := make([]uint8, 0x60)
	copy(data, "C64S tape image file")
	binary.LittleEndian.PutUint16(data[0x20:], 0x100)
	binary.LittleEndian.PutUint16(data[0x22:], 1)
	binary.LittleEndian.PutUint16(data[0x24:], 1)
	entry := data[0x40:]
	entry[0] = 1
	entry[1] = 0x82
	binary.LittleEndian.PutUint16(entry[2:], 0x0401)
	binary.LittleEndian.PutUint16(entry[4:], uint16(0x0401+len(program)))
	binary.LittleEndian.PutUint32(entry[8:], 0x60)
	copy(entry[0x10:], "PROGRAM")
	data = append(data, program...)

	name := filepath.Join(t.TempDir(), "test.t64")
	if err := os.WriteFile(name, data, 0644); err != nil {
		t.Fatal(err)
	}
	p := newTestPETWith(t, "3032", func(c *Configuration) { c.AddFiles([]string{name}) })
	testWaitText(t, p, "FROM T64")
}
