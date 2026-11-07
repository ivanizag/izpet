package component

/*
MOS 6520 Peripheral Interface Adapter (PIA), the same as the Motorola 6821
See:

	http://archive.6502.org/datasheets/mos_6520.pdf

Registers, selected by RS1 and RS0:

	0: PRA or DDRA, as the bit 2 of CRA selects
	1: CRA
	2: PRB or DDRB, as the bit 2 of CRB selects
	3: CRB

Control registers:

	bit 0:   C1 interrupt enable
	bit 1:   C1 active edge, 1 for the rising one
	bit 2:   1 to reach the peripheral register, 0 for the direction register
	bit 3-5: C2 control. With bit 5 off C2 is an input, with bit 3 enabling
	         its interrupt and bit 4 selecting the edge as bit 1 for C1. With
	         bit 5 on it is an output: set to bit 3 when bit 4 is on, a
	         handshake (bit 3 off) or a pulse (bit 3 on) when bit 4 is off.
	bit 6:   C2 interrupt flag, read only
	bit 7:   C1 interrupt flag, read only

Reading the peripheral register clears the two flags of its side.

Not implemented: the timing of the handshake and pulse outputs. A pulse is
too short for the PET to see and the handshake goes back up on the next C1
edge; the line stays up in both modes.
*/
type MOS6520 struct {
	a mos6520Port
	b mos6520Port
}

type mos6520Port struct {
	or  uint8 // Output register
	ddr uint8 // Data direction register, 1 is output
	in  uint8 // The levels on the input pins
	cr  uint8 // Control register

	c1, c2 bool // Levels of the control lines
}

const (
	mos6520CrC1IrqEnable   uint8 = 1 << 0
	mos6520CrC1RisingEdge  uint8 = 1 << 1
	mos6520CrPeripheral    uint8 = 1 << 2
	mos6520CrC2IrqEnable   uint8 = 1 << 3 // With C2 as input
	mos6520CrC2OutputLevel uint8 = 1 << 3 // With C2 as manual output
	mos6520CrC2RisingEdge  uint8 = 1 << 4 // With C2 as input
	mos6520CrC2Manual      uint8 = 1 << 4 // With C2 as output
	mos6520CrC2Output      uint8 = 1 << 5
	mos6520CrC2Flag        uint8 = 1 << 6
	mos6520CrC1Flag        uint8 = 1 << 7
)

// Read returns the value of a register
func (p *MOS6520) Read(reg uint8) uint8 {
	switch reg & 3 {
	case 0:
		return p.a.readPeripheral(true)
	case 1:
		return p.a.cr
	case 2:
		return p.b.readPeripheral(false)
	default:
		return p.b.cr
	}
}

// Write sets the value of a register
func (p *MOS6520) Write(reg uint8, value uint8) {
	switch reg & 3 {
	case 0:
		p.a.writePeripheral(value)
	case 1:
		p.a.writeControl(value)
	case 2:
		p.b.writePeripheral(value)
	default:
		p.b.writeControl(value)
	}
}

func (port *mos6520Port) readPeripheral(isA bool) uint8 {
	if port.cr&mos6520CrPeripheral == 0 {
		return port.ddr
	}
	port.cr &^= mos6520CrC1Flag | mos6520CrC2Flag
	if isA {
		// The port A reads the pins, an output pin pulled down outside
		// reads as 0
		return port.pins()
	}
	// The port B reads the output register for the output pins
	return (port.or & port.ddr) | (port.in &^ port.ddr)
}

func (port *mos6520Port) writePeripheral(value uint8) {
	if port.cr&mos6520CrPeripheral == 0 {
		port.ddr = value
	} else {
		port.or = value
	}
}

func (port *mos6520Port) writeControl(value uint8) {
	// The flags are read only
	port.cr = (port.cr & (mos6520CrC1Flag | mos6520CrC2Flag)) | (value & 0x3f)
	if port.cr&mos6520CrC2Output != 0 {
		port.c2 = port.c2OutputLevel()
	}
}

// pins returns the levels of the port pins. The inputs are pulled up when
// nothing drives them, and the outputs can be pulled down from outside.
func (port *mos6520Port) pins() uint8 {
	return (port.or | ^port.ddr) & port.in
}

func (port *mos6520Port) c2OutputLevel() bool {
	if port.cr&mos6520CrC2Manual != 0 {
		return port.cr&mos6520CrC2OutputLevel != 0
	}
	// Handshake and pulse modes, the line rests high
	return true
}

func (port *mos6520Port) setC1(level bool) {
	if port.c1 == level {
		return
	}
	port.c1 = level
	if level == (port.cr&mos6520CrC1RisingEdge != 0) {
		port.cr |= mos6520CrC1Flag
	}
}

func (port *mos6520Port) setC2(level bool) {
	if port.cr&mos6520CrC2Output != 0 || port.c2 == level {
		return
	}
	port.c2 = level
	if level == (port.cr&mos6520CrC2RisingEdge != 0) {
		port.cr |= mos6520CrC2Flag
	}
}

func (port *mos6520Port) irq() bool {
	if port.cr&mos6520CrC1Flag != 0 && port.cr&mos6520CrC1IrqEnable != 0 {
		return true
	}
	return port.cr&mos6520CrC2Flag != 0 &&
		port.cr&mos6520CrC2Output == 0 &&
		port.cr&mos6520CrC2IrqEnable != 0
}

func (port *mos6520Port) reset() {
	port.or = 0
	port.ddr = 0
	port.cr = 0
	port.c2 = false
}

// Reset clears the registers, as the RES pin
func (p *MOS6520) Reset() {
	p.a.reset()
	p.b.reset()
}

// InterruptAsserted returns true when IRQA or IRQB is asserted. On the PET
// both go to the processor IRQ line.
func (p *MOS6520) InterruptAsserted() bool {
	return p.a.irq() || p.b.irq()
}

// GetPortA returns the levels of the port A pins
func (p *MOS6520) GetPortA() uint8 {
	return p.a.pins()
}

// GetPortB returns the levels of the port B pins
func (p *MOS6520) GetPortB() uint8 {
	return p.b.pins()
}

// SetInputA sets the levels driven from outside on the port A pins
func (p *MOS6520) SetInputA(value uint8) {
	p.a.in = value
}

// SetInputB sets the levels driven from outside on the port B pins
func (p *MOS6520) SetInputB(value uint8) {
	p.b.in = value
}

// SetCA1 drives the CA1 pin
func (p *MOS6520) SetCA1(level bool) {
	p.a.setC1(level)
}

// SetCA2 drives the CA2 pin, ignored while it is an output
func (p *MOS6520) SetCA2(level bool) {
	p.a.setC2(level)
}

// SetCB1 drives the CB1 pin
func (p *MOS6520) SetCB1(level bool) {
	p.b.setC1(level)
}

// SetCB2 drives the CB2 pin, ignored while it is an output
func (p *MOS6520) SetCB2(level bool) {
	p.b.setC2(level)
}

// GetCA2 returns the level of the CA2 pin when it is an output, high otherwise
func (p *MOS6520) GetCA2() bool {
	return p.a.cr&mos6520CrC2Output == 0 || p.a.c2
}

// GetCB2 returns the level of the CB2 pin when it is an output, high otherwise
func (p *MOS6520) GetCB2() bool {
	return p.b.cr&mos6520CrC2Output == 0 || p.b.c2
}
