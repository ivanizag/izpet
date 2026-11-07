package activities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// answerYes waits for the question of HEADER and SCRATCH and answers it
func answerYes(t *testing.T, p *izpet.PET) {
	t.Helper()
	waitForText(t, p, "ARE YOU SURE", 120)
	typeText(t, p, "Y\r")
	waitForCursor(t, p, "READY.")
}

/*
diskScreenshots is the disk commands of BASIC 4 on an 8032 with an 8050: a
disk formatted, a program saved and loaded, a data file written and read,
the error status, a file renamed and one scratched, and a folder of the host
in the drive 1
*/
func diskScreenshots(t *testing.T) {
	const page, label = "disk", "CBM 8032"
	album := NewAlbum(filepath.Join(activityImages, page))
	folder := filepath.Join(t.TempDir(), "myfolder")
	must(t, os.Mkdir(folder, 0o755))
	config := testConfig(t, "8032")
	config.Disk = filepath.Join(t.TempDir(), "work.d80")
	config.Disk1 = folder
	p := buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)

	// The disk formatted, and its empty directory
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "HEADER \"WORK\",D0,I01\r")
	answerYes(t, p)
	typeText(t, p, "DIRECTORY\r")
	waitForText(t, p, "BLOCKS FREE", 120)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "header", label))

	// A program saved, and the directory again
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 PRINT \"HELLO FROM THE DISK\"\r")
	typeText(t, p, "DSAVE \"HELLO\"\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	dir := Record(p, label)
	dir.Type("DIRECTORY\r", 2)
	dir.Run(30, 3)
	dir.Type("DLOAD \"HELLO\"\r", 2)
	dir.Run(30, 3)
	dir.Type("RUN\r", 2)
	dir.Run(60, 3)
	must(t, album.SaveRecording(dir, "dload", 300))

	// A data file, written and read
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 DOPEN#1,\"NOTES\",W\r")
	typeText(t, p, "20 PRINT#1,\"BUY MORE DISKS\"\r")
	typeText(t, p, "30 DCLOSE#1\r")
	typeText(t, p, "RUN\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 DOPEN#1,\"NOTES\"\r")
	typeText(t, p, "20 INPUT#1,A$\r")
	typeText(t, p, "30 DCLOSE#1\r")
	typeText(t, p, "40 PRINT A$\r")
	typeText(t, p, "RUN\r")
	waitForText(t, p, "BUY MORE DISKS", 120)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "notes", label))

	// The error status
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "DLOAD \"NOTHING\"\r")
	waitForText(t, p, "FILE NOT FOUND", 120)
	typeText(t, p, "PRINT DS$\r")
	waitForText(t, p, "62,FILE NOT FOUND", 120)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "ds", label))

	// Renamed and scratched
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "RENAME \"HELLO\" TO \"GREETING\"\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "SCRATCH \"NOTES\"\r")
	answerYes(t, p)
	typeText(t, p, "DIRECTORY\r")
	waitForText(t, p, "BLOCKS FREE", 120)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "scratch", label))

	// The folder of the host, in the drive 1
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "DLOAD \"GREETING\"\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "DSAVE \"GREETING\",D1\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "DIRECTORY D1\r")
	waitForText(t, p, "BLOCKS FREE", 120)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "folder", label))
	if _, err := os.Stat(filepath.Join(folder, "greeting.prg")); err != nil {
		t.Errorf("the program is not in the folder: %v", err)
	}
}
