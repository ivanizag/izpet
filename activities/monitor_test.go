package activities

import (
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

/*
monitorScreenshots is the machine language monitor of a 3032: entered with
SYS 1024, the registers, the memory of the screen, a routine typed in hex
that fills the screen, and the same done in BASIC, timed
*/
func monitorScreenshots(t *testing.T) {
	const page, label = "monitor", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	p := buildTestPET(t, testConfig(t, "3032"))
	waitForText(t, p, "READY.", 600)

	// The monitor, and the memory of the screen
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "SYS 1024\r")
	waitForText(t, p, "PC  IRQ", 60)
	typeText(t, p, "M 8000 8007\r")
	waitForText(t, p, ".:  8000", 60)
	waitForCursorAt(t, p, ".:  8000", 1)
	must(t, album.ScreenshotOf(p, "memory", label))

	// The routine, a line of 8 bytes at a time, and looked at again
	pressKey(p, izpet.KeyHome, true)
	typing := Record(p, label)
	typing.Run(10, 2)
	typing.Type(":  033A A2 00 A9 51 9D 00 80 9D\r", 2)
	typing.Type(":  0342 00 81 9D 00 82 9D 00 83\r", 2)
	typing.Type(":  034A E8 D0 F1 60 00 00 00 00\r", 2)
	typing.Type("M 033A 0351\r", 2)
	typing.Run(60, 3)
	must(t, album.SaveRecording(typing, "routine", 300))

	// Back to BASIC, and the screen filled by BASIC, timed
	typeText(t, p, "X\r")
	waitForText(t, p, "READY.", 60)
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 FOR I=32768 TO 33767:POKE I,81:NEXT\r")
	basic := Record(p, label)
	basic.Type("RUN\r", 2)
	start := p.GetFrames()
	for !hasText(p, "READY.") {
		if p.GetFrames()-start > 60*60 {
			t.Fatal("BASIC did not finish")
		}
		basic.Run(6, 6)
	}
	t.Logf("BASIC took %.1f seconds", float64(p.GetFrames()-start)/60)
	basic.Run(60, 6)
	must(t, album.SaveRecording(basic, "basic", 200))

	// And by the routine
	pressKey(p, izpet.KeyHome, true)
	routine := Record(p, label)
	routine.Run(20, 2)
	routine.Type("SYS 826\r", 2)
	routine.Run(60, 3)
	must(t, album.SaveRecording(routine, "sys", 200))
}
