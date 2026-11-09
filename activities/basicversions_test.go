package activities

import (
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// basicSpeedProgram fills memory with strings, then times a loop and a
// garbage collection, the one FRE(0) forces
const basicSpeedProgram = "10 DIM A$(600):FOR I=0 TO 600:A$(I)=STR$(I):NEXT\r" +
	"20 T=TI:FOR I=1 TO 1000:NEXT:PRINT \"LOOP\";INT((TI-T)/6)/10\r" +
	"30 T=TI:X=FRE(0):PRINT \"FRE(0)\";INT((TI-T)/6)/10\r" +
	"RUN\r"

// basicStart builds a model with a disk and waits for BASIC
func basicStart(t *testing.T, model string, disk string) *izpet.PET {
	t.Helper()
	config := testConfig(t, model)
	config.Disk = disk
	p := buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)
	waitForCursor(t, p, "READY.")
	return p
}

// basicClear clears the screen, with SHIFT and HOME
func basicClear(p *izpet.PET) {
	pressKey(p, izpet.KeyHome, true)
}

// basicMixedCase prints a text with lowercase and uppercase letters, in the
// character set with the lowercase letters
func basicMixedCase(t *testing.T, p *izpet.PET) {
	t.Helper()
	basicClear(p)
	typeText(t, p, "POKE 59468,14\r")
	typeExact(t, p, "print \"Hello World\"\r")
	waitForCursor(t, p, "READY.")
}

// basicSpeed runs the speed program and waits for its results
func basicSpeed(t *testing.T, p *izpet.PET) {
	t.Helper()
	basicClear(p)
	typeText(t, p, "NEW\r")
	basicClear(p)
	typeText(t, p, basicSpeedProgram)
	waitForText(t, p, "FRE(0) ", 60*60)
	waitForCursor(t, p, "READY.")
}

/*
basicVersionsScreenshots compares the three BASICs: BASIC 1 on a 2001 with
its lowercase the other way round and no disks, BASIC 2 on a 3032 with the
disks, the monitor and the message of WAIT 6502, BASIC 4 on a 4032 with its
disk commands, and the speed of the three
*/
func basicVersionsScreenshots(t *testing.T) {
	const page = "basic-versions"
	album := NewAlbum(filepath.Join(activityImages, page))
	disk := filepath.Join(t.TempDir(), "basics.d64")

	// BASIC 1
	const label1 = "PET 2001, BASIC 1"
	p := basicStart(t, "2001", disk)
	must(t, album.Machine(p, "2001", label1))
	basicMixedCase(t, p)
	must(t, album.ScreenshotOf(p, "case1", label1))

	basicClear(p)
	typeText(t, p, "POKE 59468,12\r")
	basicClear(p)
	typeText(t, p, "SYS 1024\r")
	waitForText(t, p, "?ILLEGAL QUANTITY", 60)
	waitForCursor(t, p, "READY.")
	typeText(t, p, "LOAD \"$\",8\r")
	waitForText(t, p, "LOADING", 120)
	p.RunFrames(600)
	p.PutKey(izpet.KeyStop, true)
	waitForText(t, p, "BREAK", 120)
	p.PutKey(izpet.KeyStop, false)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "nodisk1", label1))

	// BASIC 2
	const label2 = "CBM 3032, BASIC 2"
	p = basicStart(t, "3032", disk)
	must(t, album.Machine(p, "3032", label2))
	basicMixedCase(t, p)
	must(t, album.ScreenshotOf(p, "case2", label2))

	basicClear(p)
	typeText(t, p, "POKE 59468,12\r")
	basicClear(p)
	typeText(t, p, "OPEN 15,8,15,\"N0:BASICS,01\":CLOSE 15\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "10 PRINT \"HELLO\"\rSAVE \"HELLO\",8\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "LOAD \"$\",8\r")
	waitForText(t, p, "LOADING", 120)
	waitForCursor(t, p, "READY.")
	typeText(t, p, "LIST\r")
	waitForText(t, p, "BLOCKS FREE", 120)
	waitForCursor(t, p, "READY.")
	typeText(t, p, "DIRECTORY\r")
	waitForText(t, p, "?SYNTAX", 60)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "disk2", label2))

	basicClear(p)
	typeText(t, p, "SYS 1024\r")
	waitForCursorAt(t, p, ".;", 1)
	must(t, album.ScreenshotOf(p, "monitor2", label2))
	typeText(t, p, "X\r")
	waitForCursor(t, p, "READY.")

	// The message of WAIT 6502, from a fresh start for the banner
	p = basicStart(t, "3032", disk)
	typeText(t, p, "WAIT 6502,1\r")
	waitForText(t, p, "MICROSOFT!", 60)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "microsoft", label2))
	basicClear(p)
	p.RunFrames(10)
	pressKeys(p, izpet.KeyCrsrDown, false, 6)
	typeText(t, p, "WAIT 6502,10\r")
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "microsoft10", label2))

	// BASIC 4
	const label4 = "CBM 4032, BASIC 4"
	p = basicStart(t, "4032", disk)
	must(t, album.Machine(p, "4032", label4))
	basicClear(p)
	typeText(t, p, "DIRECTORY\r")
	waitForText(t, p, "BLOCKS FREE", 120)
	waitForCursor(t, p, "READY.")
	typeText(t, p, "PRINT DS$\r")
	waitForText(t, p, "00, OK", 60)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "disk4", label4))

	// The speed of the three
	for _, m := range []struct{ model, label, name string }{
		{"2001", label1, "speed1"},
		{"3032", label2, "speed2"},
		{"4032", label4, "speed4"},
	} {
		p = basicStart(t, m.model, disk)
		basicSpeed(t, p)
		must(t, album.ScreenshotOf(p, m.name, m.label))
	}
}
