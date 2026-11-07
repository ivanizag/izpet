package activities

import (
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

/*
firstProgramScreenshots is BASIC on a 3032: commands typed and run at once,
a program typed, run and listed, a line of it changed with the screen
editor, and a loop stopped with RUN/STOP
*/
func firstProgramScreenshots(t *testing.T) {
	const page, label = "first-program", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	p := buildTestPET(t, testConfig(t, "3032"))
	waitForText(t, p, "READY.", 600)

	// A command, run at once
	typeText(t, p, "PRINT \"HELLO\"\r")
	typeText(t, p, "PRINT 2+3*4\r")
	waitForText(t, p, " 14", 60)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "commands", label))

	// The program, typed, recorded
	pressKey(p, izpet.KeyHome, true)
	typing := Record(p, label)
	typing.Run(10, 2)
	typing.Type("10 PRINT \"WHAT IS YOUR NAME\";\r", 2)
	typing.Type("20 INPUT N$\r", 2)
	typing.Type("30 FOR I=1 TO 5\r", 2)
	typing.Type("40 PRINT \"HELLO, \";N$\r", 2)
	typing.Type("50 NEXT I\r", 2)
	typing.Run(30, 3)
	must(t, album.SaveRecording(typing, "typing", 200))

	// Run, with a name typed
	running := Record(p, label)
	running.Type("RUN\r", 2)
	running.Run(20, 2)
	running.Type("ADA\r", 2)
	running.Run(60, 3)
	must(t, album.SaveRecording(running, "running", 300))

	// The listing
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "LIST\r")
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "list", label))

	// The line 40 changed on the screen: the cursor taken to it, HELLO
	// typed over, and RETURN
	editing := Record(p, label)
	editing.Run(10, 2)
	up := lineOf(t, p, "READY.") + 1 - lineOf(t, p, "40 PRINT")
	for i := 0; i < up; i++ {
		pressKey(p, izpet.KeyCrsrDown, true)
		editing.Capture(10)
	}
	// LIST puts a space before the line number
	for i := 0; i < len(" 40 PRINT \""); i++ {
		pressKey(p, izpet.KeyCrsrRight, false)
		editing.Capture(5)
	}
	editing.Type("HOWDY", 2)
	editing.Run(20, 2)
	editing.Type("\r", 2)
	editing.Run(20, 2)
	must(t, album.SaveRecording(editing, "editing", 200))

	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "RUN\r")
	p.RunFrames(20)
	typeText(t, p, "GRACE\r")
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "howdy", label))

	// A loop for ever, stopped with RUN/STOP
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 PRINT \"PET \";\r")
	typeText(t, p, "20 GOTO 10\r")
	typeText(t, p, "RUN\r")
	p.RunFrames(90)
	pressKey(p, izpet.KeyStop, false)
	waitForText(t, p, "BREAK IN", 60)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "break", label))
}
