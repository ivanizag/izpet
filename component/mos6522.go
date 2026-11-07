package component

/*
MOS 6522 Versatile Interface Adapter (VIA)
See:

	http://archive.6502.org/datasheets/mos_6522_preliminary_nov_1977.pdf
	http://archive.6502.org/datasheets/rockwell_r6522_via.pdf

Taken from izmac, that took it from izapple2. On the PET it has the IEEE-488
handshake lines, the user port, the character set select on CA2 and the sound
on CB2.

Implemented: ports A and B, timers T1 and T2, the IFR/IER interrupt logic,
the CA/CB control lines as inputs and as manual outputs, and the shift
register shifting out under T2 control, the free running mode the PET uses
for sound.
Not implemented: the handshake and pulse outputs (they rest high), the T2
pulse counting mode, PB7 driven by T1 and the other shift register modes.

Registers:

	0: ORB/IRB   1: ORA/IRA   2: DDRB   3: DDRA
	4: T1C-L     5: T1C-H     6: T1L-L  7: T1L-H
	8: T2C-L     9: T2C-H    10: SR    11: ACR
	12: PCR     13: IFR      14: IER   15: ORA no handshake

The timers advance with Tick(elapsedCycles).
*/
type MOS6522 struct {
	ora, orb   uint8 // Output registers
	ira, irb   uint8 // Input registers, the values on the input pins
	ddra, ddrb uint8 // Data direction registers, 1 is output

	t1counter int64 // Signed to detect underflows when catching up
	t1latch   uint16
	t1fired   bool
	t2counter int64
	t2latchL  uint8
	t2fired   bool

	sr, acr, pcr uint8
	ifr, ier     uint8

	srCycles  int64 // Cycles towards the next shift
	srBits    int   // Bits left to shift out, in the not free running mode
	srOutput  bool  // CB2 level driven by the shift register
	srShifted bool  // The shift register has driven CB2 since the mode was set

	// The levels on the control pins, to spot the transitions
	ca1, ca2, cb1, cb2 bool
}

const (
	mos6522IntCA2 uint8 = 1 << 0
	mos6522IntCA1 uint8 = 1 << 1
	mos6522IntSR  uint8 = 1 << 2
	mos6522IntCB2 uint8 = 1 << 3
	mos6522IntCB1 uint8 = 1 << 4
	mos6522IntT2  uint8 = 1 << 5
	mos6522IntT1  uint8 = 1 << 6

	mos6522AcrT1FreeRunning uint8 = 1 << 6

	// The shift register modes, in the bits 2 to 4 of the ACR
	mos6522AcrShiftMode         uint8 = 7 << 2
	mos6522AcrShiftOutFreeT2    uint8 = 4 << 2
	mos6522AcrShiftOutT2        uint8 = 5 << 2
	mos6522AcrShiftOutUnderT2   uint8 = 4 << 2 // Mask shared by the two above
	mos6522AcrShiftOutUnderMask uint8 = 6 << 2

	// The peripheral control register: the low nibble is for the port A
	// lines and the high one for the port B ones
	mos6522PcrCA1PositiveEdge uint8 = 1 << 0
	mos6522PcrCA2PositiveEdge uint8 = 1 << 2
	mos6522PcrCA2Output       uint8 = 1 << 3
	mos6522PcrCB1PositiveEdge uint8 = 1 << 4
	mos6522PcrCB2PositiveEdge uint8 = 1 << 6
	mos6522PcrCB2Output       uint8 = 1 << 7
)

// Read returns the value of a register
func (v *MOS6522) Read(reg uint8) uint8 {
	switch reg & 0x0f {
	case 0:
		v.ifr &^= mos6522IntCB1 | mos6522IntCB2
		return (v.irb &^ v.ddrb) | (v.orb & v.ddrb)
	case 1:
		v.ifr &^= mos6522IntCA1 | mos6522IntCA2
		return v.GetPortA()
	case 15:
		return v.GetPortA()
	case 2:
		return v.ddrb
	case 3:
		return v.ddra
	case 4:
		v.ifr &^= mos6522IntT1
		return uint8(uint16(v.t1counter))
	case 5:
		return uint8(uint16(v.t1counter) >> 8)
	case 6:
		return uint8(v.t1latch)
	case 7:
		return uint8(v.t1latch >> 8)
	case 8:
		v.ifr &^= mos6522IntT2
		return uint8(uint16(v.t2counter))
	case 9:
		return uint8(uint16(v.t2counter) >> 8)
	case 10:
		v.ifr &^= mos6522IntSR
		v.startShift()
		return v.sr
	case 11:
		return v.acr
	case 12:
		return v.pcr
	case 13:
		ifr := v.ifr & 0x7f
		if ifr&v.ier != 0 {
			ifr |= 0x80
		}
		return ifr
	case 14:
		return v.ier | 0x80
	}
	return 0
}

// Write sets the value of a register
func (v *MOS6522) Write(reg uint8, value uint8) {
	switch reg & 0x0f {
	case 0:
		v.ifr &^= mos6522IntCB1 | mos6522IntCB2
		v.orb = value
	case 1:
		v.ifr &^= mos6522IntCA1 | mos6522IntCA2
		v.ora = value
	case 15:
		v.ora = value
	case 2:
		v.ddrb = value
	case 3:
		v.ddra = value
	case 4, 6:
		v.t1latch = (v.t1latch & 0xff00) | uint16(value)
	case 5:
		// Load the counter from the latch and start counting
		v.t1latch = (v.t1latch & 0x00ff) | uint16(value)<<8
		v.t1counter = int64(v.t1latch)
		v.t1fired = false
		v.ifr &^= mos6522IntT1
	case 7:
		v.t1latch = (v.t1latch & 0x00ff) | uint16(value)<<8
		v.ifr &^= mos6522IntT1
	case 8:
		v.t2latchL = value
	case 9:
		v.t2counter = int64(value)<<8 | int64(v.t2latchL)
		v.t2fired = false
		v.ifr &^= mos6522IntT2
	case 10:
		v.ifr &^= mos6522IntSR
		v.sr = value
		v.startShift()
	case 11:
		if (v.acr^value)&mos6522AcrShiftMode != 0 {
			v.srShifted = false
		}
		v.acr = value
	case 12:
		v.pcr = value
	case 13:
		// Writing ones clears the flags
		v.ifr &^= value & 0x7f
	case 14:
		if value&0x80 != 0 {
			v.ier |= value & 0x7f
		} else {
			v.ier &^= value & 0x7f
		}
	}
}

// Tick advances the timers by the elapsed CPU cycles
func (v *MOS6522) Tick(elapsedCycles uint64) {
	v.t1counter -= int64(elapsedCycles)
	if v.t1counter < 0 {
		if v.acr&mos6522AcrT1FreeRunning != 0 {
			// Reload from the latch and interrupt on each underflow
			period := int64(v.t1latch) + 2
			v.t1counter += period * (1 + (-v.t1counter-1)/period)
			v.ifr |= mos6522IntT1
		} else {
			// One shot: interrupt once, the counter rolls over
			if !v.t1fired {
				v.ifr |= mos6522IntT1
				v.t1fired = true
			}
			v.t1counter = int64(uint16(v.t1counter))
		}
	}

	if v.isShiftingOutUnderT2() {
		v.tickShift(int64(elapsedCycles))
		return
	}

	v.t2counter -= int64(elapsedCycles)
	if v.t2counter < 0 {
		// One shot: interrupt once, the counter rolls over
		if !v.t2fired {
			v.ifr |= mos6522IntT2
			v.t2fired = true
		}
		v.t2counter = int64(uint16(v.t2counter))
	}
}

/*
The shift register shifting out at the rate of the low byte of T2: a bit
every T2L+2 cycles, the top bit first, on CB2. In the free running mode the
bits go round for ever, which makes a tone of the bit pattern written to the
register. That is how the PET makes sound.
*/

func (v *MOS6522) isShiftingOutUnderT2() bool {
	return v.acr&mos6522AcrShiftOutUnderMask == mos6522AcrShiftOutUnderT2
}

func (v *MOS6522) startShift() {
	v.srBits = 8
}

func (v *MOS6522) tickShift(cycles int64) {
	period := int64(v.t2latchL) + 2
	freeRunning := v.acr&mos6522AcrShiftMode == mos6522AcrShiftOutFreeT2
	v.srCycles += cycles
	for v.srCycles >= period {
		v.srCycles -= period
		if !freeRunning {
			if v.srBits == 0 {
				v.srCycles = 0
				return
			}
			v.srBits--
			if v.srBits == 0 {
				v.ifr |= mos6522IntSR
			}
		}
		v.srOutput = v.sr&0x80 != 0
		v.sr = v.sr<<1 | v.sr>>7
		v.srShifted = true
	}
}

// InterruptAsserted returns the state of the IRQ output line
func (v *MOS6522) InterruptAsserted() bool {
	return v.ifr&v.ier&0x7f != 0
}

// Reset clears the registers as the RES pin. The timers and the shift
// register are not affected
func (v *MOS6522) Reset() {
	v.ora, v.orb = 0, 0
	v.ddra, v.ddrb = 0, 0
	v.acr, v.pcr = 0, 0
	v.ifr, v.ier = 0, 0
	v.srShifted = false
}

// GetPortA returns the values on the port A pins
func (v *MOS6522) GetPortA() uint8 {
	return (v.ora & v.ddra) | (v.ira &^ v.ddra)
}

// GetPortB returns the values on the port B pins
func (v *MOS6522) GetPortB() uint8 {
	return (v.orb & v.ddrb) | (v.irb &^ v.ddrb)
}

// SetInputA sets the values on the port A input pins
func (v *MOS6522) SetInputA(value uint8) {
	v.ira = value
}

// SetInputB sets the values on the port B input pins
func (v *MOS6522) SetInputB(value uint8) {
	v.irb = value
}

// SetCA1 drives the CA1 pin
func (v *MOS6522) SetCA1(level bool) {
	v.setControlLine(&v.ca1, level,
		v.pcr&mos6522PcrCA1PositiveEdge != 0, mos6522IntCA1)
}

// SetCA2 drives the CA2 pin, ignored while it is programmed as an output
func (v *MOS6522) SetCA2(level bool) {
	if v.pcr&mos6522PcrCA2Output != 0 {
		return
	}
	v.setControlLine(&v.ca2, level,
		v.pcr&mos6522PcrCA2PositiveEdge != 0, mos6522IntCA2)
}

// SetCB1 drives the CB1 pin
func (v *MOS6522) SetCB1(level bool) {
	v.setControlLine(&v.cb1, level,
		v.pcr&mos6522PcrCB1PositiveEdge != 0, mos6522IntCB1)
}

// SetCB2 drives the CB2 pin, ignored while it is programmed as an output
func (v *MOS6522) SetCB2(level bool) {
	if v.pcr&mos6522PcrCB2Output != 0 {
		return
	}
	v.setControlLine(&v.cb2, level,
		v.pcr&mos6522PcrCB2PositiveEdge != 0, mos6522IntCB2)
}

// GetCA2 returns the level of the CA2 pin. As an input, or in the handshake
// and pulse modes, it is high.
func (v *MOS6522) GetCA2() bool {
	return controlOutputLevel(v.pcr >> 1)
}

// GetCB2 returns the level of the CB2 pin: the shift register when it is
// shifting out, else as GetCA2() for CA2
func (v *MOS6522) GetCB2() bool {
	if v.isShiftingOutUnderT2() && v.srShifted {
		return v.srOutput
	}
	return controlOutputLevel(v.pcr >> 5)
}

// controlOutputLevel decodes the 3 bits of the PCR for CA2 or CB2. Only 110
// drives the line low.
func controlOutputLevel(mode uint8) bool {
	return mode&7 != 6
}

// setControlLine raises the interrupt flag when the pin moves to the level
// the peripheral control register asks for
func (v *MOS6522) setControlLine(pin *bool, level bool, positiveEdge bool, flag uint8) {
	if *pin == level {
		return
	}
	*pin = level

	if level == positiveEdge {
		v.ifr |= flag
	}
}
