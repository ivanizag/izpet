package ieee488

import (
	"testing"
)

// testHandler records what it is told and talks from a buffer
type testHandler struct {
	listened []uint8
	talked   []uint8
	received []uint8
	lastEOI  bool
	unlisten int
	toSend   []uint8
}

func (h *testHandler) Listen(s uint8)            { h.listened = append(h.listened, s) }
func (h *testHandler) Talk(s uint8)              { h.talked = append(h.talked, s) }
func (h *testHandler) Receive(b uint8, eoi bool) { h.received = append(h.received, b); h.lastEOI = eoi }
func (h *testHandler) Unlisten()                 { h.unlisten++ }
func (h *testHandler) Untalk()                   {}
func (h *testHandler) Next()                     { h.toSend = h.toSend[1:] }
func (h *testHandler) Peek() (uint8, bool, bool) {
	if len(h.toSend) == 0 {
		return 0, false, false
	}
	return h.toSend[0], len(h.toSend) == 1, true
}

// testController drives the bus as the PET kernal does
type testController struct {
	t   *testing.T
	bus *Bus
	l   Lines
}

func (c *testController) set() {
	c.bus.SetController(c.l)
}

// send sends a byte as talker, it returns false if the device is not present
func (c *testController) send(b uint8, eoi bool) bool {
	c.l.NDAC = true
	c.l.NRFD = true
	c.set()
	lv := c.bus.Levels()
	if lv.NRFD && lv.NDAC {
		return false
	}
	c.l.Data = ^b
	c.l.EOI = !eoi
	c.set()
	if !c.bus.Levels().NRFD {
		c.t.Fatal("the listener is not ready")
	}
	c.l.DAV = false
	c.set()
	if !c.bus.Levels().NDAC {
		c.t.Fatal("the byte was not accepted")
	}
	c.l.DAV = true
	c.l.Data = 0xff
	c.l.EOI = true
	c.set()
	return true
}

func (c *testController) command(b uint8) bool {
	c.l.ATN = false
	c.set()
	ok := c.send(b, false)
	return ok
}

func (c *testController) release() {
	c.l.ATN = true
	c.set()
}

// receive reads a byte as listener, it returns false on a timeout
func (c *testController) receive() (uint8, bool, bool) {
	c.l.NDAC = false
	c.l.NRFD = true
	c.set()
	for i := 0; i < 100 && c.bus.Levels().DAV; i++ {
		c.bus.Tick(10)
	}
	lv := c.bus.Levels()
	if lv.DAV {
		return 0, false, false
	}
	c.l.NRFD = false
	c.set()
	b := ^c.bus.Levels().Data
	eoi := !c.bus.Levels().EOI
	c.l.NDAC = true
	c.set()
	if !c.bus.Levels().DAV {
		c.t.Fatal("DAV still low after the byte was accepted")
	}
	c.l.NDAC = false
	c.set()
	return b, eoi, true
}

func newTestBus(t *testing.T) (*testController, *testHandler) {
	h := &testHandler{}
	bus := NewBus()
	bus.Attach(NewPeripheral(8, h))
	return &testController{t: t, bus: bus, l: Released()}, h
}

func TestListen(t *testing.T) {
	c, h := newTestBus(t)
	if !c.command(0x28) || !c.command(0x62) {
		t.Fatal("the device did not accept the commands")
	}
	c.release()
	c.send('A', false)
	c.send('B', true)
	c.command(0x3f)
	c.release()

	if len(h.listened) != 1 || h.listened[0] != 0x62 {
		t.Errorf("listened %v", h.listened)
	}
	if string(h.received) != "AB" || !h.lastEOI {
		t.Errorf("received %q, eoi %v", h.received, h.lastEOI)
	}
	if h.unlisten != 1 {
		t.Errorf("unlistened %v times", h.unlisten)
	}
}

func TestDeviceNotPresent(t *testing.T) {
	c, _ := newTestBus(t)
	c.command(0x29) // LISTEN 9
	c.command(0x62)
	c.release()
	if c.send('A', false) {
		t.Error("the device 9 should not be present")
	}
}

func TestTalk(t *testing.T) {
	c, h := newTestBus(t)
	h.toSend = []uint8("HI")
	c.command(0x48)
	c.command(0x60)
	// The turnaround of the kernal: NRFD high before NDAC goes low
	c.l.NRFD = true
	c.set()
	c.l.NDAC = false
	c.release()

	b1, eoi1, ok1 := c.receive()
	b2, eoi2, ok2 := c.receive()
	_, _, ok3 := c.receive()
	if !ok1 || !ok2 || b1 != 'H' || b2 != 'I' || eoi1 || !eoi2 {
		t.Errorf("received %q %v %v, %q %v %v", b1, eoi1, ok1, b2, eoi2, ok2)
	}
	if ok3 {
		t.Error("there should be nothing more to receive")
	}
}

func TestTalkerWaitsForTheListener(t *testing.T) {
	c, h := newTestBus(t)
	h.toSend = []uint8("X")
	c.command(0x48)
	c.command(0x60)
	c.release()

	// NRFD goes high with NDAC still released: there is no listener yet
	c.l.NRFD = true
	c.l.NDAC = true
	c.set()
	c.bus.Tick(1000)
	if !c.bus.Levels().DAV {
		t.Fatal("the talker should wait for a listener")
	}
	b, _, ok := c.receive()
	if !ok || b != 'X' || len(h.toSend) != 0 {
		t.Errorf("received %q, %v", b, ok)
	}
}
