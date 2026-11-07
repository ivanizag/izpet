package izpet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMaxFrames = 600 // 10 seconds

func newTestPET(t *testing.T, model string) *PET {
	t.Helper()
	config := NewConfiguration()
	config.Model = model
	config.Speed = SpeedFull
	err := config.Validate()
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewPET(config)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// testWaitText runs until the text is on the screen, case insensitive
func testWaitText(t *testing.T, p *PET, text string) {
	t.Helper()
	text = strings.ToUpper(text)
	found := p.RunUntil(testMaxFrames, func() bool {
		return strings.Contains(strings.ToUpper(p.GetScreenText()), text)
	})
	if !found {
		t.Fatalf("'%v' not found on the screen:\n%v", text, p.GetScreenText())
	}
}

// testType types a text and waits until it is typed
func testType(t *testing.T, p *PET, text string) {
	t.Helper()
	p.TypeText(text)
	if !p.RunUntil(testMaxFrames*10, func() bool { return !p.IsTyping() }) {
		t.Fatal("typing did not end")
	}
}

// testWritePrg writes a .prg file
func testWritePrg(t *testing.T, address uint16, data []uint8) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "test.prg")
	content := append([]uint8{uint8(address), uint8(address >> 8)}, data...)
	err := os.WriteFile(name, content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	return name
}
