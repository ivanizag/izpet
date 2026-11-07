package izpet

import (
	"fmt"
	"image"
	"sync/atomic"

	"github.com/ivanizag/iz6502"
)

// PET represents all the components and state of the emulated machine
type PET struct {
	Name     string
	model    *model
	cpu      *iz6502.State
	mmu      *memoryManager
	io       *ioE8Page
	video    *video
	keyboard *keyboard
	sound    sound

	commandChannel chan command

	cycles          uint64
	frames          uint64
	cycleDurationNs float64 // Current speed. Inverse of the cpu clock in Ghz, 0 for full speed
	paused          atomic.Bool
	cpuTrace        bool
	currentFreqMHz  atomic.Uint64

	// Programs waiting for BASIC to be ready to be loaded
	pendingPrograms []pendingProgram
}

// NewPET builds a PET from a configuration
func NewPET(config *Configuration) (*PET, error) {
	m, err := getModel(config.Model)
	if err != nil {
		return nil, err
	}

	var p PET
	p.Name = m.description
	p.model = m
	p.commandChannel = make(chan command, 100)
	p.cpuTrace = config.hasTracer("cpu")
	p.cycleDurationNs = config.cycleDurationNs

	ramKB := m.ramKB
	if config.RamKB != 0 {
		ramKB = config.RamKB
	}
	if ramKB < 1 || ramKB > 32 {
		return nil, fmt.Errorf("the RAM size must be from 1 to 32 KB, not %v", ramKB)
	}

	screenSize := 0x400
	if m.video == videoCRTC80 {
		screenSize = 0x800
	}
	p.mmu = newMemoryManager(&p, ramKB, screenSize)

	roms := m.roms
	if config.Hz50 {
		if m.editor50Hz == "" {
			return nil, fmt.Errorf("the model %v has no 50 Hz editor ROM", m.name)
		}
		roms = append([]romChip(nil), roms...)
		for i := range roms {
			if roms[i].address == 0xe000 {
				roms[i].file = m.editor50Hz
			}
		}
	}
	roms = append(roms, config.optionRoms...)
	for _, chip := range roms {
		data, err := LoadResource(chip.file)
		if err != nil {
			return nil, err
		}
		err = p.mmu.loadRom(chip.address, data)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", chip.file, err)
		}
	}

	charGen, err := LoadResource(m.charGen)
	if err != nil {
		return nil, err
	}

	p.io = newIOE8Page(&p, m.video != videoTTL)
	p.video = newVideo(m.video, charGen, p.io.crtc)
	p.keyboard = newKeyboard(m.keyboard)

	p.cpu = iz6502.NewNMOS6502(p.mmu)
	p.cpu.SetTrace(p.cpuTrace)

	for _, program := range config.programs {
		err = p.LoadProgram(program)
		if err != nil {
			return nil, err
		}
	}
	return &p, nil
}

// GetImage returns the image of the screen at the end of the last frame
func (p *PET) GetImage() *image.RGBA {
	return p.video.render()
}

// Is80Columns returns true for the 80 columns models, their pixels are
// half as wide as the ones of the 40 columns models
func (p *PET) Is80Columns() bool {
	return p.model.video == videoCRTC80
}

// HasKey returns true if the keyboard of the model has the key
func (p *PET) HasKey(key Key) bool {
	return p.keyboard.hasKey(key)
}

// IsChicletKeyboard returns true for the chiclet keyboard of the 2001. It has
// the same matrix as the graphics keyboard.
func (p *PET) IsChicletKeyboard() bool {
	return p.model.basic == 1
}

// IsKeyDown returns true while a key is down on the keyboard matrix, pressed
// with PutKey or typing a character. Safe to call from any goroutine.
func (p *PET) IsKeyDown(key Key) bool {
	return p.keyboard.isKeyDown(key)
}

// Keys returns the keys of the keyboard of the model
func (p *PET) Keys() []Key {
	return p.keyboard.keys()
}

// IsBusinessKeyboard returns true for the business keyboard
func (p *PET) IsBusinessKeyboard() bool {
	return p.model.keyboard == keyboardBusiness
}

// PutKey presses or releases a key. Safe to call from any goroutine.
func (p *PET) PutKey(key Key, pressed bool) {
	p.keyboard.putKey(key, pressed)
}

// ReleaseAllKeys releases the keys pressed with PutKey
func (p *PET) ReleaseAllKeys() {
	p.keyboard.releaseAll()
}

// PutChar types a character with the keys needed, it returns false if the
// keyboard has no key for it. Safe to call from any goroutine.
func (p *PET) PutChar(ch rune) bool {
	return p.keyboard.putChar(ch)
}

// TypeText types a text, see keyboard.putText. Safe to call from any
// goroutine.
func (p *PET) TypeText(text string) {
	p.keyboard.putText(text)
}

// IsTyping returns true while there are characters waiting to be typed
func (p *PET) IsTyping() bool {
	return p.keyboard.isTyping()
}

// IsPaused returns true when emulator is paused
func (p *PET) IsPaused() bool {
	return p.paused.Load()
}

// IsFullSpeed returns true when the emulation runs as fast as it can
func (p *PET) IsFullSpeed() bool {
	return p.cycleDurationNs == 0
}

// GetCycles returns the cycles run since the start
func (p *PET) GetCycles() uint64 {
	return p.cycles
}

// GetFrames returns the frames run since the start
func (p *PET) GetFrames() uint64 {
	return p.frames
}

// GetPC returns the program counter
func (p *PET) GetPC() uint16 {
	pc, _ := p.cpu.GetPCAndSP()
	return pc
}

// Peek returns the byte at an address as the CPU would read it, without
// side effects for the RAM and ROM, but reading the I/O has them
func (p *PET) Peek(address uint16) uint8 {
	return p.mmu.Peek(address)
}

// Poke writes a byte as the CPU would do
func (p *PET) Poke(address uint16, value uint8) {
	p.mmu.Poke(address, value)
}

// DumpRegisters returns the registers of the CPU as a string
func (p *PET) DumpRegisters() string {
	pc, sp := p.cpu.GetPCAndSP()
	a, x, y, flags := p.cpu.GetAXYP()
	return fmt.Sprintf("PC=$%04x A=$%02x X=$%02x Y=$%02x SP=$%02x P=$%02x",
		pc, a, x, y, sp, flags)
}

// Disasm returns the disassembly of some instructions
func (p *PET) Disasm(address uint16, count int) string {
	s := ""
	for i := 0; i < count; i++ {
		var line string
		line, address = p.cpu.DisasmInstruction(address)
		s += line + "\n"
	}
	return s
}

// GetMessageImage returns an image of a text drawn as the screen of the
// machine would show it, for the frontends to show the help. The text fits
// in 40 columns and 25 rows, or 80 columns on the 80 columns models.
func (p *PET) GetMessageImage(message string) *image.RGBA {
	return p.video.renderMessage(message, p.model.basic == 1)
}

// GetGlyph returns the 8 rows of pixels of a screen code in the graphics
// character set of the model, the top bit is the leftmost pixel
func (p *PET) GetGlyph(screenCode uint8) [8]uint8 {
	var glyph [8]uint8
	base := int(screenCode&0x7f) * charGenLines
	for i := range glyph {
		glyph[i] = p.video.charGen[(base+i)%len(p.video.charGen)]
	}
	return glyph
}
