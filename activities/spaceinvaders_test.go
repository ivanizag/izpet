package activities

import (
	"path/filepath"
	"testing"

	"github.com/ivanizag/izpet"
)

// invadersShip is the column of the left of the ship, on the row before the
// last, or -1 when it is not there
func invadersShip(p *izpet.PET) int {
	for c := 0; c < 40; c++ {
		if p.Peek(uint16(0x8000+23*40+c)) != 0x20 {
			return c
		}
	}
	return -1
}

// invadersPlay plays for some frames, recorded, a step every few of them:
// the ship goes to the column of the step and fires. The keys 4 and 6 move
// it and A fires.
func invadersPlay(p *izpet.PET, r *Recording, frames int, target func(step int) int) {
	const every = 3
	for step := 0; step*every < frames; step++ {
		ship, to := invadersShip(p), target(step)
		p.PutKey('4', ship >= 0 && ship+2 > to)
		p.PutKey('6', ship >= 0 && ship+2 < to)
		p.PutKey('A', step%4 < 2)
		r.Run(every, every)
	}
	p.ReleaseAllKeys()
}

/*
spaceInvadersScreenshots is a game of Space Invaders on a 3032: how to hear
it, the title and the demo, the game started and a recording of the ship
moving and firing, its end the picture of the page
*/
func spaceInvadersScreenshots(t *testing.T) {
	const page, label = "space-invaders", "CBM 3032"
	album := NewAlbum(filepath.Join(activityImages, page))
	p := buildTestPET(t, testConfig(t, "3032"))
	waitForText(t, p, "READY.", 600)
	must(t, p.LoadFile("../software/space-invaders.prg"))
	waitForText(t, p, "SOUND EFFECTS", 600)
	p.RunFrames(30)
	must(t, album.Machine(p, "sound", label))

	// The title and the demo, until a key is pressed
	p.PutKey(izpet.KeySpace, true)
	title := Record(p, label)
	title.Run(6, 6)
	p.PutKey(izpet.KeySpace, false)
	title.Run(60*24, 6)
	must(t, album.SaveRecording(title, "title", 100))

	pressKey(p, izpet.KeySpace, false)
	waitForText(t, p, "ANY KEY TO START", 120)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "push", label))

	// The game, the ship going from side to side and firing
	pressKey(p, izpet.KeySpace, false)
	p.RunFrames(120)
	sweep := func(step int) int {
		x := (step / 2) % 60
		if x >= 30 {
			x = 60 - x
		}
		return 5 + x
	}
	playing := Record(p, label)
	invadersPlay(p, playing, 60*20, sweep)
	must(t, album.SaveRecording(playing, "playing", 200))
	must(t, album.ScreenshotOf(p, "game", label))
}
