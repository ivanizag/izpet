package activities

import (
	"path/filepath"
	"strings"
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

// invadersBomb returns true for the screen codes of the bombs
func invadersBomb(ch uint8) bool {
	return ch == '$' || ch == '*' || ch == 0x5d || ch == 0x47
}

/*
invadersTarget is the column the left of the ship goes to: the nearest
column of invaders, or away from a bomb falling near it. With no invader
in sight it drifts to the right.
*/
func invadersTarget(p *izpet.PET, ship int) int {
	target, distance := ship+1, 40
	for c := 0; c < 40; c++ {
		for r := 2; r < 17; r++ {
			ch := p.Peek(uint16(0x8000 + r*40 + c))
			if ch == 0x20 || invadersBomb(ch) {
				continue
			}
			if d := max(c-ship, ship-c); d < distance {
				target, distance = c, d
			}
			break
		}
	}
	for c := max(ship-1, 0); c <= min(ship+4, 39); c++ {
		for r := 17; r < 23; r++ {
			if invadersBomb(p.Peek(uint16(0x8000 + r*40 + c))) {
				if c > ship {
					return ship - 2
				}
				return ship + 4
			}
		}
	}
	return target
}

// invadersPlayer plays a game, firing every other step
type invadersPlayer struct {
	p    *izpet.PET
	step int
}

/*
play plays a step every few frames until done says so, recorded when there
is a recording: the keys 4 and 6 move the ship and A fires. The keys are
left as they are, for the next call.
*/
func (i *invadersPlayer) play(r *Recording, done func() bool) {
	const every = 3
	p := i.p
	for ; !done(); i.step++ {
		ship := invadersShip(p)
		to := invadersTarget(p, ship)
		p.PutKey('4', ship >= 0 && ship > to)
		p.PutKey('6', ship >= 0 && ship < to)
		p.PutKey('A', i.step%2 == 0)
		if r != nil {
			r.Run(every, every)
		} else {
			p.RunFrames(every)
		}
	}
}

// invadersPoints returns the points of the mystery ship shot, the number
// on the second row of the screen, or "" when there is none
func invadersPoints(p *izpet.PET) string {
	var row strings.Builder
	for c := 0; c < 40; c++ {
		if ch := p.Peek(uint16(0x8000 + 40 + c)); ch >= '0' && ch <= '9' {
			row.WriteByte(ch)
		}
	}
	return row.String()
}

// invadersScore is the score on the top left, and the ships left
func invadersScore(p *izpet.PET) (string, uint8) {
	return p.GetScreenText()[6:10], p.Peek(0x8000 + 29)
}

/*
spaceInvadersScreenshots is a game of Space Invaders on a 3032: how to hear
it, the title and the demo, a recording of the game, the mystery ship seen
and shot, game over and the title again. The game goes the same way every
time, the player does what it sees on the screen.
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

	// The game, the ship going for the invaders and away from the bombs
	pressKey(p, izpet.KeySpace, false)
	p.RunFrames(120)
	player := invadersPlayer{p: p}
	playing := Record(p, label)
	frames := 0
	player.play(playing, func() bool { frames += 3; return frames > 60*20 })
	must(t, album.SaveRecording(playing, "playing", 200))

	// The mystery ship, and later one shot, its points where it was
	over := func() bool { _, ships := invadersScore(p); return ships == '0' }
	player.play(nil, func() bool {
		return over() || strings.Contains(p.GetScreenText(), "◤●●●◥")
	})
	must(t, album.ScreenshotOf(p, "mystery", label))
	player.play(nil, func() bool {
		return over() || invadersPoints(p) != ""
	})
	if over() {
		t.Fatal("the game ended before the mystery ship was shot")
	}
	must(t, album.ScreenshotOf(p, "hit", label))

	// The last ship lost, game over and the title again
	player.play(nil, over)
	p.ReleaseAllKeys()
	waitForText(t, p, "GAME OVER", 300)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "over", label))
	waitForText(t, p, "PLAY", 60*30)
	p.RunFrames(30)
	must(t, album.ScreenshotOf(p, "again", label))
}
