package activities

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ivanizag/izpet"
)

// microchessReply is the last move written in the right column, where
// Microchess writes its answers
func microchessReply(p *izpet.PET) string {
	reply := ""
	for _, line := range strings.Split(p.GetScreenText(), "\n") {
		runes := []rune(line)
		if len(runes) > 33 {
			text := strings.TrimSpace(string(runes[33:]))
			if strings.HasPrefix(text, ":") && len(text) > 1 {
				reply = text[1:]
			}
		}
	}
	return reply
}

// microchessMove types a move and waits for the answer of the computer,
// returning it and the seconds it took
func microchessMove(t *testing.T, p *izpet.PET, move string) (string, float64) {
	t.Helper()
	before := microchessReply(p)
	typeText(t, p, move+"\r")
	start := p.GetFrames()
	if !p.RunUntil(60*60*30, func() bool {
		reply := microchessReply(p)
		return reply != before && reply != move && strings.Contains(reply, "-")
	}) {
		t.Fatalf("no answer to %v:\n%v", move, p.GetScreenText())
	}
	p.RunFrames(30)
	return microchessReply(p), float64(p.GetFrames()-start) / 60
}

/*
microchessScreenshots is a game of Microchess on a 3032: the title, the
board, a move and the answer of the computer, castling, and the level of
play raised
*/
func microchessScreenshots(t *testing.T) {
	const page, label = "microchess", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	p := buildTestPET(t, testConfig(t, "3032"))
	waitForText(t, p, "READY.", 600)
	must(t, p.LoadFile("../software/microchess.prg"))
	p.RunFrames(30)
	typeText(t, p, "SYS 1039\r")
	waitForText(t, p, "PETER JENNINGS", 120)
	p.RunFrames(30)
	must(t, album.Machine(p, "title", label))

	typeText(t, p, "C\r")
	waitForText(t, p, "CLOCK", 120)
	p.RunFrames(60)
	must(t, album.ScreenshotOf(p, "board", label))

	// The first move, and the answer
	first := Record(p, label)
	first.Run(30, 3)
	first.Type("E2E4\r", 3)
	if !p.RunUntil(60*30, func() bool {
		first.Run(6, 6)
		return microchessReply(p) == "E7-E5"
	}) {
		t.Fatalf("no answer to E2E4:\n%v", p.GetScreenText())
	}
	first.Run(60, 6)
	must(t, album.SaveRecording(first, "first", 300))

	// Up to castling, the king two squares to the side
	for _, move := range []string{"G1F3", "F1C4", "E1G1"} {
		reply, seconds := microchessMove(t, p, move)
		t.Logf("%v %v in %.1f s", move, reply, seconds)
	}
	waitForText(t, p, ":D7-D5", 1)
	must(t, album.ScreenshotOf(p, "castled", label))

	// A capture, the level raised and a capture again, timed
	reply, seconds := microchessMove(t, p, "E4D5")
	t.Logf("E4D5 %v in %.1f s", reply, seconds)
	typeText(t, p, "IQ=6\r")
	waitForText(t, p, "IQ= 6", 60)
	reply, seconds = microchessMove(t, p, "D5C6")
	t.Logf("D5C6 %v in %.1f s", reply, seconds)
	if reply != "E4-F3" {
		t.Fatalf("the answer to D5C6 is %v", reply)
	}
	must(t, album.ScreenshotOf(p, "iq", label))
}
