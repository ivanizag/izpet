package activities

import (
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// graphicsKeys are the keys typed with SHIFT for some graphics characters
// of the graphics keyboard
func graphicsKeys() map[rune]izpet.Key {
	return map[rune]izpet.Key{
		'♠': 'A', '♥': 'S', '♦': 'Z', '♣': 'X', '●': 'Q', '○': 'W',
		'╭': 'U', '╮': 'I', '╰': 'J', '╯': 'K', '─': '@', '│': ']',
	}
}

// typeGraphics types a text with graphics characters, each with SHIFT and
// its key, and the RVS key for '{' and SHIFT and RVS, OFF, for '}'
func typeGraphics(t *testing.T, p *izpet.PET, text string) {
	t.Helper()
	keys := graphicsKeys()
	for _, ch := range text {
		switch {
		case ch == '{':
			pressKey(p, izpet.KeyRvs, false)
		case ch == '}':
			pressKey(p, izpet.KeyRvs, true)
		case keys[ch] != izpet.KeyNone:
			pressKey(p, keys[ch], true)
		default:
			typeText(t, p, string(ch))
		}
	}
}

/*
drawingScreenshots is drawing with the graphics characters of the keyboard on
a 3032: the symbols typed with SHIFT, a box, reverse video, the one line
maze, and characters poked into the screen memory
*/
func drawingScreenshots(t *testing.T) {
	const page, label = "drawing", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	p := buildTestPET(t, testConfig(t, "3032"))
	waitForText(t, p, "READY.", 600)

	// The suits, typed with SHIFT, in a PRINT
	pressKey(p, izpet.KeyHome, true)
	typeGraphics(t, p, "PRINT \"♠♥♦♣ ●○\"\r")
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "suits", label))

	// A box, a line at a time
	pressKey(p, izpet.KeyHome, true)
	typeGraphics(t, p, "10 PRINT \"╭────────╮\"\r")
	typeGraphics(t, p, "20 PRINT \"│ MY PET │\"\r")
	typeGraphics(t, p, "30 PRINT \"╰────────╯\"\r")
	typeText(t, p, "RUN\r")
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "box", label))

	// Reverse video
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeGraphics(t, p, "PRINT \"{ MY PET }\"\r")
	waitForCursor(t, p, "READY.")
	must(t, album.ScreenshotOf(p, "reverse", label))

	// The maze, recorded
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 PRINT CHR$(205.5+RND(1));:GOTO 10\r")
	// Forty seconds, shown four times faster: 24 frames of the machine,
	// 40 hundredths, for each picture of 10
	maze := Record(p, label)
	maze.Type("RUN\r", 2)
	for frame := 0; frame < 40*60; frame += 24 {
		p.RunFrames(24)
		maze.Capture(10)
	}
	must(t, album.SaveRecording(maze, "maze", 300))
	pressKey(p, izpet.KeyStop, false)
	waitForCursor(t, p, "READY.")

	// The screen memory
	typeText(t, p, "NEW\r")
	pressKey(p, izpet.KeyHome, true)
	typeText(t, p, "10 PRINT CHR$(147)\r")
	typeText(t, p, "20 FOR I=0 TO 39\r")
	typeText(t, p, "30 POKE 32768+I,81:POKE 33728+I,81\r")
	typeText(t, p, "40 NEXT I\r")
	typeText(t, p, "50 FOR I=0 TO 24\r")
	typeText(t, p, "60 POKE 32768+I*40,81:POKE 32807+I*40,81\r")
	typeText(t, p, "70 NEXT I\r")
	typeText(t, p, "80 GOTO 80\r")
	poking := Record(p, label)
	poking.Type("RUN\r", 2)
	poking.Run(180, 3)
	must(t, album.SaveRecording(poking, "poke", 200))
}
