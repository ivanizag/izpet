package activities

import (
	"strings"
	"testing"

	"github.com/ivanizag/izpet"
)

/*
What the activities of izpet's documentation run on: a machine of the tests'
configuration, and the folder their pictures go to. The ROMs are in the
program, there is nothing to download.
*/

// activityImages is where the pictures of the pages go, a folder each
const activityImages = "../doc/activities/images"

// testConfig returns the configuration of a model, running as fast as it can
func testConfig(t *testing.T, model string) *izpet.Configuration {
	t.Helper()
	config := izpet.NewConfiguration()
	config.Model = model
	config.Speed = izpet.SpeedFull
	return config
}

func buildTestPET(t *testing.T, config *izpet.Configuration) *izpet.PET {
	t.Helper()
	must(t, config.Validate())
	p, err := izpet.NewPET(config)
	must(t, err)
	return p
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// waitForText runs the machine until the text is on the screen, case
// insensitive
func waitForText(t *testing.T, p *izpet.PET, text string, maxFrames uint64) {
	t.Helper()
	text = strings.ToUpper(text)
	if !p.RunUntil(maxFrames, func() bool {
		return strings.Contains(strings.ToUpper(p.GetScreenText()), text)
	}) {
		t.Fatalf("'%v' not found on the screen:\n%v", text, p.GetScreenText())
	}
}

// hasText returns true if the text is on the screen, case insensitive
func hasText(p *izpet.PET, text string) bool {
	return strings.Contains(strings.ToUpper(p.GetScreenText()), strings.ToUpper(text))
}

// waitForCursor runs the machine until the cursor, after the last line with
// a text, is shown: it blinks, and a picture taken at the wrong time has none
func waitForCursor(t *testing.T, p *izpet.PET, after string) {
	t.Helper()
	columns := 40
	if p.Is80Columns() {
		columns = 80
	}
	waitForText(t, p, after, 600)
	lines := strings.Split(strings.ToUpper(p.GetScreenText()), "\n")
	row := -1
	for i, line := range lines {
		if strings.Contains(line, strings.ToUpper(after)) {
			row = i
		}
	}
	if row < 0 {
		t.Fatalf("'%v' not found on the screen", after)
	}
	address := uint16(0x8000 + (row+1)*columns)
	if !p.RunUntil(120, func() bool { return p.Peek(address)&0x80 != 0 }) {
		t.Fatalf("the cursor is not shown")
	}
}

// pressKey presses a key, with shift when asked, for a few frames and lets
// it go, as a hand would
func pressKey(p *izpet.PET, key izpet.Key, shift bool) {
	if shift {
		p.PutKey(izpet.KeyShiftLeft, true)
	}
	p.PutKey(key, true)
	p.RunFrames(3)
	p.PutKey(key, false)
	if shift {
		p.PutKey(izpet.KeyShiftLeft, false)
	}
	p.RunFrames(3)
}

// pressKeys presses a key a number of times
func pressKeys(p *izpet.PET, key izpet.Key, shift bool, times int) {
	for i := 0; i < times; i++ {
		pressKey(p, key, shift)
	}
}

// typeText types a text and waits until it is typed. The letters are typed
// unshifted, uppercase in the graphics mode, as BASIC wants them.
func typeText(t *testing.T, p *izpet.PET, text string) {
	t.Helper()
	p.TypeText(text)
	waitTyped(t, p)
}

// typeExact types the characters as they are: the lowercase letters
// unshifted and the uppercase ones with shift, the graphics in the graphics
// mode
func typeExact(t *testing.T, p *izpet.PET, text string) {
	t.Helper()
	for _, ch := range text {
		if !p.PutChar(ch) {
			t.Fatalf("no key for %q", ch)
		}
	}
	waitTyped(t, p)
}

func waitTyped(t *testing.T, p *izpet.PET) {
	t.Helper()
	if !p.RunUntil(60*60*5, func() bool { return !p.IsTyping() }) {
		t.Fatal("typing did not end")
	}
}

// lineOf returns the row of the screen with a text, the last one if there
// are several
func lineOf(t *testing.T, p *izpet.PET, text string) int {
	t.Helper()
	row := -1
	for i, line := range strings.Split(strings.ToUpper(p.GetScreenText()), "\n") {
		if strings.Contains(line, strings.ToUpper(text)) {
			row = i
		}
	}
	if row < 0 {
		t.Fatalf("'%v' not found on the screen", text)
	}
	return row
}
