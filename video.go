package izpet

import (
	"image"
	"image/color"
	"strings"
	"sync"

	"github.com/ivanizag/izpet/component"
)

/*
The video of the PET. There is no frame buffer, the screen RAM at $8000 is
read through the character ROM as the beam goes.

The timing is the heartbeat of the machine: the vertical blanking reaches PIA1
CB1, the 60 Hz interrupt, and VIA PB5, the retrace bit BASIC 1 waits on before
writing to the screen.

The discrete logic boards have a fixed timing: 64 cycles per scan line and 260
lines, 200 of them with text, for 16640 cycles and 60.1 frames per second.
The blanking signal goes low after the last text line and stays low for 60
lines.

On the CRTC boards the 6545 makes the timing from its registers, as the editor
ROM programs them for 50 or 60 Hz. It clocks a character per cycle, the 80
column board shows two characters per clock. The blanking signal for PB5 is
the 6545 display enable, the interrupt is on its vertical sync.

Two address lines of the 6545 not used to read the screen have uses: MA12 low
reverses the whole screen, and MA13 selects the upper half of a 4K character
ROM, not used with the 2K ones.

See:
  - VICE, src/pet/pet.c and src/crtc/crtc.c
  - PET-Interfaces.txt, by William Levak
*/

const (
	ttlCyclesPerLine = 64
	ttlLinesPerFrame = 260
	ttlTextLines     = 200

	charWidth      = 8
	charGenLines   = 8 // Pixel lines of each glyph in the ROM
	charGenGlyphs  = 128
	charGenSetSize = charGenGlyphs * charGenLines

	crtcCanvasRows      = 25
	crtcCanvasCharLines = 10
	crtcRevSwitch       = 0x1000 // MA12
)

type video struct {
	board   videoBoard
	crtc    *component.MC6845
	charGen []uint8
	hwCols  int // Characters shown per clock

	// Beam position
	lineCycle uint64
	line      int

	// Timing of the current frame
	cyclesPerLine uint64
	linesPerFrame int
	textLines     int
	vsyncStart    int
	vsyncEnd      int

	// Signals
	blank bool // Outside the text area, PB5 and the CB1 of the TTL boards
	vsync bool // The CB1 of the CRTC boards

	// The state of the screen at the end of the last frame, for the
	// frontends to render
	snapshotMutex sync.Mutex
	snapshot      videoSnapshot
}

type videoSnapshot struct {
	screen       [0x800]uint8
	lowerCase    bool
	crtcRegs     [18]uint8
	crtcRunnning bool
}

func newVideo(board videoBoard, charGen []uint8, crtc *component.MC6845) *video {
	var v video
	v.board = board
	v.crtc = crtc
	v.charGen = charGen
	v.hwCols = 1
	if board == videoCRTC80 {
		v.hwCols = 2
	}
	v.startFrame()
	return &v
}

// startFrame latches the timing of the next frame
func (v *video) startFrame() {
	v.line = 0
	v.cyclesPerLine = ttlCyclesPerLine
	v.linesPerFrame = ttlLinesPerFrame
	v.textLines = ttlTextLines
	v.vsyncStart = ttlTextLines
	v.vsyncEnd = ttlLinesPerFrame
	if v.board == videoTTL || !v.isCrtcProgrammed() {
		return
	}

	c := v.crtc
	charLines := int(c.Reg(component.MC6845ScanLines)&0x1f) + 1
	v.cyclesPerLine = uint64(c.Reg(component.MC6845HorizontalTotal)) + 1
	v.linesPerFrame = (int(c.Reg(component.MC6845VerticalTotal)&0x7f)+1)*charLines +
		int(c.Reg(component.MC6845VerticalAdjust)&0x1f)
	v.textLines = int(c.Reg(component.MC6845VerticalDisplayed)&0x7f) * charLines
	v.vsyncStart = int(c.Reg(component.MC6845VerticalSync)&0x7f) * charLines
	v.vsyncEnd = v.vsyncStart + c.VerticalSyncLines()
}

// isCrtcProgrammed is false while the CRTC has values that would not make a
// frame, at power on before the editor ROM sets it up
func (v *video) isCrtcProgrammed() bool {
	c := v.crtc
	cycles := (int(c.Reg(component.MC6845HorizontalTotal)) + 1) *
		(int(c.Reg(component.MC6845VerticalTotal)&0x7f) + 1) *
		(int(c.Reg(component.MC6845ScanLines)&0x1f) + 1)
	return c.Reg(component.MC6845HorizontalTotal) >= 16 && cycles >= 5000
}

// tick advances the beam, it returns true when a frame ends
func (v *video) tick(cycles uint64) bool {
	frameEnded := false
	v.lineCycle += cycles
	for v.lineCycle >= v.cyclesPerLine {
		v.lineCycle -= v.cyclesPerLine
		v.line++
		if v.line >= v.linesPerFrame {
			v.startFrame()
			frameEnded = true
		}
		v.updateSignals()
	}
	return frameEnded
}

func (v *video) updateSignals() {
	v.blank = v.line >= v.textLines
	v.vsync = v.line >= v.vsyncStart && v.line < v.vsyncEnd
	if v.crtc != nil {
		v.crtc.VBlank = v.blank
	}
}

// retraceSignal is the level on PIA1 CB1, low on the retrace
func (v *video) retraceSignal() bool {
	if v.board == videoTTL {
		return !v.blank
	}
	return !v.vsync
}

// takeSnapshot keeps what is needed to render the frame
func (v *video) takeSnapshot(screen []uint8, lowerCase bool) {
	v.snapshotMutex.Lock()
	defer v.snapshotMutex.Unlock()

	copy(v.snapshot.screen[:], screen)
	v.snapshot.lowerCase = lowerCase
	if v.crtc != nil {
		for i := range v.snapshot.crtcRegs {
			v.snapshot.crtcRegs[i] = v.crtc.Reg(i)
		}
		v.snapshot.crtcRunnning = v.isCrtcProgrammed()
	}
}

var (
	colorBackground = color.RGBA{0x00, 0x00, 0x00, 0xff}
	colorForeground = color.RGBA{0x41, 0xff, 0x00, 0xff} // Green phosphor
)

// canvasSize is the size in pixels of the images of the screen
func (v *video) canvasSize() (int, int) {
	switch v.board {
	case videoCRTC40:
		return 40 * charWidth, crtcCanvasRows * crtcCanvasCharLines
	case videoCRTC80:
		return 80 * charWidth, crtcCanvasRows * crtcCanvasCharLines
	default:
		return 40 * charWidth, 25 * charGenLines
	}
}

// textLayout is the geometry of the text on the screen
type textLayout struct {
	columns    int
	rows       int
	charLines  int  // Scan lines per row
	start      int  // Offset on the screen RAM of the first character
	screenSize int  // The screen RAM wraps around at this size
	reverse    bool // The whole screen in reverse video
	lowerCase  bool // The character set with lowercase
}

// render draws the last frame
func (v *video) render() *image.RGBA {
	v.snapshotMutex.Lock()
	s := v.snapshot
	v.snapshotMutex.Unlock()

	layout := textLayout{
		columns:    40,
		rows:       25,
		charLines:  charGenLines,
		screenSize: 0x400,
		lowerCase:  s.lowerCase,
	}
	if v.board != videoTTL {
		if !s.crtcRunnning {
			return v.newCanvas()
		}
		layout.columns = int(s.crtcRegs[component.MC6845HorizontalDisplayed]) * v.hwCols
		layout.rows = int(s.crtcRegs[component.MC6845VerticalDisplayed] & 0x7f)
		layout.charLines = int(s.crtcRegs[component.MC6845ScanLines]&0x1f) + 1
		ma := uint16(s.crtcRegs[component.MC6845StartHigh]&0x3f)<<8 |
			uint16(s.crtcRegs[component.MC6845StartLow])
		layout.start = int(ma&0x3ff) * v.hwCols
		layout.reverse = ma&crtcRevSwitch == 0
		layout.screenSize = 0x400 * v.hwCols
	}

	return v.renderText(s.screen[:], &layout)
}

// newCanvas returns an image of the screen with nothing on it
func (v *video) newCanvas() *image.RGBA {
	width, height := v.canvasSize()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0] = colorBackground.R
		img.Pix[i+1] = colorBackground.G
		img.Pix[i+2] = colorBackground.B
		img.Pix[i+3] = colorBackground.A
	}
	return img
}

// renderText draws the screen codes through the character ROM
func (v *video) renderText(screen []uint8, layout *textLayout) *image.RGBA {
	img := v.newCanvas()
	width, height := v.canvasSize()

	charSet := 0
	if layout.lowerCase {
		charSet = charGenSetSize
	}

	for row := 0; row < layout.rows; row++ {
		for col := 0; col < layout.columns; col++ {
			code := screen[(layout.start+row*layout.columns+col)%layout.screenSize]
			inverse := (code&0x80 != 0) != layout.reverse
			glyph := charSet + int(code&0x7f)*charGenLines
			for line := 0; line < layout.charLines; line++ {
				y := row*layout.charLines + line
				if y >= height {
					break
				}
				bits := uint8(0)
				if line < charGenLines {
					bits = v.charGen[(glyph+line)%len(v.charGen)]
				}
				if inverse {
					bits = ^bits
				}
				for x := 0; x < charWidth; x++ {
					px := col*charWidth + x
					if px >= width {
						break
					}
					if bits&(0x80>>x) != 0 {
						offset := y*img.Stride + px*4
						img.Pix[offset+0] = colorForeground.R
						img.Pix[offset+1] = colorForeground.G
						img.Pix[offset+2] = colorForeground.B
					}
				}
			}
		}
	}
	return img
}

/*
renderMessage draws a text with the character ROM of the machine, as the
screen would show it, in the lowercase character set. The frontends use it
for the help. The character ROM of BASIC 1 has the cases swapped.
*/
func (v *video) renderMessage(message string, swapCase bool) *image.RGBA {
	columns := 40 * v.hwCols
	rows := 25
	charLines := charGenLines
	if v.board != videoTTL {
		charLines = crtcCanvasCharLines
	}

	screen := make([]uint8, columns*rows)
	for i := range screen {
		screen[i] = 0x20
	}
	for row, line := range strings.Split(message, "\n") {
		col := 0
		for _, ch := range line {
			if row < rows && col < columns {
				screen[row*columns+col] = runeToScreenCode(ch, swapCase)
			}
			col++
		}
	}

	layout := textLayout{
		columns:    columns,
		rows:       rows,
		charLines:  charLines,
		screenSize: len(screen),
		lowerCase:  true,
	}
	return v.renderText(screen, &layout)
}

// runeToScreenCode returns the screen code of a character in the lowercase
// character set
func runeToScreenCode(ch rune, swapCase bool) uint8 {
	if swapCase {
		switch {
		case ch >= 'a' && ch <= 'z':
			ch = ch - 'a' + 'A'
		case ch >= 'A' && ch <= 'Z':
			ch = ch - 'A' + 'a'
		}
	}
	switch {
	case ch == '@':
		return 0x00
	case ch >= 'a' && ch <= 'z':
		return uint8(ch-'a') + 0x01
	case ch >= 'A' && ch <= 'Z':
		return uint8(ch-'A') + 0x41
	case ch == '[':
		return 0x1b
	case ch == '\\':
		return 0x1c
	case ch == ']':
		return 0x1d
	case ch >= 0x20 && ch < 0x40:
		return uint8(ch)
	default:
		return '?'
	}
}
