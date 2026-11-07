package ieee488

/*
Peripheral is a device on the bus with an address that listens and talks as
the Commodore drives and printers do. It does the handshakes and the
addressing, and passes the rest to a Handler.

The commands are sent by the PET with ATN low, and every device accepts them:

	$20+n LISTEN device n     $3f UNLISTEN
	$40+n TALK device n       $5f UNTALK
	$60+s secondary address s, the channel
	$e0+s CLOSE channel s
	$f0+s OPEN channel s, the name follows as data

The acceptor handshake, as listener: with NRFD high the device is ready.
When the talker pulls DAV low the device pulls NRFD low, takes the byte and
releases NDAC. When DAV goes back high it pulls NDAC low and releases NRFD.

The source handshake, as talker: when the listeners are ready, with NDAC low
and NRFD high, the device puts the byte on the lines, with EOI low on the
last one, and pulls DAV low. When NDAC goes high it releases DAV and the
data.

The device does not answer at once as talker: its processor takes a few
microseconds. The PET relies on it, when it starts a TALK it releases NRFD a
few cycles before NDAC, and a device answering at once would take that for a
listener ready and the release of NDAC for the byte accepted. The byte is
put on the bus after talkerDelay cycles with the listener ready.

A device not addressed releases all its lines. The PET sees NRFD and NDAC
both high when it starts sending a byte and reports the device not present.
A talker with nothing to send never pulls DAV low, the PET times out.
*/
type Peripheral struct {
	address uint8
	handler Handler

	listening bool
	talking   bool
	secondary bool // The last command addressed this device, a secondary may follow

	accepted bool // The byte on the bus has been taken, waiting for DAV to go high
	sending  bool // A byte is on the bus, waiting for NDAC to go high
	drive    Lines

	waiting  bool   // The listener is ready, the byte goes on the bus after the delay
	waitLeft uint64 // Cycles left of the delay
	levels   Lines  // The levels last seen
}

// talkerDelay is the time a talker takes to put a byte on the bus, in
// cycles of 1 µs
const talkerDelay = 40

// Handler does what a peripheral is asked for
type Handler interface {
	// Listen is called when the device is told to listen with a secondary
	// address, $60+channel, $e0+channel for CLOSE or $f0+channel for OPEN
	Listen(secondary uint8)
	// Talk is called when the device is told to talk with a secondary
	// address
	Talk(secondary uint8)
	// Receive is called with each byte sent to the device while listening
	Receive(b uint8, eoi bool)
	// Unlisten is called when the device stops listening
	Unlisten()
	// Untalk is called when the device stops talking
	Untalk()
	// Peek returns the next byte to send while talking, and if it is the
	// last. It returns false if there is nothing to send.
	Peek() (b uint8, eoi bool, ok bool)
	// Next moves past the byte returned by Peek, once it is accepted
	Next()
}

const (
	cmdListen    = 0x20
	cmdUnlisten  = 0x3f
	cmdTalk      = 0x40
	cmdUntalk    = 0x5f
	cmdSecondary = 0x60
	cmdMask      = 0xe0
)

// NewPeripheral returns a device with an address, 0 to 30
func NewPeripheral(address uint8, handler Handler) *Peripheral {
	p := &Peripheral{address: address, handler: handler}
	p.Reset()
	return p
}

// Address returns the device number
func (p *Peripheral) Address() uint8 {
	return p.address
}

// Reset puts the device in the idle state
func (p *Peripheral) Reset() {
	p.listening = false
	p.talking = false
	p.secondary = false
	p.accepted = false
	p.sending = false
	p.waiting = false
	p.drive = Released()
}

// Tick lets the time pass for a talker waiting to put a byte on the bus. It
// returns true if the device changed its lines.
func (p *Peripheral) Tick(cycles uint64) bool {
	if !p.waiting {
		return false
	}
	if cycles < p.waitLeft {
		p.waitLeft -= cycles
		return false
	}
	p.waiting = false
	before := p.drive
	if p.talking && p.isListenerReady(p.levels) {
		p.putByte()
	}
	return p.drive != before
}

// IsWaiting returns true while a talker waits to put a byte on the bus
func (p *Peripheral) IsWaiting() bool {
	return p.waiting
}

// Update reacts to the levels of the bus
func (p *Peripheral) Update(l Lines) Lines {
	p.levels = l
	switch {
	case !l.ATN:
		// Every device accepts the commands
		p.stopSending()
		p.accept(l, true)
	case p.listening:
		p.accept(l, false)
	case p.talking:
		p.talk(l)
	default:
		p.drive = Released()
		p.accepted = false
	}
	return p.drive
}

func (p *Peripheral) accept(l Lines, command bool) {
	if !l.DAV && !p.accepted {
		// Take the byte
		b := ^l.Data
		eoi := !l.EOI
		p.accepted = true
		p.drive.NRFD = false
		p.drive.NDAC = true
		if command {
			p.command(b)
		} else {
			p.handler.Receive(b, eoi)
		}
		return
	}
	if l.DAV {
		// Ready for the next byte
		p.accepted = false
		p.drive.NDAC = false
		p.drive.NRFD = true
	}
}

func (p *Peripheral) command(b uint8) {
	switch {
	case b == cmdUnlisten:
		if p.listening {
			p.listening = false
			p.handler.Unlisten()
		}
		p.secondary = false
	case b == cmdUntalk:
		if p.talking {
			p.talking = false
			p.handler.Untalk()
		}
		p.secondary = false
	case b&cmdMask == cmdListen:
		if b&0x1f == p.address {
			p.listening = true
			p.secondary = true
		} else {
			p.secondary = false
		}
	case b&cmdMask == cmdTalk:
		if b&0x1f == p.address {
			p.talking = true
			p.secondary = true
		} else {
			// There is only one talker
			if p.talking {
				p.talking = false
				p.handler.Untalk()
			}
			p.secondary = false
		}
	case b >= cmdSecondary:
		if p.secondary {
			if p.listening {
				p.handler.Listen(b)
			} else if p.talking {
				p.handler.Talk(b)
			}
		}
		p.secondary = false
	}
}

func (p *Peripheral) talk(l Lines) {
	if !p.drive.NRFD || !p.drive.NDAC {
		// Not a listener anymore. The levels seen still have the lines
		// this device held, wait for the bus to settle without them.
		p.drive.NRFD = true
		p.drive.NDAC = true
		return
	}

	if p.sending {
		if l.NDAC {
			// Accepted
			p.stopSending()
			p.handler.Next()
		}
		return
	}

	if !p.isListenerReady(l) {
		p.waiting = false
		return
	}
	if !p.waiting {
		p.waiting = true
		p.waitLeft = talkerDelay
	}
}

// isListenerReady returns true when a listener is there, holding NDAC low,
// and ready, with NRFD high
func (p *Peripheral) isListenerReady(l Lines) bool {
	return l.NRFD && !l.NDAC && l.DAV && !p.sending
}

func (p *Peripheral) putByte() {
	if b, eoi, ok := p.handler.Peek(); ok {
		p.drive.Data = ^b
		p.drive.EOI = !eoi
		p.drive.DAV = false
		p.sending = true
	}
}

func (p *Peripheral) stopSending() {
	p.sending = false
	p.waiting = false
	p.drive.Data = 0xff
	p.drive.EOI = true
	p.drive.DAV = true
}
