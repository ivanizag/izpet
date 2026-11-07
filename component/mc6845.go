package component

/*
MC6845 CRT Controller, as the MOS 6545 the PET uses
See:

	Motorola MC6845 datasheet
	MOS 6545 datasheet

Taken from izapple2, where it drives the Videx cards. Only the registers are
here, the PET video board decides what to do with them.

Registers:

	R0: horizontal total - 1        R1: horizontal displayed
	R2: horizontal sync position    R3: sync widths, vertical on the high nibble
	R4: vertical total - 1          R5: vertical total adjust, in scan lines
	R6: vertical displayed          R7: vertical sync position
	R8: mode control                R9: scan lines per character row - 1
	R10, R11: cursor start and end  R12, R13: display start address
	R14, R15: cursor address        R16, R17: light pen address
*/
type MC6845 struct {
	reg [18]uint8 // Internal registers R0 to R17
	sel uint8     // Selected address register AR

	// VBlank is the vertical blanking bit of the 6545 status register, to
	// be updated by the owner
	VBlank bool
}

// Registers used by the video boards
const (
	MC6845HorizontalTotal     = 0
	MC6845HorizontalDisplayed = 1
	MC6845SyncWidth           = 3
	MC6845VerticalTotal       = 4
	MC6845VerticalAdjust      = 5
	MC6845VerticalDisplayed   = 6
	MC6845VerticalSync        = 7
	MC6845ScanLines           = 9
	MC6845StartHigh           = 12
	MC6845StartLow            = 13
)

// Read returns the status register with rs low, or the selected register
func (m *MC6845) Read(rs bool) uint8 {
	if !rs {
		// The 6545 status register. The 6845 has nothing there.
		if m.VBlank {
			return 0x20
		}
		return 0x00
	} else if m.sel >= 14 && m.sel <= 17 {
		// Only R14 to R17 are readable
		return m.reg[m.sel]
	}
	return 0x00
}

// Write selects a register with rs low, or writes the selected register
func (m *MC6845) Write(rs bool, value uint8) {
	if !rs {
		m.sel = value & 0x1f
	} else if m.sel <= 15 {
		// R0 to R15 are writable
		m.reg[m.sel] = value
	}
}

// Reg returns the value of a register
func (m *MC6845) Reg(r int) uint8 {
	return m.reg[r]
}

// StartAddress returns the display start address, R12 and R13
func (m *MC6845) StartAddress() uint16 {
	return uint16(m.reg[MC6845StartHigh]&0x3f)<<8 | uint16(m.reg[MC6845StartLow])
}

// VerticalSyncLines returns the width of the vertical sync pulse in scan lines
func (m *MC6845) VerticalSyncLines() int {
	lines := int(m.reg[MC6845SyncWidth] >> 4)
	if lines == 0 {
		lines = 16
	}
	return lines
}
