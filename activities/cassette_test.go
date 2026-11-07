package activities

import (
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// tapeTimelapse runs the machine until a text is on the screen, taking a
// picture every 30 frames shown for a tenth of a second, five times faster
// than the PET. It returns the seconds it took.
func tapeTimelapse(t *testing.T, p *izpet.PET, r *Recording, text string) float64 {
	t.Helper()
	start := p.GetFrames()
	for !hasText(p, text) {
		if p.GetFrames()-start > 60*300 {
			t.Fatalf("'%v' did not come", text)
		}
		p.RunFrames(30)
		r.Capture(10)
	}
	return float64(p.GetFrames()-start) / 60
}

/*
cassetteScreenshots is two programs saved on a blank tape of a 3032, the tape
rewound, and the second one loaded, the first found and skipped on the way
*/
func cassetteScreenshots(t *testing.T) {
	const page, label = "cassette", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	tape := filepath.Join(t.TempDir(), "mytape.tap")
	config := testConfig(t, "3032")
	config.Tape = tape
	p := buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)

	// The first program, saved, recorded
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 PRINT \"HELLO FROM THE TAPE\"\r")
	saving := Record(p, label)
	saving.Type("SAVE \"HELLO\"\r", 2)
	seconds := tapeTimelapse(t, p, saving, "READY.")
	saving.Run(30, 3)
	must(t, album.SaveRecording(saving, "saving", 300))
	t.Logf("saving HELLO took %.1f seconds", seconds)
	waitForCursor(t, p, "READY.")

	// The second, after it on the tape
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 PRINT CHR$(205.5+RND(1));:GOTO 10\r")
	typeText(t, p, "SAVE \"MAZE\"\r")
	start := p.GetFrames()
	waitForText(t, p, "READY.", 60*120)
	t.Logf("saving MAZE took %.1f seconds", float64(p.GetFrames()-start)/60)
	p.RunFrames(60) // The motor stops, the tape is written to its file

	// A new PET, as the next day, with the tape rewound
	p = buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)
	pressKey(p, izpet.KeyHome, true)
	loading := Record(p, label)
	loading.Type("LOAD \"MAZE\"\r", 2)
	seconds = tapeTimelapse(t, p, loading, "READY.")
	loading.Run(30, 3)
	must(t, album.SaveRecording(loading, "loading", 300))
	t.Logf("loading MAZE took %.1f seconds", seconds)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "loaded", label))
	typeText(t, p, "RUN\r")
	p.RunFrames(600)
	must(t, album.ScreenshotOf(p, "running", label))
}
