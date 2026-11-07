// Package ieee488 is the IEEE-488 bus of the PET and the devices on it, the
// disk drives and the printers. It knows nothing of the PET.
package ieee488

/*
The IEEE-488 bus, or GPIB, as Commodore used it.

All the lines are open collector and active low: a line is low when any
device pulls it down, and high, released, when none does. The levels are
kept as booleans, true for high. A logical 1 on a data line is a low level.

	DIO1-8 data
	EOI    end of the data, the last byte is sent with it low
	DAV    data valid, driven by the talker
	NRFD   not ready for data, driven by the listeners
	NDAC   data not accepted, driven by the listeners
	ATN    attention, driven by the controller, the PET, to send commands
	SRQ    service request, not used by the Commodore devices

The devices react to the changes of the lines at once, not in a time of
their own: when the PET changes a line the bus settles, every device seeing
the new levels and changing its lines until none changes anymore.

See:
  - "The PET revealed", the IEEE-488 chapter
  - VICE, src/parallel/parallel.c
*/

// Lines are the levels of the lines of the bus, true for high
type Lines struct {
	Data uint8 // The levels of DIO1-8, a 0 bit is a low level, a logical 1
	EOI  bool
	DAV  bool
	NRFD bool
	NDAC bool
	ATN  bool
	SRQ  bool
}

// Released returns the lines with nothing driving them, all high
func Released() Lines {
	return Lines{Data: 0xff, EOI: true, DAV: true, NRFD: true, NDAC: true, ATN: true, SRQ: true}
}

func (l Lines) and(o Lines) Lines {
	return Lines{
		Data: l.Data & o.Data,
		EOI:  l.EOI && o.EOI,
		DAV:  l.DAV && o.DAV,
		NRFD: l.NRFD && o.NRFD,
		NDAC: l.NDAC && o.NDAC,
		ATN:  l.ATN && o.ATN,
		SRQ:  l.SRQ && o.SRQ,
	}
}

// Device is something on the bus
type Device interface {
	// Update sees the levels of the bus and returns the lines the device
	// drives, Released() for none
	Update(levels Lines) Lines
	// Reset puts the device in its idle state
	Reset()
}

// Ticker is a device that needs the time to pass, see Peripheral
type Ticker interface {
	// Tick lets the cycles pass, it returns true if the device changed
	// the lines it drives
	Tick(cycles uint64) bool
	// IsWaiting returns true while the device needs Tick
	IsWaiting() bool
}

// Bus is the IEEE-488 bus with the controller and the devices on it
type Bus struct {
	devices    []Device
	drives     []Lines
	controller Lines
	levels     Lines
}

// maxSettleRounds bounds the reactions of the devices to a change
const maxSettleRounds = 32

// NewBus returns an empty bus
func NewBus() *Bus {
	var b Bus
	b.controller = Released()
	b.levels = Released()
	return &b
}

// Attach puts a device on the bus
func (b *Bus) Attach(d Device) {
	b.devices = append(b.devices, d)
	b.drives = append(b.drives, Released())
	b.settle()
}

// SetController sets the lines driven by the controller, the PET, and lets
// the devices react
func (b *Bus) SetController(lines Lines) {
	if lines == b.controller {
		return
	}
	b.controller = lines
	b.settle()
}

// Levels returns the levels of the lines
func (b *Bus) Levels() Lines {
	return b.levels
}

// Tick lets the time pass for the devices that need it
func (b *Bus) Tick(cycles uint64) {
	changed := false
	for _, d := range b.devices {
		if t, ok := d.(Ticker); ok && t.IsWaiting() {
			changed = t.Tick(cycles) || changed
		}
	}
	if changed {
		b.settle()
	}
}

// IsWaiting returns true while a device needs the time to pass
func (b *Bus) IsWaiting() bool {
	for _, d := range b.devices {
		if t, ok := d.(Ticker); ok && t.IsWaiting() {
			return true
		}
	}
	return false
}

// Reset puts all the devices in their idle state
func (b *Bus) Reset() {
	for i, d := range b.devices {
		d.Reset()
		b.drives[i] = Released()
	}
	b.settle()
}

func (b *Bus) settle() {
	for round := 0; round < maxSettleRounds; round++ {
		b.levels = b.controller
		for _, drive := range b.drives {
			b.levels = b.levels.and(drive)
		}

		changed := false
		for i, d := range b.devices {
			drive := d.Update(b.levels)
			if drive != b.drives[i] {
				b.drives[i] = drive
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}
