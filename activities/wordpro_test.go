package activities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// copyFile copies a file, for the activities to work on a copy of the disks
// of the repository
func copyFile(t *testing.T, from string, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	must(t, err)
	must(t, os.WriteFile(to, data, 0o644))
}

// controlKey presses OFF/RVS, the control key of WordPro, and then a key
func controlKey(t *testing.T, p *izpet.PET, keys string) {
	t.Helper()
	pressKey(p, izpet.KeyRvs, false)
	typeExact(t, p, keys)
}

/*
wordProScreenshots is a letter written with WordPro 4 on an 8032: the
program loaded and started, the letter typed with a format line, kept on a
disk, the directory, the letter recalled and printed
*/
func wordProScreenshots(t *testing.T) {
	const page, label = "wordpro", "CBM 8032"
	album := NewAlbum(filepath.Join(activityImages, page))
	dir := t.TempDir()
	disk := filepath.Join(dir, "wordpro4.d64")
	copyFile(t, "../software/wordpro4.d64", disk)
	printed := filepath.Join(dir, "printed.txt")
	config := testConfig(t, "8032")
	config.Disk = disk
	config.Disk1 = filepath.Join(dir, "letters.d64")
	config.Printer = printed
	p := buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)

	// Loaded and started, and its questions answered
	typeText(t, p, "DLOAD \"WORDPRO 4\"\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "RUN\r")
	waitForText(t, p, "MAIN TEXT", 600)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "start", label))
	typeText(t, p, "150\r")
	waitForText(t, p, "PRINTER DEVICE", 120)
	typeText(t, p, "\r")
	waitForText(t, p, "SPINWRITER", 120)
	typeText(t, p, "P")
	waitForText(t, p, "DISK DRIVE DEVICE", 120)
	typeText(t, p, "\r")
	waitForText(t, p, "C=  1", 120)
	p.RunFrames(30)

	// The letter, with a format line, typed and recorded
	typing := Record(p, label)
	typing.Run(20, 2)
	pressKey(p, izpet.KeyRvs, false)
	typing.TypeExact("/lm10:rm60:ju1\r", 2)
	typing.TypeExact("Dear Grace,\r\r"+
		"The new PET arrived today. It came with a word processor, so this is "+
		"the first letter I write without a typewriter. I can change any word "+
		"before it goes to paper, and keep the letter on a disk to print it "+
		"again whenever I want.\r\rBest regards,\rAda\r", 2)
	typing.Run(60, 3)
	must(t, album.SaveRecording(typing, "typing", 300))

	// Kept on the disk of the drive 1
	pressKey(p, izpet.KeyHome, true)
	typeExact(t, p, "m1")
	typeExact(t, p, "letter\r")
	waitForText(t, p, "STORED", 600)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "stored", label))

	// The directory, that takes the place of the text
	controlKey(t, p, "1\r")
	waitForText(t, p, "BLOCKS FREE", 600)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "directory", label))

	// Recalled, and printed
	pressKey(p, izpet.KeyHome, true)
	typeExact(t, p, "r")
	typeExact(t, p, "1:letter\r")
	waitForText(t, p, "BEST REGARDS", 600)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "recalled", label))
	controlKey(t, p, "o\r")
	p.RunFrames(900)
	must(t, p.Close())
	data, err := os.ReadFile(printed)
	must(t, err)
	t.Logf("printed:\n%s", data)
	must(t, os.WriteFile(album.Path("printed.txt"), data, 0o644))
}
