package activities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// visiCalcEntry types an entry at a coordinate, going there with >
func visiCalcEntry(t *testing.T, p *izpet.PET, coordinate string, entry string) {
	t.Helper()
	typeText(t, p, ">"+coordinate+"\r")
	p.RunFrames(10)
	typeExact(t, p, entry+"\r")
	p.RunFrames(20)
}

/*
visiCalcScreenshots is a budget in VisiCalc on an 8032: the program loaded,
the labels, the values and a sum typed, a value changed and the sum
recalculated, the sheet saved on a disk and printed
*/
func visiCalcScreenshots(t *testing.T) {
	const page, label = "visicalc", "CBM 8032"
	album := NewAlbum(filepath.Join(activityImages, page))
	dir := t.TempDir()
	disk := filepath.Join(dir, "visicalc.d64")
	copyFile(t, "../software/visicalc.d64", disk)
	printed := filepath.Join(dir, "printed.txt")
	config := testConfig(t, "8032")
	config.Disk = disk
	config.Disk1 = filepath.Join(dir, "sheets.d64")
	config.Printer = printed
	p := buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)

	// Loaded, timed
	typeText(t, p, "DLOAD \"VC*\"\r")
	waitForCursor(t, p, "READY.")
	typeText(t, p, "RUN\r")
	start := p.GetFrames()
	waitForText(t, p, "1.70AM", 60*120)
	t.Logf("VisiCalc loaded in %.1f seconds", float64(p.GetFrames()-start)/60)
	p.RunFrames(60)
	must(t, album.ScreenshotOf(p, "empty", label))

	// The budget
	for _, e := range [][2]string{
		{"A1", "budget"},
		{"A3", "rent"}, {"B3", "400"},
		{"A4", "food"}, {"B4", "250"},
		{"A5", "car"}, {"B5", "120"},
		{"A6", "phone"}, {"B6", "30"},
		{"A8", "total"}, {"B8", "@sum(b3.b6)"},
	} {
		visiCalcEntry(t, p, e[0], e[1])
	}
	for _, coordinate := range []string{"A7", "B7"} {
		typeText(t, p, ">"+coordinate+"\r")
		typeText(t, p, "/--\r")
		p.RunFrames(20)
	}
	typeText(t, p, ">B8\r")
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "budget", label))

	// What if the food costs 300
	typeText(t, p, ">B4\r")
	p.RunFrames(30)
	whatIf := Record(p, label)
	whatIf.Run(30, 3)
	whatIf.Type("300", 2)
	whatIf.Run(30, 3)
	whatIf.Type("\r", 2)
	whatIf.Run(90, 3)
	must(t, album.SaveRecording(whatIf, "what-if", 300))

	// Saved on the disk of the drive 1
	typeText(t, p, "/S")
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "storage", label))
	typeText(t, p, "S")
	p.RunFrames(30)
	typeText(t, p, "1:budget.vc\r")
	p.RunFrames(600)
	if p.DiskName(1) == "" {
		t.Fatal("no disk in the drive 1")
	}

	// Printed, from A1 to B8, on a PET printer
	typeText(t, p, ">A1\r")
	typeText(t, p, "/PP")
	p.RunFrames(30)
	typeText(t, p, "B8\r")
	p.RunFrames(600)
	must(t, p.Close())
	data, err := os.ReadFile(printed)
	must(t, err)
	t.Logf("printed:\n%s", data)
	must(t, os.WriteFile(album.Path("printed.txt"), data, 0o644))
}
