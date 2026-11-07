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
