package izpet

import (
	"github.com/ivanizag/izpet/component"
	"github.com/ivanizag/izpet/ieee488"
)

/*
The I/O page at $e800. The chips are selected by single address bits, an
address with several of them set reaches several chips:

	A4: PIA1 at $e810, keyboard, cassette #1, the retrace interrupt
	A5: PIA2 at $e820, IEEE-488 data
	A6: VIA at $e840, IEEE-488 handshake, user port, sound, character set
	A7: CRTC at $e880, on the CRTC boards

A read from several chips returns the AND of them.

PIA1:
	PA0-3: keyboard row select           CA1: cassette #1 read
	PA4:   cassette #1 sense             CA2: IEEE EOI out, screen blank on the 2001
	PA5:   cassette #2 sense             CB1: retrace, the 60 Hz interrupt
	PA6:   IEEE EOI in                   CB2: cassette #1 motor
	PA7:   diagnostic sense
	PB0-7: keyboard columns

PIA2:
	PA0-7: IEEE DI1-8                    CA1: IEEE ATN in
	PB0-7: IEEE DO1-8                    CA2: IEEE NDAC out
	                                     CB1: IEEE SRQ in
	                                     CB2: IEEE DAV out

VIA:
	PA0-7: user port                     CA1: user port
	PB0:   IEEE NDAC in                  CA2: character set, high for lowercase
	PB1:   IEEE NRFD out                 CB1: cassette #2 read
	PB2:   IEEE ATN out                  CB2: user port and sound
	PB3:   cassette write
	PB4:   cassette #2 motor
	PB5:   retrace, low on the vertical blanking
	PB6:   IEEE NRFD in
	PB7:   IEEE DAV in

See:
  - PET-Interfaces.txt, by William Levak
  - VICE, src/pet/petpia1.c, petpia2.c and petvia.c
*/

const (
	ioSelectPia1 = 0x10
	ioSelectPia2 = 0x20
	ioSelectVia  = 0x40
	ioSelectCrtc = 0x80

	pia1PA4Sense1 = 0x10
	pia1PA5Sense2 = 0x20
	pia1PA6EOI    = 0x40
	viaPB0NDAC    = 0x01
	viaPB1NRFD    = 0x02
	viaPB2ATN     = 0x04
	viaPB3Write   = 0x08
	viaPB4Motor2  = 0x10
	viaPB5Retrace = 0x20
	viaPB6NRFD    = 0x40
	viaPB7DAV     = 0x80
	viaPBOutputs  = viaPB1NRFD | viaPB2ATN | viaPB3Write | viaPB4Motor2
	pia1PAUnwired = 0x8f // The keyboard row outputs and the diagnostic sense
)

type ioE8Page struct {
	p    *PET
	pia1 component.MOS6520
	pia2 component.MOS6520
	via  component.MOS6522
	crtc *component.MC6845 // nil without CRTC
}

func newIOE8Page(p *PET, hasCrtc bool) *ioE8Page {
	var io ioE8Page
	io.p = p
	if hasCrtc {
		io.crtc = &component.MC6845{}
	}
	return &io
}

func (io *ioE8Page) reset() {
	io.pia1.Reset()
	io.pia2.Reset()
	io.via.Reset()

	// Nothing on the user port
	io.via.SetInputA(0xff)
	io.pia1.SetInputB(0xff)
	io.pia2.SetInputB(0xff)
	io.via.SetCA1(true)
	io.updateInputs()
	io.updateBus()
}

// updateInputs sets the levels the chips see on their input pins
func (io *ioE8Page) updateInputs() {
	p := io.p
	bus := p.bus.Levels()

	// PIA1: keyboard, cassette sense, EOI
	pa := uint8(pia1PAUnwired)
	if !p.cassettes[0].inserted {
		pa |= pia1PA4Sense1
	}
	if !p.cassettes[1].inserted {
		pa |= pia1PA5Sense2
	}
	if bus.EOI {
		pa |= pia1PA6EOI
	}
	io.pia1.SetInputA(pa)
	io.pia1.SetInputB(p.keyboard.readRow(io.pia1.GetPortA() & 0x0f))
	io.pia1.SetCA1(p.cassettes[0].level)

	// PIA2: IEEE data, ATN and SRQ
	io.pia2.SetInputA(bus.Data)
	io.pia2.SetCA1(bus.ATN)
	io.pia2.SetCB1(bus.SRQ)

	// VIA: IEEE handshake, retrace, cassette #2
	pb := uint8(viaPBOutputs)
	if bus.NDAC {
		pb |= viaPB0NDAC
	}
	if !p.video.blank {
		pb |= viaPB5Retrace
	}
	if bus.NRFD {
		pb |= viaPB6NRFD
	}
	if bus.DAV {
		pb |= viaPB7DAV
	}
	io.via.SetInputB(pb)
	io.via.SetCB1(p.cassettes[1].level)
}

// updateBus sets the IEEE-488 lines driven by the PET and lets the devices
// react
func (io *ioE8Page) updateBus() {
	viaPB := io.via.GetPortB()
	io.p.bus.SetController(ieee488.Lines{
		Data: io.pia2.GetPortB(),
		EOI:  io.pia1.GetCA2(),
		DAV:  io.pia2.GetCB2(),
		NRFD: viaPB&viaPB1NRFD != 0,
		NDAC: io.pia2.GetCA2(),
		ATN:  viaPB&viaPB2ATN != 0,
		SRQ:  true,
	})
}

// updateRetrace drives the retrace signals from the video
func (io *ioE8Page) updateRetrace() {
	io.pia1.SetCB1(io.p.video.retraceSignal())
}

// cassetteLines returns the motor of each deck, on when true, and the
// level of the write line
func (io *ioE8Page) cassetteLines() (motor1 bool, motor2 bool, write bool) {
	viaPB := io.via.GetPortB()
	return !io.pia1.GetCB2(), viaPB&viaPB4Motor2 == 0, viaPB&viaPB3Write != 0
}

func (io *ioE8Page) peek(address uint8) uint8 {
	io.updateInputs()

	value := uint8(0xff)
	selected := false
	if address&ioSelectPia1 != 0 {
		value &= io.pia1.Read(address)
		selected = true
	}
	if address&ioSelectPia2 != 0 {
		value &= io.pia2.Read(address)
		selected = true
	}
	if address&ioSelectVia != 0 {
		value &= io.via.Read(address)
		selected = true
	}
	if address&ioSelectCrtc != 0 && io.crtc != nil {
		value &= io.crtc.Read(address&1 != 0)
		selected = true
	}

	if !selected {
		return openBus(addressIO)
	}
	return value
}

func (io *ioE8Page) poke(address uint8, value uint8) {
	if address&ioSelectPia1 != 0 {
		io.pia1.Write(address, value)
	}
	if address&ioSelectPia2 != 0 {
		io.pia2.Write(address, value)
	}
	if address&ioSelectVia != 0 {
		io.via.Write(address, value)
	}
	if address&ioSelectCrtc != 0 && io.crtc != nil {
		io.crtc.Write(address&1 != 0, value)
	}
	if address&(ioSelectPia1|ioSelectPia2|ioSelectVia) != 0 {
		io.updateBus()
	}
}

func (io *ioE8Page) tick(cycles uint64) {
	io.via.Tick(cycles)
}

func (io *ioE8Page) irq() bool {
	return io.pia1.InterruptAsserted() ||
		io.pia2.InterruptAsserted() ||
		io.via.InterruptAsserted()
}

// isLowerCase returns true when the character set with lowercase letters
// is selected
func (io *ioE8Page) isLowerCase() bool {
	return io.via.GetCA2()
}
