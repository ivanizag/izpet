package izpet

import (
	"strings"
	"testing"
)

// A BASIC listing pasted in uppercase, with CR LF, is typed as BASIC in the
// graphics mode and runs
func TestPasteGraphicsMode(t *testing.T) {
	p := newTestPET(t, "3032")
	testWaitText(t, p, "READY.")
	p.SendPaste("10 PRINT \"SUM\";6*7\r\n20 END\r\nRUN\r\n")
	testWaitText(t, p, "SUM 42")
}

// In the lowercase mode the letters are typed as they are, the uppercase
// ones with shift
func TestPasteLowercaseMode(t *testing.T) {
	p := newTestPET(t, "8032")
	testWaitText(t, p, "READY.")
	p.SendPaste("print \"Hello\"\n")
	p.RunUntil(testMaxFrames, func() bool { return !p.IsTyping() })
	p.RunFrames(10)
	if !strings.Contains(p.GetScreenText(), "\nHello") {
		t.Errorf("Hello expected with the H uppercase, the screen is:\n%v", p.GetScreenText())
	}
}
