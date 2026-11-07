package izpet

import (
	"github.com/ivanizag/izpet/component"
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

	viaPB5Retrace = 0x20
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
	io.reset()
	return &io
}

func (io *ioE8Page) reset() {
	io.pia1.Reset()
	io.pia2.Reset()
	io.via.Reset()

	// Nothing on the IEEE-488 bus, the user port or the cassette ports
	io.pia1.SetInputA(0xff)
	io.pia1.SetInputB(0xff)
	io.pia2.SetInputA(0xff)
	io.pia2.SetInputB(0xff)
	io.via.SetInputA(0xff)
	io.via.SetInputB(0xff)
	io.pia1.SetCA1(true)
	io.pia2.SetCA1(true)
	io.pia2.SetCB1(true)
	io.via.SetCA1(true)
	io.via.SetCB1(true)
}

// updateInputs refreshes the inputs that depend on the outputs
func (io *ioE8Page) updateInputs() {
	row := io.pia1.GetPortA() & 0x0f
	io.pia1.SetInputB(io.p.keyboard.readRow(row))
}

// updateRetrace drives the retrace signals from the video
func (io *ioE8Page) updateRetrace() {
	v := io.p.video
	io.pia1.SetCB1(v.retraceSignal())
	pb := uint8(0xff)
	if v.blank {
		pb &^= viaPB5Retrace
	}
	io.via.SetInputB(pb)
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
