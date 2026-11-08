package activities

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ivanizag/izpet"
)

// dogStarPrompt is the question of Dog Star Adventure, on the last line
// with nothing typed after it
const dogStarPrompt = "WHAT SHOULD I DO?"

// dogStarWaiting returns true when the last line of the screen is a
// question with nothing typed after it, the cursor of the game blinking
func dogStarWaiting(p *izpet.PET, question string) bool {
	lines := strings.Split(strings.TrimRight(p.GetScreenText(), "\n "), "\n")
	return strings.TrimRight(lines[len(lines)-1], " ▒") == question
}

/*
dogStarType types a line a key at a time, as the game reads it with GET
in BASIC and loses keys typed faster, and waits for the next question. The
recording, if any, takes pictures on the way.
*/
func dogStarType(t *testing.T, p *izpet.PET, r *Recording, text string, question string) {
	t.Helper()
	run := func(frames int) {
		if r != nil {
			r.Run(frames, 3)
		} else {
			p.RunFrames(uint64(frames))
		}
	}
	for _, ch := range text {
		p.TypeText(string(ch))
		for p.IsTyping() {
			run(3)
		}
		run(6)
	}
	start := p.GetFrames()
	for !dogStarWaiting(p, question) || p.GetFrames()-start < 60 {
		if p.GetFrames()-start > 60*60 {
			t.Fatalf("no %v after %q:\n%v", question, text, p.GetScreenText())
		}
		run(6)
	}
	run(30)
}

/*
dogStarScreenshots is the start of Dog Star Adventure on a 3032 with a
disk drive: the program loaded from the disk, the story, a walk off the
ship with the commands of the game, the things taken, and the game saved
on the disk and loaded back
*/
func dogStarScreenshots(t *testing.T) {
	const page, label = "dog-star", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	disk := filepath.Join(t.TempDir(), "dogstar.d64")
	copyFile(t, "../software/dogstar.d64", disk)
	config := testConfig(t, "3032")
	config.Disk = disk
	p := buildTestPET(t, config)
	waitForText(t, p, "READY.", 600)

	// Loaded from the disk, timed
	typeText(t, p, "LOAD \"DOGSTAR\",8\r")
	start := p.GetFrames()
	waitForCursor(t, p, "READY.")
	t.Logf("loaded in %.1f seconds", float64(p.GetFrames()-start)/60)
	typeText(t, p, "RUN\r")
	start = p.GetFrames()
	waitForText(t, p, "INSTRUCTIONS?", 60*60)
	t.Logf("the question in %.1f seconds", float64(p.GetFrames()-start)/60)
	p.RunFrames(30)
	must(t, album.Machine(p, "start", label))

	// The story, the first page of the instructions
	typeText(t, p, "Y")
	waitForText(t, p, "PRESS RETURN", 60*10)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "story", label))
	for _, text := range []string{"DISGUISED AS", "ALTER EGO", "SAVE WILL CAUSE"} {
		typeText(t, p, "\r")
		waitForText(t, p, text, 60*10)
		waitForText(t, p, "PRESS RETURN TO", 60*10)
		p.RunFrames(30)
		if text == "ALTER EGO" {
			must(t, album.ScreenshotOf(p, "commands", label))
		}
	}
	for !hasText(p, dogStarPrompt) {
		typeText(t, p, "\r")
		p.RunFrames(120)
	}
	dogStarType(t, p, nil, "", dogStarPrompt)
	must(t, album.ScreenshotOf(p, "cockpit", label))

	// Off the ship, to the flight deck
	walk := Record(p, label)
	walk.Run(30, 3)
	for _, command := range []string{"S", "D", "S", "E"} {
		dogStarType(t, p, walk, command+"\r", dogStarPrompt)
	}
	must(t, album.SaveRecording(walk, "walk", 300))

	// Things taken
	for _, command := range []string{"TAKE FUEL", "E", "N", "TAKE BLASTER", "INVENTORY"} {
		dogStarType(t, p, nil, command+"\r", dogStarPrompt)
	}
	must(t, album.ScreenshotOf(p, "inventory", label))

	// Saved on the disk, the game ends
	dogStarType(t, p, nil, "SAVE\r", "TAPE OR DISK? D")
	dogStarType(t, p, nil, "\r", "DRIVE 1 OR? 0")
	dogStarType(t, p, nil, "\r", "TITLE? *")
	typeText(t, p, "FALCON\r")
	waitForText(t, p, "THANKS FOR PLAYING", 60*30)
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "saved", label))

	// And loaded back
	typeText(t, p, "RUN\r")
	waitForText(t, p, "INSTRUCTIONS?", 60*60)
	typeText(t, p, "N")
	waitForText(t, p, dogStarPrompt, 60*10)
	dogStarType(t, p, nil, "LOAD\r", "TAPE OR DISK? D")
	dogStarType(t, p, nil, "\r", "DRIVE 1 OR? 0")
	dogStarType(t, p, nil, "\r", "TITLE? *")
	dogStarType(t, p, nil, "FALCON\r", dogStarPrompt)
	dogStarType(t, p, nil, "INVENTORY\r", dogStarPrompt)
	must(t, album.ScreenshotOf(p, "loaded", label))
	t.Log(p.GetScreenText())
}
